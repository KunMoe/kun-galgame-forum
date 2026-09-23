package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kun-galgame-api/internal/admin/repository"
	"kun-galgame-api/internal/middleware"
	overviewapiv1 "kun-galgame-api/internal/overview/apiv1"
	"kun-galgame-api/internal/testdb"
	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/pkg/perm"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	ovAdmin   = 950000001
	ovMod     = 950000002
	ovGrantee = 950000003

	ovIDMin = 950000100
	ovIDMax = 950000199
)

var ovMetrics = []string{
	"topic_count", "reply_count", "topic_comment_count", "work_count", "galgame_resource_count",
	"galgame_comment_count", "website_count", "website_comment_count", "direct_message_count",
}

type ovVerifier struct{}

func (ovVerifier) Verify(_ context.Context, raw string) (*oauth.AccessClaims, error) {
	switch raw {
	case "grantee-bearer":
		return &oauth.AccessClaims{ID: ovGrantee, Name: "grantee", Roles: []string{"user"}, ClientID: "kungal-app"}, nil
	case "admin-bearer":
		return &oauth.AccessClaims{ID: ovAdmin, Name: "admin", Roles: []string{"user", "admin"}, ClientID: "kungal-app"}, nil
	}
	return nil, fmt.Errorf("bad token")
}

type ovFix struct {
	app  *App
	db   *gorm.DB
	rdb  *redis.Client
	spec *specConformance
}

// 15:00 on 2031-03-10 in Asia/Shanghai: far from any row other tests write.
var ovNow = time.Date(2031, 3, 10, 7, 0, 0, 0, time.UTC)

func newOverviewFix(t *testing.T) *ovFix {
	t.Helper()
	db := testdb.Open(t)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: 0})
	t.Cleanup(func() { _ = rdb.Close() })
	f := &ovFix{db: db, rdb: rdb}
	f.app = &App{
		Fiber:      newFiber(),
		Config:     testConfig(),
		DB:         db,
		Redis:      rdb,
		Authn:      middleware.NewAuthenticator(rdb, nil, middleware.NewBearer(ovVerifier{}, rdb, nil)),
		OverviewV1: overviewapiv1.New(repository.NewOverviewRepository(db), func() time.Time { return ovNow }),
	}
	f.app.setupRoutes()
	f.spec = newSpecConformance(t)
	f.session(t, "sess-ov-admin", ovAdmin, "user", "admin")
	f.session(t, "sess-ov-mod", ovMod, "user", "moderator")
	f.clean(t)
	t.Cleanup(func() { f.clean(t) })
	return f
}

