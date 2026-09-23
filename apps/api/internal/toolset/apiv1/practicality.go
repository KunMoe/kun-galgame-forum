package apiv1

import (
	"context"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"
)

func (s *Service) putToolsetPracticality(ctx context.Context, in *putPracticalityInput) (*practicalityOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	row, _, p := s.visibleToolset(ctx, in.ToolsetID)
	if p != nil {
		return nil, p
	}
	if err := s.store.UpsertPracticality(row.ID, user.ID, in.Body.Rating); err != nil {
		return nil, problem.Internal(err)
	}
	prac, err := s.store.Practicality([]int{row.ID})
	if err != nil {
		return nil, problem.Internal(err)
	}
	agg := prac[row.ID]
	dist := distOf(agg)
	if agg.Count == 0 {
		dist = emptyDist()
	}
	return &practicalityOutput{Body: ToolsetPracticality{
		Object: "toolset_practicality", ToolsetID: repr.ID(row.ID),
		PracticalityAverage: agg.Average, PracticalityCount: agg.Count, PracticalityDistribution: starCounts(dist),
		Viewer: &PracticalityViewer{PracticalityRating: &in.Body.Rating},
	}}, nil
}
