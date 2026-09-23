package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/pkg/userclient"
)

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

func groupResourceMeta(rows []model.GalgameResourceMeta) (platforms, languages map[int][]string) {
	platforms = make(map[int][]string)
	languages = make(map[int][]string)
	for _, r := range rows {
		if r.Platform != "" {
			platforms[r.WorkID] = appendUniqueStr(platforms[r.WorkID], r.Platform)
		}
		if r.Language != "" {
			languages[r.WorkID] = appendUniqueStr(languages[r.WorkID], r.Language)
		}
	}
	return
}

func frozenCreatorBrief(row repository.GalgameLocalRow, userMap map[int]userclient.User) dto.UserBrief {
	id := userclient.DerefID(row.CreatorUserID)
	if id <= 0 {
		return dto.UserBrief{}
	}
	return userBriefToDTO(userMap[id])
}

func frozenCreatorIDs(ids []int, localMap map[int]repository.GalgameLocalRow) []int {
	return userclient.CollectIDs(ids, func(id int) int {
		return userclient.DerefID(localMap[id].CreatorUserID)
	})
}

// renderIntros renders each language's markdown in place. The rows arrive in
// the order the site presents them, so the first one is also what IntroText
// takes — but IntroText needs the source markdown, not this.
func renderIntros(rows []dto.GalgameIntro) []dto.GalgameIntro {
	out := make([]dto.GalgameIntro, 0, len(rows))
	for _, r := range rows {
		r.Intro = markdown.Render(r.Intro)
		out = append(out, r)
	}
	return out
}

func firstIntroText(rows []dto.GalgameIntro) string {
	if len(rows) == 0 {
		return ""
	}
	return rows[0].Intro
}

func galgameDetailFromNextMoe(g dto.NextMoeGalgameDetailFull, users map[string]dto.NextMoeUser) dto.GalgameDetail {
	return dto.GalgameDetail{
		ID:                         g.ID,
		VndbID:                     g.VndbID,
		User:                       lookupNextMoeUser(users, g.UserID),
		Name:                       g.Name,
		NameOriginal:               g.NameOriginal,
		Introduction:               renderIntros(g.Intros),
		IntroText:                  firstIntroText(g.Intros),
		ContentLimit:               g.ContentLimit,
		Status:                     g.Status,
		OriginalLanguage:           g.OriginalLanguage,
		AgeLimit:                   g.AgeLimit,
		ReleaseDate:                g.ReleaseDate,
		ReleaseDateTBA:             g.ReleaseDateTBA,
		EffectiveBannerHash:        g.EffectiveBannerHash,
		EffectiveBannerURL:         g.EffectiveBannerURL,
		EffectiveBannerWidth:       g.EffectiveBannerWidth,
		EffectiveBannerHeight:      g.EffectiveBannerHeight,
		EffectiveBannerThumbhash:   g.EffectiveBannerThumbhash,
		EffectivePortraitHash:      g.EffectivePortraitHash,
		EffectivePortraitURL:       g.EffectivePortraitURL,
		EffectivePortraitWidth:     g.EffectivePortraitWidth,
		EffectivePortraitHeight:    g.EffectivePortraitHeight,
		EffectivePortraitThumbhash: g.EffectivePortraitThumbhash,
		ExternalRatings:            sliceOrEmpty(g.ExternalRatings),
		Playtimes:                  sliceOrEmpty(g.Playtimes),
		Refs:                       g.Refs,
		Covers:                     coversFromNextMoe(g.Covers),
		Screenshots:                screenshotsFromNextMoe(g.Screenshots),
		Contributor:                contributorsFromNextMoe(g.Contributor, users),
		Alias:                      nextMoeAliasesToNames(g.Alias),
		Engine:                     enginesFromNextMoe(g.Engine),
		Official:                   officialsFromNextMoe(g.Official),
		Series:                     seriesFromNextMoe(g.Series),
		Tag:                        tagsFromNextMoe(g.Tag),
		Staff:                      staffFromNextMoe(g.Staff),
		Characters:                 charactersFromNextMoe(g.Characters),
		Created:                    g.Created,
		Updated:                    g.Updated,
	}
}

