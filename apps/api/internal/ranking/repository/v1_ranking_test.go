package repository

import (
	"testing"

	"kun-galgame-api/internal/testdb"
)

func TestTopWorksSkipsWorksCatalogWillNotRender(t *testing.T) {
	db := testdb.Open(t)
	const listed, unlisted = 2_000_330_000, 2_000_330_001
	cleanup := func() { db.Exec("DELETE FROM galgame WHERE id IN (?, ?)", listed, unlisted) }
	cleanup()
	t.Cleanup(cleanup)
	if err := db.Exec(`INSERT INTO galgame (id, published, content_limit, view, catalog_rendered, created, updated)
		VALUES (?, true, 'sfw', 2000000000, true, now(), now()), (?, true, 'sfw', 2000000001, false, now(), now())`, listed, unlisted).Error; err != nil {
		t.Fatal(err)
	}

	rows, err := NewRankingRepository(db).TopWorks("views", true, true, true, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.ID == unlisted {
			t.Fatalf("the top views ranking holds a work catalog will not render: %v", rows)
		}
	}
	if len(rows) == 0 || rows[0].ID != listed {
		t.Errorf("rows = %v, want %d first", rows, listed)
	}
}

func TestTopWorksSFWFailsClosedOnUnsyncedVerdict(t *testing.T) {
	db := testdb.Open(t)
	const safe, unsynced = 2_000_331_000, 2_000_331_001
	cleanup := func() { db.Exec("DELETE FROM galgame WHERE id IN (?, ?)", safe, unsynced) }
	cleanup()
	t.Cleanup(cleanup)
	if err := db.Exec(`INSERT INTO galgame (id, published, content_limit, view, catalog_rendered, created, updated)
		VALUES (?, true, 'sfw', 2000000000, true, now(), now()), (?, true, NULL, 2000000002, true, now(), now())`, safe, unsynced).Error; err != nil {
		t.Fatal(err)
	}

	rows, err := NewRankingRepository(db).TopWorks("views", false, true, true, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.ID == unsynced {
			t.Fatalf("an SFW ranking holds a work without a verdict: %v", rows)
		}
	}
	if len(rows) == 0 || rows[0].ID != safe {
		t.Errorf("rows = %v, want %d first", rows, safe)
	}
}

func TestTopWorksLeavesOutOtherOriginalLanguages(t *testing.T) {
	db := testdb.Open(t)
	const japanese, english, chinese, unsynced = 2_000_332_000, 2_000_332_001, 2_000_332_002, 2_000_332_003
	ids := []int{japanese, english, chinese, unsynced}
	cleanup := func() { db.Exec("DELETE FROM galgame WHERE id IN ?", ids) }
	cleanup()
	t.Cleanup(cleanup)
	if err := db.Exec(`INSERT INTO galgame (id, published, content_limit, view, catalog_rendered, original_language, created, updated)
		VALUES (?, true, 'sfw', 2000000000, true, 'ja', now(), now()), (?, true, 'sfw', 2000000003, true, 'en', now(), now()),
		       (?, true, 'sfw', 2000000002, true, 'zh-Hant', now(), now()), (?, true, 'sfw', 2000000001, true, NULL, now(), now())`,
		japanese, english, chinese, unsynced).Error; err != nil {
		t.Fatal(err)
	}

	got := func(includeAll bool) []int {
		rows, err := NewRankingRepository(db).TopWorks("views", true, true, includeAll, 4, nil)
		if err != nil {
			t.Fatal(err)
		}
		out := make([]int, len(rows))
		for i, r := range rows {
			out[i] = r.ID
		}
		return out
	}
	if rows := got(false); len(rows) < 3 || rows[0] != chinese || rows[1] != unsynced || rows[2] != japanese {
		t.Errorf("default ranking = %v, want %d, %d, %d first", rows, chinese, unsynced, japanese)
	}
	if rows := got(true); len(rows) == 0 || rows[0] != english {
		t.Errorf("every-language ranking = %v, want %d first", rows, english)
	}
}

func TestTopUsersByResourcesSkipsWorksCatalogWillNotRender(t *testing.T) {
	db := testdb.Open(t)
	const uploader, listed, unlisted = 2_000_333_999, 2_000_333_000, 2_000_333_001
	cleanup := func() {
		db.Exec("DELETE FROM galgame_resource WHERE user_id = ?", uploader)
		db.Exec("DELETE FROM galgame WHERE id IN (?, ?)", listed, unlisted)
	}
	cleanup()
	t.Cleanup(cleanup)
	if err := db.Exec(`INSERT INTO galgame (id, published, catalog_rendered, created, updated)
		VALUES (?, true, true, now(), now()), (?, true, false, now(), now())`, listed, unlisted).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO galgame_resource (work_id, user_id, status, updated)
		SELECT CASE WHEN n = 1 THEN ?::int ELSE ?::int END, ?, 0, now() FROM generate_series(1, 3) n`, listed, unlisted, uploader).Error; err != nil {
		t.Fatal(err)
	}

	value := func() float64 {
		t.Helper()
		rows, err := NewRankingRepository(db).TopUsers("resources", 1_000_000)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range rows {
			if r.ID == uploader {
				return r.Value
			}
		}
		return 0
	}
	if v := value(); v != 1 {
		t.Errorf("resources = %v, want 1: the two on an unlisted work do not count", v)
	}
	if err := db.Exec("UPDATE galgame SET catalog_rendered = true WHERE id = ?", unlisted).Error; err != nil {
		t.Fatal(err)
	}
	if v := value(); v != 3 {
		t.Errorf("restored resources = %v, want 3", v)
	}
}
