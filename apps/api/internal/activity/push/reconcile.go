package push

import (
	"context"
	"log/slog"
	"time"

	"kun-galgame-api/internal/activity/repository"
	"kun-galgame-api/pkg/communityclient"
)

const reconcilePage = 1000

func (p *Pusher) Reconcile(ctx context.Context) {
	start := time.Now()
	stored, err := p.loadStored(ctx)
	if err != nil {
		slog.Warn("activity push: reconcile list failed", "error", err)
		return
	}
	local := map[string]struct{}{}
	var tombs []communityclient.ActivityWriteItem
	enqueued := 0
	var after *Key
	for {
		rev := time.Now().UnixMicro()
		rows, err := p.pageLocal(after, claimLimit)
		if err != nil {
			slog.Warn("activity push: reconcile local page failed", "error", err)
			return
		}
		if len(rows) == 0 {
			break
		}
		n, err := p.reconcileLocal(ctx, rows, stored, local, &tombs, rev)
		if err != nil {
			slog.Warn("activity push: reconcile page failed", "error", err)
			return
		}
		enqueued += n
		last := rows[len(rows)-1]
		after = &Key{Type: last.TypeStr, SourceID: last.SourceID}
		if len(rows) < claimLimit {
			break
		}
	}
	for key, view := range stored {
		if _, ok := local[key]; !ok && !view.Removed {
			tombs = append(tombs, Tombstone(key, view.ActorID, 0))
		}
	}
	tombstoned, err := p.sendTombstones(ctx, tombs)
	if err != nil {
		slog.Warn("activity push: reconcile tombstones failed", "error", err)
		return
	}
	slog.Info("activity push: reconcile finished",
		"stored", len(stored), "local", len(local), "enqueued", enqueued, "tombstoned", tombstoned,
		"duration", time.Since(start))
}

func (p *Pusher) loadStored(ctx context.Context) (map[string]communityclient.SiteActivityView, error) {
	out := map[string]communityclient.SiteActivityView{}
	cursor := ""
	for {
		page, err := p.community.ListSiteActivities(ctx, cursor, reconcilePage)
		if err != nil {
			return nil, err
		}
		if page == nil || len(page.Activities) == 0 {
			return out, nil
		}
		for _, v := range page.Activities {
			out[v.Key] = v
		}
		if page.NextCursor == "" || page.NextCursor == cursor {
			return out, nil
		}
		cursor = page.NextCursor
	}
}

func (p *Pusher) reconcileLocal(ctx context.Context, recs []feedRecord, stored map[string]communityclient.SiteActivityView, local map[string]struct{}, tombs *[]communityclient.ActivityWriteItem, rev int64) (int, error) {
	rows := make([]repository.FeedRow, len(recs))
	for i := range recs {
		rows[i] = recs[i].FeedRow
	}
	assembled, failed, err := p.assembleRows(ctx, rows)
	if err != nil {
		return 0, err
	}
	names, err := p.pageNames(assembled)
	if err != nil {
		return 0, err
	}
	n := 0
	for i, rec := range recs {
		if !IsPushed(rec.TypeStr) {
			continue
		}
		key := KeyOf(rec.TypeStr, rec.SourceID)
		local[key] = struct{}{}
		if failed[i] {
			slog.Warn("activity push: reconcile skipped a row that does not assemble", "key", key)
			continue
		}
		view, isStored := stored[key]
		act := assembled[i]
		if act == nil || act.Performer == nil {
			if isStored && !view.Removed {
				*tombs = append(*tombs, Tombstone(key, view.ActorID, 0))
			}
			continue
		}
		desired, err := MapLive(rec.TypeStr, rec.SourceID, act, names[i], rec.IsNSFW, false, p.origin, rev, rec.Created)
		if err != nil {
			slog.Warn("activity push: reconcile live item not sendable", "key", key, "error", err)
			continue
		}
		if !isStored || itemDiffers(desired, view) {
			if err := p.enqueue(Key{Type: rec.TypeStr, SourceID: rec.SourceID}); err != nil {
				return n, err
			}
			n++
		}
	}
	return n, nil
}

func (p *Pusher) sendTombstones(ctx context.Context, tombs []communityclient.ActivityWriteItem) (int, error) {
	n := 0
	for start := 0; start < len(tombs); start += claimLimit {
		batch := tombs[start:min(start+claimLimit, len(tombs))]
		rev := time.Now().UnixMicro()
		for i := range batch {
			batch[i].Revision = rev
		}
		if _, err := p.community.WriteActivities(ctx, batch); err != nil {
			return n, err
		}
		n += len(batch)
	}
	return n, nil
}

func itemDiffers(want communityclient.ActivityWriteItem, have communityclient.SiteActivityView) bool {
	if have.Removed {
		return true
	}
	if want.ActorID != have.ActorID {
		return true
	}
	if want.Verb != have.Verb || want.ObjectKind != have.ObjectKind || want.ObjectLabel != have.ObjectLabel {
		return true
	}
	if want.Title != have.Title || want.Excerpt != have.Excerpt || want.URL != have.URL {
		return true
	}
	if want.ContentLimit != have.ContentLimit {
		return true
	}
	if !sameOptString(want.CoverImageHash, have.CoverImageHash) {
		return true
	}
	if !sameOptInt(want.WorkID, have.WorkID) {
		return true
	}
	return !sameTime(want.OccurredAt, have.OccurredAt)
}

func sameOptString(want string, have *string) bool {
	if want == "" {
		return have == nil || *have == ""
	}
	return have != nil && *have == want
}

func sameOptInt(want int64, have *int64) bool {
	if want == 0 {
		return have == nil || *have == 0
	}
	return have != nil && *have == want
}

func sameTime(a, b string) bool {
	if a == b {
		return true
	}
	ta, okA := parseInstant(a)
	tb, okB := parseInstant(b)
	return okA && okB && ta.Equal(tb)
}

func parseInstant(s string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, occurredFmt, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
