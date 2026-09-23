package renumber

import (
	"fmt"

	"gorm.io/gorm"
)

type textCol struct {
	table, col string
	authored   bool
}

func rewriteAllLinks(tx *gorm.DB, m *Map, opts Options, rep *Report) error {
	if err := rewritePathLinks(tx, "message", "link", rep); err != nil {
		return err
	}
	if err := rewritePathLinks(tx, "feed_activity", "link", rep); err != nil {
		return err
	}
	cols := []textCol{
		{"topic", "content", true},
		{"topic_reply", "content", true},
		{"topic_comment", "content", true},
		{"todo", "content", true},
		{"doc_article", "content_markdown", true},
		{"message", "content", false},
		{"feed_activity", "content", false},
		{"galgame_resource_link", "url", false},
	}
	if opts.IncludeDM {
		cols = append(cols,
			textCol{"chat_message", "content", true},
			textCol{"chat_room", "last_message_content", true},
		)
	}
	for _, c := range cols {
		if err := rewriteTextCol(tx, m, c, rep); err != nil {
			return err
		}
	}
	return nil
}

func rewritePathLinks(tx *gorm.DB, table, col string, rep *Report) error {
	src := table + "." + col
	st := rep.link(src)
	matching, err := count(tx, fmt.Sprintf(
		`SELECT COUNT(*) FROM %s WHERE %s ~ '^/galgame/[0-9]+($|[?#/])'`, table, col))
	if err != nil {
		return fmt.Errorf("count matching %s: %w", src, err)
	}
	st.Matching = matching
	changing, err := count(tx, fmt.Sprintf(`
		SELECT COUNT(*) FROM %s t
		JOIN galgame_renumber_2026 m
		  ON m.old_id = substring(t.%s from '^/galgame/([0-9]+)')::bigint
		 AND m.old_id <> m.new_id
		WHERE t.%s ~ '^/galgame/[0-9]+($|[?#/])'`, table, col, col))
	if err != nil {
		return fmt.Errorf("count changing %s: %w", src, err)
	}
	st.Changing = changing
	n, err := exec(tx, fmt.Sprintf(`
		UPDATE %s t SET %s = regexp_replace(t.%s, '^/galgame/[0-9]+', '/galgame/' || m.new_id::text)
		FROM galgame_renumber_2026 m
		WHERE t.%s ~ '^/galgame/[0-9]+($|[?#/])'
		  AND substring(t.%s from '^/galgame/([0-9]+)')::bigint = m.old_id
		  AND m.old_id <> m.new_id`, table, col, col, col, col))
	if err != nil {
		return fmt.Errorf("rewrite %s: %w", src, err)
	}
	st.Rewritten = n
	return nil
}

type textRow struct {
	ID   int64  `gorm:"column:id"`
	Text string `gorm:"column:text"`
}

func rewriteTextCol(tx *gorm.DB, m *Map, c textCol, rep *Report) error {
	src := c.table + "." + c.col
	st := rep.link(src)
	var rows []textRow
	if err := tx.Raw(fmt.Sprintf(
		`SELECT id, %s AS text FROM %s WHERE %s ~ 'galgame/[0-9]'`, c.col, c.table, c.col),
	).Scan(&rows).Error; err != nil {
		return fmt.Errorf("select %s: %w", src, err)
	}
	st.Matching = int64(len(rows))
	lookup := m.Lookup
	var rewritten int64
	for _, row := range rows {
		next, n := RewriteLinks(row.Text, lookup)
		if n == 0 || next == row.Text {
			continue
		}
		st.Changing++
		if _, err := exec(tx, fmt.Sprintf("UPDATE %s SET %s = ? WHERE id = ?", c.table, c.col), next, row.ID); err != nil {
			return fmt.Errorf("update %s id=%d: %w", src, row.ID, err)
		}
		rewritten++
		if c.authored {
			rep.Content = append(rep.Content, ContentChange{
				Table: c.table, ID: row.ID, Before: row.Text, After: next,
			})
		}
	}
	st.Rewritten = rewritten
	return nil
}
