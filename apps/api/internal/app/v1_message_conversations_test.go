package app

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	mRoomBob    = 960000201
	mRoomOther  = 960000202
	mRoomBanned = 960000203
	mRoomStaff  = 960000205

	mMsgBob1     = 960000301
	mMsgAlice1   = 960000302
	mMsgBob2     = 960000303
	mMsgAlice2   = 960000304
	mMsgBobPeer  = 960000305
	mMsgAliceRec = 960000306

	mMsgOtherOnly = 960000310
	mMsgBanned    = 960000311
	mMsgStaff     = 960000312
)

const conversationsPath = "/api/v1/me/conversations"

func (f *writeFix) seedConversations(t *testing.T, run func(string, ...any), base time.Time) {
	t.Helper()
	now := base.Add(8 * time.Hour)
	insertRoom := func(id int, name string, at time.Time) {
		t.Helper()
		run(`INSERT INTO chat_room (id, name, type, last_message_content, last_message_time, last_message_sender_id, last_message_sender_name, created, updated)
			VALUES (?, ?, 'private', 'SHOULD_NOT_READ', ?, 0, 'x', ?, ?)`,
			id, name, at, at, at)
	}
	insertPart := func(roomID, userID int) {
		t.Helper()
		run(`INSERT INTO chat_room_participant (chat_room_id, user_id, created, updated) VALUES (?, ?, ?, ?)`,
			roomID, userID, now, now)
	}
	insertMsg := func(id, roomID, sender, receiver int, content string, at time.Time) {
		t.Helper()
		name := fmt.Sprintf("%d-%d", minInt(sender, receiver), maxInt(sender, receiver))
		run(`INSERT INTO chat_message (id, chat_room_id, chatroom_name, sender_id, receiver_id, content, is_recall, created, updated)
			VALUES (?, ?, ?, ?, ?, ?, false, ?, ?)`,
			id, roomID, name, sender, receiver, content, at, at)
	}

	bobName := fmt.Sprintf("%d-%d", w3UserAlice, w3UserBob)
	otherName := fmt.Sprintf("%d-%d", w3UserAlice, w3UserOther)
	bannedName := fmt.Sprintf("%d-%d", w3UserAlice, w3UserBanned)
	staffName := fmt.Sprintf("%d-%d", w3UserAlice, w3UserStaff)

	t0 := now
	t1 := now.Add(time.Minute)
	t2 := now.Add(2 * time.Minute)
	t3 := now.Add(3 * time.Minute)
	t4 := now.Add(4 * time.Minute)
	t5 := now.Add(5 * time.Minute)
	insertRoom(mRoomBob, bobName, t5)
	insertRoom(mRoomOther, otherName, t0)
	insertRoom(mRoomBanned, bannedName, t0)
	insertRoom(mRoomStaff, staffName, t0)

	insertPart(mRoomBob, w3UserAlice)
	insertPart(mRoomBob, w3UserBob)
	insertPart(mRoomOther, w3UserAlice)
	insertPart(mRoomOther, w3UserOther)
	insertPart(mRoomBanned, w3UserAlice)
	insertPart(mRoomBanned, w3UserBanned)
	insertPart(mRoomStaff, w3UserAlice)
	insertPart(mRoomStaff, w3UserStaff)

	insertMsg(mMsgBob1, mRoomBob, w3UserBob, w3UserAlice, "hello from bob", t0)
	insertMsg(mMsgAlice1, mRoomBob, w3UserAlice, w3UserBob, "hello from alice", t1)
	insertMsg(mMsgBob2, mRoomBob, w3UserBob, w3UserAlice, "second from bob", t2)
	insertMsg(mMsgAlice2, mRoomBob, w3UserAlice, w3UserBob, "second from alice", t3)
	insertMsg(mMsgBobPeer, mRoomBob, w3UserBob, w3UserAlice, "bob peer message", t4)
	insertMsg(mMsgAliceRec, mRoomBob, w3UserAlice, w3UserBob, "alice will recall", t5)

	same := now.Add(-time.Hour)
	insertMsg(mMsgOtherOnly, mRoomOther, w3UserAlice, w3UserOther, "only mine", same)
	insertMsg(mMsgStaff, mRoomStaff, w3UserStaff, w3UserAlice, "from staff", same)
	insertMsg(mMsgBanned, mRoomBanned, w3UserBanned, w3UserAlice, "from banned", now.Add(-time.Minute))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (f *writeFix) convList(t *testing.T, session, query string) (*http.Response, map[string]any) {
	t.Helper()
	resp, body := f.doJSON(t, http.MethodGet, conversationsPath+query, session, "/me/conversations", "", nil, nil)
	return resp, problemMap(t, body)
}

func (f *writeFix) dmList(t *testing.T, session, userID, query string) (*http.Response, map[string]any) {
	t.Helper()
	path := conversationsPath + "/" + userID + "/messages" + query
	resp, body := f.doJSON(t, http.MethodGet, path, session, "/me/conversations/{user_id}/messages", "", nil, nil)
	return resp, problemMap(t, body)
}

func (f *writeFix) sendDM(t *testing.T, session, userID, key string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	path := conversationsPath + "/" + userID + "/messages"
	resp, body := f.doJSON(t, http.MethodPost, path, session, "/me/conversations/{user_id}/messages", key, nil, payload)
	return resp, problemMap(t, body)
}

func peerIDs(t *testing.T, body map[string]any) []string {
	t.Helper()
	raw, _ := body["items"].([]any)
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		row, _ := item.(map[string]any)
		peer, _ := row["peer"].(map[string]any)
		out = append(out, strID(peer["id"]))
	}
	return out
}

