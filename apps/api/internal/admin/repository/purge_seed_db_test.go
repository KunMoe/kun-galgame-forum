package repository

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

type seeder struct {
	t    *testing.T
	db   *gorm.DB
	seq  int
	rows []seededRow
	cols map[string][]seedColumn
}

type seededRow struct {
	table string
	pk    map[string]any
}

type seedColumn struct {
	Name     string
	DataType string
}

func newSeeder(t *testing.T, db *gorm.DB) *seeder {
	t.Helper()
	s := &seeder{t: t, db: db, cols: map[string][]seedColumn{}}
	t.Cleanup(s.cleanup)
	return s
}

func (s *seeder) required(table string) []seedColumn {
	if cols, ok := s.cols[table]; ok {
		return cols
	}
	var cols []seedColumn
	if err := s.db.Raw(`SELECT column_name AS name, data_type FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = ? AND is_nullable = 'NO' AND column_default IS NULL
		ORDER BY ordinal_position`, table).Scan(&cols).Error; err != nil {
		s.t.Fatal(err)
	}
	s.cols[table] = cols
	return cols
}

func (s *seeder) filler(c seedColumn) any {
	s.seq++
	switch c.DataType {
	case "integer", "bigint", "smallint", "numeric", "double precision", "real":
		return 0
	case "boolean":
		return false
	case "timestamp with time zone", "timestamp without time zone":
		return time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC)
	case "date":
		return "2026-05-01"
	case "jsonb", "json":
		return "{}"
	case "ARRAY":
		return "{}"
	default:
		return fmt.Sprintf("s%d", s.seq)
	}
}

func (s *seeder) insert(table string, vals map[string]any) map[string]any {
	s.t.Helper()
	cols := make([]string, 0, len(vals))
	args := make([]any, 0, len(vals))
	for k, v := range vals {
		cols = append(cols, k)
		args = append(args, v)
	}
	for _, c := range s.required(table) {
		if _, given := vals[c.Name]; given {
			continue
		}
		cols = append(cols, c.Name)
		args = append(args, s.filler(c))
	}
	quoted := make([]string, len(cols))
	marks := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = `"` + c + `"`
		marks[i] = "?"
	}
	var raw string
	q := fmt.Sprintf(`INSERT INTO %q AS x (%s) VALUES (%s) RETURNING to_jsonb(x)::text`,
		table, strings.Join(quoted, ", "), strings.Join(marks, ", "))
	if err := s.db.Raw(q, args...).Scan(&raw).Error; err != nil {
		s.t.Fatalf("seed %s: %v", table, err)
	}
	var row map[string]any
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&row); err != nil {
		s.t.Fatal(err)
	}
	pk := map[string]any{}
	for _, c := range primaryKey(s.t, s.db, table) {
		pk[c] = row[c]
	}
	s.rows = append(s.rows, seededRow{table: table, pk: pk})
	return row
}

func (s *seeder) run(q string, args ...any) {
	s.t.Helper()
	if err := s.db.Exec(q, args...).Error; err != nil {
		s.t.Fatalf("seed: %v\n%s", err, q)
	}
}

func (s *seeder) cleanup() {
	for i := len(s.rows) - 1; i >= 0; i-- {
		r := s.rows[i]
		where := make([]string, 0, len(r.pk))
		args := make([]any, 0, len(r.pk))
		for k, v := range r.pk {
			where = append(where, fmt.Sprintf("%q::text = ?", k))
			args = append(args, fmt.Sprint(v))
		}
		_ = s.db.Exec(fmt.Sprintf(`DELETE FROM %q WHERE %s`, r.table, strings.Join(where, " AND ")), args...).Error
	}
}

func primaryKey(t *testing.T, db *gorm.DB, table string) []string {
	t.Helper()
	var cols []string
	if err := db.Raw(`SELECT a.attname FROM pg_index i
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY (i.indkey)
		WHERE i.indrelid = format('public.%I', ?::text)::regclass AND i.indisprimary
		ORDER BY a.attnum`, table).Scan(&cols).Error; err != nil {
		t.Fatal(err)
	}
	return cols
}

func snapshot(t *testing.T, db *gorm.DB) map[string]map[string]bool {
	t.Helper()
	var tables []string
	if err := db.Raw(`SELECT c.relname FROM pg_class c
		WHERE c.relnamespace = 'public'::regnamespace AND c.relkind = 'r' AND NOT c.relispartition
		  AND c.relname NOT IN ('user_purge_archive', '_migrations') ORDER BY 1`).Scan(&tables).Error; err != nil {
		t.Fatal(err)
	}
	out := make(map[string]map[string]bool, len(tables))
	for _, tbl := range tables {
		var rows []string
		if err := db.Raw(fmt.Sprintf(`SELECT to_jsonb(x)::text FROM %q x`, tbl)).Scan(&rows).Error; err != nil {
			t.Fatal(err)
		}
		set := make(map[string]bool, len(rows))
		for _, r := range rows {
			set[r] = true
		}
		out[tbl] = set
	}
	return out
}

func diffSnapshots(before, after map[string]map[string]bool) (gone, added map[string][]string) {
	gone, added = map[string][]string{}, map[string][]string{}
	for tbl, rows := range before {
		for r := range rows {
			if !after[tbl][r] {
				gone[tbl] = append(gone[tbl], r)
			}
		}
	}
	for tbl, rows := range after {
		for r := range rows {
			if !before[tbl][r] {
				added[tbl] = append(added[tbl], r)
			}
		}
	}
	return gone, added
}
