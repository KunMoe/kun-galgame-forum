package apiv1

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"

	"golang.org/x/sync/errgroup"
)

func (s *Service) followedAmong(ctx context.Context, user *middleware.UserInfo, folders []catalogclient.Folder) map[int64]bool {
	token := accessToken(ctx)
	if user == nil || token == "" || s.catalog == nil {
		return map[int64]bool{}
	}
	ids := make([]int64, 0, len(folders))
	for _, f := range folders {
		if f.Visibility != catalogclient.FolderVisibilityPrivate && int(f.OwnerUID) != user.ID {
			ids = append(ids, f.ID)
		}
	}
	if len(ids) == 0 {
		return map[int64]bool{}
	}
	got, err := s.catalog.FollowedAmong(ctx, token, ids)
	if err != nil {
		slog.Warn("collection: follow state unreadable", "user_id", user.ID, "upstream_status", upstreamStatus(err), "err", err)
		return map[int64]bool{}
	}
	return got
}

func (s *Service) viewerFollows(ctx context.Context, user *middleware.UserInfo, folder catalogclient.Folder) bool {
	return s.followedAmong(ctx, user, []catalogclient.Folder{folder})[folder.ID]
}

func (s *Service) summaries(ctx context.Context, folders []catalogclient.Folder, user *middleware.UserInfo, following map[int64]bool, includeNSFW bool) ([]CollectionSummary, *problem.Problem) {
	return s.summariesOf(ctx, folders, user, "", func(f catalogclient.Folder) viewerState {
		return viewerState{following: following[f.ID]}
	}, includeNSFW)
}

func (s *Service) bumpCollectionView(folderID int64) int {
	if s.store == nil || !s.store.Ready() {
		return 0
	}
	n, err := s.store.BumpCollectionView(folderID)
	if err != nil {
		slog.Warn("collection: view count not bumped", "folder_id", folderID, "err", err)
	}
	return n
}

const directoryCursorScope = "collection_directory"

func (s *Service) listCollections(ctx context.Context, in *listCollectionsInput) (*collectionSummaryListOutput, error) {
	if s.catalog == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	var workID int64
	if in.ContainsWorkID != "" {
		id, ok := parseWorkID(in.ContainsWorkID)
		if !ok {
			return nil, invalidParameter(problem.AtParameter("contains_work_id", problem.ReasonInvalidFormat,
				"contains_work_id must be a positive decimal integer", nil))
		}
		workID = int64(id)
	}
	fp := collect.Fingerprint(in.Sort, strconv.FormatBool(in.IncludeNSFW), in.ContainsWorkID, strconv.Itoa(in.Limit), in.Tag, in.Q)
	upstream, p := unwrapUpstreamCursor(in.Cursor, directoryCursorScope, fp)
	if p != nil {
		return nil, p
	}
	page, err := s.catalog.FolderDirectory(ctx, catalogclient.DirectoryQuery{
		Sort: in.Sort, IncludeNSFW: in.IncludeNSFW, ContainsWorkID: workID, Cursor: upstream, Limit: in.Limit,
		Tag: strings.TrimSpace(in.Tag), Q: strings.TrimSpace(in.Q),
	})
	if err != nil {
		return nil, mapUserPlane(err, false)
	}
	user := callerUser(ctx)
	items, p := s.summaries(ctx, page.Folders, user, s.followedAmong(ctx, user, page.Folders), in.IncludeNSFW)
	if p != nil {
		return nil, p
	}
	return &collectionSummaryListOutput{Body: repr.NewList(items, wrapUpstreamCursor(directoryCursorScope, fp, page.NextCursor))}, nil
}

const followedCursorScope = "followed_collections"

func (s *Service) listFollowedCollections(ctx context.Context, in *listFollowedCollectionsInput) (*collectionSummaryListOutput, error) {
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	if s.catalog == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	fp := collect.Fingerprint(strconv.Itoa(user.ID), strconv.Itoa(in.Limit))
	upstream, p := unwrapUpstreamCursor(in.Cursor, followedCursorScope, fp)
	if p != nil {
		return nil, p
	}
	page, err := s.catalog.FollowedFolders(ctx, token, upstream, in.Limit)
	if err != nil {
		return nil, mapUserPlane(err, true)
	}
	items, p := s.summariesOf(ctx, page.Folders, user, "", func(f catalogclient.Folder) viewerState {
		return viewerState{following: true, newItems: page.Follows[f.ID].NewItemCount}
	}, in.IncludeNSFW)
	if p != nil {
		return nil, p
	}
	return &collectionSummaryListOutput{Body: repr.NewList(items, wrapUpstreamCursor(followedCursorScope, fp, page.NextCursor))}, nil
}

