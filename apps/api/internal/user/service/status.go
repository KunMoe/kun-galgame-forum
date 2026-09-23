package service

import (
	"context"
	"fmt"

	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/pkg/role"
)

type MeStatus struct {
	Moemoepoint             int
	HasCheckedInToday       bool
	HasUnreadMessages       bool
	IsCreator               bool
	ToolsetUploadTodayBytes int64
}

func (s *UserService) Me(ctx context.Context, userID int) (*MeStatus, error) {
	if err := s.stateRepo.Ensure(userID); err != nil {
		return nil, err
	}
	state, err := s.stateRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}

	localMuted, chatMuted := msgService.SplitMuted(state.MutedNotificationTypes)
	unreadMessage, err := s.userStatsRepo.CountUnreadMessages(userID, localMuted)
	if err != nil {
		return nil, err
	}
	var unreadChat int64
	if !chatMuted {
		unreadChat, err = s.userStatsRepo.CountUnreadChatMessages(userID)
		if err != nil {
			return nil, err
		}
	}

	u, ok, err := s.userClient.User(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUpstream, err)
	}

	return &MeStatus{
		Moemoepoint:             state.Moemoepoint,
		HasCheckedInToday:       state.DailyCheckIn == 1,
		HasUnreadMessages:       unreadMessage+unreadChat > 0,
		IsCreator:               ok && role.IsCreator(u.Roles),
		ToolsetUploadTodayBytes: state.DailyToolsetUploadBytes,
	}, nil
}
