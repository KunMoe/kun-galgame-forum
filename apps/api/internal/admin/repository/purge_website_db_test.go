package repository

import (
	"fmt"
	"testing"
	"time"

	"kun-galgame-api/internal/testdb"
)

func TestPurgeKeepsSharedWebsites(t *testing.T) {
	db := testdb.Open(t)
	const (
		purged, other = 930000981, 930000982
		catID, tagID  = 930000781, 930000782
		listed, liked = 930000881, 930000882
	)
	cleanup := func() {
		for _, q := range []string{
			`DELETE FROM galgame_website_like WHERE website_id IN (930000881, 930000882)`,
			`DELETE FROM galgame_website_favorite WHERE website_id IN (930000881, 930000882)`,
			`DELETE FROM galgame_website_tag_relation WHERE galgame_website_id IN (930000881, 930000882)`,
			`DELETE FROM galgame_website WHERE id IN (930000881, 930000882)`,
			`DELETE FROM galgame_website_tag WHERE id = 930000782`,
			`DELETE FROM galgame_website_category WHERE id = 930000781`,
		} {
			_ = db.Exec(q).Error
		}
	}
	cleanup()
	t.Cleanup(cleanup)
	now := time.Now()
	run := func(q string, args ...any) {
		t.Helper()
		if err := db.Exec(q, args...).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, q)
		}
	}
	run(`INSERT INTO galgame_website_category (id, name, created, updated) VALUES (?, 'purge-test', ?, ?)`, catID, now, now)
	run(`INSERT INTO galgame_website_tag (id, level, name, created, updated) VALUES (?, 5, 'purge-test', ?, ?)`, tagID, now, now)
	site := func(id, owner int) {
		run(`INSERT INTO galgame_website (id, name, url, create_time, category_id, user_id, like_count, favorite_count, created, updated)
			VALUES (?, ?, ?, '', ?, ?, 1, 0, ?, ?)`, id, fmt.Sprintf("purge site %d", id), fmt.Sprintf("purge-%d.example", id), catID, owner, now, now)
	}
	site(listed, purged)
	site(liked, other)
	run(`INSERT INTO galgame_website_like (user_id, website_id, created, updated) VALUES (?, ?, ?, ?), (?, ?, ?, ?)`,
		other, listed, now, now, purged, liked, now, now)
	run(`INSERT INTO galgame_website_favorite (user_id, website_id, created, updated) VALUES (?, ?, ?, ?)`, other, listed, now, now)
	run(`UPDATE galgame_website SET favorite_count = 1 WHERE id = ?`, listed)
	run(`INSERT INTO galgame_website_tag_relation (galgame_website_id, galgame_website_tag_id, created, updated) VALUES (?, ?, ?, ?)`,
		listed, tagID, now, now)

	repo := NewPurgeRepository(db)
	if _, err := repo.PurgeUserContent(purged, 1); err != nil {
		t.Fatal(err)
	}
	scalar := func(q string, args ...any) int {
		t.Helper()
		var n int
		if err := db.Raw(q, args...).Scan(&n).Error; err != nil {
			t.Fatal(err)
		}
		return n
	}
	var owner int
	if err := db.Raw(`SELECT user_id FROM galgame_website WHERE id = ?`, listed).Scan(&owner).Error; err != nil {
		t.Fatal(err)
	}
	var defaultOwner int
	if err := db.Raw(`SELECT column_default::int FROM information_schema.columns
		WHERE table_name = 'galgame_website' AND column_name = 'user_id'`).Scan(&defaultOwner).Error; err != nil {
		t.Fatal(err)
	}
	if n := scalar(`SELECT COUNT(*) FROM galgame_website WHERE id = ?`, listed); n != 1 || owner != defaultOwner {
		t.Fatalf("the purged user's listing: rows %d owner %d, want 1 owned by %d", n, owner, defaultOwner)
	}
	for _, c := range []struct {
		what string
		q    string
		want int
	}{
		{"another user's like on the listing", `SELECT COUNT(*) FROM galgame_website_like WHERE website_id = 930000881 AND user_id = 930000982`, 1},
		{"another user's favorite on the listing", `SELECT COUNT(*) FROM galgame_website_favorite WHERE website_id = 930000881`, 1},
		{"the listing's tag link", `SELECT COUNT(*) FROM galgame_website_tag_relation WHERE galgame_website_id = 930000881`, 1},
		{"the listing's like_count", `SELECT like_count FROM galgame_website WHERE id = 930000881`, 1},
		{"the purged user's own like", `SELECT COUNT(*) FROM galgame_website_like WHERE user_id = 930000981`, 0},
		{"the recounted like_count of the site they liked", `SELECT like_count FROM galgame_website WHERE id = 930000882`, 0},
		{"the purged user's listing feed actor", `SELECT COUNT(*) FROM feed_activity WHERE type = 'GALGAME_WEBSITE_CREATION' AND user_id = 930000981`, 0},
	} {
		if got := scalar(c.q); got != c.want {
			t.Errorf("%s: %d, want %d", c.what, got, c.want)
		}
	}
	if again, err := repo.CountUserContent(purged); err != nil || again.Websites != 0 {
		t.Errorf("a second preview still counts %d websites (%v)", again.Websites, err)
	}
}
