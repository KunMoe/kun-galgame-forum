package client

import (
	"context"
	"encoding/json"
	"net/url"

	"kun-galgame-api/pkg/errors"
)

type CatalogRankingConsensus struct {
	Score       float64  `json:"score"`
	Lower       float64  `json:"lower"`
	Upper       float64  `json:"upper"`
	Percentile  float64  `json:"percentile"`
	SourceCount int      `json:"source_count"`
	VoteCount   int      `json:"vote_count"`
	Divergence  *float64 `json:"divergence"`
}

type CatalogRankingSource struct {
	Source     string   `json:"source"`
	Score      float64  `json:"score"`
	VoteCount  int      `json:"vote_count"`
	Mean       float64  `json:"mean"`
	Lower      float64  `json:"lower"`
	Upper      float64  `json:"upper"`
	Median     *float64 `json:"median"`
	Spread     *float64 `json:"spread"`
	Percentile float64  `json:"percentile"`
}

type CatalogRankingEntry struct {
	Rank      int                      `json:"rank"`
	Value     float64                  `json:"value"`
	Work      CatalogWorkListItem      `json:"work"`
	Consensus *CatalogRankingConsensus `json:"consensus"`
	PerSource []CatalogRankingSource   `json:"per_source"`
}

type CatalogRatingScale struct {
	Source   string  `json:"source"`
	ScaleMin float64 `json:"scale_min"`
	ScaleMax float64 `json:"scale_max"`
}

type CatalogRankingPage struct {
	Facets catFacets             `json:"facets"`
	Items  []CatalogRankingEntry `json:"items"`
	Total  int64                 `json:"total"`
	Model  struct {
		FittedAt string               `json:"fitted_at"`
		Sources  []CatalogRatingScale `json:"sources"`
	} `json:"model"`
}

func (c *GalgameClient) CatalogRankings(ctx context.Context, q url.Values) (*CatalogRankingPage, *errors.AppError) {
	data, appErr := c.CatalogGet(ctx, "/catalog/rankings", q)
	if appErr != nil {
		return nil, appErr
	}
	var page CatalogRankingPage
	if err := json.Unmarshal(data, &page); err != nil {
		return nil, errors.ErrInternal("解析 Catalog 排行响应失败")
	}
	return &page, nil
}
