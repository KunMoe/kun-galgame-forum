package repository

import (
	"testing"
	"time"

	"kun-galgame-api/internal/testdb"
)

func TestProfileCountsSkipWorksCatalogWillNotRender(t *testing.T) {
	db := testdb.Open(t)
	const owner, listed, unlisted, unmirrored = 2_000_342_999, 2_000_342_000, 2_000_342_001, 2_000_342_002
	cleanup := func() {
		db.Exec("DELETE FROM galgame_rating WHERE user_id = ?", owner)
		db.Exec("DELETE FROM galgame_resource WHERE user_id = ?", owner)
		db.Exec("DELETE FROM galgame WHERE id IN (?, ?)", listed, unlisted)
	}
	cleanup()
	t.Cleanup(cleanup)
	for _, q := range []string{
		`INSERT INTO galgame (id, published, catalog_rendered, created, updated)
			VALUES (@listed, true, true, now(), now()), (@unlisted, true, false, now(), now())`,
		`INSERT INTO galgame_resource (work_id, user_id, updated) VALUES (@listed, @owner, now()), (@unlisted, @owner, now())`,
		`INSERT INTO galgame_rating (id, work_id, user_id, recommend, overall, updated)
			VALUES (nextval('galgame_rating_id_seq'), @listed, @owner, 'yes', 8, now()),
			       (nextval('galgame_rating_id_seq'), @unlisted, @owner, 'yes', 8, now()),
			       (nextval('galgame_rating_id_seq'), @unmirrored, @owner, 'yes', 8, now())`,
	} {
		if err := db.Exec(q, map[string]any{"listed": listed, "unlisted": unlisted, "unmirrored": unmirrored, "owner": owner}).Error; err != nil {
			t.Fatal(err)
		}
	}

	counts, err := NewUserStatsRepository(db).ProfileCounts(owner, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if counts.GalgameRatingCount != 2 || counts.GalgameResourceCount != 1 {
		t.Errorf("ratings %d resources %d, want 2 and 1: a work with no local row counts, an unlisted one does not",
			counts.GalgameRatingCount, counts.GalgameResourceCount)
	}

	if err := db.Exec("UPDATE galgame SET catalog_rendered = true WHERE id = ?", unlisted).Error; err != nil {
		t.Fatal(err)
	}
	counts, err = NewUserStatsRepository(db).ProfileCounts(owner, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if counts.GalgameRatingCount != 3 || counts.GalgameResourceCount != 2 {
		t.Errorf("restored: ratings %d resources %d, want 3 and 2", counts.GalgameRatingCount, counts.GalgameResourceCount)
	}
}
