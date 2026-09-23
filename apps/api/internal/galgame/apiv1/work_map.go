package apiv1

import (
	"cmp"
	"context"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/entityapiv1"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/playstate"
	"kun-galgame-api/internal/galgame/resourcevocab"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/pkg/catalogclient"
)

const contributorMax = 50

var (
	coverSlots = map[string]bool{
		"main": true, "pkgfront": true, "dig": true, "pkgback": true,
		"pkgcontent": true, "pkgside": true, "pkgmed": true, "other": true,
	}
	attrRoles = map[string]bool{
		"developer": true, "publisher": true, "circle": true, "brand": true,
	}
	companyKinds = map[string]bool{
		"game_brand": true, "bunko": true, "publisher": true,
		"anime_studio": true, "doujin_circle": true, "group": true,
	}
	characterKinds = map[string]bool{
		"main": true, "secondary": true, "appears": true,
	}
	spoilerLevels = []string{"none", "minor", "major"}
)

func spoilerOf(n int) string {
	if n >= 0 && n < len(spoilerLevels) {
		return spoilerLevels[n]
	}
	return spoilerLevels[len(spoilerLevels)-1]
}

func catalogTime(s string) repr.DateTime {
	if s == "" {
		return repr.Timestamp(time.Unix(0, 0).UTC())
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z", time.RFC3339Nano} {
		if t, err := time.Parse(layout, s); err == nil {
			return repr.Timestamp(t)
		}
	}
	return repr.Timestamp(time.Unix(0, 0).UTC())
}

func contentRatingOf(rating string) string {
	if rating == "r18" {
		return "r18"
	}
	return "all_ages"
}

func originalLanguageOf(olang string) *string {
	return workrepr.Lang(clientProductLocale(olang))
}

func clientProductLocale(olang string) string {
	tag := strings.ToLower(strings.TrimSpace(olang))
	switch {
	case tag == "":
		return ""
	case tag == "ja" || strings.HasPrefix(tag, "ja-"):
		return "ja-jp"
	case tag == "zh-hant" || strings.HasPrefix(tag, "zh-hant-") ||
		tag == "zh-tw" || tag == "zh-hk":
		return "zh-tw"
	case tag == "zh" || strings.HasPrefix(tag, "zh"):
		return "zh-cn"
	case tag == "en" || strings.HasPrefix(tag, "en-"):
		return "en-us"
	default:
		return olang
	}
}

func aliasesOf(d *client.CatalogWorkDetail, displayName string) []entityapiv1.AliasName {
	out := []entityapiv1.AliasName{}
	seen := map[string]bool{displayName: true, "": true}
	for _, t := range d.Titles {
		if seen[t.Title] || len(t.Title) > 512 {
			continue
		}
		seen[t.Title] = true
		out = append(out, entityapiv1.AliasName(t.Title))
		if len(out) >= 1000 {
			break
		}
	}
	return out
}

func linksOf(d *client.CatalogWorkDetail) []workrepr.CatalogLink {
	out := []workrepr.CatalogLink{}
	for _, l := range d.Links {
		out = workrepr.AppendLink(out, l.Source, l.URL)
	}
	return out
}

func externalRefsOf(d *client.CatalogWorkDetail) []WorkExternalRef {
	out := []WorkExternalRef{}
	for _, r := range d.Refs {
		site := strings.ToLower(r.Source)
		if r.ExternalID == "" || len(r.ExternalID) > 64 || !siteOK(site) {
			continue
		}
		out = append(out, WorkExternalRef{Site: site, ExternalID: r.ExternalID})
	}
	return out
}

func siteOK(site string) bool {
	if site == "" || len(site) > 64 {
		return false
	}
	first := site[0]
	if !((first >= 'a' && first <= 'z') || (first >= '0' && first <= '9')) {
		return false
	}
	for i := 1; i < len(site); i++ {
		c := site[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
			continue
		}
		return false
	}
	return true
}

func coverSlotOf(kind string) string {
	k := strings.ToLower(kind)
	if coverSlots[k] {
		return k
	}
	return "other"
}

