package agent_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kun-galgame-api/internal/agent"
	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

const (
	quotaExhausted = `{"type":"about:blank","title":"Payment Required","status":402,"detail":"free units exhausted","code":"FREE_UNITS_EXHAUSTED"}`
	persona        = `{"name":"琥珀","greetings":["你好"],"web_search_available":true}`
)

type resolver struct{ id middleware.Identity }

func (r resolver) ResolveIdentity(fiber.Ctx) middleware.Identity { return r.id }

var signedIn = resolver{middleware.Identity{
	Outcome:     middleware.IdentitySessionOK,
	User:        &middleware.UserInfo{ID: 7},
	AccessToken: "user-token",
}}

type seen struct {
	calls                                                          int
	method, path, query, auth, contentType, idem, cookie, ip, last string
	body                                                           string
}

func relayApp(t *testing.T, r apiv1.IdentityResolver, upstream http.HandlerFunc) (*fiber.App, *seen) {
	t.Helper()
	s := &seen{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		b, _ := io.ReadAll(req.Body)
		*s = seen{s.calls + 1, req.Method, req.URL.Path, req.URL.RawQuery, req.Header.Get("Authorization"),
			req.Header.Get("Content-Type"), req.Header.Get("Idempotency-Key"), req.Header.Get("Cookie"),
			req.Header.Get("CF-Connecting-IP"), req.Header.Get("Last-Event-ID"), string(b)}
		upstream(w, req)
	}))
	t.Cleanup(srv.Close)
	return appWith(r, srv.URL+"/"), s
}

func appWith(r apiv1.IdentityResolver, base string) *fiber.App {
	app := fiber.New()
	apiv1.Setup(app, apiv1.Deps{Resolver: r, Mount: agent.NewRelay(base, r).Mount})
	return app
}

func do(t *testing.T, app *fiber.App, req *http.Request) (*http.Response, string) {
	t.Helper()
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
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

func TestAgentRelayForwardsUnderTheCallersToken(t *testing.T) {
	app, s := relayApp(t, signedIn, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
		}
		_, _ = w.Write([]byte(persona))
	})

	resp, body := do(t, app, httptest.NewRequest(http.MethodGet, "/api/v1/agent/conversations?starred=false&q=%E7%8C%AB", nil))
	if resp.StatusCode != http.StatusOK || body != persona {
		t.Fatalf("read: %d %s", resp.StatusCode, body)
	}
	if s.path != "/v2/agent/conversations" || s.query != "starred=false&q=%E7%8C%AB" || s.auth != "Bearer user-token" {
		t.Fatalf("upstream saw %+v", s)
	}
	if resp.Header.Get("Cache-Control") != "no-store" || resp.Header.Get("X-Request-ID") == "" {
		t.Fatalf("headers: %v", resp.Header)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/conversations/12/messages", strings.NewReader(`{"parts":[]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "k-1")
	req.Header.Set("Cookie", "kungal_session=secret")
	resp, _ = do(t, app, req)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("send: %d", resp.StatusCode)
	}
	if s.method != http.MethodPost || s.body != `{"parts":[]}` || s.contentType != "application/json" ||
		s.idem != "k-1" || s.cookie != "" {
		t.Fatalf("upstream saw %+v", s)
	}
}

func TestAgentRelayPassesProblemsThrough(t *testing.T) {
	app, _ := relayApp(t, signedIn, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(quotaExhausted))
	})
	for _, path := range []string{"/api/v1/agent/conversations/1/messages", "/api/v1/agent/runs/9/stream"} {
		method := http.MethodPost
		if strings.HasSuffix(path, "stream") {
			method = http.MethodGet
		}
		resp, body := do(t, app, httptest.NewRequest(method, path, nil))
		if resp.StatusCode != http.StatusPaymentRequired || body != quotaExhausted ||
			resp.Header.Get("Content-Type") != "application/problem+json" {
			t.Fatalf("%s: %d %s %v", path, resp.StatusCode, body, resp.Header)
		}
	}
}

