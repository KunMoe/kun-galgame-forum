package service

import (
	"context"
	"log/slog"
	"maps"
	"slices"
	"sync"
	"time"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/utils"

	"github.com/redis/go-redis/v9"
)

const (
	mirrorSyncTimeout = 15 * time.Minute

	// Named for content_limit because that is what it was when it was written,
	// and renaming it would clear the cursor — which restarts the drain at the
	// head of catalog's inventory for no gain.
	mirrorCursorKey = "catalog:changes:content-limit:cursor"

	// 100 pages a tick is 10000 works, and the ten-minute beat gives 60000 an
	// hour — enough that the 110k-work night of a machine re-grade drains inside
	// two hours, and enough to walk the whole inventory in about the same time.
	mirrorChannelPages = 100

	// One catalog request a tick. On the two-minute beat that re-asks the forum's
	// ~16k rows about every five and a half hours, and stays far under the
	// 100/min the /v2 limiter allows the site's single egress address.
	mirrorVerifyRows = 100

	// A tick that would unlist more than this many rows unlists none of them.
	// Catalog has changed how a read face answers under the forum before, and a
	// library blanked by one bad answer is worse than one that is a few hours stale.
	mirrorFlipFloor   = 20
	mirrorFlipPercent = 5
)

type mergeQueue interface {
	Enqueue(ctx context.Context, oldID int, survivor int64)
}

// Keeps the local galgame row's copies of catalog's data in step: content_limit
// (the editorial display verdict) and release_date.
//
// Both exist for the same reason — /galgame and the entity pages page ids in SQL
// and only then hydrate them from catalog, so a column the WHERE and the ORDER
// BY can reach has to live here — and both are catalog's to change, so neither
// survives being written once and left alone. release_date was left alone: its
// backfill command went out with the retired wiki lanes in 157d7abb, and by
// 2026-09-07 half the rows on /galgame had none, which is what "sorted by
// release date" was actually sorting.
//
// Two lanes, and they answer different questions. The mirror channel carries
// every catalog-side change: an editor flipping display_nsfw or correcting a
// date reaches the local lists within a tick. The verify lane re-asks about
// every local row in turn, because the channel alone left rows wrong for good:
// a stub created this minute has no change to its name, a row created after its
// work was merged away never hears of it, a non-galgame medium is on no /v2 face,
// and on 2026-09-24 ten verdicts had changed upstream without reaching the forum.
type GalgameCatalogMirror struct {
	galgameClient *client.GalgameClient
	galgameRepo   *repository.GalgameRepository
	rdb           *redis.Client
	merges        mergeQueue
	maxPages      int
	verifyRows    int

	running sync.Mutex
}

func NewGalgameCatalogMirror(
	galgameClient *client.GalgameClient,
	galgameRepo *repository.GalgameRepository,
	rdb *redis.Client,
	merges mergeQueue,
) *GalgameCatalogMirror {
	return &GalgameCatalogMirror{
		galgameClient: galgameClient,
		galgameRepo:   galgameRepo,
		rdb:           rdb,
		merges:        merges,
		maxPages:      mirrorChannelPages,
		verifyRows:    mirrorVerifyRows,
	}
}

