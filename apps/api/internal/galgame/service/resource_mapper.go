package service

import (
	"encoding/json"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/resourcevocab"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/pkg/userclient"
)

func decodeProviderNames(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return []string{}
	}
	return out
}

func collectIDs(rows []model.GalgameResourceRow) (workIDs, userIDs []int) {
	workIDs = make([]int, 0, len(rows))
	userIDs = make([]int, 0, len(rows))
	for _, r := range rows {
		workIDs = append(workIDs, r.WorkID)
		userIDs = append(userIDs, r.UserID)
	}
	return
}

func collectAggregate(aggs []model.ResourceAggregate) (platforms, languages, types []string) {
	platforms, languages, types = []string{}, []string{}, []string{}
	for _, a := range aggs {
		if a.Platform != "" {
			platforms = appendUniqueStr(platforms, a.Platform)
		}
		if a.Language != "" {
			languages = appendUniqueStr(languages, a.Language)
		}
		if a.Type != "" {
			types = appendUniqueStr(types, a.Type)
		}
	}
	return
}

func appendUniqueStr(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}

func userBriefToDTO(u userclient.User) dto.UserBrief {
	return dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar}
}

func axesFromRow(r model.GalgameResourceRow) (langs, plats, runs []string) {
	langs, plats, runs = []string(r.Languages), []string(r.Platforms), []string(r.Runtimes)
	if len(langs) == 0 {
		langs = []string(resourcevocab.LegacyLanguage(r.Language))
	}
	if len(plats) == 0 && len(runs) == 0 {
		p, rt := resourcevocab.LegacyPlatform(r.Platform)
		plats, runs = []string(p), []string(rt)
	}
	return
}

func rowToCard(r model.GalgameResourceRow, u userclient.User, isLiked bool) dto.ResourceCard {
	langs, plats, runs := axesFromRow(r)
	return dto.ResourceCard{
		ID:            r.ID,
		View:          r.View,
		WorkID:        r.WorkID,
		User:          userBriefToDTO(u),
		Type:          r.Type,
		Title:         r.Title,
		VersionLabel:  r.VersionLabel,
		Language:      r.Language,
		Platform:      r.Platform,
		Languages:     langs,
		Platforms:     plats,
		Runtimes:      runs,
		Size:          r.Size,
		Status:        r.Status,
		Download:      r.Download,
		LikeCount:     r.LikeCount,
		IsLiked:       isLiked,
		CommentCount:  r.CommentCount,
		LinkDomain:    "",
		ProviderNames: decodeProviderNames(r.ProviderName),
		Note:          r.Note,
		NoteHtml:      markdown.RenderHardWrap(r.Note),
		Created:       r.Created,
		Edited:        r.Edited,
	}
}

func rowToMeta(
	r model.GalgameResourceRow,
	links []string,
	isLiked bool,
	owner userclient.User,
) dto.ResourceMeta {
	linkDomain := ""
	if len(links) > 0 {
		linkDomain = links[0]
	}
	langs, plats, runs := axesFromRow(r)
	return dto.ResourceMeta{
		ID:            r.ID,
		View:          r.View,
		WorkID:        r.WorkID,
		User:          userBriefToDTO(owner),
		Type:          r.Type,
		Title:         r.Title,
		VersionLabel:  r.VersionLabel,
		Language:      r.Language,
		Platform:      r.Platform,
		Languages:     langs,
		Platforms:     plats,
		Runtimes:      runs,
		Size:          r.Size,
		Status:        r.Status,
		Download:      r.Download,
		LikeCount:     r.LikeCount,
		IsLiked:       isLiked,
		CommentCount:  r.CommentCount,
		LinkDomain:    linkDomain,
		ProviderNames: decodeProviderNames(r.ProviderName),
		Note:          r.Note,
		NoteHtml:      markdown.RenderHardWrap(r.Note),
		Created:       r.Created,
		Edited:        r.Edited,
	}
}

func rowToDownloadDetail(
	r model.GalgameResourceRow,
	links []string,
	isLiked bool,
	owner userclient.User,
) dto.ResourceDownloadDetail {
	return dto.ResourceDownloadDetail{
		ResourceMeta: rowToMeta(r, links, isLiked, owner),
		Link:         links,
		Code:         r.Code,
		Password:     r.Password,
	}
}
