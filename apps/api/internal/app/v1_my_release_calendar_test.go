package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	calendarapiv1 "kun-galgame-api/internal/galgame/calendarapiv1"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/testdb"
	"kun-galgame-api/pkg/problem"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

type myCalHit struct {
	method string
	path   string
	auth   string
	query  url.Values
}

type myCalFix struct {
	app  *App
	rdb  *redis.Client
	spec *specConformance
	mu   sync.Mutex
	hits []myCalHit
}

func newMyCalFix(t *testing.T) *myCalFix {
	t.Helper()
	db := testdb.Open(t)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: 0})
	t.Cleanup(func() { _ = rdb.Close() })

	f := &myCalFix{rdb: rdb}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := url.Values{}
		for k, vs := range r.URL.Query() {
			q[k] = append([]string(nil), vs...)
		}
		f.mu.Lock()
		f.hits = append(f.hits, myCalHit{
			method: r.Method, path: r.URL.Path, auth: r.Header.Get("Authorization"), query: q,
		})
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"items":[],"month":"2026-09","total":0,"meta":{"today":"2026-09-23","min_month":"2026-01","max_month":"2026-11","has_prev":false,"has_next":false}}`)
	}))
	t.Cleanup(srv.Close)

	gc := client.New(srv.URL, "app-key-not-user", geCDN).WithRedis(rdb)
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		loc = time.UTC
	}
	cfg := testConfig()
	cfg.NextMoeAPI.ImageCDNBase = geCDN
	f.app = &App{
		Fiber:  newFiber(),
		Config: cfg,
		DB:     db,
		Redis:  rdb,
		Authn:  middleware.NewAuthenticator(rdb, nil, middleware.NewBearer(w3Verifier{}, rdb, nil)),
		GalgameCalendarV1: calendarapiv1.New(gc, db, geCDN).WithClock(func() time.Time {
			return time.Date(2026, 9, 23, 12, 0, 0, 0, loc)
		}),
	}
	f.app.setupRoutes()
	f.spec = newSpecConformance(t)
	myCalBind(t, rdb, "sess-mycal", w3UserAlice, "user-calendar-token")
	return f
}

func myCalBind(t *testing.T, rdb *redis.Client, cookie string, userID int, token string) {
	t.Helper()
	data, err := json.Marshal(middleware.SessionData{
		UserInfo:         middleware.UserInfo{ID: userID, Name: "n", Roles: []string{"user"}},
		OAuthAccessToken: token,
		OAuthExpiresAt:   time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := rdb.Set(context.Background(), middleware.SessionKey(cookie), data, middleware.SessionTTL).Err(); err != nil {
		t.Fatal(err)
	}
}

func (f *myCalFix) get(t *testing.T, rawURL, specPath, session string) (*http.Response, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, rawURL, nil)
	if session != "" {
		req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: session})
	}
	resp, err := f.app.Fiber.Test(req, fiber.TestConfig{Timeout: 15 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	f.spec.check(t, http.MethodGet, specPath, resp, body)
	var m map[string]any
	if len(body) > 0 {
		if err := json.Unmarshal(body, &m); err != nil {
			t.Fatalf("%s: %v: %s", rawURL, err, body)
		}
	}
	return resp, m
}

func TestV1MyReleaseCalendarUsesTheUserTokenAndNoCache(t *testing.T) {
	f := newMyCalFix(t)
	for range 2 {
		resp, body := f.get(t, "/api/v1/me/release-calendar?month=2026-09", "/me/release-calendar", "sess-mycal")
		geStatus(t, resp, body, http.StatusOK, "")
		if body["object"] != "release_calendar_month" || body["calendar_month"] != "2026-09" {
			t.Fatalf("month %+v", body)
		}
	}
	f.mu.Lock()
	hits := append([]myCalHit(nil), f.hits...)
	f.mu.Unlock()
	if len(hits) != 2 {
		t.Fatalf("upstream hits %d, want 2 (the second call must not be served from cache): %+v", len(hits), hits)
	}
	for _, h := range hits {
		if h.method != http.MethodGet || h.path != "/v2/me/calendar" {
			t.Fatalf("upstream %s %s", h.method, h.path)
		}
		if h.auth != "Bearer user-calendar-token" || strings.Contains(h.auth, "app-key-not-user") {
			t.Fatalf("authorization %q", h.auth)
		}
		if h.query.Get("olang") != "" || h.query.Get("exclude_company_kind") != "" {
			t.Fatalf("personal query must not send olang or exclude_company_kind: %v", h.query)
		}
		if h.query.Get("month") != "2026-09" {
			t.Fatalf("month query %v", h.query)
		}
	}
	keys, err := f.rdb.Keys(context.Background(), "*").Result()
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range keys {
		if strings.HasPrefix(key, "nmcache:v2:") {
			t.Fatalf("personal calendar wrote a shared cache key %s among %v", key, keys)
		}
	}
}

func TestV1MyReleaseCalendarRequiresLogin(t *testing.T) {
	f := newMyCalFix(t)
	resp, body := f.get(t, "/api/v1/me/release-calendar?month=2026-09", "/me/release-calendar", "")
	wantCode(t, resp, body, http.StatusUnauthorized, problem.CodeMissingCredential)
}

func TestV1MyReleaseCalendarForwardsBlockedTags(t *testing.T) {
	f := newMyCalFix(t)
	for _, c := range []struct{ url, spec string }{
		{"/api/v1/me/release-calendar?month=2026-09&exclude_tag_ids=11,12&exclude_tag_trees=13", "/me/release-calendar"},
		{"/api/v1/me/release-calendar/pending?exclude_tag_ids=11,12&exclude_tag_trees=13", "/me/release-calendar/pending"},
		{"/api/v1/me/release-calendar/tba?exclude_tag_ids=11,12&exclude_tag_trees=13", "/me/release-calendar/tba"},
		{"/api/v1/me/release-calendar/upcoming?exclude_tag_ids=11,12&exclude_tag_trees=13", "/me/release-calendar/upcoming"},
	} {
		resp, body := f.get(t, c.url, c.spec, "sess-mycal")
		geStatus(t, resp, body, http.StatusOK, "")
	}
	f.mu.Lock()
	hits := append([]myCalHit(nil), f.hits...)
	f.mu.Unlock()
	if len(hits) == 0 {
		t.Fatal("no upstream hits")
	}
	for _, h := range hits {
		if h.query.Get("exclude_tag_id") != "11,12" || h.query.Get("exclude_tag_tree") != "13" {
			t.Fatalf("%s did not forward the blocked tags: %v", h.path, h.query)
		}
	}
}
