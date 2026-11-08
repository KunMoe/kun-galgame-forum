package apiv1

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"time"

	"kun-galgame-api/pkg/catalogclient"
)

// Every galgame page reads the directory for the collections holding its work,
// and every collection page reads it again for related ones: catalog saw 20-40
// directory reads per 10 seconds with no forum-side cache (2026-11-08).
const directoryCacheTTL = 30 * time.Second

type cachedDirectoryFolder struct {
	ID            int64    `json:"id"`
	OwnerUID      int64    `json:"owner_uid"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Visibility    string   `json:"visibility"`
	IsDefault     bool     `json:"is_default"`
	ItemCount     int      `json:"item_count"`
	FollowerCount int      `json:"follower_count"`
	ContentRating string   `json:"content_rating"`
	RecentWorkIDs []int64  `json:"recent_work_ids"`
	CoverWorkID   int64    `json:"cover_work_id"`
	CopiedFromID  int64    `json:"copied_from_id"`
	Tags          []string `json:"tags"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

type cachedDirectoryPage struct {
	Folders    []cachedDirectoryFolder `json:"folders"`
	NextCursor string                  `json:"next_cursor"`
}

func directoryCacheKey(q catalogclient.DirectoryQuery) string {
	raw, _ := json.Marshal(q)
	sum := sha256.Sum256(raw)
	return "kungal:folder-directory:v1:" + hex.EncodeToString(sum[:16])
}

func (s *Service) folderDirectory(ctx context.Context, q catalogclient.DirectoryQuery) (*catalogclient.FolderPage, error) {
	key := directoryCacheKey(q)
	if s.rdb != nil {
		if raw, err := s.rdb.Get(ctx, key).Bytes(); err == nil {
			var cached cachedDirectoryPage
			if json.Unmarshal(raw, &cached) == nil {
				return cached.page(), nil
			}
		}
	}
	page, err := s.catalog.FolderDirectory(ctx, q)
	if err != nil {
		return nil, err
	}
	if s.rdb != nil {
		raw, mErr := json.Marshal(cacheDirectoryPage(page))
		if mErr == nil {
			if err := s.rdb.Set(ctx, key, raw, directoryCacheTTL).Err(); err != nil {
				slog.Warn("collection: directory cache write failed", "err", err)
			}
		}
	}
	return page, nil
}

func cacheDirectoryPage(page *catalogclient.FolderPage) cachedDirectoryPage {
	out := cachedDirectoryPage{Folders: make([]cachedDirectoryFolder, 0, len(page.Folders)), NextCursor: page.NextCursor}
	for _, f := range page.Folders {
		out.Folders = append(out.Folders, cachedDirectoryFolder{
			ID: f.ID, OwnerUID: f.OwnerUID, Name: f.Name, Description: f.Description, Visibility: f.Visibility,
			IsDefault: f.IsDefault, ItemCount: f.ItemCount, FollowerCount: f.FollowerCount, ContentRating: f.ContentRating,
			RecentWorkIDs: f.RecentWorkIDs, CoverWorkID: f.CoverWorkID, CopiedFromID: f.CopiedFromID, Tags: f.Tags,
			CreatedAt: f.CreatedAt, UpdatedAt: f.UpdatedAt,
		})
	}
	return out
}

func (c cachedDirectoryPage) page() *catalogclient.FolderPage {
	folders := make([]catalogclient.Folder, 0, len(c.Folders))
	for _, f := range c.Folders {
		folders = append(folders, catalogclient.Folder{
			ID: f.ID, OwnerUID: f.OwnerUID, Name: f.Name, Description: f.Description, Visibility: f.Visibility,
			IsDefault: f.IsDefault, ItemCount: f.ItemCount, FollowerCount: f.FollowerCount, ContentRating: f.ContentRating,
			RecentWorkIDs: f.RecentWorkIDs, CoverWorkID: f.CoverWorkID, CopiedFromID: f.CopiedFromID, Tags: nonNilStrings(f.Tags),
			CreatedAt: f.CreatedAt, UpdatedAt: f.UpdatedAt,
		})
	}
	return &catalogclient.FolderPage{Folders: folders, NextCursor: c.NextCursor}
}

func nonNilStrings(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
