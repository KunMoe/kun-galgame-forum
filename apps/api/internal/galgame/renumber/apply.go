package renumber

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"kun-galgame-api/internal/galgame/repository"

	"gorm.io/gorm"
)

const Park int64 = 1_000_000_000

const insertBatch = 1000

type Options struct {
	IncludeDM bool
}

var userTriggerTables = []string{
	"galgame",
	"galgame_resource",
	"galgame_rating",
	"galgame_activity",
	"galgame_quiz",
	"message",
	"topic",
	"topic_reply",
	"topic_comment",
	"todo",
}

type parkCol struct {
	table, col, extra string
}

var parkCols = []parkCol{
	{"galgame", "id", ""},
	{"galgame_like", "galgame_id", ""},
	{"galgame_favorite", "galgame_id", ""},
	{"galgame_rating", "galgame_id", ""},
	{"galgame_resource", "galgame_id", ""},
	{"galgame_activity", "galgame_id", ""},
	{"galgame_quiz", "galgame_id", "galgame_id IS NOT NULL"},
	{"galgame_comment_community_map", "galgame_id", ""},
	{"feed_activity", "galgame_id", ""},
	{"feed_activity", "source_id", "type = 'GALGAME_CREATION'"},
	{"galgame_merge_discarded", "old_gid", ""},
	{"galgame_merge_discarded", "new_gid", ""},
	{"galgame_view_daily", "entity_id", ""},
	{"galgame_collection_item", "galgame_id", ""},
	{"galgame_quiz_galgame", "galgame_id", ""},
	{"galgame_contributor", "galgame_id", ""},
}

type ledgerRow struct {
	OldID int64  `gorm:"column:old_id;primaryKey"`
	NewID int64  `gorm:"column:new_id"`
	How   string `gorm:"column:how"`
}

func (ledgerRow) TableName() string { return "galgame_renumber_2026" }

type galgamePick struct {
	ID            int64     `gorm:"column:id"`
	Published     bool      `gorm:"column:published"`
	ResourceCount int       `gorm:"column:resource_count"`
	Created       time.Time `gorm:"column:created"`
}

func Apply(tx *gorm.DB, m *Map, opts Options) (*Report, error) {
	if m == nil {
		return nil, fmt.Errorf("map is nil")
	}
	rep := newReport()
	if err := runGuards(tx, opts.IncludeDM, rep); err != nil {
		return rep, err
	}
	preIDs, err := loadMapIntoTx(tx, m, rep)
	if err != nil {
		return rep, err
	}
	if err := checkParkRange(tx, rep); err != nil {
		return rep, err
	}
	losers, err := runFolds(tx, m, rep)
	if err != nil {
		return rep, err
	}
	if err := snapshotSums(tx, rep); err != nil {
		return rep, err
	}
	if err := disableTriggers(tx, rep); err != nil {
		return rep, err
	}
	if err := hopAll(tx, rep); err != nil {
		return rep, err
	}
	if err := rewriteAllLinks(tx, m, opts, rep); err != nil {
		return rep, err
	}
	if err := enableTriggers(tx, rep); err != nil {
		return rep, err
	}
	if err := assertInvariants(tx, m, preIDs, losers, rep); err != nil {
		return rep, err
	}
	rep.Ended = time.Now()
	return rep, nil
}

func runGuards(tx *gorm.DB, includeDM bool, rep *Report) error {
	if err := tx.Exec("SET LOCAL lock_timeout = '10s'").Error; err != nil {
		return fmt.Errorf("set lock_timeout: %w", err)
	}
	if err := tx.Exec("LOCK TABLE " + strings.Join(lockTables(includeDM), ", ") + " IN ACCESS EXCLUSIVE MODE").Error; err != nil {
		return fmt.Errorf("lock tables: %w", err)
	}
	for _, c := range parkCols {
		n, err := count(tx, "SELECT COUNT(*) FROM "+c.table)
		if err != nil {
			return fmt.Errorf("count %s: %w", c.table, err)
		}
		key := c.table
		if c.col != "id" && c.col != "galgame_id" {
			key = c.table + "." + c.col
		}
		st := rep.table(key)
		if st.Rows == 0 {
			st.Rows = n
		}
	}
	n, err := count(tx, "SELECT COUNT(*) FROM galgame")
	if err != nil {
		return err
	}
	rep.GalgameBefore = n
	return nil
}