func tagsOf(d *client.CatalogWorkDetail, includeNSFW bool) []WorkTag {
	out := []WorkTag{}
	for _, t := range d.Tags {
		if t.CanonicalID == 0 || t.Tier == client.TagTierHidden {
			continue
		}
		if t.Sexual && !includeNSFW {
			continue
		}
		kind := t.Kind
		if kind != "content" && kind != "meta" {
			if kind != "" {
				slog.Warn("work tags: unknown tag_kind taken as meta", "tag_id", t.CanonicalID, "tag_kind", kind)
			}
			kind = "meta"
		}
		name := workrepr.Name(cmp.Or(t.DisplayName, t.Name), "", client.LocalizedValues(t.Localized))
		out = append(out, WorkTag{
			TagSummary: entityapiv1.TagSummary{
				Object: "tag", ID: repr.ID(int(t.CanonicalID)), CatalogName: name,
				TagKind: kind, IsSexual: t.Sexual, CatalogWorkCount: max(t.WorkCount, 0),
			},
			Spoiler: spoilerOf(t.Spoiler),
		})
	}
	return out
}

func creditsOf(d *client.CatalogWorkDetail) []WorkCreditGroup {
	out := []WorkCreditGroup{}
	for _, g := range d.Credits {
		if g.RoleKey == "" || strings.ContainsAny(g.RoleKey, " \t\n") {
			continue
		}
		people := []WorkCreditPerson{}
		for _, p := range g.Credits {
			voiced := []VoicedCharacter{}
			if p.Character != "" && len(p.Character) <= 512 {
				voiced = append(voiced, VoicedCharacter(p.Character))
			}
			people = append(people, WorkCreditPerson{
				CreditNameRef: entityapiv1.CreditNameRef{
					Object: "credit_name", ID: repr.ID(int(p.ID)),
					CatalogName: workrepr.Name(p.DisplayName, p.Latin, client.LocalizedValues(p.Localized)),
				},
				VoicedCharacters: voiced,
			})
		}
		name := g.RoleName
		if len(name) > 128 {
			name = string([]rune(name)[:128])
		}
		out = append(out, WorkCreditGroup{RoleKey: g.RoleKey, DisplayName: name, People: people})
	}
	return out
}

func charactersOf(ctx context.Context, d *client.CatalogWorkDetail, cdn string) []WorkCharacter {
	out := []WorkCharacter{}
	for _, c := range d.Characters {
		kind := strings.ToLower(c.Kind)
		if !characterKinds[kind] {
			slog.Warn("work characters: unknown character_kind, row dropped", "character_id", c.ID, "character_kind", c.Kind)
			continue
		}
		voices := make([]entityapiv1.CreditNameRef, 0, len(c.Voices))
		for i := range c.Voices {
			v := &c.Voices[i]
			voices = append(voices, entityapiv1.CreditNameRef{
				Object: "credit_name", ID: repr.ID(int(v.ID)),
				CatalogName: workrepr.Name(v.DisplayName, v.Latin, client.LocalizedValues(v.Localized)),
				Lang:        workrepr.Lang(v.Lang),
			})
		}
		identity := c.Identity
		if len(identity) > 512 {
			identity = string([]rune(identity)[:512])
		}
		out = append(out, WorkCharacter{
			CharacterRef: entityapiv1.CharacterRef{
				Object: "character", ID: repr.ID(int(c.ID)),
				CatalogName: workrepr.Name(c.DisplayName, c.Latin, client.LocalizedValues(c.Localized)),
			},
			Image:         workrepr.ImageFromURL(cdn, c.Image, c.ImageMeta.Width, c.ImageMeta.Height, c.ImageMeta.Thumbhash, nil),
			Figure:        workrepr.ImageFromURL(cdn, c.Figure, c.FigureMeta.Width, c.FigureMeta.Height, c.FigureMeta.Thumbhash, nil),
			CharacterKind: kind,
			Spoiler:       spoilerOf(c.Spoiler),
			Identity:      identity,
			Voices:        voices,
		})
	}
	return out
}

func playtimesOf(d *client.CatalogWorkDetail) []WorkPlaytimeAggregate {
	out := []WorkPlaytimeAggregate{}
	for _, r := range d.Playtimes {
		site := strings.ToLower(r.Source)
		if !siteOK(site) {
			continue
		}
		out = append(out, WorkPlaytimeAggregate{Site: site, Minutes: max(r.Minutes, 0), VoteCount: max(r.VoteCount, 0)})
	}
	return out
}

