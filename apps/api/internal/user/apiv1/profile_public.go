package apiv1

import (
	"context"
	"errors"
	"slices"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/user/service"
	"kun-galgame-api/pkg/problem"
)

type getUserInput struct {
	UserID string `path:"user_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"User id."`
}

type getUserOutput struct {
	Body UserProfile
}

func (s *Users) getUser(ctx context.Context, in *getUserInput) (*getUserOutput, error) {
	if prob := s.readyUsers(); prob != nil {
		return nil, prob
	}
	id, ok := repr.ParseID(repr.DecimalID(in.UserID))
	if !ok {
		return nil, notFound()
	}
	p, err := s.users.Profile(ctx, id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return nil, notFound()
		}
		if errors.Is(err, service.ErrUpstream) {
			return nil, unavailable(err)
		}
		return nil, problem.Internal(err)
	}
	return &getUserOutput{Body: mapUserProfile(s.cdn, p)}, nil
}

func mapUserProfile(cdn string, p *service.PublicProfile) UserProfile {
	name := p.Account.Name
	bio := p.Account.Bio
	roles := make([]UserRole, 0, len(displayRoles))
	for _, r := range displayRoles {
		if !slices.Contains(p.Account.Roles, r) {
			continue
		}
		roles = append(roles, UserRole(r))
	}
	return UserProfile{
		Object:      "user",
		ID:          repr.ID(p.Account.ID),
		Name:        &name,
		Avatar:      repr.NewImage(cdn, p.Account.AvatarImageHash, nil),
		Bio:         &bio,
		Roles:       roles,
		CreatedAt:   repr.Timestamp(p.CreatedAt),
		Moemoepoint: p.Moemoepoint,
		Counts: UserCounts{
			TopicCount:                 p.Counts.TopicCount,
			PollCount:                  p.Counts.PollCount,
			LotteryCount:               p.Counts.LotteryCount,
			ReplyCount:                 p.Counts.ReplyCount,
			TopicCommentCount:          p.Counts.TopicCommentCount,
			CommunityCommentCount:      p.Counts.CommunityCommentCount,
			PublishedGalgameCount:      p.Counts.PublishedGalgameCount,
			ContributedGalgameCount:    p.Counts.ContributedGalgameCount,
			PublishedGalgameTodayCount: p.Counts.PublishedGalgameTodayCount,
			GalgameRatingCount:         p.Counts.GalgameRatingCount,
			GalgameResourceCount:       p.Counts.GalgameResourceCount,
			ToolsetCount:               p.Counts.ToolsetCount,
			ToolsetResourceCount:       p.Counts.ToolsetResourceCount,
			ReceivedUpvoteCount:        p.Counts.ReceivedUpvoteCount,
			ReceivedLikeCount:          p.Counts.ReceivedLikeCount,
			ReceivedDislikeCount:       p.Counts.ReceivedDislikeCount,
			TopicTodayCount:            p.Counts.TopicTodayCount,
		},
	}
}
