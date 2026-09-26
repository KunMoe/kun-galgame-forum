package notify

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"kun-galgame-api/internal/community/anchor"
	msgRepo "kun-galgame-api/internal/message/repository"
	"kun-galgame-api/pkg/communityclient"

	"github.com/redis/go-redis/v9"
)

const (
	cursorKey     = "community:notification:feed:after"
	feedInterval  = 15 * time.Second
	feedPageLimit = 500
	feedMaxPages  = 20
	resolveBatch  = 100
	tickWait      = 2 * time.Minute
)

type Poller struct {
	community *communityclient.Client
	messages  *msgRepo.MessageRepository
	anchors   *anchor.Resolver
	rdb       *redis.Client
}

func New(community *communityclient.Client, messages *msgRepo.MessageRepository, anchors *anchor.Resolver, rdb *redis.Client) *Poller {
	return &Poller{community: community, messages: messages, anchors: anchors, rdb: rdb}
}

func (p *Poller) Start() func() {
	if p == nil || p.community == nil || !p.community.Configured() || p.rdb == nil || p.messages == nil {
		return func() {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		p.RunOnce(ctx)
		t := time.NewTicker(feedInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				p.RunOnce(ctx)
			}
		}
	}()
	return func() {
		cancel()
		<-done
	}
}

func (p *Poller) RunOnce(ctx context.Context) {
	if p.community == nil || !p.community.Configured() {
		return
	}
	runCtx, cancel := context.WithTimeout(ctx, tickWait)
	defer cancel()

	after, err := p.readCursor(runCtx)
	if err != nil {
		slog.Warn("community notification feed: cursor read failed", "error", err)
		return
	}

	for page := 0; page < feedMaxPages; page++ {
		feed, err := p.community.NotificationFeed(runCtx, after, feedPageLimit)
		if err != nil {
			slog.Warn("community notification feed: page fetch failed", "after", after, "error", err)
			return
		}
		if len(feed.Notifications) == 0 {
			return
		}
		if err := p.writePage(runCtx, feed.Notifications); err != nil {
			slog.Warn("community notification feed: page write failed", "after", after, "error", err)
			return
		}
		after = feed.NextAfter
		p.writeCursor(runCtx, after)
	}
}

func (p *Poller) writePage(ctx context.Context, notes []communityclient.NotificationView) error {
	postIDs := make([]int64, 0, len(notes))
	seenPost := make(map[int64]bool, len(notes))
	refs := make([]anchor.Ref, 0, len(notes))
	seenRef := make(map[anchor.Ref]bool, len(notes))
	for _, n := range notes {
		if n.Kind == communityclient.InboxFollowed || n.Kind == communityclient.InboxFolloweeActivity {
			continue
		}
		if n.PostID != nil && *n.PostID > 0 && !seenPost[*n.PostID] {
			seenPost[*n.PostID] = true
			postIDs = append(postIDs, *n.PostID)
		}
		ref := anchor.Ref{Kind: n.AnchorKind, ID: n.AnchorID}
		if !seenRef[ref] {
			seenRef[ref] = true
			refs = append(refs, ref)
		}
	}
	posts, err := p.resolvePosts(ctx, postIDs)
	if err != nil {
		return err
	}
	targets := p.anchors.Resolve(refs)

	for i := range notes {
		n := notes[i]
		if n.Kind == communityclient.InboxFollowed || n.Kind == communityclient.InboxFolloweeActivity {
			if n.Kind == communityclient.InboxFolloweeActivity && n.ItemCount == 0 {
				if err := p.messages.DeleteCommunityMirror(n.ID, n.Seq); err != nil {
					return err
				}
				continue
			}
			msg := MapNotification(n, nil, nil)
			if msg == nil {
				continue
			}
			if err := p.messages.UpsertCommunityMirror(msg); err != nil {
				return err
			}
			continue
		}
		target, ok := targets[anchor.Ref{Kind: n.AnchorKind, ID: n.AnchorID}]
		if !ok {
			continue
		}
		var post *communityclient.PostView
		if n.PostID != nil {
			if pv, hit := posts[*n.PostID]; hit {
				post = &pv
			}
		}
		msg := MapNotification(n, post, &target)
		if msg == nil {
			continue
		}
		if err := p.messages.UpsertCommunityMirror(msg); err != nil {
			return err
		}
	}
	return nil
}

// A transient failure holds the page: a row written now would keep an empty
// preview forever, because only a growing fold is ever sent again. A rejection
// is logged and skipped so one bad batch cannot wedge the feed.
func (p *Poller) resolvePosts(ctx context.Context, ids []int64) (map[int64]communityclient.PostView, error) {
	out := make(map[int64]communityclient.PostView, len(ids))
	for i := 0; i < len(ids); i += resolveBatch {
		end := min(i+resolveBatch, len(ids))
		res, err := p.community.ResolvePosts(ctx, ids[i:end])
		if err != nil {
			var apiErr *communityclient.APIError
			if errors.As(err, &apiErr) && apiErr.Status >= 400 && apiErr.Status < 500 {
				slog.Warn("community notification feed: resolve posts rejected", "error", err)
				continue
			}
			return nil, err
		}
		for _, ap := range res.Posts {
			out[ap.Post.ID] = ap.Post
		}
	}
	return out, nil
}

func (p *Poller) readCursor(ctx context.Context) (int64, error) {
	v, err := p.rdb.Get(ctx, cursorKey).Int64()
	if err == redis.Nil {
		max, mErr := p.messages.MaxCommunitySeq()
		if mErr != nil {
			return 0, mErr
		}
		p.writeCursor(ctx, max)
		return max, nil
	}
	return v, err
}

func (p *Poller) writeCursor(ctx context.Context, after int64) {
	if err := p.rdb.Set(ctx, cursorKey, after, 0).Err(); err != nil {
		slog.Warn("community notification feed: cursor write failed", "after", after, "error", err)
	}
}