func TestV1ConversationsTraversal(t *testing.T) {
	f := newMessageFix(t)

	resp, body := f.convList(t, "sess-alice", "")
	if resp.StatusCode != http.StatusOK || body["object"] != "list" {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	full := peerIDs(t, body)
	for _, id := range full {
		if id == strconv.Itoa(w3UserBanned) {
			t.Fatalf("banned peer listed: %v", full)
		}
	}
	if len(full) != 3 {
		t.Fatalf("alice should see bob, other, staff; got %v", full)
	}
	for _, item := range body["items"].([]any) {
		row, _ := item.(map[string]any)
		assertConversationID(t, row)
	}

	var walked []string
	cursor := ""
	for page := 0; page < 10; page++ {
		q := "?limit=1"
		if cursor != "" {
			q += "&cursor=" + cursor
		}
		resp, body := f.convList(t, "sess-alice", q)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("page %d: %d %+v", page, resp.StatusCode, body)
		}
		walked = append(walked, peerIDs(t, body)...)
		next, ok := body["next_cursor"].(string)
		if !ok {
			break
		}
		cursor = next
	}
	if fmt.Sprint(walked) != fmt.Sprint(full) {
		t.Fatalf("conversation traversal\n got %v\nwant %v", walked, full)
	}
}

func TestV1Conversations_M8_ListDoesNotCreateRoom(t *testing.T) {
	f := newMessageFix(t)

	before := f.scalar(t, `SELECT COUNT(*) FROM chat_room`)
	peer := strconv.Itoa(w3UserGrant)
	resp, body := f.dmList(t, "sess-alice", peer, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("C3 empty %d %+v", resp.StatusCode, body)
	}
	if ids := listIDs(t, body); len(ids) != 0 {
		t.Fatalf("expected empty list, got %v", ids)
	}
	after := f.scalar(t, `SELECT COUNT(*) FROM chat_room`)
	if after != before {
		t.Fatalf("C3 created a room: %d -> %d", before, after)
	}
}

func TestV1Conversations_M9_RecallPeerIs403(t *testing.T) {
	f := newMessageFix(t)

	path := conversationsPath + "/" + strconv.Itoa(w3UserBob) + "/messages/" + strconv.Itoa(mMsgBobPeer)
	resp, raw := f.doJSON(t, http.MethodPatch, path, "sess-alice",
		"/me/conversations/{user_id}/messages/{message_id}", "", nil,
		map[string]any{"state": "recalled"})
	body := problemMap(t, raw)
	if resp.StatusCode != http.StatusForbidden || body["code"] != "PERMISSION_REQUIRED" {
		t.Fatalf("recall peer %d %+v", resp.StatusCode, body)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM chat_message WHERE id = ? AND is_recall = false`, mMsgBobPeer); n != 1 {
		t.Fatalf("peer's message was recalled: unread/sent rows=%d", n)
	}
}

func TestV1Conversations_M10_RecalledContentIsEmpty(t *testing.T) {
	f := newMessageFix(t)

	path := conversationsPath + "/" + strconv.Itoa(w3UserBob) + "/messages/" + strconv.Itoa(mMsgAliceRec)
	resp, raw := f.doJSON(t, http.MethodPatch, path, "sess-alice",
		"/me/conversations/{user_id}/messages/{message_id}", "", nil,
		map[string]any{"state": "recalled"})
	body := problemMap(t, raw)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("recall own %d %+v", resp.StatusCode, body)
	}
	if body["state"] != "recalled" {
		t.Errorf("state %v", body["state"])
	}
	assertEmptyDocument(t, body["content"])
	if body["recalled_at"] == nil {
		t.Fatal("recalled_at is null")
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM chat_room WHERE id = ? AND last_message_content = ''`, mRoomBob); n != 1 {
		t.Fatalf("latest recall did not clear last_message_content")
	}

	resp, raw = f.doJSON(t, http.MethodPatch, path, "sess-alice",
		"/me/conversations/{user_id}/messages/{message_id}", "", nil,
		map[string]any{"state": "recalled"})
	again := problemMap(t, raw)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("recall again %d %+v", resp.StatusCode, again)
	}
	if again["recalled_at"] != body["recalled_at"] {
		t.Fatalf("second recall changed recalled_at: %v vs %v", again["recalled_at"], body["recalled_at"])
	}
}

