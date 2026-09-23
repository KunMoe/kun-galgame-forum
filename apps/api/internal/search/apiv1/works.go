package apiv1

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	"kun-galgame-api/internal/galgame/client"
	galgameService "kun-galgame-api/internal/galgame/service"
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
	Body repr.PageList[repr.WorkRef]
}

func (s *Service) searchWorks(ctx context.Context, in *worksInput) (*worksOutput, error) {
	if prob := s.ready(); prob != nil {
		return nil, prob
	}
	if s.galgame == nil {
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
		"include": {galgameService.CatalogCardInclude},
		"sort":    {worksSort[in.Sort]},
	}
	if in.CompanyID != "" {
		params.Set("company_id", in.CompanyID)
	}
	if len(in.TagIDs) > 0 {
		tags := make([]string, len(in.TagIDs))
		for i, id := range in.TagIDs {
			tags[i] = string(id)
		}
		params.Set("tag_id", strings.Join(tags, ","))
	}
	if from != "" {
		params.Set("released_after", from)
	}
	if to != "" {
		params.Set("released_before", to)
	}
	client.ApplyWorksGate(params, !in.IncludeNSFW)

	res, appErr := s.galgame.CatalogWorksSearch(ctx, params)
	if appErr != nil {
		return nil, problem.Unavailable(appErr)
	}
	items := make([]repr.WorkRef, 0, len(res.Items))
	for i := range res.Items {
		if !client.CatalogItemRenderable(&res.Items[i]) {
			continue
		}
		items = append(items, galgameapiv1.WorkRefOf(ctx, &res.Items[i], s.cdn))
	}
	total, relation := collect.ClampTotal(int(res.Total))
	return &worksOutput{Body: repr.NewPageList(items, total, relation)}, nil
}
