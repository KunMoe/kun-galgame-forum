package repository

import (
	"encoding/json"
	"testing"

	"gorm.io/gorm"
)

const (
	ufUserBoth    = 2_000_350_911
	ufUserDead    = 2_000_350_912
	ufUserPurged  = 2_000_350_913
	ufFolded      = 2_000_350_003
	ufDay         = "2026-10-01"
	ufSurvivorRes = 2_000_350_301
)

func unfoldFixture(t *testing.T) (*gorm.DB, func(string, ...any)) {
	t.Helper()
	db, run := mergeFixture(t)
	cleanup := func() {
		ids := []int{mfDead, mfSurvivor, ufFolded}
		db.Exec("DELETE FROM galgame_merge_fold WHERE old_work_id IN ? OR new_work_id IN ?", ids, ids)
		db.Exec("DELETE FROM galgame_merge_discarded WHERE old_work_id IN ?", ids)
		db.Exec("DELETE FROM galgame_like WHERE work_id IN ?", ids)
		db.Exec("DELETE FROM galgame_contributor WHERE work_id IN ?", ids)
		db.Exec("DELETE FROM galgame_quiz_galgame WHERE work_id IN ?", ids)
		db.Exec("DELETE FROM galgame_view_daily WHERE entity_id IN ?", ids)
		db.Exec("DELETE FROM galgame_resource WHERE work_id IN ?", ids)
		db.Exec("DELETE FROM galgame WHERE id IN ?", ids)
		db.Exec("DELETE FROM user_purge_archive WHERE target_user_id = ?", ufUserPurged)
	}
	cleanup()
	t.Cleanup(cleanup)
	return db, run
}

func seedUnfold(t *testing.T, db *gorm.DB, run func(string, ...any)) {
	t.Helper()
	seed(t, db, mfDead, ptr("sfw"))
	run(`UPDATE galgame SET view = 7, created = '2021-01-01' WHERE id = ?`, mfDead)
	run(`INSERT INTO galgame (id, view, published, created, updated) VALUES (?, 2, false, '2024-01-01', now())`, mfSurvivor)
	run(`INSERT INTO galgame_like (work_id, user_id, updated) VALUES (?, ?, now()), (?, ?, now()), (?, ?, now())`,
		mfDead, ufUserBoth, mfSurvivor, ufUserBoth, mfDead, ufUserDead)
	run(`INSERT INTO galgame_contributor (work_id, user_id, first_at, last_at, revision_count, source)
		VALUES (?, ?, '2025-01-01', '2025-02-01', 3, 1), (?, ?, '2025-03-01', '2025-04-01', 5, 1)`,
		mfDead, ufUserBoth, mfSurvivor, ufUserBoth)
	run(`INSERT INTO galgame_quiz (id, user_id, category, type, difficulty, spoiler_level, question, description, content, explanation, hide_galgame, view, answer_count, correct_count, favorite_count, quality_sum, quality_count, comment_count, status_update_time, created, updated)
		VALUES (?, ?, 'plot', 'single', 3, 'none', 'q', '', '{"options":["a","b"],"answer":1}'::jsonb, '', false, 0, 0, 0, 0, 0, 0, 0, now(), now(), now())`,
		mfQuiz, mfAuthor)
	run(`INSERT INTO galgame_quiz_galgame (quiz_id, work_id) VALUES (?, ?)`, mfQuiz, mfDead)
	run(`INSERT INTO galgame_view_daily (entity_id, day, count) VALUES (?, ?, 3), (?, ?, 2)`, mfDead, ufDay, mfSurvivor, ufDay)
	run(`INSERT INTO feed_activity (type, source_id, user_id, work_id, content, link, created)
		VALUES ('GALGAME_COMMENT_CREATION', ?, ?, ?, 'c', '/galgame/2000350001', now())`, mfComment, mfAuthor, mfDead)
}

