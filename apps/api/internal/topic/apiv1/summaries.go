package apiv1

import (
	"context"

	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/problem"
)

// Summaries renders rows another face ranked itself, in the order given, with
// the rows of banned authors left out, exactly as the topic list renders them.
func (s *Service) Summaries(ctx context.Context, rows []repository.TopicKeysetRow) ([]TopicSummary, *problem.Problem) {
	if s == nil || s.topics == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	rendered, prob := s.summaries(ctx, rows)
	if prob != nil {
		return nil, prob
	}
	out := make([]TopicSummary, 0, len(rendered))
	for _, item := range rendered {
		if item != nil {
			out = append(out, *item)
		}
	}
	return out, nil
}
