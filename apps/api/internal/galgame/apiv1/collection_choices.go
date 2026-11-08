package apiv1

import (
	"context"
	"log/slog"
	"sync"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"

	"golang.org/x/sync/errgroup"
)

type listMyCollectionsForWorkInput struct {
	WorkID string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Work id."`
	Page   int    `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit  int    `query:"limit" minimum:"1" maximum:"100" default:"24" doc:"Page size. 1–100, default 24. Values above 100 are rejected, not clamped."`
}

type collectionChoicePageOutput struct {
	Body repr.PageList[CollectionChoice]
}

// The picker used /me/collections?work_id=, which reads a preview page per
// folder it never draws: 2+N user-token calls a click, and a 12-folder user
// curating at one work per 8 s ran out of catalog's per-uid bucket
// (2026-09-24 07:05). This face costs 2 whatever N is.
func (s *Service) listMyCollectionsForWork(ctx context.Context, in *listMyCollectionsForWorkInput) (*collectionChoicePageOutput, error) {
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
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
	if s.catalog == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	pg := browsePage(in.Page, in.Limit)
	if prob := pg.CheckDepth(); prob != nil {
		return nil, prob
	}
	folders, err := s.catalog.MyFolders(ctx, token)
	if err != nil {
		return nil, mapUserPlane(err, true)
	}
	contains, err := s.foldersHolding(ctx, token, workID)
	if err != nil {
		return nil, mapUserPlane(err, true)
	}
	sortFolders(folders)
	n, rel := collect.ClampTotal(len(folders))
	page := pageOfFolders(folders, pg.Page, pg.Limit)
	recent := s.recentWorkRefs(ctx, page)
	notes := s.workNotes(ctx, token, page, contains, workID)
	items := make([]CollectionChoice, 0, pg.Limit)
	for _, f := range page {
		if c, ok := s.toChoice(f, user, contains[f.ID]); ok {
			if ref, ok := recent[f.ID]; ok {
				c.RecentWork = &ref
			}
			c.WorkNote = noteOrNull(notes[f.ID])
			items = append(items, c)
		}
	}
	return &collectionChoicePageOutput{Body: repr.NewPageList(items, n, rel)}, nil
}

func (s *Service) toChoice(folder catalogclient.Folder, user *middleware.UserInfo, hasWork bool) (CollectionChoice, bool) {
	vis, ok := folderVisibility(folder.Visibility, folder.ID)
	if !ok {
		return CollectionChoice{}, false
	}
	updated, ok := folderTime(folder.UpdatedAt, folder.ID, "updated_at")
	if !ok {
		return CollectionChoice{}, false
	}
	return CollectionChoice{
		Object: "collection", ID: repr.ID(int(folder.ID)),
		Title:       truncateFolderText(folder, "name", folder.Name, 100),
		Description: truncateFolderText(folder, "description", folder.Description, 500),
		Visibility:  vis, IsDefault: folder.IsDefault, ItemCount: max(folder.ItemCount, 0),
		UpdatedAt: updated, Viewer: s.collectionViewer(user, folder, viewerState{hasWork: hasWork}),
	}, true
}

func (s *Service) recentWorkRefs(ctx context.Context, folders []catalogclient.Folder) map[int64]repr.WorkRef {
	out := map[int64]repr.WorkRef{}
	ids := []int{}
	for _, f := range folders {
		if len(f.RecentWorkIDs) > 0 {
			ids = append(ids, int(f.RecentWorkIDs[0]))
		}
	}
	if len(ids) == 0 || s.hydrator == nil {
		return out
	}
	sums, p := s.hydrator.ByIDs(ctx, ids, true)
	if p != nil {
		slog.Warn("collection: picker covers unreadable", "err", p)
		return out
	}
	byID := make(map[int]repr.WorkRef, len(sums))
	for _, sum := range sums {
		if id, ok := repr.ParseID(sum.ID); ok {
			byID[id] = sum.WorkRef
		}
	}
	for _, f := range folders {
		if len(f.RecentWorkIDs) == 0 {
			continue
		}
		if ref, ok := byID[int(f.RecentWorkIDs[0])]; ok {
			out[f.ID] = ref
		}
	}
	return out
}

func (s *Service) workNotes(ctx context.Context, token string, folders []catalogclient.Folder, holding map[int64]bool, workID int) map[int64]string {
	out := map[int64]string{}
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(previewFetchParallel)
	for _, f := range folders {
		if !holding[f.ID] {
			continue
		}
		g.Go(func() error {
			it, err := s.catalog.MyFolderItem(gctx, token, f.ID, int64(workID))
			if err != nil {
				slog.Warn("collection: picker note unreadable", "folder_id", f.ID, "work_id", workID, "err", err)
				return nil
			}
			mu.Lock()
			out[f.ID] = it.Note
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()
	return out
}
