package apiv1

import (
	"context"

	"kun-galgame-api/internal/activity/repository"
	"kun-galgame-api/pkg/problem"
)

func (s *Service) AssembleForPush(ctx context.Context, rows []repository.FeedRow) ([]*Activity, *problem.Problem) {
	return s.assemble(ctx, rows, true)
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
