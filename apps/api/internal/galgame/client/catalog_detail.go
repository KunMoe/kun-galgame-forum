package client

import (
	"context"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

func CatalogDetailToFull(ctx context.Context, d *catWorkDetail, gid int) dto.NextMoeGalgameDetailFull {
	characters := catalogRosterToNextMoe(ctx, d.Characters)
	f := dto.NextMoeGalgameDetailFull{
		ID:               gid,
		ContentLimit:     contentLimitOf(d.Claim, d.ContentRating),
		AgeLimit:         ageLimitFromRating(d.ContentRating),
		OriginalLanguage: productLocale(d.OLang),
		ReleaseDate:      d.ReleaseDate,
		ReleaseDateTBA:   false,
		Updated:          d.Updated,
		Created:          d.Created,
		Refs:             refsMap(d.Refs),
		Staff:            catalogStaffFromCredits(ctx, d.Credits, characters),
		Characters:       characters,
		Covers:           catalogCoversToNextMoe(d),
		Screenshots:      catalogScreenshotsToNextMoe(d),
		Contributor:      []dto.NextMoeContributor{},
	}
	f.VndbID = f.Refs["vndb"]

	f.Name, f.NameOriginal = CatalogEntityNames(ctx, d.Localized, d.DisplayName, d.Latin)
	f.Alias = catalogAliases(d, f.Name)

	f.Intros = OrderIntros(d.introRows())

	if d.Claim != nil {
		f.Status = statusFromClaimState(d.Claim.State)
	}

	if hash, url, w, h, thumb := detailHero(d.CoverSlots, f.Covers); url != "" {
		f.EffectiveBannerHash = hash
		f.EffectiveBannerURL = url
		f.EffectiveBannerWidth = w
		f.EffectiveBannerHeight = h
		f.EffectiveBannerThumbhash = thumb
	}

	if hash, url, w, h, thumb := detailPortrait(d.CoverSlots, f.Covers); url != "" {
		f.EffectivePortraitHash = hash
		f.EffectivePortraitURL = url
		f.EffectivePortraitWidth = w
		f.EffectivePortraitHeight = h
		f.EffectivePortraitThumbhash = thumb
	}

	f.ExternalRatings = catalogExternalRatings(d.Ratings)
	f.Playtimes = catalogPlaytimes(d.Playtimes)
	f.FavoriteCount = catalogFavoriteCount(d.Popularity)

	labelAt := make(map[int64]int, len(d.Labels))
	for _, l := range d.Labels {
		if i, seen := labelAt[l.ID]; seen {
			f.Official[i].Official.Roles = appendUniqueStr(f.Official[i].Official.Roles, l.AttributionRole())
			continue
		}
		labelAt[l.ID] = len(f.Official)
		f.Official = append(f.Official, dto.NextMoeOfficialRel{Official: dto.NextMoeOfficial{
			ID: int(l.ID), Name: l.Name(ctx),
			Category:     l.LabelKind,
			Roles:        appendUniqueStr(nil, l.AttributionRole()),
			Lang:         l.Lang,
			Alias:        []dto.NextMoeAlias{},
			GalgameCount: l.WorkCount,
		}})
	}
	for _, sr := range d.Series {
		f.Series = append(f.Series, dto.NextMoeSeriesRef{ID: int(sr.ID), Name: sr.Label(ctx)})
	}
	for _, e := range d.Engines {
		var ew dto.NextMoeEngineWithAlias
		ew.Engine.ID = int(e.ID)
		ew.Engine.Name = e.Label(ctx)
		ew.Engine.Alias = []string{}
		ew.Engine.GalgameCount = e.WorkCount
		f.Engine = append(f.Engine, ew)
	}
	for _, t := range d.Tags {
		if t.CanonicalID == 0 {
			continue
		}
		if t.Tier == TagTierHidden {
			continue
		}
		f.Tag = append(f.Tag, dto.NextMoeTagWithSpoiler{
			SpoilerLevel: t.Spoiler,
			Tag: dto.NextMoeTag{
				ID: int(t.CanonicalID), Name: t.Label(),
				Category:     catalogTagCategory(t.Kind, t.Sexual),
				GalgameCount: t.WorkCount,
			},
		})
	}
	return f
}

func detailHero(slots *catCoverSlots, covers []dto.NextMoeGalgameCover) (hash, url string, w, h int, thumb string) {
	if slots != nil {
		if s := slots.Banner; s != nil {
			return hashFromURL(s.URL), s.URL, s.Width, s.Height, s.Thumbhash
		}
		if s := slots.Portrait; s != nil {
			return hashFromURL(s.URL), s.URL, s.Width, s.Height, s.Thumbhash
		}
	} else if c := legacyLandscapeCover(covers); c != nil {
		return c.ImageHash, c.CDNURL, c.Width, c.Height, c.Thumbhash
	}
	if len(covers) == 0 {
		return "", "", 0, 0, ""
	}
	c := covers[0]
	return c.ImageHash, c.CDNURL, c.Width, c.Height, c.Thumbhash
}

// cover_slots.portrait is filled from any cover when no portrait-shaped one exists, so it is not
// guaranteed to be taller than wide; consumers must crop rather than trust the aspect ratio.
func detailPortrait(slots *catCoverSlots, covers []dto.NextMoeGalgameCover) (hash, url string, w, h int, thumb string) {
	if slots != nil {
		if s := slots.Portrait; s != nil {
			return hashFromURL(s.URL), s.URL, s.Width, s.Height, s.Thumbhash
		}
		return "", "", 0, 0, ""
	}
	if c := legacyPortraitCover(covers); c != nil {
		return c.ImageHash, c.CDNURL, c.Width, c.Height, c.Thumbhash
	}
	return "", "", 0, 0, ""
}

func legacyPortraitCover(covers []dto.NextMoeGalgameCover) *dto.NextMoeGalgameCover {
	for i := range covers {
		c := &covers[i]
		if c.Width > 0 && c.Height > 0 && c.Height*20 > c.Width*21 {
			return c
		}
	}
	return nil
}

func catalogExternalRatings(rows []catRating) []dto.GalgameExternalRating {
	out := make([]dto.GalgameExternalRating, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.GalgameExternalRating{
			Source: r.Source, Score: r.Score, VoteCount: r.VoteCount, Rank: r.Rank,
			Distribution: catalogRatingBuckets(r.Distribution),
			Stats:        catalogRatingStats(r.Stats),
		})
	}
	return out
}

