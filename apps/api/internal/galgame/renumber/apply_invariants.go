package renumber

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type sumCol struct {
	key, table, col, extra string
	kind                   byte
}

func sumCols() []sumCol {
	return []sumCol{
		{"galgame", "galgame", "id", "", 'A'},
		{"galgame_activity", "galgame_activity", "galgame_id", "", 'B'},
		{"galgame_quiz", "galgame_quiz", "galgame_id", "t.galgame_id IS NOT NULL", 'B'},
		{"galgame_comment_community_map", "galgame_comment_community_map", "galgame_id", "", 'B'},
		{"feed_activity", "feed_activity", "galgame_id", "", 'B'},
		{"feed_activity.source_id", "feed_activity", "source_id", "t.type = 'GALGAME_CREATION'", 'B'},
		{"galgame_merge_discarded.old_gid", "galgame_merge_discarded", "old_gid", "", 'B'},
		{"galgame_merge_discarded.new_gid", "galgame_merge_discarded", "new_gid", "", 'B'},
		{"galgame_view_daily", "galgame_view_daily", "entity_id", "", 'C'},
		{"galgame_collection_item", "galgame_collection_item", "galgame_id", "", 'C'},
		{"galgame_quiz_galgame", "galgame_quiz_galgame", "galgame_id", "", 'C'},
		{"galgame_contributor", "galgame_contributor", "galgame_id", "", 'C'},
	}
}

func snapshotSums(tx *gorm.DB, rep *Report) error {
	for _, c := range sumCols() {
		n, err := mappedSum(tx, c)
		if err != nil {
			return fmt.Errorf("sum before %s: %w", c.key, err)
		}
		rep.table(c.key).SumBefore = n
	}
	return nil
}

func mappedSum(tx *gorm.DB, c sumCol) (int64, error) {
	q := fmt.Sprintf(`
		SELECT COALESCE(SUM(COALESCE(m.new_id, t.%s)), 0)::bigint
		FROM %s t
		LEFT JOIN galgame_renumber_2026 m ON m.old_id = t.%s`, c.col, c.table, c.col)
	if c.extra != "" {
		q += " WHERE " + c.extra
	}
	return sumInt(tx, q)
}

func rawSum(tx *gorm.DB, c sumCol) (int64, error) {
	q := fmt.Sprintf("SELECT COALESCE(SUM(t.%s), 0)::bigint FROM %s t", c.col, c.table)
	if c.extra != "" {
		q += " WHERE " + c.extra
	}
	return sumInt(tx, q)
}

func assertInvariants(tx *gorm.DB, m *Map, preIDs []int64, losers map[int64]bool, rep *Report) error {
	after, err := count(tx, "SELECT COUNT(*) FROM galgame")
	if err != nil {
		return err
	}
	rep.GalgameAfter = after
	if after != rep.GalgameBefore-int64(len(losers)) {
		return fmt.Errorf("galgame count: got %d, want %d (before %d minus %d losers)",
			after, rep.GalgameBefore-int64(len(losers)), rep.GalgameBefore, len(losers))
	}

	expected := map[int64]int64{}
	for _, id := range preIDs {
		if losers[id] {
			continue
		}
		newID, ok := m.Lookup(id)
		if !ok {
			newID = id
		}
		if prev, dup := expected[newID]; dup {
			return fmt.Errorf("two pre-run ids (%d and %d) map to remaining id %d", prev, id, newID)
		}
		expected[newID] = id
	}
	var got []int64
	if err := tx.Raw("SELECT id FROM galgame").Scan(&got).Error; err != nil {
		return err
	}
	if int64(len(got)) != after {
		return fmt.Errorf("galgame id list %d, count %d", len(got), after)
	}
	seen := map[int64]bool{}
	for _, id := range got {
		old, ok := expected[id]
		if !ok {
			return fmt.Errorf("remaining galgame.id %d is not Lookup of any surviving pre-run id", id)
		}
		if seen[id] {
			return fmt.Errorf("duplicate remaining galgame.id %d (pre-run %d)", id, old)
		}
		seen[id] = true
	}
	if len(seen) != len(expected) {
		return fmt.Errorf("remaining galgame ids %d, expected survivors %d", len(seen), len(expected))
	}

	var occupied []string
	for _, c := range parkCols {
		n, err := count(tx, parkCountSQL(c), Park)
		if err != nil {
			return fmt.Errorf("park leftover %s.%s: %w", c.table, c.col, err)
		}
		if n > 0 {
			occupied = append(occupied, fmt.Sprintf("%s.%s=%d", c.table, c.col, n))
		}
	}
	if len(occupied) > 0 {
		rep.ParkOccupied = occupied
		return fmt.Errorf("park values remain: %s", strings.Join(occupied, ", "))
	}

	for _, c := range sumCols() {
		got, err := rawSum(tx, c)
		if err != nil {
			return fmt.Errorf("sum after %s: %w", c.key, err)
		}
		st := rep.table(c.key)
		if c.table == "galgame_merge_discarded" {
			continue
		}
		st.SumAfter = got
		switch c.kind {
		case 'A', 'B':
			if got != st.SumBefore {
				return fmt.Errorf("invariant %s: sum before %d after %d", c.key, st.SumBefore, got)
			}
		case 'C':
			if got != st.SumBefore-st.DroppedSum {
				return fmt.Errorf("invariant %s: sum before %d after %d dropped_sum %d (want after = before - dropped)",
					c.key, st.SumBefore, got, st.DroppedSum)
			}
		}
	}

	for _, table := range []string{"galgame_like", "galgame_favorite", "galgame_rating", "galgame_resource"} {
		n, err := count(tx, fmt.Sprintf(`
			SELECT COUNT(*) FROM %s t
			LEFT JOIN galgame g ON g.id = t.galgame_id
			WHERE g.id IS NULL`, table))
		if err != nil {
			return fmt.Errorf("fk check %s: %w", table, err)
		}
		if n > 0 {
			return fmt.Errorf("%s has %d row(s) pointing at a missing galgame.id", table, n)
		}
	}

	for src, st := range rep.Links {
		if st.Rewritten != st.Changing {
			return fmt.Errorf("link %s: rewritten %d, changing %d", src, st.Rewritten, st.Changing)
		}
	}

	quoted := make([]string, len(userTriggerTables))
	for i, name := range userTriggerTables {
		quoted[i] = "'" + name + "'"
	}
	var disabled []string
	if err := tx.Raw(`
		SELECT c.relname || '.' || t.tgname
		FROM pg_trigger t
		JOIN pg_class c ON c.oid = t.tgrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public'
		  AND c.relname IN (` + strings.Join(quoted, ",") + `)
		  AND NOT t.tgisinternal
		  AND t.tgenabled = 'D'
		ORDER BY 1`).Scan(&disabled).Error; err != nil {
		return fmt.Errorf("trigger state: %w", err)
	}
	if len(disabled) > 0 {
		return fmt.Errorf("user triggers still disabled: %s", strings.Join(disabled, ", "))
	}
	return nil
}