func TestV1Conversations_M11_MissingPeerIs404(t *testing.T) {
	f := newMessageFix(t)

	beforeRooms := f.scalar(t, `SELECT COUNT(*) FROM chat_room`)
	beforeMsgs := f.scalar(t, `SELECT COUNT(*) FROM chat_message`)
	resp, body := f.sendDM(t, "sess-alice", "999999999", keyUUID(9601), map[string]any{
		"content_markdown": "hello ghost",
	})
	if resp.StatusCode != http.StatusNotFound || body["code"] != "NOT_FOUND" {
		t.Fatalf("missing peer %d %+v", resp.StatusCode, body)
	}
	if f.scalar(t, `SELECT COUNT(*) FROM chat_room`) != beforeRooms {
		t.Fatal("missing peer created a room")
	}
	if f.scalar(t, `SELECT COUNT(*) FROM chat_message`) != beforeMsgs {
		t.Fatal("missing peer inserted a message")
	}
}

func TestV1Conversations_M12_UnreadIgnoresOwnMessages(t *testing.T) {
	f := newMessageFix(t)

	resp, raw := f.doJSON(t, http.MethodGet, conversationsPath+"/"+strconv.Itoa(w3UserOther),
		"sess-alice", "/me/conversations/{user_id}", "", nil, nil)
	body := problemMap(t, raw)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get conversation %d %+v", resp.StatusCode, body)
	}
	if asInt(body["unread_count"]) != 0 {
		t.Fatalf("own-only conversation unread_count=%v want 0", body["unread_count"])
	}
	assertConversationID(t, body)

	_, listed := f.convList(t, "sess-alice", "")
	for _, item := range listed["items"].([]any) {
		row, _ := item.(map[string]any)
		peer, _ := row["peer"].(map[string]any)
		assertConversationID(t, row)
		if strID(peer["id"]) == strconv.Itoa(w3UserOther) && asInt(row["unread_count"]) != 0 {
			t.Fatalf("C1 unread_count for own-only conversation is %v", row["unread_count"])
		}
	}
}

