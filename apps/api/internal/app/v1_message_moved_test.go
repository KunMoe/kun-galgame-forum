package app

import (
	"net/http"
	"strconv"
	"testing"
	"time"
)

func (f *writeFix) moveDirectMessagesToChat(t *testing.T) {
	t.Helper()
	f.Config.Chat.Enabled = true
	f.Fiber = newFiber()
	f.setupRoutes()
}

func TestV1DirectMessageWritesAreRefusedOnceMoved(t *testing.T) {
	f := newMessageFix(t)
	f.moveDirectMessagesToChat(t)
	bob := strconv.Itoa(w3UserBob)
	messages := `SELECT COUNT(*) FROM chat_message WHERE chat_room_id = ? AND is_recall = false`
	reads := `SELECT COUNT(*) FROM chat_message_read_by WHERE user_id = ?`
	beforeMessages, beforeReads := f.scalar(t, messages, mRoomBob), f.scalar(t, reads, w3UserAlice)

	resp, body := f.sendDM(t, "sess-alice", bob, keyUUID(9710), map[string]any{"content_markdown": "after the move"})
	mustCode(t, resp, body, http.StatusGone, "DIRECT_MESSAGES_MOVED")

	resp, raw := f.doJSON(t, http.MethodPatch, conversationsPath+"/"+bob+"/messages/"+strconv.Itoa(mMsgAliceRec),
		"sess-alice", "/me/conversations/{user_id}/messages/{message_id}", "", nil, map[string]any{"state": "recalled"})
	mustCode(t, resp, problemMap(t, raw), http.StatusGone, "DIRECT_MESSAGES_MOVED")

	resp, raw = f.doJSON(t, http.MethodPut, conversationsPath+"/"+bob+"/read-marker",
		"sess-alice", "/me/conversations/{user_id}/read-marker", "", nil, map[string]any{"up_to_id": strconv.Itoa(mMsgBobPeer)})
	mustCode(t, resp, problemMap(t, raw), http.StatusGone, "DIRECT_MESSAGES_MOVED")

	if got := f.scalar(t, messages, mRoomBob); got != beforeMessages {
		t.Fatalf("a refused write changed the room: %d -> %d live messages", beforeMessages, got)
	}
	if got := f.scalar(t, reads, w3UserAlice); got != beforeReads {
		t.Fatalf("a refused read marker wrote receipts: %d -> %d", beforeReads, got)
	}

	if resp, list := f.dmList(t, "sess-alice", bob, ""); resp.StatusCode != http.StatusOK || len(list["items"].([]any)) == 0 {
		t.Fatalf("the old history stays readable: %d %+v", resp.StatusCode, list)
	}
	if resp, list := f.convList(t, "sess-alice", ""); resp.StatusCode != http.StatusOK || len(list["items"].([]any)) == 0 {
		t.Fatalf("the old conversations stay listed: %d %+v", resp.StatusCode, list)
	}
}

func TestV1DirectMessageWritesStillWorkBeforeTheMove(t *testing.T) {
	f := newMessageFix(t)
	resp, raw := f.doJSON(t, http.MethodPut, conversationsPath+"/"+strconv.Itoa(w3UserBob)+"/read-marker",
		"sess-alice", "/me/conversations/{user_id}/read-marker", "", nil, map[string]any{"up_to_id": strconv.Itoa(mMsgBob2)})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("read marker with chat off: %d %s", resp.StatusCode, raw)
	}
}

func TestV1GetMeStopsCountingDirectMessagesOnceMoved(t *testing.T) {
	f := newMeFix(t)
	f.cleanupMessageDomain(t)
	t.Cleanup(func() { f.cleanupMessageDomain(t) })
	f.seedConversations(t, func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}, time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC))

	unread := func() any {
		t.Helper()
		resp, body := f.call(t, http.MethodGet, mePath, "/me", "sess-alice", "", nil, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("get me %d %+v", resp.StatusCode, body)
		}
		return body["has_unread_messages"]
	}
	if got := unread(); got != true {
		t.Fatalf("bob's unread direct messages before the move: has_unread_messages = %v, want true", got)
	}
	f.moveDirectMessagesToChat(t)
	if got := unread(); got != false {
		t.Fatalf("after the move nothing can mark them read: has_unread_messages = %v, want false", got)
	}
}
