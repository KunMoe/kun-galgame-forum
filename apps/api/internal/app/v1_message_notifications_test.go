package app

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"
)

const (
	mNotifOlder = 960000110
	mNotifTieA  = 960000111
	mNotifTieB  = 960000112
	mNotifTieC  = 960000113
	mNotifNewer = 960000114
	mNotifLike2 = 960000115

	mNotifMutedLiked   = 960000120
	mNotifMutedReplied = 960000121

	mNotifBanA  = 960000130
	mNotifBanB  = 960000131
	mNotifFillA = 960000132
	mNotifFillB = 960000133
	mNotifFillC = 960000134

	mNotifLow  = 960000140
	mNotifHigh = 960000141

	mNotifBobOwn   = 960000150
	mNotifAliceDel = 960000151
	mNotifMirror   = 960000152
	mNotifUnknown  = 960000160

	mNotifMin = 960000100
	mNotifMax = 960000199
	mRoomMin  = 960000200
	mRoomMax  = 960000299
	mChatMin  = 960000300
	mChatMax  = 960000399
)

const notificationsPath = "/api/v1/me/notifications"

func newMessageFix(t *testing.T) *writeFix {
	t.Helper()
	f := newWriteFix(t, nil)
	f.alice(t)
	f.seedMessages(t)
	return f
}

func (f *writeFix) seedMessages(t *testing.T) {
	t.Helper()
	f.cleanupMessageDomain(t)
	t.Cleanup(func() { f.cleanupMessageDomain(t) })

	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("seed messages: %v\n%s", err, q)
		}
	}
	insertNotif := func(id, sender, receiver int, typ, status, content string, at time.Time) {
		t.Helper()
		run(`INSERT INTO message (id, content, link, status, type, sender_id, receiver_id, created, updated, item_count, actor_count)
			VALUES (?, ?, '/topic/1', ?, ?, ?, ?, ?, ?, 1, 1)`,
			id, content, status, typ, sender, receiver, at, at)
	}

	tie := base.Add(10 * time.Minute)
	insertNotif(mNotifOlder, w3UserBob, w3UserAlice, "liked", "read", "older liked", base)
	insertNotif(mNotifTieA, w3UserBob, w3UserAlice, "replied", "unread", "tie a", tie)
	insertNotif(mNotifTieB, w3UserBob, w3UserAlice, "replied", "unread", "tie b", tie)
	insertNotif(mNotifTieC, w3UserBob, w3UserAlice, "replied", "unread", "tie c", tie)
	insertNotif(mNotifNewer, w3UserBob, w3UserAlice, "commented", "unread", "newer commented", base.Add(20*time.Minute))
	insertNotif(mNotifLike2, w3UserBob, w3UserAlice, "liked", "unread", "second liked", base.Add(5*time.Minute))

	insertNotif(mNotifMutedLiked, w3UserBob, w3UserAlice, "liked", "unread", "muted liked", base.Add(2*time.Hour))
	insertNotif(mNotifMutedReplied, w3UserBob, w3UserAlice, "replied", "unread", "unmuted replied", base.Add(2*time.Hour+time.Minute))

	banAt := base.Add(10 * time.Hour)
	insertNotif(mNotifBanA, w3UserBanned, w3UserAlice, "replied", "unread", "banned a", banAt)
	insertNotif(mNotifBanB, w3UserBanned, w3UserAlice, "replied", "unread", "banned b", banAt.Add(-time.Second))
	insertNotif(mNotifFillA, w3UserBob, w3UserAlice, "replied", "unread", "fill a", banAt.Add(-2*time.Second))
	insertNotif(mNotifFillB, w3UserBob, w3UserAlice, "replied", "unread", "fill b", banAt.Add(-3*time.Second))
	insertNotif(mNotifFillC, w3UserBob, w3UserAlice, "replied", "unread", "fill c", banAt.Add(-4*time.Second))

	insertNotif(mNotifLow, w3UserBob, w3UserAlice, "mentioned", "unread", "low", base.Add(4*time.Hour))
	insertNotif(mNotifHigh, w3UserBob, w3UserAlice, "mentioned", "unread", "high", base.Add(4*time.Hour+time.Minute))

	insertNotif(mNotifBobOwn, w3UserAlice, w3UserBob, "liked", "unread", "bob's", base)
	insertNotif(mNotifAliceDel, w3UserBob, w3UserAlice, "commented", "unread", "delete me", base.Add(5*time.Hour))
	run(`INSERT INTO message (id, content, link, status, type, sender_id, receiver_id, created, updated, item_count, actor_count, community_notification_id)
		VALUES (?, 'mirror', '/topic/2', 'unread', 'replied', ?, ?, ?, ?, 1, 1, ?)`,
		mNotifMirror, w3UserBob, w3UserAlice, base.Add(6*time.Hour), base.Add(6*time.Hour), int64(960000001))
	insertNotif(mNotifUnknown, w3UserBob, w3UserAlice, "admin", "unread", "unknown type", base.Add(7*time.Hour))

	f.seedConversations(t, run, base)
}

