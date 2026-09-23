package renumber

import (
	"strings"
	"testing"

	"kun-galgame-api/internal/testdb"

	"gorm.io/gorm"
)

const (
	idA     int64 = 800000923
	idB     int64 = 800000922
	idWin   int64 = 800000628
	idLose  int64 = 800000739
	idP     int64 = 800000115
	idQ     int64 = 800000107
	idWork  int64 = 800000082
	idGhost int64 = 800000500
	uidLike int64 = 800000001
	uidColl int64 = 800000002
	uidCon  int64 = 800000003
	uidQuiz int64 = 800000004
)

const applyTSV = "" +
	"800000923\t800000922\tclaim\n" +
	"800000922\t800000921\tclaim\n" +
	"800000739\t800000628\tcurated\n" +
	"800000115\t800000082\tcurated\n" +
	"800000107\t800000082\tcurated\n" +
	"800000500\t800000628\tclaim\n" +
	"800000600\t800000650\tcurated\n" +
	"800000601\t800000650\tcurated\n"

func TestApplyRenumber(t *testing.T) {
	db := testdb.Open(t)
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	resID, msgID, topicID, quizID := seedApply(t, tx)

	m, err := LoadMap(strings.NewReader(applyTSV))
	if err != nil {
		t.Fatal(err)
	}
	rep, err := Apply(tx, m, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Folds) != 2 {
		t.Fatalf("folds=%d, want 2", len(rep.Folds))
	}

	assertMissing(t, tx, "galgame", "id", idA, idLose, idP, idQ)
	assertPresent(t, tx, "galgame", "id", 800000922, 800000921, idWin, idWork)

	if got := scalar(t, tx, "SELECT CASE WHEN published THEN 1 ELSE 0 END FROM galgame WHERE id = ?", 800000922); got != 1 {
		t.Fatalf("shifted A should still be published, got %d", got)
	}

	if n := scalar(t, tx, "SELECT COUNT(*) FROM galgame_like WHERE user_id = ? AND galgame_id = ?", uidLike, idWin); n != 1 {
		t.Fatalf("likes on survivor=%d, want 1", n)
	}
	if n := scalar(t, tx, "SELECT COUNT(*) FROM galgame_like WHERE user_id = ? AND galgame_id = ?", uidLike, idLose); n != 0 {
		t.Fatalf("likes on loser=%d", n)
	}

	if n := scalar(t, tx, "SELECT COUNT(*) FROM galgame_collection_item WHERE collection_id IN (SELECT id FROM galgame_collection WHERE user_id = ?) AND galgame_id = ?", uidColl, idWork); n != 1 {
		t.Fatalf("collection items at work=%d, want 1", n)
	}
	if n := scalar(t, tx, "SELECT COUNT(*) FROM galgame_merge_discarded WHERE table_name = 'galgame_collection_item' AND old_gid = ? AND new_gid = ?", idQ, idWork); n != 1 {
		t.Fatalf("archived collection item=%d", n)
	}

	if n := scalar(t, tx, `SELECT "count" FROM galgame_view_daily WHERE entity_id = ? AND day = '2026-09-01'`, idWin); n != 10 {
		t.Fatalf("view_daily count=%d, want 10", n)
	}
	if n := scalar(t, tx, `SELECT "count" FROM galgame_view_daily WHERE entity_id = 800000650 AND day = '2026-09-01'`); n != 11 {
		t.Fatalf("two legacy numbers of one work on one day: count=%d, want 11", n)
	}

	if n := scalar(t, tx, "SELECT galgame_id FROM galgame_resource WHERE id = ?", resID); n != 800000922 {
		t.Fatalf("resource galgame_id=%d", n)
	}
	if n := scalar(t, tx, "SELECT source_id FROM feed_activity WHERE type = 'GALGAME_CREATION' AND galgame_id = 800000922"); n != 800000922 {
		t.Fatalf("creation source_id=%d", n)
	}
	var link string
	if err := tx.Raw("SELECT link FROM feed_activity WHERE type = 'GALGAME_CREATION' AND source_id = 800000922").Scan(&link).Error; err != nil {
		t.Fatal(err)
	}
	if link != "/galgame/800000922" {
		t.Fatalf("creation link=%q", link)
	}

	var msgLink string
	if err := tx.Raw("SELECT link FROM message WHERE id = ?", msgID).Scan(&msgLink).Error; err != nil {
		t.Fatal(err)
	}
	if msgLink != "/galgame/800000922?comment=77" {
		t.Fatalf("message.link=%q", msgLink)
	}
	var body string
	if err := tx.Raw("SELECT content FROM topic WHERE id = ?", topicID).Scan(&body).Error; err != nil {
		t.Fatal(err)
	}
	wantBody := "see https://www.kungal.com/galgame/800000922 and https://mikugame.icu/galgame/800000923"
	if body != wantBody {
		t.Fatalf("topic.content=%q", body)
	}

	if n := scalar(t, tx, "SELECT COUNT(*) FROM galgame_quiz_galgame WHERE quiz_id = ? AND galgame_id = ?", quizID, idWork); n != 1 {
		t.Fatalf("quiz_galgame rows at work=%d", n)
	}
	if n := scalar(t, tx, "SELECT COUNT(*) FROM galgame_merge_discarded WHERE table_name = 'galgame_quiz_galgame' AND old_gid = ? AND new_gid = ?", idGhost, idWin); n != 1 {
		t.Fatalf("archived quiz_galgame=%d", n)
	}
	if n := scalar(t, tx, "SELECT revision_count FROM galgame_contributor WHERE galgame_id = ? AND user_id = ?", idWin, uidCon); n != 5 {
		t.Fatalf("contributor revision_count=%d", n)
	}

	st := rep.link("message.link")
	if st == nil || st.Rewritten != 1 {
		t.Fatalf("message.link rewritten=%v", st)
	}
}

