package client

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"kun-galgame-api/pkg/errors"
)

const catalogIDsChunk = 100

const catalogSpoilerCeiling = 2

// The works list gates localized{} and latin behind include=names, the same
// switch as the four-slot names block it replaces. Dropping "names" here
// because nothing reads that block any more leaves every list row with only
// display_name, so every Chinese title on the site reverts to the original.
const (
	catalogBriefInclude = "names,covers,refs,labels"
)

func openPopulation(q url.Values) url.Values {
	q.Set("nsfw", "true")
	return q
}

func OpenPopulation(q url.Values) url.Values { return openPopulation(q) }

// openPopulation is unconditional: content_rating is how old you must be to
// play the game, content_limit is the entry's own display verdict, and only the
// second one is kungal's gate. Leaving the age gate closed for an SFW reader
// left 493 of the 7,781 listed games visible, because 93% of them are r18. The
// v2 cutover dropped this call and rewrote the assertions that pinned it, so a
// green suite is not evidence here — 4b669e54 is.
func applyWorksGate(q url.Values, contentLimit string) url.Values {
	openPopulation(q)
	switch contentLimit {
	case "sfw", "nsfw":
		q.Set("content_limit", contentLimit)
	}
	return q
}

func ApplyWorksGate(q url.Values, isSFW bool) url.Values {
	return applyWorksGate(q, contentLimitFor(isSFW))
}

func contentLimitFor(isSFW bool) string {
	if isSFW {
		return "sfw"
	}
	return "all"
}

func (c *GalgameClient) worksByCatalogIDs(ctx context.Context, ids []int64, include, contentLimit string) ([]CatalogWorkListItem, *errors.AppError) {
	var out []CatalogWorkListItem
	for start := 0; start < len(ids); start += catalogIDsChunk {
		end := min(start+catalogIDsChunk, len(ids))
		chunk := ids[start:end]

		raw := make([]string, len(chunk))
		for i, id := range chunk {
			raw[i] = strconv.FormatInt(id, 10)
		}
		q := url.Values{
			"ids":   {strings.Join(raw, ",")},
			"limit": {strconv.Itoa(catalogIDsChunk)},
		}
		if include != "" {
			q.Set("include", include)
		}
		applyWorksGate(q, contentLimit)

		data, appErr := c.CatalogGet(ctx, "/catalog/works", q)
		if appErr != nil {
			return nil, appErr
		}
		var parsed catWorksListData
		if err := json.Unmarshal(data, &parsed); err != nil {
			return nil, errors.ErrInternal("解析 Catalog 作品列表响应失败")
		}
		out = append(out, parsed.Items...)
	}
	return out, nil
}

func (c *GalgameClient) CatalogRowsByWorkIDs(ctx context.Context, ids []int, include, contentLimit string) (map[int]CatalogWorkListItem, *errors.AppError) {
	if len(ids) == 0 {
		return map[int]CatalogWorkListItem{}, nil
	}
	catIDs := make([]int64, 0, len(ids))
	seen := make(map[int]bool, len(ids))
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			catIDs = append(catIDs, int64(id))
		}
	}
	if len(catIDs) == 0 {
		return map[int]CatalogWorkListItem{}, nil
	}
	rows, appErr := c.worksByCatalogIDs(ctx, catIDs, include, contentLimit)
	if appErr != nil {
		return nil, appErr
	}
	out := make(map[int]CatalogWorkListItem, len(rows))
	for i := range rows {
		row := rows[i]
		if !row.isRenderable() {
			continue
		}
		if row.ID > 0 {
			out[int(row.ID)] = row
		}
	}
	return out, nil
}

// CatalogMirror is the set of catalog fields the local galgame row keeps a copy
// of, so that SQL can filter and order on them before anything is hydrated.
// Both belong to catalog; the local columns are caches and never edited here.
type CatalogMirror struct {
	ContentLimit string
	// Catalog's own string, at whatever precision it knows: "2026", "2026-08"
	// or "2026-08-27". Empty means catalog has no date for this work, which is
	// an answer — not the same as "not asked yet".
	ReleaseDate string
}

func mirrorOf(row *CatalogWorkListItem) CatalogMirror {
	m := CatalogMirror{ContentLimit: row.ContentLimit}
	if row.ReleaseDate != nil {
		m.ReleaseDate = *row.ReleaseDate
	}
	return m
}

