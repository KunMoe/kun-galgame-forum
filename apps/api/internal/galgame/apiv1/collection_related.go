package apiv1

import (
	"cmp"
	"context"
	"log/slog"
	"slices"
	"sync"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/catalogclient"

	"golang.org/x/sync/errgroup"
)

const (
	relatedCollectionsMax = 8
	relatedSeedWorks      = 3
	relatedPerSeed        = 10
)

type relatedCollectionsInput struct {
	CollectionID string `path:"collection_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Catalog folder id."`
	IncludeNSFW  bool   `query:"include_nsfw" default:"false" doc:"When true, collections holding an adult work may be recommended and their art may show adult works. Default false."`
}

func (s *Service) listRelatedCollections(ctx context.Context, in *relatedCollectionsInput) (*collectionSummaryListOutput, error) {
	folder, token, _, err := s.visibleFolder(ctx, in.CollectionID)
	if err != nil {
		return nil, err
	}
	empty := &collectionSummaryListOutput{Body: repr.NewList([]CollectionSummary{}, nil)}
	if folder.Visibility == catalogclient.FolderVisibilityPrivate {
		return empty, nil
	}
	items, iErr := s.loadFolderItems(ctx, folder, token)
	if iErr != nil {
		slog.Warn("collection: related seeds unreadable", "folder_id", folder.ID, "err", iErr)
	}
	slices.SortStableFunc(items, func(a, b catalogclient.FolderItem) int { return cmp.Compare(a.Position, b.Position) })

	var (
		mu     sync.Mutex
		score  = map[int64]int{}
		byID   = map[int64]catalogclient.Folder{}
		gather = func(page *catalogclient.FolderPage) {
			mu.Lock()
			defer mu.Unlock()
			for _, f := range page.Folders {
				if f.ID == folder.ID {
					continue
				}
				score[f.ID]++
				byID[f.ID] = f
			}
		}
	)
	g, gctx := errgroup.WithContext(ctx)
	for _, it := range items[:min(relatedSeedWorks, len(items))] {
		g.Go(func() error {
			page, err := s.catalog.FolderDirectory(gctx, catalogclient.DirectoryQuery{
				Sort: catalogclient.FolderSortFollowersDesc, IncludeNSFW: in.IncludeNSFW,
				ContainsWorkID: it.WorkID, Limit: relatedPerSeed,
			})
			if err != nil {
				slog.Warn("collection: related lookup failed", "folder_id", folder.ID, "work_id", it.WorkID, "err", err)
				return nil
			}
			gather(page)
			return nil
		})
	}
	_ = g.Wait()
	if len(byID) < relatedCollectionsMax {
		page, err := s.catalog.FolderDirectory(ctx, catalogclient.DirectoryQuery{
			Sort: catalogclient.FolderSortFollowersDesc, IncludeNSFW: in.IncludeNSFW, Limit: relatedCollectionsMax * 2,
		})
		if err != nil {
			return nil, mapUserPlane(err, false)
		}
		for _, f := range page.Folders {
			if _, seen := byID[f.ID]; !seen && f.ID != folder.ID {
				byID[f.ID] = f
			}
		}
	}
	picked := make([]catalogclient.Folder, 0, len(byID))
	for _, f := range byID {
		picked = append(picked, f)
	}
	slices.SortFunc(picked, func(a, b catalogclient.Folder) int {
		return cmp.Or(cmp.Compare(score[b.ID], score[a.ID]), cmp.Compare(b.FollowerCount, a.FollowerCount), cmp.Compare(b.ID, a.ID))
	})
	picked = picked[:min(relatedCollectionsMax, len(picked))]
	user := callerUser(ctx)
	out, p := s.summaries(ctx, picked, user, s.followedAmong(ctx, user, picked), in.IncludeNSFW)
	if p != nil {
		return nil, p
	}
	return &collectionSummaryListOutput{Body: repr.NewList(out, nil)}, nil
}
