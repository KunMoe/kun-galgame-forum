package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/problem"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

func serve(t *testing.T) *App {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: miniredis.RunT(t).Addr(), MaxRetries: 0})
	t.Cleanup(func() { _ = rdb.Close() })
	a := &App{Fiber: newFiber(), Config: testConfig(), Redis: rdb, Authn: middleware.NewAuthenticator(rdb, nil, nil)}
	a.Fiber.Get("/api/_test/panic", func(fiber.Ctx) error { panic("legacy boom") })
	a.Fiber.Get("/api/v1/_test/panic", func(fiber.Ctx) error { panic("v1 boom") })
	a.setupRoutes()
	return a
}

func get(t *testing.T, a *App, method, path string) (*http.Response, []byte) {
	t.Helper()
	return send(t, a, httptest.NewRequest(method, path, nil))
}

func send(t *testing.T, a *App, req *http.Request) (*http.Response, []byte) {
	t.Helper()
	resp, err := a.Fiber.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp, body
}

func TestV1ErrorsAreProblems(t *testing.T) {
	a := serve(t)
	for _, tc := range []struct {
		method, path string
		status       int
		code         string
	}{
		{http.MethodGet, "/api/v1/nope", http.StatusNotFound, problem.CodeNotFound},
		{http.MethodPost, "/api/v1/problems", http.StatusMethodNotAllowed, problem.CodeMethodNotAllowed},
		{http.MethodGet, "/api/v1/_test/panic", http.StatusInternalServerError, problem.CodeInternalError},
	} {
		resp, body := get(t, a, tc.method, tc.path)
		var p struct {
			Code      string `json:"code"`
			RequestID string `json:"request_id"`
		}
		if err := json.Unmarshal(body, &p); err != nil {
			t.Fatalf("%s %s: %v: %s", tc.method, tc.path, err, body)
		}
		if resp.StatusCode != tc.status || p.Code != tc.code {
			t.Errorf("%s %s = %d %s, want %d %s", tc.method, tc.path, resp.StatusCode, p.Code, tc.status, tc.code)
		}
		if ct := resp.Header.Get("Content-Type"); ct != problem.ContentType {
			t.Errorf("%s %s Content-Type %q", tc.method, tc.path, ct)
		}
		if resp.Header.Get("Cache-Control") != "no-store" {
			t.Errorf("%s %s Cache-Control %q", tc.method, tc.path, resp.Header.Get("Cache-Control"))
		}
		if id := resp.Header.Get(problem.HeaderRequestID); id == "" || id != p.RequestID {
			t.Errorf("%s %s request id header %q body %q", tc.method, tc.path, id, p.RequestID)
		}
	}
}

func TestV1WrongMethodListsTheAllowedOnes(t *testing.T) {
	resp, _ := get(t, serve(t), http.MethodDelete, "/api/v1/problems/reasons")
	if resp.StatusCode != http.StatusMethodNotAllowed || resp.Header.Get("Allow") != "GET, HEAD" {
		t.Errorf("status %d Allow %q, want 405 GET, HEAD", resp.StatusCode, resp.Header.Get("Allow"))
	}
}

func TestV1HeadAnswersLikeGet(t *testing.T) {
	resp, body := get(t, serve(t), http.MethodHead, "/api/v1/problems")
	if resp.StatusCode != http.StatusOK || len(body) != 0 || resp.Header.Get(problem.HeaderRequestID) == "" {
		t.Errorf("HEAD = %d, %d body bytes, request id %q", resp.StatusCode, len(body), resp.Header.Get(problem.HeaderRequestID))
	}
}

func TestLegacyErrorsKeepTheEnvelope(t *testing.T) {
	var logs bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(prev)

	a := serve(t)
	session, err := json.Marshal(middleware.SessionData{
		UserInfo:         middleware.UserInfo{ID: 1, Name: "n"},
		OAuthAccessToken: "access",
		OAuthExpiresAt:   time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Redis.Set(context.Background(), middleware.SessionKey("sess"), session, middleware.SessionTTL).Err(); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name, path string
		signedIn   bool
		status     int
		want       string
		logged     bool
	}{
		{"retired route, anonymous", "/api/nope", false, http.StatusUnauthorized, "", false},
		{"retired route, signed in", "/api/nope", true, http.StatusNotFound, `{"code":233,"message":"页面版本已过期，请刷新页面后重试"}`, false},
		{"panic", "/api/_test/panic", false, http.StatusInternalServerError, `{"code":233,"message":"服务器内部错误"}`, true},
	} {
		logs.Reset()
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		if tc.signedIn {
			req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "sess"})
		}
		resp, body := send(t, a, req)
		if resp.StatusCode != tc.status || (tc.want != "" && string(body) != tc.want) {
			t.Errorf("%s: %d %s, want %d %s", tc.name, resp.StatusCode, body, tc.status, tc.want)
		}
		if resp.Header.Get(problem.HeaderRequestID) != "" || resp.Header.Get("Cache-Control") != "" {
			t.Errorf("%s: gained v1 headers", tc.name)
		}
		if got := strings.Contains(logs.String(), "level=ERROR"); got != tc.logged {
			t.Errorf("%s: logged an ERROR = %v, want %v: %s", tc.name, got, tc.logged, logs.String())
		}
	}
}

func TestCORSCarriesTheV1Headers(t *testing.T) {
	a := serve(t)
	pre := httptest.NewRequest(http.MethodOptions, "/api/v1/problems", nil)
	pre.Header.Set("Origin", "https://www.kungal.com")
	pre.Header.Set("Access-Control-Request-Method", http.MethodPost)
	pre.Header.Set("Access-Control-Request-Headers", "idempotency-key,x-request-id")
	resp, err := a.Fiber.Test(pre)
	if err != nil {
		t.Fatal(err)
	}
	allowed := strings.ToLower(resp.Header.Get("Access-Control-Allow-Headers"))
	for _, h := range []string{"idempotency-key", "x-request-id"} {
		if !strings.Contains(allowed, h) {
			t.Errorf("preflight Access-Control-Allow-Headers %q lacks %s", allowed, h)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/problems", nil)
	req.Header.Set("Origin", "https://www.kungal.com")
	resp, err = a.Fiber.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	exposed := resp.Header.Get("Access-Control-Expose-Headers")
	for _, h := range []string{"X-Request-ID", "Idempotency-Replayed", "Retry-After"} {
		if !strings.Contains(exposed, h) {
			t.Errorf("Access-Control-Expose-Headers %q lacks %s", exposed, h)
		}
	}
}
