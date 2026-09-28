package repository

import (
	"slices"
	"testing"

	"kun-galgame-api/internal/testdb"

	"gorm.io/gorm"
)

const (
	mfDead     = 2_000_350_001
	mfSurvivor = 2_000_350_002
	mfQuiz     = 2_000_350_101
	mfComment  = 2_000_350_201
	mfAuthor   = 2_000_350_901
)

func mergeFixture(t *testing.T) (*gorm.DB, func(string, ...any)) {
	t.Helper()
	db := testdb.Open(t)
	cleanup := func() {
		db.Exec("DELETE FROM feed_activity WHERE user_id = ?", mfAuthor)
		db.Exec("DELETE FROM activity_push_queue WHERE source_id = ?", mfComment)
		db.Exec("DELETE FROM galgame_quiz WHERE id = ?", mfQuiz)
		db.Exec("DELETE FROM galgame_view_daily WHERE entity_id IN (?, ?)", mfDead, mfSurvivor)
		db.Exec("DELETE FROM galgame_resource WHERE work_id IN (?, ?)", mfDead, mfSurvivor)
		db.Exec("DELETE FROM galgame WHERE id IN (?, ?)", mfDead, mfSurvivor)
	}
	cleanup()
	t.Cleanup(cleanup)
	run := func(q string, args ...any) {
		t.Helper()
		if err := db.Exec(q, args...).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db, run
}

func scalar(t *testing.T, db *gorm.DB, q string, args ...any) int {
	t.Helper()
	var n int
	if err := db.Raw(q, args...).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func TestFoldFollowsAMergedWorkWithNoLocalRow(t *testing.T) {
	db, run := mergeFixture(t)
	seed(t, db, mfSurvivor, ptr("sfw"))
	run(`INSERT INTO galgame_quiz (id, user_id, category, type, difficulty, spoiler_level, question, description, content, explanation, hide_galgame, view, answer_count, correct_count, favorite_count, quality_sum, quality_count, comment_count, status_update_time, created, updated)
		VALUES (?, ?, 'plot', 'single', 3, 'none', 'q', '', '{"options":["a","b"],"answer":1}'::jsonb, '', false, 0, 0, 0, 0, 0, 0, 0, now(), now(), now())`,
		mfQuiz, mfAuthor)
	run(`INSERT INTO galgame_quiz_galgame (quiz_id, work_id) VALUES (?, ?)`, mfQuiz, mfDead)
	run(`INSERT INTO feed_activity (type, source_id, user_id, work_id, content, link, created)
		VALUES ('GALGAME_COMMENT_CREATION', ?, ?, ?, 'c', ?, now())`,
		mfComment, mfAuthor, mfDead, "/galgame/2000350001")
	run(`INSERT INTO galgame_view_daily (entity_id, day, count) VALUES (?, CURRENT_DATE, 3), (?, CURRENT_DATE, 2)`,
		mfDead, mfSurvivor)

	repo := NewGalgameMergeRepository(db)
	const untouched = mfDead + 50
	ids, err := repo.ReferencedIDsIn([]int{mfDead, untouched})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(ids, []int{mfDead}) {
		t.Fatalf("referenced = %v, want [%d]: the dead work has no galgame row but three tables still name it", ids, mfDead)
	}

	if _, err := repo.Fold(mfDead, mfSurvivor); err != nil {
		t.Fatalf("fold: %v", err)
	}

	if n := scalar(t, db, `SELECT count(*) FROM galgame_quiz_galgame WHERE quiz_id = ? AND work_id = ?`, mfQuiz, mfSurvivor); n != 1 {
		t.Errorf("quiz links on the survivor = %d, want 1", n)
	}
	var feed struct {
		WorkID int
		Link   string
	}
	db.Raw(`SELECT work_id, link FROM feed_activity WHERE type = 'GALGAME_COMMENT_CREATION' AND source_id = ?`, mfComment).Scan(&feed)
	if feed.WorkID != mfSurvivor || feed.Link != "/galgame/2000350002" {
		t.Errorf("feed row = %+v, want it on the survivor", feed)
	}
	if n := scalar(t, db, `SELECT count FROM galgame_view_daily WHERE entity_id = ? AND day = CURRENT_DATE`, mfSurvivor); n != 5 {
		t.Errorf("survivor's views today = %d, want 5", n)
	}
	if n := scalar(t, db, `SELECT view_7d FROM galgame WHERE id = ?`, mfSurvivor); n != 5 {
		t.Errorf("survivor view_7d = %d, want 5 after the recount", n)
	}
	if n := scalar(t, db, `SELECT count(*) FROM galgame WHERE id = ?`, mfDead); n != 0 {
		t.Errorf("fold created a galgame row for the dead work")
	}

	ids, err = repo.ReferencedIDsIn([]int{mfDead})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Errorf("referenced after fold = %v, want none so a replay is a no-op", ids)
	}
}

func TestFoldStillRetiresALocalRow(t *testing.T) {
	db, _ := mergeFixture(t)
	seed(t, db, mfDead, ptr("sfw"))

	if _, err := NewGalgameMergeRepository(db).Fold(mfDead, mfSurvivor); err != nil {
		t.Fatalf("fold: %v", err)
	}

	if n := scalar(t, db, `SELECT count(*) FROM galgame WHERE id = ?`, mfDead); n != 0 {
		t.Errorf("dead galgame row still present")
	}
	if n := scalar(t, db, `SELECT count(*) FROM galgame WHERE id = ? AND published`, mfSurvivor); n != 1 {
		t.Errorf("survivor row missing or not published")
	}
	if n := scalar(t, db, `SELECT resource_count FROM galgame WHERE id = ?`, mfSurvivor); n != 1 {
		t.Errorf("survivor resource_count = %d, want the dead work's resource", n)
	}
}