func (f *writeFix) cleanupMessageDomain(t *testing.T) {
	t.Helper()
	u0, u1 := 930000001, 930000999
	_ = f.db.Exec(`DELETE FROM chat_message_read_by WHERE user_id BETWEEN ? AND ? OR chat_message_id BETWEEN ? AND ?`,
		u0, u1, mChatMin, mChatMax).Error
	_ = f.db.Exec(`DELETE FROM chat_message WHERE id BETWEEN ? AND ? OR sender_id BETWEEN ? AND ? OR receiver_id BETWEEN ? AND ?
		OR chat_room_id BETWEEN ? AND ? OR chat_room_id IN (SELECT id FROM chat_room WHERE name LIKE '930000%')`,
		mChatMin, mChatMax, u0, u1, u0, u1, mRoomMin, mRoomMax).Error
	_ = f.db.Exec(`DELETE FROM chat_room_participant WHERE user_id BETWEEN ? AND ? OR chat_room_id BETWEEN ? AND ?
		OR chat_room_id IN (SELECT id FROM chat_room WHERE name LIKE '930000%')`,
		u0, u1, mRoomMin, mRoomMax).Error
	_ = f.db.Exec(`DELETE FROM chat_room WHERE id BETWEEN ? AND ? OR name LIKE '930000%'`, mRoomMin, mRoomMax).Error
	_ = f.db.Exec(`DELETE FROM message WHERE id BETWEEN ? AND ? OR receiver_id BETWEEN ? AND ? OR sender_id BETWEEN ? AND ?`,
		mNotifMin, mNotifMax, u0, u1, u0, u1).Error
}

func (f *writeFix) notifList(t *testing.T, session, query string) (*http.Response, map[string]any) {
	t.Helper()
	resp, body := f.doJSON(t, http.MethodGet, notificationsPath+query, session, "/me/notifications", "", nil, nil)
	return resp, problemMap(t, body)
}

func listIDs(t *testing.T, body map[string]any) []string {
	t.Helper()
	raw, _ := body["items"].([]any)
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		row, _ := item.(map[string]any)
		out = append(out, strID(row["id"]))
	}
	return out
}

func (f *writeFix) muteAlice(t *testing.T, dbTypes string) {
	t.Helper()
	if err := f.db.Exec(
		`UPDATE kungal_user_state SET muted_notification_types = ?::jsonb WHERE user_id = ?`,
		dbTypes, w3UserAlice,
	).Error; err != nil {
		t.Fatal(err)
	}
}

