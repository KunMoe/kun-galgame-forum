package service

import (
	"context"
	"log/slog"
	"time"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/catalogclient"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type GalgameEditRevisionSync struct {
	catalog       *catalogclient.Client
	galgameClient *client.GalgameClient
	db            *gorm.DB
	rdb           *redis.Client
	batch         int
}

func NewGalgameEditRevisionSync(
	catalog *catalogclient.Client,
	galgameClient *client.GalgameClient,
	db *gorm.DB,
	rdb *redis.Client,
) *GalgameEditRevisionSync {
	return &GalgameEditRevisionSync{
		catalog: catalog, galgameClient: galgameClient,
		db: db, rdb: rdb, batch: editRevisionBatch,
	}
}

const (
	editRevisionCursorKey  = "catalog:rev:cron:since"
	editRevisionEntityType = catalogclient.EntityTypeWork
	editRevisionBatch      = 100
	editRevisionMaxPages   = 50
)

func (s *GalgameEditRevisionSync) Run() {
	if s.catalog == nil || !s.catalog.AppConfigured() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	since, seeded, err := s.readCursor(ctx)
	if err != nil {
		slog.Warn("读取 catalog 修订游标失败 (本轮跳过)", "error", err)
		return
	}
	if !seeded {
		head, err := s.feedHead(ctx)
		if err != nil {
			slog.Warn("catalog 修订 feed 定位队首失败, 下一轮重试", "error", err)
			return
		}
		s.writeCursor(ctx, head)
		slog.Info("catalog 修订同步已初始化游标 (不回填历史)", "head", head)
		return
	}

	maxSeen := since
	startedFrom := since
	applied := 0

	for pages := 1; pages <= editRevisionMaxPages; pages++ {
		page, err := s.catalog.EditRevisionsSince(ctx, maxSeen, s.batch, editRevisionEntityType)
		if err != nil {
			slog.Warn("catalog 修订 feed 拉取失败", "error", err, "since", maxSeen)
			break
		}
		if len(page.Items) == 0 {
			break
		}

		holding := false
		for i := range page.Items {
			rev := &page.Items[i]
			if err := s.upsert(rev, int(rev.EntityID)); err != nil {
				slog.Warn("catalog 修订入库失败, 持有游标重试", "rev_id", rev.ID, "error", err)
				holding = true
				break
			}
			if rev.ID > maxSeen {
				maxSeen = rev.ID
			}
			applied++
		}
		if holding || len(page.Items) < s.batch {
			break
		}
	}

	if maxSeen > startedFrom {
		s.writeCursor(ctx, maxSeen)
		slog.Info("catalog 修订同步完成", "from", startedFrom, "to", maxSeen, "applied", applied)
	}
}

func (s *GalgameEditRevisionSync) feedHead(ctx context.Context) (int64, error) {
	var head int64
	for pages := 1; pages <= editRevisionMaxPages; pages++ {
		page, err := s.catalog.EditRevisionsSince(ctx, head, s.batch, editRevisionEntityType)
		if err != nil {
			return 0, err
		}
		if len(page.Items) == 0 {
			break
		}
		head = page.Items[len(page.Items)-1].ID
		if len(page.Items) < s.batch {
			break
		}
	}
	return head, nil
}

func isTimelineEdit(action int16) bool {
	return action != catalogclient.EditActionCreated
}

func (s *GalgameEditRevisionSync) upsert(rev *catalogclient.EditRevisionFeedItem, workID int) error {
	if !isTimelineEdit(rev.Action) {
		return nil
	}
	if workID == 0 {
		return nil
	}
	return s.db.Exec(`
		INSERT INTO galgame_activity (edit_revision_id, wiki_revision_number, work_id, user_id, type, created)
		VALUES (?, ?, ?, ?, 'GALGAME_EDIT', ?)
		ON CONFLICT (edit_revision_id) DO NOTHING
	`, rev.ID, rev.Seq, workID, rev.ActorUID, rev.CreatedAt).Error
}

func (s *GalgameEditRevisionSync) readCursor(ctx context.Context) (int64, bool, error) {
	v, err := s.rdb.Get(ctx, editRevisionCursorKey).Int64()
	if err == redis.Nil {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return v, true, nil
}

func (s *GalgameEditRevisionSync) writeCursor(ctx context.Context, id int64) {
	if err := s.rdb.Set(ctx, editRevisionCursorKey, id, 0).Err(); err != nil {
		slog.Warn("写入 catalog 修订游标失败", "id", id, "error", err)
	}
}