// RunMirror drains catalog's changes feed from the stored cursor and applies the
// mirrored fields of every changed work the forum has a row for.
//
// The cursor advances one page at a time and only after that page's rows are in
// the database, so a failure mid-drain keeps the pages already applied and
// resumes at the first one that is not — the whole-round abort of the old sweep
// cost every chunk after the first error.
//
// The last page of a drain has no next_cursor, so the stored cursor stays on the
// page before it and that page is read again next tick. That re-read is the only
// way to notice rows appended since, and re-applying it is free: both UPDATEs
// are keyed IS DISTINCT FROM.
func (s *GalgameCatalogMirror) RunMirror() {
	if s.galgameClient == nil || s.rdb == nil {
		return
	}
	if !s.running.TryLock() {
		slog.Info("catalog 镜像同步仍在进行, 跳过本轮", "mode", "信道")
		return
	}
	defer s.running.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), mirrorSyncTimeout)
	defer cancel()

	cursor, err := s.rdb.Get(ctx, mirrorCursorKey).Result()
	if err != nil && err != redis.Nil {
		slog.Warn("读取 catalog 变更游标失败 (本轮跳过)", "error", err)
		return
	}
	bootstrap := cursor == ""

	var changed, gone, matched, limits, dates, unlisted int64
	pages := 0
	for ; pages < s.maxPages; pages++ {
		page, appErr := s.galgameClient.CatalogChanges(ctx, cursor, client.CatalogChangesLimit)
		if appErr != nil {
			slog.Warn("catalog 变更信道拉取失败, 游标保持不动", "pages", pages, "error", appErr.Message)
			break
		}
		if len(page.Items) == 0 && page.NextCursor == "" {
			break
		}
		ids := make([]int64, 0, len(page.Items))
		var goneIDs []int
		for _, it := range page.Items {
			if it.Gone {
				gone++
				goneIDs = append(goneIDs, int(it.ID))
				continue
			}
			ids = append(ids, it.ID)
		}
		changed += int64(len(page.Items))

		rows, hidden, appErr := s.galgameClient.MirrorByCatalogIDs(ctx, ids)
		if appErr != nil {
			slog.Warn("catalog 变更条目水合失败, 游标保持不动", "pages", pages, "error", appErr.Message)
			break
		}
		matched += int64(len(rows))
		n, d, err := s.apply(rows)
		if err != nil {
			slog.Warn("catalog 镜像入库失败, 游标保持不动", "pages", pages, "error", err)
			break
		}
		limits += n
		dates += d
		u, err := s.settle(ctx, hidden, goneIDs, len(page.Items), "信道")
		if err != nil {
			slog.Warn("catalog 镜像入库失败, 游标保持不动", "pages", pages, "error", err)
			break
		}
		unlisted += u

		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
		if err := s.rdb.Set(ctx, mirrorCursorKey, cursor, 0).Err(); err != nil {
			slog.Warn("写入 catalog 变更游标失败, 下轮将重放本页", "error", err)
			break
		}
	}

	if changed == 0 {
		return
	}
	slog.Info("galgame catalog 镜像信道同步完成", "bootstrap", bootstrap,
		"pages", pages, "changed", changed, "gone", gone, "matched", matched,
		"content_limit", limits, "release_date", dates, "unlisted", unlisted)
}

// RunVerify re-asks catalog about the local rows it has gone longest without
// asking about. Only an answer changes a row: a failed request leaves both
// catalog_rendered and catalog_checked_at alone, so the next tick asks again.
func (s *GalgameCatalogMirror) RunVerify() {
	if s.galgameClient == nil || s.galgameRepo == nil {
		return
	}
	ids := s.galgameRepo.MirrorVerifyIDs(s.verifyRows)
	if len(ids) == 0 {
		return
	}
	if !s.running.TryLock() {
		slog.Info("catalog 镜像同步仍在进行, 跳过本轮", "mode", "核对")
		return
	}
	defer s.running.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), mirrorSyncTimeout)
	defer cancel()
	s.verify(ctx, ids)
}

func (s *GalgameCatalogMirror) verify(ctx context.Context, ids []int) {
	asked := make([]int64, len(ids))
	for i, id := range ids {
		asked[i] = int64(id)
	}
	rows, hidden, appErr := s.galgameClient.MirrorByCatalogIDs(ctx, asked)
	if appErr != nil {
		slog.Warn("catalog 镜像核对拉取失败, 本轮不改任何行", "requested", len(ids), "error", appErr.Message)
		return
	}
	limits, dates, err := s.apply(rows)
	if err != nil {
		slog.Warn("catalog 镜像核对入库失败", "error", err)
		return
	}
	answered := make(map[int]bool, len(rows)+len(hidden))
	for id := range rows {
		answered[id] = true
	}
	for _, id := range hidden {
		answered[id] = true
	}
	var absent []int
	for _, id := range ids {
		if !answered[id] {
			absent = append(absent, id)
		}
	}
	unlisted, err := s.settle(ctx, hidden, absent, len(ids), "核对")
	if err != nil {
		slog.Warn("catalog 镜像核对入库失败", "error", err)
		return
	}
	slog.Info("galgame catalog 镜像核对完成", "requested", len(ids), "rendered", len(rows),
		"unlisted", unlisted, "content_limit", limits, "release_date", dates)
}