func TestV1Conversations_M13_ExistingRoomNameIs201(t *testing.T) {
	f := newMessageFix(t)

	name := fmt.Sprintf("%d-%d", w3UserAlice, w3UserGrant)
	now := time.Now().UTC()
	if err := f.db.Exec(
		`INSERT INTO chat_room (id, name, type, last_message_content, created, updated) VALUES (?, ?, 'private', '', ?, ?)`,
		960000204, name, now, now,
	).Error; err != nil {
		t.Fatal(err)
	}

	resp, body := f.sendDM(t, "sess-alice", strconv.Itoa(w3UserGrant), keyUUID(9602), map[string]any{
		"content_markdown": "first to grant",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first send into existing name %d %+v", resp.StatusCode, body)
	}
	if body["object"] != "direct_message" {
		t.Errorf("object %v", body["object"])
	}
}

func TestV1DirectMessagesTraversal(t *testing.T) {
	f := newMessageFix(t)

	peer := strconv.Itoa(w3UserBob)
	resp, body := f.dmList(t, "sess-alice", peer, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	full := listIDs(t, body)
	if len(full) != 6 {
		t.Fatalf("bob conversation has 6 messages, got %v", full)
	}

	var walked []string
	cursor := ""
	for page := 0; page < 10; page++ {
		q := "?limit=2"
		if cursor != "" {
			q += "&cursor=" + cursor
		}
		resp, body := f.dmList(t, "sess-alice", peer, q)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("page %d: %d %+v", page, resp.StatusCode, body)
		}
		walked = append(walked, listIDs(t, body)...)
		next, ok := body["next_cursor"].(string)
		if !ok {
			break
		}
		cursor = next
	}
	if fmt.Sprint(walked) != fmt.Sprint(full) {
		t.Fatalf("message traversal\n got %v\nwant %v", walked, full)
	}
}

func TestV1DirectMessageSendValidation(t *testing.T) {
	f := newMessageFix(t)
	peer := strconv.Itoa(w3UserBob)

	resp, body := f.sendDM(t, "sess-alice", peer, keyUUID(9610), map[string]any{
		"content_markdown": "   ",
	})
	if resp.StatusCode != http.StatusUnprocessableEntity || body["code"] != "VALIDATION_FAILED" {
		t.Fatalf("blank %d %+v", resp.StatusCode, body)
	}
	errs, _ := body["errors"].([]any)
	if len(errs) == 0 {
		t.Fatalf("errors %v", body["errors"])
	}
	e, _ := errs[0].(map[string]any)
	if e["pointer"] != "/content_markdown" || e["reason"] != "REQUIRED" {
		t.Fatalf("field error %+v", e)
	}

	resp, body = f.sendDM(t, "sess-alice", peer, keyUUID(9611), map[string]any{
		"content_markdown": strings.Repeat("x", 1001),
	})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("too long %d %+v", resp.StatusCode, body)
	}
	errs, _ = body["errors"].([]any)
	e, _ = errs[0].(map[string]any)
	if e["pointer"] != "/content_markdown" || e["reason"] != "TOO_LONG" {
		t.Fatalf("too long field %+v", e)
	}

	resp, raw := f.doJSON(t, http.MethodPost, conversationsPath+"/"+peer+"/messages", "sess-alice",
		"/me/conversations/{user_id}/messages", "", nil, map[string]any{"content_markdown": "x"})
	body = problemMap(t, raw)
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_PARAMETER" {
		t.Fatalf("missing Idempotency-Key %d %+v", resp.StatusCode, body)
	}
}

func TestV1DirectMessageSendIdempotent(t *testing.T) {
	f := newMessageFix(t)
	peer := strconv.Itoa(w3UserOther)
	before := f.scalar(t, `SELECT COUNT(*) FROM chat_message WHERE sender_id = ?`, w3UserAlice)
	key := keyUUID(9620)
	payload := map[string]any{"content_markdown": "replayed hello"}

	resp, first := f.sendDM(t, "sess-alice", peer, key, payload)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first %d %+v", resp.StatusCode, first)
	}
	resp, second := f.sendDM(t, "sess-alice", peer, key, payload)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("replay %d %+v", resp.StatusCode, second)
	}
	if strID(first["id"]) != strID(second["id"]) {
		t.Fatalf("replay minted a second message: %v vs %v", first["id"], second["id"])
	}
	after := f.scalar(t, `SELECT COUNT(*) FROM chat_message WHERE sender_id = ?`, w3UserAlice)
	if after != before+1 {
		t.Fatalf("replay wrote a row: %d -> %d", before, after)
	}
}

func TestV1DirectMessageSendUnavailable(t *testing.T) {
	f := newMessageFix(t)
	f.failOA.Store(true)
	resp, body := f.sendDM(t, "sess-alice", strconv.Itoa(w3UserBob), keyUUID(9630), map[string]any{
		"content_markdown": "should not write",
	})
	if resp.StatusCode != http.StatusServiceUnavailable || body["code"] != "SERVICE_UNAVAILABLE" {
		t.Fatalf("failOA %d %+v", resp.StatusCode, body)
	}
}

