package repository

import (
	"slices"
	"testing"

	"kun-galgame-api/internal/testdb"

	"gorm.io/gorm"
)

func seedRenderedPair(t *testing.T, db *gorm.DB, base, owner int) (listed, unlisted int) {
	t.Helper()
	listed, unlisted = base, base+1
	cleanup := func() {
		db.Exec("DELETE FROM galgame_rating WHERE work_id IN (?, ?)", listed, unlisted)
		db.Exec("DELETE FROM galgame_resource WHERE work_id IN (?, ?)", listed, unlisted)
		db.Exec("DELETE FROM galgame WHERE id IN (?, ?)", listed, unlisted)
	}
	cleanup()
	t.Cleanup(cleanup)
	for _, id := range []int{listed, unlisted} {
		seed(t, db, id, ptr("sfw"))
		if err := db.Exec(`UPDATE galgame SET creator_user_id = ? WHERE id = ?`, owner, id).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Exec(`INSERT INTO galgame_rating (work_id, user_id, overall, recommend, updated)
			VALUES (?, ?, 7, 'yes', now())`, id, owner).Error; err != nil {
			t.Fatal(err)
		}
	}
	db.Exec("UPDATE galgame_resource SET user_id = ? WHERE work_id IN (?, ?)", owner, listed, unlisted)
	db.Exec("UPDATE galgame SET catalog_rendered = false WHERE id = ?", unlisted)
	return listed, unlisted
}

func TestListsSkipWorksCatalogWillNotRender(t *testing.T) {
	db := testdb.Open(t)
	const owner = 2_000_320_999
	listed, _ := seedRenderedPair(t, db, 2_000_320_000, owner)

	ids, total, err := NewGalgameRepository(db).PublishedIDsByCreator(owner, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || !slices.Equal(ids, []int{listed}) {
		t.Errorf("creator's works = %v (total %d), want only %d", ids, total, listed)
	}

	for _, sfw := range []bool{false, true} {
		n, err := NewResourceV1Store(db).Count(ResourceListFilter{UploaderID: owner, IncludeNSFW: !sfw})
		if err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("resources (sfw=%v) = %d, want only the listed work's", sfw, n)
		}

		ratings, total, err := NewRatingStore(db).List(RatingQuery{AuthorID: owner, SFWOnly: sfw, Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		if total != 1 || len(ratings) != 1 || ratings[0].WorkID != listed {
			t.Errorf("ratings (sfw=%v) = %d rows (total %d), want only the listed work's", sfw, len(ratings), total)
		}
	}
}
