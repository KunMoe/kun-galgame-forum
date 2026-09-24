package apiv1

import (
	"context"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"
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
	items := make([]CollectionChoice, 0, pg.Limit)
	for _, f := range pageOfFolders(folders, pg.Page, pg.Limit) {
		if c, ok := s.toChoice(f, user, contains[f.ID]); ok {
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
		Title:      truncateFolderText(folder, "name", folder.Name, 100),
		Visibility: vis, IsDefault: folder.IsDefault, ItemCount: max(folder.ItemCount, 0),
		UpdatedAt: updated, Viewer: s.collectionViewer(user, folder, hasWork),
	}, true
}
