package app

import (
	"net/http"
	"testing"
)

const prefsPath = mePath + "/preferences"

func ifMatch(v string) http.Header {
	h := http.Header{}
	h.Set("If-Match", v)
	return h
}

func TestV1GetPreferences(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.call(t, http.MethodGet, prefsPath, "/me/preferences", "sess-alice", "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get preferences %d %+v", resp.StatusCode, body)
	}
	if body["object"] != "preferences" || body["version"] != float64(4) || body["written_at"] != "2026-09-22T08:31:00Z" {
		t.Fatalf("preferences %+v", body)
	}
	if got := resp.Header.Get("ETag"); got != `"4"` {
		t.Fatalf("ETag %q, want \"4\"", got)
	}
	doc, _ := body["doc"].(map[string]any)
	if doc["show_rating"] != false {
		t.Fatalf("doc %+v", doc)
	}
}

func TestV1PutPreferences(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.call(t, http.MethodPut, prefsPath, "/me/preferences", "sess-alice", "", ifMatch(`"4"`),
		map[string]any{"doc": map[string]any{"ok": true}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put preferences %d %+v", resp.StatusCode, body)
	}
	if body["version"] != float64(5) || resp.Header.Get("ETag") != `"5"` {
		t.Fatalf("put preferences %+v etag %q", body, resp.Header.Get("ETag"))
	}
}

func TestV1PutPreferencesIfMatchConflict(t *testing.T) {
	f := newMeFix(t)
	f.prefCode.Store(18006)
	f.prefHTTP.Store(412)
	resp, body := f.call(t, http.MethodPut, prefsPath, "/me/preferences", "sess-alice", "", ifMatch(`"3"`),
		map[string]any{"doc": map[string]any{}})
	mustCode(t, resp, body, http.StatusPreconditionFailed, "PRECONDITION_FAILED")
}

func TestV1PutPreferencesScopeMissing(t *testing.T) {
	f := newMeFix(t)
	f.prefCode.Store(18001)
	f.prefHTTP.Store(403)
	resp, body := f.call(t, http.MethodPut, prefsPath, "/me/preferences", "sess-alice", "", nil,
		map[string]any{"doc": map[string]any{}})
	mustCode(t, resp, body, http.StatusForbidden, "SCOPE_REQUIRED")
}

func TestV1PutPreferencesTooLarge(t *testing.T) {
	f := newMeFix(t)
	f.prefCode.Store(18005)
	f.prefHTTP.Store(413)
	resp, body := f.call(t, http.MethodPut, prefsPath, "/me/preferences", "sess-alice", "", nil,
		map[string]any{"doc": map[string]any{}})
	mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	fieldErr(t, body, "pointer", "/doc", "TOO_LONG")
}

func TestV1PutPreferencesUpstreamRejectsIfMatch(t *testing.T) {
	f := newMeFix(t)
	f.prefCode.Store(7)
	f.prefHTTP.Store(400)
	resp, body := f.call(t, http.MethodPut, prefsPath, "/me/preferences", "sess-alice", "", ifMatch(`"1"`),
		map[string]any{"doc": map[string]any{}})
	mustCode(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
	fieldErr(t, body, "header", "If-Match", "INVALID_FORMAT")
}

func TestV1PutPreferencesMalformedIfMatch(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.call(t, http.MethodPut, prefsPath, "/me/preferences", "sess-alice", "", ifMatch("*"),
		map[string]any{"doc": map[string]any{}})
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("If-Match * → %d %+v, want a 4xx validation problem", resp.StatusCode, body)
	}
}

func TestV1PutPreferencesOtherFailureIs503(t *testing.T) {
	f := newMeFix(t)
	f.prefCode.Store(1)
	f.prefHTTP.Store(500)
	resp, body := f.call(t, http.MethodPut, prefsPath, "/me/preferences", "sess-alice", "", nil,
		map[string]any{"doc": map[string]any{}})
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1PutNsfwDisplay(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.call(t, http.MethodPut, mePath+"/nsfw-display", "/me/nsfw-display", "sess-alice", "", nil,
		map[string]any{"nsfw_display": "blur"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put nsfw %d %+v", resp.StatusCode, body)
	}
	if body["object"] != "nsfw_display" || body["nsfw_display"] != "blur" {
		t.Fatalf("nsfw %+v", body)
	}
	if _, ok := body["adult_confirmed"]; ok {
		t.Fatalf("adult_confirmed is retired and must not be sent: %+v", body)
	}
}

func TestV1PutNsfwDisplayUnknownValue(t *testing.T) {
	f := newMeFix(t)
	resp, body := f.call(t, http.MethodPut, mePath+"/nsfw-display", "/me/nsfw-display", "sess-alice", "", nil,
		map[string]any{"nsfw_display": "sometimes"})
	mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
}

func TestV1PutNsfwDisplayUpstreamFailureIs503(t *testing.T) {
	f := newMeFix(t)
	f.nsfwFail.Store(true)
	resp, body := f.call(t, http.MethodPut, mePath+"/nsfw-display", "/me/nsfw-display", "sess-alice", "", nil,
		map[string]any{"nsfw_display": "hide"})
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1PutNsfwDisplayScopeMissing(t *testing.T) {
	f := newMeFix(t)
	f.nsfwNoScope.Store(true)
	resp, body := f.call(t, http.MethodPut, mePath+"/nsfw-display", "/me/nsfw-display", "sess-alice", "", nil,
		map[string]any{"nsfw_display": "show"})
	mustCode(t, resp, body, http.StatusForbidden, "SCOPE_REQUIRED")
}
