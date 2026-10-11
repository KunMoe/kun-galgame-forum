package apiv1

import (
	"context"

	"kun-galgame-api/internal/galgame/client"
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