func TestApplyRollbackLeavesRow(t *testing.T) {
	db := testdb.Open(t)
	const id, dest int64 = 800000801, 800000800
	cleanup := func() {
		db.Exec("DELETE FROM galgame WHERE id IN (?, ?)", id, dest)
	}
	cleanup()
	t.Cleanup(cleanup)
	if err := db.Exec("INSERT INTO galgame (id, updated) VALUES (?, now())", id).Error; err != nil {
		t.Fatal(err)
	}

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	m, err := LoadMap(strings.NewReader("800000801\t800000800\tclaim\n"))
	if err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if _, err := Apply(tx, m, Options{}); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatal(err)
	}
	if n := scalar(t, db, "SELECT id FROM galgame WHERE id = ?", id); n != id {
		t.Fatalf("dry-run rollback changed id to %d", n)
	}
}

func seedApply(t *testing.T, tx *gorm.DB) (resID, msgID, topicID, quizID int64) {
	t.Helper()
	mustExec(t, tx, `INSERT INTO galgame (id, published, created, updated) VALUES
		(?, true,  '2020-01-01', now()),
		(?, false, '2020-02-01', now()),
		(?, true,  '2020-03-01', now()),
		(?, false, '2020-04-01', now()),
		(?, true,  '2020-01-01', now()),
		(?, false, '2021-01-01', now())`,
		idA, idB, idWin, idLose, idP, idQ)

	mustExec(t, tx, `INSERT INTO galgame_like (galgame_id, user_id, updated) VALUES (?, ?, now()), (?, ?, now())`,
		idLose, uidLike, idWin, uidLike)

	resID = insertID(t, tx, `INSERT INTO galgame_resource (galgame_id, user_id, updated) VALUES (?, ?, now()) RETURNING id`, idA, uidLike)

	mustExec(t, tx, `INSERT INTO galgame_view_daily (entity_id, day, count) VALUES
		(?, '2026-09-01', 3),
		(?, '2026-09-01', 5),
		(?, '2026-09-01', 2),
		(800000600, '2026-09-01', 4),
		(800000601, '2026-09-01', 7)`, idLose, idWin, idGhost)

	colID := insertID(t, tx, `INSERT INTO galgame_collection (user_id, name, description, visibility, is_default, updated)
		VALUES (?, 'g0a', '', 'public', false, now()) RETURNING id`, uidColl)
	mustExec(t, tx, `INSERT INTO galgame_collection_item (collection_id, galgame_id, user_id, created, updated) VALUES
		(?, ?, ?, '2020-01-01', now()),
		(?, ?, ?, '2021-01-01', now())`,
		colID, idP, uidColl, colID, idQ, uidColl)

	quizID = insertID(t, tx, `INSERT INTO galgame_quiz (user_id, type, question, spoiler_level, content)
		VALUES (?, 'single', 'q', 'none', '{}'::jsonb) RETURNING id`, uidQuiz)
	mustExec(t, tx, `INSERT INTO galgame_quiz_galgame (quiz_id, galgame_id) VALUES (?, ?), (?, ?)`,
		quizID, idP, quizID, idQ)
	quizHop := insertID(t, tx, `INSERT INTO galgame_quiz (user_id, type, question, spoiler_level, content)
		VALUES (?, 'single', 'q2', 'none', '{}'::jsonb) RETURNING id`, uidQuiz)
	mustExec(t, tx, `INSERT INTO galgame_quiz_galgame (quiz_id, galgame_id) VALUES (?, ?), (?, ?)`,
		quizHop, idGhost, quizHop, idWin)

	mustExec(t, tx, `INSERT INTO galgame_contributor (galgame_id, user_id, first_at, last_at, revision_count, source) VALUES
		(?, ?, '2020-01-01', '2020-06-01', 2, 1),
		(?, ?, '2019-01-01', '2021-01-01', 3, 1)`,
		idGhost, uidCon, idWin, uidCon)

	msgID = insertID(t, tx, `INSERT INTO message (link, type, sender_id, receiver_id, updated)
		VALUES (?, 'liked', ?, ?, now()) RETURNING id`,
		"/galgame/800000923?comment=77", uidLike, uidLike+1)

	topicID = insertID(t, tx, `INSERT INTO topic (title, content, category, user_id, updated)
		VALUES ('g0a', ?, 'galgame', ?, now()) RETURNING id`,
		"see https://www.kungal.com/galgame/800000923 and https://mikugame.icu/galgame/800000923", uidLike)
	return
}