func (s *Service) followCollection(ctx context.Context, in *collectionFollowInput) (*collectionFollowOutput, error) {
	_, token, folderID, p := s.followPrelude(ctx, in.CollectionID)
	if p != nil {
		return nil, p
	}
	folder, err := s.catalog.FollowFolder(ctx, token, folderID)
	if err != nil {
		return nil, mapUserPlane(err, true)
	}
	return &collectionFollowOutput{Body: followEngagement(folderID, new(max(folder.FollowerCount, 0)), true)}, nil
}

func (s *Service) unfollowCollection(ctx context.Context, in *collectionFollowInput) (*collectionFollowOutput, error) {
	_, token, folderID, p := s.followPrelude(ctx, in.CollectionID)
	if p != nil {
		return nil, p
	}
	if err := s.catalog.UnfollowFolder(ctx, token, folderID); err != nil {
		return nil, mapUserPlane(err, true)
	}
	var count *int
	folder, err := s.catalog.PublicFolder(ctx, folderID)
	switch {
	case err == nil:
		count = new(max(folder.FollowerCount, 0))
	case !errors.Is(err, catalogclient.ErrNotFound):
		slog.Warn("collection: follower count unreadable after unfollow", "folder_id", folderID, "err", err)
	}
	return &collectionFollowOutput{Body: followEngagement(folderID, count, false)}, nil
}

func (s *Service) followPrelude(ctx context.Context, rawID string) (*middleware.UserInfo, string, int64, *problem.Problem) {
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, "", 0, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, "", 0, p
	}
	folderID, ok := parseCollectionID(rawID)
	if !ok {
		return nil, "", 0, notFound()
	}
	if s.catalog == nil {
		return nil, "", 0, problem.Unavailable(errUnconfigured)
	}
	return user, token, folderID, nil
}

func followEngagement(folderID int64, count *int, following bool) CollectionFollowEngagement {
	return CollectionFollowEngagement{
		Object: "collection_follow_engagement", CollectionID: repr.ID(int(folderID)),
		FollowerCount: count, Viewer: &CollectionFollowEngagementViewer{IsFollowing: following},
	}
}

func (s *Service) createCollectionReport(ctx context.Context, in *createCollectionReportInput) (*noContentOutput, error) {
	if _, p := s.requireActive(ctx); p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	if s.catalog == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	report := catalogclient.FolderReport{
		FolderID: in.Body.CollectionID, Reason: in.Body.Reason,
		Note: derefOr(in.Body.Note, ""), WorkID: derefOr(in.Body.WorkID, ""),
	}
	if err := s.catalog.ReportFolder(ctx, token, report, idempotencyKey(ctx)); err != nil {
		var api *catalogclient.UserAPIError
		if errors.As(err, &api) && api.Status == http.StatusTooManyRequests {
			return nil, problem.New(problem.CodeRateLimited, "This account has filed too many reports; try again later.")
		}
		return nil, mapUserPlaneAt(err, true, reportPointer)
	}
	return &noContentOutput{}, nil
}

func reportPointer(upstream string) (string, bool) {
	switch upstream {
	case "/folder_id":
		return "/collection_id", true
	case "/reason":
		return "/report_category", true
	case "/work_id", "/note":
		return upstream, true
	}
	return "", false
}

// CollectionSummariesByID reads public folders one by one: catalog has no ids
// lane on /v2/folders. A folder that is gone or private is simply absent.
func (s *Service) CollectionSummariesByID(ctx context.Context, ids []int64, includeNSFW bool) map[int64]CollectionSummary {
	out := map[int64]CollectionSummary{}
	if s.catalog == nil || len(ids) == 0 {
		return out
	}
	folders := make([]catalogclient.Folder, len(ids))
	found := make([]bool, len(ids))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(previewFetchParallel)
	for i, id := range ids {
		g.Go(func() error {
			f, err := s.catalog.PublicFolder(gctx, id)
			if err != nil {
				if !errors.Is(err, catalogclient.ErrNotFound) {
					slog.Warn("collection: activity folder unreadable", "folder_id", id, "err", err)
				}
				return nil
			}
			folders[i], found[i] = *f, true
			return nil
		})
	}
	_ = g.Wait()
	present := make([]catalogclient.Folder, 0, len(ids))
	for i := range folders {
		if found[i] {
			present = append(present, folders[i])
		}
	}
	items, p := s.summariesOf(ctx, present, nil, "", func(catalogclient.Folder) viewerState { return viewerState{} }, includeNSFW)
	if p != nil {
		slog.Warn("collection: activity summaries failed", "err", p)
		return out
	}
	for _, it := range items {
		if id, ok := repr.ParseID(it.ID); ok {
			out[int64(id)] = it
		}
	}
	return out
}
