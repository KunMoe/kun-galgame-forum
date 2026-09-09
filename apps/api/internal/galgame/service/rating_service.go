package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RatingService struct {
	ratingRepo    *repository.RatingRepository
	galgameClient *client.GalgameClient
	userClient    *userclient.Client
	check         *gate.CheckService
	scan          *gate.ScanService
	helpers       InteractionHelpers
}

func NewRatingService(
	ratingRepo *repository.RatingRepository,
	galgameClient *client.GalgameClient,
	userClient *userclient.Client,
	check *gate.CheckService,
	scan *gate.ScanService,
) *RatingService {
	return &RatingService{
		ratingRepo:    ratingRepo,
		galgameClient: galgameClient,
		userClient:    userClient,
		check:         check,
		scan:          scan,
	}
}

func ratingReward(summaryLen int) int {
	switch {
	case summaryLen >= constants.RatingLenThresholdHigh:
		return constants.RatingRewardHigh
	case summaryLen >= constants.RatingLenThresholdMedium:
		return constants.RatingRewardMedium
	default:
		return constants.RatingRewardLow
	}
}

func (s *RatingService) GetAllRatings(
	ctx context.Context,
	req *dto.RatingListRequest,
	isSFW bool,
) (*dto.RatingListPage, *errors.AppError) {
	sortOrder := req.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	rows, total := s.ratingRepo.ListPaginated(model.RatingFilter{
		SpoilerLevel: req.SpoilerLevel,
		PlayStatus:   req.PlayStatus,
		GalgameType:  req.GalgameType,
		SortField:    req.SortField,
		SortOrder:    sortOrder,
		Page:         req.Page,
		Limit:        req.Limit,
	})

	userIDs := make([]int, len(rows))
	galgameIDs := make([]int, len(rows))
	for i, r := range rows {
		userIDs[i] = r.UserID
		galgameIDs[i] = r.GalgameID
	}
	userMap := s.userClient.Hydrate(ctx, userIDs)
	briefMap := s.fetchGalgameBriefsPublic(ctx, galgameIDs, isSFW)

	cards := make([]dto.RatingCard, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		b, hasBrief := briefMap[r.GalgameID]
		if !hasBrief {
			continue
		}
		cards = append(cards, ratingRowToCard(r, u, b))
	}

	return &dto.RatingListPage{RatingData: cards, Total: total}, nil
}

func (s *RatingService) GetRatingDetail(
	ctx context.Context,
	ratingID, currentUserID int,
) (*dto.RatingDetail, *errors.AppError) {
	row, ok := s.ratingRepo.FindByID(ratingID)
	if !ok {
		return nil, errors.ErrNotFound("评分不存在")
	}

	if author, _, _ := s.userClient.User(ctx, row.UserID); !userclient.IsRenderable(author) {
		return nil, errors.ErrNotFound("评分不存在")
	}

	s.ratingRepo.IncrementView(ratingID)
	row.View++

	likerIDs := s.ratingRepo.FindLikerIDs(ratingID)
	isLiked := containsInt(likerIDs, currentUserID)

	uidSet := map[int]struct{}{row.UserID: {}}
	for _, id := range likerIDs {
		uidSet[id] = struct{}{}
	}
	uids := make([]int, 0, len(uidSet))
	for id := range uidSet {
		uids = append(uids, id)
	}
	userMap := s.userClient.Hydrate(ctx, uids)

	galgame := s.buildRatingGalgame(ctx, row.GalgameID)

	authorBriefs := make([]dto.UserBrief, 0, len(likerIDs))
	for _, id := range likerIDs {
		u := userMap[id]
		if !userclient.IsRenderable(u) {
			continue
		}
		authorBriefs = append(authorBriefs, userBriefToDTO(u))
	}

	detail := &dto.RatingDetail{
		ID:           row.ID,
		User:         userBriefToDTO(userMap[row.UserID]),
		Recommend:    row.Recommend,
		Overall:      row.Overall,
		View:         row.View,
		GalgameType:  rawJSON(row.GalgameType),
		PlayStatus:   row.PlayStatus,
		ShortSummary: row.ShortSummary,
		SpoilerLevel: row.SpoilerLevel,
		RatingScores: rowToScores(row),
		LikeCount:    len(likerIDs),
		IsLiked:      isLiked,
		LikedUsers:   authorBriefs,
		Created:      row.Created,
		Updated:      row.Updated,
		Galgame:      galgame,
	}
	return detail, nil
}