func TestUnfoldPutsEverythingBackAndKeepsWhatCameAfter(t *testing.T) {
	db, run := unfoldFixture(t)
	seedUnfold(t, db, run)
	repo := NewGalgameMergeRepository(db)

	counts, err := repo.Fold(mfDead, mfSurvivor)
	if err != nil {
		t.Fatalf("fold: %v", err)
	}
	if counts.FoldID == 0 {
		t.Fatal("fold wrote no ledger")
	}
	run(`INSERT INTO galgame_resource (id, work_id, user_id, platform, language, type, updated) VALUES (?, ?, 1, 'windows', 'ja', 'game', now())`,
		ufSurvivorRes, mfSurvivor)

	report, err := repo.Unfold(mfDead)
	if err != nil {
		t.Fatalf("unfold: %v", err)
	}
	if report == nil || len(report.Problems) != 0 {
		t.Fatalf("report = %+v, want a clean undo", report)
	}

	var dead struct {
		View      int
		Published bool
	}
	db.Raw(`SELECT view, published FROM galgame WHERE id = ?`, mfDead).Scan(&dead)
	if dead.View != 7 || !dead.Published {
		t.Errorf("old work row = %+v, want it back as it was", dead)
	}
	if n := scalar(t, db, `SELECT count(*) FROM galgame_resource WHERE work_id = ? AND id <> ?`, mfDead, ufSurvivorRes); n != 1 {
		t.Errorf("old work resources = %d, want its own one back", n)
	}
	if n := scalar(t, db, `SELECT work_id FROM galgame_resource WHERE id = ?`, ufSurvivorRes); n != mfSurvivor {
		t.Errorf("a resource posted on the survivor after the merge moved to %d", n)
	}
	if n := scalar(t, db, `SELECT count(*) FROM galgame_like WHERE work_id = ?`, mfDead); n != 2 {
		t.Errorf("old work likes = %d, want both (one moved, one dropped)", n)
	}
	if n := scalar(t, db, `SELECT count(*) FROM galgame_like WHERE work_id = ? AND user_id = ?`, mfSurvivor, ufUserBoth); n != 1 {
		t.Errorf("survivor lost its own like")
	}
	if n := scalar(t, db, `SELECT revision_count FROM galgame_contributor WHERE work_id = ? AND user_id = ?`, mfDead, ufUserBoth); n != 3 {
		t.Errorf("old work contributor revisions = %d, want 3", n)
	}
	var survivorContrib struct {
		RevisionCount int
		FirstAt       string
	}
	db.Raw(`SELECT revision_count, first_at::date::text AS first_at FROM galgame_contributor WHERE work_id = ? AND user_id = ?`,
		mfSurvivor, ufUserBoth).Scan(&survivorContrib)
	if survivorContrib.RevisionCount != 5 || survivorContrib.FirstAt != "2025-03-01" {
		t.Errorf("survivor contributor = %+v, want 5 revisions from 2025-03-01 again", survivorContrib)
	}
	if n := scalar(t, db, `SELECT count(*) FROM galgame_quiz_galgame WHERE quiz_id = ? AND work_id = ?`, mfQuiz, mfDead); n != 1 {
		t.Errorf("quiz link not back on the old work")
	}
	if n := scalar(t, db, `SELECT count FROM galgame_view_daily WHERE entity_id = ? AND day = ?`, mfDead, ufDay); n != 3 {
		t.Errorf("old work views = %d, want 3", n)
	}
	if n := scalar(t, db, `SELECT count FROM galgame_view_daily WHERE entity_id = ? AND day = ?`, mfSurvivor, ufDay); n != 2 {
		t.Errorf("survivor views = %d, want 2", n)
	}
	var feed struct {
		WorkID int
		Link   string
	}
	db.Raw(`SELECT work_id, link FROM feed_activity WHERE type = 'GALGAME_COMMENT_CREATION' AND source_id = ?`, mfComment).Scan(&feed)
	if feed.WorkID != mfDead || feed.Link != "/galgame/2000350001" {
		t.Errorf("feed row = %+v, want it back on the old work", feed)
	}
	var survivor struct {
		View          int
		Published     bool
		ResourceCount int
	}
	db.Raw(`SELECT view, published, resource_count FROM galgame WHERE id = ?`, mfSurvivor).Scan(&survivor)
	if survivor.View != 2 || !survivor.Published || survivor.ResourceCount != 1 {
		t.Errorf("survivor = %+v, want its own views, its own resource, and published because of it", survivor)
	}
	if n := scalar(t, db, `SELECT count(*) FROM galgame_merge_fold WHERE id = ? AND undone_at IS NOT NULL`, counts.FoldID); n != 1 {
		t.Errorf("fold not marked undone")
	}
	if again, err := repo.Unfold(mfDead); err != nil || again != nil {
		t.Errorf("a second undo = %+v %v, want nothing to do", again, err)
	}
}

