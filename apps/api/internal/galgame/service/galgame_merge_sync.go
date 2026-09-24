package service

import (
	"context"
	"log/slog"
	"math"
	"strconv"
	"sync"
	"time"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/errors"

	"github.com/redis/go-redis/v9"
)

const (
	mergeSyncTimeout = 15 * time.Minute

	// Losing this key replays catalog's whole merge history, which is not a
	// fault: a fold whose local row is already gone is a no-op. Deleting it is
	// also how an operator forces the replay, which is what -replay does.
	MergeCursorKey = "catalog:redirects:merge:cursor"

	mergeDeferredKey = "catalog:redirects:merge:deferred:v2"

	// 100 pages of 100 is 10,000 merges a tick. Catalog's entire work-merge
	// history was 3,270 rows when this was written, so the first drain finishes
	// in one tick and every later tick reads one nearly empty page.
	mergeSyncPages = 100
)

// Catalog's merge soft-deletes the retired work, so its page 404s while the
// local row still holds resources, ratings and collection entries.
// /v2/catalog/redirects is the only place the survivor is named after that.
type galgameMerger interface {
	LocalIDsIn(ids []int) []int
	Fold(oldWorkID, newWorkID int) (repository.MergeCounts, error)
}

type survivorHydrator interface {
	MirrorByCatalogIDs(ctx context.Context, ids []int64) (rendered, hidden map[int]client.CatalogMirror, appErr *errors.AppError)
}

type GalgameMergeSync struct {
	galgameClient *client.GalgameClient
	survivors     survivorHydrator
	mergeRepo     galgameMerger
	rdb           *redis.Client
	maxPages      int

	running sync.Mutex
}

func NewGalgameMergeSync(
	galgameClient *client.GalgameClient,
	mergeRepo *repository.GalgameMergeRepository,
	rdb *redis.Client,
) *GalgameMergeSync {
	s := &GalgameMergeSync{
		galgameClient: galgameClient,
		rdb:           rdb,
		maxPages:      mergeSyncPages,
	}
	if galgameClient != nil {
		s.survivors = galgameClient
	}
	if mergeRepo != nil {
		s.mergeRepo = mergeRepo
	}
	return s
}

func (s *GalgameMergeSync) Run() {
	if s.galgameClient == nil || s.mergeRepo == nil || s.rdb == nil {
		return
	}
	if !s.running.TryLock() {
		slog.Info("galgame 合并同步仍在进行, 跳过本轮")
		return
	}
	defer s.running.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), mergeSyncTimeout)
	defer cancel()

	cursor, err := s.rdb.Get(ctx, MergeCursorKey).Result()
	if err != nil && err != redis.Nil {
		slog.Warn("读取 catalog 合并游标失败 (本轮跳过)", "error", err)
		return
	}
	bootstrap := cursor == ""

	folded, deferred := s.retryDeferred(ctx)

	var seen int
	pages := 0
	for ; pages < s.maxPages; pages++ {
		page, appErr := s.galgameClient.CatalogRedirects(ctx, cursor, client.CatalogRedirectsLimit)
		if appErr != nil {
			slog.Warn("catalog 合并信道拉取失败, 游标保持不动", "pages", pages, "error", appErr.Message)
			break
		}
		if len(page.Items) == 0 && page.NextCursor == "" {
			break
		}
		seen += len(page.Items)

		f, d := s.applyPage(ctx, page.Items)
		folded += f
		deferred += d

		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
		if err := s.rdb.Set(ctx, MergeCursorKey, cursor, 0).Err(); err != nil {
			slog.Warn("写入 catalog 合并游标失败, 下轮将重放本页", "error", err)
			break
		}
	}

	if seen == 0 && folded == 0 && deferred == 0 {
		return
	}
	slog.Info("galgame 合并同步完成", "bootstrap", bootstrap,
		"pages", pages, "redirects", seen, "folded", folded, "deferred", deferred)
}

func (s *GalgameMergeSync) applyPage(ctx context.Context, items []client.CatalogRedirect) (folded, deferred int) {
	survivorWork := make(map[int]int64, len(items))
	for _, it := range items {
		if it.OldID > math.MaxInt32 {
			continue
		}
		if _, seen := survivorWork[int(it.OldID)]; !seen {
			survivorWork[int(it.OldID)] = it.CurrentID
		}
	}
	return s.fold(ctx, survivorWork)
}

func (s *GalgameMergeSync) retryDeferred(ctx context.Context) (folded, deferred int) {
	parked, err := s.rdb.HGetAll(ctx, mergeDeferredKey).Result()
	if err != nil && err != redis.Nil {
		slog.Warn("读取待合并暂存失败 (本轮跳过重试)", "error", err)
		return 0, 0
	}
	if len(parked) == 0 {
		return 0, 0
	}
	survivorWork := make(map[int]int64, len(parked))
	for rawOld, rawWork := range parked {
		oldID, err1 := strconv.Atoi(rawOld)
		work, err2 := strconv.ParseInt(rawWork, 10, 64)
		if err1 != nil || err2 != nil {
			s.unparkRaw(ctx, rawOld)
			continue
		}
		survivorWork[oldID] = work
	}
	return s.fold(ctx, survivorWork)
}

