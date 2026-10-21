package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/pkg/catalogclient"

	"github.com/redis/go-redis/v9"
)

const (
	// Losing this key replays every rating NextMoe holds, which is not a fault:
	// the mirror upserts by id. Deleting it is how the mirror is rebuilt.
	RatingMirrorCursorKey = "catalog:ratings:changes:cursor"

	ratingMirrorTimeout = 5 * time.Minute
	ratingMirrorPages   = 100
)

type ratingFeed interface {
	RatingChanges(ctx context.Context, cursor string) (*catalogclient.RatingChangePage, error)
}

type ratingMirrorStore interface {
	MirrorRating(r model.GalgameRating) error
	DropRating(id int) error
	HideWork(workID int) error
}

type workDeletion func(ctx context.Context, workID int64) (bool, error)

// RatingMirror keeps galgame_rating equal to NextMoe's ratings off
// /v2/ratings/changes, on every work whether or not the forum has a local row
// for it. NextMoe owns the ratings since 2026-10-11.
type RatingMirror struct {
	feed    ratingFeed
	store   ratingMirrorStore
	rdb     *redis.Client
	clients map[string]bool
	deleted workDeletion

	running sync.Mutex
}

// WithWorkDeletion lets the mirror tell a rating NextMoe deleted from one it
// hid with its deleted work. Since infra #454 both arrive as gone: true, and
// dropping the second takes its likes and the forum-only game types with it,
// which the restore that sends the rating back cannot return.
func (m *RatingMirror) WithWorkDeletion(deleted workDeletion) *RatingMirror {
	m.deleted = deleted
	return m
}

// NewRatingMirror takes the OAuth clients whose ratings were written through
// this forum: the web's own and the forum's apps.
func NewRatingMirror(feed ratingFeed, store ratingMirrorStore, rdb *redis.Client, localClients []string) *RatingMirror {
	clients := make(map[string]bool, len(localClients))
	for _, c := range localClients {
		if c != "" {
			clients[c] = true
		}
	}
	return &RatingMirror{feed: feed, store: store, rdb: rdb, clients: clients}
}

// WrittenHere is the forum's test for "this site wrote it". NextMoe binds
// moyu's client to the kungal site as well, so the site name cannot answer it.
func (m *RatingMirror) WrittenHere(clientID string) bool {
	return m.clients[clientID]
}

func (m *RatingMirror) Row(r *catalogclient.Rating) model.GalgameRating {
	a := r.Aspects
	row := model.GalgameRating{
		ID: int(r.ID), WorkID: int(r.WorkID), UserID: int(r.UserID), Overall: r.Overall,
		Art: aspect(a.Art), Story: aspect(a.Story), Music: aspect(a.Music), Character: aspect(a.Character),
		Route: aspect(a.Route), System: aspect(a.System), Voice: aspect(a.Voice), ReplayValue: aspect(a.ReplayValue),
		Recommend: r.Recommend, SpoilerLevel: r.SpoilerLevel, ShortSummary: r.ShortSummary,
		ClientID: r.ClientID, Site: r.Site, WrittenHere: m.WrittenHere(r.ClientID),
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if s := r.WorkState; s != nil {
		state := s.State
		row.AuthorState, row.AuthorCompletion = &state, s.Completion
	}
	return row
}

func aspect(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func (m *RatingMirror) Run() {
	if m == nil || m.feed == nil || m.store == nil || m.rdb == nil {
		return
	}
	if !m.running.TryLock() {
		slog.Info("评分镜像仍在进行, 跳过本轮")
		return
	}
	defer m.running.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), ratingMirrorTimeout)
	defer cancel()

	cursor, err := m.rdb.Get(ctx, RatingMirrorCursorKey).Result()
	if err != nil && err != redis.Nil {
		slog.Warn("读取评分镜像游标失败 (本轮跳过)", "error", err)
		return
	}
	bootstrap := cursor == ""

	var seen, written, dropped, pages int
	for ; pages < ratingMirrorPages; pages++ {
		page, err := m.feed.RatingChanges(ctx, cursor)
		if err != nil {
			slog.Warn("评分镜像信道拉取失败, 游标保持不动", "pages", pages, "error", err)
			break
		}
		seen += len(page.Items)
		w, d, err := m.apply(ctx, page.Items)
		written += w
		dropped += d
		if err != nil {
			slog.Error("评分镜像写入失败, 游标保持不动", "pages", pages, "error", err)
			break
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
		if err := m.rdb.Set(ctx, RatingMirrorCursorKey, cursor, 0).Err(); err != nil {
			slog.Warn("写入评分镜像游标失败, 下轮将重放本页", "error", err)
			break
		}
	}
	if seen == 0 {
		return
	}
	slog.Info("评分镜像同步完成", "bootstrap", bootstrap, "pages", pages, "changes", seen,
		"written", written, "dropped", dropped)
}

func (m *RatingMirror) apply(ctx context.Context, items []catalogclient.RatingChange) (written, dropped int, err error) {
	deletedWork := map[int64]bool{}
	for _, it := range items {
		if it.ID <= 0 {
			continue
		}
		if it.Gone || it.Rating == nil {
			hide, err := m.workGone(ctx, it.WorkID, deletedWork)
			if err != nil {
				return written, dropped, err
			}
			if hide {
				if err := m.store.HideWork(int(it.WorkID)); err != nil {
					return written, dropped, err
				}
				continue
			}
			if err := m.store.DropRating(int(it.ID)); err != nil {
				return written, dropped, err
			}
			dropped++
			continue
		}
		if it.Rating.WorkID <= 0 || it.Rating.UserID <= 0 {
			slog.Warn("评分镜像: 跳过缺少作品或作者的评分", "rating_id", it.ID)
			continue
		}
		if err := m.store.MirrorRating(m.Row(it.Rating)); err != nil {
			return written, dropped, err
		}
		written++
	}
	return written, dropped, nil
}

func (m *RatingMirror) workGone(ctx context.Context, workID int64, seen map[int64]bool) (bool, error) {
	if m.deleted == nil || workID <= 0 {
		return false, nil
	}
	if gone, ok := seen[workID]; ok {
		return gone, nil
	}
	gone, err := m.deleted(ctx, workID)
	if err != nil {
		return false, err
	}
	seen[workID] = gone
	return gone, nil
}