func lookupNextMoeUser(users map[string]dto.NextMoeUser, userID int) dto.UserBrief {
	if u, ok := users[fmt.Sprintf("%d", userID)]; ok {
		return dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar}
	}
	return dto.UserBrief{ID: userID}
}

func sliceOrEmpty[T any](rows []T) []T {
	if rows == nil {
		return []T{}
	}
	return rows
}

func coversFromNextMoe(rows []dto.NextMoeGalgameCover) []dto.GalgameCover {
	out := make([]dto.GalgameCover, len(rows))
	for i, r := range rows {
		out[i] = dto.GalgameCover{
			ImageHash: r.ImageHash, SortOrder: r.SortOrder,
			Sexual: r.Sexual, Violence: r.Violence,
			Source: r.Source, SourceKey: r.SourceKey,
			Kind:      r.Kind,
			CDNURL:    r.CDNURL,
			Width:     r.Width,
			Height:    r.Height,
			Thumbhash: r.Thumbhash,
			ID:        r.ID,
			VoteCount: r.VoteCount,
		}
	}
	return out
}

func screenshotsFromNextMoe(rows []dto.NextMoeGalgameScreenshot) []dto.GalgameScreenshot {
	out := make([]dto.GalgameScreenshot, len(rows))
	for i, r := range rows {
		out[i] = dto.GalgameScreenshot{
			ImageHash: r.ImageHash, SortOrder: r.SortOrder, Caption: r.Caption,
			Sexual: r.Sexual, Violence: r.Violence,
			Source: r.Source, SourceKey: r.SourceKey,
			CDNURL:    r.CDNURL,
			Width:     r.Width,
			Height:    r.Height,
			Thumbhash: r.Thumbhash,
		}
	}
	return out
}

func contributorsFromNextMoe(contribs []dto.NextMoeContributor, users map[string]dto.NextMoeUser) []dto.UserBrief {
	out := make([]dto.UserBrief, len(contribs))
	for i, c := range contribs {
		out[i] = lookupNextMoeUser(users, c.UserID)
	}
	return out
}

func nextMoeAliasesToNames(aliases []dto.NextMoeAlias) []string {
	out := make([]string, len(aliases))
	for i, a := range aliases {
		out[i] = a.Name
	}
	return out
}

func enginesFromNextMoe(engines []dto.NextMoeEngineWithAlias) []dto.GalgameDetailEngine {
	out := make([]dto.GalgameDetailEngine, len(engines))
	for i, e := range engines {
		alias := e.Engine.Alias
		if alias == nil {
			alias = []string{}
		}
		out[i] = dto.GalgameDetailEngine{
			ID:           e.Engine.ID,
			Name:         e.Engine.Name,
			Alias:        alias,
			GalgameCount: e.Engine.GalgameCount,
		}
	}
	return out
}

func seriesFromNextMoe(refs []dto.NextMoeSeriesRef) []dto.GalgameDetailSeries {
	out := make([]dto.GalgameDetailSeries, len(refs))
	for i, r := range refs {
		out[i] = dto.GalgameDetailSeries{ID: r.ID, Name: r.Name}
	}
	return out
}

func officialsFromNextMoe(rels []dto.NextMoeOfficialRel) []dto.GalgameDetailOfficial {
	out := make([]dto.GalgameDetailOfficial, len(rels))
	for i, rel := range rels {
		out[i] = dto.GalgameDetailOfficial{
			ID:           rel.Official.ID,
			Name:         rel.Official.Name,
			Link:         rel.Official.Link,
			Category:     rel.Official.Category,
			Roles:        emptyStrSliceIfNil(rel.Official.Roles),
			Lang:         rel.Official.Lang,
			Alias:        nextMoeAliasesToNames(rel.Official.Alias),
			GalgameCount: rel.Official.GalgameCount,
		}
	}
	return out
}