func TestUnfoldFollowsAChainAndDeletesASurvivorItCreated(t *testing.T) {
	db, run := unfoldFixture(t)
	seed(t, db, mfDead, ptr("sfw"))
	repo := NewGalgameMergeRepository(db)

	if _, err := repo.Fold(mfDead, mfSurvivor); err != nil {
		t.Fatalf("fold: %v", err)
	}
	if _, err := repo.Fold(mfSurvivor, ufFolded); err != nil {
		t.Fatalf("second fold: %v", err)
	}
	if n := scalar(t, db, `SELECT count(*) FROM galgame_resource WHERE work_id = ?`, ufFolded); n != 1 {
		t.Fatalf("chain did not carry the resource to %d", ufFolded)
	}

	report, err := repo.Unfold(mfDead)
	if err != nil {
		t.Fatalf("unfold: %v", err)
	}
	if n := scalar(t, db, `SELECT count(*) FROM galgame_resource WHERE work_id = ?`, mfDead); n != 1 {
		t.Errorf("the resource did not come back from the end of the chain")
	}
	if len(report.Problems) != 1 || report.Problems[0].Reason != UnfoldSurvivorFolded {
		t.Errorf("problems = %+v, want the survivor row reported as folded further", report.Problems)
	}
	_ = run
}

func TestUnfoldDeletesASurvivorRowItCreatedAndRecordsAPurgedUser(t *testing.T) {
	db, run := unfoldFixture(t)
	seed(t, db, mfDead, ptr("sfw"))
	run(`INSERT INTO galgame (id, updated) VALUES (?, now())`, mfSurvivor)
	run(`INSERT INTO galgame_like (work_id, user_id, updated) VALUES (?, ?, now()), (?, ?, now())`,
		mfDead, ufUserPurged, mfSurvivor, ufUserPurged)
	repo := NewGalgameMergeRepository(db)
	if _, err := repo.Fold(mfDead, mfSurvivor); err != nil {
		t.Fatalf("fold: %v", err)
	}
	run(`INSERT INTO user_purge_archive (purge_id, operator_id, target_user_id, table_name, operation, row_pk, row_data)
		VALUES (gen_random_uuid(), 1, ?, 'galgame_like', 'delete', '{}', '{}')`, ufUserPurged)

	report, err := repo.Unfold(mfDead)
	if err != nil {
		t.Fatalf("unfold: %v", err)
	}
	if len(report.Problems) != 1 || report.Problems[0].Reason != UnfoldPurgedUser {
		t.Errorf("problems = %+v, want the purged user's like held back", report.Problems)
	}
	var stored []UnfoldProblem
	var raw string
	db.Raw(`SELECT undo_problems::text FROM galgame_merge_fold WHERE old_work_id = ?`, mfDead).Scan(&raw)
	if err := json.Unmarshal([]byte(raw), &stored); err != nil || len(stored) != 1 {
		t.Errorf("undo_problems = %s, want the problem recorded for the admin page", raw)
	}

	db2, run2 := unfoldFixture(t)
	seed(t, db2, mfDead, ptr("sfw"))
	repo2 := NewGalgameMergeRepository(db2)
	if _, err := repo2.Fold(mfDead, mfSurvivor); err != nil {
		t.Fatalf("fold: %v", err)
	}
	if _, err := repo2.Unfold(mfDead); err != nil {
		t.Fatalf("unfold: %v", err)
	}
	if n := scalar(t, db2, `SELECT count(*) FROM galgame WHERE id = ?`, mfSurvivor); n != 0 {
		t.Errorf("the survivor row the fold created outlived the undo")
	}
	_ = run2
}

