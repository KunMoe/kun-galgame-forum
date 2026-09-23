package apiv1

import (
	"context"
	"strconv"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/linkcheck"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

func (s *Service) putGalgameResourceLike(ctx context.Context, in *likeInput) (*engagementOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	row, _, p := s.visibleResource(ctx, in.ResourceID)
	if p != nil {
		return nil, p
	}
	if user.ID == row.UserID {
		return nil, selfLikeForbidden()
	}
	links, err := s.store.Links(row.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	var jobs []pendingAward
	err = s.store.InTx(func(tx *gorm.DB) error {
		jobs = jobs[:0]
		id, inserted, err := s.store.InsertLike(tx, row.ID, user.ID)
		if err != nil || !inserted {
			return err
		}
		if err := s.store.AdjustLikeCount(tx, row.ID, 1); err != nil {
			return err
		}
		if err := s.store.CreateGalgameMessage(tx, user.ID, row.UserID, "liked", previewOf(links), row.WorkID); err != nil {
			return err
		}
		jobs = append(jobs, pendingAward{
			userID: row.UserID,
			delta:  1,
			reason: moemoepoint.ReasonLiked,
			ref:    moemoepoint.Ref("galgame_resource", row.ID),
			key:    moemoepoint.Key("liked", "galgame_resource_like_"+strconv.FormatInt(id, 10)),
		})
		return nil
	})
	if err := s.afterCommit(err, jobs); err != nil {
		return nil, problem.Internal(err)
	}
	return s.engagementOut(row.ID, user.ID)
}

func (s *Service) deleteGalgameResourceLike(ctx context.Context, in *likeInput) (*engagementOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	row, _, p := s.visibleResource(ctx, in.ResourceID)
	if p != nil {
		return nil, p
	}
	var jobs []pendingAward
	err := s.store.InTx(func(tx *gorm.DB) error {
		jobs = jobs[:0]
		id, deleted, err := s.store.DeleteLike(tx, row.ID, user.ID)
		if err != nil || !deleted {
			return err
		}
		if err := s.store.AdjustLikeCount(tx, row.ID, -1); err != nil {
			return err
		}
		if user.ID == row.UserID {
			return nil
		}
		jobs = append(jobs, pendingAward{
			userID: row.UserID,
			delta:  -1,
			reason: moemoepoint.ReasonLiked,
			ref:    moemoepoint.Ref("galgame_resource", row.ID),
			key:    moemoepoint.Key("unliked", "galgame_resource_like_"+strconv.FormatInt(id, 10)),
		})
		return nil
	})
	if err := s.afterCommit(err, jobs); err != nil {
		return nil, problem.Internal(err)
	}
	return s.engagementOut(row.ID, user.ID)
}

func (s *Service) engagementOut(resourceID, userID int) (*engagementOutput, error) {
	n, err := s.store.LikeCount(resourceID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	liked, err := s.store.HasLiked(resourceID, userID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &engagementOutput{Body: GalgameResourceEngagement{
		Object: "galgame_resource_engagement", ResourceID: repr.ID(resourceID), LikeCount: n,
		Viewer: &GalgameResourceEngagementViewer{HasLiked: liked},
	}}, nil
}

func (s *Service) createGalgameResourceExpiryReport(ctx context.Context, in *expiryReportInput) (*expiryReportOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	row, _, p := s.visibleResource(ctx, in.ResourceID)
	if p != nil {
		return nil, p
	}
	if row.Status == 1 {
		return &expiryReportOutput{Body: GalgameResourceExpiryReport{
			Object: "galgame_resource_expiry_report", ResourceID: repr.ID(row.ID),
			Verdict: "dead", State: "expired",
		}}, nil
	}
	links, err := s.store.Links(row.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	verdict := "unchecked"
	checker := s.shareCheck()
	if checker != nil && len(links) > 0 {
		switch checker.CheckShare(ctx, links, row.Code) {
		case linkcheck.StatusAlive:
			return &expiryReportOutput{Body: GalgameResourceExpiryReport{
				Object: "galgame_resource_expiry_report", ResourceID: repr.ID(row.ID),
				Verdict: "alive", State: "valid",
			}}, nil
		case linkcheck.StatusDead:
			verdict = "dead"
		default:
			verdict = "unchecked"
		}
	}
	err = s.store.InTx(func(tx *gorm.DB) error {
		if err := s.store.SetStatus(tx, row.ID, 1); err != nil {
			return err
		}
		return s.store.CreateGalgameMessage(tx, user.ID, row.UserID, "expired", previewOf(links), row.WorkID)
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &expiryReportOutput{Body: GalgameResourceExpiryReport{
		Object: "galgame_resource_expiry_report", ResourceID: repr.ID(row.ID),
		Verdict: verdict, State: "expired",
	}}, nil
}

func (s *Service) putWorkResourcePublishBan(ctx context.Context, in *workIDInput) (*publishBanOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	if !user.Can(perm.GalgameBanResourcePublish) {
		return nil, permissionRequired()
	}
	workID, ok := parseID(in.WorkID)
	if !ok {
		return nil, notFound()
	}
	if _, p := s.lookupWork(ctx, workID); p != nil {
		return nil, p
	}
	if err := s.store.PutPublishBan(workID); err != nil {
		return nil, problem.Internal(err)
	}
	return &publishBanOutput{Body: WorkResourcePublishBan{
		Object: "work_resource_publish_ban", WorkID: repr.ID(workID), IsResourcePublishBanned: true,
	}}, nil
}

func (s *Service) deleteWorkResourcePublishBan(ctx context.Context, in *workIDInput) (*publishBanOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	if !user.Can(perm.GalgameBanResourcePublish) {
		return nil, permissionRequired()
	}
	workID, ok := parseID(in.WorkID)
	if !ok {
		return nil, notFound()
	}
	if _, p := s.lookupWork(ctx, workID); p != nil {
		return nil, p
	}
	if err := s.store.ClearPublishBan(workID); err != nil {
		return nil, problem.Internal(err)
	}
	banned, err := s.store.IsPublishBanned(workID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &publishBanOutput{Body: WorkResourcePublishBan{
		Object: "work_resource_publish_ban", WorkID: repr.ID(workID), IsResourcePublishBanned: banned,
	}}, nil
}
