package app

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestV1UpdateLogsWalk(t *testing.T) {
	f := newUpdateFix(t, nil)
	want := f.sqlIDs(t, `SELECT id::text FROM update_log ORDER BY created DESC, id DESC`)
	if len(want) < 12 {
		t.Fatalf("seed too thin: %v", want)
	}
	for _, limit := range []int{2, 3} {
		got := f.walkIDs(t, "/api/v1/update-logs", "/update-logs", "", limit)
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("limit %d walked %v, want %v", limit, got, want)
		}
	}
}

func TestV1UpdateLogShape(t *testing.T) {
	f := newUpdateFix(t, nil)
	resp, body := f.up(t, http.MethodGet, fmt.Sprintf("/api/v1/update-logs/%d", upLogMin+1), "/update-logs/{update_log_id}", "", nil, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get %d %+v", resp.StatusCode, body)
	}
	if body["object"] != "update_log" || body["change_type"] != "perf" || body["release_version"] != "4.0.1" ||
		body["text"] != "log 1\nsecond line" || body["created_at"] != "2026-05-01T10:00:00Z" || body["viewer"] != nil {
		t.Errorf("anonymous shape %+v", body)
	}

	_, body = f.up(t, http.MethodGet, fmt.Sprintf("/api/v1/update-logs/%d", upLogMin), "/update-logs/{update_log_id}", "sess-alice", nil, "", nil)
	if v, _ := body["viewer"].(map[string]any); v == nil || v["can_edit"] != false || v["can_delete"] != false {
		t.Errorf("user viewer %+v", body["viewer"])
	}
	_, body = f.up(t, http.MethodGet, fmt.Sprintf("/api/v1/update-logs/%d", upLogMin), "/update-logs/{update_log_id}", "sess-staff", nil, "", nil)
	if v, _ := body["viewer"].(map[string]any); v == nil || v["can_edit"] != true || v["can_delete"] != true {
		t.Errorf("staff viewer %+v", body["viewer"])
	}
	bearer := http.Header{"Authorization": {"Bearer staff-token"}}
	_, body = f.up(t, http.MethodGet, fmt.Sprintf("/api/v1/update-logs/%d", upLogMin), "/update-logs/{update_log_id}", "", bearer, "", nil)
	if v, _ := body["viewer"].(map[string]any); v == nil || v["can_edit"] != false || v["can_delete"] != false {
		t.Errorf("a Bearer moderator carries no staff power: %+v", body["viewer"])
	}

	resp, body = f.up(t, http.MethodGet, "/api/v1/update-logs/930000598", "/update-logs/{update_log_id}", "", nil, "", nil)
	if resp.StatusCode != http.StatusNotFound || body["code"] != "NOT_FOUND" {
		t.Errorf("missing %d %+v", resp.StatusCode, body)
	}
}

func TestV1UpdateLogWrites(t *testing.T) {
	f := newUpdateFix(t, nil)
	create := map[string]any{"change_type": "fix", "release_version": "  4.4.94 ", "text": "fixed the thing"}
	resp, body := f.up(t, http.MethodPost, "/api/v1/update-logs", "/update-logs", "sess-staff", nil, keyUUID(9101), create)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, body)
	}
	id := strID(body["id"])
	if resp.Header.Get("Location") != "/api/v1/update-logs/"+id || body["release_version"] != "4.4.94" || body["change_type"] != "fix" {
		t.Errorf("created %s %+v", resp.Header.Get("Location"), body)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM feed_activity WHERE type = 'UPDATE_LOG_CREATION' AND source_id = ?`, asInt(id)); n != 1 {
		t.Errorf("feed card %d", n)
	}

	resp, body = f.up(t, http.MethodPatch, "/api/v1/update-logs/"+id, "/update-logs/{update_log_id}", "sess-staff", nil, "",
		map[string]any{"change_type": "perf"})
	if resp.StatusCode != http.StatusOK || body["change_type"] != "perf" || body["text"] != "fixed the thing" || body["release_version"] != "4.4.94" {
		t.Errorf("patch %d %+v", resp.StatusCode, body)
	}

	resp, body = f.up(t, http.MethodPatch, "/api/v1/update-logs/"+id, "/update-logs/{update_log_id}", "sess-staff", nil, "",
		map[string]any{"text": "   "})
	if resp.StatusCode != http.StatusUnprocessableEntity || body["code"] != "VALIDATION_FAILED" {
		t.Errorf("blank text %d %+v", resp.StatusCode, body)
	} else if e := problemErrorsOf(body); len(e) != 1 || e[0]["pointer"] != "/text" || e[0]["reason"] != "TOO_SHORT" {
		t.Errorf("blank text errors %+v", e)
	}
	resp, body = f.up(t, http.MethodPost, "/api/v1/update-logs", "/update-logs", "sess-staff", nil, "",
		map[string]any{"change_type": "fix", "release_version": "   ", "text": "x"})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("blank version %d %+v", resp.StatusCode, body)
	} else if e := problemErrorsOf(body); len(e) != 1 || e[0]["pointer"] != "/release_version" || e[0]["reason"] != "TOO_SHORT" {
		t.Errorf("blank version errors %+v", e)
	}
	resp, body = f.up(t, http.MethodPost, "/api/v1/update-logs", "/update-logs", "sess-staff", nil, "",
		map[string]any{"change_type": "pref", "release_version": "1.0.0", "text": "x"})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("retired token pref %d %+v", resp.StatusCode, body)
	}
	resp, body = f.up(t, http.MethodPost, "/api/v1/update-logs", "/update-logs", "sess-staff", nil, "",
		map[string]any{"change_type": "fix", "release_version": "1.0.0", "text": strings.Repeat("a", 1001)})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("too long %d %+v", resp.StatusCode, body)
	}

	resp, _ = f.up(t, http.MethodDelete, "/api/v1/update-logs/"+id, "/update-logs/{update_log_id}", "sess-staff", nil, "", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete %d", resp.StatusCode)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM update_log WHERE id = ?`, asInt(id)); n != 0 {
		t.Error("delete left the row")
	}
	resp, body = f.up(t, http.MethodDelete, "/api/v1/update-logs/"+id, "/update-logs/{update_log_id}", "sess-staff", nil, "", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("delete again %d %+v", resp.StatusCode, body)
	}
	resp, body = f.up(t, http.MethodPatch, "/api/v1/update-logs/"+id, "/update-logs/{update_log_id}", "sess-staff", nil, "", map[string]any{})
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("patch a deleted entry %d %+v", resp.StatusCode, body)
	}
}

