package app

import (
	"fmt"
	"hash/fnv"
	"net/http"
	"testing"
	"time"

	"kun-galgame-api/internal/moemoepoint"
)

func TestV1GetMeRoundTrip(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.call(t, http.MethodGet, mePath, "/me", "sess-alice", "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get me %d %+v", resp.StatusCode, body)
	}
	if body["object"] != "me" || body["id"] != fmt.Sprint(w3UserAlice) {
		t.Fatalf("me %+v", body)
	}
	if body["has_checked_in_today"] != false {
		t.Fatalf("has_checked_in_today %v", body["has_checked_in_today"])
	}
	if _, ok := body["moemoepoint"].(float64); !ok {
		t.Fatalf("moemoepoint %v", body["moemoepoint"])
	}
}

func TestV1GetMeUnreadCountFailureIs500(t *testing.T) {
	f := newMeFix(t)
	f.deadStats(t)
	resp, body := f.call(t, http.MethodGet, mePath, "/me", "sess-alice", "", nil, nil)
	mustCode(t, resp, body, http.StatusInternalServerError, "INTERNAL_ERROR")
}

func TestV1GetMeFindByIDFailureIs500(t *testing.T) {
	f := newMeFix(t)
	f.deadState(t)
	resp, body := f.call(t, http.MethodGet, mePath, "/me", "sess-alice", "", nil, nil)
	mustCode(t, resp, body, http.StatusInternalServerError, "INTERNAL_ERROR")
}

