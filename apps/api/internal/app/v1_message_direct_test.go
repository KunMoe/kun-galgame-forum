package app

import (
	"net/http"
	"strconv"
	"testing"
)

func (f *writeFix) dmGet(t *testing.T, session, userID string, messageID int) (*http.Response, map[string]any) {
	t.Helper()
	path := conversationsPath + "/" + userID + "/messages/" + strconv.Itoa(messageID)
	resp, body := f.doJSON(t, http.MethodGet, path, session, "/me/conversations/{user_id}/messages/{message_id}", "", nil, nil)
	return resp, problemMap(t, body)
}

func viewerIsMine(t *testing.T, body map[string]any) bool {
	t.Helper()
	v, _ := body["viewer"].(map[string]any)
	b, _ := v["is_mine"].(bool)
	return b
}

func TestV1Conversations_M12b_ReadMarkerUnreadIgnoresOwnMessages(t *testing.T) {
	f := newMessageFix(t)
	upTo := mMsgBob2
	bobAbove := f.scalar(t, `SELECT COUNT(*) FROM chat_message WHERE chat_room_id = ? AND sender_id = ? AND id > ?`,
		mRoomBob, w3UserBob, upTo)
	aliceAbove := f.scalar(t, `SELECT COUNT(*) FROM chat_message WHERE chat_room_id = ? AND sender_id = ? AND id > ?`,
		mRoomBob, w3UserAlice, upTo)
	if bobAbove == 0 || aliceAbove == 0 {
		t.Fatalf("seed must have both sides above up_to_id; bob=%d alice=%d", bobAbove, aliceAbove)
	}

	resp, raw := f.doJSON(t, http.MethodPut, conversationsPath+"/"+strconv.Itoa(w3UserBob)+"/read-marker",
		"sess-alice", "/me/conversations/{user_id}/read-marker", "", nil,
		map[string]any{"up_to_id": strconv.Itoa(upTo)})
	body := problemMap(t, raw)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("C6 %d %+v", resp.StatusCode, body)
	}
	if asInt(body["unread_count"]) != bobAbove {
		t.Fatalf("unread_count %v want %d (bob's messages above up_to_id); counting own above would be %d",
			body["unread_count"], bobAbove, bobAbove+aliceAbove)
	}
}

func TestV1DirectMessage_C7_GetIsMine(t *testing.T) {
	f := newMessageFix(t)

	resp, alice := f.dmGet(t, "sess-alice", strconv.Itoa(w3UserBob), mMsgBob2)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("alice get %d %+v", resp.StatusCode, alice)
	}
	if alice["object"] != "direct_message" || strID(alice["id"]) != strconv.Itoa(mMsgBob2) {
		t.Fatalf("alice body %+v", alice)
	}
	if viewerIsMine(t, alice) {
		t.Fatalf("alice on bob's message: is_mine true")
	}

	resp, bob := f.dmGet(t, "sess-bob", strconv.Itoa(w3UserAlice), mMsgBob2)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bob get %d %+v", resp.StatusCode, bob)
	}
	if strID(bob["id"]) != strconv.Itoa(mMsgBob2) {
		t.Fatalf("bob body %+v", bob)
	}
	if !viewerIsMine(t, bob) {
		t.Fatalf("bob on own message: is_mine false")
	}
}

func TestV1DirectMessage_C7_OtherRoomIs404(t *testing.T) {
	f := newMessageFix(t)

	resp, body := f.dmGet(t, "sess-alice", strconv.Itoa(w3UserBob), mMsgOtherOnly)
	if resp.StatusCode != http.StatusNotFound || body["code"] != "NOT_FOUND" {
		t.Fatalf("other room %d %+v", resp.StatusCode, body)
	}
}

func TestV1DirectMessage_C7_RecalledEmptyDocument(t *testing.T) {
	f := newMessageFix(t)

	if err := f.db.Exec(`UPDATE chat_message SET is_recall = true, recall_time = now() WHERE id = ?`, mMsgAliceRec).Error; err != nil {
		t.Fatal(err)
	}
	resp, body := f.dmGet(t, "sess-alice", strconv.Itoa(w3UserBob), mMsgAliceRec)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get recalled %d %+v", resp.StatusCode, body)
	}
	if body["state"] != "recalled" {
		t.Fatalf("state %v", body["state"])
	}
	assertEmptyDocument(t, body["content"])
}

func TestV1DirectMessage_C7_Unavailable(t *testing.T) {
	f := newMessageFix(t)
	f.failOA.Store(true)
	resp, body := f.dmGet(t, "sess-alice", strconv.Itoa(w3UserBob), mMsgBob2)
	if resp.StatusCode != http.StatusServiceUnavailable || body["code"] != "SERVICE_UNAVAILABLE" {
		t.Fatalf("failOA %d %+v", resp.StatusCode, body)
	}
}