func externalRatingsOf(d *client.CatalogWorkDetail) []WorkExternalRating {
	out := []WorkExternalRating{}
	for _, r := range d.Ratings {
		site := strings.ToLower(r.Source)
		if !siteOK(site) {
			continue
		}
		buckets := []WorkExternalRatingBucket{}
		for _, b := range r.Distribution {
			buckets = append(buckets, WorkExternalRatingBucket{Bucket: float64(b.Score), VoteCount: max(b.Count, 0)})
		}
		var stats *WorkExternalRatingStats
		if r.Stats != nil {
			stats = &WorkExternalRatingStats{
				Mean: r.Stats.Average, Stdev: r.Stats.Stdev,
				Lowest: r.Stats.Min, Highest: r.Stats.Max,
			}
		}
		var rank *int
		if r.Rank != nil && *r.Rank >= 1 {
			rank = r.Rank
		}
		out = append(out, WorkExternalRating{
			Object: "work_external_rating", Site: site, RatingValue: r.Score,
			VoteCount: max(r.VoteCount, 0), SourceRank: rank, Buckets: buckets, Stats: stats,
		})
	}
	return out
}

func favoriteCountOf(d *client.CatalogWorkDetail) int {
	for _, r := range d.Popularity {
		if r.Source == "nextmoe" && r.Metric == "favorites" {
			return max(int(r.Value), 0)
		}
	}
	return 0
}

func resourceTypesOf(keys []string) []workrepr.ResourceType {
	present := map[string]bool{}
	for _, k := range keys {
		k = resourcevocab.CompatType(k)
		for _, v := range resourcevocab.TypeKeys {
			if k == v {
				present[k] = true
				break
			}
		}
	}
	out := make([]workrepr.ResourceType, 0, len(present))
	for _, k := range resourcevocab.TypeKeys {
		if present[k] {
			out = append(out, workrepr.ResourceType(k))
		}
	}
	return out
}

func viewerPlaytime(got *catalogclient.PlaytimeSelf, ws *catalogclient.WorkStateRecord) *WorkViewerPlaytime {
	minutes := 0
	if got != nil && got.Minutes >= catalogclient.PlaytimeMinutesFloor {
		minutes = got.Minutes
	}
	var state *string
	if ws != nil {
		if flat := playstate.FromCatalog(ws.State, ws.Completion); flat != "" {
			state = &flat
		}
	}
	if minutes == 0 && state == nil {
		return nil
	}
	return &WorkViewerPlaytime{Minutes: minutes, PlayState: state}
}

func (s *Service) companiesOf(ctx context.Context, d *client.CatalogWorkDetail) []WorkCompany {
	type acc struct {
		id, workCount                      int64
		displayName, labelKind, kind, lang string
		roles                              []string
	}
	order := []int64{}
	byID := map[int64]*acc{}
	names := map[int64]repr.CatalogName{}
	for _, l := range d.Labels {
		a, ok := byID[l.ID]
		if !ok {
			a = &acc{
				id: l.ID, workCount: int64(l.WorkCount),
				displayName: l.DisplayName, labelKind: l.LabelKind, kind: l.Kind, lang: l.Lang,
			}
			byID[l.ID] = a
			order = append(order, l.ID)
			names[l.ID] = workrepr.Name(l.DisplayName, "", client.LocalizedValues(l.Localized))
		}
		role := l.AttributionRole()
		if attrRoles[role] {
			dup := false
			for _, r := range a.roles {
				if r == role {
					dup = true
					break
				}
			}
			if !dup {
				a.roles = append(a.roles, role)
			}
		}
	}
	out := make([]WorkCompany, 0, len(order))
	detail := s.detailCatalog()
	for _, id := range order {
		a := byID[id]
		roles := make([]AttributionRole, len(a.roles))
		for i, r := range a.roles {
			roles[i] = AttributionRole(r)
		}
		kind := "game_brand"
		if companyKinds[a.labelKind] {
			kind = a.labelKind
		} else if companyKinds[a.kind] {
			kind = a.kind
		}
		co := WorkCompany{
			CompanySummary: entityapiv1.CompanySummary{
				Object: "company", ID: repr.ID(int(id)), CatalogName: names[id],
				CompanyKind: kind, Aliases: []entityapiv1.AliasName{},
				CatalogWorkCount: max(int(a.workCount), 0),
			},
			Links:            []workrepr.CatalogLink{},
			AttributionRoles: roles,
			Lang:             workrepr.Lang(a.lang),
		}
		if detail != nil && id > 0 {
			rec, found, _, appErr := detail.CatalogLabel(ctx, strconv.FormatInt(id, 10))
			if appErr == nil && found && rec != nil {
				name := workrepr.Name(rec.DisplayName, rec.Latin, client.LocalizedValues(rec.Localized))
				co.CatalogName = name
				if companyKinds[rec.Kind] {
					co.CompanyKind = rec.Kind
				}
				co.Logo = workrepr.ImageFromHash(s.cdn, rec.LogoHash)
				co.Aliases = aliasNames(rec.Aliases.Values(name.DisplayName))
				co.CatalogWorkCount = max(rec.WorkCount, 0)
				if lang := workrepr.Lang(rec.Lang); lang != nil {
					co.Lang = lang
				}
				links := []workrepr.CatalogLink{}
				for _, ln := range rec.Links {
					links = workrepr.AppendLink(links, ln.Source, ln.URL)
				}
				co.Links = links
			}
		}
		out = append(out, co)
	}
	return out
}