type CatalogWorkDetail struct {
	ID            int64                       `json:"id"`
	DisplayName   string                      `json:"display_name"`
	Localized     map[string]catLocalizedName `json:"localized"`
	Latin         string                      `json:"latin"`
	OLang         string                      `json:"olang"`
	ContentRating string                      `json:"content_rating"`
	ContentLimit  string                      `json:"content_limit"`
	ReleaseDate   *string                     `json:"release_date"`
	Updated       string                      `json:"updated"`
	Created       string                      `json:"created"`
	Claim         *catClaim                   `json:"claim"`

	Titles []struct {
		Lang    string `json:"lang"`
		Title   string `json:"title"`
		Kind    string `json:"kind"`
		Machine bool   `json:"machine"`
	} `json:"titles"`
	Refs []catRef     `json:"refs"`
	Tags []catWorkTag `json:"tags"`
	// Wave 212's second half renames this block to "intros", matching every
	// other catalog face. Both keys decode so the rename is not a cutover.
	Intro  []CatalogIntro `json:"intro"`
	Intros []CatalogIntro `json:"intros"`
	Covers []struct {
		ID        int64  `json:"id"`
		Hash      string `json:"hash"`
		URL       string `json:"url"`
		Kind      string `json:"kind"`
		Sexual    *int   `json:"sexual"`
		Violence  int    `json:"violence"`
		Source    string `json:"source"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		Thumbhash string `json:"thumbhash"`
		VoteCount int    `json:"vote_count"`
	} `json:"covers"`
	CoverSlots  *catCoverSlots `json:"cover_slots"`
	Screenshots []struct {
		URL       string `json:"url"`
		Caption   string `json:"caption"`
		Sexual    *int   `json:"sexual"`
		Violence  int    `json:"violence"`
		Source    string `json:"source"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		Thumbhash string `json:"thumbhash"`
	} `json:"screenshots"`
	Labels     []catWorkLabel     `json:"labels"`
	Engines    []catWorkEngine    `json:"engines"`
	Links      []catWorkLink      `json:"links"`
	Series     []catWorkSeries    `json:"series"`
	Credits    []catCreditGroup   `json:"credits"`
	Characters []catWorkCharacter `json:"characters"`
	Ratings    []catRating        `json:"ratings"`
	Playtimes  []catPlaytime      `json:"playtimes"`
	Popularity []catPopularity    `json:"popularity"`
}

// The favourite count on a game page comes from here, not from
// galgame_local.favorite_count: catalog computes nextmoe/favorites as the
// number of distinct people holding the work in any folder on any site, so a
// favourite added from the patch site shows up in this number and never in the
// local counter. include=popularity was already being requested and thrown
// away.
type catPopularity struct {
	Source string `json:"source"`
	Metric string `json:"metric"`
	Value  int64  `json:"value"`
}

type catWorkCharacter struct {
	ID          int64                       `json:"id"`
	DisplayName string                      `json:"display_name"`
	Localized   map[string]catLocalizedName `json:"localized"`
	Latin       string                      `json:"latin"`
	Kind        string                      `json:"kind"`
	Spoiler     int                         `json:"spoiler"`
	Image       string                      `json:"image"`
	Figure      string                      `json:"figure"`
	Identity    string                      `json:"identity"`
	Voices      []CatalogPerson             `json:"voices"`
	ImageMeta   ArtMeta                     `json:"-"`
	FigureMeta  ArtMeta                     `json:"-"`
}

type catCreditGroup struct {
	RoleKey  string          `json:"role_key"`
	RoleName string          `json:"role_name"`
	Credits  []catCreditItem `json:"credits"`
}

type catCreditItem struct {
	ID          int64                       `json:"id"`
	DisplayName string                      `json:"display_name"`
	Localized   map[string]catLocalizedName `json:"localized"`
	Latin       string                      `json:"latin"`
	CharacterID int64                       `json:"character_id"`
	Character   string                      `json:"character"`
}

func (c *catCreditItem) Name(ctx context.Context) string {
	return CatalogEntityName(ctx, c.Localized, c.DisplayName, c.Latin)
}

type catWorkTag struct {
	Name        string                      `json:"name"`
	DisplayName string                      `json:"display_name"`
	Localized   map[string]catLocalizedName `json:"localized"`
	Source      string                      `json:"source"`
	CanonicalID int64                       `json:"canonical_id"`
	Tier        string                      `json:"tier"`
	Kind        string                      `json:"kind"`
	Spoiler     int                         `json:"spoiler"`
	Sexual      bool                        `json:"sexual"`
	WorkCount   int                         `json:"work_count"`
}

type catWorkSeries struct {
	ID          int64                       `json:"id"`
	Name        string                      `json:"name"`
	DisplayName string                      `json:"display_name"`
	Localized   map[string]catLocalizedName `json:"localized"`
}

func (c *GalgameClient) CatalogWorkExists(ctx context.Context, workID int) (bool, *errors.AppError) {
	_, found, _, err := c.CatalogWorkDetail(ctx, workID)
	return found, err
}

func (c *GalgameClient) CatalogWorkDetail(ctx context.Context, workID int) (*CatalogWorkDetail, bool, int64, *errors.AppError) {
	if workID <= 0 {
		return nil, false, 0, nil
	}
	// The tag panel's 剧透等级 filter defaults to level 0 and reveals the rest on
	// demand, so it needs the rows to filter: asking for spoilers=0 here made
	// levels 1 and 2 match nothing, forever. SEO text must still cut back to
	// level 0 — see pages/galgame/[id]/index.vue.
	q := url.Values{
		"spoilers": {strconv.Itoa(catalogSpoilerCeiling)},
		"include":  {"credits"},
	}
	openPopulation(q)
	data, found, movedTo, appErr := c.catalogGetRecord(ctx, "/catalog/works/"+strconv.Itoa(workID), q)
	if appErr != nil {
		return nil, false, 0, appErr
	}
	if movedTo != 0 {
		return nil, false, movedTo, nil
	}
	if !found {
		return nil, false, 0, nil
	}
	var d CatalogWorkDetail
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, false, 0, errors.ErrInternal("解析 Catalog 作品详情响应失败")
	}
	if d.Claim != nil && d.Claim.State == claimStateHidden {
		return nil, false, 0, nil
	}
	c.hydrateRosterArt(d.Characters)
	return &d, true, 0, nil
}