func lockTables(includeDM bool) []string {
	tables := []string{
		"galgame",
		"doc_article",
		"feed_activity",
		"galgame_activity",
		"galgame_collection_item",
		"galgame_comment_community_map",
		"galgame_contributor",
		"galgame_favorite",
		"galgame_like",
		"galgame_merge_discarded",
		"galgame_quiz",
		"galgame_quiz_galgame",
		"galgame_rating",
		"galgame_rating_like",
		"galgame_redirect",
		"galgame_renumber_2026",
		"galgame_resource",
		"galgame_resource_link",
		"galgame_view_daily",
		"message",
		"todo",
		"topic",
		"topic_comment",
		"topic_reply",
	}
	if includeDM {
		tables = append(tables, "chat_message", "chat_room")
	}
	return tables
}

func checkParkRange(tx *gorm.DB, rep *Report) error {
	var maxOld int64
	if err := tx.Raw("SELECT COALESCE(MAX(old_id), 0) FROM galgame_renumber_2026 WHERE old_id <> new_id").
		Scan(&maxOld).Error; err != nil {
		return fmt.Errorf("max changing id: %w", err)
	}
	if maxOld+Park > math.MaxInt32 {
		return fmt.Errorf("changing id %d + park overflows integer", maxOld)
	}
	var occupied []string
	for _, c := range parkCols {
		n, err := count(tx, parkCountSQL(c), Park)
		if err != nil {
			return fmt.Errorf("park check %s.%s: %w", c.table, c.col, err)
		}
		if n > 0 {
			occupied = append(occupied, fmt.Sprintf("%s.%s=%d", c.table, c.col, n))
		}
	}
	if len(occupied) > 0 {
		rep.ParkOccupied = occupied
		return fmt.Errorf("park range occupied: %s", strings.Join(occupied, ", "))
	}
	return nil
}

func parkCountSQL(c parkCol) string {
	q := "SELECT COUNT(*) FROM " + c.table + " WHERE " + c.col +
		" IN (SELECT old_id + ? FROM galgame_renumber_2026 WHERE old_id <> new_id)"
	if c.extra != "" {
		q += " AND " + c.extra
	}
	return q
}

func loadMapIntoTx(tx *gorm.DB, m *Map, rep *Report) ([]int64, error) {
	var ids []int64
	if err := tx.Raw("SELECT id FROM galgame ORDER BY id").Scan(&ids).Error; err != nil {
		return nil, fmt.Errorf("list galgame ids: %w", err)
	}
	m.AddUnchanged(ids)
	rep.MapLen = m.Len()
	rep.TSVRows = m.TSVRows()
	rep.HowCounts = m.HowCounts()

	var n int64
	if err := tx.Raw("SELECT COUNT(*) FROM galgame_renumber_2026").Scan(&n).Error; err != nil {
		return nil, fmt.Errorf("count galgame_renumber_2026: %w", err)
	}
	if n > 0 {
		return nil, fmt.Errorf("galgame_renumber_2026 is not empty (%d row(s))", n)
	}
	entries := m.Entries()
	rows := make([]ledgerRow, len(entries))
	for i, e := range entries {
		rows[i] = ledgerRow{OldID: e.OldID, NewID: e.NewID, How: e.How}
	}
	if len(rows) > 0 {
		if err := tx.CreateInBatches(rows, insertBatch).Error; err != nil {
			return nil, fmt.Errorf("insert galgame_renumber_2026: %w", err)
		}
	}
	return ids, nil
}