func catalogPlaytimes(rows []catPlaytime) []dto.GalgamePlaytime {
	if len(rows) == 0 {
		return nil
	}
	out := make([]dto.GalgamePlaytime, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.GalgamePlaytime{
			Source: r.Source, Minutes: r.Minutes, VoteCount: r.VoteCount,
		})
	}
	return out
}

func catalogRatingBuckets(rows []catRatingBucket) []dto.GalgameRatingBucket {
	if len(rows) == 0 {
		return nil
	}
	out := make([]dto.GalgameRatingBucket, 0, len(rows))
	for _, b := range rows {
		out = append(out, dto.GalgameRatingBucket{Score: b.Score, Count: b.Count})
	}
	return out
}

func catalogRatingStats(s *catRatingStats) *dto.GalgameRatingStats {
	if s == nil {
		return nil
	}
	return &dto.GalgameRatingStats{
		Average: s.Average, Stdev: s.Stdev, Min: s.Min, Max: s.Max,
	}
}

func legacyLandscapeCover(covers []dto.NextMoeGalgameCover) *dto.NextMoeGalgameCover {
	for i := range covers {
		c := &covers[i]
		if c.Width > 0 && c.Height > 0 && c.Height*20 <= c.Width*21 {
			return c
		}
	}
	return nil
}

func catalogTagCategory(kind string, sexual bool) string {
	if sexual {
		return "sexual"
	}
	return kind
}