func unfoldAll(t *testing.T, repo *GalgameMergeRepository, oldID int) []*UnfoldReport {
	t.Helper()
	var reports []*UnfoldReport
	for range 8 {
		r, err := repo.Unfold(oldID)
		if err != nil {
			t.Fatalf("unfold: %v", err)
		}
		if r == nil {
			return reports
		}
		reports = append(reports, r)
	}
	t.Fatal("unfold never ran out of folds")
	return nil
}

func TestUnfoldWhenTheOldWorkHadNoLocalRow(t *testing.T) {
	db, run := unfoldFixture(t)
	run(`INSERT INTO galgame (id, view, updated) VALUES (?, 4, now())`, mfSurvivor)
	run(`INSERT INTO galgame_quiz (id, user_id, category, type, difficulty, spoiler_level, question, description, content, explanation, hide_galgame, view, answer_count, correct_count, favorite_count, quality_sum, quality_count, comment_count, status_update_time, created, updated)
		VALUES (?, ?, 'plot', 'single', 3, 'none', 'q', '', '{"options":["a","b"],"answer":1}'::jsonb, '', false, 0, 0, 0, 0, 0, 0, 0, now(), now(), now())`,
		mfQuiz, mfAuthor)
	run(`INSERT INTO galgame_quiz_galgame (quiz_id, work_id) VALUES (?, ?)`, mfQuiz, mfDead)
	repo := NewGalgameMergeRepository(db)
	if _, err := repo.Fold(mfDead, mfSurvivor); err != nil {
		t.Fatalf("fold: %v", err)
	}

	reports := unfoldAll(t, repo, mfDead)
	if len(reports) != 1 || len(reports[0].Problems) != 0 {
		t.Fatalf("reports = %+v", reports)
	}
	if n := scalar(t, db, `SELECT count(*) FROM galgame_quiz_galgame WHERE quiz_id = ? AND work_id = ?`, mfQuiz, mfDead); n != 1 {
		t.Errorf("quiz link not back")
	}
	if n := scalar(t, db, `SELECT view FROM galgame WHERE id = ?`, mfSurvivor); n != 4 {
		t.Errorf("survivor view = %d, want its own 4", n)
	}
}

func TestUnfoldRecordsARowTheDatabaseRefusesAndCarriesOn(t *testing.T) {
	db, run := unfoldFixture(t)
	seed(t, db, mfDead, ptr("sfw"))
	run(`INSERT INTO galgame (id, updated) VALUES (?, now())`, mfSurvivor)
	run(`INSERT INTO galgame_quiz (id, user_id, category, type, difficulty, spoiler_level, question, description, content, explanation, hide_galgame, view, answer_count, correct_count, favorite_count, quality_sum, quality_count, comment_count, status_update_time, created, updated)
		VALUES (?, ?, 'plot', 'single', 3, 'none', 'q', '', '{"options":["a","b"],"answer":1}'::jsonb, '', false, 0, 0, 0, 0, 0, 0, 0, now(), now(), now())`,
		mfQuiz, mfAuthor)
	run(`INSERT INTO galgame_quiz_galgame (quiz_id, work_id) VALUES (?, ?), (?, ?)`, mfQuiz, mfDead, mfQuiz, mfSurvivor)
	repo := NewGalgameMergeRepository(db)
	if _, err := repo.Fold(mfDead, mfSurvivor); err != nil {
		t.Fatalf("fold: %v", err)
	}
	run(`DELETE FROM galgame_quiz WHERE id = ?`, mfQuiz)

	reports := unfoldAll(t, repo, mfDead)
	if len(reports) != 1 || len(reports[0].Problems) != 1 || reports[0].Problems[0].Reason != UnfoldGone {
		t.Fatalf("reports = %+v, want the deleted quiz's link recorded as gone", reports)
	}
	if n := scalar(t, db, `SELECT count(*) FROM galgame_resource WHERE work_id = ?`, mfDead); n != 1 {
		t.Errorf("the rest of the undo did not happen")
	}
}

