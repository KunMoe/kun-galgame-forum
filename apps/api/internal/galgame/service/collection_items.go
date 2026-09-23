package service

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"strconv"
	"time"

	"kun-galgame-api/pkg/catalogclient"
)

const folderItemsCacheTTL = 6 * time.Hour

type cachedFolderItem struct {
	WorkID    int64  `json:"w"`
	CreatedAt string `json:"c"`
}

func folderItemsCacheKey(folder *catalogclient.Folder) string {
	return "kungal:folder-items:v2:" + strconv.FormatInt(folder.ID, 10) + ":" + folder.UpdatedAt
}

// loadFolderContents prefers the public lane even when the viewer owns the
// folder. A public folder is public to its owner too, and reading it with
// their token is what walked 3,560 items through /v2/me/folders/{id}/items
// on every page of /galgame/collection/4535 (user 90769, 2026-09-20).
func (s *CollectionService) loadFolderContents(
	ctx context.Context, folderID int64, isOwner bool, token string,
) (*catalogclient.Folder, []catalogclient.FolderItem, error) {
	folder, err := s.catalog.PublicFolder(ctx, folderID)
	if err == nil {
		items, iErr := s.folderItems(ctx, folder, "")
		return folder, items, iErr
	}
	if !isOwner || token == "" || !stderrors.Is(err, catalogclient.ErrNotFound) {
		return nil, nil, err
	}
	folder, err = s.catalog.MyFolder(ctx, token, folderID)
	if err != nil {
		return nil, nil, err
	}
	items, iErr := s.folderItems(ctx, folder, token)
	return folder, items, iErr
}

func (s *CollectionService) folderItems(
	ctx context.Context, folder *catalogclient.Folder, token string,
) ([]catalogclient.FolderItem, error) {
	if folder == nil || folder.ItemCount == 0 {
		return []catalogclient.FolderItem{}, nil
	}
	if items, ok := s.cachedFolderItems(ctx, folder); ok {
		return items, nil
	}
	var (
		items []catalogclient.FolderItem
		err   error
	)
	if token != "" {
		items, err = s.catalog.MyFolderItems(ctx, token, folder.ID)
	} else {
		items, err = s.catalog.PublicFolderItems(ctx, folder.ID)
	}
	if err != nil {
		return nil, err
	}
	s.storeFolderItems(ctx, folder, items)
	return items, nil
}

func (s *CollectionService) cachedFolderItems(ctx context.Context, folder *catalogclient.Folder) ([]catalogclient.FolderItem, bool) {
	if s.rdb == nil {
		return nil, false
	}
	raw, err := s.rdb.Get(ctx, folderItemsCacheKey(folder)).Bytes()
	if err != nil {
		return nil, false
	}
	var rows []cachedFolderItem
	if json.Unmarshal(raw, &rows) != nil {
		return nil, false
	}
	items := make([]catalogclient.FolderItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, catalogclient.FolderItem{
			FolderID: folder.ID, WorkID: row.WorkID, CreatedAt: row.CreatedAt,
		})
	}
	return items, true
}

func (s *CollectionService) storeFolderItems(ctx context.Context, folder *catalogclient.Folder, items []catalogclient.FolderItem) {
	if s.rdb == nil || folder == nil {
		return
	}
	rows := make([]cachedFolderItem, 0, len(items))
	for _, it := range items {
		rows = append(rows, cachedFolderItem{WorkID: it.WorkID, CreatedAt: it.CreatedAt})
	}
	raw, err := json.Marshal(rows)
	if err != nil {
		return
	}
	_ = s.rdb.Set(ctx, folderItemsCacheKey(folder), raw, folderItemsCacheTTL).Err()
}

func (s *CollectionService) listFoldersForViewer(
	ctx context.Context, ownerID, viewerID int, token string,
) ([]catalogclient.Folder, string, error) {
	if viewerID == ownerID && token != "" {
		folders, err := s.catalog.MyFolders(ctx, token)
		if err == nil {
			return folders, token, nil
		}
		if !isQuotaError(err) {
			return nil, "", err
		}
		folders, err = s.catalog.PublicFolders(ctx, int64(ownerID))
		return folders, "", err
	}
	folders, err := s.catalog.PublicFolders(ctx, int64(ownerID))
	return folders, "", err
}
