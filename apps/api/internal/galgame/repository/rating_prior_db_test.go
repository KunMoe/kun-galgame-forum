package repository

import (
	"math"
	"testing"

	"kun-galgame-api/internal/testdb"
)

func TestRatingPriorMeanLeavesOutHiddenWorks(t *testing.T) {
	db := testdb.Open(t)
	const hidden, unmirrored = 2_000_470_001, 2_000_470_002
	clean := func() {
		db.Exec(`DELETE FROM galgame_rating WHERE work_id IN (?, ?)`, hidden, unmirrored)
		db.Exec(`DELETE FROM galgame WHERE id = ?`, hidden)
	}
	clean()
	t.Cleanup(clean)
	run := func(q string, args ...any) {
		t.Helper()
		if err := db.Exec(q, args...).Error; err != nil {
			t.Fatal(err)
		}
	}
	run(`INSERT INTO galgame (id, updated, catalog_rendered) VALUES (?, now(), false)`, hidden)
	run(`INSERT INTO galgame_rating (id, recommend, overall, updated, user_id, work_id) VALUES
		(2000470101, 'yes', 1, now(), 1, ?), (2000470102, 'yes', 1, now(), 2, ?), (2000470103, 'yes', 9, now(), 3, ?)`,
		hidden, hidden, unmirrored)
	avg := func(q string, args ...any) float64 {
		var m float64
		if err := db.Raw(q, args...).Scan(&m).Error; err != nil {
			t.Fatal(err)
		}
		return m
	}

	got, err := RatingPriorMean(db)
	if err != nil {
		t.Fatal(err)
	}
	if want := avg(`SELECT COALESCE(AVG(overall), 0) FROM galgame_rating WHERE work_id <> ?`, hidden); math.Abs(got-want) > 1e-9 {
		t.Errorf("mean with the work hidden = %v, want %v (its ratings out, the unmirrored work's in)", got, want)
	}

	run(`UPDATE galgame SET catalog_rendered = true WHERE id = ?`, hidden)
	got, err = RatingPriorMean(db)
	if err != nil {
		t.Fatal(err)
	}
	if want := avg(`SELECT COALESCE(AVG(overall), 0) FROM galgame_rating`); math.Abs(got-want) > 1e-9 {
		t.Errorf("mean after restore = %v, want %v", got, want)
	}
}