func tagsFromNextMoe(tags []dto.NextMoeTagWithSpoiler) []dto.GalgameDetailTag {
	out := make([]dto.GalgameDetailTag, len(tags))
	for i, t := range tags {
		out[i] = dto.GalgameDetailTag{
			ID:           t.Tag.ID,
			Name:         t.Tag.Name,
			Category:     t.Tag.Category,
			SpoilerLevel: t.SpoilerLevel,
			GalgameCount: t.Tag.GalgameCount,
		}
	}
	return out
}

func staffFromNextMoe(groups []dto.NextMoeStaffGroup) []dto.GalgameDetailStaff {
	out := make([]dto.GalgameDetailStaff, len(groups))
	for i, g := range groups {
		people := make([]dto.GalgameDetailStaffName, len(g.People))
		for j, p := range g.People {
			people[j] = dto.GalgameDetailStaffName{
				ID: p.ID, Name: p.Name, Latin: p.Latin, Characters: p.Characters,
			}
		}
		out[i] = dto.GalgameDetailStaff{RoleKey: g.RoleKey, RoleName: g.RoleName, People: people}
	}
	return out
}

func charactersFromNextMoe(chars []dto.NextMoeGalgameCharacter) []dto.GalgameDetailCharacter {
	out := make([]dto.GalgameDetailCharacter, len(chars))
	for i, c := range chars {
		voices := make([]dto.GalgameDetailCharacterVoice, len(c.Voices))
		for j, v := range c.Voices {
			voices[j] = dto.GalgameDetailCharacterVoice{ID: v.ID, Name: v.Name}
		}
		out[i] = dto.GalgameDetailCharacter{
			ID: c.ID, Name: c.Name, NameOriginal: c.NameOriginal, Latin: c.Latin,
			Kind: c.Kind, Spoiler: c.Spoiler, Identity: c.Identity,
			Image: c.Image, Figure: c.Figure,
			ImageMeta: c.ImageMeta, FigureMeta: c.FigureMeta,
			Voices: voices,
		}
	}
	return out
}

func withoutSexualTags(tags []dto.GalgameDetailTag) []dto.GalgameDetailTag {
	out := make([]dto.GalgameDetailTag, 0, len(tags))
	for _, t := range tags {
		if t.Category == "sexual" {
			continue
		}
		out = append(out, t)
	}
	return out
}

func detailRatingFromRow(
	r repository.GalgameDetailRatingRow,
	user userclient.User,
	isLiked bool,
	workID int,
	g dto.NextMoeGalgameDetailFull,
) dto.GalgameDetailRating {
	return dto.GalgameDetailRating{
		ID:           r.ID,
		User:         userBriefToDTO(user),
		Recommend:    r.Recommend,
		Overall:      r.Overall,
		View:         r.View,
		GalgameType:  rawJSON(r.GalgameType),
		PlayStatus:   r.PlayStatus,
		ShortSummary: r.ShortSummary,
		SpoilerLevel: r.SpoilerLevel,
		Art:          r.Art,
		Story:        r.Story,
		Music:        r.Music,
		Character:    r.Character,
		Route:        r.Route,
		System:       r.System,
		Voice:        r.Voice,
		ReplayValue:  r.ReplayValue,
		LikeCount:    r.LikeCount,
		IsLiked:      isLiked,
		WorkID:       workID,
		Created:      r.Created,
		Updated:      r.Updated,
		Galgame: dto.GalgameDetailRatingGalgame{
			ID:           g.ID,
			ContentLimit: g.ContentLimit,
			Name:         g.Name,
		},
	}
}

func emptyStrSliceIfNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func rawJSON(s string) json.RawMessage {
	if s == "" {
		return json.RawMessage("[]")
	}
	return json.RawMessage(s)
}
