package repository

import (
	"testing"

	"kun-galgame-api/internal/testdb"
)

func TestUnrenderedRowIDsFollowsTheSubjectsWork(t *testing.T) {
	db := testdb.Open(t)
	const (
		work   = 2_000_460_001
		quiz   = 2_000_460_101
		user   = 2_000_460_901
		ownRow = 2_000_460_201
		other  = 2_000_460_203
	)
	clean := func() {
		db.Exec(`DELETE FROM feed_activity WHERE source_id BETWEEN ? AND ?`, ownRow, other)
		db.Exec(`DELETE FROM galgame_quiz WHERE id = ?`, quiz)
		db.Exec(`DELETE FROM galgame WHERE id = ?`, work)
	}
	clean()
	t.Cleanup(clean)
	run := func(q string, args ...any) {
		t.Helper()
		if err := db.Exec(q, args...).Error; err != nil {
			t.Fatal(err)
		}
	}
	run(`INSERT INTO galgame (id, updated, catalog_rendered) VALUES (?, now(), false)`, work)
	run(`INSERT INTO galgame_quiz (id, user_id, category, type, difficulty, spoiler_level, question, description, content, explanation, hide_galgame, view, answer_count, correct_count, favorite_count, quality_sum, quality_count, comment_count, status_update_time, created, updated)
		VALUES (?, ?, 'plot', 'single', 3, 'none', 'q', '', '{"options":["a","b"],"answer":1}'::jsonb, '', false, 0, 0, 0, 0, 0, 0, 0, now(), now(), now())`,
		quiz, user)
	run(`INSERT INTO galgame_quiz_galgame (quiz_id, work_id) VALUES (?, ?)`, quiz, work)
	run(`INSERT INTO feed_activity (type, source_id, user_id, work_id, content, link, created) VALUES
		('GALGAME_RESOURCE_CREATION', ?, ?, ?, 'r', '/galgame/resource/1', now()),
		('TOPIC_CREATION', ?, ?, 0, 't', '/topic/1', now())`,
		ownRow, user, work, other, user)
	var ids []int64
	if err := db.Raw(`SELECT id FROM feed_activity WHERE (type = 'GALGAME_QUIZ_CREATION' AND source_id = ?) OR source_id IN (?, ?)
		ORDER BY source_id`, quiz, ownRow, other).Scan(&ids).Error; err != nil || len(ids) != 3 {
		t.Fatalf("ids %v %v", ids, err)
	}
	repo := NewActivityRepository(db)

	hidden, err := repo.UnrenderedRowIDs(ids)
	if err != nil {
		t.Fatal(err)
	}
	if len(hidden) != 2 || hidden[ids[2]] {
		t.Errorf("hidden = %v, want the work's own row and its quiz's row, not the topic", hidden)
	}

	run(`UPDATE galgame SET catalog_rendered = true WHERE id = ?`, work)
	if hidden, err = repo.UnrenderedRowIDs(ids); err != nil || len(hidden) != 0 {
		t.Errorf("after restore hidden = %v %v, want none", hidden, err)
	}
}
