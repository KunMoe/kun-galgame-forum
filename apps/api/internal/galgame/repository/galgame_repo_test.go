package repository

import (
	"slices"
	"testing"
	"time"

	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/testdb"
)

// galgame_resource is ON DELETE CASCADE, so an unguarded delete of a local row
// takes the resources with it and says nothing. A draft claim carrying a
// published resource is reachable — publishing a resource sets `published`
// without moving the claim state — and catalog will happily delete that draft.
func TestDeleteLocalDraftRefusesARowThatCarriesAResource(t *testing.T) {
	db := testdb.Open(t)
	repo := NewGalgameRepository(db)

	const base = 2_000_200_000
	bare, withResource := base, base+1
	all := []int{bare, withResource}

	cleanup := func() {
		db.Exec("DELETE FROM galgame_resource WHERE work_id = ANY(?::int[])", intArrayLit(all))
		db.Exec("DELETE FROM galgame WHERE id = ANY(?::int[])", intArrayLit(all))
	}
	cleanup()
	defer cleanup()

	for _, id := range all {
		if err := db.Create(&model.GalgameLocal{ID: id}).Error; err != nil {
			t.Fatalf("seed galgame %d: %v", id, err)
		}
	}
	if err := db.Create(&model.GalgameResource{
		WorkID: withResource, UserID: 1, Platform: "windows", Language: "ja", Type: "game",
	}).Error; err != nil {
		t.Fatalf("seed resource: %v", err)
	}

	if err := repo.DeleteLocalDraft(bare); err != nil {
		t.Fatalf("DeleteLocalDraft(bare): %v", err)
	}
	if err := repo.DeleteLocalDraft(withResource); err != nil {
		t.Fatalf("DeleteLocalDraft(withResource): %v", err)
	}

	var rows int64
	db.Table("galgame").Where("id = ?", bare).Count(&rows)
	if rows != 0 {
		t.Error("an empty draft row must be gone; nothing else will ever clean it up")
	}
	db.Table("galgame").Where("id = ?", withResource).Count(&rows)
	if rows != 1 {
		t.Error("a row carrying a resource must survive rather than cascade it away")
	}
	db.Table("galgame_resource").Where("work_id = ?", withResource).Count(&rows)
	if rows != 1 {
		t.Error("the resource was cascade-deleted")
	}
}

// The mirror has to be able to say "catalog has no date for this work" and have
// the row stop coming back. Before 092 the only marker was release_date itself,
// so a TBA work was indistinguishable from a row nobody had asked about, and a
// capped fill lane would spend every tick re-asking about the same ones.
func TestReleaseDateMirrorConfirmsRowsWithNoDate(t *testing.T) {
	db := testdb.Open(t)
	repo := NewGalgameRepository(db)

	const base = 2_000_300_000
	dated, tba, orphan, untouched := base, base+1, base+2, base+3
	all := []int{dated, tba, orphan, untouched}

	cleanup := func() { db.Exec("DELETE FROM galgame WHERE id = ANY(?::int[])", intArrayLit(all)) }
	cleanup()
	defer cleanup()

	for _, id := range all {
		if err := db.Create(&model.GalgameLocal{ID: id, ContentLimit: ptr("sfw")}).Error; err != nil {
			t.Fatalf("seed galgame %d: %v", id, err)
		}
	}

	if _, err := repo.SetReleaseDates(map[int]string{dated: "2026-08-27", tba: ""}); err != nil {
		t.Fatalf("SetReleaseDates: %v", err)
	}

	var rows []struct {
		ID       int
		Released *time.Time `gorm:"column:release_date"`
		SyncedAt *time.Time `gorm:"column:release_date_synced_at"`
	}
	db.Table("galgame").Select("id, release_date, release_date_synced_at").
		Where("id = ANY(?::int[])", intArrayLit(all)).Order("id").Scan(&rows)
	if len(rows) != len(all) {
		t.Fatalf("read back %d rows, want %d", len(rows), len(all))
	}
	if rows[0].Released == nil || rows[0].Released.Format(time.DateOnly) != "2026-08-27" {
		t.Errorf("dated row = %v, want 2026-08-27", rows[0].Released)
	}
	if rows[0].SyncedAt == nil {
		t.Error("a written date must mark the row confirmed")
	}
	if rows[1].Released != nil {
		t.Errorf("a TBA row must keep a NULL date, got %v", rows[1].Released)
	}
	if rows[1].SyncedAt == nil {
		t.Error("no date is still an answer: the row has to be marked confirmed or it never converges")
	}
	if rows[2].SyncedAt != nil || rows[3].SyncedAt != nil {
		t.Error("a row the write never named must stay unconfirmed")
	}
}

