package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/utils"

	"github.com/redis/go-redis/v9"
)

const (
	mirrorSyncChunk   = 500
	mirrorSyncTimeout = 15 * time.Minute

	// Named for content_limit because that is what it was when it was written,
	// and renaming it would clear the cursor — which restarts the drain at the
	// head of catalog's inventory for no gain.
	mirrorCursorKey = "catalog:changes:content-limit:cursor"

	// 100 pages a tick is 10000 works, and the ten-minute beat gives 60000 an
	// hour — enough that the 110k-work night of a machine re-grade drains inside
	// two hours, and enough to walk the whole inventory in about the same time.
	mirrorChannelPages = 100

	// The fill lane asks catalog about local rows by gid, which costs a ref
	// lookup plus a works read per chunk. Two chunks a tick walks the forum's
	// ~11.5k rows in about two hours and stays far under the 100/min the /v2
	// limiter allows the site's single egress address.
	mirrorFillRows = 2 * mirrorSyncChunk

	// A local row catalog has no work for cannot be resolved by asking again a
	// minute later, but it can be resolved by asking again after a submission is
	// approved. The nightly full sweep used to be what re-asked; the mirror
	// channel never will, because a row that is missing upstream is exactly the
	// row the channel has nothing to say about.
	mirrorOrphanTTL = 6 * time.Hour
)

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
// every catalog-side change — an editor flipping display_nsfw or correcting a
// date reaches the local lists within a tick. The fill lane carries the local
// side: a stub row created by a user this minute has no catalog change to its
// name and would otherwise stay unmirrored until its work happens to be touched
// upstream.
type GalgameCatalogMirror struct {
	galgameClient *client.GalgameClient
	galgameRepo   *repository.GalgameRepository
	rdb           *redis.Client
	maxPages      int
	fillRows      int

	running sync.Mutex
	// Local rows catalog has no work for. They stay unmirrored, so without this
	// the ten-minute pass would ask about the same orphans forever.
	unresolvedMu sync.Mutex
	unresolved   map[int]time.Time
}

func NewGalgameCatalogMirror(
	galgameClient *client.GalgameClient,
	galgameRepo *repository.GalgameRepository,
	rdb *redis.Client,
) *GalgameCatalogMirror {
	return &GalgameCatalogMirror{
		galgameClient: galgameClient,
		galgameRepo:   galgameRepo,
		rdb:           rdb,
		maxPages:      mirrorChannelPages,
		fillRows:      mirrorFillRows,
		unresolved:    map[int]time.Time{},
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

	var changed, gone, matched, limits, dates int64
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
		for _, it := range page.Items {
			if it.Gone {
				gone++
				continue
			}
			ids = append(ids, it.ID)
		}
		changed += int64(len(page.Items))

		rows, appErr := s.galgameClient.MirrorByCatalogIDs(ctx, ids)
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
		"content_limit", limits, "release_date", dates)
}

// RunPending resolves rows catalog has not confirmed since the last pass, so a
// game published at noon is filtered and sorted correctly the same day instead
// of leaving a hole in every SFW page and a NULL at the end of every date sort
// until its catalog work next changes.
func (s *GalgameCatalogMirror) RunPending() {
	if s.galgameClient == nil || s.galgameRepo == nil {
		return
	}
	ids := s.galgameRepo.MirrorPendingIDs(s.fillRows, s.memoisedOrphans())
	if len(ids) == 0 {
		return
	}
	if !s.running.TryLock() {
		slog.Info("catalog 镜像同步仍在进行, 跳过本轮", "mode", "增量")
		return
	}
	defer s.running.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), mirrorSyncTimeout)
	defer cancel()

	var seen, limits, dates int64
	for start := 0; start < len(ids); start += mirrorSyncChunk {
		chunk := ids[start:min(start+mirrorSyncChunk, len(ids))]
		rows, appErr := s.galgameClient.MirrorByGIDs(ctx, chunk)
		if appErr != nil {
			slog.Warn("catalog 镜像拉取失败, 本轮中止", "offset", start, "error", appErr.Message)
			break
		}
		n, d, err := s.apply(rows)
		if err != nil {
			slog.Warn("catalog 镜像入库失败, 本轮中止", "offset", start, "error", err)
			break
		}
		s.rememberUnresolved(chunk, rows)
		seen += int64(len(rows))
		limits += n
		dates += d
	}
	slog.Info("galgame catalog 镜像增量同步完成",
		"requested", len(ids), "resolved", seen, "content_limit", limits, "release_date", dates)
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
	for gid, row := range rows {
		limits[gid] = row.ContentLimit
		// Recorded as "no date" rather than left unconfirmed: the row cannot be
		// ordered by a string nothing can read, and leaving it pending would
		// re-ask and re-warn about it on every tick for as long as it exists.
		date, ok := utils.NormalizeCatalogReleaseDate(row.ReleaseDate)
		if !ok {
			slog.Warn("catalog 发售日期无法解析, 按无日期记录",
				"galgame_id", gid, "release_date", row.ReleaseDate)
		}
		dates[gid] = date
	}
	limitCount, err := s.galgameRepo.SetContentLimits(groupByContentLimit(limits))
	if err != nil {
		return limitCount, 0, err
	}
	dateCount, err := s.galgameRepo.SetReleaseDates(dates)
	return limitCount, dateCount, err
}

func (s *GalgameCatalogMirror) memoisedOrphans() []int {
	now := time.Now()
	s.unresolvedMu.Lock()
	defer s.unresolvedMu.Unlock()
	out := make([]int, 0, len(s.unresolved))
	for id, until := range s.unresolved {
		if now.Before(until) {
			out = append(out, id)
		}
	}
	return out
}

func (s *GalgameCatalogMirror) rememberUnresolved(asked []int, got map[int]client.CatalogMirror) {
	until := time.Now().Add(mirrorOrphanTTL)
	s.unresolvedMu.Lock()
	defer s.unresolvedMu.Unlock()
	for _, id := range asked {
		if _, ok := got[id]; ok {
			delete(s.unresolved, id)
			continue
		}
		s.unresolved[id] = until
	}
}

func groupByContentLimit(limits map[int]string) map[string][]int {
	out := make(map[string][]int, 2)
	for gid, limit := range limits {
		switch limit {
		case "sfw", "nsfw":
			out[limit] = append(out[limit], gid)
		}
	}
	return out
}