// settle unlists the rows catalog answered for but will not render. hidden came
// back under a hidden claim. absent did not come back at all, and each local one
// is asked about on the detail face: a merge with a survivor catalog renders is
// handed to merge-fold, and anything but an answer leaves the row untouched.
func (s *GalgameCatalogMirror) settle(ctx context.Context, hidden, absent []int, asked int, lane string) (int64, error) {
	if s.galgameRepo == nil {
		return 0, nil
	}
	absent, err := s.galgameRepo.LocalAmong(absent)
	if err != nil {
		return 0, err
	}
	hidden, err = s.galgameRepo.LocalAmong(hidden)
	if err != nil {
		return 0, err
	}
	flips, err := s.galgameRepo.RenderedAmong(append(slices.Clone(hidden), absent...))
	if err != nil {
		return 0, err
	}
	if limit := max(mirrorFlipFloor, (asked*mirrorFlipPercent+99)/100); len(flips) > limit {
		slog.Warn("catalog 镜像将一次下架过多条目, 本轮全部不下架",
			"lane", lane, "asked", asked, "would_unlist", len(flips), "limit", limit)
		return 0, nil
	}

	unrendered := slices.Clone(hidden)
	for _, id := range absent {
		movedTo, found, appErr := s.galgameClient.WorkFate(ctx, int64(id))
		switch {
		case appErr != nil:
			slog.Warn("catalog 作品去向查询失败, 本行不动", "work_id", id, "error", appErr.Message)
			continue
		case found:
			slog.Warn("catalog 批量面与详情面对同一作品答法不同, 本行不动", "work_id", id)
			continue
		case movedTo != 0:
			survivor, _, appErr := s.galgameClient.MirrorByCatalogIDs(ctx, []int64{movedTo})
			if appErr != nil {
				slog.Warn("catalog 合并幸存条目查询失败, 本行不动", "work_id", id, "survivor", movedTo, "error", appErr.Message)
				continue
			}
			if _, ok := survivor[int(movedTo)]; ok && s.merges != nil {
				s.merges.Enqueue(ctx, id, movedTo)
			}
		}
		unrendered = append(unrendered, id)
	}
	if err := s.galgameRepo.MarkCatalogChecked(unrendered, false); err != nil {
		return 0, err
	}
	return int64(len(unrendered)), nil
}

// apply writes one hydrated batch. The two columns are written separately
// because they disagree about what an unusable answer is: a verdict outside
// sfw/nsfw is dropped, while "no date" is a date the row has to record.
func (s *GalgameCatalogMirror) apply(rows map[int]client.CatalogMirror) (int64, int64, error) {
	if s.galgameRepo == nil || len(rows) == 0 {
		return 0, 0, nil
	}
	limits := make(map[int]string, len(rows))
	dates := make(map[int]string, len(rows))
	for workID, row := range rows {
		limits[workID] = row.ContentLimit
		// Recorded as "no date" rather than left unconfirmed: the row cannot be
		// ordered by a string nothing can read, and leaving it pending would
		// re-ask and re-warn about it on every tick for as long as it exists.
		date, ok := utils.NormalizeCatalogReleaseDate(row.ReleaseDate)
		if !ok {
			slog.Warn("catalog 发售日期无法解析, 按无日期记录",
				"work_id", workID, "release_date", row.ReleaseDate)
		}
		dates[workID] = date
	}
	limitCount, err := s.galgameRepo.SetContentLimits(groupByContentLimit(limits))
	if err != nil {
		return limitCount, 0, err
	}
	dateCount, err := s.galgameRepo.SetReleaseDates(dates)
	if err != nil {
		return limitCount, dateCount, err
	}
	return limitCount, dateCount, s.galgameRepo.MarkCatalogChecked(slices.Collect(maps.Keys(rows)), true)
}

func groupByContentLimit(limits map[int]string) map[string][]int {
	out := make(map[string][]int, 2)
	for workID, limit := range limits {
		switch limit {
		case "sfw", "nsfw":
			out[limit] = append(out[limit], workID)
		}
	}
	return out
}
