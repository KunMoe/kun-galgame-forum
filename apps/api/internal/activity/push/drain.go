package push

import (
	"context"
	"log/slog"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	activityapiv1 "kun-galgame-api/internal/activity/apiv1"
	"kun-galgame-api/internal/activity/repository"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/problem"
)

const claimLimit = 100

type prepared struct {
	claim Claim
	item  *communityclient.ActivityWriteItem
}

func (p *Pusher) RunOnce(ctx context.Context) (bool, error) {
	claims, err := p.Claim(ctx)
	if err != nil {
		return false, err
	}
	if len(claims) == 0 {
		return false, nil
	}
	rev := time.Now().UnixMicro()
	handled, err := p.PushClaimed(ctx, claims, rev)
	if err != nil {
		return true, err
	}
	return true, p.Ack(ctx, handled)
}

func (p *Pusher) Claim(_ context.Context) ([]Claim, error) {
	return p.claim(claimLimit)
}

func (p *Pusher) PushClaimed(ctx context.Context, claims []Claim, rev int64) ([]Claim, error) {
	keys := make([]Key, len(claims))
	for i, c := range claims {
		keys[i] = Key{Type: c.Type, SourceID: c.SourceID}
	}
	feeds, err := p.feedByKeys(keys)
	if err != nil {
		return nil, err
	}
	sent, err := p.sentByKeys(keys)
	if err != nil {
		return nil, err
	}
	var rows []repository.FeedRow
	present := map[Key]int{}
	nsfw := map[Key]bool{}
	for i := range claims {
		k := keys[i]
		rec, ok := feeds[k]
		if !ok {
			continue
		}
		present[k] = len(rows)
		rows = append(rows, rec.FeedRow)
		nsfw[k] = rec.IsNSFW
	}
	assembled, failed, err := p.assembleRows(ctx, rows)
	if err != nil {
		slog.Warn("activity push: assemble failed", "error", err)
		return nil, err
	}
	names, err := p.pageNames(assembled)
	if err != nil {
		return nil, err
	}
	var (
		items   []communityclient.ActivityWriteItem
		handled []Claim
		pending []prepared
	)
	for i, c := range claims {
		k := keys[i]
		if !IsPushed(c.Type) {
			handled = append(handled, c)
			continue
		}
		idx, ok := present[k]
		if ok && failed[idx] {
			slog.Warn("activity push: row does not assemble, moved to the back of the queue", "key", KeyOf(c.Type, c.SourceID))
			if err := p.requeue(k); err != nil {
				return nil, err
			}
			continue
		}
		var act *activityapiv1.Activity
		if ok {
			act = assembled[idx]
		}
		if act != nil && act.Performer != nil {
			item, err := MapLive(c.Type, c.SourceID, act, names[idx], nsfw[k], c.Backfill, p.origin, rev, rows[idx].Created)
			if err != nil {
				slog.Warn("activity push: live item not sendable", "key", KeyOf(c.Type, c.SourceID), "error", err)
				handled = append(handled, c)
				continue
			}
			pending = append(pending, prepared{claim: c, item: &item})
			items = append(items, item)
			continue
		}
		prev, had := sent[k]
		if had && !prev.Removed {
			item := Tombstone(KeyOf(c.Type, c.SourceID), int64(prev.ActorID), rev)
			pending = append(pending, prepared{claim: c, item: &item})
			items = append(items, item)
			continue
		}
		handled = append(handled, c)
	}
	if len(items) == 0 {
		return handled, nil
	}
	done, err := p.sendItems(ctx, items, pending, rev)
	if err != nil {
		return nil, err
	}
	return append(handled, done...), nil
}

// assembleRows retries a failed batch row by row: an upstream outage fails every
// row and holds the batch, while one row that never assembles must not hold the
// queue behind it.
func (p *Pusher) assembleRows(ctx context.Context, rows []repository.FeedRow) ([]*activityapiv1.Activity, map[int]bool, error) {
	if len(rows) == 0 {
		return nil, nil, nil
	}
	acts, prob := p.assemble.AssembleForPush(ctx, rows)
	if prob == nil {
		return acts, nil, nil
	}
	acts = make([]*activityapiv1.Activity, len(rows))
	failed := map[int]bool{}
	for i := range rows {
		one, onePro := p.assemble.AssembleForPush(ctx, rows[i:i+1])
		if onePro != nil {
			failed[i] = true
			continue
		}
		acts[i] = one[0]
	}
	if len(failed) == len(rows) {
		return nil, nil, problemErr(prob)
	}
	return acts, failed, nil
}

func (p *Pusher) Ack(_ context.Context, claims []Claim) error {
	for _, c := range claims {
		if err := p.ack(c); err != nil {
			return err
		}
	}
	return nil
}

func problemErr(prob *problem.Problem) error {
	if prob == nil {
		return nil
	}
	return prob
}

func pathID(path, prefix string) (int, bool) {
	rest, ok := strings.CutPrefix(path, prefix)
	if !ok {
		return 0, false
	}
	rest, _, _ = strings.Cut(rest, "?")
	rest, _, _ = strings.Cut(rest, "/")
	id, err := strconv.Atoi(rest)
	return id, err == nil && id > 0
}

func pathTail(path, prefix string) string {
	rest, ok := strings.CutPrefix(path, prefix)
	if !ok {
		return ""
	}
	rest, _, _ = strings.Cut(rest, "?")
	return rest
}

func (p *Pusher) pageNames(acts []*activityapiv1.Activity) (map[int]string, error) {
	toolIDs := map[int]int{}
	siteURLs := map[int]string{}
	for i, a := range acts {
		if a == nil {
			continue
		}
		switch string(a.ActivityType) {
		case "toolset_comment_creation":
			if id, ok := pathID(a.Path, "/toolset/"); ok {
				toolIDs[i] = id
			}
		case "galgame_website_comment_creation":
			if u := pathTail(a.Path, "/website/"); u != "" {
				siteURLs[i] = u
			}
		}
	}
	tools, err := p.namesByToolset(slices.Collect(maps.Values(toolIDs)))
	if err != nil {
		return nil, err
	}
	sites, err := p.namesByWebsiteURL(slices.Collect(maps.Values(siteURLs)))
	if err != nil {
		return nil, err
	}
	out := make(map[int]string, len(toolIDs)+len(siteURLs))
	for i, id := range toolIDs {
		out[i] = tools[id]
	}
	for i, u := range siteURLs {
		out[i] = sites[u]
	}
	return out, nil
}