func TestAgentRelayStreamsTheRun(t *testing.T) {
	const frames = "id: 1\ndata: {\"type\":\"start\"}\n\nid: 2\ndata: {\"type\":\"finish\"}\n\n"
	app, s := relayApp(t, signedIn, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Vercel-Ai-Ui-Message-Stream", "v1")
		for _, f := range strings.SplitAfter(frames, "\n\n") {
			_, _ = w.Write([]byte(f))
			w.(http.Flusher).Flush()
		}
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/runs/41/stream", nil)
	req.Header.Set("Last-Event-ID", "0")
	resp, body := do(t, app, req)
	if resp.StatusCode != http.StatusOK || body != frames {
		t.Fatalf("stream: %d %q", resp.StatusCode, body)
	}
	if s.path != "/v2/agent/runs/41/stream" || s.last != "0" {
		t.Fatalf("upstream saw %+v", s)
	}
	if resp.Header.Get("Content-Type") != "text/event-stream" || resp.Header.Get("X-Accel-Buffering") != "no" ||
		resp.Header.Get("Cache-Control") != "no-cache, no-transform" || resp.Header.Get("X-Vercel-Ai-Ui-Message-Stream") != "v1" {
		t.Fatalf("headers: %v", resp.Header)
	}
}

func TestAgentRelayKeepsAFinishedRunsNoContent(t *testing.T) {
	app, _ := relayApp(t, signedIn, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	resp, body := do(t, app, httptest.NewRequest(http.MethodGet, "/api/v1/agent/runs/41/stream", nil))
	if resp.StatusCode != http.StatusNoContent || body != "" {
		t.Fatalf("finished run: %d %q", resp.StatusCode, body)
	}
}

func TestAgentRelaySandboxesAnUpload(t *testing.T) {
	app, s := relayApp(t, signedIn, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "private, max-age=3600")
		_, _ = w.Write([]byte("<svg/>"))
	})
	resp, body := do(t, app, httptest.NewRequest(http.MethodGet, "/api/v1/agent/uploads/up_1", nil))
	if resp.StatusCode != http.StatusOK || body != "<svg/>" || s.path != "/v2/agent/uploads/up_1" {
		t.Fatalf("upload: %d %q %+v", resp.StatusCode, body, s)
	}
	if resp.Header.Get("Content-Security-Policy") == "" || resp.Header.Get("X-Content-Type-Options") != "nosniff" ||
		resp.Header.Get("Content-Type") != "image/svg+xml" || resp.Header.Get("Cache-Control") != "private, max-age=3600" {
		t.Fatalf("headers: %v", resp.Header)
	}
}

func TestAgentRelayRefusesAnonymousBeforeUpstream(t *testing.T) {
	app, s := relayApp(t, resolver{middleware.Identity{Outcome: middleware.IdentityAnonymous}},
		func(http.ResponseWriter, *http.Request) {})
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/agent/persona"},
		{http.MethodPost, "/api/v1/agent/conversations"},
		{http.MethodPut, "/api/v1/agent/messages/1/feedback"},
		{http.MethodPatch, "/api/v1/agent/conversations/1"},
		{http.MethodDelete, "/api/v1/agent/conversations/1"},
		{http.MethodGet, "/api/v1/agent/runs/1/stream"},
		{http.MethodGet, "/api/v1/agent/uploads/1"},
	} {
		resp, body := do(t, app, httptest.NewRequest(c.method, c.path, nil))
		if resp.StatusCode != http.StatusUnauthorized || problemCode(t, body) != "MISSING_CREDENTIAL" {
			t.Fatalf("%s %s anonymous: %d %s", c.method, c.path, resp.StatusCode, body)
		}
	}
	if s.calls != 0 {
		t.Fatal("an anonymous call reached the agent")
	}
}

func TestAgentRelayRefusesTraversal(t *testing.T) {
	app, s := relayApp(t, signedIn, func(http.ResponseWriter, *http.Request) {})
	for _, path := range []string{
		"/api/v1/agent/conversations/..%2F..%2Fadmin",
		"/api/v1/agent/%2e%2e/%2e%2e/admin",
		"/api/v1/agent/persona%3Fx",
		"/api/v1/agent/",
		"/api/v1/agent/uploads/..%2Fpersona",
	} {
		resp, body := do(t, app, httptest.NewRequest(http.MethodGet, path, nil))
		if resp.StatusCode != http.StatusNotFound || problemCode(t, body) != "NOT_FOUND" {
			t.Fatalf("%s: %d %s", path, resp.StatusCode, body)
		}
	}
	if s.calls != 0 {
		t.Fatal("a traversal reached the agent")
	}
}

func TestAgentRelayAnswersUnavailable(t *testing.T) {
	for name, base := range map[string]string{"unset": "", "unreachable": "http://127.0.0.1:1"} {
		for _, path := range []string{"/api/v1/agent/persona", "/api/v1/agent/runs/1/stream"} {
			resp, body := do(t, appWith(signedIn, base), httptest.NewRequest(http.MethodGet, path, nil))
			if resp.StatusCode != http.StatusServiceUnavailable || problemCode(t, body) != "SERVICE_UNAVAILABLE" {
				t.Fatalf("%s %s: %d %s", name, path, resp.StatusCode, body)
			}
		}
	}
}
