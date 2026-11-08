package repository

import (
	"math"
	"strconv"
	"time"
)

const (
	FeedCollectionCreation     = "COLLECTION_CREATION"
	FeedCollectionWorkAddition = "COLLECTION_WORK_ADDITION"
	FeedCollectionComment      = "COLLECTION_COMMENT_CREATION"
)

func (s *WorkV1Store) BumpCollectionView(folderID int64) (int, error) {
	var n int64
	err := s.db.Raw(`INSERT INTO galgame_collection_view (catalog_folder_id, view_count, updated)
		VALUES (?, 1, now())
		ON CONFLICT (catalog_folder_id) DO UPDATE SET view_count = galgame_collection_view.view_count + 1, updated = now()
		RETURNING view_count`, folderID).Scan(&n).Error
	return int(min(n, math.MaxInt32)), err
}

func (s *WorkV1Store) CollectionViews(folderIDs []int64) (map[int64]int, error) {
	out := make(map[int64]int, len(folderIDs))
	if len(folderIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		ID    int64 `gorm:"column:catalog_folder_id"`
		Count int64 `gorm:"column:view_count"`
	}
	if err := s.db.Raw(`SELECT catalog_folder_id, view_count FROM galgame_collection_view WHERE catalog_folder_id IN ?`,
		folderIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ID] = int(min(r.Count, math.MaxInt32))
	}
	return out, nil
}

// feed_activity.source_id is an int; a catalog folder id that outgrows it is
// left out of the feed rather than truncated onto someone else's row.
func (s *WorkV1Store) UpsertCollectionFeed(feedType string, folderID int64, userID, workID int, content string, nsfw bool, at time.Time) error {
	if folderID <= 0 || folderID > math.MaxInt32 {
		return nil
	}
	return s.db.Exec("SELECT feed_upsert(?, ?, ?, ?, ?, ?, ?, ?)",
		feedType, int(folderID), userID, workID, content, collectionLink(folderID), nsfw, at).Error
}

func (s *WorkV1Store) InsertCollectionFeedIfAbsent(feedType string, folderID int64, userID, workID int, content string, nsfw bool, at time.Time) (bool, error) {
	if folderID <= 0 || folderID > math.MaxInt32 {
		return false, nil
	}
	res := s.db.Exec(`INSERT INTO feed_activity (type, source_id, user_id, work_id, content, link, is_nsfw, created)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT (type, source_id) DO NOTHING`,
		feedType, int(folderID), userID, workID, content, collectionLink(folderID), nsfw, at)
	return res.RowsAffected > 0, res.Error
}

func (s *WorkV1Store) DeleteCollectionFeed(folderID int64) error {
	return s.db.Exec(`DELETE FROM feed_activity WHERE type IN ? AND link = ?`,
		[]string{FeedCollectionCreation, FeedCollectionWorkAddition, FeedCollectionComment}, collectionLink(folderID)).Error
}

func collectionLink(folderID int64) string {
	return "/collection/" + strconv.FormatInt(folderID, 10)
}