func (c *GalgameClient) hydrateRosterArt(chars []catWorkCharacter) {
	urls := make([]string, 0, len(chars)*2)
	for _, ch := range chars {
		urls = append(urls, ch.Image, ch.Figure)
	}
	meta := c.resolveArtMeta(urls)
	if meta == nil {
		return
	}
	for i := range chars {
		chars[i].ImageMeta = meta[chars[i].Image]
		chars[i].FigureMeta = meta[chars[i].Figure]
	}
}

type CatalogWorksPage struct {
	Items      []CatalogWorkListItem
	NextCursor string
	Total      int64
	Count      int64
	Month      string
	Year       string
	Meta       catCalendarMeta
}

func (c *GalgameClient) CatalogWorksList(ctx context.Context, q url.Values) (*CatalogWorksPage, *errors.AppError) {
	data, appErr := c.CatalogGet(ctx, "/catalog/works", q)
	if appErr != nil {
		return nil, appErr
	}
	var parsed catWorksListData
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Catalog 作品列表响应失败")
	}
	page := &CatalogWorksPage{Items: parsed.Items}
	if parsed.NextCursor != nil {
		page.NextCursor = *parsed.NextCursor
	}
	return page, nil
}

func (c *GalgameClient) CatalogMemberWorkIDs(ctx context.Context, filter url.Values, isSFW bool, pageCap int) ([]int, *errors.AppError) {
	members, appErr := c.catalogMembers(ctx, filter, isSFW, pageCap)
	if appErr != nil {
		return nil, appErr
	}
	ids := make([]int, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.WorkID)
	}
	return ids, nil
}

type CatalogRollupMember struct {
	WorkID int
	Via    *CatalogLabelVia
}

func (c *GalgameClient) CatalogLabelRollupMembers(ctx context.Context, labelID, sort string, isSFW bool, pageCap int) ([]CatalogRollupMember, *errors.AppError) {
	q := url.Values{"label_id": {labelID}, "label_rollup": {"1"}}
	if sort != "" {
		q.Set("sort", sort)
	}
	return c.catalogMembers(ctx, q, isSFW, pageCap)
}

func (c *GalgameClient) catalogMembers(ctx context.Context, filter url.Values, isSFW bool, pageCap int) ([]CatalogRollupMember, *errors.AppError) {
	members := []CatalogRollupMember{}
	cursor := ""
	for page := 0; page < pageCap; page++ {
		q := url.Values{}
		for k, v := range filter {
			q[k] = v
		}
		q.Set("limit", strconv.Itoa(catalogIDsChunk))
		ApplyWorksGate(q, isSFW)
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		res, appErr := c.CatalogWorksList(ctx, q)
		if appErr != nil {
			return nil, appErr
		}
		for i := range res.Items {
			if !res.Items[i].isRenderable() {
				continue
			}
			if id := int(res.Items[i].ID); id > 0 {
				members = append(members, CatalogRollupMember{
					WorkID: id,
					Via:    res.Items[i].ViaLabel,
				})
			}
		}
		if res.NextCursor == "" {
			break
		}
		cursor = res.NextCursor
	}
	return members, nil
}

func (c *GalgameClient) CatalogWorksSearch(ctx context.Context, q url.Values) (*CatalogWorksPage, *errors.AppError) {
	data, appErr := c.CatalogGet(ctx, "/catalog/works/search", q)
	if appErr != nil {
		return nil, appErr
	}
	var parsed catWorksSearchData
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Catalog 搜索响应失败")
	}
	return &CatalogWorksPage{Items: parsed.Items, Total: parsed.Total}, nil
}

func (c *GalgameClient) CatalogCalendar(ctx context.Context, bucket string, q url.Values) (*CatalogWorksPage, *errors.AppError) {
	data, appErr := c.CatalogGet(ctx, "/catalog/calendar"+bucket, q)
	if appErr != nil {
		return nil, appErr
	}
	var parsed catWorksListData
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, errors.ErrInternal("解析 Catalog 月历响应失败")
	}
	count := parsed.Count
	if count == 0 {
		count = parsed.Total
	}
	page := &CatalogWorksPage{
		Items: parsed.Items, Count: count,
		Month: parsed.Month, Year: parsed.Year, Meta: parsed.Meta,
	}
	if parsed.NextCursor != nil {
		page.NextCursor = *parsed.NextCursor
	}
	return page, nil
}
