package renumber

import (
	"fmt"

	"gorm.io/gorm"
)

func hopAll(tx *gorm.DB, rep *Report) error {
	if err := hopPlain(tx, "galgame", "id", "", rep); err != nil {
		return err
	}
	for _, c := range []struct{ table, col, extra string }{
		{"galgame_activity", "galgame_id", ""},
		{"galgame_quiz", "galgame_id", "t.galgame_id IS NOT NULL"},
		{"galgame_comment_community_map", "galgame_id", ""},
		{"feed_activity", "galgame_id", ""},
		{"feed_activity", "source_id", "t.type = 'GALGAME_CREATION'"},
		{"galgame_merge_discarded", "old_gid", ""},
		{"galgame_merge_discarded", "new_gid", ""},
	} {
		if err := hopPlain(tx, c.table, c.col, c.extra, rep); err != nil {
			return err
		}
	}
	if err := checkDiscardedSums(tx, rep); err != nil {
		return err
	}
	if err := hopViewDaily(tx, rep); err != nil {
		return err
	}
	if err := hopCollectionItem(tx, rep); err != nil {
		return err
	}
	if err := hopQuizGalgame(tx, rep); err != nil {
		return err
	}
	return hopContributor(tx, rep)
}

func hopPlain(tx *gorm.DB, table, col, extra string, rep *Report) error {
	where := "t." + col + " IN (SELECT old_id FROM galgame_renumber_2026 WHERE old_id <> new_id)"
	if extra != "" {
		where += " AND " + extra
	}
	n, err := exec(tx, fmt.Sprintf("UPDATE %s t SET %s = t.%s + ? WHERE %s", table, col, col, where), Park)
	if err != nil {
		return fmt.Errorf("hop1 %s.%s: %w", table, col, err)
	}
	land := fmt.Sprintf(
		"UPDATE %s t SET %s = m.new_id FROM galgame_renumber_2026 m WHERE t.%s = m.old_id + ? AND m.old_id <> m.new_id",
		table, col, col)
	if extra != "" {
		land += " AND " + extra
	}
	if _, err := exec(tx, land, Park); err != nil {
		return fmt.Errorf("hop2 %s.%s: %w", table, col, err)
	}
	st := rep.table(statKey(table, col))
	st.Changed += n
	return nil
}

