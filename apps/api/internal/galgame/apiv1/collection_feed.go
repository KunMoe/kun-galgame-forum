package apiv1

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/catalogclient"
)

func (s *Service) feedReady() bool {
	return s.store != nil && s.store.Ready()
}

func (s *Service) feedCollectionCreated(folder catalogclient.Folder, userID int) {
	if !s.feedReady() || folder.Visibility != catalogclient.FolderVisibilityPublic {
		return
	}
	if err := s.store.UpsertCollectionFeed(repository.FeedCollectionCreation, folder.ID, userID, 0,
		folder.Name, false, time.Now()); err != nil {
		slog.Warn("collection: feed write failed", "folder_id", folder.ID, "err", err)
	}
}

// One row per folder: a run of additions moves it to the top instead of
// filling the tab with one row per work.
func (s *Service) feedCollectionWorkAdded(ctx context.Context, folderID int64, userID, workID int, work *client.CatalogWorkListItem) {
	if !s.feedReady() || s.catalog == nil {
		return
	}
	folder, err := s.catalog.PublicFolder(ctx, folderID)
	if errors.Is(err, catalogclient.ErrNotFound) {
		return
	}
	if err != nil {
		slog.Warn("collection: feed folder unreadable", "folder_id", folderID, "err", err)
		return
	}
	if folder.Visibility != catalogclient.FolderVisibilityPublic {
		return
	}
	nsfw := folder.ContentRating == "r18" || workNSFW(work)
	if err := s.store.UpsertCollectionFeed(repository.FeedCollectionWorkAddition, folderID, userID, workID,
		folder.Name, nsfw, time.Now()); err != nil {
		slog.Warn("collection: feed write failed", "folder_id", folderID, "err", err)
	}
}

func (s *Service) feedCollectionWithdrawn(folderID int64) {
	if !s.feedReady() {
		return
	}
	if err := s.store.DeleteCollectionFeed(folderID); err != nil {
		slog.Warn("collection: feed withdraw failed", "folder_id", folderID, "err", err)
	}
}