func aliasNames(values []string) []entityapiv1.AliasName {
	out := make([]entityapiv1.AliasName, 0, len(values))
	for _, v := range values {
		if v != "" && len(v) <= 512 {
			out = append(out, entityapiv1.AliasName(v))
		}
	}
	return out
}

func (s *Service) enginesOf(ctx context.Context, d *client.CatalogWorkDetail) []entityapiv1.Engine {
	out := []entityapiv1.Engine{}
	detail := s.detailCatalog()
	for _, e := range d.Engines {
		eng := entityapiv1.Engine{
			Object: "engine", ID: repr.ID(int(e.ID)),
			CatalogName:      workrepr.Name(cmp.Or(e.DisplayName, e.Name), "", client.LocalizedValues(e.Localized)),
			Aliases:          []entityapiv1.AliasName{},
			CatalogWorkCount: max(e.WorkCount, 0),
		}
		if detail != nil && e.ID > 0 {
			rec, found, appErr := detail.CatalogEngine(ctx, strconv.FormatInt(e.ID, 10))
			if appErr == nil && found && rec != nil {
				name := workrepr.Name(cmp.Or(rec.DisplayName, rec.Name), "", client.LocalizedValues(rec.Localized))
				eng.CatalogName = name
				eng.Aliases = aliasNames(rec.Aliases.Values(name.DisplayName))
				eng.Description = rec.Description
				eng.CatalogWorkCount = max(rec.WorkCount, 0)
			}
		}
		out = append(out, eng)
	}
	return out
}

func (s *Service) seriesOf(ctx context.Context, d *client.CatalogWorkDetail) []entityapiv1.SeriesSummary {
	out := []entityapiv1.SeriesSummary{}
	detail := s.detailCatalog()
	for _, sr := range d.Series {
		card := entityapiv1.SeriesSummary{
			Object: "series", ID: repr.ID(int(sr.ID)),
			CatalogName: workrepr.Name(cmp.Or(sr.DisplayName, sr.Name), "", client.LocalizedValues(sr.Localized)),
			SampleWorks: []entityapiv1.SeriesSampleWork{},
		}
		if detail != nil && sr.ID > 0 {
			rec, found, appErr := detail.CatalogSeries(ctx, strconv.FormatInt(sr.ID, 10))
			if appErr == nil && found && rec != nil {
				card.CatalogName = workrepr.Name(cmp.Or(rec.DisplayName, rec.Name), "", client.LocalizedValues(rec.Localized))
				card.HasNSFWWorks = rec.HasNSFW
				card.CatalogWorkCount = max(rec.WorkCount, 0)
			}
			s.fillSeriesSamples(ctx, int(sr.ID), &card)
		}
		out = append(out, card)
	}
	return out
}

