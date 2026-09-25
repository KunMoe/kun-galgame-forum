package client

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"kun-galgame-api/pkg/errors"
)

type CatalogCharacterQuery struct {
	Q        string
	TraitIDs []int
	MatchAny bool
	Genders  []string
	Sort     string
	Page     int
	Limit    int
	NSFW     bool
}

type V2Image struct {
	URL       string  `json:"url"`
	Hash      string  `json:"hash"`
	Width     *int    `json:"width"`
	Height    *int    `json:"height"`
	Thumbhash *string `json:"thumbhash"`
	Sexual    *string `json:"sexual"`
}

type CatalogCharacterRow struct {
	ID              int64                       `json:"id,string"`
	DisplayName     string                      `json:"display_name"`
	Latin           *string                     `json:"latin"`
	Localized       map[string]catLocalizedName `json:"localized"`
	Image           *V2Image                    `json:"image"`
	WorkCount       int                         `json:"work_count"`
	MatchedTraitIDs []string                    `json:"matched_trait_ids"`
}

type CatalogCharacterPage struct {
	Items []CatalogCharacterRow
	Total int
}

// CatalogCharacterList always sends page=, which keeps every combination of
// filters on catalog's search-index lane; a request without it on the default
// sort falls to the SQL lane, whose order and freshness differ.
func (c *GalgameClient) CatalogCharacterList(ctx context.Context, in CatalogCharacterQuery) (*CatalogCharacterPage, *errors.AppError) {
	q := url.Values{
		"include": {"image,work_count"},
		"page":    {strconv.Itoa(in.Page)},
		"limit":   {strconv.Itoa(in.Limit)},
		"sort":    {in.Sort},
	}
	if in.NSFW {
		q.Set("nsfw", "true")
	}
	if in.Q != "" {
		q.Set("q", in.Q)
	}
	if len(in.TraitIDs) > 0 {
		ids := make([]string, len(in.TraitIDs))
		for i, id := range in.TraitIDs {
			ids[i] = strconv.Itoa(id)
		}
		q.Set("trait_id", strings.Join(ids, ","))
		if in.MatchAny {
			q.Set("trait_match", "any")
		}
	}
	if len(in.Genders) > 0 {
		q.Set("gender", strings.Join(in.Genders, ","))
	}
	var body struct {
		Items []CatalogCharacterRow `json:"items"`
		Total int                   `json:"total"`
	}
	if appErr := c.getV2Native(ctx, "/v2/catalog/characters", q, &body); appErr != nil {
		return nil, appErr
	}
	return &CatalogCharacterPage{Items: body.Items, Total: body.Total}, nil
}

func (r *CatalogCharacterRow) MatchedIDs() []int {
	out := make([]int, 0, len(r.MatchedTraitIDs))
	for _, raw := range r.MatchedTraitIDs {
		if id, err := strconv.Atoi(raw); err == nil {
			out = append(out, id)
		}
	}
	return out
}
