package push

import (
	"context"
	"time"

	"kun-galgame-api/pkg/communityclient"
)

type preparedPresentation struct {
	claim presentationClaim
	item  *communityclient.AnchorPresentationItem
}

func (p *Presenter) RunOnce(ctx context.Context) (bool, error) {
	claims, err := p.claimPresentations(claimLimit)
	if err != nil {
		return false, err
	}
	if len(claims) == 0 {
		return false, nil
	}
	// Revision is the read time: stamp before the first state read, or a later tombstone outranks a live write made after the read.
	rev := time.Now().UnixMicro()
	handled, err := p.pushClaimedPresentations(ctx, claims, rev)
	if err != nil {
		return true, err
	}
	return true, p.ackPresentations(handled)
}

func (p *Presenter) pushClaimedPresentations(ctx context.Context, claims []presentationClaim, rev int64) ([]presentationClaim, error) {
	keys := make([]presentationKey, len(claims))
	for i, c := range claims {
		keys[i] = c.key()
	}
	st, err := p.loadPresentationState(ctx, keys)
	if err != nil {
		return nil, err
	}
	sent, err := p.presentationSentByKeys(keys)
	if err != nil {
		return nil, err
	}
	var (
		items   []communityclient.AnchorPresentationItem
		handled []presentationClaim
		pending []preparedPresentation
	)
	for _, c := range claims {
		k := c.key()
		if item, live := p.livePresentation(ctx, k, st, rev); live {
			pending = append(pending, preparedPresentation{claim: c, item: &item})
			items = append(items, item)
			continue
		}
		prev, had := sent[k]
		if had && !prev.Removed {
			item := presentationTombstone(k, rev)
			pending = append(pending, preparedPresentation{claim: c, item: &item})
			items = append(items, item)
			continue
		}
		handled = append(handled, c)
	}
	if len(items) == 0 {
		return handled, nil
	}
	done, err := p.sendPresentations(ctx, items, pending)
	if err != nil {
		return nil, err
	}
	return append(handled, done...), nil
}