func (s *Service) fillSeriesSamples(ctx context.Context, seriesID int, card *entityapiv1.SeriesSummary) {
	detail := s.detailCatalog()
	if detail == nil || s.lists == nil {
		return
	}
	res, appErr := detail.CatalogWorksSearch(ctx, client.OpenPopulation(url.Values{
		"series_id": {strconv.Itoa(seriesID)},
		"page":      {"1"},
		"limit":     {"100"},
		"include":   {workrepr.RowInclude},
		"sort":      {"released_asc"},
	}))
	if appErr != nil || res == nil {
		return
	}
	rows := make([]client.CatalogWorkListItem, 0, len(res.Items))
	ids := make([]int, 0, len(res.Items))
	for i := range res.Items {
		if client.CatalogItemRenderable(&res.Items[i]) && res.Items[i].ID > 0 {
			rows = append(rows, res.Items[i])
			ids = append(ids, int(res.Items[i].ID))
		}
	}
	if len(ids) == 0 {
		return
	}
	listed, _ := s.lists.ListIDs(model.GalgameListFilter{RestrictIDs: ids, Page: 1, Limit: len(ids), SortOrder: "desc"})
	isListed := map[int]bool{}
	for _, id := range listed {
		isListed[id] = true
	}
	for i := range rows {
		if !isListed[int(rows[i].ID)] {
			continue
		}
		card.ListedWorkCount++
		if len(card.SampleWorks) < 5 {
			card.SampleWorks = append(card.SampleWorks, entityapiv1.SeriesSampleWork{
				WorkRef: workrepr.Ref(ctx, &rows[i], s.cdn),
				Banner:  workrepr.Banner(&rows[i], s.cdn),
			})
		}
	}
}

func coversOf(d *client.CatalogWorkDetail, cdn string, tallies []catalogclient.CoverTally, viewer *WorkCoverViewer) []WorkCover {
	byHash := map[string]catalogclient.CoverTally{}
	for _, t := range tallies {
		if t.ImageHash != "" {
			byHash[t.ImageHash] = t
		}
	}
	out := []WorkCover{}
	for i, c := range d.Covers {
		if i > 9999 {
			slog.Warn("work covers: sort_order above 9999, row dropped", "index", i)
			continue
		}
		hash := c.Hash
		if hash == "" {
			hash = hashFromURL(c.URL)
		}
		img := workrepr.ImageFromURL(cdn, c.URL, c.Width, c.Height, c.Thumbhash, c.Sexual)
		if img == nil {
			continue
		}
		site := strings.ToLower(c.Source)
		if !siteOK(site) {
			continue
		}
		item := WorkCover{
			Object: "work_cover", Image: img, CoverSlot: coverSlotOf(c.Kind),
			Site: site, SortOrder: i, Viewer: viewer,
		}
		id := c.ID
		if t, ok := byHash[hash]; ok {
			if id <= 0 {
				id = t.ID
			}
			item.VoteCount = max(t.VoteCount, 0)
			if viewer != nil {
				v := WorkCoverViewer{HasVoted: t.Voted}
				item.Viewer = &v
			}
		}
		if id <= 0 {
			slog.Warn("work covers: cover row has no id, row dropped", "index", i)
			continue
		}
		item.ID = repr.ID(int(id))
		out = append(out, item)
	}
	return out
}

func screenshotsOf(d *client.CatalogWorkDetail, cdn string) []WorkScreenshot {
	out := []WorkScreenshot{}
	for i, sh := range d.Screenshots {
		if i > 9999 {
			slog.Warn("work screenshots: sort_order above 9999, row dropped", "index", i)
			continue
		}
		img := workrepr.ImageFromURL(cdn, sh.URL, sh.Width, sh.Height, sh.Thumbhash, sh.Sexual)
		if img == nil {
			continue
		}
		site := strings.ToLower(sh.Source)
		if !siteOK(site) {
			continue
		}
		caption := sh.Caption
		if r := []rune(caption); len(r) > 512 {
			caption = string(r[:512])
		}
		out = append(out, WorkScreenshot{
			Object: "work_screenshot", Image: img, Caption: caption, Site: site, SortOrder: i,
		})
	}
	return out
}

func hashFromURL(u string) string {
	if u == "" {
		return ""
	}
	base := u
	if i := strings.LastIndexByte(base, '/'); i >= 0 {
		base = base[i+1:]
	}
	return strings.TrimSuffix(base, ".webp")
}