func (s *RatingService) CreateRating(
	ctx context.Context,
	userID int,
	req *dto.CreateRatingRequest,
) (*dto.CreatedRating, *errors.AppError) {
	if s.ratingRepo.ExistsByUserGalgame(req.GalgameID, userID) {
		return nil, errors.ErrBadRequest("您已经发布过该 Galgame 的评分了")
	}

	authorID := int64(userID)
	decision, matched := s.check.Decision(ctx, req.ShortSummary, &authorID)
	if decision == gate.DecisionDeny {
		return nil, gate.ErrContentBlocked()
	}

	galgameTypeJSON, _ := json.Marshal(req.GalgameType)
	rating := &model.GalgameRating{
		Recommend:    req.Recommend,
		Overall:      req.Overall,
		GalgameType:  galgameTypeJSON,
		PlayStatus:   req.PlayStatus,
		ShortSummary: req.ShortSummary,
		SpoilerLevel: req.SpoilerLevel,
		Art:          req.Art, Story: req.Story, Music: req.Music,
		Character: req.Character, Route: req.Route, System: req.System,
		Voice: req.Voice, ReplayValue: req.ReplayValue,
		GalgameID: req.GalgameID, UserID: userID,
	}
	reward := ratingReward(len(req.ShortSummary))

	txErr := s.ratingRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&model.GalgameLocal{ID: req.GalgameID}).Error; err != nil {
			return err
		}
		if err := s.ratingRepo.Create(tx, rating); err != nil {
			return err
		}
		s.helpers.AdjustMoemoepoint(tx, userID, reward,
			moemoepoint.ReasonContentApproved, moemoepoint.Ref("galgame_rating", rating.ID))
		return nil
	})
	if txErr != nil {
		return nil, errors.ErrInternal("创建评分失败")
	}

	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindGalgameRating, "subject_id", rating.ID, "author_id", userID, "matched", matched)
	}
	s.scan.ScanBg(gate.SubjectKindGalgameRating, strconv.Itoa(rating.ID), req.ShortSummary, int64(userID))

	user, _, _ := s.userClient.User(ctx, userID)
	briefMap := s.fetchGalgameBriefs(ctx, []int{req.GalgameID})

	return &dto.CreatedRating{
		ID:           rating.ID,
		User:         userBriefToDTO(user),
		Recommend:    rating.Recommend,
		Overall:      rating.Overall,
		View:         rating.View,
		GalgameType:  rawJSON(string(rating.GalgameType)),
		PlayStatus:   rating.PlayStatus,
		ShortSummary: rating.ShortSummary,
		SpoilerLevel: rating.SpoilerLevel,
		RatingScores: dto.RatingScores{
			Art: rating.Art, Story: rating.Story, Music: rating.Music,
			Character: rating.Character, Route: rating.Route, System: rating.System,
			Voice: rating.Voice, ReplayValue: rating.ReplayValue,
		},
		LikeCount: 0, IsLiked: false,
		Created: rating.CreatedAt.Format(time.RFC3339),
		Updated: rating.UpdatedAt.Format(time.RFC3339),
		Galgame: dto.RatingGalgameBrief{
			ID:           req.GalgameID,
			ContentLimit: briefMap[req.GalgameID].ContentLimit,
			Name:         briefMap[req.GalgameID].Name,
		},
	}, nil
}

func (s *RatingService) UpdateRating(
	ctx context.Context,
	userID int,
	req *dto.UpdateRatingRequest,
) (int, *errors.AppError) {
	rating, err := s.ratingRepo.FindRatingForWrite(req.GalgameRatingID)
	if err != nil {
		return 0, errors.ErrNotFound("评分不存在")
	}
	if rating.UserID != userID {
		return 0, errors.ErrForbidden("您无权限修改他人评分")
	}

	authorID := int64(rating.UserID)
	decision, matched := s.check.Decision(ctx, req.ShortSummary, &authorID)
	if decision == gate.DecisionDeny {
		return 0, gate.ErrContentBlocked()
	}

	pointDiff := ratingReward(len(req.ShortSummary)) - ratingReward(len(rating.ShortSummary))
	galgameTypeJSON, _ := json.Marshal(req.GalgameType)
	fields := map[string]any{
		"recommend":     req.Recommend,
		"overall":       req.Overall,
		"galgame_type":  galgameTypeJSON,
		"play_status":   req.PlayStatus,
		"short_summary": req.ShortSummary,
		"spoiler_level": req.SpoilerLevel,
		"art":           req.Art, "story": req.Story, "music": req.Music,
		"character": req.Character, "route": req.Route, "system": req.System,
		"voice": req.Voice, "replay_value": req.ReplayValue,
	}

	txErr := s.ratingRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.ratingRepo.Update(tx, req.GalgameRatingID, fields); err != nil {
			return err
		}
		s.helpers.AdjustMoemoepoint(tx, userID, pointDiff,
			moemoepoint.ReasonContentApproved, moemoepoint.Ref("galgame_rating", req.GalgameRatingID))
		return nil
	})
	if txErr != nil {
		return 0, errors.ErrInternal("更新评分失败")
	}

	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindGalgameRating, "subject_id", req.GalgameRatingID, "author_id", rating.UserID, "matched", matched)
	}
	s.scan.ScanBg(gate.SubjectKindGalgameRating, strconv.Itoa(req.GalgameRatingID), req.ShortSummary, int64(rating.UserID))
	return rating.GalgameID, nil
}

