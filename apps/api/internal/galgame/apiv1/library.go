package apiv1

import (
	"context"
	"log/slog"
	"net/url"
	"strconv"
	"strings"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/workrepr"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/problem"
)

type worksSearch interface {
	CatalogWorksSearch(ctx context.Context, q url.Values) (*client.CatalogWorksPage, *legacyErrors.AppError)
}

func (s *Service) worksSearch() worksSearch {
	w, _ := s.works.(worksSearch)
	return w
}

func (s *Service) listLibraryWorks(ctx context.Context, in *listLibraryWorksInput) (*workSummaryPageOutput, error) {
	if s == nil || s.hydrator == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	search := s.worksSearch()
	if search == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	pg := browsePage(in.Page, in.Limit)
	if prob := pg.CheckDepth(); prob != nil {
		return nil, prob
	}
	q := strings.TrimSpace(in.Q)
	if in.Q != "" && q == "" {
		minLen := 1
		return nil, problem.New(problem.CodeInvalidParameter, "q is blank.",
			problem.AtParameter("q", problem.ReasonTooShort, "q must contain a non-space character",
				&problem.FieldParams{MinLength: &minLen}))
	}
	catalogSort, ok := libraryWorkSorts[string(in.Sort)]
	if !ok {
		if string(in.Sort) == "" {
			catalogSort = "popularity"
		} else {
			return nil, problem.New(problem.CodeUnknownSort, "The sort token is not in this collection's vocabulary.",
				problem.AtParameter("sort", problem.ReasonUnknownValue, "not in this collection's closed vocabulary", nil))
		}
	}
	from, to, prob := boundRange(in.ReleasedFrom, in.ReleasedTo, "released_from", "released_to", true)
	if prob != nil {
		return nil, prob
	}
	params := url.Values{
		"page":    {strconv.Itoa(pg.Page)},
		"limit":   {strconv.Itoa(pg.Limit)},
		"include": {workrepr.RowInclude},
		"sort":    {catalogSort},
	}
	if q != "" {
		params.Set("q", q)
	}
	if from != "" {
		params.Set("released_after", from)
	}
	if to != "" {
		params.Set("released_before", to)
	}
	client.ApplyWorksGate(params, !in.IncludeNSFW)
	res, appErr := search.CatalogWorksSearch(ctx, params)
	if appErr != nil {
		return nil, catalogUnavailable(appErr)
	}
	rows := make([]client.CatalogWorkListItem, 0, len(res.Items))
	for i := range res.Items {
		if !client.CatalogItemRenderable(&res.Items[i]) {
			slog.Warn("library-works: catalog row not renderable, dropped", "work_id", res.Items[i].ID)
			continue
		}
		rows = append(rows, res.Items[i])
	}
	items, p := s.hydrator.FromRows(ctx, rows)
	if p != nil {
		return nil, p
	}
	n, rel := collect.ClampTotal(int(res.Total))
	return &workSummaryPageOutput{Body: repr.NewPageList(items, n, rel)}, nil
}