func mustExec(t *testing.T, tx *gorm.DB, q string, args ...any) {
	t.Helper()
	if err := tx.Exec(q, args...).Error; err != nil {
		t.Fatalf("%v\n%s", err, q)
	}
}

func insertID(t *testing.T, tx *gorm.DB, q string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := tx.Raw(q, args...).Scan(&id).Error; err != nil {
		t.Fatalf("%v\n%s", err, q)
	}
	if id == 0 {
		t.Fatalf("RETURNING id was 0\n%s", q)
	}
	return id
}

func scalar(t *testing.T, tx *gorm.DB, q string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := tx.Raw(q, args...).Scan(&n).Error; err != nil {
		t.Fatalf("%v\n%s", err, q)
	}
	return n
}

func assertMissing(t *testing.T, tx *gorm.DB, table, col string, ids ...int64) {
	t.Helper()
	for _, id := range ids {
		if n := scalar(t, tx, "SELECT COUNT(*) FROM "+table+" WHERE "+col+" = ?", id); n != 0 {
			t.Fatalf("%s.%s=%d still present", table, col, id)
		}
	}
}

func assertPresent(t *testing.T, tx *gorm.DB, table, col string, ids ...int64) {
	t.Helper()
	for _, id := range ids {
		if n := scalar(t, tx, "SELECT COUNT(*) FROM "+table+" WHERE "+col+" = ?", id); n != 1 {
			t.Fatalf("%s.%s=%d count=%d, want 1", table, col, id, n)
		}
	}
}