func runFolds(tx *gorm.DB, m *Map, rep *Report) (map[int64]bool, error) {
	var rows []galgamePick
	if err := tx.Raw("SELECT id, published, resource_count, created FROM galgame").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("load galgame for folds: %w", err)
	}
	groups := map[int64][]galgamePick{}
	for _, row := range rows {
		newID, ok := m.Lookup(row.ID)
		if !ok {
			newID = row.ID
		}
		groups[newID] = append(groups[newID], row)
	}
	losers := map[int64]bool{}
	newIDs := make([]int64, 0, len(groups))
	for newID, g := range groups {
		if len(g) > 1 {
			newIDs = append(newIDs, newID)
		}
	}
	sort.Slice(newIDs, func(i, j int) bool { return newIDs[i] < newIDs[j] })
	repo := repository.NewGalgameMergeRepository(tx)
	for _, newID := range newIDs {
		g := groups[newID]
		surv := pickSurvivor(g, newID)
		var lost []int64
		var moved, dropped int64
		var comments int
		for _, row := range g {
			if row.ID == surv.ID {
				continue
			}
			oldInt, err := intID(row.ID)
			if err != nil {
				return nil, err
			}
			survInt, err := intID(surv.ID)
			if err != nil {
				return nil, err
			}
			counts, err := repo.FoldTx(tx, oldInt, survInt)
			if err != nil {
				return nil, fmt.Errorf("fold %d -> %d (work %d): %w", row.ID, surv.ID, newID, err)
			}
			losers[row.ID] = true
			lost = append(lost, row.ID)
			moved += counts.Moved
			dropped += counts.Dropped
			comments += counts.Comments
		}
		sort.Slice(lost, func(i, j int) bool { return lost[i] < lost[j] })
		rep.Folds = append(rep.Folds, FoldRecord{
			NewID: newID, Survivor: surv.ID, Losers: lost,
			Moved: moved, Dropped: dropped, Comments: comments,
		})
		rep.FoldedLosers += len(lost)
	}
	return losers, nil
}

func pickSurvivor(rows []galgamePick, newID int64) galgamePick {
	best := rows[0]
	for _, row := range rows[1:] {
		if survivorBetter(row, best, newID) {
			best = row
		}
	}
	return best
}

func survivorBetter(a, b galgamePick, newID int64) bool {
	aEq, bEq := a.ID == newID, b.ID == newID
	if aEq != bEq {
		return aEq
	}
	if a.Published != b.Published {
		return a.Published
	}
	if a.ResourceCount != b.ResourceCount {
		return a.ResourceCount > b.ResourceCount
	}
	if !a.Created.Equal(b.Created) {
		return a.Created.Before(b.Created)
	}
	return a.ID < b.ID
}

func disableTriggers(tx *gorm.DB, rep *Report) error {
	for _, table := range userTriggerTables {
		if err := tx.Exec("ALTER TABLE " + table + " DISABLE TRIGGER USER").Error; err != nil {
			return fmt.Errorf("disable triggers on %s: %w", table, err)
		}
	}
	rep.TriggersDisabled = append([]string{}, userTriggerTables...)
	return nil
}

func enableTriggers(tx *gorm.DB, rep *Report) error {
	for _, table := range userTriggerTables {
		if err := tx.Exec("ALTER TABLE " + table + " ENABLE TRIGGER USER").Error; err != nil {
			return fmt.Errorf("enable triggers on %s: %w", table, err)
		}
	}
	rep.TriggersEnabled = append([]string{}, userTriggerTables...)
	return nil
}

func intID(n int64) (int, error) {
	if n > int64(math.MaxInt) || n < 1 {
		return 0, fmt.Errorf("id %d does not fit a positive int", n)
	}
	return int(n), nil
}

func exec(tx *gorm.DB, q string, args ...any) (int64, error) {
	res := tx.Exec(q, args...)
	return res.RowsAffected, res.Error
}

func count(tx *gorm.DB, q string, args ...any) (int64, error) {
	var n int64
	if err := tx.Raw(q, args...).Scan(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

func sumInt(tx *gorm.DB, q string, args ...any) (int64, error) {
	var n int64
	if err := tx.Raw(q, args...).Scan(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}
