package client

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"sync"

	"kun-galgame-api/pkg/errors"
)

type CatalogTaxonomyItem struct {
	ID          int64                       `json:"id"`
	Name        string                      `json:"name"`
	DisplayName string                      `json:"display_name"`
	Latin       string                      `json:"latin"`
	Localized   map[string]catLocalizedName `json:"localized"`
	Kind        string                      `json:"kind"`
	Tier        string                      `json:"tier"`
	WorkCount   int                         `json:"work_count"`
	Description string                      `json:"description"`
	Aliases     catAliases                  `json:"aliases"`
	Sexual      bool                        `json:"sexual"`
	HasNSFW     *bool                       `json:"has_nsfw"`
	LogoHash    string                      `json:"logo_hash"`
}

const TagTierHidden = "hidden"

type CatalogTaxonomyPage struct {
	Items      []CatalogTaxonomyItem `json:"items"`
	NextCursor *string               `json:"next_cursor"`
	Total      int64                 `json:"total"`
}

type CatalogLabelDetail struct {
	ID          int64                       `json:"id"`
	DisplayName string                      `json:"display_name"`
	Latin       string                      `json:"latin"`
	Localized   map[string]catLocalizedName `json:"localized"`
	Kind        string                      `json:"kind"`
	Lang        string                      `json:"lang"`
	Aliases     catAliases                  `json:"aliases"`
	WorkCount   int                         `json:"work_count"`
	Intros      []CatalogIntro              `json:"intros"`
	Links       []CatalogLabelLink          `json:"links"`
	LogoHash    string                      `json:"logo_hash"`
}

type CatalogLabelLink struct {
	Source string `json:"source"`
	URL    string `json:"url"`
}

type CatalogTagDetail struct {
	ID          int64                       `json:"id"`
	Name        string                      `json:"name"`
	DisplayName string                      `json:"display_name"`
	Localized   map[string]catLocalizedName `json:"localized"`
	Tier        string                      `json:"tier"`
	Kind        string                      `json:"kind"`
	WorkCount   int                         `json:"work_count"`
	Sexual      bool                        `json:"sexual"`
	Intros      []CatalogIntro              `json:"intros"`
}

type CatalogEngineDetail struct {
	ID          int64                       `json:"id"`
	Name        string                      `json:"name"`
	DisplayName string                      `json:"display_name"`
	Localized   map[string]catLocalizedName `json:"localized"`
	Description string                      `json:"description"`
	Aliases     catAliases                  `json:"aliases"`
	WorkCount   int                         `json:"work_count"`
}

type CatalogSeriesDetail struct {
	ID          int64                       `json:"id"`
	HasNSFW     *bool                       `json:"has_nsfw"`
	Name        string                      `json:"name"`
	DisplayName string                      `json:"display_name"`
	WorkCount   int                         `json:"work_count"`
	Localized   map[string]catLocalizedName `json:"localized"`
	Intros      []CatalogIntro              `json:"intros"`
}

func (c *GalgameClient) CatalogTaxonomyList(ctx context.Context, entity string, q url.Values) (*CatalogTaxonomyPage, *errors.AppError) {
	if q == nil {
		q = url.Values{}
	}
	if q.Get("include_total") == "" {
		q.Set("include_total", "true")
	}
	data, appErr := c.CatalogGet(ctx, "/catalog/"+entity, q)
	if appErr != nil {
		return nil, appErr
	}
	var page CatalogTaxonomyPage
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, errors.ErrInternal("解析 Catalog 词表响应失败")
	}
	return &page, nil
}

func (c *GalgameClient) CatalogLabel(ctx context.Context, id string) (*CatalogLabelDetail, bool, int64, *errors.AppError) {
	var rec CatalogLabelDetail
	found, movedTo, appErr := c.catalogTaxonomyDetail(ctx, "labels", id, &rec)
	return &rec, found, movedTo, appErr
}

func (c *GalgameClient) CatalogTag(ctx context.Context, id string) (*CatalogTagDetail, bool, *errors.AppError) {
	var rec CatalogTagDetail
	found, _, appErr := c.catalogTaxonomyDetail(ctx, "tags", id, &rec)
	return &rec, found, appErr
}

func (c *GalgameClient) CatalogEngine(ctx context.Context, id string) (*CatalogEngineDetail, bool, *errors.AppError) {
	var rec CatalogEngineDetail
	found, _, appErr := c.catalogTaxonomyDetail(ctx, "engines", id, &rec)
	return &rec, found, appErr
}