func TestV1GetMeUserLookupFailureIs503(t *testing.T) {
	f := newMeFix(t)
	f.failOA.Store(true)
	f.UserClient.Invalidate(w3UserAlice)
	resp, body := f.call(t, http.MethodGet, mePath, "/me", "sess-alice", "", nil, nil)
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1CheckInUsesBeijingDate(t *testing.T) {
	f := newMeFix(t)
	now := time.Date(2026, 9, 23, 17, 30, 0, 0, time.UTC)
	f.UserService.WithClock(func() time.Time { return now })

	resp, body := f.call(t, http.MethodPost, mePath+"/check-ins", "/me/check-ins", "sess-alice", keyUUID(800), nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("check-in %d %+v", resp.StatusCode, body)
	}
	if body["check_in_date"] != "2026-09-24" {
		t.Fatalf("check_in_date %v", body["check_in_date"])
	}
	wantKey := moemoepoint.Key("daily_checkin", fmt.Sprintf("%d_%s", w3UserAlice, "2026-09-24"))
	h := fnv.New32a()
	_, _ = h.Write([]byte(wantKey))
	delta := int(h.Sum32() % 8)
	if delta != 0 {
		got, _ := f.lastAwardKey.Load().(string)
		if got != wantKey {
			t.Fatalf("idempotency key %q, want %q", got, wantKey)
		}
	}
	if asInt(body["moemoepoint_awarded"]) != delta {
		t.Fatalf("awarded %v, want %d", body["moemoepoint_awarded"], delta)
	}
}

func TestV1CheckInDeltaIsDeterministic(t *testing.T) {
	f := newMeFix(t)
	now := time.Date(2026, 9, 23, 17, 30, 0, 0, time.UTC)
	f.UserService.WithClock(func() time.Time { return now })

	resp, first := f.call(t, http.MethodPost, mePath+"/check-ins", "/me/check-ins", "sess-alice", keyUUID(801), nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first %d %+v", resp.StatusCode, first)
	}
	if err := f.UserState.ResetDailyCheckIn(w3UserAlice); err != nil {
		t.Fatal(err)
	}
	resp, second := f.call(t, http.MethodPost, mePath+"/check-ins", "/me/check-ins", "sess-alice", keyUUID(802), nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("second %d %+v", resp.StatusCode, second)
	}
	if first["moemoepoint_awarded"] != second["moemoepoint_awarded"] {
		t.Fatalf("delta %v then %v", first["moemoepoint_awarded"], second["moemoepoint_awarded"])
	}
}

func TestV1CheckInRestoresGateWhenAwardFails(t *testing.T) {
	f := newMeFix(t)
	now := time.Date(2026, 9, 23, 17, 30, 0, 0, time.UTC)
	f.UserService.WithClock(func() time.Time { return now })
	wantKey := moemoepoint.Key("daily_checkin", fmt.Sprintf("%d_%s", w3UserAlice, "2026-09-24"))
	h := fnv.New32a()
	_, _ = h.Write([]byte(wantKey))
	if h.Sum32()%8 == 0 {
		t.Skip("this user-date pair awards 0 and does not call OAuth")
	}
	f.awardFail.Store(true)
	resp, body := f.call(t, http.MethodPost, mePath+"/check-ins", "/me/check-ins", "sess-alice", keyUUID(803), nil, nil)
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")

	resp, me := f.call(t, http.MethodGet, mePath, "/me", "sess-alice", "", nil, nil)
	if resp.StatusCode != http.StatusOK || me["has_checked_in_today"] != false {
		t.Fatalf("after fail %d %+v", resp.StatusCode, me)
	}
	f.awardFail.Store(false)
	resp, retry := f.call(t, http.MethodPost, mePath+"/check-ins", "/me/check-ins", "sess-alice", keyUUID(804), nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("retry %d %+v", resp.StatusCode, retry)
	}
}

func TestV1CheckInSecondIsAlreadyExists(t *testing.T) {
	f := newMeFix(t)
	now := time.Date(2026, 9, 23, 17, 30, 0, 0, time.UTC)
	f.UserService.WithClock(func() time.Time { return now })
	resp, _ := f.call(t, http.MethodPost, mePath+"/check-ins", "/me/check-ins", "sess-alice", keyUUID(805), nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first %d", resp.StatusCode)
	}
	resp, body := f.call(t, http.MethodPost, mePath+"/check-ins", "/me/check-ins", "sess-alice", keyUUID(806), nil, nil)
	mustCode(t, resp, body, http.StatusConflict, "ALREADY_EXISTS")
}

func TestV1GetMeIgnoresMutedNotificationTypes(t *testing.T) {
	f := newMeFix(t)
	exec := func(sql string, args ...any) {
		t.Helper()
		if err := f.db.Exec(sql, args...).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	t.Cleanup(func() {
		f.db.Exec(`DELETE FROM message WHERE id IN (930000901, 930000902)`)
		f.db.Exec(`UPDATE kungal_user_state SET muted_notification_types = '[]' WHERE user_id = ?`, w3UserAlice)
	})
	exec(`UPDATE kungal_user_state SET muted_notification_types = '["liked"]' WHERE user_id = ?`, w3UserAlice)
	exec(`INSERT INTO message (id, type, sender_id, receiver_id, status, updated)
		VALUES (930000901, 'liked', ?, ?, 'unread', now())`, w3UserBob, w3UserAlice)

	unread := func() any {
		t.Helper()
		resp, body := f.call(t, http.MethodGet, mePath, "/me", "sess-alice", "", nil, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("get me %d %+v", resp.StatusCode, body)
		}
		return body["has_unread_messages"]
	}
	if got := unread(); got != false {
		t.Fatalf("only a muted type is unread: has_unread_messages = %v, want false", got)
	}
	exec(`INSERT INTO message (id, type, sender_id, receiver_id, status, updated)
		VALUES (930000902, 'replied', ?, ?, 'unread', now())`, w3UserBob, w3UserAlice)
	if got := unread(); got != true {
		t.Fatalf("an unmuted type is unread: has_unread_messages = %v, want true", got)
	}
}

func TestV1GetMeNotificationCountFailureIs500WithChatMuted(t *testing.T) {
	f := newMeFix(t)
	t.Cleanup(func() {
		f.db.Exec(`UPDATE kungal_user_state SET muted_notification_types = '[]' WHERE user_id = ?`, w3UserAlice)
	})
	if err := f.db.Exec(`UPDATE kungal_user_state SET muted_notification_types = '["chat"]' WHERE user_id = ?`, w3UserAlice).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	f.deadStats(t)
	resp, body := f.call(t, http.MethodGet, mePath, "/me", "sess-alice", "", nil, nil)
	mustCode(t, resp, body, http.StatusInternalServerError, "INTERNAL_ERROR")
}

func TestV1UserFacesRequireACredential(t *testing.T) {
	f := newMeFix(t)
	cases := []struct {
		method, url, spec string
		body              any
	}{
		{http.MethodGet, mePath, "/me", nil},
		{http.MethodPost, mePath + "/check-ins", "/me/check-ins", nil},
		{http.MethodGet, mePath + "/moemoepoint-entries", "/me/moemoepoint-entries", nil},
		{http.MethodGet, mePath + "/preferences", "/me/preferences", nil},
		{http.MethodPut, mePath + "/preferences", "/me/preferences", map[string]any{"doc": map[string]any{}}},
		{http.MethodPut, mePath + "/nsfw-display", "/me/nsfw-display", map[string]any{"nsfw_display": "hide"}},
		{http.MethodGet, "/api/v1/users?q=kun", "/users", nil},
		{http.MethodPatch, mePath + "/profile", "/me/profile", map[string]any{"bio": "x"}},
		{http.MethodGet, mePath + "/creator-status", "/me/creator-status", nil},
		{http.MethodPost, mePath + "/creator-applications", "/me/creator-applications", map[string]any{"statement": "x"}},
	}
	for _, c := range cases {
		t.Run(c.method+" "+c.spec, func(t *testing.T) {
			resp, body := f.call(t, c.method, c.url, c.spec, "", "", nil, c.body)
			mustCode(t, resp, body, http.StatusUnauthorized, "MISSING_CREDENTIAL")
		})
	}
	resp, body := f.putAvatar(t, "", "png-bytes", "image/png", "a.png")
	mustCode(t, resp, body, http.StatusUnauthorized, "MISSING_CREDENTIAL")
}
