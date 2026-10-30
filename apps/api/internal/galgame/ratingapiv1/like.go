package ratingapiv1

import (
	"context"
	"strconv"
	"unicode/utf8"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

type engagementOutput struct {
	Body RatingEngagement
}

func (s *Service) likeRating(ctx context.Context, in *ratingPathInput) (*engagementOutput, error) {
	return s.setLike(ctx, in.RatingID, true)
}

func (s *Service) unlikeRating(ctx context.Context, in *ratingPathInput) (*engagementOutput, error) {
	return s.setLike(ctx, in.RatingID, false)
}

func (s *Service) setLike(ctx context.Context, rawID string, want bool) (*engagementOutput, error) {
	if !s.ready() {
		return nil, problem.Internal(errUnconfigured)
	}
	viewer := v1.User(ctx)
	if p := s.requireActive(ctx, viewer); p != nil {
		return nil, p
	}
	r, p := s.owned(ctx, rawID)
	if p != nil {
		return nil, p
	}
	if p := s.readable(ctx, r); p != nil {
		return nil, p
	}
	if r.UserID == viewer.ID {
		return nil, problem.New(problem.CodeSelfLikeForbidden, "Users cannot like their own ratings.")
	}
	likeID := 0
	err := s.Store.DB().Transaction(func(tx *gorm.DB) error {
		var err error
		likeID, err = s.Store.SetLike(tx, r.ID, viewer.ID, want)
		if err != nil || likeID == 0 || !want || s.Notify == nil {
			return err
		}
		preview := markdown.MaskPlainSpoilers(r.ShortSummary)
		if utf8.RuneCountInString(preview) > constants.TextPreviewLength {
			preview = string([]rune(preview)[:constants.TextPreviewLength])
		}
		return s.Notify(tx, viewer.ID, r.UserID, preview, r.WorkID, r.ID)
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	if likeID > 0 && s.Award != nil {
		delta, event := 1, "liked"
		if !want {
			delta, event = -1, "unliked"
		}
		s.Award(r.UserID, delta, moemoepoint.ReasonLiked, moemoepoint.Ref("galgame_rating", r.ID),
			moemoepoint.Key(event, "galgame_rating_like_"+strconv.Itoa(likeID)))
	}
	fresh, err := s.Store.Get(r.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &engagementOutput{Body: RatingEngagement{
		Object:    "rating_engagement",
		RatingID:  repr.ID(r.ID),
		LikeCount: max(fresh.LikeCount, 0),
		Viewer:    RatingEngagementViewer{HasLiked: want},
	}}, nil
}
