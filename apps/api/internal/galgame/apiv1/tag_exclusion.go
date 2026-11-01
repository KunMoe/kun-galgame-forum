package apiv1

import (
	"context"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/workrepr"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/problem"
)

type tagExcluder interface {
	ExcludedWorkIDs(ctx context.Context, e client.TagExclusion) ([]int, *legacyErrors.AppError)
}

func (s *Service) excludedWorkIDs(ctx context.Context, e client.TagExclusion) ([]int, *problem.Problem) {
	if e.Empty() {
		return nil, nil
	}
	x, _ := s.works.(tagExcluder)
	if x == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	ids, appErr := x.ExcludedWorkIDs(ctx, e)
	if appErr != nil {
		return nil, catalogUnavailable(appErr)
	}
	return ids, nil
}

type entityResolver interface {
	EntityWorkIDs(ctx context.Context, tagIDs []int64, companyID int64) ([]int, bool, *legacyErrors.AppError)
}

func (s *Service) entityWorkIDs(ctx context.Context, in workrepr.EntityFilterInput) ([]int, bool, *problem.Problem) {
	tags, company := in.Tags(), in.Company()
	if len(tags) == 0 && company == 0 {
		return nil, false, nil
	}
	r, _ := s.works.(entityResolver)
	if r == nil {
		return nil, false, problem.Unavailable(errUnconfigured)
	}
	ids, ok, appErr := r.EntityWorkIDs(ctx, tags, company)
	if appErr != nil {
		return nil, false, catalogUnavailable(appErr)
	}
	return ids, ok, nil
}