func hopViewDaily(tx *gorm.DB, rep *Report) error {
	st := rep.table("galgame_view_daily")
	n, err := exec(tx, `
		UPDATE galgame_view_daily t SET entity_id = t.entity_id + ?
		WHERE t.entity_id IN (SELECT old_id FROM galgame_renumber_2026 WHERE old_id <> new_id)`, Park)
	if err != nil {
		return fmt.Errorf("hop1 galgame_view_daily: %w", err)
	}
	st.Changed += n

	intoExisting, err := sumInt(tx, `
		SELECT COALESCE(SUM(m.new_id), 0)::bigint FROM galgame_view_daily t
		JOIN galgame_renumber_2026 m ON t.entity_id = m.old_id + ? AND m.old_id <> m.new_id
		JOIN galgame_view_daily x ON x.entity_id = m.new_id AND x.day = t.day`, Park)
	if err != nil {
		return fmt.Errorf("galgame_view_daily existing collisions: %w", err)
	}
	intoExistingN, err := count(tx, `
		SELECT COUNT(*) FROM galgame_view_daily t
		JOIN galgame_renumber_2026 m ON t.entity_id = m.old_id + ? AND m.old_id <> m.new_id
		JOIN galgame_view_daily x ON x.entity_id = m.new_id AND x.day = t.day`, Park)
	if err != nil {
		return err
	}
	extraSum, err := sumInt(tx, `
		SELECT COALESCE(SUM(s.new_id * (s.cnt - 1)), 0)::bigint FROM (
			SELECT m.new_id, t.day, COUNT(*) AS cnt
			FROM galgame_view_daily t
			JOIN galgame_renumber_2026 m ON t.entity_id = m.old_id + ? AND m.old_id <> m.new_id
			WHERE NOT EXISTS (
				SELECT 1 FROM galgame_view_daily x WHERE x.entity_id = m.new_id AND x.day = t.day
			)
			GROUP BY m.new_id, t.day
			HAVING COUNT(*) > 1
		) s`, Park)
	if err != nil {
		return fmt.Errorf("galgame_view_daily parked collisions: %w", err)
	}
	extraN, err := count(tx, `
		SELECT COALESCE(SUM(s.cnt - 1), 0)::bigint FROM (
			SELECT m.new_id, t.day, COUNT(*) AS cnt
			FROM galgame_view_daily t
			JOIN galgame_renumber_2026 m ON t.entity_id = m.old_id + ? AND m.old_id <> m.new_id
			WHERE NOT EXISTS (
				SELECT 1 FROM galgame_view_daily x WHERE x.entity_id = m.new_id AND x.day = t.day
			)
			GROUP BY m.new_id, t.day
			HAVING COUNT(*) > 1
		) s`, Park)
	if err != nil {
		return err
	}

	if _, err := exec(tx, `
		INSERT INTO galgame_view_daily (entity_id, day, count)
		SELECT m.new_id, t.day, SUM(t.count)
		FROM galgame_view_daily t
		JOIN galgame_renumber_2026 m ON t.entity_id = m.old_id + ? AND m.old_id <> m.new_id
		GROUP BY m.new_id, t.day
		ON CONFLICT (entity_id, day) DO UPDATE SET count = galgame_view_daily.count + EXCLUDED.count`, Park); err != nil {
		return fmt.Errorf("land galgame_view_daily: %w", err)
	}
	if _, err := exec(tx, `
		DELETE FROM galgame_view_daily t
		USING galgame_renumber_2026 m
		WHERE t.entity_id = m.old_id + ? AND m.old_id <> m.new_id`, Park); err != nil {
		return fmt.Errorf("delete parked galgame_view_daily: %w", err)
	}
	st.Merged += intoExistingN + extraN
	st.DroppedSum += intoExisting + extraSum
	return nil
}

func hopCollectionItem(tx *gorm.DB, rep *Report) error {
	st := rep.table("galgame_collection_item")
	n, err := exec(tx, `
		UPDATE galgame_collection_item t SET galgame_id = t.galgame_id + ?
		WHERE t.galgame_id IN (SELECT old_id FROM galgame_renumber_2026 WHERE old_id <> new_id)`, Park)
	if err != nil {
		return fmt.Errorf("hop1 galgame_collection_item: %w", err)
	}
	st.Changed += n
	if _, err := exec(tx, `
		WITH ordered AS (
			SELECT t.id, m.new_id,
				ROW_NUMBER() OVER (PARTITION BY t.collection_id, m.new_id ORDER BY t.created ASC, t.id ASC) AS rn
			FROM galgame_collection_item t
			JOIN galgame_renumber_2026 m ON t.galgame_id = m.old_id + ? AND m.old_id <> m.new_id
		)
		UPDATE galgame_collection_item t SET galgame_id = ordered.new_id
		FROM ordered
		WHERE t.id = ordered.id AND ordered.rn = 1
		  AND NOT EXISTS (
			SELECT 1 FROM galgame_collection_item x
			WHERE x.collection_id = t.collection_id AND x.galgame_id = ordered.new_id
		  )`, Park); err != nil {
		return fmt.Errorf("land galgame_collection_item: %w", err)
	}
	return archiveParked(tx, "galgame_collection_item", "galgame_id", st)
}

