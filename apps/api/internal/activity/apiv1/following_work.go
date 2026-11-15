package apiv1

import (
	"context"
	"log/slog"

	"kun-galgame-api/internal/activity/repository"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/problem"
)

type itemExtras struct {
	times   map[int]repository.TopicTimesRow
	works   map[int]content.WorkPreview
	targets map[objectRef]content.EntityTarget
}

func (s *Service) itemExtras(ctx context.Context, items []communityclient.ActivityItemView, includeNSFW bool) (itemExtras, *problem.Problem) {
	times, prob := s.topicTimes(items)
	if prob != nil {
		return itemExtras{}, prob
	}
	return itemExtras{times: times, works: s.workPreviews(ctx, items, includeNSFW), targets: s.objectTargets(ctx, items)}, nil
}

// A work preview only decorates an item, so a catalog outage leaves the items
// without one instead of failing the feed.
func (s *Service) workPreviews(ctx context.Context, items []communityclient.ActivityItemView, includeNSFW bool) map[int]content.WorkPreview {
	if s.catalog == nil {
		return nil
	}
	workIDs := ids{}
	for _, it := range items {
		if id := itemWorkID(it); id > 0 {
			workIDs.add(id)
		}
	}
	if len(workIDs) == 0 {
		return nil
	}
	rows, prob := s.catalogRows(ctx, workIDs.list(), !includeNSFW, previewInclude)
	if prob != nil {
		slog.Warn("activity: work previews unavailable", "err", prob)
		return nil
	}
	out := make(map[int]content.WorkPreview, len(rows))
	for id, row := range rows {
		if client.CatalogItemRenderable(&row) {
			out[id] = content.NewWorkPreview(ctx, &row, s.cdn)
		}
	}
	return out
}

func (x itemExtras) work(it communityclient.ActivityItemView) *content.WorkPreview {
	if p, ok := x.works[itemWorkID(it)]; ok {
		return &p
	}
	return nil
}