// catalogAliases lists every other title the work is known by. titles[] is the
// full row set rather than the four slots the old election squeezed them into,
// so a Korean or untagged title now reaches the reader instead of vanishing.
func catalogAliases(d *catWorkDetail, rendered string) []dto.NextMoeAlias {
	out := []dto.NextMoeAlias{}
	seen := map[string]bool{rendered: true, d.DisplayName: true, "": true}
	for _, t := range d.Titles {
		if seen[t.Title] {
			continue
		}
		seen[t.Title] = true
		out = append(out, dto.NextMoeAlias{Name: t.Title})
	}
	return out
}

func (d *catWorkDetail) introRows() []CatalogIntro {
	if len(d.Intros) > 0 {
		return d.Intros
	}
	return d.Intro
}

func catalogRosterToNextMoe(ctx context.Context, chars []catWorkCharacter) []dto.NextMoeGalgameCharacter {
	out := make([]dto.NextMoeGalgameCharacter, 0, len(chars))
	for _, c := range chars {
		if c.DisplayName == "" && c.Latin == "" {
			continue
		}
		voices := make([]dto.NextMoeCharacterVoice, 0, len(c.Voices))
		for _, v := range c.Voices {
			voices = append(voices, dto.NextMoeCharacterVoice{ID: int(v.ID), Name: v.Name(ctx)})
		}
		name, original := CatalogEntityNames(ctx, c.Localized, c.DisplayName, c.Latin)
		out = append(out, dto.NextMoeGalgameCharacter{
			ID: int(c.ID), Name: name, NameOriginal: original, Latin: c.Latin,
			Kind: c.Kind, Spoiler: c.Spoiler, Identity: c.Identity,
			Image: c.Image, Figure: c.Figure,
			ImageMeta: ArtMetaDTO(c.ImageMeta), FigureMeta: ArtMetaDTO(c.FigureMeta),
			Voices: voices,
		})
	}
	return out
}

func catalogCoversToNextMoe(d *catWorkDetail) []dto.NextMoeGalgameCover {
	out := make([]dto.NextMoeGalgameCover, 0, len(d.Covers))
	for i, c := range d.Covers {
		hash := c.Hash
		if hash == "" {
			hash = hashFromURL(c.URL)
		}
		out = append(out, dto.NextMoeGalgameCover{
			ID: c.ID, ImageHash: hash, SortOrder: i,
			Sexual: c.Sexual, Violence: c.Violence, Kind: c.Kind, Source: c.Source,
			CDNURL: c.URL, Width: c.Width, Height: c.Height, Thumbhash: c.Thumbhash,
			VoteCount: c.VoteCount,
		})
	}
	return out
}

func catalogScreenshotsToNextMoe(d *catWorkDetail) []dto.NextMoeGalgameScreenshot {
	out := make([]dto.NextMoeGalgameScreenshot, 0, len(d.Screenshots))
	for i, s := range d.Screenshots {
		out = append(out, dto.NextMoeGalgameScreenshot{
			ImageHash: hashFromURL(s.URL), SortOrder: i, Caption: s.Caption,
			Sexual: s.Sexual, Violence: s.Violence, Source: s.Source,
			CDNURL: s.URL, Width: s.Width, Height: s.Height, Thumbhash: s.Thumbhash,
		})
	}
	return out
}

type GalgameLink struct {
	Name   string `json:"name"`
	Link   string `json:"link"`
	Source string `json:"source"`
}

func (c *GalgameClient) CatalogWorkLinks(ctx context.Context, gid int) ([]GalgameLink, *errors.AppError) {
	d, found, appErr := c.CatalogWorkDetail(ctx, gid)
	if appErr != nil {
		return nil, appErr
	}
	out := []GalgameLink{}
	if !found {
		return out, nil
	}
	for _, l := range d.Links {
		out = append(out, GalgameLink{Name: linkDisplayName(l), Link: l.URL, Source: l.Source})
	}
	return out, nil
}

func linkDisplayName(l catWorkLink) string {
	return LinkDisplayName(l.Source, l.URL)
}

func appendUniqueStr(slice []string, val string) []string {
	if val == "" {
		return slice
	}
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}
