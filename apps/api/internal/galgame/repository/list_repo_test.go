package repository

import (
	"testing"

	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/testdb"

	"gorm.io/gorm"
)

// NULL means the content-limit sync has not reached the row yet, and it MUST
// stay listed: the column ships empty, so a filter that reads NULL as "not sfw"
// empties every list between the migration and the first sync run — and keeps
// the rows catalog has no work for hidden forever. Catalog's own gate at
// hydrate time is what actually hides a work; this predicate only decides which
// ids get that far, so erring open costs a card, erring closed costs the page.
func TestListIDsSFWFilter(t *testing.T) {
	db := testdb.Open(t)

	const base = 2_000_100_000
	unsynced, safe, adult, unlisted := base, base+1, base+2, base+3
	all := []int{unsynced, safe, adult, unlisted}

	cleanup := func() {
		db.Exec("DELETE FROM galgame_resource WHERE work_id = ANY(?::int[])", intArrayLit(all))
		db.Exec("DELETE FROM galgame WHERE id = ANY(?::int[])", intArrayLit(all))
	}
	cleanup()
	defer cleanup()

	seed(t, db, unsynced, nil)
	seed(t, db, safe, ptr("sfw"))
	seed(t, db, adult, ptr("nsfw"))
	seed(t, db, unlisted, ptr("sfw"))
	db.Exec("UPDATE galgame SET catalog_rendered = false WHERE id = ?", unlisted)

	repo := NewGalgameListRepository(db)
	for name, tc := range map[string]struct {
		filter model.GalgameListFilter
		want   []int
	}{
		"sfw reader keeps unsynced and sfw": {
			model.GalgameListFilter{SFWOnly: true},
			[]int{unsynced, safe},
		},
		"nsfw reader keeps every rendered row": {
			model.GalgameListFilter{},
			[]int{unsynced, safe, adult},
		},
		"the resource-filter lane gates too": {
			model.GalgameListFilter{SFWOnly: true, Type: "game"},
			[]int{unsynced, safe},
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := tc.filter
			f.RestrictIDs, f.Page, f.Limit, f.SortOrder = all, 1, 10, "desc"
			ids, total, err := repo.ListIDs(f)
			if err != nil {
				t.Fatalf("ListIDs: %v", err)
			}
			if total != int64(len(tc.want)) {
				t.Errorf("total = %d, want %d — the pager counts what the reader can reach", total, len(tc.want))
			}
			if !sameSet(ids, tc.want) {
				t.Errorf("ids = %v, want %v", ids, tc.want)
			}
		})
	}
}

func seed(t *testing.T, db *gorm.DB, id int, contentLimit *string) {
	t.Helper()
	if err := db.Create(&model.GalgameLocal{
		ID: id, Published: true, ContentLimit: contentLimit,
	}).Error; err != nil {
		t.Fatalf("seed galgame %d: %v", id, err)
	}
	if err := db.Create(&model.GalgameResource{
		WorkID: id, UserID: 1, Platform: "windows", Language: "ja", Type: "game",
	}).Error; err != nil {
		t.Fatalf("seed resource for %d: %v", id, err)
	}
}

func ptr(s string) *string { return &s }

func sameSet(got, want []int) bool {
	if len(got) != len(want) {
		return false
	}
	seen := make(map[int]bool, len(got))
	for _, id := range got {
		seen[id] = true
	}
	for _, id := range want {
		if !seen[id] {
			return false
		}
	}
	return true
}

// The catalog membership behind an entity page is mostly works the forum has no
// row for, so ranking it by a forum column has to answer two questions at once:
// order the rows that can be ranked, and keep the rest where catalog put them.
func TestOrderRestrictIDsKeepsUnknownMembersLast(t *testing.T) {
	db := testdb.Open(t)

	const base = 2_000_300_000
	quiet, busy, middling := base, base+1, base+2
	absent, absentToo := base+3, base+4
	local := []int{quiet, busy, middling}
	all := []int{absent, quiet, busy, absentToo, middling}

	cleanup := func() {
		db.Exec("DELETE FROM galgame_resource WHERE work_id = ANY(?::int[])", intArrayLit(local))
		db.Exec("DELETE FROM galgame WHERE id = ANY(?::int[])", intArrayLit(local))
	}
	cleanup()
	defer cleanup()

	for id, views := range map[int]int{quiet: 1, busy: 900, middling: 50} {
		seed(t, db, id, nil)
		if err := db.Model(&model.GalgameLocal{}).Where("id = ?", id).
			Update("view", views).Error; err != nil {
			t.Fatalf("set view on %d: %v", id, err)
		}
	}

	repo := NewGalgameListRepository(db)
	got := repo.OrderRestrictIDs(all, model.GalgameListFilter{
		SortField: "view", SortOrder: "desc",
	})
	want := []int{busy, middling, quiet, absent, absentToo}
	if len(got) != len(want) {
		t.Fatalf("ordered %d ids, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v — ranked rows first, then catalog's own order", got, want)
		}
	}

	// Ascending must not promote the members with no row: their view count is
	// unknown, not zero, and treating it as zero buries every ranked entry.
	got = repo.OrderRestrictIDs(all, model.GalgameListFilter{
		SortField: "view", SortOrder: "asc",
	})
	want = []int{quiet, middling, busy, absent, absentToo}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("asc order = %v, want %v", got, want)
		}
	}

	// A field whose SQL does not run falls back to catalog's order, which puts
	// `absent` first — so every chip is covered by asserting only the tail.
	for _, field := range []string{
		"time", "created", "view", "view_1d", "view_7d", "view_30d", "release_date", "rating",
	} {
		got = repo.OrderRestrictIDs(all, model.GalgameListFilter{
			SortField: field, SortOrder: "desc",
		})
		if len(got) != len(all) {
			t.Errorf("%s: ordered %d ids, want %d", field, len(got), len(all))
			continue
		}
		if got[3] != absent || got[4] != absentToo {
			t.Errorf("%s: order = %v, want the two members with no forum row last", field, got)
		}
	}
}

