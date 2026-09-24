package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"kun-galgame-api/internal/admin/repository"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/role"
	"kun-galgame-api/pkg/userclient"
)

type PurgeService struct {
	repo       *repository.PurgeRepository
	userClient *userclient.Client
	community  *communityclient.Client
	catalog    *catalogclient.Client
}

func NewPurgeService(repo *repository.PurgeRepository, userClient *userclient.Client,
	community *communityclient.Client, catalog *catalogclient.Client) *PurgeService {
	return &PurgeService{repo: repo, userClient: userClient, community: community, catalog: catalog}
}

var (
	ErrPurgeTargetProtected = errors.New("purge: the target holds the moderation capability")
	ErrPurgeLotteryDrawing  = repository.ErrLotteryDrawing
)

type UnavailableError struct {
	Stage string
	Err   error
}

func (e *UnavailableError) Error() string { return "purge " + e.Stage + ": " + e.Err.Error() }
func (e *UnavailableError) Unwrap() error { return e.Err }

type UserContentPreview struct {
	Counts         repository.UserContentCounts
	CommunityPosts *int64
	Protected      bool
	AccountActive  bool
}

type target struct {
	protected bool
	active    bool
}

// Staff are never purged: their content includes site documentation other
// users read, and the purge exists for spam accounts. A lookup error refuses;
// a user OAuth does not know is a gone account and purgeable.
func (s *PurgeService) lookupTarget(ctx context.Context, userID int) (target, error) {
	s.userClient.Invalidate(userID)
	u, found, err := s.userClient.User(ctx, userID)
	if err != nil {
		return target{}, &UnavailableError{Stage: "account lookup", Err: err}
	}
	return target{
		protected: found && role.CanModerate(u.Roles),
		active:    found && u.Status == 0,
	}, nil
}

func (s *PurgeService) Preview(ctx context.Context, userID int) (UserContentPreview, error) {
	t, err := s.lookupTarget(ctx, userID)
	if err != nil {
		return UserContentPreview{}, err
	}
	counts, err := s.repo.CountUserContent(userID)
	if err != nil {
		return UserContentPreview{}, err
	}
	preview := UserContentPreview{Counts: counts, Protected: t.protected, AccountActive: t.active}
	resp, err := s.community.AuthorStats(ctx, []int64{int64(userID)})
	if err != nil {
		slog.Warn("purge preview: community author stats unavailable", "target_id", userID, "error", err)
		return preview, nil
	}
	var posts int64
	for _, st := range resp.Stats {
		if st.AuthorID == int64(userID) {
			posts = st.VisiblePosts
			break
		}
	}
	preview.CommunityPosts = &posts
	return preview, nil
}

// Purge deletes locally first, then the catalog folders, then the community
// posts. The remote steps are idempotent and the local one finds nothing the
// second time, so an operator retries a purge that failed after the local
// commit. operatorToken is the acting admin's own access token: the catalog
// judges their standing instead of taking this site's word for it.
func (s *PurgeService) Purge(ctx context.Context, operatorID, userID int, operatorToken string) error {
	t, err := s.lookupTarget(ctx, userID)
	if err != nil {
		return err
	}
	if t.protected {
		return ErrPurgeTargetProtected
	}
	receipt, err := s.repo.PurgeUserContent(userID, operatorID)
	if err != nil {
		return err
	}
	var archived int64
	for _, n := range receipt.Archived {
		archived += n
	}
	slog.Info("purge: local content purged and archived",
		"purge_id", receipt.PurgeID, "operator_id", operatorID, "target_id", userID,
		"archived_rows", archived, "archived_by_table", receipt.Archived)

	folders, err := s.catalog.PurgeUserFolders(ctx, operatorToken, int64(userID))
	if err != nil {
		slog.Error("purge: catalog folder purge failed; local rows are purged, retry the purge",
			"purge_id", receipt.PurgeID, "operator_id", operatorID, "target_id", userID, "error", err)
		return remoteError("catalog folders", err, catalogRetryable(err))
	}

	purged, err := s.community.AuthorPurge(ctx, int64(userID))
	if err != nil {
		slog.Error("purge: community AuthorPurge failed; local rows and folders are purged, retry the purge",
			"purge_id", receipt.PurgeID, "operator_id", operatorID, "target_id", userID, "error", err)
		return remoteError("community posts", err, communityRetryable(err))
	}

	slog.Info("purge: user content purged",
		"purge_id", receipt.PurgeID, "operator_id", operatorID, "target_id", userID,
		"archived_rows", archived,
		"catalog_folders_deleted", folders.FoldersDeleted,
		"catalog_folder_items_deleted", folders.ItemsDeleted,
		"community_posts_purged", purged.PostsPurged,
		"community_reactions_deleted", purged.ReactionsDeleted,
		"community_anchor_subscriptions_deleted", purged.AnchorSubscriptionsDeleted,
		"community_notifications_deleted", purged.NotificationsDeleted)
	return nil
}

// An upstream 4xx means this site sent a request it should not have, or the
// operator lacks standing there: a retry cannot fix it, so it is not a 503.
func remoteError(stage string, err error, retryable bool) error {
	if retryable {
		return &UnavailableError{Stage: stage, Err: err}
	}
	return fmt.Errorf("purge %s: %w", stage, err)
}

func catalogRetryable(err error) bool {
	var apiErr *catalogclient.UserAPIError
	if errors.As(err, &apiErr) {
		return apiErr.Status == http.StatusTooManyRequests
	}
	return !errors.Is(err, catalogclient.ErrUnauthorized) &&
		!errors.Is(err, catalogclient.ErrInsufficientScope) &&
		!errors.Is(err, catalogclient.ErrNotFound) &&
		!errors.Is(err, catalogclient.ErrNotConfigured)
}

func communityRetryable(err error) bool {
	return !errors.Is(err, communityclient.ErrForbidden) && !errors.Is(err, communityclient.ErrNotConfigured)
}
