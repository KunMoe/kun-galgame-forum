package app

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func (f *writeFix) recallWipeText(t *testing.T, id int) string {
	t.Helper()
	var s string
	if err := f.db.Raw(`SELECT content FROM chat_message WHERE id = ?`, id).Scan(&s).Error; err != nil {
		t.Fatal(err)
	}
	return s
}

func (f *writeFix) recallWipeXmin(t *testing.T, id int) string {
	t.Helper()
	var s string
	if err := f.db.Raw(`SELECT xmin::text FROM chat_message WHERE id = ?`, id).Scan(&s).Error; err != nil {
		t.Fatal(err)
	}
	return s
}

func (f *writeFix) recallWipeTime(t *testing.T, id int) time.Time {
	t.Helper()
	var ts time.Time
	if err := f.db.Raw(`SELECT recall_time FROM chat_message WHERE id = ?`, id).Scan(&ts).Error; err != nil {
		t.Fatal(err)
	}
	return ts
}

func assertRoomBobKeptTexts(t *testing.T, f *writeFix) {
	t.Helper()
	want := []struct {
		id   int
		text string
	}{
		{mMsgBob1, "hello from bob"},
		{mMsgAlice1, "hello from alice"},
		{mMsgBob2, "second from bob"},
		{mMsgAlice2, "second from alice"},
		{mMsgBobPeer, "bob peer message"},
	}
	for _, row := range want {
		if got := f.recallWipeText(t, row.id); got != row.text {
			t.Fatalf("message %d content %q want %q", row.id, got, row.text)
		}
	}
}

func migration131Path(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "migrations", "131_chat_message_recall_wipe.up.sql")
}

func execMigration131(t *testing.T, f *writeFix) {
	t.Helper()
	raw, err := os.ReadFile(migration131Path(t))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := f.db.DB()
	if err != nil {
		t.Fatal(err)
	}
	// cmd/migrate Execs the whole file; splitting on ';' tested the file's shape instead of the effect.
	if _, err := sqlDB.Exec(string(raw)); err != nil {
		t.Fatalf("run 131: %v", err)
	}
}

func TestV1Conversations_R1_RecallWipesStoredText(t *testing.T) {
	f := newMessageFix(t)

	path := conversationsPath + "/" + strconv.Itoa(w3UserBob) + "/messages/" + strconv.Itoa(mMsgAliceRec)
	resp, raw := f.doJSON(t, http.MethodPatch, path, "sess-alice",
		"/me/conversations/{user_id}/messages/{message_id}", "", nil,
		map[string]any{"state": "recalled"})
	body := problemMap(t, raw)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("recall own %d %+v", resp.StatusCode, body)
	}

	if got := f.recallWipeText(t, mMsgAliceRec); got != "" {
		t.Fatalf("recalled content %q want empty", got)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM chat_message WHERE id = ? AND is_recall = true`, mMsgAliceRec); n != 1 {
		t.Fatalf("is_recall rows=%d", n)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM chat_message WHERE id = ? AND recall_time IS NOT NULL`, mMsgAliceRec); n != 1 {
		t.Fatalf("recall_time is null")
	}
	assertRoomBobKeptTexts(t, f)
}

func TestV1Conversations_R3_ReplayedRecallNoWrite(t *testing.T) {
	f := newMessageFix(t)

	path := conversationsPath + "/" + strconv.Itoa(w3UserBob) + "/messages/" + strconv.Itoa(mMsgAliceRec)
	resp, raw := f.doJSON(t, http.MethodPatch, path, "sess-alice",
		"/me/conversations/{user_id}/messages/{message_id}", "", nil,
		map[string]any{"state": "recalled"})
	body := problemMap(t, raw)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("recall own %d %+v", resp.StatusCode, body)
	}

	xmin := f.recallWipeXmin(t, mMsgAliceRec)
	recalledAt := f.recallWipeTime(t, mMsgAliceRec)

	resp, raw = f.doJSON(t, http.MethodPatch, path, "sess-alice",
		"/me/conversations/{user_id}/messages/{message_id}", "", nil,
		map[string]any{"state": "recalled"})
	again := problemMap(t, raw)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("recall again %d %+v", resp.StatusCode, again)
	}
	if again["state"] != "recalled" {
		t.Fatalf("state %v", again["state"])
	}
	assertEmptyDocument(t, again["content"])
	if got := f.recallWipeXmin(t, mMsgAliceRec); got != xmin {
		t.Fatalf("replay changed xmin: %s vs %s", got, xmin)
	}
	if got := f.recallWipeTime(t, mMsgAliceRec); !got.Equal(recalledAt) {
		t.Fatalf("replay changed recall_time: %v vs %v", got, recalledAt)
	}
}

func TestV1Migration131_RecallWipe(t *testing.T) {
	f := newMessageFix(t)
	fixed := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	if err := f.db.Exec(
		`UPDATE chat_message SET is_recall = true, recall_time = ? WHERE id = ?`,
		fixed, mMsgAliceRec,
	).Error; err != nil {
		t.Fatal(err)
	}

	execMigration131(t, f)

	if n := f.scalar(t, `SELECT COUNT(*) FROM chat_message WHERE id = ?`, mMsgAliceRec); n != 1 {
		t.Fatalf("recalled row missing after 131: count=%d", n)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM chat_message WHERE id = ? AND is_recall = true`, mMsgAliceRec); n != 1 {
		t.Fatalf("is_recall rows=%d", n)
	}
	if got := f.recallWipeText(t, mMsgAliceRec); got != "" {
		t.Fatalf("recalled content %q want empty", got)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM chat_message WHERE id = ? AND recall_time = ?`, mMsgAliceRec, fixed); n != 1 {
		t.Fatalf("recall_time drifted from %v", fixed)
	}
	assertRoomBobKeptTexts(t, f)

	ids := []int{mMsgBob1, mMsgAlice1, mMsgBob2, mMsgAlice2, mMsgBobPeer, mMsgAliceRec}
	xmin := make([]string, len(ids))
	for i, id := range ids {
		xmin[i] = f.recallWipeXmin(t, id)
	}

	execMigration131(t, f)

	for i, id := range ids {
		if got := f.recallWipeXmin(t, id); got != xmin[i] {
			t.Fatalf("second 131 changed xmin of %d: %s vs %s", id, got, xmin[i])
		}
	}
}
