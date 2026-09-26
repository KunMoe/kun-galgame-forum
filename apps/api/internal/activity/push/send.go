package push

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"kun-galgame-api/pkg/communityclient"
)

func (p *Pusher) sendItems(ctx context.Context, items []communityclient.ActivityWriteItem, pending []prepared, rev int64) ([]Claim, error) {
	if len(items) == 0 {
		return nil, nil
	}
	out, err := p.community.WriteActivities(ctx, items)
	if err != nil {
		if isUnprocessable(err) {
			return p.bisect(ctx, items, pending, rev, err)
		}
		return nil, err
	}
	return p.applyOutcomes(out, pending, rev)
}

func (p *Pusher) bisect(ctx context.Context, items []communityclient.ActivityWriteItem, pending []prepared, rev int64, cause error) ([]Claim, error) {
	if len(items) == 1 {
		slog.Error("activity push: item rejected by community", "key", items[0].Key, "error", cause)
		return []Claim{pending[0].claim}, nil
	}
	mid := len(items) / 2
	left, err := p.sendItems(ctx, items[:mid], pending[:mid], rev)
	if err != nil {
		return nil, err
	}
	right, err := p.sendItems(ctx, items[mid:], pending[mid:], rev)
	if err != nil {
		return nil, err
	}
	return append(left, right...), nil
}

func (p *Pusher) applyOutcomes(resp *communityclient.ActivityWriteResponse, pending []prepared, rev int64) ([]Claim, error) {
	if resp == nil {
		return nil, errors.New("activity push: empty write response")
	}
	if len(resp.Results) != len(pending) {
		return nil, fmt.Errorf("activity push: %d results for %d items", len(resp.Results), len(pending))
	}
	handled := make([]Claim, 0, len(pending))
	for i, prep := range pending {
		handled = append(handled, prep.claim)
		r := resp.Results[i]
		switch r.Outcome {
		case "created", "updated", "removed", "restored":
			if err := p.upsertSent(Key{Type: prep.claim.Type, SourceID: prep.claim.SourceID},
				int(prep.item.ActorID), rev, prep.item.Removed); err != nil {
				return nil, err
			}
		case "stale":
		case "invalid":
			slog.Warn("activity push: community marked item invalid", "key", r.Key, "reason", r.Reason)
		}
	}
	return handled, nil
}

func isUnprocessable(err error) bool {
	var api *communityclient.APIError
	return errors.As(err, &api) && api.Status == 422
}
