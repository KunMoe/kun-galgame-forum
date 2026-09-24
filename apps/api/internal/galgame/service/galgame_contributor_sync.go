package service

import (
	"context"
	"log/slog"
	"time"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/catalogclient"

	"github.com/redis/go-redis/v9"
)

type GalgameContributorSync struct {
	catalog *catalogclient.Client
	repo    *repository.GalgameContributorRepository
	rdb     *redis.Client
	batch   int
}

func NewGalgameContributorSync(
	catalog *catalogclient.Client,
	repo *repository.GalgameContributorRepository,
	rdb *redis.Client,
) *GalgameContributorSync {
	return &GalgameContributorSync{
		catalog: catalog, repo: repo, rdb: rdb, batch: contributorFeedBatch,
	}
}

const (
	contributorCursorKey = "catalog:contrib:cron:since"
	contributorSite      = client.ClaimSiteKungal

	contributorFeedBatch    = 100
	contributorMaxPagesRun  = 50
	contributorFeedPageWait = 10 * time.Minute
)

func (s *GalgameContributorSync) Run() {
	if s.catalog == nil || !s.catalog.AppConfigured() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), contributorFeedPageWait)
	defer cancel()

	since, err := s.readCursor(ctx)
	if err != nil {
		slog.Warn("读取贡献者游标失败 (本轮跳过)", "error", err)
		return
	}

	startedFrom, maxSeen := since, since
	for pages := 1; pages <= contributorMaxPagesRun; pages++ {
		page, err := s.catalog.WorkRevisionsAfter(ctx, maxSeen, s.batch, contributorSite)
		if err != nil {
			slog.Warn("catalog 修订 feed 拉取失败 (贡献者)", "error", err, "after", maxSeen)
			break
		}
		if len(page.Items) == 0 {
			break
		}
		touches, workIDs := contributorTouches(page.Items)
		if err := s.repo.UpsertRevisionTouches(touches); err != nil {
			slog.Warn("贡献者入库失败, 持有游标重试", "after", maxSeen, "error", err)
			break
		}
		if err := s.repo.RefreshContributorCounts(workIDs); err != nil {
			slog.Warn("贡献者计数刷新失败", "galgames", len(workIDs), "error", err)
		}
		maxSeen = maxRevisionID(page.Items, maxSeen)
		if len(page.Items) < s.batch {
			break
		}
	}

	if maxSeen > startedFrom {
		s.writeCursor(ctx, maxSeen)
		slog.Info("贡献者同步完成", "from", startedFrom, "to", maxSeen)
	}
}

func contributorTouches(items []catalogclient.WorkRevisionFeedItem) ([]repository.ContributorTouch, []int64) {
	type key struct{ workID, uid int64 }
	index := map[key]int{}
	touches := make([]repository.ContributorTouch, 0, len(items))
	workIDs := make([]int64, 0, len(items))
	seenWorkID := map[int64]bool{}

	for i := range items {
		it := &items[i]
		if it.WorkID <= 0 {
			continue
		}
		workID := it.WorkID
		if !seenWorkID[workID] {
			seenWorkID[workID] = true
			workIDs = append(workIDs, workID)
		}
		for _, uid := range contributorUIDs(it) {
			k := key{workID, uid}
			if at, ok := index[k]; ok {
				t := &touches[at]
				t.Count++
				if it.CreatedAt.Before(t.FirstAt) {
					t.FirstAt = it.CreatedAt
				}
				if it.CreatedAt.After(t.LastAt) {
					t.LastAt = it.CreatedAt
				}
				continue
			}
			index[k] = len(touches)
			touches = append(touches, repository.ContributorTouch{
				WorkID: workID, UserID: uid, Count: 1,
				FirstAt: it.CreatedAt, LastAt: it.CreatedAt,
			})
		}
	}
	return touches, workIDs
}

func contributorUIDs(it *catalogclient.WorkRevisionFeedItem) []int64 {
	uids := make([]int64, 0, 2)
	if it.ActorUID > 0 {
		uids = append(uids, it.ActorUID)
	}
	if it.AmenderUID != nil && *it.AmenderUID > 0 && *it.AmenderUID != it.ActorUID {
		uids = append(uids, *it.AmenderUID)
	}
	return uids
}

func maxRevisionID(items []catalogclient.WorkRevisionFeedItem, current int64) int64 {
	for i := range items {
		if items[i].ID > current {
			current = items[i].ID
		}
	}
	return current
}

func (s *GalgameContributorSync) readCursor(ctx context.Context) (int64, error) {
	v, err := s.rdb.Get(ctx, contributorCursorKey).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return v, nil
}

func (s *GalgameContributorSync) writeCursor(ctx context.Context, id int64) {
	if err := s.rdb.Set(ctx, contributorCursorKey, id, 0).Err(); err != nil {
		slog.Warn("写入贡献者游标失败", "id", id, "error", err)
	}
}
