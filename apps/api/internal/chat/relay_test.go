package chat_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/chat"
	"kun-galgame-api/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

// Captured from the chat service (infra origin/main, 2026-09-27); a stub
// written by hand once agreed with a client that was wrong in the same way.
const (
	scopeRequired = `{"type":"https://developer.nextmoe.dev/problems/platform/scope-required","title":"Scope required","status":403,"detail":"this operation requires the chat:read scope.","instance":"/v2/chat/state","code":"SCOPE_REQUIRED","request_id":"req_01M3HRAYMTW4BQAV0XXY4H8YAE","errors":[]}`
	chatState     = `{"object":"chat_state","last_update_seq":7,"unread_conversation_count":1,"unread_message_count":2,"request_count":1}`
)

type resolver struct{ id middleware.Identity }

func (r resolver) ResolveIdentity(fiber.Ctx) middleware.Identity { return r.id }

var signedIn = resolver{middleware.Identity{
	Outcome:     middleware.IdentitySessionOK,
	User:        &middleware.UserInfo{ID: 7},
	AccessToken: "user-token",
}}

type seen struct {
	calls                                                      int
	method, path, query, auth, contentType, idem, cookie, body string
}

func relayApp(t *testing.T, r apiv1.IdentityResolver, upstream http.HandlerFunc) (*fiber.App, *seen) {
	t.Helper()
	s := &seen{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		b, _ := io.ReadAll(req.Body)
		*s = seen{s.calls + 1, req.Method, req.URL.Path, req.URL.RawQuery, req.Header.Get("Authorization"),
			req.Header.Get("Content-Type"), req.Header.Get("Idempotency-Key"), req.Header.Get("Cookie"), string(b)}
		upstream(w, req)
	}))
	t.Cleanup(srv.Close)
	return appWith(r, srv.URL+"/"), s
}

func appWith(r apiv1.IdentityResolver, base string) *fiber.App {
	app := fiber.New()
	apiv1.Setup(app, apiv1.Deps{Resolver: r, Mount: chat.NewRelay(base, r).Mount})
	return app
}

func do(t *testing.T, app *fiber.App, req *http.Request) (*http.Response, string) {
	t.Helper()
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

func problemCode(t *testing.T, body string) string {
	t.Helper()
	var p struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		t.Fatalf("not a problem: %s", body)
	}
	return p.Code
}

func TestChatRelayForwardsUnderTheCallersToken(t *testing.T) {
	app, s := relayApp(t, signedIn, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
		}
		_, _ = w.Write([]byte(chatState))
	})

	resp, body := do(t, app, httptest.NewRequest(http.MethodGet, "/api/v1/chat/conversations?folder=requests&limit=20", nil))
	if resp.StatusCode != http.StatusOK || body != chatState {
		t.Fatalf("read: %d %s", resp.StatusCode, body)
	}
	if s.path != "/v2/chat/conversations" || s.query != "folder=requests&limit=20" || s.auth != "Bearer user-token" {
		t.Fatalf("upstream saw %+v", s)
	}
	if resp.Header.Get("Content-Type") != "application/json" || resp.Header.Get("Cache-Control") != "no-store" ||
		resp.Header.Get("X-Request-ID") == "" {
		t.Fatalf("headers: %v", resp.Header)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/conversations/12/messages", strings.NewReader(`{"text":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "k-1")
	req.Header.Set("Cookie", "kungal_session=secret")
	resp, _ = do(t, app, req)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("send: %d", resp.StatusCode)
	}
	if s.method != http.MethodPost || s.body != `{"text":"hi"}` || s.contentType != "application/json" || s.idem != "k-1" ||
		s.cookie != "" {
		t.Fatalf("upstream saw %+v", s)
	}
}

func TestChatRelayKeepsNoContent(t *testing.T) {
	app, s := relayApp(t, signedIn, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	resp, body := do(t, app, httptest.NewRequest(http.MethodDelete, "/api/v1/chat/conversations/3/pins/9", nil))
	if resp.StatusCode != http.StatusNoContent || body != "" || s.path != "/v2/chat/conversations/3/pins/9" {
		t.Fatalf("unpin: %d %q %+v", resp.StatusCode, body, s)
	}
}

func TestChatRelayPassesProblemsThrough(t *testing.T) {
	app, _ := relayApp(t, signedIn, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(scopeRequired))
	})
	resp, body := do(t, app, httptest.NewRequest(http.MethodGet, "/api/v1/chat/state", nil))
	if resp.StatusCode != http.StatusForbidden || body != scopeRequired {
		t.Fatalf("problem: %d %s", resp.StatusCode, body)
	}
	if resp.Header.Get("Content-Type") != "application/problem+json" || resp.Header.Get("Retry-After") != "7" {
		t.Fatalf("headers: %v", resp.Header)
	}
}

func TestChatRelayRefusesAnonymousBeforeUpstream(t *testing.T) {
	app, s := relayApp(t, resolver{middleware.Identity{Outcome: middleware.IdentityAnonymous}},
		func(http.ResponseWriter, *http.Request) {})
	for _, m := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		resp, body := do(t, app, httptest.NewRequest(m, "/api/v1/chat/conversations/1/accept", nil))
		if resp.StatusCode != http.StatusUnauthorized || problemCode(t, body) != "MISSING_CREDENTIAL" {
			t.Fatalf("%s anonymous: %d %s", m, resp.StatusCode, body)
		}
	}
	if s.calls != 0 {
		t.Fatal("an anonymous call reached chat")
	}
}

func TestChatRelayRefusesTraversal(t *testing.T) {
	app, s := relayApp(t, signedIn, func(http.ResponseWriter, *http.Request) {})
	for _, path := range []string{
		"/api/v1/chat/conversations/..%2F..%2Ftrust%2Fcallback",
		"/api/v1/chat/%2e%2e/%2e%2e/trust/callback",
		"/api/v1/chat/state%3Fx",
		"/api/v1/chat/",
	} {
		resp, body := do(t, app, httptest.NewRequest(http.MethodGet, path, nil))
		if resp.StatusCode != http.StatusNotFound || problemCode(t, body) != "NOT_FOUND" {
			t.Fatalf("%s: %d %s", path, resp.StatusCode, body)
		}
	}
	if s.calls != 0 {
		t.Fatal("a traversal reached chat")
	}
}

func TestChatRelayAnswersUnavailable(t *testing.T) {
	for name, base := range map[string]string{"unset": "", "unreachable": "http://127.0.0.1:1"} {
		resp, body := do(t, appWith(signedIn, base), httptest.NewRequest(http.MethodGet, "/api/v1/chat/state", nil))
		if resp.StatusCode != http.StatusServiceUnavailable || problemCode(t, body) != "SERVICE_UNAVAILABLE" ||
			resp.Header.Get("Retry-After") == "" {
			t.Fatalf("%s: %d %s", name, resp.StatusCode, body)
		}
	}
}

func TestChatRelayLeavesOtherV1PathsToTheSpec(t *testing.T) {
	app, _ := relayApp(t, signedIn, func(http.ResponseWriter, *http.Request) {})
	resp, body := do(t, app, httptest.NewRequest(http.MethodGet, "/api/v1/chats/state", nil))
	if resp.StatusCode != http.StatusNotFound || problemCode(t, body) != "NOT_FOUND" {
		t.Fatalf("unmatched: %d %s", resp.StatusCode, body)
	}
}
