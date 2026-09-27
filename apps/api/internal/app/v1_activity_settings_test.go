package app

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"kun-galgame-api/pkg/communityclient"
)

const activitySettingsPath = "/me/activity-settings"

func (u *followingUpstream) serveActivitySettings(w http.ResponseWriter, r *http.Request, path, body string) {
	id, err := strconv.ParseInt(strings.TrimSuffix(strings.TrimPrefix(path, "/users/"), "/activity-settings"), 10, 64)
	if err != nil {
		writeEnvelope(w, http.StatusBadRequest, 40000, "bad id", nil)
		return
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	out := communityclient.ActivitySettings{UserID: id}
	switch r.Method {
	case http.MethodGet:
		out.Hidden = u.settings[id]
	case http.MethodPut:
		var req struct {
			Hidden *bool `json:"hidden"`
		}
		if json.Unmarshal([]byte(body), &req) != nil || req.Hidden == nil {
			writeEnvelope(w, http.StatusUnprocessableEntity, 42200, "hidden is required", nil)
			return
		}
		u.settings[id] = *req.Hidden
		out.Hidden = *req.Hidden
		at := "2026-09-27T08:00:00.123456Z"
		out.UpdatedAt = &at
	default:
		writeEnvelope(w, http.StatusNotFound, 40400, "no route", nil)
		return
	}
	writeEnvelope(w, 200, 0, "", out)
}

func (f *followingFix) activitySettings(t *testing.T, method, session string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	return f.callJSON(t, method, "/api/v1"+activitySettingsPath, activitySettingsPath, session, "", payload)
}

func TestV1ActivitySettingsRoundTrip(t *testing.T) {
	f := newFollowingFix(t)
	settingsPath := "/users/" + strconv.Itoa(w3UserAlice) + "/activity-settings"

	resp, body := f.activitySettings(t, http.MethodGet, "sess-alice", nil)
	if resp.StatusCode != http.StatusOK || body["object"] != "activity_settings" || body["is_hidden"] != false {
		t.Fatalf("unset %d %+v", resp.StatusCode, body)
	}
	if last := f.cm.last(); last.Method != http.MethodGet || last.Path != settingsPath {
		t.Fatalf("read %s %s, want the caller's own settings", last.Method, last.Path)
	}

	resp, body = f.activitySettings(t, http.MethodPut, "sess-alice", map[string]any{"is_hidden": true})
	if resp.StatusCode != http.StatusOK || body["is_hidden"] != true {
		t.Fatalf("hide %d %+v", resp.StatusCode, body)
	}
	if last := f.cm.last(); last.Method != http.MethodPut || last.Path != settingsPath || last.Body != `{"hidden":true}` {
		t.Fatalf("hide sent %s %s %s", last.Method, last.Path, last.Body)
	}
	if _, body = f.activitySettings(t, http.MethodGet, "sess-alice", nil); body["is_hidden"] != true {
		t.Fatalf("read after hide %+v", body)
	}
	if _, body = f.activitySettings(t, http.MethodGet, "sess-bob", nil); body["is_hidden"] != false {
		t.Fatalf("another account read alice's switch: %+v", body)
	}

	resp, body = f.activitySettings(t, http.MethodPut, "sess-alice", map[string]any{"is_hidden": false})
	if resp.StatusCode != http.StatusOK || body["is_hidden"] != false || f.cm.last().Body != `{"hidden":false}` {
		t.Fatalf("show %d %+v sent %s", resp.StatusCode, body, f.cm.last().Body)
	}
}

func TestV1ActivitySettingsValidation(t *testing.T) {
	f := newFollowingFix(t)
	for _, payload := range []map[string]any{{}, {"is_hidden": "yes"}, {"is_hidden": nil}} {
		before := len(f.cm.reqs)
		resp, body := f.activitySettings(t, http.MethodPut, "sess-alice", payload)
		mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
		if len(f.cm.reqs) != before {
			t.Errorf("payload %v reached community", payload)
		}
	}
}

func TestV1ActivitySettingsNeedsSignIn(t *testing.T) {
	f := newFollowingFix(t)
	resp, body := f.activitySettings(t, http.MethodGet, "", nil)
	mustCode(t, resp, body, http.StatusUnauthorized, "MISSING_CREDENTIAL")
	resp, body = f.activitySettings(t, http.MethodPut, "", map[string]any{"is_hidden": true})
	mustCode(t, resp, body, http.StatusUnauthorized, "MISSING_CREDENTIAL")
	if len(f.cm.reqs) != 0 {
		t.Errorf("anonymous calls reached community: %+v", f.cm.reqs)
	}
}

func TestV1ActivitySettingsCommunityDown(t *testing.T) {
	f := newFollowingFix(t)
	f.cm.down.Store(true)
	resp, body := f.activitySettings(t, http.MethodGet, "sess-alice", nil)
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
	resp, body = f.activitySettings(t, http.MethodPut, "sess-alice", map[string]any{"is_hidden": true})
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}
