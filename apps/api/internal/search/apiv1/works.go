package apiv1

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/utils"
)

var worksSort = map[string]string{
	"relevance_desc":  "relevance",
	"popularity_desc": "popularity",
	"updated_desc":    "updated",
	"released_desc":   "released_desc",
	"released_asc":    "released_asc",
}

type worksOutput struct {
	Body workrepr.FacetedWorkPage
}

func (s *Service) searchWorks(ctx context.Context, in *worksInput) (*worksOutput, error) {
	if prob := s.ready(); prob != nil {
		return nil, prob
	}
	if s.galgame == nil || s.works == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	q := strings.TrimSpace(in.Q)
	if _, prob := keywordsOf(q); prob != nil {
		return nil, prob
	}
	if prob := in.CheckDepth(); prob != nil {
		return nil, prob
	}
	from, _ := utils.ParseReleaseLowerBound(in.ReleasedFrom)
	to, _ := utils.ParseReleaseUpperBound(in.ReleasedTo)
	if from != "" && to != "" && from > to {
		return nil, problem.New(problem.CodeInvalidParameter, "released_from is after released_to.",
			problem.AtParameter("released_from", problem.ReasonInconsistentWith, "released_to", nil))
	}

	params := url.Values{
		"q":       {q},
		"page":    {strconv.Itoa(in.Page)},
		"limit":   {strconv.Itoa(in.Limit)},
		"include": {workrepr.RowInclude},
		"sort":    {worksSort[in.Sort]},
	}
	in.EntityFilterInput.Apply(params)
	params.Set("facets", client.EntityFacets)
	if from != "" {
		params.Set("released_after", from)
	}
	if to != "" {
		params.Set("released_before", to)
	}
	client.ApplyWorksGate(params, !in.IncludeNSFW)
	in.Exclusion().Apply(params)

	res, appErr := s.galgame.CatalogWorksSearch(ctx, params)
	if appErr != nil {
		return nil, problem.Unavailable(appErr)
	}
	rows := make([]client.CatalogWorkListItem, 0, len(res.Items))
	for i := range res.Items {
		if client.CatalogItemRenderable(&res.Items[i]) {
			rows = append(rows, res.Items[i])
		}
	}
	items, prob := s.works.FromRows(ctx, rows)
	if prob != nil {
		return nil, prob
	}
	total, relation := collect.ClampTotal(int(res.Total))
	facets := workrepr.NewWorkFacets(res.Facets.Tags(), res.Facets.Companies())
	return &worksOutput{Body: workrepr.NewFacetedWorkPage(items, total, relation, facets)}, nil
}
