package service

import (
	"context"
	"log/slog"
	"time"

	"kun-galgame-api/pkg/userclient"
)

type heldRatingAuthors interface {
	Users(ctx context.Context, ids []int) (map[int]userclient.User, error)
}

type heldRatingSweepStore interface {
	HeldRatingUsers() ([]int, error)
	DropHeldRatingsOf(userIDs []int) (int64, error)
}

// HeldRatingSweep drops the held ratings of accounts deleted since. A held
// rating's own gone already arrived, so nothing on the ratings feed will drop
// it, and an account deleted on NextMoe never reaches the forum's purge. Every
// held author is checked each run rather than following the deleted-account
// stream: a rating can be held after its author's deletion has passed by on
// any stream cursor.
type HeldRatingSweep struct {
	users heldRatingAuthors
	store heldRatingSweepStore
}

func NewHeldRatingSweep(users heldRatingAuthors, store heldRatingSweepStore) *HeldRatingSweep {
	return &HeldRatingSweep{users: users, store: store}
}

func (s *HeldRatingSweep) Run() {
	if s == nil || s.users == nil || s.store == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
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
