package repository

import (
	"testing"

	"kun-galgame-api/internal/testdb"
)

func TestFavoriteRowsFollowTheirTopic(t *testing.T) {
	db := testdb.Open(t)
	const (
		topic = 2_000_480_001
		fan   = 2_000_480_901
		other = 2_000_480_902
	)
	clean := func() {
		db.Exec(`DELETE FROM topic_favorite WHERE topic_id = ?`, topic)
		db.Exec(`DELETE FROM topic WHERE id = ?`, topic)
		db.Exec(`DELETE FROM feed_activity WHERE user_id IN (?, ?)`, fan, other)
	}
	clean()
	t.Cleanup(clean)
	run := func(q string, args ...any) {
		t.Helper()
		if err := db.Exec(q, args...).Error; err != nil {
			t.Fatal(err)
		}
	}
	rows := func() (n int, scope, link string) {
		t.Helper()
		if err := db.Raw(`SELECT count(*), COALESCE(max(access_scope), ''), COALESCE(max(link), '') FROM feed_activity WHERE type = 'TOPIC_FAVORITE' AND user_id = ?`, fan).
			Row().Scan(&n, &scope, &link); err != nil {
			t.Fatal(err)
		}
		return n, scope, link
	}

	run(`INSERT INTO topic (id, title, content, category, user_id, updated) VALUES (?, 't', 'c', 'galgame', ?, now())`, topic, other)
	run(`INSERT INTO topic_favorite (topic_id, user_id, updated) VALUES (?, ?, now())`, topic, fan)
	if n, scope, link := rows(); n != 1 || scope != "public" || link != "/topic/2000480001" {
		t.Fatalf("favorite rows %d scope %q link %q, want 1 public /topic/2000480001", n, scope, link)
	}

	run(`UPDATE topic SET access_scope = 'login' WHERE id = ?`, topic)
	if n, scope, _ := rows(); n != 1 || scope != "login" {
		t.Errorf("after login: favorite rows %d scope %q, want 1 login", n, scope)
	}
	run(`UPDATE topic SET access_scope = 'role' WHERE id = ?`, topic)
	if n, _, _ := rows(); n != 0 {
		t.Errorf("a role topic kept %d favorite rows", n)
	}
	run(`UPDATE topic SET access_scope = 'public' WHERE id = ?`, topic)
	if n, _, _ := rows(); n != 1 {
		t.Errorf("back to public: %d favorite rows, want 1", n)
	}

	run(`DELETE FROM topic_favorite WHERE topic_id = ? AND user_id = ?`, topic, fan)
	if n, _, _ := rows(); n != 0 {
		t.Errorf("an unfavorited topic kept its row")
	}
	var queued int
	if err := db.Raw(`SELECT count(*) FROM activity_push_queue WHERE type = 'TOPIC_FAVORITE'`).Scan(&queued).Error; err != nil {
		t.Fatal(err)
	}
	if queued == 0 {
		t.Error("favorite rows never reached the push queue")
	}
}
