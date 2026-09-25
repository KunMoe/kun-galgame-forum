package client

import (
	"context"
	"encoding/json"
	"net/url"

	"kun-galgame-api/pkg/errors"
)

const catalogTraitPageCap = 60

type catTraitRef struct {
	ID int64 `json:"id,string"`
}

type CatalogTrait struct {
	ID             int64                       `json:"id,string"`
	DisplayName    string                      `json:"display_name"`
	NameZh         string                      `json:"name_zh"`
	Localized      map[string]catLocalizedName `json:"localized"`
	Parents        []catTraitRef               `json:"parents"`
	GroupID        *int64                      `json:"group_id,string"`
	RootOrder      *int                        `json:"root_order"`
	Sexual         bool                        `json:"is_sexual"`
	Searchable     bool                        `json:"is_searchable"`
	Aliases        []string                    `json:"aliases"`
	Description    string                      `json:"description"`
	Intros         []CatalogIntro              `json:"intros"`
	CharacterCount int                         `json:"character_count"`
	// SFWCharacterCount is character_count under nsfw=false: a trait with an
	// adult descendant matches fewer characters there.
	SFWCharacterCount int `json:"-"`
}

func (t *CatalogTrait) ParentIDs() []int64 {
	out := make([]int64, 0, len(t.Parents))
	for _, p := range t.Parents {
		out = append(out, p.ID)
	}
	return out
}

// CatalogTraitVocabulary walks the whole character-trait vocabulary, adult
// traits included; the reader's gate is applied downstream.
func (c *GalgameClient) CatalogTraitVocabulary(ctx context.Context) ([]CatalogTrait, *errors.AppError) {
	all, appErr := walkTraits(ctx, c, url.Values{"nsfw": {"true"}, "view": {"full"}, "include": {"character_count"}})
	if appErr != nil {
		return nil, appErr
	}
	sfw, appErr := walkTraits(ctx, c, url.Values{"include": {"character_count"}, "fields": {"character_count"}})
	if appErr != nil {
		return nil, appErr
	}
	counts := make(map[int64]int, len(sfw))
	for _, t := range sfw {
		counts[t.ID] = t.CharacterCount
	}
	for i := range all {
		all[i].SFWCharacterCount = counts[all[i].ID]
	}
	return all, nil
}

func walkTraits(ctx context.Context, c *GalgameClient, base url.Values) ([]CatalogTrait, *errors.AppError) {
	out := make([]CatalogTrait, 0, 4096)
	cursor := ""
	for range catalogTraitPageCap {
		q := url.Values{"limit": {"100"}}
		for k, v := range base {
			q[k] = v
		}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var page struct {
			Items      []CatalogTrait `json:"items"`
			NextCursor *string        `json:"next_cursor"`
		}
		if appErr := c.getV2Native(ctx, "/v2/catalog/traits", q, &page); appErr != nil {
			return nil, appErr
		}
		out = append(out, page.Items...)
		if page.NextCursor == nil || *page.NextCursor == "" {
			return out, nil
		}
		cursor = *page.NextCursor
	}
	return out, nil
}

// getV2Native decodes a /v2 body as catalog wrote it. getV2Raw runs the v1
// compatibility rewriter, which turns string ids into numbers and renames keys.
func (c *GalgameClient) getV2Native(ctx context.Context, v2Path string, q url.Values, dst any) *errors.AppError {
	status, body, appErr := c.getV2(ctx, v2Path, q)
	if appErr != nil {
		return appErr
	}
	if status < 200 || status >= 300 {
		return c.v2Error(status, body)
	}
	if err := json.Unmarshal(body, dst); err != nil {
		return errors.ErrInternal("解析 Catalog 响应失败")
	}
	return nil
}
