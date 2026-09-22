package service

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"strconv"
	"sync"
	"time"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"

	"github.com/redis/go-redis/v9"
)

const (
	mergeSyncTimeout = 15 * time.Minute

	// Losing this key replays catalog's whole merge history, which is not a
	// fault: a fold whose local row is already gone is a no-op. Deleting it is
	// also how an operator forces the replay, which is what -replay does.
	MergeCursorKey = "catalog:redirects:merge:cursor"

	// A gid whose survivor cannot be resolved YET. The cursor moves on whether
	// or not a page folded anything, so without somewhere to park these the one
	// merge that arrived early is dropped and never looked at again — which is
	// not hypothetical: on 2026-08-29 gid 206987 was merged into work 226964,
	// and that work is claimed, undeleted, and still absent from the works face,
	// so the survivor gid 63095 does not resolve either.
	mergeDeferredKey = "catalog:redirects:merge:deferred"

	// 100 pages of 100 is 10,000 merges a tick. Catalog's entire work-merge
	// history was 3,270 rows when this was written, so the first drain finishes
	// in one tick and every later tick reads one nearly empty page.
	mergeSyncPages = 100
)

// Folds a forum galgame row into the row that survived catalog's merge.
//
// Catalog owns duplicate detection and the forum has no say in it, but the merge
// deletes the binding the forum navigates by: retireSource nulls the source
// work's site and product_work_id and soft-deletes the row, so the gid resolves
// to nothing and its page 404s while still holding resources, ratings and
// collection entries. /v2/catalog/redirects is the only place the survivor is
// named after that, and it names it in catalog ids.
type galgameMerger interface {
	LocalIDsIn(ids []int) []int
	Fold(oldGID, newGID int) (repository.MergeCounts, error)
	RedirectTarget(oldGID int) (int, bool)
}