func TestV1UpdateLogWritesNeedTheirPermissions(t *testing.T) {
	f := newUpdateFix(t, nil)
	log := fmt.Sprintf("/api/v1/update-logs/%d", upLogMin)
	bearer := http.Header{"Authorization": {"Bearer staff-token"}}
	body := map[string]any{"change_type": "fix", "release_version": "1.0.0", "text": "x"}
	for _, c := range []struct {
		name, method, url, spec, session string
		hdr                              http.Header
		payload                          any
		status                           int
		code                             string
	}{
		{"create as a user", http.MethodPost, "/api/v1/update-logs", "/update-logs", "sess-alice", nil, body, 403, "PERMISSION_REQUIRED"},
		{"create anonymously", http.MethodPost, "/api/v1/update-logs", "/update-logs", "", nil, body, 401, "MISSING_CREDENTIAL"},
		{"create as a Bearer moderator", http.MethodPost, "/api/v1/update-logs", "/update-logs", "", bearer, body, 403, "PERMISSION_REQUIRED"},
		{"edit as a user", http.MethodPatch, log, "/update-logs/{update_log_id}", "sess-alice", nil, map[string]any{"text": "y"}, 403, "PERMISSION_REQUIRED"},
		{"edit a missing entry as a user", http.MethodPatch, "/api/v1/update-logs/930000598", "/update-logs/{update_log_id}", "sess-alice", nil, map[string]any{}, 403, "PERMISSION_REQUIRED"},
		{"delete as a user", http.MethodDelete, log, "/update-logs/{update_log_id}", "sess-alice", nil, nil, 403, "PERMISSION_REQUIRED"},
		{"delete as a Bearer moderator", http.MethodDelete, log, "/update-logs/{update_log_id}", "", bearer, nil, 403, "PERMISSION_REQUIRED"},
		{"create as a banned staff account", http.MethodPost, "/api/v1/update-logs", "/update-logs", "sess-banned", nil, body, 403, "ACCOUNT_BANNED"},
	} {
		resp, got := f.up(t, c.method, c.url, c.spec, c.session, c.hdr, "", c.payload)
		if resp.StatusCode != c.status || got["code"] != c.code {
			t.Errorf("%s: %d %+v", c.name, resp.StatusCode, got)
		}
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM update_log WHERE id = ? AND content = 'log 0' || chr(10) || 'second line'`, upLogMin); n != 1 {
		t.Error("a refused write changed the entry")
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM update_log`); n != 12 {
		t.Errorf("a refused create wrote a row: %d", n)
	}
}

func TestV1UpdateLogWritesFailClosedWhenOAuthIsDown(t *testing.T) {
	f := newUpdateFix(t, nil)
	f.failOA.Store(true)
	resp, body := f.up(t, http.MethodPost, "/api/v1/update-logs", "/update-logs", "sess-staff", nil, "",
		map[string]any{"change_type": "fix", "release_version": "1.0.0", "text": "x"})
	if resp.StatusCode != http.StatusServiceUnavailable || body["code"] != "SERVICE_UNAVAILABLE" {
		t.Errorf("OAuth down %d %+v", resp.StatusCode, body)
	}
}

func TestV1UpdateLogListErrors(t *testing.T) {
	f := newUpdateFix(t, nil)
	for _, c := range []struct{ url, code string }{
		{"/api/v1/update-logs?cursor=cur_garbage", "INVALID_CURSOR"},
		{"/api/v1/update-logs?limit=101", "LIMIT_TOO_LARGE"},
	} {
		resp, body := f.up(t, http.MethodGet, c.url, "/update-logs", "", nil, "", nil)
		if resp.StatusCode != http.StatusBadRequest || body["code"] != c.code {
			t.Errorf("%s: %d %+v", c.url, resp.StatusCode, body)
		}
	}
	_, first := f.up(t, http.MethodGet, "/api/v1/todos?limit=1", "/todos", "", nil, "", nil)
	cur, _ := first["next_cursor"].(string)
	resp, body := f.up(t, http.MethodGet, "/api/v1/update-logs?cursor="+cur, "/update-logs", "", nil, "", nil)
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_CURSOR" {
		t.Errorf("a todo cursor on the update-log list %d %+v", resp.StatusCode, body)
	}
}

func problemErrorsOf(body map[string]any) []map[string]any {
	raw, _ := body["errors"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, e := range raw {
		if m, ok := e.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}
