package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"

	"github.com/gofiber/fiber/v3"
)

type fakeVoteFace struct {
	mu       sync.Mutex
	requests []recordedRequest
	status   int
	body     string
}

func (f *fakeVoteFace) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &body)
		}
		f.mu.Lock()
		f.requests = append(f.requests, recordedRequest{
			Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: body,
			Auth: r.Header.Get("Authorization"),
		})
		f.mu.Unlock()
		if f.status != 0 {
			w.WriteHeader(f.status)
			_, _ = w.Write([]byte(f.body))
			return
		}
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"cover_id":88,"vote_count":4,"voted":true}}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func voteTestApp(t *testing.T, catalogURL string, user *middleware.UserInfo) *fiber.App {
	t.Helper()
	cc := catalogclient.New(catalogclient.Config{BaseURL: catalogURL})
	h := NewCoverVoteHandler(cc, client.New(fakeGalgame(t).URL, "nm_test", ""))

	app := fiber.New()
	api := app.Group("/api")
	authed := api.Group("", func(c fiber.Ctx) error {
		if user == nil {
			return c.Status(401).JSON(fiber.Map{"code": 205, "message": "用户登录失效"})
		}
		c.Locals(string(middleware.UserInfoKey), user)
		c.Locals(string(middleware.OAuthAccessTokenKey), "user-jwt")
		return c.Next()
	})
	authed.Put("/galgame/:id/cover/:coverId/vote", h.Vote)
	authed.Delete("/galgame/:id/cover/:coverId/vote", h.Unvote)
	return app
}

func TestCoverVoteTravelsAsTheUser(t *testing.T) {
	fake := &fakeVoteFace{}
	app := voteTestApp(t, fake.server(t).URL, plainUser)

	status, raw := doJSON(t, app, "PUT", "/api/galgame/1/cover/88/vote", "")
	if status != http.StatusOK {
		t.Fatalf("vote: status = %d body %s", status, raw)
	}
	if len(fake.requests) != 1 {
		t.Fatalf("expected exactly one catalog call, got %+v", fake.requests)
	}
	req := fake.requests[0]
	if req.Method != "PUT" || req.Path != "/v2/me/cover-votes/88" {
		t.Fatalf("vote hit %s %s", req.Method, req.Path)
	}
	if req.Auth != "Bearer user-jwt" {
		t.Fatalf("auth = %q, want the session bearer", req.Auth)
	}
	if !strings.Contains(string(raw), `"vote_count":4`) {
		t.Fatalf("tally not returned to the browser: %s", raw)
	}
}

func TestCoverUnvoteUsesDelete(t *testing.T) {
	fake := &fakeVoteFace{status: http.StatusOK,
		body: `{"code":0,"message":"ok","data":{"cover_id":88,"vote_count":3,"voted":false}}`}
	app := voteTestApp(t, fake.server(t).URL, plainUser)

	status, raw := doJSON(t, app, "DELETE", "/api/galgame/1/cover/88/vote", "")
	if status != http.StatusOK {
		t.Fatalf("unvote: status = %d body %s", status, raw)
	}
	if fake.requests[0].Method != "DELETE" {
		t.Fatalf("unvote used %s", fake.requests[0].Method)
	}
	if !strings.Contains(string(raw), `"voted":false`) {
		t.Fatalf("withdrawal state not returned: %s", raw)
	}
}

func TestCoverVoteStaleGrantAsksForReauth(t *testing.T) {
	fake := &fakeVoteFace{status: http.StatusForbidden,
		body: `{"code":233,"message":"missing required scope: catalog:edit"}`}
	app := voteTestApp(t, fake.server(t).URL, plainUser)

	status, raw := doJSON(t, app, "PUT", "/api/galgame/1/cover/88/vote", "")
	if status != http.StatusForbidden {
		t.Fatalf("status = %d body %s, want 403", status, raw)
	}
	var env struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("bad envelope %s", raw)
	}
	if env.Code != 235 {
		t.Fatalf("code = %d, want 235 (re-login required); body %s", env.Code, raw)
	}
}

func TestCoverVoteErrorPassThrough(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		body       string
		wantStatus int
		wantCode   int
	}{
		{"unknown cover", http.StatusNotFound, `{"code":233,"message":"no such cover"}`, 404, 233},
		{"dead token", http.StatusUnauthorized, `{"code":205,"message":"bad token"}`, 401, 205},
		{"client problem", http.StatusForbidden, `{"code":233,"message":"client is not bound to a catalog site"}`, 403, 233},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeVoteFace{status: tc.status, body: tc.body}
			app := voteTestApp(t, fake.server(t).URL, plainUser)
			status, raw := doJSON(t, app, "PUT", "/api/galgame/1/cover/88/vote", "")
			if status != tc.wantStatus {
				t.Fatalf("status = %d body %s, want %d", status, raw, tc.wantStatus)
			}
			var env struct {
				Code int `json:"code"`
			}
			_ = json.Unmarshal(raw, &env)
			if env.Code != tc.wantCode {
				t.Fatalf("code = %d, want %d (body %s)", env.Code, tc.wantCode, raw)
			}
		})
	}
}

func TestCoverVoteRequiresLogin(t *testing.T) {
	app := voteTestApp(t, "http://127.0.0.1:1", nil)
	if status, raw := doJSON(t, app, "PUT", "/api/galgame/1/cover/88/vote", ""); status != http.StatusUnauthorized {
		t.Fatalf("status = %d body %s, want 401", status, raw)
	}
}

func TestCoverVoteDegradesWhenUnconfigured(t *testing.T) {
	app := voteTestApp(t, "", plainUser)
	if status, raw := doJSON(t, app, "PUT", "/api/galgame/1/cover/88/vote", ""); status != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body %s, want 503", status, raw)
	}
}
