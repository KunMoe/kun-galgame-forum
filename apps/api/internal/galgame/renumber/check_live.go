package renumber

import (
	"context"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"
)

const liveBatch = 100

type LiveResolver func(ctx context.Context, gids []int) (map[int]int64, error)

func CheckLive(ctx context.Context, db *gorm.DB, m *Map, resolve LiveResolver) (*Report, error) {
	if m == nil {
		return nil, fmt.Errorf("map is nil")
	}
	if resolve == nil {
		return nil, fmt.Errorf("live resolver is nil")
	}
	rep := newReport()
	rep.Mode = "check-live"
	rep.TSVRows = m.TSVRows()
	rep.HowCounts = m.HowCounts()
	rep.MapLen = m.Len()
	rep.LiveMismatches = []LiveMismatch{}

	var ids []int64
	if err := db.Raw("SELECT id FROM galgame ORDER BY id").Scan(&ids).Error; err != nil {
		return rep, fmt.Errorf("list galgame ids: %w", err)
	}
	gids := make([]int, 0, len(ids))
	for _, id := range ids {
		n, err := intID(id)
		if err != nil {
			return rep, err
		}
		gids = append(gids, n)
	}

	live := make(map[int]int64, len(gids))
	for i := 0; i < len(gids); i += liveBatch {
		end := i + liveBatch
		if end > len(gids) {
			end = len(gids)
		}
		got, err := resolve(ctx, gids[i:end])
		if err != nil {
			return rep, fmt.Errorf("catalog work ids: %w", err)
		}
		for gid, id := range got {
			live[gid] = id
		}
	}

	for _, gid := range gids {
		old := int64(gid)
		mapNew, named := m.Lookup(old)
		if !named {
			mapNew = old
		}
		liveNew, found := live[gid]
		if !found {
			liveNew = 0
		}
		if liveNew != mapNew {
			rep.LiveMismatches = append(rep.LiveMismatches, LiveMismatch{
				OldID: old, MapNewID: mapNew, LiveNewID: liveNew,
			})
		}
	}
	sort.Slice(rep.LiveMismatches, func(i, j int) bool {
		return rep.LiveMismatches[i].OldID < rep.LiveMismatches[j].OldID
	})
	rep.Ended = time.Now()
	return rep, nil
}