func (s *GalgameMergeSync) fold(ctx context.Context, survivorWork map[int]int64) (folded, deferred int) {
	if len(survivorWork) == 0 {
		return 0, 0
	}
	olds := make([]int, 0, len(survivorWork))
	for id := range survivorWork {
		olds = append(olds, id)
	}
	candidates := s.mergeRepo.LocalIDsIn(olds)
	if len(candidates) < len(olds) {
		live := make(map[int]bool, len(candidates))
		for _, id := range candidates {
			live[id] = true
		}
		for _, id := range olds {
			if !live[id] {
				s.unpark(ctx, id)
			}
		}
	}

	renderable := s.renderableSurvivors(ctx, candidates, survivorWork)
	for _, oldID := range candidates {
		work := survivorWork[oldID]
		newID := int(work)
		if work <= 0 || int64(newID) != work {
			s.park(ctx, oldID, work)
			deferred++
			continue
		}
		if newID != oldID && !renderable[newID] {
			s.park(ctx, oldID, work)
			deferred++
			continue
		}
		n, d := s.commitFold(ctx, oldID, newID, work)
		folded += n
		deferred += d
	}
	return folded, deferred
}

// Folding into a survivor catalog will not render moves resources, ratings and
// collection entries onto a page nobody can open: 206987's survivor 226964 is a
// hidden claim. Such a fold, and any batch whose survivors could not be looked
// up, stays parked and is retried.
func (s *GalgameMergeSync) renderableSurvivors(ctx context.Context, olds []int, survivorWork map[int]int64) map[int]bool {
	out := map[int]bool{}
	if s.survivors == nil {
		return out
	}
	var ids []int64
	for _, oldID := range olds {
		if work := survivorWork[oldID]; work > 0 && work != int64(oldID) {
			ids = append(ids, work)
		}
	}
	if len(ids) == 0 {
		return out
	}
	rows, _, appErr := s.survivors.MirrorByCatalogIDs(ctx, ids)
	if appErr != nil {
		slog.Warn("galgame 合并幸存条目查询失败, 本批全部暂存", "survivors", len(ids), "error", appErr.Message)
		return out
	}
	for id := range rows {
		out[id] = true
	}
	return out
}

func (s *GalgameMergeSync) commitFold(ctx context.Context, oldID, newID int, work int64) (folded, deferred int) {
	if newID == oldID {
		s.unpark(ctx, oldID)
		return 0, 0
	}

	counts, ferr := s.mergeRepo.Fold(oldID, newID)
	if ferr != nil {
		slog.Warn("galgame 合并失败, 已暂存待重试", "old_work_id", oldID, "new_work_id", newID, "error", ferr)
		s.park(ctx, oldID, work)
		return 0, 1
	}
	s.unpark(ctx, oldID)
	fields := []any{"old_work_id", oldID, "new_work_id", newID, "work", work,
		"moved", counts.Moved, "dropped", counts.Dropped}
	if counts.Dropped > 0 {
		fields = append(fields, "dropped_rows_archived_in", "galgame_merge_discarded")
	}
	slog.Info("galgame 已并入幸存条目", fields...)
	if counts.Comments > 0 {
		slog.Warn("被合并条目的评论区留在了原锚点, 需要 infra 侧改锚",
			"comments", counts.Comments,
			"old_anchor", "site_game:"+strconv.Itoa(oldID),
			"new_anchor", "site_game:"+strconv.Itoa(newID))
	}
	return 1, 0
}

// Enqueue folds a merge the redirect cursor went past before its local row
// existed: rows created after their work was merged away were never folded.
func (s *GalgameMergeSync) Enqueue(ctx context.Context, oldID int, survivor int64) {
	s.park(ctx, oldID, survivor)
}

func (s *GalgameMergeSync) park(ctx context.Context, oldID int, work int64) {
	if s.rdb == nil {
		return
	}
	if err := s.rdb.HSet(ctx, mergeDeferredKey,
		strconv.Itoa(oldID), strconv.FormatInt(work, 10)).Err(); err != nil {
		slog.Warn("暂存待合并条目失败, 该合并将丢失", "old_work_id", oldID, "work", work, "error", err)
	}
}

func (s *GalgameMergeSync) unpark(ctx context.Context, oldID int) {
	s.unparkRaw(ctx, strconv.Itoa(oldID))
}

func (s *GalgameMergeSync) unparkRaw(ctx context.Context, field string) {
	if s.rdb == nil {
		return
	}
	if err := s.rdb.HDel(ctx, mergeDeferredKey, field).Err(); err != nil {
		slog.Warn("清除待合并暂存失败", "old_work_id", field, "error", err)
	}
}
