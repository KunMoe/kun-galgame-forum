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

	rows, err := NewRankingRepository(db).TopWorks("views", true, true, 2)
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