func hopQuizGalgame(tx *gorm.DB, rep *Report) error {
	st := rep.table("galgame_quiz_galgame")
	n, err := exec(tx, `
		UPDATE galgame_quiz_galgame t SET galgame_id = t.galgame_id + ?
		WHERE t.galgame_id IN (SELECT old_id FROM galgame_renumber_2026 WHERE old_id <> new_id)`, Park)
	if err != nil {
		return fmt.Errorf("hop1 galgame_quiz_galgame: %w", err)
	}
	st.Changed += n
	if _, err := exec(tx, `
		WITH ordered AS (
			SELECT t.quiz_id, t.galgame_id, m.new_id,
				ROW_NUMBER() OVER (PARTITION BY t.quiz_id, m.new_id ORDER BY t.galgame_id) AS rn
			FROM galgame_quiz_galgame t
			JOIN galgame_renumber_2026 m ON t.galgame_id = m.old_id + ? AND m.old_id <> m.new_id
		)
		UPDATE galgame_quiz_galgame t SET galgame_id = ordered.new_id
		FROM ordered
		WHERE t.quiz_id = ordered.quiz_id AND t.galgame_id = ordered.galgame_id
		  AND ordered.rn = 1
		  AND NOT EXISTS (
			SELECT 1 FROM galgame_quiz_galgame x
			WHERE x.quiz_id = t.quiz_id AND x.galgame_id = ordered.new_id
		  )`, Park); err != nil {
		return fmt.Errorf("land galgame_quiz_galgame: %w", err)
	}
	return archiveParked(tx, "galgame_quiz_galgame", "galgame_id", st)
}

func archiveParked(tx *gorm.DB, table, col string, st *TableStat) error {
	dropped, err := sumInt(tx, fmt.Sprintf(`
		SELECT COALESCE(SUM(m.new_id), 0)::bigint FROM %s t
		JOIN galgame_renumber_2026 m ON t.%s = m.old_id + ? AND m.old_id <> m.new_id`, table, col), Park)
	if err != nil {
		return fmt.Errorf("dropped sum %s: %w", table, err)
	}
	archived, err := exec(tx, fmt.Sprintf(`
		INSERT INTO galgame_merge_discarded (old_gid, new_gid, table_name, row)
		SELECT m.old_id, m.new_id, ?, to_jsonb(t)
		FROM %s t
		JOIN galgame_renumber_2026 m ON t.%s = m.old_id + ? AND m.old_id <> m.new_id`, table, col),
		table, Park)
	if err != nil {
		return fmt.Errorf("archive %s: %w", table, err)
	}
	if _, err := exec(tx, fmt.Sprintf(`
		DELETE FROM %s t USING galgame_renumber_2026 m
		WHERE t.%s = m.old_id + ? AND m.old_id <> m.new_id`, table, col), Park); err != nil {
		return fmt.Errorf("delete parked %s: %w", table, err)
	}
	st.Archived += archived
	st.DroppedSum += dropped
	return nil
}

