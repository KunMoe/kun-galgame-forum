package apiv1

import (
	"context"
	"net/url"
	"strconv"

	"kun-galgame-api/internal/apiv1/repr"
	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/problem"
)

const suggestionLimit = 12

func (s *Service) listWorkSuggestions(ctx context.Context, in *listWorkSuggestionsInput) (*listWorkSuggestionsOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	if len(in.Q) > 100 {
		return nil, validationFailed(problem.AtParameter("q", problem.ReasonTooLong, "longer than the field allows", &problem.FieldParams{MaxLength: intPtr(100)}))
	}
	qtext := trimSpace(in.Q)
	if qtext == "" {
		return nil, validationFailed(problem.AtParameter("q", problem.ReasonTooShort, "too short once surrounding whitespace is removed", &problem.FieldParams{MinLength: intPtr(1)}))
	}
	cat := s.catalog()
	if cat == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	q := url.Values{
		"q":       {qtext},
		"limit":   {strconv.Itoa(suggestionLimit)},
		"sort":    {"relevance"},
		"include": {"names,covers"},
	}
	client.ApplyWorksGate(q, !in.IncludeNSFW)
	res, appErr := cat.CatalogWorksSearch(ctx, q)
	if appErr != nil {
		return nil, problem.Unavailable(appErr)
	}
	items := make([]repr.WorkRef, 0, suggestionLimit)
	for i := range res.Items {
		if !client.CatalogItemRenderable(&res.Items[i]) {
			continue
		}
		if res.Items[i].ID <= 0 {
			continue
		}
		it := res.Items[i]
		items = append(items, galgameapiv1.WorkRefOf(ctx, &it, s.cdn))
		if len(items) >= suggestionLimit {
			break
		}
	}
	return &listWorkSuggestionsOutput{Body: repr.NewList(items, nil)}, nil
}
