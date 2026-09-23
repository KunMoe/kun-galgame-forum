package app

import (
	"fmt"
	"net/http"
	"testing"
)

func TestV1GetNotificationPreferencesEmpty(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.getMuted(t, "sess-alice")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get muted %d %+v", resp.StatusCode, body)
	}
	if body["object"] != "notification_preferences" {
		t.Fatalf("object %v", body["object"])
	}
	if fmt.Sprint(body["muted_types"]) != "[]" {
		t.Fatalf("muted_types %v", body["muted_types"])
	}
}

func TestV1GetNotificationPreferencesNoStateRow(t *testing.T) {
	f := newMeFix(t)
	f.execSQL(t, `DELETE FROM kungal_user_state WHERE user_id = ?`, w3UserAlice)
	resp, body := f.getMuted(t, "sess-alice")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get muted %d %+v", resp.StatusCode, body)
	}
	if fmt.Sprint(body["muted_types"]) != "[]" {
		t.Fatalf("muted_types %v", body["muted_types"])
	}
}

func TestV1GetNotificationPreferencesTranslatesFromDB(t *testing.T) {
	f := newMeFix(t)
	t.Cleanup(func() {
		f.db.Exec(`UPDATE kungal_user_state SET muted_notification_types = '[]' WHERE user_id = ?`, w3UserAlice)
	})
	f.execSQL(t, `UPDATE kungal_user_state SET muted_notification_types = '["favorite","nope","chat"]' WHERE user_id = ?`, w3UserAlice)
	resp, body := f.getMuted(t, "sess-alice")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get muted %d %+v", resp.StatusCode, body)
	}
	if fmt.Sprint(body["muted_types"]) != "[favorited chat]" {
		t.Fatalf("muted_types %v, want [favorited chat]", body["muted_types"])
	}
}

func TestV1GetNotificationPreferencesFindByIDFailureIs500(t *testing.T) {
	f := newMeFix(t)
	f.deadState(t)
	resp, body := f.getMuted(t, "sess-alice")
	mustCode(t, resp, body, http.StatusInternalServerError, "INTERNAL_ERROR")
}

func TestV1PutNotificationPreferencesUnknownToken(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.putMuted(t, "sess-alice", []string{"favorite"})
	mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	fieldErr(t, body, "pointer", "/muted_types/0", "UNKNOWN_VALUE")
}

func TestV1PutNotificationPreferencesStoresDBValues(t *testing.T) {
	f := newMeFix(t)
	t.Cleanup(func() {
		f.db.Exec(`DELETE FROM message WHERE id = ?`, u2MsgFavorite)
		f.db.Exec(`UPDATE kungal_user_state SET muted_notification_types = '[]' WHERE user_id = ?`, w3UserAlice)
	})
	resp, body := f.putMuted(t, "sess-alice", []string{"favorited", "chat"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put muted %d %+v", resp.StatusCode, body)
	}
	if fmt.Sprint(body["muted_types"]) != "[favorited chat]" {
		t.Fatalf("response muted_types %v", body["muted_types"])
	}
	got := f.mutedStored(t, w3UserAlice)
	if fmt.Sprint(got) != "[favorite chat]" {
		t.Fatalf("stored %v, want [favorite chat]", got)
	}

	f.execSQL(t, `INSERT INTO message (id, type, sender_id, receiver_id, status, updated)
		VALUES (?, 'favorite', ?, ?, 'unread', now())`, u2MsgFavorite, w3UserBob, w3UserAlice)
	resp, me := f.call(t, http.MethodGet, mePath, "/me", "sess-alice", "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get me %d %+v", resp.StatusCode, me)
	}
	if me["has_unread_messages"] != false {
		t.Fatalf("has_unread_messages %v after muting favorited through v1, want false", me["has_unread_messages"])
	}
}
