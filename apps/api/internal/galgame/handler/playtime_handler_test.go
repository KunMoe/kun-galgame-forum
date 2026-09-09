package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"

	"github.com/gofiber/fiber/v3"
)

type fakePlaytimeFace struct {
	mu       sync.Mutex
	requests []recordedRequest
	status   int
	body     string
	// What the self-read reports back after the write. A second application of
	// the same user can hold a larger number than the one just written.
	foldMinutes int
	workState   string
}

func (f *fakePlaytimeFace) server(t *testing.T) *httptest.Server {
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
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/work-states") {
			if r.Method == http.MethodGet {
				if f.workState != "" {
					_, _ = w.Write([]byte(f.workState))
					return
				}
				w.WriteHeader(http.StatusNotFound)
				return
			}
			state, _ := body["state"].(string)
			comp := "null"
			if v, ok := body["completion"]; ok {
				b, _ := json.Marshal(v)
				comp = string(b)
			}
			_, _ = w.Write([]byte(`{"object":"work_state","work_id":"1000","state":"` + state +
				`","completion":` + comp + `,"created_at":"2026-08-20T00:00:00Z","updated_at":"2026-08-20T00:00:00Z"}`))
			return
		}
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"work_id":1000,"minutes":` +
				strconv.Itoa(f.foldMinutes) + `}}`))
			return
		}
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"work_id":1000,"minutes":720}}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func playtimeTestApp(t *testing.T, catalogURL string, user *middleware.UserInfo) *fiber.App {
	t.Helper()
	cc := catalogclient.New(catalogclient.Config{BaseURL: catalogURL})
	gc := client.New(fakeGalgame(t).URL, "nm_test", "")
	h := NewPlaytimeHandler(service.NewPlaytimeService(nil, gc, cc, "forum"))

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
	authed.Put("/galgame/:gid/playtime", h.Report)
	return app
}

func TestReportPlaytimeTravelsAsTheUserAndAnswersTheFold(t *testing.T) {
	fake := &fakePlaytimeFace{foldMinutes: 900}
	app := playtimeTestApp(t, fake.server(t).URL, plainUser)

	status, raw := doJSON(t, app, "PUT", "/api/galgame/1/playtime",
		`{"minutes":720,"status":"done"}`)
	if status != http.StatusOK {
		t.Fatalf("report: status = %d body %s", status, raw)
	}

	if len(fake.requests) != 4 {
		t.Fatalf("want work-state read, playtime write, work-state write, playtime fold, got %+v", fake.requests)
	}
	if fake.requests[0].Method != http.MethodGet || fake.requests[0].Path != "/v2/me/work-states/1000" {
		t.Errorf("work-state read = %s %s", fake.requests[0].Method, fake.requests[0].Path)
	}
	write := fake.requests[1]
	if write.Method != http.MethodPut || write.Path != "/v2/me/playtimes/1000" {
		t.Errorf("write went to %s %s, want PUT /v2/me/playtimes/1000", write.Method, write.Path)
	}
	if write.Auth != "Bearer user-jwt" {
		t.Errorf("auth = %q, want the user's own bearer, never the s2s pair", write.Auth)
	}
	if write.Body["minutes"] != float64(720) {
		t.Errorf("body = %v", write.Body)
	}
	if _, ok := write.Body["status"]; ok {
		t.Errorf("playtime PUT must not send status, got %v", write.Body)
	}
	ws := fake.requests[2]
	if ws.Method != http.MethodPut || ws.Path != "/v2/me/work-states/1000" {
		t.Errorf("work-state write = %s %s", ws.Method, ws.Path)
	}
	if ws.Body["state"] != "done" {
		t.Errorf("work-state body = %v", ws.Body)
	}
	if _, ok := ws.Body["completion"]; ok {
		t.Errorf("no prior completion, PUT must omit it, got %v", ws.Body)
	}
	read := fake.requests[3]
	if read.Method != http.MethodGet || read.Path != "/v2/me/playtimes/1000" {
		t.Errorf("fold read = %s %s", read.Method, read.Path)
	}

	var env struct {
		Data struct {
			Minutes int    `json:"minutes"`
			Status  string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("decode: %v (%s)", err, raw)
	}
	// 900, not the 720 just written: another app of the same user holds more,
	// and echoing our own figure would contradict the number on the page.
	if env.Data.Minutes != 900 || env.Data.Status != "done" {
		t.Errorf("answered %+v, want the folded 900 with stored state done", env.Data)
	}
}

func TestReportPlaytimePreservesUpstreamCompletion(t *testing.T) {
	fake := &fakePlaytimeFace{
		foldMinutes: 720,
		workState:   `{"object":"work_state","work_id":"1000","state":"doing","completion":"all","created_at":"2026-08-01T00:00:00Z","updated_at":"2026-08-01T00:00:00Z"}`,
	}
	app := playtimeTestApp(t, fake.server(t).URL, plainUser)

	status, raw := doJSON(t, app, "PUT", "/api/galgame/1/playtime",
		`{"minutes":720,"status":"done"}`)
	if status != http.StatusOK {
		t.Fatalf("report: status = %d body %s", status, raw)
	}
	var put map[string]any
	for _, req := range fake.requests {
		if req.Method == http.MethodPut && strings.Contains(req.Path, "/work-states") {
			put = req.Body
		}
	}
	if put["state"] != "done" || put["completion"] != "all" {
		t.Errorf("work-state PUT = %v, want state=done and completion=all preserved", put)
	}
}

func TestReportPlaytimeRejectsBadInputWithoutCallingCatalog(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"negative", `{"minutes":-1}`},
		{"over the ceiling", `{"minutes":60001}`},
		{"unknown status", `{"minutes":600,"status":"finished_all"}`},
		{"wish", `{"minutes":600,"status":"wish"}`},
		{"old word", `{"minutes":600,"status":"playing"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakePlaytimeFace{}
			app := playtimeTestApp(t, fake.server(t).URL, plainUser)

			status, raw := doJSON(t, app, "PUT", "/api/galgame/1/playtime", tc.body)
			if status != http.StatusBadRequest {
				t.Fatalf("status = %d body %s", status, raw)
			}
			if len(fake.requests) != 0 {
				t.Fatalf("catalog was called anyway: %+v", fake.requests)
			}
		})
	}
}

func TestReportPlaytimeAsksForReauthOnAScopeDenial(t *testing.T) {
	fake := &fakePlaytimeFace{
		status: http.StatusForbidden,
		body:   `{"code":233,"message":"the access token is missing the playtime:write scope"}`,
	}
	app := playtimeTestApp(t, fake.server(t).URL, plainUser)

	status, raw := doJSON(t, app, "PUT", "/api/galgame/1/playtime", `{"minutes":720,"status":"done"}`)
	if status != http.StatusForbidden {
		t.Fatalf("status = %d body %s", status, raw)
	}
	var env struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(raw, &env)
	if env.Code != errors.CodeReauthRequired {
		t.Errorf("code = %d, want CodeReauthRequired so the client offers a re-login", env.Code)
	}
	if !strings.Contains(env.Message, "重新登录") {
		t.Errorf("message = %q", env.Message)
	}
}
