package service

import (
	"context"
	"log/slog"
	"time"

	"kun-galgame-api/pkg/communityclient"
)

type MessageService struct {
	community *communityclient.Client
}

func NewMessageService(community *communityclient.Client) *MessageService {
	return &MessageService{community: community}
}

func (s *MessageService) ForwardRead(userID int, ids []int64) {
	s.forwardRead(userID, ids)
}

func (s *MessageService) forwardRead(userID int, ids []int64) {
	if s.community == nil || !s.community.Configured() || len(ids) == 0 {
		return
	}
	copied := append([]int64(nil), ids...)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		// Forward by id, never {all:true}: all would also mark rows dispatched
		// upstream but not mirrored yet, so the user would never see them unread.
		for i := 0; i < len(copied); i += 100 {
			end := i + 100
			if end > len(copied) {
				end = len(copied)
			}
			if _, err := s.community.MarkNotificationsRead(ctx, int64(userID), copied[i:end]); err != nil {
				slog.Warn("community notification read forward failed", "user_id", userID, "error", err)
				continue
			}
		}
	}()
}
