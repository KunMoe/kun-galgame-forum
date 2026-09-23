package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"kun-galgame-api/internal/middleware"
	trustapiv1 "kun-galgame-api/internal/trust/apiv1"
	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/pkg/trustclient"
	"kun-galgame-api/pkg/userclient"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

const (
	tsUserMod     = 940000001
	tsUserPlain   = 940000002
	tsUserBanned  = 940000003
	tsUserGrantee = 940000004
	tsReporterA   = 940000011
	tsReporterB   = 940000012

	tsSite  = "kungal"
	tsOther = "moyu"

	tsReviewPath = "/admin/review-items/{review_item_id}"
)

type tsVerifier struct{}

func (tsVerifier) Verify(_ context.Context, raw string) (*oauth.AccessClaims, error) {
	switch raw {
	case "grantee-bearer":
		return &oauth.AccessClaims{ID: tsUserGrantee, Name: "grantee", Roles: []string{"user"}, ClientID: "kungal-app"}, nil
	case "mod-bearer":
		return &oauth.AccessClaims{ID: tsUserMod, Name: "mod", Roles: []string{"user"}, SiteRoles: []string{"moderator"}, ClientID: "kungal-app"}, nil
	}
	return nil, fmt.Errorf("bad token")
}

type fakeTrust struct {
	mu          sync.Mutex
	reasons     []trustclient.ReasonView
	items       []trustclient.ReviewItem
	reports     map[int64][]trustclient.Report
	submitted   []trustclient.ReportRequest
	calls       []string
	failReasons atomic.Bool
	failSubmit  atomic.Int32
	failAdmin   atomic.Int32
}

type trustFix struct {
	app    *App
	spec   *specConformance
	rdb    *redis.Client
	trust  *fakeTrust
	failOA atomic.Bool
}

func tsTokenUser(r *http.Request) int64 {
	uid, _ := strconv.ParseInt(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer tok-"), 10, 64)
	return uid
}

func trustEnvelope(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	code := 0
	if status != http.StatusOK {
		code = status
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"code": code, "message": "", "data": data})
}

