package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	galgameService "kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/infrastructure/cron"
	"kun-galgame-api/internal/user/model"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

type PublicProfile struct {
	Account     userclient.User
	CreatedAt   time.Time
	Moemoepoint int
	Counts      ProfileCounts
}

type ProfileCounts struct {
	TopicCount                 int
	PollCount                  int
	LotteryCount               int
	ReplyCount                 int
	TopicCommentCount          int
	CommunityCommentCount      *int
	PublishedGalgameCount      int
	ContributedGalgameCount    int
	PublishedGalgameTodayCount int
	GalgameRatingCount         int
	GalgameResourceCount       int
	ToolsetCount               int
	ToolsetResourceCount       int
	ReceivedUpvoteCount        int
	ReceivedLikeCount          int
	ReceivedDislikeCount       int
	TopicTodayCount            int
}

func (s *UserService) Profile(ctx context.Context, userID int) (*PublicProfile, error) {
	u, ok, err := s.userClient.User(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUpstream, err)
	}
	if !ok || !userclient.IsRenderable(u) {
		return nil, ErrNotFound
	}

	loc := cron.ScheduleLocation()
	now := s.now().In(loc)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).UTC()
	local, err := s.userStatsRepo.ProfileCounts(userID, todayStart)
	if err != nil {
		return nil, err
	}

	state, err := s.loadState(userID)
	if err != nil {
		return nil, err
	}

	created := parseAccountCreated(u.CreatedAt)
	if created.IsZero() && state != nil {
		created = state.CreatedAt
	}

	moe := 0
	if state != nil {
		moe = state.Moemoepoint
	}

	var gs galgameService.GalgameUserStats
	if s.galgameStats != nil {
		gs = s.galgameStats.Stats(ctx, int64(userID))
	}

	counts := ProfileCounts{
		TopicCount:                 int(local.TopicCount),
		PollCount:                  int(local.PollCount),
		LotteryCount:               int(local.LotteryCount),
		ReplyCount:                 int(local.ReplyCount),
		TopicCommentCount:          int(local.TopicCommentCount),
		PublishedGalgameCount:      int(gs.Published),
		ContributedGalgameCount:    gs.Contributed,
		PublishedGalgameTodayCount: gs.PublishedToday,
		GalgameRatingCount:         int(local.GalgameRatingCount),
		GalgameResourceCount:       int(local.GalgameResourceCount),
		ToolsetCount:               int(local.ToolsetCount),
		ToolsetResourceCount:       int(local.ToolsetResourceCount),
		ReceivedUpvoteCount:        int(local.ReceivedUpvoteCount),
		ReceivedLikeCount:          int(local.ReceivedLikeCount),
		ReceivedDislikeCount:       int(local.ReceivedDislikeCount),
		TopicTodayCount:            int(local.TopicTodayCount),
	}
	if n, known := s.communityVisiblePosts(ctx, userID); known {
		v := int(n)
		counts.CommunityCommentCount = &v
	}

	return &PublicProfile{
		Account:     u,
		CreatedAt:   created,
		Moemoepoint: moe,
		Counts:      counts,
	}, nil
}

func (s *UserService) loadState(userID int) (*model.KungalUserState, error) {
	state, err := s.stateRepo.FindByID(userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return state, nil
}

func parseAccountCreated(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t
	}
	return time.Time{}
}
