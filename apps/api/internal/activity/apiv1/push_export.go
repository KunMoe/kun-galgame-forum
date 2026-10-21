package apiv1

import (
	"context"

	"kun-galgame-api/internal/activity/repository"
	"kun-galgame-api/pkg/problem"
)

// AssembleForPush answers nil for a row tied to a work catalog will not render,
// so the reconcile withdraws it from the central feed and sends it again once
// the work is back.
func (s *Service) AssembleForPush(ctx context.Context, rows []repository.FeedRow) ([]*Activity, *problem.Problem) {
	acts, prob := s.assemble(ctx, rows, true)
	if prob != nil {
		return nil, prob
	}
	ids := make([]int64, len(rows))
	for i, row := range rows {
		ids[i] = row.RowID
	}
	hidden, err := s.repo.UnrenderedRowIDs(ids)
	if err != nil {
		return nil, problem.Internal(err)
	}
	for i, row := range rows {
		if hidden[row.RowID] {
			acts[i] = nil
		}
	}
	return acts, nil
}

type FeedTypePair struct {
	Kind string
	Feed string
}

func FeedTypePairs() []FeedTypePair {
	out := make([]FeedTypePair, len(feedTypes))
	for i, t := range feedTypes {
		out[i] = FeedTypePair{Kind: t.kind, Feed: t.feed}
	}
	return out
}

func KindOfFeed(feed string) string { return kindByFeed[feed] }