func (ft *fakeTrust) serve(w http.ResponseWriter, r *http.Request) {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	ft.calls = append(ft.calls, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
	path := r.URL.Path
	if strings.HasPrefix(path, "/api/v1/admin/trust/") {
		if code := int(ft.failAdmin.Load()); code != 0 {
			trustEnvelope(w, code, nil)
			return
		}
	}
	switch {
	case r.Method == http.MethodGet && path == "/api/v1/trust/report-reasons":
		if ft.failReasons.Load() {
			trustEnvelope(w, http.StatusInternalServerError, nil)
			return
		}
		trustEnvelope(w, http.StatusOK, map[string]any{"reasons": ft.reasons})
	case r.Method == http.MethodPost && path == "/api/v1/trust/reports":
		var req trustclient.ReportRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if code := int(ft.failSubmit.Load()); code != 0 {
			trustEnvelope(w, code, nil)
			return
		}
		known := false
		for _, rs := range ft.reasons {
			known = known || rs.Key == req.ReasonKey
		}
		if !known {
			trustEnvelope(w, http.StatusUnprocessableEntity, nil)
			return
		}
		ft.submitted = append(ft.submitted, req)
		trustEnvelope(w, http.StatusOK, map[string]any{"report_id": len(ft.submitted)})
	case r.Method == http.MethodGet && path == "/api/v1/admin/trust/review-items":
		ft.list(w, r)
	case strings.HasPrefix(path, "/api/v1/admin/trust/review-items/"):
		rest := strings.TrimPrefix(path, "/api/v1/admin/trust/review-items/")
		idStr, verb, _ := strings.Cut(rest, "/")
		id, _ := strconv.ParseInt(idStr, 10, 64)
		it := ft.find(id)
		if it == nil {
			trustEnvelope(w, http.StatusNotFound, nil)
			return
		}
		switch verb {
		case "":
			reports := ft.reports[id]
			if reports == nil {
				reports = []trustclient.Report{}
			}
			trustEnvelope(w, http.StatusOK, trustclient.ReviewItemDetail{Item: *it, Reports: reports})
		case "claim":
			if it.Status != 0 {
				trustEnvelope(w, http.StatusConflict, nil)
				return
			}
			uid, now := tsTokenUser(r), time.Now()
			it.Status, it.ClaimedBy, it.ClaimedAt = 1, &uid, &now
			trustEnvelope(w, http.StatusOK, map[string]any{"ok": true})
		case "decide":
			var req trustclient.DecideRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.Decision != "dismissed" && req.Decision != "actioned" ||
				req.Decision == "actioned" && (req.Action == nil || req.ReasonCode == "") {
				trustEnvelope(w, http.StatusBadRequest, nil)
				return
			}
			if it.Status != 0 && it.Status != 1 {
				trustEnvelope(w, http.StatusConflict, nil)
				return
			}
			uid, now := tsTokenUser(r), time.Now()
			it.Status, it.DecidedBy, it.DecidedAt = 3, &uid, &now
			if req.Decision == "actioned" {
				it.Status = 2
			}
			trustEnvelope(w, http.StatusOK, map[string]any{"decided": true})
		}
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (ft *fakeTrust) find(id int64) *trustclient.ReviewItem {
	for i := range ft.items {
		if ft.items[i].ID == id {
			return &ft.items[i]
		}
	}
	return nil
}

func (ft *fakeTrust) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var rows []trustclient.ReviewItem
	for _, it := range ft.items {
		if site := q.Get("site"); site != "" && it.Site != site {
			continue
		}
		if st := q.Get("status"); st != "" && st != strconv.Itoa(int(it.Status)) {
			continue
		}
		rows = append(rows, it)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Priority != rows[j].Priority {
			return rows[i].Priority > rows[j].Priority
		}
		return rows[i].ID > rows[j].ID
	})
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	start := min((page-1)*limit, len(rows))
	end := min(start+limit, len(rows))
	items := rows[start:end]
	if items == nil {
		items = []trustclient.ReviewItem{}
	}
	trustEnvelope(w, http.StatusOK, trustclient.ReviewItemPage{Items: items, Total: int64(len(rows))})
}

func (ft *fakeTrust) callsMatching(prefix string) []string {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	var out []string
	for _, c := range ft.calls {
		if strings.HasPrefix(c, prefix) {
			out = append(out, c)
		}
	}
	return out
}

func (ft *fakeTrust) submissions() []trustclient.ReportRequest {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	return append([]trustclient.ReportRequest(nil), ft.submitted...)
}

func (ft *fakeTrust) item(id int64) trustclient.ReviewItem {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	return *ft.find(id)
}

func newTrustFix(t *testing.T) *trustFix {
	t.Helper()
	f := &trustFix{trust: &fakeTrust{reports: map[int64][]trustclient.Report{}}}
	f.seed()

	trustSrv := httptest.NewServer(http.HandlerFunc(f.trust.serve))
	t.Cleanup(trustSrv.Close)

	oauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f.failOA.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		known := map[int]map[string]any{
			tsUserMod:     {"name": "mod", "status": 0},
			tsUserPlain:   {"name": "plain", "status": 0},
			tsUserBanned:  {"name": "banned", "status": 1},
			tsUserGrantee: {"name": "grantee", "status": 0},
			tsReporterA:   {"name": "reporter-a", "status": 0},
		}
		var users []map[string]any
		for _, raw := range strings.Split(r.URL.Query().Get("ids"), ",") {
			id, _ := strconv.Atoi(raw)
			if u, ok := known[id]; ok {
				users = append(users, map[string]any{"id": id, "name": u["name"], "status": u["status"], "roles": []string{"user"}})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"users": users, "not_found": []int{}}})
	}))
	t.Cleanup(oauthSrv.Close)

	mr := miniredis.RunT(t)
	f.rdb = redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: 0})
	t.Cleanup(func() { _ = f.rdb.Close() })

	tc := trustclient.New(trustclient.Config{BaseURL: trustSrv.URL, ClientID: "c", ClientSecret: "s"})
	uc := userclient.New(userclient.Config{
		BaseURL: oauthSrv.URL, ClientID: "c", ClientSecret: "s",
		ImageCDNBase: "https://image.test.example", HTTPTimeout: 2 * time.Second,
	})
	f.app = &App{
		Fiber:   newFiber(),
		Config:  testConfig(),
		Redis:   f.rdb,
		Authn:   middleware.NewAuthenticator(f.rdb, nil, middleware.NewBearer(tsVerifier{}, f.rdb, nil)),
		TrustV1: trustapiv1.New(tc, uc, tsSite, "https://image.test.example"),
	}
	f.app.setupRoutes()
	f.spec = newSpecConformance(t)
	for _, s := range []struct {
		token string
		uid   int
		roles []string
	}{
		{"sess-mod", tsUserMod, []string{"user", "moderator"}},
		{"sess-plain", tsUserPlain, []string{"user"}},
		{"sess-banned", tsUserBanned, []string{"user"}},
	} {
		f.putSession(t, s.token, s.uid, s.roles)
	}
	return f
}