func (f *ovFix) session(t *testing.T, token string, uid int, roles ...string) {
	t.Helper()
	data, err := json.Marshal(middleware.SessionData{
		UserInfo:         middleware.UserInfo{ID: uid, Name: "n", Roles: roles},
		OAuthAccessToken: "access",
		OAuthExpiresAt:   time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.rdb.Set(context.Background(), middleware.SessionKey(token), data, middleware.SessionTTL).Err(); err != nil {
		t.Fatal(err)
	}
}

func (f *ovFix) clean(t *testing.T) {
	t.Helper()
	for _, q := range []string{
		`DELETE FROM feed_activity WHERE source_id BETWEEN ? AND ?`,
		`DELETE FROM chat_message WHERE id BETWEEN ? AND ?`,
		`DELETE FROM chat_room WHERE id BETWEEN ? AND ?`,
		`DELETE FROM galgame_website WHERE id BETWEEN ? AND ?`,
		`DELETE FROM galgame_website_category WHERE id BETWEEN ? AND ?`,
		`DELETE FROM galgame_resource WHERE id BETWEEN ? AND ?`,
		`DELETE FROM galgame WHERE id BETWEEN ? AND ?`,
		`DELETE FROM topic_comment WHERE id BETWEEN ? AND ?`,
		`DELETE FROM topic_reply WHERE id BETWEEN ? AND ?`,
		`DELETE FROM topic WHERE id BETWEEN ? AND ?`,
	} {
		if err := f.db.Exec(q, ovIDMin, ovIDMax).Error; err != nil {
			t.Fatalf("clean: %v\n%s", err, q)
		}
	}
}

func bj(day, hour int) time.Time {
	return time.Date(2031, 3, day, hour, 30, 0, 0, time.FixedZone("CST", 8*3600))
}

// seed places every row inside 2031-03-06..2031-03-10 (Beijing), except one
// topic on 03-05 that only an off-by-one window would pick up. 03-07 stays empty.
func (f *ovFix) seed(t *testing.T) {
	t.Helper()
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, q)
		}
	}
	topic := func(id, status int, at time.Time) {
		run(`INSERT INTO topic (id, title, content, view, status, category, status_update_time, created, updated,
			user_id, is_nsfw, access_scope, cover_images, like_count, dislike_count, reply_count, comment_count,
			favorite_count, upvote_count, view_7d, view_30d, hidden_by, last_reply_floor)
			VALUES (?, 't', 'b', 0, ?, 'galgame', ?, ?, ?, ?, false, 'public', '', 0, 0, 0, 0, 0, 0, 0, 0, '', 0)`,
			id, status, at, at, at, ovAdmin)
	}
	topic(ovIDMin+1, 0, bj(6, 9))
	topic(ovIDMin+2, 1, bj(10, 1)) // 2031-03-09T17:30Z: a UTC bucket would put it on 03-09
	topic(ovIDMin+3, 0, bj(5, 20))
	run(`INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, like_count, created, updated)
		VALUES (?, 'r', 1, ?, ?, 0, 0, ?, ?)`, ovIDMin+10, ovAdmin, ovIDMin+1, bj(8, 12), bj(8, 12))
	run(`INSERT INTO topic_comment (id, content, topic_id, topic_reply_id, user_id, target_user_id, created, updated)
		VALUES (?, 'c', ?, ?, ?, ?, ?, ?)`, ovIDMin+20, ovIDMin+1, ovIDMin+10, ovAdmin, ovAdmin, bj(8, 13), bj(8, 13))
	for i, published := range []bool{true, true, false} {
		run(`INSERT INTO galgame (id, created, updated, published) VALUES (?, ?, ?, ?)`,
			ovIDMin+30+i, bj(8, 10), bj(8, 10), published)
	}
	run(`INSERT INTO galgame_resource (id, work_id, user_id, created, updated) VALUES (?, ?, ?, ?, ?)`,
		ovIDMin+40, ovIDMin+30, ovAdmin, bj(9, 10), bj(9, 10))
	run(`INSERT INTO galgame_website_category (id, name, label, description, created, updated, sort_order)
		VALUES (?, 'ovcat', 'ovcat', '', ?, ?, 0)`, ovIDMin+50, bj(9, 10), bj(9, 10))
	run(`INSERT INTO galgame_website (id, name, url, create_time, category_id, created, updated)
		VALUES (?, 'w', 'https://w.example', '2020', ?, ?, ?)`, ovIDMin+51, ovIDMin+50, bj(9, 11), bj(9, 11))
	run(`INSERT INTO chat_room (id, name, avatar, type, last_message_content, last_message_sender_name, created, updated)
		VALUES (?, 'ovroom', '', 'private', '', '', ?, ?)`, ovIDMin+60, bj(9, 10), bj(9, 10))
	for i := range 2 {
		run(`INSERT INTO chat_message (id, chatroom_name, content, chat_room_id, sender_id, created, updated)
			VALUES (?, 'ovroom', 'm', ?, ?, ?, ?)`, ovIDMin+61+i, ovIDMin+60, ovAdmin, bj(10, 9), bj(10, 9))
	}
	feed := func(id int, kind string, n int) {
		for i := range n {
			run(`INSERT INTO feed_activity (type, source_id, created) VALUES (?, ?, ?)`, kind, id+i, bj(9, 20))
		}
	}
	feed(ovIDMin+70, "GALGAME_COMMENT_CREATION", 3)
	feed(ovIDMin+80, "GALGAME_WEBSITE_COMMENT_CREATION", 2)
}

