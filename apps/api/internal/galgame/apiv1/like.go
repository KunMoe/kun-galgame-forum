package apiv1

import (
	"context"
	"strconv"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

type workLikeInput struct {
	WorkID string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Work id."`
}

type workEngagementOutput struct {
	Body WorkEngagement
}

func (s *Service) requireActive(ctx context.Context) (*middleware.UserInfo, *problem.Problem) {
	user := v1.User(ctx)
	if user == nil {
		return nil, problem.New(problem.CodeMissingCredential, "The request has no credentials.")
	}
	if s.users == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	users, err := s.users.Users(ctx, []int{user.ID})
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	if u, ok := users[user.ID]; ok && !userclient.IsRenderable(u) {
		return nil, problem.New(problem.CodeAccountBanned, "The signed-in user's account is banned.")
	}
	return user, nil
}

func (s *Service) catalogWork(ctx context.Context, workID int) (*client.CatalogWorkListItem, *problem.Problem) {
	if s.works == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	rows, appErr := s.works.CatalogRowsByWorkIDs(ctx, []int{workID}, "names", "all")
	if appErr != nil {
		return nil, catalogUnavailable(appErr)
	}
	row, ok := rows[workID]
	if !ok || !client.CatalogItemRenderable(&row) {
		return nil, notFound()
	}
	cp := row
	return &cp, nil
}

func (s *Service) localCreator(workID int) (int, *problem.Problem) {
	if s.store == nil || !s.store.Ready() {
		return 0, nil
	}
	local, ok, err := s.store.FindLocal(workID)
	if err != nil {
		return 0, problem.Internal(err)
	}
	if !ok || local.CreatorUserID == nil {
		return 0, nil
	}
	return *local.CreatorUserID, nil
}

func (s *Service) putWorkLike(ctx context.Context, in *workLikeInput) (*workEngagementOutput, error) {
	if s == nil || s.store == nil || !s.store.Ready() {
		return nil, problem.Internal(errUnconfigured)
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	workID, ok := parseWorkID(in.WorkID)
	if !ok {
		return nil, notFound()
	}
	row, p := s.catalogWork(ctx, workID)
	if p != nil {
		return nil, p
	}
	ownerID, p := s.localCreator(workID)
	if p != nil {
		return nil, p
	}
	if ownerID != 0 && ownerID == user.ID {
		return nil, selfLikeForbidden()
	}
	preview := workNamePreview(ctx, row)
	var jobs []pendingAward
	err := s.store.InTx(func(tx *gorm.DB) error {
		jobs = jobs[:0]
		if err := s.store.EnsureLocal(tx, workID); err != nil {
			return err
		}
		id, inserted, err := s.store.InsertLike(tx, user.ID, workID)
		if err != nil || !inserted {
			return err
		}
		if err := s.store.AdjustLikeCount(tx, workID, 1); err != nil {
			return err
		}
		if ownerID <= 0 {
			return nil
		}
		if err := s.store.CreateLikedMessage(tx, user.ID, ownerID, preview, workID); err != nil {
			return err
		}
		jobs = append(jobs, pendingAward{
			userID: ownerID, delta: 1, reason: moemoepoint.ReasonLiked,
			ref: moemoepoint.Ref("galgame", workID),
			key: moemoepoint.Key("liked", "galgame_like_"+strconv.FormatInt(id, 10)),
		})
		return nil
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	s.flushAwards(jobs)
	return s.workEngagement(workID, user.ID)
}

func (s *Service) deleteWorkLike(ctx context.Context, in *workLikeInput) (*workEngagementOutput, error) {
	if s == nil || s.store == nil || !s.store.Ready() {
		return nil, problem.Internal(errUnconfigured)
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	workID, ok := parseWorkID(in.WorkID)
	if !ok {
		return nil, notFound()
	}
	if _, p := s.catalogWork(ctx, workID); p != nil {
		return nil, p
	}
	ownerID, p := s.localCreator(workID)
	if p != nil {
		return nil, p
	}
	var jobs []pendingAward
	err := s.store.InTx(func(tx *gorm.DB) error {
		jobs = jobs[:0]
		id, deleted, err := s.store.DeleteLike(tx, user.ID, workID)
		if err != nil || !deleted {
			return err
		}
		if err := s.store.AdjustLikeCount(tx, workID, -1); err != nil {
			return err
		}
		if ownerID <= 0 {
			return nil
		}
		jobs = append(jobs, pendingAward{
			userID: ownerID, delta: -1, reason: moemoepoint.ReasonLiked,
			ref: moemoepoint.Ref("galgame", workID),
			key: moemoepoint.Key("unliked", "galgame_like_"+strconv.FormatInt(id, 10)),
		})
		return nil
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	s.flushAwards(jobs)
	return s.workEngagement(workID, user.ID)
}

func (s *Service) workEngagement(workID, userID int) (*workEngagementOutput, error) {
	n, err := s.store.LikeCount(workID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	liked, err := s.store.HasLiked(userID, workID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &workEngagementOutput{Body: WorkEngagement{
		Object: "work_engagement", WorkID: repr.ID(workID), LikeCount: max(n, 0),
		Viewer: &WorkEngagementViewer{HasLiked: liked},
	}}, nil
}
