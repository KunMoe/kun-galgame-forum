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
	ID              int64                       `json:"id"`
	Name            string                      `json:"name"`
	DisplayName     string                      `json:"display_name"`
	Latin           string                      `json:"latin"`
	Localized       map[string]catLocalizedName `json:"localized"`
	Kind            string                      `json:"kind"`
	Tier            string                      `json:"tier"`
	WorkCount       int                         `json:"work_count"`
	RaterCount      int64                       `json:"rater_count"`
	RatedWorkCount  int                         `json:"rated_work_count"`
	Description     string                      `json:"description"`
	Aliases         catAliases                  `json:"aliases"`
	Sexual          bool                        `json:"sexual"`
	HasNSFW         *bool                       `json:"has_nsfw"`
	LogoHash        string                      `json:"logo_hash"`
	DescendantCount int                         `json:"descendant_count"`
}

const TagTierHidden = "hidden"

type CatalogTaxonomyPage struct {
	Items      []CatalogTaxonomyItem `json:"items"`
	NextCursor *string               `json:"next_cursor"`
	Total      int64                 `json:"total"`
}

type CatalogLabelDetail struct {
	ID            int64                       `json:"id"`
	DisplayName   string                      `json:"display_name"`
	Latin         string                      `json:"latin"`
	Localized     map[string]catLocalizedName `json:"localized"`
	Kind          string                      `json:"kind"`
	Lang          string                      `json:"lang"`
	Aliases       catAliases                  `json:"aliases"`
	WorkCount     int                         `json:"work_count"`
	Intros        []CatalogIntro              `json:"intros"`
	Links         []CatalogLink               `json:"links"`
	LogoHash      string                      `json:"logo_hash"`
	FollowerCount *int                        `json:"follower_count"`
}

type CatalogTagDetail struct {
	ID              int64                       `json:"id"`
	Name            string                      `json:"name"`
	DisplayName     string                      `json:"display_name"`
	Localized       map[string]catLocalizedName `json:"localized"`
	Tier            string                      `json:"tier"`
	Kind            string                      `json:"kind"`
	WorkCount       int                         `json:"work_count"`
	Sexual          bool                        `json:"sexual"`
	Intros          []CatalogIntro              `json:"intros"`
	Parents         []CatalogTagNode            `json:"parents"`
	Children        []CatalogTagNode            `json:"children"`
	DescendantCount int                         `json:"descendant_count"`
	Ancestors       []CatalogTagNode            `json:"ancestors"`
}

type CatalogTagNode struct {
	ID          int64    `json:"id"`
	DisplayName string   `json:"display_name"`
	Sexual      bool     `json:"sexual"`
	ParentIDs   []string `json:"parent_ids"`
}

type CatalogTagFacts struct {
	Sexual          bool
	DescendantCount int
}

type CatalogEngineDetail struct {
	ID             int64                       `json:"id"`
	Name           string                      `json:"name"`
	DisplayName    string                      `json:"display_name"`
	Localized      map[string]catLocalizedName `json:"localized"`
	Description    string                      `json:"description"`
	Aliases        catAliases                  `json:"aliases"`
	WorkCount      int                         `json:"work_count"`
	RaterCount     int64                       `json:"rater_count"`
	RatedWorkCount int                         `json:"rated_work_count"`
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

func (c *GalgameClient) CatalogTagFacts(ctx context.Context, ids []int) map[int]CatalogTagFacts {
	facts, appErr := cachedBatch(
		ctx, &c.tagFactsMu, c.tagFactsCache, ids, false,
		func(missing []int) (map[int]CatalogTagFacts, *errors.AppError) {
			return c.fetchTagFacts(ctx, missing), nil
		},
	)
	if appErr != nil {
		return map[int]CatalogTagFacts{}
	}
	return facts
}

func (c *GalgameClient) fetchTagFacts(ctx context.Context, ids []int) map[int]CatalogTagFacts {
	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		out = make(map[int]CatalogTagFacts, len(ids))
	)
	for _, id := range ids {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			rec, found, appErr := c.CatalogTag(ctx, strconv.Itoa(id))
			if appErr != nil || !found {
				return
			}
			mu.Lock()
			out[id] = CatalogTagFacts{Sexual: rec.Sexual, DescendantCount: rec.DescendantCount}
			mu.Unlock()
		}(id)
	}
	wg.Wait()
	return out
}

func (c *GalgameClient) CatalogTagDescendants(ctx context.Context, id string) ([]CatalogTagNode, *errors.AppError) {
	var body struct {
		Descendants []struct {
			ID          string   `json:"id"`
			DisplayName string   `json:"display_name"`
			IsSexual    bool     `json:"is_sexual"`
			ParentIDs   []string `json:"parent_ids"`
		} `json:"descendants"`
	}
	if appErr := c.getV2Native(ctx, "/v2/catalog/tags/"+id, url.Values{"include": {"descendants"}}, &body); appErr != nil {
		return nil, appErr
	}
	out := make([]CatalogTagNode, 0, len(body.Descendants))
	for _, d := range body.Descendants {
		n, err := strconv.ParseInt(d.ID, 10, 64)
		if err != nil {
			continue
		}
		out = append(out, CatalogTagNode{ID: n, DisplayName: d.DisplayName, Sexual: d.IsSexual, ParentIDs: d.ParentIDs})
	}
	return out, nil
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