// The bounds are dates and the column is a timestamptz, so which rows a year
// contains is decided by the session time zone: production pins
// TimeZone=Asia/Shanghai in the DSN (internal/infrastructure/database/postgres.go),
// and every timestamp below straddles a boundary that moves if that changes.
// The upper bound is the other half — it arrives as the inclusive 2025-12-31 and
// only `< date + 1 day` keeps a row stamped late on New Year's Eve.
func TestListIDsCollectedFilter(t *testing.T) {
	db := testdb.Open(t)
	if err := db.Exec("SET TIME ZONE 'Asia/Shanghai'").Error; err != nil {
		t.Fatalf("pin the session zone: %v", err)
	}

	const base = 2_000_200_000
	before, newYear, midYear, lastDay, after := base, base+1, base+2, base+3, base+4
	all := []int{before, newYear, midYear, lastDay, after}

	cleanup := func() {
		db.Exec("DELETE FROM galgame_resource WHERE work_id = ANY(?::int[])", intArrayLit(all))
		db.Exec("DELETE FROM galgame WHERE id = ANY(?::int[])", intArrayLit(all))
	}
	cleanup()
	defer cleanup()

	for id, stamp := range map[int]string{
		before:  "2024-12-31 23:30:00+08",
		newYear: "2025-01-01 00:30:00+08",
		midYear: "2025-07-15 12:00:00+08",
		lastDay: "2025-12-31 23:30:00+08",
		after:   "2026-01-01 00:30:00+08",
	} {
		seed(t, db, id, nil)
		if err := db.Exec("UPDATE galgame SET created = ?::timestamptz WHERE id = ?", stamp, id).Error; err != nil {
			t.Fatalf("stamp %d: %v", id, err)
		}
	}

	repo := NewGalgameListRepository(db)
	for name, tc := range map[string]struct {
		filter model.GalgameListFilter
		want   []int
	}{
		"a year keeps both of its edges and neither neighbour": {
			model.GalgameListFilter{CollectedFrom: "2025-01-01", CollectedTo: "2025-12-31"},
			[]int{newYear, midYear, lastDay},
		},
		"a month set reaches across years": {
			model.GalgameListFilter{CollectedMonths: []int{1}},
			[]int{newYear, after},
		},
		"the resource-filter lane filters too": {
			model.GalgameListFilter{
				CollectedFrom: "2025-01-01", CollectedTo: "2025-12-31", Type: "game",
			},
			[]int{newYear, midYear, lastDay},
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := tc.filter
			f.RestrictIDs, f.Page, f.Limit, f.SortOrder = all, 1, 10, "desc"
			ids, total, err := repo.ListIDs(f)
			if err != nil {
				t.Fatalf("ListIDs: %v", err)
			}
			if total != int64(len(tc.want)) {
				t.Errorf("total = %d, want %d", total, len(tc.want))
			}
			if !sameSet(ids, tc.want) {
				t.Errorf("ids = %v, want %v", ids, tc.want)
			}
		})
	}
}

// The dropdown is built from this, so a month it offers a SFW reader must hold
// something that reader can open — otherwise the option looks broken.
func TestCollectedCalendarHonoursTheReadersGate(t *testing.T) {
	db := testdb.Open(t)
	if err := db.Exec("SET TIME ZONE 'Asia/Shanghai'").Error; err != nil {
		t.Fatalf("pin the session zone: %v", err)
	}

	const id = 2_000_200_100
	cleanup := func() {
		db.Exec("DELETE FROM galgame_resource WHERE work_id = ?", id)
		db.Exec("DELETE FROM galgame WHERE id = ?", id)
	}
	cleanup()
	defer cleanup()

	// 1999-03 is a month no real row occupies, so the assertion reads the seed
	// rather than whatever else the database happens to hold.
	seed(t, db, id, ptr("nsfw"))
	if err := db.Exec("UPDATE galgame SET created = '1999-03-15 12:00:00+08'::timestamptz WHERE id = ?", id).Error; err != nil {
		t.Fatalf("stamp the seed: %v", err)
	}

	repo := NewGalgameListRepository(db)
	holds := func(rows []CollectedMonth) bool {
		for _, r := range rows {
			if r.Year == 1999 && r.Month == 3 {
				return true
			}
		}
		return false
	}
	adultRows, err := repo.ListCollectedCalendar(false)
	if err != nil {
		t.Fatalf("ListCollectedCalendar(false): %v", err)
	}
	if !holds(adultRows) {
		t.Error("an adult reader was not offered the only month with an adult entry")
	}
	sfwRows, err := repo.ListCollectedCalendar(true)
	if err != nil {
		t.Fatalf("ListCollectedCalendar(true): %v", err)
	}
	if holds(sfwRows) {
		t.Error("a SFW reader was offered a month whose only entry the list will hide")
	}

	db.Exec("UPDATE galgame SET catalog_rendered = false WHERE id = ?", id)
	if adultRows, err = repo.ListCollectedCalendar(false); err != nil {
		t.Fatalf("ListCollectedCalendar(false): %v", err)
	}
	if holds(adultRows) {
		t.Error("a month whose only entry catalog will not render was still offered")
	}
}
