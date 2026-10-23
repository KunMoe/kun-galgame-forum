package service

import (
	"context"
	"log/slog"
	"time"

	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/userclient"
)

type heldRatingAuthors interface {
	Users(ctx context.Context, ids []int) (map[int]userclient.User, error)
}

type heldRatingSweepStore interface {
	HeldRatingUsers() ([]int, error)
	DropHeldRatingsOf(userIDs []int) (int64, error)
	DeletedHoldsOnListedWorks() ([]repository.HeldRatingRef, error)
	DropDeletedHold(id int) error
}

// HeldRatingSweep drops the held ratings nothing else will ever drop: those
// of accounts deleted since, and those whose authors deleted them while their
// work was deleted. A held rating's own gone has already arrived, and an
// account deleted on NextMoe never reaches the forum's purge.
//
// Every held author is checked each run rather than following the
// deleted-account stream: a rating can be held after its author's deletion
// has passed by on any stream cursor.
type HeldRatingSweep struct {
	users   heldRatingAuthors
	store   heldRatingSweepStore
	fate    workFate
	current currentRating
}

func NewHeldRatingSweep(users heldRatingAuthors, store heldRatingSweepStore) *HeldRatingSweep {
	return &HeldRatingSweep{users: users, store: store}
}

// WithRestoredWorks lets the sweep look at the ratings held for a deleted work
// once the work is back.
func (s *HeldRatingSweep) WithRestoredWorks(fate workFate, current currentRating) *HeldRatingSweep {
	s.fate, s.current = fate, current
	return s
}

func (s *HeldRatingSweep) Run() {
	if s == nil || s.store == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	s.dropDeletedAccounts(ctx)
	s.dropDeletedOnRestoredWorks(ctx)
}

func (s *HeldRatingSweep) dropDeletedAccounts(ctx context.Context) {
	if s.users == nil {
		return
	}
	ids, err := s.store.HeldRatingUsers()
	if err != nil || len(ids) == 0 {
		if err != nil {
			slog.Warn("暂存评分清扫: 读取作者失败", "error", err)
		}
		return
	}
	accounts, err := s.users.Users(ctx, ids)
	if err != nil {
		slog.Warn("暂存评分清扫: 账号服务不可用, 本轮跳过", "error", err)
		return
	}
	var deleted []int
	for _, id := range ids {
		if u, ok := accounts[id]; !ok || u.AnonymizedAt != nil {
			deleted = append(deleted, id)
		}
	}
	n, err := s.store.DropHeldRatingsOf(deleted)
	if err != nil {
		slog.Warn("暂存评分清扫: 删除失败", "error", err)
		return
	}
	if n > 0 {
		slog.Info("暂存评分清扫: 已删除已注销账号的暂存评分", "accounts", len(deleted), "ratings", n)
	}
}

// dropDeletedOnRestoredWorks keeps a held rating of a restored work only while
// NextMoe holds it as its author's rating of the work; the mirror writes it back
// when NextMoe sends it again. A work catalog cannot answer about is left for
// the next run.
func (s *HeldRatingSweep) dropDeletedOnRestoredWorks(ctx context.Context) {
	if s.fate == nil || s.current == nil {
		return
	}
	holds, err := s.store.DeletedHoldsOnListedWorks()
	if err != nil {
		slog.Warn("暂存评分清扫: 读取已恢复作品的暂存评分失败", "error", err)
		return
	}
	fates := map[int]string{}
	dropped := 0
	for _, h := range holds {
		reason, seen := fates[h.WorkID]
		if !seen {
			if reason, err = s.fate(ctx, int64(h.WorkID)); err != nil {
				slog.Warn("暂存评分清扫: 无法确认作品是否已恢复", "work_id", h.WorkID, "error", err)
				reason = repository.HeldDeleted
			}
			fates[h.WorkID] = reason
		}
		if reason != "" {
			continue
		}
		current, found, err := s.current(ctx, int64(h.UserID), int64(h.WorkID))
		if err != nil {
			slog.Warn("暂存评分清扫: 无法读取作者的当前评分", "rating_id", h.ID, "error", err)
			continue
		}
		if found && current == int64(h.ID) {
			continue
		}
		if err := s.store.DropDeletedHold(h.ID); err != nil {
			slog.Warn("暂存评分清扫: 删除失败", "rating_id", h.ID, "error", err)
			return
		}
		dropped++
	}
	if dropped > 0 {
		slog.Info("暂存评分清扫: 已删除作者在作品删除期间删掉的评分", "ratings", dropped)
	}
}