func TestUnfoldClearsTheOldWorksFlagsOffASurvivorItCreated(t *testing.T) {
	db, run := unfoldFixture(t)
	seed(t, db, mfDead, ptr("sfw"))
	run(`UPDATE galgame SET resource_publish_banned = true, creator_user_id = ? WHERE id = ?`, mfAuthor, mfDead)
	repo := NewGalgameMergeRepository(db)
	if _, err := repo.Fold(mfDead, mfSurvivor); err != nil {
		t.Fatalf("fold: %v", err)
	}
	run(`INSERT INTO galgame_like (work_id, user_id, updated) VALUES (?, ?, now())`, mfSurvivor, ufUserDead)

	unfoldAll(t, repo, mfDead)
	var survivor struct {
		Published             bool
		ResourcePublishBanned bool
		CreatorUserID         *int
	}
	db.Raw(`SELECT published, resource_publish_banned, creator_user_id FROM galgame WHERE id = ?`, mfSurvivor).Scan(&survivor)
	if survivor.Published || survivor.ResourcePublishBanned || survivor.CreatorUserID != nil {
		t.Errorf("survivor kept the old work's flags: %+v", survivor)
	}
}

func TestUnfoldUndoesEveryFoldOfAWork(t *testing.T) {
	db, run := unfoldFixture(t)
	seed(t, db, mfDead, ptr("sfw"))
	repo := NewGalgameMergeRepository(db)
	if _, err := repo.Fold(mfDead, mfSurvivor); err != nil {
		t.Fatalf("fold: %v", err)
	}
	run(`INSERT INTO galgame (id, updated) VALUES (?, now())`, mfDead)
	run(`INSERT INTO galgame_like (work_id, user_id, updated) VALUES (?, ?, now())`, mfDead, ufUserDead)
	if _, err := repo.Fold(mfDead, mfSurvivor); err != nil {
		t.Fatalf("second fold: %v", err)
	}

	if reports := unfoldAll(t, repo, mfDead); len(reports) != 2 {
		t.Fatalf("undid %d folds, want 2", len(reports))
	}
	if n := scalar(t, db, `SELECT count(*) FROM galgame_resource WHERE work_id = ?`, mfDead); n != 1 {
		t.Errorf("the first fold's resource stayed on the survivor")
	}
	if n := scalar(t, db, `SELECT count(*) FROM galgame_like WHERE work_id = ?`, mfDead); n != 1 {
		t.Errorf("the second fold's like stayed on the survivor")
	}
	if n := scalar(t, db, `SELECT count(*) FROM galgame WHERE id = ? AND published`, mfDead); n != 1 {
		t.Errorf("the old work lost published when its snapshot met the second fold's stub")
	}
}

func TestUnfoldFoldsTheSnapshotIntoAStubTheRevivedWorkGot(t *testing.T) {
	db, run := unfoldFixture(t)
	seed(t, db, mfDead, ptr("sfw"))
	run(`UPDATE galgame SET view = 9 WHERE id = ?`, mfDead)
	repo := NewGalgameMergeRepository(db)
	if _, err := repo.Fold(mfDead, mfSurvivor); err != nil {
		t.Fatalf("fold: %v", err)
	}
	run(`INSERT INTO galgame (id, view, updated) VALUES (?, 1, now())`, mfDead)

	unfoldAll(t, repo, mfDead)
	var dead struct {
		View      int
		Published bool
	}
	db.Raw(`SELECT view, published FROM galgame WHERE id = ?`, mfDead).Scan(&dead)
	if dead.View != 10 || !dead.Published {
		t.Errorf("old work = %+v, want the stub's view plus the snapshot's, and published", dead)
	}
}
