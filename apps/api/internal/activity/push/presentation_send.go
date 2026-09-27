package push

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"kun-galgame-api/pkg/communityclient"
)

func (p *Presenter) sendPresentations(ctx context.Context, items []communityclient.AnchorPresentationItem, pending []preparedPresentation) ([]presentationClaim, error) {
	if len(items) == 0 {
		return nil, nil
	}
	out, err := p.community.WriteAnchorPresentations(ctx, items)
	if err != nil {
		if isUnprocessable(err) {
			return p.bisectPresentations(ctx, items, pending, err)
		}
		return nil, err
	}
	return p.applyPresentationOutcomes(out, pending)
}

func (p *Presenter) bisectPresentations(ctx context.Context, items []communityclient.AnchorPresentationItem, pending []preparedPresentation, cause error) ([]presentationClaim, error) {
	if len(items) == 1 {
		slog.Error("anchor presentations: item rejected by community",
			"anchor_kind", items[0].AnchorKind, "anchor_id", items[0].AnchorID, "error", cause)
		return []presentationClaim{pending[0].claim}, nil
	}
	mid := len(items) / 2
	left, err := p.sendPresentations(ctx, items[:mid], pending[:mid])
	if err != nil {
		return nil, err
	}
	right, err := p.sendPresentations(ctx, items[mid:], pending[mid:])
	if err != nil {
		return nil, err
	}
	return append(left, right...), nil
}

func (p *Presenter) applyPresentationOutcomes(resp *communityclient.AnchorPresentationWriteResponse, pending []preparedPresentation) ([]presentationClaim, error) {
	if resp == nil {
		return nil, errors.New("anchor presentations: empty write response")
	}
	if len(resp.Results) != len(pending) {
		return nil, fmt.Errorf("anchor presentations: %d results for %d items", len(resp.Results), len(pending))
	}
	handled := make([]presentationClaim, 0, len(pending))
	for i, prep := range pending {
		handled = append(handled, prep.claim)
		r := resp.Results[i]
		switch r.Outcome {
		case "created", "updated", "removed", "restored":
			if err := p.upsertPresentationSent(prep.claim.key(), prep.item.Revision, prep.item.Removed); err != nil {
				return nil, err
			}
		case "stale":
		case "invalid":
			slog.Warn("anchor presentations: community marked item invalid",
				"anchor_kind", r.AnchorKind, "anchor_id", r.AnchorID, "reason", r.Reason)
		}
	}
	return handled, nil
}