func TestV1Notifications_M1_TieBreakTraversal(t *testing.T) {
	f := newMessageFix(t)
	// With migration 130's (receiver_id, created DESC, id DESC) index the planner
	// scans it and ties come out in id order anyway, so dropping the id
	// tie-breaker from ORDER BY survived this test. Without the index the plan
	// sorts, and only the tie-breaker orders the ties.
	if err := f.db.Exec(`DROP INDEX IF EXISTS idx_message_receiver_created_id`).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.db.Exec(`CREATE INDEX IF NOT EXISTS idx_message_receiver_created_id
			ON message (receiver_id, created DESC, id DESC)`).Error; err != nil {
			t.Errorf("restore index: %v", err)
		}
	})

	var wantIDs []int
	if err := f.db.Raw(
		`SELECT id FROM message WHERE receiver_id = ? AND type <> 'admin' AND sender_id <> ?
		 ORDER BY created DESC, id DESC`,
		w3UserAlice, w3UserBanned,
	).Scan(&wantIDs).Error; err != nil {
		t.Fatal(err)
	}
	want := make([]string, len(wantIDs))
	for i, id := range wantIDs {
		want[i] = strconv.Itoa(id)
	}

	resp, body := f.notifList(t, "sess-alice", "")
	if resp.StatusCode != http.StatusOK || body["object"] != "list" {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	full := listIDs(t, body)
	if fmt.Sprint(full) != fmt.Sprint(want) {
		t.Fatalf("full order %v\nwant %v", full, want)
	}
	first, _ := body["items"].([]any)[0].(map[string]any)
	if first["notification_type"] == nil {
		t.Fatalf("notification_type missing: %+v", first)
	}
	if _, ok := first["type"]; ok {
		t.Fatalf("legacy type field present: %+v", first)
	}

	var walked []string
	cursor := ""
	for page := 0; page < 20; page++ {
		q := "?limit=2"
		if cursor != "" {
			q += "&cursor=" + cursor
		}
		resp, body := f.notifList(t, "sess-alice", q)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("page %d: %d %+v", page, resp.StatusCode, body)
		}
		ids := listIDs(t, body)
		walked = append(walked, ids...)
		next, ok := body["next_cursor"].(string)
		if !ok {
			break
		}
		cursor = next
	}
	if fmt.Sprint(walked) != fmt.Sprint(want) {
		t.Fatalf("cursor traversal dropped or repeated a row\n got %v\nwant %v", walked, want)
	}
	seen := map[string]bool{}
	for _, id := range walked {
		if seen[id] {
			t.Fatalf("duplicate id %s in traversal %v", id, walked)
		}
		seen[id] = true
	}
	if len(walked) != len(want) {
		t.Fatalf("traversal length %d want %d", len(walked), len(want))
	}
}

func TestV1Notifications_M2_MutedPartition(t *testing.T) {
	f := newMessageFix(t)
	f.muteAlice(t, `["liked"]`)

	_, body := f.notifList(t, "sess-alice", "?is_muted=false")
	for _, id := range listIDs(t, body) {
		if id == strconv.Itoa(mNotifMutedLiked) || id == strconv.Itoa(mNotifOlder) || id == strconv.Itoa(mNotifLike2) {
			t.Fatalf("liked row %s appeared in the unmuted partition: %v", id, listIDs(t, body))
		}
	}
	foundReplied := false
	for _, id := range listIDs(t, body) {
		if id == strconv.Itoa(mNotifMutedReplied) {
			foundReplied = true
		}
	}
	if !foundReplied {
		t.Fatalf("unmuted replied row missing from default partition: %v", listIDs(t, body))
	}

	_, muted := f.notifList(t, "sess-alice", "?is_muted=true")
	mutedIDs := listIDs(t, muted)
	if len(mutedIDs) == 0 {
		t.Fatal("muted partition is empty")
	}
	for _, id := range mutedIDs {
		switch id {
		case strconv.Itoa(mNotifMutedLiked), strconv.Itoa(mNotifOlder), strconv.Itoa(mNotifLike2):
		default:
			t.Fatalf("unmuted type in muted partition: %s in %v", id, mutedIDs)
		}
	}

	_, empty := f.notifList(t, "sess-alice", "?is_muted=false&notification_type=liked")
	if ids := listIDs(t, empty); len(ids) != 0 {
		t.Fatalf("unmuted ∩ liked must be empty, got %v", ids)
	}
}

func TestV1Notifications_M3_CursorFingerprintIncludesType(t *testing.T) {
	f := newMessageFix(t)

	resp, body := f.notifList(t, "sess-alice", "?notification_type=liked&limit=1")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("liked page %d %+v", resp.StatusCode, body)
	}
	cursor, ok := body["next_cursor"].(string)
	if !ok {
		t.Fatalf("need a liked cursor: %+v", body)
	}
	resp, body = f.notifList(t, "sess-alice", "?notification_type=replied&limit=1&cursor="+cursor)
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_CURSOR" {
		t.Fatalf("liked cursor with notification_type=replied %d %+v", resp.StatusCode, body)
	}
}

func TestV1Notifications_M4_BannedActorPageRefill(t *testing.T) {
	f := newMessageFix(t)

	resp, body := f.notifList(t, "sess-alice", "?limit=2")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	ids := listIDs(t, body)
	if len(ids) < 2 && body["next_cursor"] == nil {
		t.Fatalf("a page of banned actors must refill or keep a cursor, got items %v", ids)
	}
	if len(ids) != 2 {
		t.Fatalf("expected a full page of renderable rows, got %v next=%v", ids, body["next_cursor"])
	}
	if ids[0] != strconv.Itoa(mNotifFillA) || ids[1] != strconv.Itoa(mNotifFillB) {
		t.Fatalf("refilled page %v want [%d %d]", ids, mNotifFillA, mNotifFillB)
	}
}