// The verify lane walks the table oldest-checked first, and a row it has just
// asked about goes to the back whatever catalog said, so a row catalog cannot
// answer does not head every window.
func TestMirrorVerifyOrderAndMarks(t *testing.T) {
	db := testdb.Open(t)
	repo := NewGalgameRepository(db)

	const base = 2_000_310_000
	fresh, old, older, never := base, base+1, base+2, base+3
	all := []int{fresh, old, older, never}
	cleanup := func() { db.Exec("DELETE FROM galgame WHERE id = ANY(?::int[])", intArrayLit(all)) }
	cleanup()
	defer cleanup()
	for _, id := range all {
		if err := db.Create(&model.GalgameLocal{ID: id}).Error; err != nil {
			t.Fatalf("seed galgame %d: %v", id, err)
		}
	}
	db.Exec("UPDATE galgame SET catalog_checked_at = now() WHERE id = ?", fresh)
	db.Exec("UPDATE galgame SET catalog_checked_at = now() - interval '1 hour' WHERE id = ?", old)
	db.Exec("UPDATE galgame SET catalog_checked_at = now() - interval '2 hours' WHERE id = ?", older)

	window := func() []int {
		var out []int
		for _, id := range repo.MirrorVerifyIDs(1_000_000) {
			if slices.Contains(all, id) {
				out = append(out, id)
			}
		}
		return out
	}
	if got := window(); !slices.Equal(got, []int{never, older, old, fresh}) {
		t.Fatalf("window = %v, want never-checked first, then oldest", got)
	}

	if err := repo.MarkCatalogChecked([]int{never, older}, false); err != nil {
		t.Fatalf("MarkCatalogChecked: %v", err)
	}
	if got := window(); !slices.Equal(got, []int{old, fresh, older, never}) {
		t.Fatalf("window after marking = %v, want the marked rows at the back", got)
	}
	rendered, err := repo.RenderedAmong(append(slices.Clone(all), base+9))
	if err != nil {
		t.Fatalf("RenderedAmong: %v", err)
	}
	if !slices.Equal(rendered, []int{fresh, old}) {
		t.Errorf("rendered = %v, want the two unmarked rows", rendered)
	}
	local, err := repo.LocalAmong([]int{never, base + 9})
	if err != nil {
		t.Fatalf("LocalAmong: %v", err)
	}
	if !slices.Equal(local, []int{never}) {
		t.Errorf("local = %v, want only the row that exists", local)
	}
	if err := repo.MarkCatalogChecked([]int{never}, true); err != nil {
		t.Fatalf("MarkCatalogChecked: %v", err)
	}
	if rendered, _ = repo.RenderedAmong([]int{never}); !slices.Equal(rendered, []int{never}) {
		t.Errorf("a row catalog answers for again must be listed again, got %v", rendered)
	}
}

// Re-reading a page of the changes feed is meant to be free. It is only free if
// a row whose date has not moved is left alone.
func TestSetReleaseDatesSkipsRowsThatAlreadyAgree(t *testing.T) {
	db := testdb.Open(t)
	repo := NewGalgameRepository(db)

	const id = 2_000_300_100
	cleanup := func() { db.Exec("DELETE FROM galgame WHERE id = ?", id) }
	cleanup()
	defer cleanup()
	if err := db.Create(&model.GalgameLocal{ID: id}).Error; err != nil {
		t.Fatalf("seed galgame: %v", err)
	}

	first, err := repo.SetReleaseDates(map[int]string{id: "2026-08-27"})
	if err != nil || first != 1 {
		t.Fatalf("first write = %d, %v; want 1 row", first, err)
	}
	again, err := repo.SetReleaseDates(map[int]string{id: "2026-08-27"})
	if err != nil || again != 0 {
		t.Fatalf("re-applying the same date wrote %d rows, %v; want 0", again, err)
	}
	moved, err := repo.SetReleaseDates(map[int]string{id: "2026-08-28"})
	if err != nil || moved != 1 {
		t.Fatalf("a corrected date wrote %d rows, %v; want 1", moved, err)
	}
}
