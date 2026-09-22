package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	oauthclient "kun-galgame-api/internal/user/oauth"
)

type prefsHarness struct {
	app      *fiber.App
	rdb      *redis.Client
	upstream *capturedUpstream
}

type capturedUpstream struct {
	method  string
	path    string
	ifMatch string
	body    string
}

const prefsSession = "sess-prefs"

func newPrefsHarness(t *testing.T, status int, body string) *prefsHarness {
	t.Helper()

	got := &capturedUpstream{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.method, got.path, got.ifMatch = r.Method, r.URL.Path, r.Header.Get("If-Match")
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read upstream body: %v", err)
		}
		got.body = string(raw)
		w.WriteHeader(status)
		if _, err := w.Write([]byte(body)); err != nil {
			t.Errorf("write: %v", err)
		}
	}))
	t.Cleanup(srv.Close)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: 0})
	session, err := json.Marshal(middleware.SessionData{
		UserInfo:         middleware.UserInfo{ID: 7, Name: "kun", NSFWDisplay: "blur"},
		OAuthAccessToken: "access-token",
	})
	if err != nil {
		t.Fatalf("marshal session: %v", err)
	}
	if err := rdb.Set(context.Background(), middleware.SessionKey(prefsSession), session, middleware.SessionTTL).Err(); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	h := NewContentPrefsHandler(
		oauthclient.NewClient(config.OAuthConfig{ServerURL: srv.URL, ClientID: "kungal-forum"}),
		userclient.New(userclient.Config{}),
		rdb,
	)

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		middleware.AttachIdentity(c, middleware.Identity{
			Outcome:     middleware.IdentitySessionOK,
			User:        &middleware.UserInfo{ID: 7},
			AccessToken: "access-token",
		})
		return c.Next()
	})
	app.Put("/user/nsfw", h.UpdateNSFWDisplay)
	app.Get("/user/preferences", h.GetPreferences)
	app.Put("/user/preferences", h.UpdatePreferences)

	return &prefsHarness{app: app, rdb: rdb, upstream: got}
}

func (h *prefsHarness) do(t *testing.T, method, path, body string, headers map[string]string) (int, map[string]any, *http.Response) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: prefsSession})
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := h.app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	var envelope map[string]any
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("unmarshal %s: %v", raw, err)
	}
	return resp.StatusCode, envelope, resp
}

func TestUpdateNSFWDisplayWritesTheSessionThrough(t *testing.T) {
	h := newPrefsHarness(t, 200,
		`{"code":0,"data":{"nsfw_display":"show","adult_confirmed_at":"2026-09-22T08:30:00Z"}}`)

	status, envelope, resp := h.do(t, "PUT", "/user/nsfw", `{"nsfw_display":"show"}`, nil)
	if status != 200 {
		t.Fatalf("status %d: %v", status, envelope)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control = %q; Cloudflare will extend anything else", cc)
	}

	data, _ := envelope["data"].(map[string]any)
	if data["nsfw_display"] != "show" || data["adult_confirmed"] != true {
		t.Fatalf("data = %v", data)
	}

	raw, err := h.rdb.Get(context.Background(), middleware.SessionKey(prefsSession)).Result()
	if err != nil {
		t.Fatalf("read session: %v", err)
	}
	var session middleware.SessionData
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		t.Fatalf("unmarshal session: %v", err)
	}
	if session.NSFWDisplay != "show" || !session.AdultConfirmed {
		t.Fatalf("session still holds the old stance: %+v", session.UserInfo)
	}
	if session.ContentStance() != "show" {
		t.Fatalf("stance = %q", session.ContentStance())
	}
}

func TestUpdateNSFWDisplayRejectsAnUnknownValue(t *testing.T) {
	h := newPrefsHarness(t, 200, `{"code":0,"data":{"nsfw_display":"show"}}`)

	status, _, _ := h.do(t, "PUT", "/user/nsfw", `{"nsfw_display":"peek"}`, nil)
	if status != 400 {
		t.Fatalf("status %d, want 400", status)
	}
	if h.upstream.method != "" {
		t.Fatalf("an invalid value still reached upstream: %s %s", h.upstream.method, h.upstream.path)
	}
}

// A client without the `preferences` scope is the only upstream refusal the
// browser still has to tell apart: it degrades to cookies instead of toasting.
func TestContentPrefsErrorMapping(t *testing.T) {
	h := newPrefsHarness(t, 403, `{"code":18001,"message":"缺少 scope"}`)

	status, envelope, _ := h.do(t, "PUT", "/user/nsfw", `{"nsfw_display":"blur"}`, nil)
	if status != 403 || int(envelope["code"].(float64)) != errors.CodeCloudPreferencesUnavailable {
		t.Fatalf("%d/%v, want 403/%d", status, envelope["code"], errors.CodeCloudPreferencesUnavailable)
	}
}

func TestGetPreferencesHidesTheNamespace(t *testing.T) {
	h := newPrefsHarness(t, 200,
		`{"code":0,"data":{"namespace":"kungal-forum","doc":{"show_rating":false},"version":4,"updated_at":"2026-09-22T08:31:00Z"}}`)

	status, envelope, resp := h.do(t, "GET", "/user/preferences", "", nil)
	if status != 200 {
		t.Fatalf("status %d: %v", status, envelope)
	}
	if resp.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("a preferences read must not be cacheable")
	}
	if h.upstream.path != "/auth/me/preferences/kungal-forum" {
		t.Fatalf("upstream path = %s", h.upstream.path)
	}
	data, _ := envelope["data"].(map[string]any)
	if _, leaked := data["namespace"]; leaked {
		t.Fatalf("the namespace reached the browser: %v", data)
	}
	if data["version"].(float64) != 4 {
		t.Fatalf("version = %v", data["version"])
	}
}

// The namespace is the site's own client id and nothing in the request may
// move it: a namespace the browser could name reaches the shared `global`
// document, which every other NextMoe site reads and writes.
func TestUpdatePreferencesIgnoresACallerSuppliedNamespace(t *testing.T) {
	h := newPrefsHarness(t, 200, `{"code":0,"data":{"doc":{},"version":2}}`)

	status, _, _ := h.do(t, "PUT", "/user/preferences",
		`{"namespace":"global","doc":{"show_rating":false}}`, map[string]string{"If-Match": `"1"`})
	if status != 200 {
		t.Fatalf("status %d", status)
	}
	if h.upstream.path != "/auth/me/preferences/kungal-forum" {
		t.Fatalf("upstream path = %s", h.upstream.path)
	}
	if h.upstream.ifMatch != `"1"` {
		t.Fatalf("If-Match = %q, want it forwarded verbatim", h.upstream.ifMatch)
	}
	if h.upstream.body != `{"doc":{"show_rating":false}}` {
		t.Fatalf("upstream body = %s", h.upstream.body)
	}
}

func TestUpdatePreferencesSurfacesAVersionConflict(t *testing.T) {
	h := newPrefsHarness(t, 412, `{"code":18006,"message":"版本不符"}`)

	status, envelope, _ := h.do(t, "PUT", "/user/preferences", `{"doc":{}}`, map[string]string{"If-Match": `"9"`})
	if status != 412 || int(envelope["code"].(float64)) != errors.CodeCloudPreferencesConflict {
		t.Fatalf("%d/%v, want 412/%d", status, envelope["code"], errors.CodeCloudPreferencesConflict)
	}
}