func TestV1Notifications_M5_ReadMarkerRespectsMuted(t *testing.T) {
	f := newMessageFix(t)
	f.muteAlice(t, `["liked"]`)

	mutedUnread := f.scalar(t, `SELECT COUNT(*) FROM message WHERE receiver_id = ? AND status = 'unread' AND type = 'liked'`, w3UserAlice)
	resp, raw := f.doJSON(t, http.MethodPut, notificationsPath+"/read-marker", "sess-alice",
		"/me/notifications/read-marker", "", nil,
		map[string]any{"up_to_id": strconv.Itoa(mNotifHigh), "is_muted": false})
	body := problemMap(t, raw)
	if resp.StatusCode != http.StatusOK || body["object"] != "notification_read_marker" {
		t.Fatalf("mark %d %+v", resp.StatusCode, body)
	}
	after := f.scalar(t, `SELECT COUNT(*) FROM message WHERE receiver_id = ? AND status = 'unread' AND type = 'liked'`, w3UserAlice)
	if after != mutedUnread {
		t.Fatalf("muted partition unread changed: %d -> %d", mutedUnread, after)
	}
}

func TestV1Notifications_M6_ReadMarkerRespectsUpToID(t *testing.T) {
	f := newMessageFix(t)

	resp, raw := f.doJSON(t, http.MethodPut, notificationsPath+"/read-marker", "sess-alice",
		"/me/notifications/read-marker", "", nil,
		map[string]any{"up_to_id": strconv.Itoa(mNotifLow)})
	body := problemMap(t, raw)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("mark %d %+v", resp.StatusCode, body)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM message WHERE id = ? AND status = 'unread'`, mNotifHigh); n != 1 {
		t.Fatalf("id > up_to_id must stay unread, status rows=%d", n)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM message WHERE id = ? AND status = 'unread'`, mNotifLow); n != 0 {
		t.Fatalf("up_to_id itself must be marked, unread rows=%d", n)
	}
}

func TestV1Notifications_M7_DeleteForeignIs404(t *testing.T) {
	f := newMessageFix(t)

	path := notificationsPath + "/" + strconv.Itoa(mNotifBobOwn)
	resp, raw := f.doJSON(t, http.MethodDelete, path, "sess-alice", "/me/notifications/{notification_id}", "", nil, nil)
	body := problemMap(t, raw)
	if resp.StatusCode != http.StatusNotFound || body["code"] != "NOT_FOUND" {
		t.Fatalf("delete bob's notification %d %+v", resp.StatusCode, body)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM message WHERE id = ?`, mNotifBobOwn); n != 1 {
		t.Fatalf("bob's notification is gone: %d rows", n)
	}
}

func TestV1NotificationsDeleteOwn(t *testing.T) {
	f := newMessageFix(t)

	path := notificationsPath + "/" + strconv.Itoa(mNotifAliceDel)
	resp, raw := f.doJSON(t, http.MethodDelete, path, "sess-alice", "/me/notifications/{notification_id}", "", nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete own %d %s", resp.StatusCode, raw)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM message WHERE id = ?`, mNotifAliceDel); n != 0 {
		t.Fatalf("still there: %d", n)
	}
}

func TestV1NotificationsSummary(t *testing.T) {
	f := newMessageFix(t)
	f.muteAlice(t, `["liked"]`)

	unmuted := f.scalar(t, `SELECT COUNT(*) FROM message WHERE receiver_id = ? AND status = 'unread' AND type <> 'liked'`, w3UserAlice)
	muted := f.scalar(t, `SELECT COUNT(*) FROM message WHERE receiver_id = ? AND status = 'unread' AND type = 'liked'`, w3UserAlice)

	resp, raw := f.doJSON(t, http.MethodGet, notificationsPath+"/summary", "sess-alice",
		"/me/notifications/summary", "", nil, nil)
	body := problemMap(t, raw)
	if resp.StatusCode != http.StatusOK || body["object"] != "notification_summary" {
		t.Fatalf("summary %d %+v", resp.StatusCode, body)
	}
	if asInt(body["unread_count"]) != unmuted {
		t.Errorf("unread_count %v want %d", body["unread_count"], unmuted)
	}
	if asInt(body["muted_unread_count"]) != muted {
		t.Errorf("muted_unread_count %v want %d", body["muted_unread_count"], muted)
	}
	latest, _ := body["latest"].(map[string]any)
	if latest == nil {
		t.Fatal("latest is null")
	}
	if latest["object"] != "notification" {
		t.Errorf("latest object %v", latest["object"])
	}
	if latest["notification_type"] == nil {
		t.Errorf("latest notification_type missing: %+v", latest)
	}
}