func TestV1ConversationGetAndReadMarker(t *testing.T) {
	f := newMessageFix(t)

	resp, raw := f.doJSON(t, http.MethodGet, conversationsPath+"/"+strconv.Itoa(w3UserAlice),
		"sess-alice", "/me/conversations/{user_id}", "", nil, nil)
	body := problemMap(t, raw)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("self %d %+v", resp.StatusCode, body)
	}

	resp, raw = f.doJSON(t, http.MethodGet, conversationsPath+"/"+strconv.Itoa(w3UserBanned),
		"sess-alice", "/me/conversations/{user_id}", "", nil, nil)
	body = problemMap(t, raw)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("banned peer %d %+v", resp.StatusCode, body)
	}

	resp, raw = f.doJSON(t, http.MethodGet, conversationsPath+"/"+strconv.Itoa(w3UserGrant),
		"sess-alice", "/me/conversations/{user_id}", "", nil, nil)
	body = problemMap(t, raw)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("empty conversation %d %+v", resp.StatusCode, body)
	}
	if asInt(body["message_count"]) != 0 || body["last_message"] != nil {
		t.Fatalf("empty conversation %+v", body)
	}
	assertConversationID(t, body)
	if strID(body["id"]) != strconv.Itoa(w3UserGrant) {
		t.Fatalf("empty conversation id %v want %d", body["id"], w3UserGrant)
	}

	resp, raw = f.doJSON(t, http.MethodPut, conversationsPath+"/"+strconv.Itoa(w3UserBob)+"/read-marker",
		"sess-alice", "/me/conversations/{user_id}/read-marker", "", nil,
		map[string]any{"up_to_id": strconv.Itoa(mMsgBob2)})
	body = problemMap(t, raw)
	if resp.StatusCode != http.StatusOK || body["object"] != "direct_message_read_marker" {
		t.Fatalf("read-marker %d %+v", resp.StatusCode, body)
	}
	if asInt(body["marked_count"]) < 1 {
		t.Errorf("marked_count %v", body["marked_count"])
	}
	if asInt(body["unread_count"]) != 1 {
		t.Errorf("unread_count %v want 1 (bob's later message)", body["unread_count"])
	}

	resp, body = f.dmList(t, "sess-alice", strconv.Itoa(w3UserBob), "?limit=101")
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "LIMIT_TOO_LARGE" {
		t.Fatalf("limit %d %+v", resp.StatusCode, body)
	}
	resp, body = f.dmList(t, "sess-alice", strconv.Itoa(w3UserBob), "?cursor=cur_nope")
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_CURSOR" {
		t.Fatalf("cursor %d %+v", resp.StatusCode, body)
	}

	resp, raw = f.doJSON(t, http.MethodPut, conversationsPath+"/"+strconv.Itoa(w3UserGrant)+"/read-marker",
		"sess-alice", "/me/conversations/{user_id}/read-marker", "", nil,
		map[string]any{"up_to_id": "1"})
	body = problemMap(t, raw)
	if resp.StatusCode != http.StatusOK || asInt(body["marked_count"]) != 0 || asInt(body["unread_count"]) != 0 {
		t.Fatalf("missing room read-marker %d %+v", resp.StatusCode, body)
	}
}

func TestV1DirectMessageSelfAndMissingMessage(t *testing.T) {
	f := newMessageFix(t)

	resp, body := f.dmList(t, "sess-alice", strconv.Itoa(w3UserAlice), "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("messages with self %d %+v", resp.StatusCode, body)
	}

	path := conversationsPath + "/" + strconv.Itoa(w3UserBob) + "/messages/999999999"
	resp, raw := f.doJSON(t, http.MethodPatch, path, "sess-alice",
		"/me/conversations/{user_id}/messages/{message_id}", "", nil,
		map[string]any{"state": "recalled"})
	body = problemMap(t, raw)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing message %d %+v", resp.StatusCode, body)
	}
}

func assertConversationID(t *testing.T, row map[string]any) {
	t.Helper()
	peer, _ := row["peer"].(map[string]any)
	if strID(row["id"]) == "" || strID(row["id"]) != strID(peer["id"]) {
		t.Fatalf("conversation id %v != peer.id %v", row["id"], peer["id"])
	}
}

func assertEmptyDocument(t *testing.T, raw any) {
	t.Helper()
	doc, _ := raw.(map[string]any)
	if doc == nil {
		t.Fatal("content is null")
	}
	children, ok := doc["children"].([]any)
	if !ok {
		t.Fatalf("content.children %v", doc["children"])
	}
	if len(children) != 0 {
		t.Fatalf("content.children len %d want 0: %v", len(children), children)
	}
}

func TestV1ConversationC6Unavailable(t *testing.T) {
	f := newMessageFix(t)
	f.failOA.Store(true)
	resp, raw := f.doJSON(t, http.MethodPut, conversationsPath+"/"+strconv.Itoa(w3UserBob)+"/read-marker",
		"sess-alice", "/me/conversations/{user_id}/read-marker", "", nil,
		map[string]any{"up_to_id": "1"})
	body := problemMap(t, raw)
	if resp.StatusCode != http.StatusServiceUnavailable || body["code"] != "SERVICE_UNAVAILABLE" {
		t.Fatalf("C6 failOA %d %+v", resp.StatusCode, body)
	}
}
