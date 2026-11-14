package repository

import (
	"testing"

	"kun-galgame-api/internal/testdb"
)

func TestLikeRowsFollowTheirTopicAndReply(t *testing.T) {
	db := testdb.Open(t)
	const (
		topic = 2_000_470_001
		reply = 2_000_470_101
		liker = 2_000_470_901
		other = 2_000_470_902
		work  = 2_000_470_501
	)
	clean := func() {
		db.Exec(`DELETE FROM galgame_like WHERE work_id = ?`, work)
		db.Exec(`DELETE FROM galgame WHERE id = ?`, work)
		db.Exec(`DELETE FROM topic_reply_reaction WHERE topic_reply_id = ?`, reply)
		db.Exec(`DELETE FROM topic_reaction WHERE topic_id = ?`, topic)
		db.Exec(`DELETE FROM topic_reply WHERE id = ?`, reply)
		db.Exec(`DELETE FROM topic WHERE id = ?`, topic)
		db.Exec(`DELETE FROM feed_activity WHERE user_id IN (?, ?)`, liker, other)
	}
	clean()
	t.Cleanup(clean)
	run := func(q string, args ...any) {
		t.Helper()
		if err := db.Exec(q, args...).Error; err != nil {
			t.Fatal(err)
		}
	}
	rows := func(kind string) (n int, scope string) {
		t.Helper()
		db.Raw(`SELECT count(*), COALESCE(max(access_scope), '') FROM feed_activity WHERE type = ? AND user_id = ?`, kind, liker).
			Row().Scan(&n, &scope)
		return n, scope
	}

	run(`INSERT INTO topic (id, title, content, category, user_id, updated) VALUES (?, 't', 'c', 'galgame', ?, now())`, topic, other)
	run(`INSERT INTO topic_reply (id, topic_id, user_id, updated) VALUES (?, ?, ?, now())`, reply, topic, other)
	run(`INSERT INTO topic_reaction (topic_id, user_id, reaction) VALUES (?, ?, 'like'), (?, ?, 'dislike')`, topic, liker, topic, other)
	run(`INSERT INTO topic_reply_reaction (topic_reply_id, user_id, reaction) VALUES (?, ?, 'like')`, reply, liker)

	if n, scope := rows("TOPIC_LIKE"); n != 1 || scope != "public" {
		t.Fatalf("topic like rows %d scope %q, want 1 public", n, scope)
	}
	var dislikes int
	db.Raw(`SELECT count(*) FROM feed_activity WHERE type = 'TOPIC_LIKE' AND user_id = ?`, other).Scan(&dislikes)
	if dislikes != 0 {
		t.Errorf("a dislike became a like row")
	}
	if n, _ := rows("TOPIC_REPLY_LIKE"); n != 1 {
		t.Fatalf("reply like rows %d, want 1", n)
	}

	run(`UPDATE topic SET access_scope = 'login' WHERE id = ?`, topic)
	if n, scope := rows("TOPIC_LIKE"); n != 1 || scope != "login" {
		t.Errorf("after login: topic like rows %d scope %q, want 1 login", n, scope)
	}
	if _, scope := rows("TOPIC_REPLY_LIKE"); scope != "login" {
		t.Errorf("after login: reply like scope %q", scope)
	}

	run(`UPDATE topic SET access_scope = 'role' WHERE id = ?`, topic)
	if n, _ := rows("TOPIC_LIKE"); n != 0 {
		t.Errorf("a role topic kept %d like rows", n)
	}
	if n, _ := rows("TOPIC_REPLY_LIKE"); n != 0 {
		t.Errorf("a role topic kept %d reply like rows", n)
	}

	run(`UPDATE topic SET access_scope = 'public' WHERE id = ?`, topic)
	if n, scope := rows("TOPIC_LIKE"); n != 1 || scope != "public" {
		t.Errorf("back to public: topic like rows %d scope %q", n, scope)
	}

	run(`UPDATE topic_reply SET status = 1 WHERE id = ?`, reply)
	if n, _ := rows("TOPIC_REPLY_LIKE"); n != 0 {
		t.Errorf("a hidden reply kept %d like rows", n)
	}
	run(`UPDATE topic_reply SET status = 0 WHERE id = ?`, reply)
	if n, _ := rows("TOPIC_REPLY_LIKE"); n != 1 {
		t.Errorf("a shown reply has %d like rows, want 1", n)
	}

	run(`UPDATE topic_reaction SET reaction = 'dislike' WHERE topic_id = ? AND user_id = ?`, topic, liker)
	if n, _ := rows("TOPIC_LIKE"); n != 0 {
		t.Errorf("a like turned dislike kept its row")
	}

	run(`INSERT INTO galgame (id, updated) VALUES (?, now())`, work)
	run(`INSERT INTO galgame_like (work_id, user_id, updated) VALUES (?, ?, now())`, work, liker)
	var workID int
	db.Raw(`SELECT work_id FROM feed_activity WHERE type = 'GALGAME_LIKE' AND user_id = ?`, liker).Scan(&workID)
	if workID != work {
		t.Errorf("galgame like row work_id %d, want %d", workID, work)
	}
	run(`DELETE FROM galgame_like WHERE work_id = ? AND user_id = ?`, work, liker)
	if n, _ := rows("GALGAME_LIKE"); n != 0 {
		t.Errorf("an unliked work kept its row")
	}

	var queued int
	db.Raw(`SELECT count(*) FROM activity_push_queue WHERE type IN ('TOPIC_LIKE', 'GALGAME_LIKE')`).Scan(&queued)
	if queued == 0 {
		t.Error("like rows never reached the push queue")
	}
}
