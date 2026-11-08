// backfill-collection-feed seeds the home feed's Gal 收藏夹 tab from the
// catalog folder directory. The forum records collection activity only as it
// happens, so before this ran the tab was empty for folders made earlier.
//
// Per directory folder it inserts a COLLECTION_CREATION row at the folder's
// created_at and, when the folder reports recent works, one
// COLLECTION_WORK_ADDITION row for the newest of them at its updated_at.
// Existing rows are left alone (ON CONFLICT DO NOTHING), so a rerun, or a run
// after live activity has been recorded, changes nothing that is already there.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"

	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/logger"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	dryRun := flag.Bool("dry-run", false, "Count what would be written without writing")
	maxFolders := flag.Int("max", 5000, "Stop after this many directory folders")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}
	logger.Init(cfg.Server.Mode)

	cat := catalogclient.New(catalogclient.Config{BaseURL: cfg.Catalog.BaseURL, AppKey: cfg.NextMoeAPI.APIKey})
	if !cat.AppConfigured() {
		slog.Error("catalog client not configured: set KUN_CATALOG_API_BASE and KUN_NEXTMOE_API_KEY")
		os.Exit(1)
	}
	store := repository.NewWorkV1Store(database.NewPostgres(cfg.Database, cfg.Server.Mode))

	ctx := context.Background()
	cursor := ""
	seen, created, added := 0, 0, 0
	for seen < *maxFolders {
		page, err := cat.FolderDirectory(ctx, catalogclient.DirectoryQuery{
			Sort: catalogclient.FolderSortCreatedDesc, IncludeNSFW: true, Cursor: cursor, Limit: 100,
		})
		if err != nil {
			slog.Error("读取收藏夹目录失败", "error", err, "seen", seen)
			os.Exit(1)
		}
		for _, f := range page.Folders {
			seen++
			nsfw := f.ContentRating == "r18"
			at, ok := parseTime(f.CreatedAt)
			if !ok {
				slog.Warn("跳过: created_at 无法解析", "folder_id", f.ID, "value", f.CreatedAt)
				continue
			}
			if *dryRun {
				created++
				if len(f.RecentWorkIDs) > 0 {
					added++
				}
				continue
			}
			ok, err := store.InsertCollectionFeedIfAbsent(repository.FeedCollectionCreation, f.ID, int(f.OwnerUID), 0, f.Name, nsfw, at)
			if err != nil {
				slog.Error("写入失败", "folder_id", f.ID, "error", err)
				os.Exit(1)
			}
			if ok {
				created++
			}
			if len(f.RecentWorkIDs) == 0 {
				continue
			}
			updated, okU := parseTime(f.UpdatedAt)
			if !okU {
				updated = at
			}
			ok, err = store.InsertCollectionFeedIfAbsent(repository.FeedCollectionWorkAddition, f.ID, int(f.OwnerUID),
				int(f.RecentWorkIDs[0]), f.Name, nsfw, updated)
			if err != nil {
				slog.Error("写入失败", "folder_id", f.ID, "error", err)
				os.Exit(1)
			}
			if ok {
				added++
			}
		}
		if page.NextCursor == "" || len(page.Folders) == 0 {
			break
		}
		cursor = page.NextCursor
	}
	slog.Info("回填完成", "folders", seen, "creation_rows", created, "addition_rows", added, "dry_run", *dryRun)
}

func parseTime(raw string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339Nano, raw)
	return t, err == nil
}