func (f *trustFix) putSession(t *testing.T, token string, uid int, roles []string) {
	t.Helper()
	data, err := json.Marshal(middleware.SessionData{
		UserInfo:         middleware.UserInfo{ID: uid, Name: "n", Roles: roles},
		OAuthAccessToken: fmt.Sprintf("tok-%d", uid),
		OAuthExpiresAt:   time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.rdb.Set(context.Background(), middleware.SessionKey(token), data, middleware.SessionTTL).Err(); err != nil {
		t.Fatal(err)
	}
}

func tsTime(hour int) time.Time {
	return time.Date(2026, 9, 1, hour, 0, 0, 0, time.UTC)
}

func (f *trustFix) seed() {
	ft := f.trust
	ft.reasons = []trustclient.ReasonView{
		{ID: 1, Key: "abuse", NameCN: "辱骂/骚扰", Severity: 2},
		{ID: 2, Key: "spam", NameCN: "垃圾信息", Severity: 1},
	}
	sev := int16(2)
	weight := float32(1.5)
	note := "flagged excerpt"
	mod := int64(tsUserMod)
	id := int64(9000)
	add := func(site string, status, source int16, priority float32) int64 {
		id++
		it := trustclient.ReviewItem{
			ID: id, Site: site, SubjectKind: "forum_topic", SubjectID: strconv.FormatInt(id, 10),
			Source: source, Priority: priority, Status: status, CreatedAt: tsTime(int(id % 20)),
		}
		switch source {
		case 0:
			it.Severity, it.ReportWeightSum = &sev, &weight
		default:
			it.ContextNote = &note
		}
		if status >= 1 {
			it.ClaimedBy, it.ClaimedAt = &mod, ptrTime(tsTime(1))
		}
		if status >= 2 {
			it.DecidedBy, it.DecidedAt = &mod, ptrTime(tsTime(2))
		}
		ft.items = append(ft.items, it)
		return id
	}
	// Seven pending items share one priority so a page boundary lands inside
	// the tie and only the id tie-breaker keeps the walk stable.
	for range 7 {
		add(tsSite, 0, 0, 2)
	}
	add(tsSite, 0, 1, 5)
	add(tsSite, 1, 0, 3)
	add(tsSite, 2, 3, 1)
	add(tsSite, 3, 1, 1)
	add(tsSite, 3, 99, 0.5)
	add(tsOther, 0, 0, 9)
	add(tsOther, 3, 0, 9)

	snap := "the content"
	onSite := "https://www.kungal.com/topic/9001"
	offSite := "https://evil.example/topic/9001"
	ft.reports[9001] = []trustclient.Report{
		{ID: 71, ReporterID: tsReporterA, ReasonID: 2, SubjectSnapshot: &snap, SubjectURL: &onSite, Weight: 1, CreatedAt: tsTime(3)},
		{ID: 72, ReporterID: tsReporterB, ReasonID: 99, SubjectURL: &offSite, Weight: 0.5, CreatedAt: tsTime(4)},
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

func (f *trustFix) call(t *testing.T, method, rawURL, specPath, session string, hdr http.Header, payload any) (*http.Response, map[string]any) {
	t.Helper()
	var r io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, rawURL, r)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if session != "" {
		req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: session})
	}
	for k, vs := range hdr {
		req.Header[k] = vs
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
	f.spec.checkPath(t, method, specPath, resp, body)
	out := map[string]any{}
	if len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, &out); err != nil {
			t.Fatalf("body is not a JSON object: %v\n%s", err, body)
		}
	}
	return resp, out
}

func wantProblem(t *testing.T, resp *http.Response, body map[string]any, status int, code string) {
	t.Helper()
	if resp.StatusCode != status || body["code"] != code {
		t.Fatalf("got %d %v, want %d %s: %+v", resp.StatusCode, body["code"], status, code, body)
	}
}

func wantField(t *testing.T, body map[string]any, pointer, reason string) {
	t.Helper()
	errs, _ := body["errors"].([]any)
	for _, e := range errs {
		m, _ := e.(map[string]any)
		if (m["pointer"] == pointer || m["parameter"] == pointer) && m["reason"] == reason {
			return
		}
	}
	t.Fatalf("no error at %s with reason %s: %+v", pointer, reason, body)
}
