package repository

import (
	"slices"
	"testing"

	"kun-galgame-api/internal/testdb"
)

func TestUserWorksSkipWorksCatalogWillNotRender(t *testing.T) {
	db := testdb.Open(t)
	const owner, listed, unlisted = 2_000_340_999, 2_000_340_000, 2_000_340_001
	cleanup := func() { db.Exec("DELETE FROM galgame WHERE id IN (?, ?)", listed, unlisted) }
	cleanup()
	t.Cleanup(cleanup)
	if err := db.Exec(`INSERT INTO galgame (id, published, content_limit, creator_user_id, catalog_rendered, created, updated)
		VALUES (?, true, 'sfw', ?, true, now(), now()), (?, true, 'sfw', ?, false, now(), now())`, listed, owner, unlisted, owner).Error; err != nil {
		t.Fatal(err)
	}

	for _, nsfw := range []bool{false, true} {
		ids, total, err := NewUserContentRepository(db).ListUserWorkIDs(UserWorkQuery{
			OwnerID: owner, Relation: "published", IncludeNSFW: nsfw, Limit: 10,
		})
		if err != nil {
			t.Fatal(err)
		}
		if total != 1 || !slices.Equal(ids, []int{listed}) {
			t.Errorf("works (nsfw=%v) = %v (total %d), want only %d", nsfw, ids, total, listed)
		}
	}
}

func TestUserWorksSFWFailsClosedOnUnsyncedVerdict(t *testing.T) {
	db := testdb.Open(t)
	const owner, safe, unsynced = 2_000_341_999, 2_000_341_000, 2_000_341_001
	cleanup := func() { db.Exec("DELETE FROM galgame WHERE id IN (?, ?)", safe, unsynced) }
	cleanup()
	t.Cleanup(cleanup)
	if err := db.Exec(`INSERT INTO galgame (id, published, content_limit, creator_user_id, catalog_rendered, created, updated)
		VALUES (?, true, 'sfw', ?, true, now(), now()), (?, true, NULL, ?, true, now(), now())`, safe, owner, unsynced, owner).Error; err != nil {
		t.Fatal(err)
	}

	for nsfw, want := range map[bool][]int{false: {safe}, true: {unsynced, safe}} {
		ids, _, err := NewUserContentRepository(db).ListUserWorkIDs(UserWorkQuery{
			OwnerID: owner, Relation: "published", IncludeNSFW: nsfw, Limit: 10,
		})
		if err != nil {
			t.Fatal(err)
		}
		slices.Sort(ids)
		slices.Sort(want)
		if !slices.Equal(ids, want) {
			t.Errorf("works (nsfw=%v) = %v, want %v: a row without a verdict is not sfw", nsfw, ids, want)
		}
	}
}