func (f *ovFix) get(t *testing.T, rawURL, specPath, session, bearer string) (*http.Response, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, rawURL, nil)
	if session != "" {
		req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: session})
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := f.app.Fiber.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	f.spec.checkPath(t, http.MethodGet, specPath, resp, body)
	out := map[string]any{}
	if len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, &out); err != nil {
			t.Fatalf("body: %v\n%s", err, body)
		}
	}
	return resp, out
}

func (f *ovFix) totals(t *testing.T) map[string]int {
	t.Helper()
	resp, body := f.get(t, "/api/v1/admin/overview", "/admin/overview", "sess-ov-admin", "")
	if resp.StatusCode != http.StatusOK || body["object"] != "admin_overview" {
		t.Fatalf("totals %d %+v", resp.StatusCode, body)
	}
	out := map[string]int{}
	for _, k := range ovMetrics {
		out[k] = asInt(body[k])
	}
	return out
}

func TestV1OverviewTotals(t *testing.T) {
	f := newOverviewFix(t)
	before := f.totals(t)
	f.seed(t)
	after := f.totals(t)
	want := map[string]int{
		"topic_count": 3, "reply_count": 1, "topic_comment_count": 1, "work_count": 2,
		"galgame_resource_count": 1, "galgame_comment_count": 3, "website_count": 1,
		"website_comment_count": 2, "direct_message_count": 2,
	}
	for _, k := range ovMetrics {
		if got := after[k] - before[k]; got != want[k] {
			t.Errorf("%s grew by %d, want %d", k, got, want[k])
		}
	}
}

func (f *ovFix) days(t *testing.T, n int) []map[string]any {
	t.Helper()
	resp, body := f.get(t, fmt.Sprintf("/api/v1/admin/overview/daily?days=%d", n), "/admin/overview/daily", "sess-ov-admin", "")
	if resp.StatusCode != http.StatusOK || body["object"] != "list" {
		t.Fatalf("daily %d %+v", resp.StatusCode, body)
	}
	raw, _ := body["items"].([]any)
	out := make([]map[string]any, len(raw))
	for i, it := range raw {
		out[i], _ = it.(map[string]any)
	}
	return out
}

func TestV1OverviewDaily(t *testing.T) {
	f := newOverviewFix(t)
	f.seed(t)
	got := f.days(t, 5)
	if len(got) != 5 {
		t.Fatalf("days=5 gave %d buckets", len(got))
	}
	want := map[string]map[string]int{
		"2031-03-06": {"topic_count": 1},
		"2031-03-07": {},
		"2031-03-08": {"reply_count": 1, "topic_comment_count": 1, "work_count": 2},
		"2031-03-09": {"galgame_resource_count": 1, "website_count": 1, "galgame_comment_count": 3, "website_comment_count": 2},
		"2031-03-10": {"topic_count": 1, "direct_message_count": 2},
	}
	order := []string{"2031-03-06", "2031-03-07", "2031-03-08", "2031-03-09", "2031-03-10"}
	for i, day := range order {
		b := got[i]
		if b["object"] != "overview_day" || b["bucket_date"] != day {
			t.Fatalf("bucket %d is %v %v, want %s", i, b["object"], b["bucket_date"], day)
		}
		for _, k := range ovMetrics {
			if asInt(b[k]) != want[day][k] {
				t.Errorf("%s %s = %v, want %d", day, k, b[k], want[day][k])
			}
		}
	}

	wide := f.days(t, 6)
	if len(wide) != 6 || wide[0]["bucket_date"] != "2031-03-05" || asInt(wide[0]["topic_count"]) != 1 {
		t.Errorf("days=6 first bucket %+v", wide[0])
	}
	if one := f.days(t, 1); len(one) != 1 || one[0]["bucket_date"] != "2031-03-10" {
		t.Errorf("days=1 %+v", one)
	}
}

