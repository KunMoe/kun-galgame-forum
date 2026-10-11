package app

import (
	"net/http"
	"strconv"
	"testing"
)

const pgUserNoState = 930000830

type pgRead struct {
	url, spec string
}

func pgReads(userID int) []pgRead {
	id := strconv.Itoa(userID)
	return []pgRead{
		{"/api/v1/users/" + id, "/users/{user_id}"},
		{"/api/v1/users/" + id + "/topics?relation=authored", "/users/{user_id}/topics"},
		{"/api/v1/users/" + id + "/replies?relation=authored", "/users/{user_id}/replies"},
		{"/api/v1/users/" + id + "/toolsets", "/users/{user_id}/toolsets"},
		{"/api/v1/quizzes?author_id=" + id, "/quizzes"},
	}
}

func (f *meFix) profileSettings(t *testing.T, session string) map[string]any {
	t.Helper()
	resp, body := f.call(t, http.MethodGet, mePath+"/profile-settings", "/me/profile-settings", session, "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get profile settings %d %+v", resp.StatusCode, body)
	}
	return body
}

func (f *meFix) putProfileSettings(t *testing.T, session string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	return f.call(t, http.MethodPut, mePath+"/profile-settings", "/me/profile-settings", session, "", nil, payload)
}

func (f *meFix) signInRequiredStored(t *testing.T, userID int) (bool, bool) {
	t.Helper()
	var rows []bool
	if err := f.db.Raw(`SELECT is_profile_sign_in_required FROM kungal_user_state WHERE user_id = ?`, userID).
		Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		return false, false
	}
	return rows[0], true
}

func (f *meFix) requireSignIn(t *testing.T, session string, required bool) {
	t.Helper()
	resp, body := f.putProfileSettings(t, session, map[string]any{"is_sign_in_required": required})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put profile settings %d %+v", resp.StatusCode, body)
	}
	if body["object"] != "profile_settings" || body["is_sign_in_required"] != required {
		t.Fatalf("profile settings %+v, want is_sign_in_required %v", body, required)
	}
}

func TestV1ProfileSettingsDefaultAndRoundTrip(t *testing.T) {
	f := newMeFix(t)
	if got := f.profileSettings(t, "sess-alice"); got["object"] != "profile_settings" || got["is_sign_in_required"] != false {
		t.Fatalf("default %+v, want false", got)
	}

	f.requireSignIn(t, "sess-alice", true)
	if stored, _ := f.signInRequiredStored(t, w3UserAlice); !stored {
		t.Fatal("stored flag false after PUT true")
	}
	if got := f.profileSettings(t, "sess-alice"); got["is_sign_in_required"] != true {
		t.Fatalf("after PUT true %+v", got)
	}
	if got := f.profileSettings(t, "sess-bob"); got["is_sign_in_required"] != false {
		t.Fatalf("another user's settings %+v, want their own false", got)
	}

	f.requireSignIn(t, "sess-alice", false)
	if got := f.profileSettings(t, "sess-alice"); got["is_sign_in_required"] != false {
		t.Fatalf("after PUT false %+v", got)
	}
}

func TestV1ProfileSettingsWithoutStateRow(t *testing.T) {
	f := newMeFix(t)
	f.addOAuthUser(pgUserNoState, "stateless", 0, nil)
	f.putSession(t, "sess-stateless", pgUserNoState)
	if _, found := f.signInRequiredStored(t, pgUserNoState); found {
		t.Fatal("fixture already holds a state row")
	}
	if got := f.profileSettings(t, "sess-stateless"); got["is_sign_in_required"] != false {
		t.Fatalf("no state row %+v, want false", got)
	}
	resp, body := f.getUser(t, "", strconv.Itoa(pgUserNoState))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("anonymous read of a user with no state row %d %+v", resp.StatusCode, body)
	}

	f.requireSignIn(t, "sess-stateless", true)
	if stored, found := f.signInRequiredStored(t, pgUserNoState); !found || !stored {
		t.Fatalf("stored %v found %v, want the write to create the row", stored, found)
	}
	resp, body = f.getUser(t, "", strconv.Itoa(pgUserNoState))
	mustCode(t, resp, body, http.StatusForbidden, "PROFILE_SIGN_IN_REQUIRED")
}

func TestV1ProfileSettingsRejectsMissingField(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.putProfileSettings(t, "sess-alice", map[string]any{})
	mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	resp, body = f.putProfileSettings(t, "sess-alice", map[string]any{"is_sign_in_required": "yes"})
	mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
}

func TestV1ProfileSettingsNeedsSignIn(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.call(t, http.MethodGet, mePath+"/profile-settings", "/me/profile-settings", "", "", nil, nil)
	mustCode(t, resp, body, http.StatusUnauthorized, "MISSING_CREDENTIAL")
	resp, body = f.putProfileSettings(t, "", map[string]any{"is_sign_in_required": true})
	mustCode(t, resp, body, http.StatusUnauthorized, "MISSING_CREDENTIAL")
}

func TestV1ProfileGateOffServesAnonymous(t *testing.T) {
	f := newMeFix(t)
	for _, r := range pgReads(w3UserAlice) {
		resp, body := f.call(t, http.MethodGet, r.url, r.spec, "", "", nil, nil)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s anonymous with the flag off: %d %+v", r.url, resp.StatusCode, body)
		}
	}
}

func TestV1ProfileGateOnRefusesAnonymousOnly(t *testing.T) {
	f := newMeFix(t)
	f.requireSignIn(t, "sess-alice", true)
	id := strconv.Itoa(w3UserAlice)

	gatedOnly := []pgRead{
		{"/api/v1/users/" + id + "/followers", "/users/{user_id}/followers"},
		{"/api/v1/users/" + id + "/following", "/users/{user_id}/following"},
		{"/api/v1/ratings?author_id=" + id, "/ratings"},
	}
	for _, r := range append(pgReads(w3UserAlice), gatedOnly...) {
		resp, body := f.call(t, http.MethodGet, r.url, r.spec, "", "", nil, nil)
		if resp.StatusCode != http.StatusForbidden || body["code"] != "PROFILE_SIGN_IN_REQUIRED" {
			t.Errorf("%s anonymous: %d %v, want 403 PROFILE_SIGN_IN_REQUIRED", r.url, resp.StatusCode, body["code"])
		}
	}
	for _, session := range []string{"sess-bob", "sess-alice"} {
		for _, r := range pgReads(w3UserAlice) {
			resp, body := f.call(t, http.MethodGet, r.url, r.spec, session, "", nil, nil)
			if resp.StatusCode != http.StatusOK {
				t.Errorf("%s as %s: %d %+v", r.url, session, resp.StatusCode, body)
			}
		}
	}
	for _, r := range pgReads(w3UserBob) {
		resp, body := f.call(t, http.MethodGet, r.url, r.spec, "", "", nil, nil)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s anonymous, another user: %d %+v", r.url, resp.StatusCode, body)
		}
	}
	resp, body := f.call(t, http.MethodGet, "/api/v1/quizzes", "/quizzes", "", "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("quizzes without author_id anonymous: %d %+v", resp.StatusCode, body)
	}

	f.requireSignIn(t, "sess-alice", false)
	resp, body = f.getUser(t, "", id)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("anonymous after turning it off: %d %+v", resp.StatusCode, body)
	}
}