func hopContributor(tx *gorm.DB, rep *Report) error {
	st := rep.table("galgame_contributor")
	n, err := exec(tx, `
		UPDATE galgame_contributor t SET galgame_id = t.galgame_id + ?
		WHERE t.galgame_id IN (SELECT old_id FROM galgame_renumber_2026 WHERE old_id <> new_id)`, Park)
	if err != nil {
		return fmt.Errorf("hop1 galgame_contributor: %w", err)
	}
	st.Changed += n

	intoSum, err := sumInt(tx, `
		SELECT COALESCE(SUM(m.new_id), 0)::bigint FROM galgame_contributor t
		JOIN galgame_renumber_2026 m ON t.galgame_id = m.old_id + ? AND m.old_id <> m.new_id
		JOIN galgame_contributor x ON x.galgame_id = m.new_id AND x.user_id = t.user_id`, Park)
	if err != nil {
		return fmt.Errorf("galgame_contributor existing collisions: %w", err)
	}
	extraSum, err := sumInt(tx, `
		SELECT COALESCE(SUM(s.new_id * (s.cnt - 1)), 0)::bigint FROM (
			SELECT m.new_id, t.user_id, COUNT(*) AS cnt
			FROM galgame_contributor t
			JOIN galgame_renumber_2026 m ON t.galgame_id = m.old_id + ? AND m.old_id <> m.new_id
			WHERE NOT EXISTS (
				SELECT 1 FROM galgame_contributor x WHERE x.galgame_id = m.new_id AND x.user_id = t.user_id
			)
			GROUP BY m.new_id, t.user_id
			HAVING COUNT(*) > 1
		) s`, Park)
	if err != nil {
		return fmt.Errorf("galgame_contributor parked collisions: %w", err)
	}

	if _, err := exec(tx, `
		UPDATE galgame_contributor t SET
			revision_count = t.revision_count + x.sum_rev,
			first_at = LEAST(t.first_at, x.min_first),
			last_at = GREATEST(t.last_at, x.max_last)
		FROM (
			SELECT m.new_id, s.user_id,
				SUM(s.revision_count) AS sum_rev,
				MIN(s.first_at) AS min_first,
				MAX(s.last_at) AS max_last
			FROM galgame_contributor s
			JOIN galgame_renumber_2026 m ON s.galgame_id = m.old_id + ? AND m.old_id <> m.new_id
			GROUP BY m.new_id, s.user_id
		) x
		WHERE t.galgame_id = x.new_id AND t.user_id = x.user_id`, Park); err != nil {
		return fmt.Errorf("merge galgame_contributor into existing: %w", err)
	}
	mergedExisting, err := exec(tx, `
		DELETE FROM galgame_contributor s
		USING galgame_renumber_2026 m, galgame_contributor t
		WHERE s.galgame_id = m.old_id + ? AND m.old_id <> m.new_id
		  AND t.galgame_id = m.new_id AND t.user_id = s.user_id`, Park)
	if err != nil {
		return fmt.Errorf("delete merged galgame_contributor: %w", err)
	}

	if _, err := exec(tx, `
		UPDATE galgame_contributor t SET
			revision_count = x.sum_rev,
			first_at = x.min_first,
			last_at = x.max_last
		FROM (
			SELECT s.id,
				SUM(s.revision_count) OVER (PARTITION BY m.new_id, s.user_id) AS sum_rev,
				MIN(s.first_at) OVER (PARTITION BY m.new_id, s.user_id) AS min_first,
				MAX(s.last_at) OVER (PARTITION BY m.new_id, s.user_id) AS max_last,
				ROW_NUMBER() OVER (PARTITION BY m.new_id, s.user_id ORDER BY s.id) AS rn
			FROM galgame_contributor s
			JOIN galgame_renumber_2026 m ON s.galgame_id = m.old_id + ? AND m.old_id <> m.new_id
		) x
		WHERE t.id = x.id AND x.rn = 1`, Park); err != nil {
		return fmt.Errorf("collapse galgame_contributor keepers: %w", err)
	}
	mergedDup, err := exec(tx, `
		DELETE FROM galgame_contributor s
		USING (
			SELECT s.id, ROW_NUMBER() OVER (PARTITION BY m.new_id, s.user_id ORDER BY s.id) AS rn
			FROM galgame_contributor s
			JOIN galgame_renumber_2026 m ON s.galgame_id = m.old_id + ? AND m.old_id <> m.new_id
		) x
		WHERE s.id = x.id AND x.rn > 1`, Park)
	if err != nil {
		return fmt.Errorf("delete duplicate galgame_contributor: %w", err)
	}
	if _, err := exec(tx, `
		UPDATE galgame_contributor t SET galgame_id = m.new_id
		FROM galgame_renumber_2026 m
		WHERE t.galgame_id = m.old_id + ? AND m.old_id <> m.new_id`, Park); err != nil {
		return fmt.Errorf("land galgame_contributor: %w", err)
	}
	st.Merged += mergedExisting + mergedDup
	st.DroppedSum += intoSum + extraSum
	return nil
}

func statKey(table, col string) string {
	if col == "id" || col == "galgame_id" {
		return table
	}
	return table + "." + col
}

func checkDiscardedSums(tx *gorm.DB, rep *Report) error {
	for _, c := range sumCols() {
		if c.table != "galgame_merge_discarded" {
			continue
		}
		got, err := rawSum(tx, c)
		if err != nil {
			return fmt.Errorf("sum after %s: %w", c.key, err)
		}
		st := rep.table(c.key)
		st.SumAfter = got
		if got != st.SumBefore {
			return fmt.Errorf("invariant %s: sum before %d after %d", c.key, st.SumBefore, got)
		}
	}
	return nil
}
