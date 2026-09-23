package cron

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"kun-galgame-api/internal/testdb"
)

func TestCollectContentImageHashesCoversLotteryPrizeImages(t *testing.T) {
	db := testdb.Open(t)
	const topicID, userID = 930000298, 930000990
	hash := strings.Repeat("c7", 32)
	cleanup := func() {
		_ = db.Exec(`DELETE FROM topic_lottery WHERE topic_id = ?`, topicID).Error
		_ = db.Exec(`DELETE FROM topic WHERE id = ?`, topicID).Error
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
	run(`INSERT INTO topic (
		id, title, content, view, status, category, status_update_time, created, updated,
		user_id, is_nsfw, access_scope, cover_images, like_count, dislike_count, reply_count, comment_count,
		favorite_count, upvote_count, view_7d, view_30d, hidden_by, last_reply_floor
	) VALUES (?, 't', 'no image token here', 0, 0, 'galgame', ?, ?, ?, ?, false, 'public', '', 0, 0, 0, 0, 0, 0, 0, 0, '', 0)`,
		topicID, now, now, now, userID)
	var lotteryID int
	if err := db.Raw(`INSERT INTO topic_lottery (topic_id, user_id, title, created, updated)
		VALUES (?, ?, 'l', ?, ?) RETURNING id`, topicID, userID, now, now).Scan(&lotteryID).Error; err != nil {
		t.Fatal(err)
	}
	run(`INSERT INTO topic_lottery_prize (lottery_id, name, image_hashes, created, updated)
		VALUES (?, 'p', ?::jsonb, ?, ?)`, lotteryID, `["`+hash+`"]`, now, now)

	got, err := collectContentImageHashes(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(got, hash) {
		t.Fatalf("a prize image stored as a bare hash in jsonb is not pinged, so the image service may collect it")
	}
}

// Columns whose name says "hash" but that do not hold an image of their own.
var notImageHashColumns = map[string]string{
	"topic_lottery.seed_hash":         "commitment hash of the draw seed, not an image",
	"topic_lottery_prize.nsfw_hashes": "marks which of the same row's image_hashes are adult; always a subset of them",
}

func TestEveryHashColumnIsPingedOrExempt(t *testing.T) {
	db := testdb.Open(t)
	var named []string
	if err := db.Raw(`SELECT table_name || '.' || column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND column_name ILIKE '%hash%' ORDER BY 1`).Scan(&named).Error; err != nil {
		t.Fatal(err)
	}
	cols, err := bareImageHashColumns(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	pinged := map[string]bool{}
	for _, c := range cols {
		pinged[c.Table+"."+c.Column] = true
	}
	for _, want := range []string{
		"doc_article.banner_image_hash", "friend_link.banner_image_hash",
		"galgame_website.icon_image_hash", "topic_lottery_prize.image_hashes",
	} {
		if !pinged[want] {
			t.Errorf("%s is not pinged", want)
		}
	}
	for _, col := range named {
		if !pinged[col] && notImageHashColumns[col] == "" {
			t.Errorf("%s looks like a hash column but is neither pinged nor exempt: name an image hash column *image_hash (text) or *image_hashes (jsonb), or add it to notImageHashColumns with the reason", col)
		}
	}
}

func TestCollectContentImageHashesCoversBareHashColumns(t *testing.T) {
	db := testdb.Open(t)
	const (
		catID, siteID, docCatID, docID, linkID = 930000791, 930000891, 930000792, 930000793, 930000794
	)
	doc, link, icon := strings.Repeat("d1", 32), strings.Repeat("e2", 32), strings.Repeat("f3", 32)
	cleanup := func() {
		_ = db.Exec(`DELETE FROM galgame_website WHERE id = ?`, siteID).Error
		_ = db.Exec(`DELETE FROM galgame_website_category WHERE id = ?`, catID).Error
		_ = db.Exec(`DELETE FROM doc_article WHERE id = ?`, docID).Error
		_ = db.Exec(`DELETE FROM doc_category WHERE id = ?`, docCatID).Error
		_ = db.Exec(`DELETE FROM friend_link WHERE id = ?`, linkID).Error
	}
	cleanup()
	t.Cleanup(cleanup)
	now := time.Now()
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO galgame_website_category (id, name, created, updated) VALUES (?, 'ping-test', ?, ?)`, []any{catID, now, now}},
		{`INSERT INTO galgame_website (id, name, url, create_time, category_id, icon_image_hash, created, updated)
			VALUES (?, 'ping test', 'ping-test.example', '', ?, ?, ?, ?)`, []any{siteID, catID, icon, now, now}},
		{`INSERT INTO doc_category (id, slug, title, updated) VALUES (?, 'ping-test', 't', ?)`, []any{docCatID, now}},
		{`INSERT INTO doc_article (id, title, slug, path, description, content_markdown, category_id, author_id, updated, banner_image_hash)
			VALUES (?, 't', 'ping-test', '/doc/ping-test', '', '', ?, 930000990, ?, ?)`, []any{docID, docCatID, now, doc}},
		{`INSERT INTO friend_link (id, category, name, link, banner_image_hash) VALUES (?, 'c', 'n', 'https://ping-test.example', ?)`, []any{linkID, link}},
	} {
		if err := db.Exec(q.sql, q.args...).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, q.sql)
		}
	}

	got, err := collectContentImageHashes(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	for name, hash := range map[string]string{"doc banner": doc, "friend-link banner": link, "website icon": icon} {
		if !slices.Contains(got, hash) {
			t.Errorf("the %s hash is not pinged", name)
		}
	}
}