func (c *GalgameClient) CatalogSeries(ctx context.Context, id string) (*CatalogSeriesDetail, bool, *errors.AppError) {
	var rec CatalogSeriesDetail
	found, _, appErr := c.catalogTaxonomyDetail(ctx, "series", id, &rec)
	return &rec, found, appErr
}

func (c *GalgameClient) catalogTaxonomyDetail(ctx context.Context, entity, id string, out any) (bool, int64, *errors.AppError) {
	q := openPopulation(url.Values{})
	data, found, movedTo, appErr := c.catalogGetRecord(ctx, "/catalog/"+entity+"/"+id, q)
	if appErr != nil || !found {
		return false, movedTo, appErr
	}
	if err := json.Unmarshal(data, out); err != nil {
		return false, 0, errors.ErrInternal("解析 Catalog 词表详情响应失败")
	}
	return true, 0, nil
}

type CatalogEntityHit struct {
	ID          int64                       `json:"id"`
	EntityType  string                      `json:"entity_type"`
	DisplayName string                      `json:"display_name"`
	Latin       string                      `json:"latin"`
	Localized   map[string]catLocalizedName `json:"localized"`
	Tier        string                      `json:"tier"`
	Kind        string                      `json:"kind"`
}

func (h *CatalogEntityHit) Name(ctx context.Context) string {
	return CatalogEntityName(ctx, h.Localized, h.DisplayName, h.Latin)
}

func (c *GalgameClient) CatalogEntitySearch(ctx context.Context, searchType, keywords string, page, limit int) ([]CatalogEntityHit, int64, *errors.AppError) {
	q := url.Values{
		"type":          {searchType},
		"q":             {keywords},
		"limit":         {strconv.Itoa(limit)},
		"include_total": {"true"},
	}
	if page > 1 {
		q.Set("page", strconv.Itoa(page))
	}
	openPopulation(q)
	data, appErr := c.CatalogGet(ctx, "/catalog/search", q)
	if appErr != nil {
		return nil, 0, appErr
	}
	var parsed struct {
		Items []CatalogEntityHit `json:"items"`
		Total int64              `json:"total"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, 0, errors.ErrInternal("解析 Catalog 实体搜索响应失败")
	}
	if parsed.Items == nil {
		parsed.Items = []CatalogEntityHit{}
	}
	return parsed.Items, parsed.Total, nil
}

func (c *GalgameClient) CatalogSexualTagIDs(ctx context.Context, ids []int) map[int]bool {
	flags, appErr := cachedBatch(
		ctx, &c.tagSexualMu, c.tagSexualCache, ids, false,
		func(missing []int) (map[int]bool, *errors.AppError) {
			return c.fetchSexualTagIDs(ctx, missing), nil
		},
	)
	if appErr != nil {
		return map[int]bool{}
	}
	return flags
}

func (c *GalgameClient) fetchSexualTagIDs(ctx context.Context, ids []int) map[int]bool {
	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		out = make(map[int]bool, len(ids))
	)
	for _, id := range ids {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rec, found, appErr := c.CatalogTag(ctx, strconv.Itoa(id))
			if appErr != nil || !found || !rec.Sexual {
				return
			}
			mu.Lock()
			out[id] = true
			mu.Unlock()
		}(id)
	}
	wg.Wait()
	return out
}

var labelSourceKeys = []string{"curated", "galgame_wiki"}

func (c *GalgameClient) LookupWikiLabel(ctx context.Context, wikiID int) (int64, bool, *errors.AppError) {
	for _, source := range labelSourceKeys {
		id, found, appErr := c.lookupLabelBySource(ctx, source, wikiID)
		if appErr != nil || found {
			return id, found, appErr
		}
	}
	return 0, false, nil
}

func (c *GalgameClient) lookupLabelBySource(ctx context.Context, source string, wikiID int) (int64, bool, *errors.AppError) {
	q := url.Values{
		"refs": {source + ":" + strconv.Itoa(wikiID)},
	}
	openPopulation(q)
	data, appErr := c.CatalogGet(ctx, "/catalog/labels", q)
	if appErr != nil {
		if appErr.StatusCode == 404 {
			return 0, false, nil
		}
		return 0, false, appErr
	}
	var parsed struct {
		Items   []CatalogTaxonomyItem `json:"items"`
		Missing []string              `json:"missing"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return 0, false, errors.ErrInternal("解析 Catalog 反查响应失败")
	}
	if len(parsed.Items) == 0 {
		return 0, false, nil
	}
	return parsed.Items[0].ID, true, nil
}