func (s *RatingService) DeleteRating(
	userID int, canModerate bool, ratingID int,
) *errors.AppError {
	rating, err := s.ratingRepo.FindRatingForWrite(ratingID)
	if err != nil {
		return errors.ErrNotFound("未找到评分")
	}
	galgameOwner := s.ratingRepo.FindGalgameOwner(rating.GalgameID)
	if rating.UserID != userID && galgameOwner != userID && !canModerate {
		return errors.ErrForbidden("没有删除该评分的权限")
	}

	refund := -ratingReward(len(rating.ShortSummary))
	txErr := s.ratingRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.ratingRepo.DeleteByID(tx, ratingID); err != nil {
			return err
		}
		s.helpers.AdjustMoemoepoint(tx, rating.UserID, refund,
			moemoepoint.ReasonContentRemoved, moemoepoint.Ref("galgame_rating", ratingID))
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("删除评分失败")
	}
	return nil
}

func (s *RatingService) ToggleRatingLike(
	userID int,
	req *dto.ToggleRatingLikeRequest,
) *errors.AppError {
	rating, err := s.ratingRepo.FindRatingForWrite(req.GalgameRatingID)
	if err != nil {
		return errors.ErrNotFound("评分不存在")
	}
	if rating.UserID == userID {
		return errors.ErrBadRequest("不能给自己的评分点赞")
	}

	preview := truncate(rating.ShortSummary, constants.TextPreviewLength)
	txErr := s.ratingRepo.DB().Transaction(func(tx *gorm.DB) error {
		existing, has := s.ratingRepo.FindLike(tx, req.GalgameRatingID, userID)
		var delta int
		if has {
			if err := s.ratingRepo.DeleteLike(tx, existing); err != nil {
				return err
			}
			delta = -1
		} else {
			if err := s.ratingRepo.CreateLike(tx, req.GalgameRatingID, userID); err != nil {
				return err
			}
			delta = 1
		}
		if err := s.ratingRepo.AdjustLikeCount(tx, req.GalgameRatingID, delta); err != nil {
			return err
		}
		s.helpers.AdjustMoemoepoint(tx, rating.UserID, delta,
			moemoepoint.ReasonLiked, moemoepoint.Ref("galgame_rating", req.GalgameRatingID))
		return s.helpers.CreateGalgameMessageWithContent(
			tx, userID, rating.UserID, "liked", preview, rating.GalgameID,
		)
	})
	if txErr != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}

func (s *RatingService) fetchGalgameBriefs(
	ctx context.Context,
	galgameIDs []int,
) map[int]client.GalgameBrief {
	if len(galgameIDs) == 0 {
		return map[int]client.GalgameBrief{}
	}
	m, _ := s.galgameClient.GetBatch(ctx, galgameIDs)
	if m == nil {
		return map[int]client.GalgameBrief{}
	}
	return m
}

func (s *RatingService) fetchGalgameBriefsPublic(
	ctx context.Context,
	galgameIDs []int,
	isSFW bool,
) map[int]client.GalgameBrief {
	if len(galgameIDs) == 0 {
		return map[int]client.GalgameBrief{}
	}
	m, _ := s.galgameClient.GetBatchPublic(ctx, galgameIDs, isSFW)
	if m == nil {
		return map[int]client.GalgameBrief{}
	}
	return m
}

func (s *RatingService) buildRatingGalgame(
	ctx context.Context,
	galgameID int,
) dto.RatingGalgameDetail {
	summary := dto.RatingGalgameDetail{
		ID:       galgameID,
		Official: []dto.RatingOfficial{},
	}
	if d, found, appErr := s.galgameClient.CatalogWorkDetail(ctx, galgameID); appErr == nil && found {
		g := client.CatalogDetailToFull(ctx, d, galgameID)
		s.galgameClient.HydrateOfficialLinks(ctx, &g)
		summary.ID = g.ID
		summary.EffectiveBannerHash = g.EffectiveBannerHash
		summary.EffectiveBannerURL = g.EffectiveBannerURL
		summary.EffectiveBannerWidth = g.EffectiveBannerWidth
		summary.EffectiveBannerHeight = g.EffectiveBannerHeight
		summary.EffectiveBannerThumbhash = g.EffectiveBannerThumbhash
		summary.ContentLimit = g.ContentLimit
		summary.AgeLimit = g.AgeLimit
		summary.OriginalLanguage = g.OriginalLanguage
		summary.Name = g.Name
		summary.NameOriginal = g.NameOriginal
		summary.Official = nextMoeOfficialsToDTO(g.Official)
	}

	sum, count := s.ratingRepo.GalgameRatingStats(galgameID)
	summary.Rating = sum
	summary.RatingCount = count
	return summary
}