func TestV1OverviewDailyDefault(t *testing.T) {
	f := newOverviewFix(t)
	resp, body := f.get(t, "/api/v1/admin/overview/daily", "/admin/overview/daily", "sess-ov-admin", "")
	items, _ := body["items"].([]any)
	if resp.StatusCode != http.StatusOK || len(items) != 30 {
		t.Fatalf("default window %d, %d buckets", resp.StatusCode, len(items))
	}
	last, _ := items[29].(map[string]any)
	if last["bucket_date"] != "2031-03-10" {
		t.Errorf("last bucket %v", last["bucket_date"])
	}
}

func TestV1OverviewRejects(t *testing.T) {
	f := newOverviewFix(t)
	for _, c := range []struct {
		name, url, spec, session, bearer string
		status                           int
		code                             string
	}{
		{"anonymous totals", "/api/v1/admin/overview", "/admin/overview", "", "", 401, "MISSING_CREDENTIAL"},
		{"anonymous daily", "/api/v1/admin/overview/daily", "/admin/overview/daily", "", "", 401, "MISSING_CREDENTIAL"},
		{"stale session", "/api/v1/admin/overview", "/admin/overview", "sess-gone", "", 401, "INVALID_CREDENTIAL"},
		{"moderator totals", "/api/v1/admin/overview", "/admin/overview", "sess-ov-mod", "", 403, "PERMISSION_REQUIRED"},
		{"moderator daily", "/api/v1/admin/overview/daily", "/admin/overview/daily", "sess-ov-mod", "", 403, "PERMISSION_REQUIRED"},
		{"admin over bearer", "/api/v1/admin/overview", "/admin/overview", "", "admin-bearer", 403, "PERMISSION_REQUIRED"},
		{"days 0", "/api/v1/admin/overview/daily?days=0", "/admin/overview/daily", "sess-ov-admin", "", 400, "INVALID_PARAMETER"},
		{"days 366", "/api/v1/admin/overview/daily?days=366", "/admin/overview/daily", "sess-ov-admin", "", 400, "INVALID_PARAMETER"},
		{"days not a number", "/api/v1/admin/overview/daily?days=week", "/admin/overview/daily", "sess-ov-admin", "", 400, "INVALID_PARAMETER"},
	} {
		t.Run(c.name, func(t *testing.T) {
			resp, body := f.get(t, c.url, c.spec, c.session, c.bearer)
			wantProblem(t, resp, body, c.status, c.code)
		})
	}
	resp, body := f.get(t, "/api/v1/admin/overview/daily?days=366", "/admin/overview/daily", "sess-ov-admin", "")
	wantField(t, body, "days", "OUT_OF_RANGE")
	_ = resp
}

func TestV1OverviewBearerNeverHasTheDashboard(t *testing.T) {
	f := newOverviewFix(t)
	perm.SetUserOverrides(map[int][]perm.Override{ovGrantee: {{Permission: perm.AdminDashboard, Effect: perm.EffectGrant}}})
	t.Cleanup(func() { perm.SetUserOverrides(nil) })
	if !perm.CanUser(ovGrantee, []string{"user"}, perm.AdminDashboard) {
		t.Fatal("the personal grant did not take")
	}
	for _, c := range []struct{ url, spec string }{
		{"/api/v1/admin/overview", "/admin/overview"},
		{"/api/v1/admin/overview/daily", "/admin/overview/daily"},
	} {
		resp, body := f.get(t, c.url, c.spec, "", "grantee-bearer")
		wantProblem(t, resp, body, http.StatusForbidden, "PERMISSION_REQUIRED")
	}
}
