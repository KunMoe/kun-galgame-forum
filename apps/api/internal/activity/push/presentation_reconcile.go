package push

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"kun-galgame-api/pkg/communityclient"
)

func (p *Presenter) Reconcile(ctx context.Context) {
	start := time.Now()
	// Revision is the read time: stamp orphanRev before listing stored, and each page's rev before that page's state read.
	orphanRev := start.UnixMicro()
	stored, err := p.loadStoredPresentations(ctx)
	if err != nil {
		slog.Warn("anchor presentations: reconcile list failed", "error", err)
		return
	}
	local := map[presentationKey]struct{}{}
	var tombs []communityclient.AnchorPresentationItem
	enqueued := 0
	var after *presentationKey
	for {
		rev := time.Now().UnixMicro()
		keys, err := p.pageLocalPresentations(after, presentationLocalPage)
		if err != nil {
			slog.Warn("anchor presentations: reconcile local page failed", "error", err)
			return
		}
		if len(keys) == 0 {
			break
		}
		n, err := p.reconcilePresentationPage(ctx, keys, stored, local, &tombs, rev)
		if err != nil {
			slog.Warn("anchor presentations: reconcile page failed", "error", err)
			return
		}
		enqueued += n
		last := keys[len(keys)-1]
		after = &last
		if len(keys) < presentationLocalPage {
			break
		}
	}
	for key, view := range stored {
		if _, ok := local[key]; !ok && !view.Removed {
			tombs = append(tombs, presentationTombstone(key, orphanRev))
		}
	}
	tombstoned, stale, err := p.sendPresentationTombstones(ctx, tombs)
	if err != nil {
		slog.Warn("anchor presentations: reconcile tombstones failed", "error", err)
		return
	}
	slog.Info("anchor presentations: reconcile finished",
		"stored", len(stored), "local", len(local), "enqueued", enqueued, "tombstoned", tombstoned, "stale", stale,
		"duration", time.Since(start))
}

func (p *Presenter) loadStoredPresentations(ctx context.Context) (map[presentationKey]communityclient.AnchorPresentationView, error) {
	out := map[presentationKey]communityclient.AnchorPresentationView{}
	cursor := ""
	for {
		page, err := p.community.ListAnchorPresentations(ctx, cursor, reconcilePage)
		if err != nil {
			return nil, err
		}
		if page == nil || len(page.Presentations) == 0 {
			return out, nil
		}
		for _, v := range page.Presentations {
			out[presentationKey{Kind: int16(v.AnchorKind), ID: v.AnchorID}] = v
		}
		if page.NextCursor == "" || page.NextCursor == cursor {
			return out, nil
		}
		cursor = page.NextCursor
	}
}

func (p *Presenter) reconcilePresentationPage(ctx context.Context, keys []presentationKey, stored map[presentationKey]communityclient.AnchorPresentationView, local map[presentationKey]struct{}, tombs *[]communityclient.AnchorPresentationItem, rev int64) (int, error) {
	st, err := p.loadPresentationState(ctx, keys)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, k := range keys {
		local[k] = struct{}{}
		view, isStored := stored[k]
		item, live := p.livePresentation(ctx, k, st, rev)
		if !live {
			if isStored && !view.Removed {
				*tombs = append(*tombs, presentationTombstone(k, rev))
			}
			continue
		}
		if !isStored || presentationDiffers(item, view) {
			if err := p.enqueuePresentation(k); err != nil {
				return n, err
			}
			n++
		}
	}
	return n, nil
}

func (p *Presenter) sendPresentationTombstones(ctx context.Context, tombs []communityclient.AnchorPresentationItem) (int, int, error) {
	removed, stale := 0, 0
	for start := 0; start < len(tombs); start += claimLimit {
		batch := tombs[start:min(start+claimLimit, len(tombs))]
		resp, err := p.community.WriteAnchorPresentations(ctx, batch)
		if err != nil {
			return removed, stale, err
		}
		if resp == nil || len(resp.Results) != len(batch) {
			n := 0
			if resp != nil {
				n = len(resp.Results)
			}
			return removed, stale, fmt.Errorf("anchor presentations: %d results for %d items", n, len(batch))
		}
		for i, r := range resp.Results {
			switch r.Outcome {
			case "created", "updated", "removed", "restored":
				it := batch[i]
				if err := p.upsertPresentationSent(presentationKey{Kind: int16(it.AnchorKind), ID: it.AnchorID}, it.Revision, true); err != nil {
					return removed, stale, err
				}
				if r.Outcome == "removed" {
					removed++
				}
			case "stale":
				stale++
			case "invalid":
				slog.Warn("anchor presentations: community marked tombstone invalid",
					"anchor_kind", r.AnchorKind, "anchor_id", r.AnchorID, "reason", r.Reason)
			}
		}
	}
	return removed, stale, nil
}

func presentationDiffers(want communityclient.AnchorPresentationItem, have communityclient.AnchorPresentationView) bool {
	if have.Removed {
		return true
	}
	if want.Title != have.Title || want.URL != have.URL || want.ContentLimit != have.ContentLimit {
		return true
	}
	if !sameOptString(want.CoverImageHash, have.CoverImageHash) {
		return true
	}
	return !sameOptInt(want.WorkID, have.WorkID)
}