func TestV1NotificationsErrors(t *testing.T) {
	f := newMessageFix(t)

	resp, body := f.notifList(t, "sess-alice", "?limit=101")
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "LIMIT_TOO_LARGE" {
		t.Fatalf("limit=101 %d %+v", resp.StatusCode, body)
	}
	resp, body = f.notifList(t, "sess-alice", "?notification_type=nope")
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "UNKNOWN_ENUM_VALUE" {
		t.Fatalf("notification_type=nope %d %+v", resp.StatusCode, body)
	}
	resp, body = f.notifList(t, "sess-alice", "?cursor=cur_not-a-cursor")
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_CURSOR" {
		t.Fatalf("bad cursor %d %+v", resp.StatusCode, body)
	}

	resp, raw := f.doJSON(t, http.MethodGet, notificationsPath, "", "/me/notifications", "", nil, nil)
	body = problemMap(t, raw)
	if resp.StatusCode != http.StatusUnauthorized || body["code"] != "MISSING_CREDENTIAL" {
		t.Fatalf("anonymous %d %+v", resp.StatusCode, body)
	}

	resp, raw = f.doJSON(t, http.MethodDelete, notificationsPath+"/999999999", "sess-alice",
		"/me/notifications/{notification_id}", "", nil, nil)
	body = problemMap(t, raw)
	if resp.StatusCode != http.StatusNotFound || body["code"] != "NOT_FOUND" {
		t.Fatalf("missing id %d %+v", resp.StatusCode, body)
	}
}

func (f *writeFix) notifGet(t *testing.T, session string, id int) (*http.Response, map[string]any) {
	t.Helper()
	path := notificationsPath + "/" + strconv.Itoa(id)
	resp, raw := f.doJSON(t, http.MethodGet, path, session, "/me/notifications/{notification_id}", "", nil, nil)
	return resp, problemMap(t, raw)
}

func TestV1Notifications_N5_GetOwnMatchesList(t *testing.T) {
	f := newMessageFix(t)

	_, listBody := f.notifList(t, "sess-alice", "")
	items, _ := listBody["items"].([]any)
	if len(items) == 0 {
		t.Fatal("empty list")
	}
	want, _ := items[0].(map[string]any)
	id := asInt(want["id"])
	if id == 0 {
		t.Fatalf("list item id %v", want["id"])
	}

	resp, got := f.notifGet(t, "sess-alice", id)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get own %d %+v", resp.StatusCode, got)
	}
	for _, k := range []string{"object", "id", "notification_type", "excerpt_markdown", "is_read", "source", "path", "created_at", "actor_count", "item_count"} {
		if fmt.Sprint(got[k]) != fmt.Sprint(want[k]) {
			t.Fatalf("%s: got %v want %v", k, got[k], want[k])
		}
	}
	ga, _ := got["actor"].(map[string]any)
	wa, _ := want["actor"].(map[string]any)
	if strID(ga["id"]) != strID(wa["id"]) {
		t.Fatalf("actor.id %v want %v", ga["id"], wa["id"])
	}
}

func TestV1Notifications_N5_NotFound(t *testing.T) {
	f := newMessageFix(t)

	cases := []struct {
		name string
		id   int
	}{
		{"other user", mNotifBobOwn},
		{"banned actor", mNotifBanA},
		{"type outside the vocabulary", mNotifUnknown},
		{"unknown id", 999999999},
	}
	for _, tc := range cases {
		resp, body := f.notifGet(t, "sess-alice", tc.id)
		if resp.StatusCode != http.StatusNotFound || body["code"] != "NOT_FOUND" {
			t.Fatalf("%s: %d %+v", tc.name, resp.StatusCode, body)
		}
	}
}
