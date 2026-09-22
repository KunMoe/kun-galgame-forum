package oauth

import (
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"kun-galgame-api/pkg/config"
)

type capturedRequest struct {
	method  string
	path    string
	ifMatch string
	body    string
}

func prefsServer(t *testing.T, status int, body string) (*Client, *capturedRequest) {
	t.Helper()
	got := &capturedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.method, got.path, got.ifMatch = r.Method, r.URL.Path, r.Header.Get("If-Match")
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		got.body = string(buf)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if _, err := w.Write([]byte(body)); err != nil {
			t.Errorf("write: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	return NewClient(config.OAuthConfig{ServerURL: srv.URL, ClientID: "kungal-forum"}), got
}

func TestPutAuthMeNSFW(t *testing.T) {
	c, got := prefsServer(t, 200,
		`{"code":0,"data":{"nsfw_display":"show","adult_confirmed_at":"2026-09-22T08:30:00Z"}}`)

	state, err := c.PutAuthMeNSFW("tok", "show")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.method != "PUT" || got.path != "/auth/me/nsfw" {
		t.Fatalf("called %s %s", got.method, got.path)
	}
	if got.body != `{"nsfw_display":"show"}` {
		t.Fatalf("body = %s", got.body)
	}
	if state.NSFWDisplay != "show" || state.AdultConfirmedAt == nil {
		t.Fatalf("state = %+v", state)
	}
}

// Setting blur or show before the account has attested is the one refusal the
// site cannot resolve itself, so it has to survive the proxy intact.
func TestPutAuthMeNSFWBeforeAttestation(t *testing.T) {
	c, _ := prefsServer(t, 400, `{"code":18008,"message":"请先完成年龄确认"}`)

	_, err := c.PutAuthMeNSFW("tok", "blur")
	var oe *Error
	if !stderrors.As(err, &oe) || oe.Code != CodeAdultConfirmationNeeded {
		t.Fatalf("err = %v, want code %d", err, CodeAdultConfirmationNeeded)
	}
}

func TestPreferencesNamespaceIsAlwaysTheClientID(t *testing.T) {
	c, got := prefsServer(t, 200, `{"code":0,"data":{"namespace":"kungal-forum","doc":{},"version":0}}`)

	if _, err := c.GetPreferences("tok"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.path != "/auth/me/preferences/kungal-forum" {
		t.Fatalf("namespace came from somewhere else: %s", got.path)
	}
	if got.ifMatch != "" {
		t.Fatalf("a read must not send If-Match, got %q", got.ifMatch)
	}
}

func TestPutPreferencesForwardsIfMatch(t *testing.T) {
	c, got := prefsServer(t, 200, `{"code":0,"data":{"namespace":"kungal-forum","doc":{"a":1},"version":3}}`)

	doc := json.RawMessage(`{"a":1}`)
	if _, err := c.PutPreferences("tok", doc, `"2"`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ifMatch != `"2"` {
		t.Fatalf("If-Match = %q", got.ifMatch)
	}
	if got.body != `{"doc":{"a":1}}` {
		t.Fatalf("body = %s", got.body)
	}
	if got.path != "/auth/me/preferences/kungal-forum" {
		t.Fatalf("path = %s", got.path)
	}
}

func TestPutPreferencesOmitsAnEmptyIfMatch(t *testing.T) {
	c, got := prefsServer(t, 200, `{"code":0,"data":{"doc":{},"version":1}}`)

	if _, err := c.PutPreferences("tok", json.RawMessage(`{}`), ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ifMatch != "" {
		t.Fatalf("an empty If-Match must not be sent as a header, got %q", got.ifMatch)
	}
}

func TestPreferencesErrors(t *testing.T) {
	for _, tc := range []struct {
		why    string
		status int
		body   string
		want   int
	}{
		{"no preferences scope", 403, `{"code":18001,"message":"缺少 scope"}`, CodePreferencesScopeMissing},
		{"stale If-Match", 412, `{"code":18006,"message":"版本不符"}`, CodePreferencesConflict},
	} {
		c, _ := prefsServer(t, tc.status, tc.body)
		_, err := c.PutPreferences("tok", json.RawMessage(`{}`), `"1"`)
		var oe *Error
		if !stderrors.As(err, &oe) || oe.Code != tc.want {
			t.Errorf("%s: err = %v, want code %d", tc.why, err, tc.want)
		}
	}
}
