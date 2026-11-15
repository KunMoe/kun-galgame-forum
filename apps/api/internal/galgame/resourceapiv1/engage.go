package apiv1

import (
	"context"
	"slices"
	"strconv"
	"time"

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
		if err := s.store.CreateResourceMessage(tx, user.ID, row.UserID, "liked", previewOf(links), row.WorkID, row.ID); err != nil {
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
	links, err := s.store.LinkRows(row.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	report := func(verdict, state string) *expiryReportOutput {
		return &expiryReportOutput{Body: GalgameResourceExpiryReport{
			Object: "galgame_resource_expiry_report", ResourceID: repr.ID(row.ID),
			Verdict: verdict, State: state, ExpiredDownloadURLs: expiredURLs(links),
		}}
	}
	if row.Status == 1 {
		return report("dead", "expired"), nil
	}

	var pending []int
	for i, l := range links {
		if l.ExpiredAt == nil {
			pending = append(pending, i)
		}
	}
	alive, unknown := false, len(links) == 0
	var deadURLs []string
	var deadLinkIDs []int
	if checker := s.shareCheck(); checker != nil && len(pending) > 0 {
		urls := make([]string, len(pending))
		for j, i := range pending {
			urls[j] = links[i].URL
		}
		for j, st := range checker.CheckEach(ctx, urls, row.Code) {
			switch st {
			case linkcheck.StatusAlive:
				alive = true
			case linkcheck.StatusDead:
				deadLinkIDs = append(deadLinkIDs, links[pending[j]].ID)
				deadURLs = append(deadURLs, urls[j])
			default:
				unknown = true
			}
		}
	} else if len(pending) > 0 {
		unknown = true
	}

	if alive && len(deadLinkIDs) == 0 {
		return report("alive", "valid"), nil
	}
	verdict, state := "partial", "valid"
	if !alive {
		verdict, state = "dead", "expired"
		if unknown {
			verdict = "unchecked"
		}
	}
	preview := deadURLs
	if len(preview) == 0 {
		preview = linkURLs(links)
	}
	now := time.Now()
	err = s.store.InTx(func(tx *gorm.DB) error {
		if err := s.store.MarkLinksExpired(tx, deadLinkIDs, now); err != nil {
			return err
		}
		if state == "expired" {
			if err := s.store.SetStatus(tx, row.ID, 1); err != nil {
				return err
			}
		}
		return s.store.CreateResourceMessage(tx, user.ID, row.UserID, "expired", previewOf(preview), row.WorkID, row.ID)
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	for i := range links {
		if links[i].ExpiredAt == nil && slices.Contains(deadLinkIDs, links[i].ID) {
			links[i].ExpiredAt = &now
		}
	}
	return report(verdict, state), nil
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