type GalgameMergeSync struct {
	galgameClient *client.GalgameClient
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

		f, d, err := s.applyPage(ctx, page.Items)
		folded += f
		deferred += d
		if err != nil {
			slog.Warn("galgame 合并应用失败, 游标保持不动", "pages", pages, "error", err)
			break
		}

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

func (s *GalgameMergeSync) applyPage(ctx context.Context, items []client.CatalogRedirect) (folded, deferred int, err error) {
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
	for rawGID, rawWork := range parked {
		gid, err1 := strconv.Atoi(rawGID)
		work, err2 := strconv.ParseInt(rawWork, 10, 64)
		if err1 != nil || err2 != nil {
			s.unparkRaw(ctx, rawGID)
			continue
		}
		survivorWork[gid] = work
	}
	folded, deferred, err = s.fold(ctx, survivorWork)
	if err != nil {
		slog.Warn("待合并暂存重试失败", "error", err)
	}
	return folded, deferred
}

// fold takes dead-gid candidates paired with the catalog work that replaced
// them, proves each pairing, and applies it. A second pass then folds forum
// pages whose curated ref moved onto the survivor — those gids still resolve,
// so the dead-gid pass drops them. Anything it cannot prove YET is parked;
// anything it can prove is wrong is dropped with a reason.
func (s *GalgameMergeSync) fold(ctx context.Context, survivorWork map[int]int64) (folded, deferred int, err error) {
	if len(survivorWork) == 0 {
		return 0, 0, nil
	}
	olds := make([]int, 0, len(survivorWork))
	for gid := range survivorWork {
		olds = append(olds, gid)
	}
	candidates := s.mergeRepo.LocalIDsIn(olds)
	// A parked gid whose local row is gone was folded by another path; nothing
	// is left to move, so stop retrying it.
	if len(candidates) < len(olds) {
		live := make(map[int]bool, len(candidates))
		for _, gid := range candidates {
			live[gid] = true
		}
		for _, gid := range olds {
			if !live[gid] {
				s.unpark(ctx, gid)
			}
		}
	}

	var dead []int
	if len(candidates) > 0 {
		// A local gid whose catalog work is merely coincidental with a merged-away
		// catalog id must not be touched, and 10,289 of the forum's gids are also
		// the catalog id of a different work. The gid that lost its work is the one
		// that now resolves to nothing, so this negative IS the proof.
		s.galgameClient.ForgetGIDs(candidates)
		alive, appErr := s.galgameClient.GIDsToCatalogIDs(ctx, candidates)
		if appErr != nil {
			return 0, 0, errors.New(appErr.Message)
		}
		dead = make([]int, 0, len(candidates))
		for _, gid := range candidates {
			if _, ok := alive[gid]; ok {
				s.unpark(ctx, gid)
				continue
			}
			dead = append(dead, gid)
		}
	}

	works := uniqueCatalogIDs(survivorWork)
	if len(works) == 0 {
		return 0, 0, nil
	}

	gidOfWork, appErr := s.galgameClient.GIDsForCatalogIDs(ctx, works)
	if appErr != nil {
		return 0, 0, errors.New(appErr.Message)
	}
	roundTrip := make([]int, 0, len(gidOfWork))
	for _, gid := range gidOfWork {
		roundTrip = append(roundTrip, gid)
	}
	back, appErr := s.galgameClient.GIDsToCatalogIDs(ctx, roundTrip)
	if appErr != nil {
		return 0, 0, errors.New(appErr.Message)
	}

	for _, oldGID := range dead {
		work := survivorWork[oldGID]
		newGID, ok := gidOfWork[work]
		if !ok {
			// Not a wrong answer, a missing one: catalog can hold a claimed,
			// undeleted work the works face still will not return, and its gid
			// is 404 on the forum too. Folding into a page nobody can open would
			// bury the content instead of moving it.
			slog.Warn("幸存作品暂时解析不出 gid, 已暂存待重试", "old_gid", oldGID, "work", work)
			s.park(ctx, oldGID, work)
			deferred++
			continue
		}
		// An unclaimed survivor answers its own catalog id as the gid, which is
		// a guess until it round-trips: without this the fold walks into
		// whichever unrelated forum game happens to carry that number.
		if back[newGID] != work {
			slog.Warn("幸存 gid 未能回指同一作品, 已暂存待重试",
				"old_gid", oldGID, "work", work, "new_gid", newGID, "resolved_work", back[newGID])
			s.park(ctx, oldGID, work)
			deferred++
			continue
		}
		n, d := s.commitFold(ctx, oldGID, newGID, work)
		folded += n
		deferred += d
	}

	sf, sd, err := s.foldStale(ctx, works, gidOfWork, back)
	folded += sf
	deferred += sd
	return folded, deferred, err
}

func uniqueCatalogIDs(survivorWork map[int]int64) []int64 {
	seen := make(map[int64]bool, len(survivorWork))
	out := make([]int64, 0, len(survivorWork))
	for _, work := range survivorWork {
		if work <= 0 || seen[work] {
			continue
		}
		seen[work] = true
		out = append(out, work)
	}
	return out
}

func (s *GalgameMergeSync) foldStale(
	ctx context.Context, works []int64, gidOfWork map[int64]int, back map[int]int64,
) (folded, deferred int, err error) {
	stale, appErr := s.galgameClient.StaleGIDsForCatalogIDs(ctx, works)
	if appErr != nil {
		return 0, 0, errors.New(appErr.Message)
	}

	candidates := make([]int, 0)
	seen := map[int]bool{}
	add := func(gid int) {
		if gid > 0 && !seen[gid] {
			seen[gid] = true
			candidates = append(candidates, gid)
		}
	}
	for _, work := range works {
		for _, gid := range stale[work] {
			add(gid)
		}
		c := gidOfWork[work]
		if wgid := int(work); int64(wgid) == work && wgid != c {
			add(wgid)
		}
	}
	if len(candidates) == 0 {
		return 0, 0, nil
	}

	s.galgameClient.ForgetGIDs(candidates)

	probe := make([]int, len(candidates), len(candidates)+len(gidOfWork))
	copy(probe, candidates)
	for _, c := range gidOfWork {
		probe = append(probe, c)
	}
	local := make(map[int]bool, len(probe))
	for _, gid := range s.mergeRepo.LocalIDsIn(probe) {
		local[gid] = true
	}

	var liveStale []int
	for _, gid := range candidates {
		if local[gid] {
			liveStale = append(liveStale, gid)
		}
	}
	if len(liveStale) == 0 {
		return 0, 0, nil
	}

	resolved, appErr := s.galgameClient.GIDsToCatalogIDs(ctx, liveStale)
	if appErr != nil {
		return 0, 0, errors.New(appErr.Message)
	}

	for _, work := range works {
		c, ok := gidOfWork[work]
		proved := ok && back[c] == work
		wgid := int(work)
		staleGIDs := append([]int(nil), stale[work]...)
		if int64(wgid) == work && wgid != c {
			staleGIDs = append(staleGIDs, wgid)
		}
		for _, staleGID := range staleGIDs {
			if staleGID == c || !local[staleGID] {
				continue
			}
			if !proved {
				s.park(ctx, staleGID, work)
				deferred++
				continue
			}
			if resolved[staleGID] != work {
				continue
			}
			// Fold seeds a missing survivor from the dead row. That is the
			// common path for a catalog id that never had a forum page; doing
			// it here would move a populated page onto a gid the claim named
			// but nobody has opened.
			if !local[c] {
				slog.Warn("canonical gid has no local row, skipping fold into nothing",
					"stale_gid", staleGID, "canonical_gid", c, "work", work)
				continue
			}
			n, d := s.commitFold(ctx, staleGID, c, work)
			folded += n
			deferred += d
			if n > 0 {
				delete(local, staleGID)
			}
		}
	}
	return folded, deferred, nil
}

func (s *GalgameMergeSync) commitFold(ctx context.Context, oldGID, newGID int, work int64) (folded, deferred int) {
	if target, chained := s.mergeRepo.RedirectTarget(newGID); chained {
		newGID = target
	}
	if newGID == oldGID {
		s.unpark(ctx, oldGID)
		return 0, 0
	}

	counts, ferr := s.mergeRepo.Fold(oldGID, newGID)
	if ferr != nil {
		slog.Warn("galgame 合并失败, 已暂存待重试", "old_gid", oldGID, "new_gid", newGID, "error", ferr)
		s.park(ctx, oldGID, work)
		return 0, 1
	}
	s.galgameClient.ForgetGIDs([]int{oldGID})
	s.unpark(ctx, oldGID)
	fields := []any{"old_gid", oldGID, "new_gid", newGID, "work", work,
		"moved", counts.Moved, "dropped", counts.Dropped}
	if counts.Dropped > 0 {
		fields = append(fields, "dropped_rows_archived_in", "galgame_merge_discarded")
	}
	slog.Info("galgame 已并入幸存条目", fields...)
	// The comment thread is anchored site_game:<gid> inside infra's community
	// service and does not move with the forum's rows. Nothing here can
	// re-anchor it, so name it instead of losing it quietly.
	if counts.Comments > 0 {
		slog.Warn("被合并条目的评论区留在了原锚点, 需要 infra 侧改锚",
			"comments", counts.Comments,
			"old_anchor", "site_game:"+strconv.Itoa(oldGID),
			"new_anchor", "site_game:"+strconv.Itoa(newGID))
	}
	return 1, 0
}

func (s *GalgameMergeSync) park(ctx context.Context, oldGID int, work int64) {
	if s.rdb == nil {
		return
	}
	if err := s.rdb.HSet(ctx, mergeDeferredKey,
		strconv.Itoa(oldGID), strconv.FormatInt(work, 10)).Err(); err != nil {
		slog.Warn("暂存待合并条目失败, 该合并将丢失", "old_gid", oldGID, "work", work, "error", err)
	}
}

func (s *GalgameMergeSync) unpark(ctx context.Context, oldGID int) {
	s.unparkRaw(ctx, strconv.Itoa(oldGID))
}

func (s *GalgameMergeSync) unparkRaw(ctx context.Context, field string) {
	if s.rdb == nil {
		return
	}
	if err := s.rdb.HDel(ctx, mergeDeferredKey, field).Err(); err != nil {
		slog.Warn("清除待合并暂存失败", "old_gid", field, "error", err)
	}
}
