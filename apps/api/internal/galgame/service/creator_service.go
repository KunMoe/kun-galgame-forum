package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/role"
	"kun-galgame-api/pkg/userclient"
)

var (
	ErrCreatorIneligible  = errors.New("creator ineligible")
	ErrAccountUnavailable = errors.New("account service unavailable")
)

func accountErr(err error) error { return fmt.Errorf("%w: %w", ErrAccountUnavailable, err) }

const (
	creatorMinMergedPRs   = 5
	creatorMinGalgames    = 10
	creatorMinReviews     = 5
	creatorReviewMinLen   = 100
	creatorMinMoemoepoint = 2000
	creatorSource         = "forum"
)

type CreatorEligibility struct {
	Eligible          bool  `json:"eligible"`
	MergedPRs         int64 `json:"merged_prs"`
	GalgamesPublished int64 `json:"galgames_published"`
	Reviews100        int64 `json:"reviews_100"`
	Moemoepoint       int64 `json:"moemoepoint"`
	NeedMergedPRs     int   `json:"need_merged_prs"`
	NeedGalgames      int   `json:"need_galgames"`
	NeedReviews       int   `json:"need_reviews"`
	NeedMoemoepoint   int   `json:"need_moemoepoint"`
}

type CreatorService struct {
	ratingRepo *repository.RatingRepository
	stats      *GalgameUserStatsService
	userClient *userclient.Client
}

func NewCreatorService(ratingRepo *repository.RatingRepository, stats *GalgameUserStatsService, userClient *userclient.Client) *CreatorService {
	return &CreatorService{ratingRepo: ratingRepo, stats: stats, userClient: userClient}
}

func (s *CreatorService) eligibility(ctx context.Context, userID int) (*CreatorEligibility, error) {
	stats := s.stats.Stats(ctx, int64(userID))
	reviews, rErr := s.ratingRepo.CountReviewsWithMinLength(userID, creatorReviewMinLen)
	if rErr != nil {
		return nil, rErr
	}
	moe, err := s.userClient.GetMoemoepoint(ctx, userID)
	if err != nil {
		return nil, accountErr(err)
	}
	e := &CreatorEligibility{
		MergedPRs:         stats.MergedEdits,
		GalgamesPublished: stats.Published,
		Reviews100:        reviews,
		Moemoepoint:       int64(moe),
		NeedMergedPRs:     creatorMinMergedPRs,
		NeedGalgames:      creatorMinGalgames,
		NeedReviews:       creatorMinReviews,
		NeedMoemoepoint:   creatorMinMoemoepoint,
	}
	e.Eligible = e.MergedPRs >= creatorMinMergedPRs ||
		e.GalgamesPublished >= creatorMinGalgames ||
		e.Reviews100 >= creatorMinReviews ||
		e.Moemoepoint >= creatorMinMoemoepoint
	return e, nil
}

func (s *CreatorService) Status(ctx context.Context, userID int, token string) (*CreatorEligibility, *userclient.CreatorApplication, bool, error) {
	e, err := s.eligibility(ctx, userID)
	if err != nil {
		return nil, nil, false, err
	}
	app, err := s.userClient.GetMyCreatorApplication(ctx, token)
	if err != nil {
		return nil, nil, false, accountErr(err)
	}
	u, ok, uErr := s.userClient.User(ctx, userID)
	if uErr != nil {
		return nil, nil, false, accountErr(uErr)
	}
	isCreator := ok && role.IsCreator(u.Roles)
	return e, app, isCreator, nil
}

func (s *CreatorService) Apply(ctx context.Context, userID int, token, message string) (*userclient.CreatorApplication, error) {
	e, err := s.eligibility(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !e.Eligible {
		return nil, ErrCreatorIneligible
	}
	evidence, _ := json.Marshal(map[string]any{
		"merged_prs":         e.MergedPRs,
		"galgames_published": e.GalgamesPublished,
		"reviews_100":        e.Reviews100,
		"moemoepoint":        e.Moemoepoint,
	})
	app, err := s.userClient.CreateCreatorApplication(ctx, token, creatorSource, evidence, message)
	if err != nil {
		return nil, accountErr(err)
	}
	return app, nil
}
