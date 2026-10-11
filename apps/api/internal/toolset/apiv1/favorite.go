package apiv1

import (
	"context"
	"strconv"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/problem"
)

func (s *Service) putToolsetFavorite(ctx context.Context, in *toolsetIDInput) (*engagementOutput, error) {
	return s.setFavorite(ctx, in.ToolsetID, true)
}

func (s *Service) deleteToolsetFavorite(ctx context.Context, in *toolsetIDInput) (*engagementOutput, error) {
	return s.setFavorite(ctx, in.ToolsetID, false)
}

func (s *Service) setFavorite(ctx context.Context, rawID string, on bool) (*engagementOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	row, _, p := s.visibleToolset(ctx, rawID)
	if p != nil {
		return nil, p
	}
	favID, changed, err := s.store.SetFavorite(row.ID, user.ID, on)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if changed && user.ID != row.UserID {
		delta, event := 1, "favorited"
		if !on {
			delta, event = -1, "unfavorited"
		}
		s.award(row.UserID, delta, moemoepoint.ReasonLiked, moemoepoint.Ref("toolset", row.ID),
			moemoepoint.Key(event, "toolset_favorite_"+strconv.FormatInt(favID, 10)))
	}
	count, favorited, err := s.store.Engagement(row.ID, user.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &engagementOutput{Body: ToolsetEngagement{
		Object: "toolset_engagement", ToolsetID: repr.ID(row.ID), FavoriteCount: count,
		Viewer: &ToolsetEngagementViewer{HasFavorited: favorited},
	}}, nil
}
