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
