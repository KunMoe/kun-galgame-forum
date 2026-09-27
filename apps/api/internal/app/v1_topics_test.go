package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"mime"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/testdb"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"gorm.io/gorm"
)

const (
	v1UserAlice  = 910000001
	v1UserBanned = 910000002
	v1UserBob    = 910000003
	v1UserGone   = 910000004
	v1TopicMin   = 910000201
	v1TopicMax   = 910000299
	v1CoverHash  = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	v1CoverHash2 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	v1SectionID  = 910000001
)

// The compiler and its cache are shared, and the concurrency tests call
// checkPath from several goroutines: without this lock jsonschema's compiler
// crashed the run with a concurrent map write.
type specConformance struct {
	mu       sync.Mutex
	compiler *jsonschema.Compiler
	doc      map[string]any
	schemas  map[string]*jsonschema.Schema
}

func newSpecConformance(t *testing.T) *specConformance {
	t.Helper()
	raw, err := apiv1.MarshalOpenAPI(V1Spec())
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	c.AssertFormat()
	if err := c.AddResource("https://kungal.local/openapi.json", decoded); err != nil {
		t.Fatal(err)
	}
	root, ok := decoded.(map[string]any)
	if !ok {
		t.Fatal("openapi document is not an object")
	}
	return &specConformance{compiler: c, doc: root, schemas: map[string]*jsonschema.Schema{}}
}

func TestV1TopicsResponseSchemasCompile(t *testing.T) {
	s := newSpecConformance(t)
	for _, status := range []string{"200", "400", "401", "403", "500", "503"} {
		ct := "application/json"
		if status != "200" {
			ct = problem.ContentType
		}
		loc := "https://kungal.local/openapi.json#/paths/" + jsonPointerToken("/topics") +
			"/get/responses/" + status + "/content/" + jsonPointerToken(ct) + "/schema"
		if _, err := s.compiler.Compile(loc); err != nil {
			t.Errorf("%s: %v", loc, err)
		}
	}
}

func jsonPointerToken(s string) string {
	s = strings.ReplaceAll(s, "~", "~0")
	return strings.ReplaceAll(s, "/", "~1")
}

func (s *specConformance) check(t *testing.T, method, specPath string, resp *http.Response, body []byte) {
	t.Helper()
	status := strconv.Itoa(resp.StatusCode)
	paths, _ := s.doc["paths"].(map[string]any)
	item, _ := paths[specPath].(map[string]any)
	op, _ := item[strings.ToLower(method)].(map[string]any)
	responses, _ := op["responses"].(map[string]any)
	if responses[status] == nil {
		t.Errorf("%s %s status %s is not a key of the operation responses", method, specPath, status)
		return
	}
	respObj, _ := responses[status].(map[string]any)
	content, _ := respObj["content"].(map[string]any)
	ct, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil || ct == "" {
		t.Errorf("%s %s Content-Type %q", method, specPath, resp.Header.Get("Content-Type"))
		return
	}
	if content[ct] == nil {
		t.Errorf("%s %s status %s Content-Type %q is not a key of that response content", method, specPath, status, ct)
		return
	}
	loc := "https://kungal.local/openapi.json#/paths/" + jsonPointerToken(specPath) +
		"/" + strings.ToLower(method) + "/responses/" + status +
		"/content/" + jsonPointerToken(ct) + "/schema"
	sch, ok := s.schemas[loc]
	if !ok {
		compiled, err := s.compiler.Compile(loc)
		if err != nil {
			t.Errorf("compile %s: %v", loc, err)
			return
		}
		s.schemas[loc] = compiled
		sch = compiled
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(body))
	if err != nil {
		t.Errorf("body is not JSON: %v\n%s", err, body)
		return
	}
	if err := sch.Validate(inst); err != nil {
		t.Errorf("%s %s %s schema: %v\n%s", method, specPath, status, err, body)
	}
}

type topicsFix struct {
	*App
	db     *gorm.DB
	rdb    *redis.Client
	spec   *specConformance
	oauth  *httptest.Server
	failOA atomic.Bool
}

func newTopicsFix(t *testing.T) *topicsFix {
	t.Helper()
	db := testdb.Open(t)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: 0})
	t.Cleanup(func() { _ = rdb.Close() })

	f := &topicsFix{db: db, rdb: rdb}
	mux := http.NewServeMux()
	mux.HandleFunc("/users/batch", func(w http.ResponseWriter, _ *http.Request) {
		if f.failOA.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, "oauth down")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"users": []map[string]any{
					{"id": v1UserAlice, "name": "alice", "status": 0, "roles": []string{"user"}},
					{"id": v1UserBanned, "name": "banned", "status": 1, "roles": []string{"user"}},
					{"id": v1UserBob, "name": "bob", "status": 0, "roles": []string{"user"}},
				},
				"not_found": []int{},
			},
		})
	})
	f.oauth = httptest.NewServer(mux)
	t.Cleanup(f.oauth.Close)

	uc := userclient.New(userclient.Config{
		BaseURL:      f.oauth.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		ImageCDNBase: "https://image.test.example",
		HTTPTimeout:  2 * time.Second,
	})
	cfg := testConfig()
	cfg.NextMoeAPI.ImageCDNBase = "https://image.test.example"
	f.App = &App{
		Fiber:      newFiber(),
		Config:     cfg,
		DB:         db,
		Redis:      rdb,
		UserClient: uc,
		Authn:      middleware.NewAuthenticator(rdb, nil, nil),
	}
	f.setupRoutes()
	f.spec = newSpecConformance(t)
	f.seed(t)
	return f
}

func (f *topicsFix) seed(t *testing.T) {
	t.Helper()
	f.cleanup(t)
	t.Cleanup(func() { f.cleanup(t) })

	base := time.Date(2026, 1, 15, 12, 0, 0, 123456000, time.UTC)
	runSQL := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	runSQL(`INSERT INTO topic_section (id, name, created, updated) VALUES (?, 'g-walkthrough', ?, ?)
		ON CONFLICT (id) DO NOTHING`, v1SectionID, base, base)

	type row struct {
		id, user, view, like, fav, up, v7, v30, status int
		cat, scope                                     string
		nsfw                                           bool
		bump, created                                  time.Time
		cover                                          string
		upvote                                         *time.Time
	}
	sameBump := base.Add(3 * time.Hour)
	rows := []row{
		{910000201, v1UserAlice, 10, 1, 1, 1, 2, 3, 0, "galgame", "public", false, sameBump, base, `["/image/` + v1CoverHash + `"]`, &base},
		{910000202, v1UserAlice, 10, 1, 1, 1, 2, 3, 0, "galgame", "public", false, sameBump, base, "", nil},
		{910000203, v1UserAlice, 20, 2, 2, 2, 4, 5, 0, "technique", "public", false, base.Add(4 * time.Hour), base.Add(2 * time.Second), `["/image/not-a-hash","/image/` + v1CoverHash2 + `_mini"]`, nil},
		{910000204, v1UserAlice, 30, 3, 2, 3, 6, 7, 0, "others", "public", false, base.Add(5 * time.Hour), base.Add(3 * time.Second), "", nil},
		{910000205, v1UserAlice, 40, 4, 4, 4, 8, 9, 0, "galgame", "public", true, base.Add(6 * time.Hour), base.Add(4 * time.Second), "", nil},
		{910000206, v1UserAlice, 50, 5, 5, 5, 10, 11, 0, "galgame", "login", false, base.Add(7 * time.Hour), base.Add(5 * time.Second), "", nil},
		{910000207, v1UserAlice, 60, 6, 6, 6, 12, 13, 0, "galgame", "role", false, base.Add(8 * time.Hour), base.Add(6 * time.Second), "", nil},
		{910000208, v1UserAlice, 70, 7, 7, 7, 14, 15, 0, "galgame", "users", false, base.Add(9 * time.Hour), base.Add(7 * time.Second), "", nil},
		{910000209, v1UserAlice, 80, 8, 8, 8, 16, 17, 1, "galgame", "public", false, base.Add(10 * time.Hour), base.Add(8 * time.Second), "", nil},
		{910000210, v1UserBanned, 90, 9, 9, 9, 18, 19, 0, "galgame", "public", false, base.Add(11 * time.Hour), base.Add(9 * time.Second), "", nil},
		{910000211, v1UserAlice, 100, 10, 10, 10, 20, 21, 0, "galgame", "public", false, base.Add(12 * time.Hour), base.Add(10 * time.Second), "", nil},
		{910000212, v1UserAlice, 100, 11, 11, 11, 20, 21, 0, "galgame", "public", false, base.Add(13 * time.Hour), base.Add(11 * time.Second), "", nil},
		{910000213, v1UserBob, 5, 0, 0, 10, 1, 1, 0, "galgame", "public", false, base.Add(14 * time.Hour), base.Add(12 * time.Second), "", nil},
		{910000214, v1UserAlice, 6, 0, 0, 0, 1, 1, 0, "galgame", "public", false, base.Add(15 * time.Hour), base.Add(13 * time.Second), "", nil},
		{910000215, v1UserAlice, 7, 12, 12, 12, 30, 40, 0, "galgame", "public", false, base.Add(16 * time.Hour), base.Add(14 * time.Second), "", nil},
		{910000216, v1UserAlice, 8, 13, 13, 13, 31, 41, 0, "galgame", "public", false, base.Add(16*time.Hour + time.Millisecond), base.Add(15 * time.Second), "", nil},
		{910000217, v1UserGone, 9, 14, 14, 14, 32, 42, 0, "galgame", "public", false, base.Add(17 * time.Hour), base.Add(16 * time.Second), "", nil},
	}
	for i := range 6 {
		rows = append(rows, row{910000218 + i, v1UserBanned, 1, 0, 0, 0, 0, 0, 0, "galgame", "public", false,
			base.Add(time.Duration(18+i) * time.Hour), base.Add(time.Duration(20+i) * time.Second), "", nil})
	}
	for _, r := range rows {
		runSQL(`INSERT INTO topic (
			id, title, content, view, status, category, status_update_time, created, updated,
			user_id, is_nsfw, access_scope, cover_images, like_count, reply_count, comment_count,
			favorite_count, upvote_count, view_7d, view_30d, upvote_time, hidden_by
		) VALUES (?, ?, 'body', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 0, ?, ?, ?, ?, ?, '')`,
			r.id, fmt.Sprintf("v1-topic-%d", r.id), r.view, r.status, r.cat, r.bump, r.created, r.created,
			r.user, r.nsfw, r.scope, r.cover, r.like, r.fav, r.up, r.v7, r.v30, r.upvote)
	}

	runSQL(`UPDATE topic SET edited = ? WHERE id = 910000201`, base.Add(2*time.Hour))
	runSQL(`INSERT INTO topic_section_relation (topic_id, topic_section_id, created, updated) VALUES (910000201, ?, ?, ?)`, v1SectionID, base, base)
	runSQL(`INSERT INTO topic_poll (title, topic_id, user_id, created, updated) VALUES ('p', 910000201, ?, ?, ?)`, v1UserAlice, base, base)
	runSQL(`INSERT INTO topic_lottery (topic_id, user_id, title, created, updated) VALUES (910000202, ?, 'l', ?, ?)`, v1UserAlice, base, base)
	runSQL(`INSERT INTO topic_reply (id, content, user_id, topic_id, created, updated) VALUES (910000301, 'answer', ?, 910000203, ?, ?)`, v1UserBob, base, base)
	runSQL(`UPDATE topic SET best_answer_id = 910000301 WHERE id = 910000203`)
	runSQL(`INSERT INTO topic_view_daily (entity_id, day, count) VALUES (910000201, CURRENT_DATE, 7), (910000202, CURRENT_DATE, 7), (910000211, CURRENT_DATE, 3)`)
}

func (f *topicsFix) cleanup(t *testing.T) {
	t.Helper()
	for _, q := range []string{
		"DELETE FROM topic_view_daily WHERE entity_id BETWEEN ? AND ?",
		"DELETE FROM topic_poll WHERE topic_id BETWEEN ? AND ?",
		"DELETE FROM topic_lottery WHERE topic_id BETWEEN ? AND ?",
		"DELETE FROM topic_section_relation WHERE topic_id BETWEEN ? AND ?",
		"UPDATE topic SET best_answer_id = NULL WHERE id BETWEEN ? AND ?",
		"DELETE FROM topic_reply WHERE topic_id BETWEEN ? AND ?",
		"DELETE FROM topic WHERE id BETWEEN ? AND ?",
	} {
		if err := f.db.Exec(q, v1TopicMin, v1TopicMax).Error; err != nil {
			t.Errorf("cleanup %q: %v", q, err)
		}
	}
	if err := f.db.Exec("DELETE FROM topic_section WHERE id = ?", v1SectionID).Error; err != nil {
		t.Errorf("cleanup topic_section: %v", err)
	}
}

func (f *topicsFix) putSession(t *testing.T, token string, userID int) {
	t.Helper()
	data, err := json.Marshal(middleware.SessionData{
		UserInfo:         middleware.UserInfo{ID: userID, Name: "alice", Roles: []string{"user"}},
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

func (f *topicsFix) get(t *testing.T, rawURL, session string, hdr http.Header) (*http.Response, []byte) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, rawURL, nil)
	if session != "" {
		req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: session})
	}
	for k, vs := range hdr {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	resp, err := f.Fiber.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("%s Cache-Control %q", rawURL, resp.Header.Get("Cache-Control"))
	}
	f.spec.check(t, http.MethodGet, "/topics", resp, body)
	if resp.StatusCode >= 400 {
		var p struct {
			RequestID string `json:"request_id"`
			Detail    string `json:"detail"`
			Code      string `json:"code"`
		}
		if err := json.Unmarshal(body, &p); err != nil {
			t.Fatalf("problem body: %v\n%s", err, body)
		}
		if id := resp.Header.Get(problem.HeaderRequestID); id == "" || id != p.RequestID {
			t.Errorf("X-Request-ID %q body request_id %q", id, p.RequestID)
		}
	}
	return resp, body
}

type topicsListBody struct {
	Object string `json:"object"`
	Items  []struct {
		ID       string `json:"id"`
		Category string `json:"category"`
		IsNSFW   bool   `json:"is_nsfw"`
	} `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

func decodeList(t *testing.T, body []byte) topicsListBody {
	t.Helper()
	var out topicsListBody
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("list body: %v\n%s", err, body)
	}
	return out
}

func decodeProblemBody(t *testing.T, body []byte) (code, reason, param string, allowed bool, detail string) {
	t.Helper()
	var p struct {
		Code   string `json:"code"`
		Detail string `json:"detail"`
		Errors []struct {
			Parameter *string `json:"parameter"`
			Reason    string  `json:"reason"`
			Params    *struct {
				Allowed *[]string `json:"allowed"`
			} `json:"params"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &p); err != nil {
		t.Fatalf("problem: %v\n%s", err, body)
	}
	if len(p.Errors) > 0 {
		reason = p.Errors[0].Reason
		if p.Errors[0].Parameter != nil {
			param = *p.Errors[0].Parameter
		}
		if p.Errors[0].Params != nil && p.Errors[0].Params.Allowed != nil {
			allowed = true
		}
	}
	return p.Code, reason, param, allowed, p.Detail
}

func TestV1TopicsTraversalParity(t *testing.T) {
	f := newTopicsFix(t)
	tokens := []string{
		"bumped_asc", "bumped_desc", "created_asc", "created_desc",
		"views_asc", "views_desc", "views_1d_asc", "views_1d_desc",
		"views_7d_asc", "views_7d_desc", "views_30d_asc", "views_30d_desc",
		"likes_asc", "likes_desc", "favorites_asc", "favorites_desc",
		"upvotes_asc", "upvotes_desc",
	}
	if len(tokens) != 18 {
		t.Fatalf("tokens %d", len(tokens))
	}
	for _, authed := range []bool{false, true} {
		session := ""
		if authed {
			session = "sess-alice"
			f.putSession(t, session, v1UserAlice)
		}
		for _, token := range tokens {
			t.Run(fmt.Sprintf("auth=%v/%s", authed, token), func(t *testing.T) {
				want := f.sqlOrder(t, token, "", false, authed)
				var got []string
				path := "/api/v1/topics?sort=" + token + "&limit=2"
				for page := 0; page < 200; page++ {
					resp, body := f.get(t, path, session, nil)
					if resp.StatusCode != http.StatusOK {
						t.Fatalf("status %d %s", resp.StatusCode, body)
					}
					list := decodeList(t, body)
					if list.NextCursor != nil && len(list.Items) != 2 {
						t.Errorf("page %d of %s holds %d items and a cursor, want 2", page, path, len(list.Items))
					}
					for _, it := range list.Items {
						got = append(got, it.ID)
					}
					if list.NextCursor == nil {
						break
					}
					path = "/api/v1/topics?sort=" + token + "&limit=2&cursor=" + *list.NextCursor
				}
				if strings.Join(got, ",") != strings.Join(want, ",") {
					t.Errorf("got %v\nwant %v", got, want)
				}
				seen := map[string]bool{}
				for _, id := range got {
					if seen[id] {
						t.Errorf("duplicate id %s", id)
					}
					seen[id] = true
				}
			})
		}
	}
}

func (f *topicsFix) sqlOrder(t *testing.T, token, category string, includeNSFW, authed bool) []string {
	t.Helper()
	spec, ok := lookupTestSort(token)
	if !ok {
		t.Fatalf("token %s", token)
	}
	expr := testSortExpr(spec.key)
	pred := "topic.status != 1"
	if authed {
		pred += " AND topic.access_scope IN ('public','login')"
	} else {
		pred += " AND topic.access_scope = 'public'"
	}
	if !includeNSFW {
		pred += " AND topic.is_nsfw = false"
	}
	if category != "" {
		pred += " AND topic.category = '" + category + "'"
	}
	type idRow struct {
		ID     int
		UserID int
	}
	var rows []idRow
	q := fmt.Sprintf("SELECT topic.id, topic.user_id FROM topic WHERE %s ORDER BY %s %s, topic.id %s", pred, expr, spec.dir, spec.dir)
	if err := f.db.Raw(q).Scan(&rows).Error; err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		if r.UserID == v1UserBanned {
			continue
		}
		out = append(out, strconv.Itoa(r.ID))
	}
	return out
}

type testSort struct {
	key, dir string
}

func lookupTestSort(token string) (testSort, bool) {
	m := map[string]testSort{
		"bumped_asc": {"status_update_time", "asc"}, "bumped_desc": {"status_update_time", "desc"},
		"created_asc": {"created", "asc"}, "created_desc": {"created", "desc"},
		"views_asc": {"view", "asc"}, "views_desc": {"view", "desc"},
		"views_1d_asc": {"view_1d", "asc"}, "views_1d_desc": {"view_1d", "desc"},
		"views_7d_asc": {"view_7d", "asc"}, "views_7d_desc": {"view_7d", "desc"},
		"views_30d_asc": {"view_30d", "asc"}, "views_30d_desc": {"view_30d", "desc"},
		"likes_asc": {"like_count", "asc"}, "likes_desc": {"like_count", "desc"},
		"favorites_asc": {"favorite_count", "asc"}, "favorites_desc": {"favorite_count", "desc"},
		"upvotes_asc": {"upvote_count", "asc"}, "upvotes_desc": {"upvote_count", "desc"},
	}
	s, ok := m[token]
	return s, ok
}

func testSortExpr(key string) string {
	if key == "view_1d" {
		return "COALESCE((SELECT SUM(d.count) FROM topic_view_daily d WHERE d.entity_id = topic.id AND d.day = CURRENT_DATE), 0)"
	}
	return "topic." + key
}

func TestV1TopicsVisibility(t *testing.T) {
	f := newTopicsFix(t)
	collectIDs := func(path, session string) map[string]bool {
		t.Helper()
		seen := map[string]bool{}
		for page := 0; page < 200; page++ {
			resp, body := f.get(t, path, session, nil)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("%s: %d %s", path, resp.StatusCode, body)
			}
			list := decodeList(t, body)
			for _, it := range list.Items {
				seen[it.ID] = true
			}
			if list.NextCursor == nil {
				return seen
			}
			if i := strings.Index(path, "cursor="); i >= 0 {
				path = path[:i] + "cursor=" + *list.NextCursor
			} else if strings.Contains(path, "?") {
				path = path + "&cursor=" + *list.NextCursor
			} else {
				path = path + "?cursor=" + *list.NextCursor
			}
		}
		t.Fatal("too many pages")
		return seen
	}

	anon := collectIDs("/api/v1/topics?limit=20", "")
	for _, id := range []string{"910000206", "910000207", "910000208", "910000209", "910000210", "910000205"} {
		if anon[id] {
			t.Errorf("anonymous saw %s", id)
		}
	}
	if !anon["910000201"] || !anon["910000203"] || !anon["910000204"] {
		t.Errorf("anonymous missing a public category row: %v", anon)
	}

	f.putSession(t, "sess-alice", v1UserAlice)
	user := collectIDs("/api/v1/topics?limit=20", "sess-alice")
	if !user["910000206"] {
		t.Error("signed-in missed login-scoped topic")
	}
	for _, id := range []string{"910000207", "910000208", "910000209", "910000210"} {
		if user[id] {
			t.Errorf("signed-in saw %s", id)
		}
	}

	sfw := collectIDs("/api/v1/topics?include_nsfw=false&limit=20", "")
	if sfw["910000205"] {
		t.Error("include_nsfw=false included NSFW")
	}
	nsfw := collectIDs("/api/v1/topics?include_nsfw=true&limit=20", "")
	if !nsfw["910000205"] {
		t.Error("include_nsfw=true missed NSFW")
	}

	gal := collectIDs("/api/v1/topics?category=galgame&limit=20", "")
	if gal["910000203"] || gal["910000204"] {
		t.Error("category=galgame leaked another category")
	}
	if !gal["910000201"] {
		t.Error("category=galgame missed a galgame topic")
	}
}

func TestV1TopicsErrors(t *testing.T) {
	f := newTopicsFix(t)

	assertErr := func(path, session string, hdr http.Header, wantStatus int, wantCode, wantParam, wantReason string, wantAllowed bool) {
		t.Helper()
		resp, body := f.get(t, path, session, hdr)
		if resp.StatusCode != wantStatus {
			t.Errorf("%s status %d, want %d (%s)", path, resp.StatusCode, wantStatus, body)
		}
		code, reason, param, allowed, _ := decodeProblemBody(t, body)
		if code != wantCode {
			t.Errorf("%s code %s, want %s (%s)", path, code, wantCode, body)
		}
		if wantParam != "" && param != wantParam {
			t.Errorf("%s errors[0].parameter %q, want %q (%s)", path, param, wantParam, body)
		}
		if wantReason != "" && reason != wantReason {
			t.Errorf("%s errors[0].reason %s, want %s (%s)", path, reason, wantReason, body)
		}
		if wantAllowed && !allowed {
			t.Errorf("%s missing params.allowed (%s)", path, body)
		}
	}

	assertErr("/api/v1/topics?limit=0", "", nil, 400, problem.CodeInvalidParameter, "limit", "", false)
	assertErr("/api/v1/topics?limit=101", "", nil, 400, problem.CodeLimitTooLarge, "limit", "", false)
	assertErr("/api/v1/topics?limit=abc", "", nil, 400, problem.CodeInvalidParameter, "limit", "", false)
	assertErr("/api/v1/topics?include_nsfw=1", "", nil, 400, problem.CodeInvalidParameter, "include_nsfw", "", false)
	assertErr("/api/v1/topics?category=all", "", nil, 400, problem.CodeUnknownEnumValue, "category", "", true)
	assertErr("/api/v1/topics?sort=hot", "", nil, 400, problem.CodeUnknownSort, "sort", "", false)
	assertErr("/api/v1/topics?cursor=garbage", "", nil, 400, problem.CodeInvalidCursor, "cursor", "", false)

	resp, body := f.get(t, "/api/v1/topics?sort=bumped_desc&limit=2", "", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("mint cursor: %s", body)
	}
	list := decodeList(t, body)
	if list.NextCursor == nil {
		t.Fatal("need a next_cursor to replay")
	}
	cur := *list.NextCursor
	assertErr("/api/v1/topics?sort=created_desc&cursor="+cur, "", nil, 400, problem.CodeInvalidCursor, "cursor", "", false)

	resp, body = f.get(t, "/api/v1/topics?include_nsfw=false&limit=2", "", nil)
	list = decodeList(t, body)
	if list.NextCursor == nil {
		t.Fatal("need a next_cursor for nsfw replay")
	}
	assertErr("/api/v1/topics?include_nsfw=true&cursor="+*list.NextCursor, "", nil, 400, problem.CodeInvalidCursor, "cursor", "", false)

	resp, body = f.get(t, "/api/v1/topics?limit=2", "", nil)
	list = decodeList(t, body)
	if list.NextCursor == nil {
		t.Fatal("need a next_cursor for auth replay")
	}
	f.putSession(t, "sess-alice", v1UserAlice)
	assertErr("/api/v1/topics?limit=2&cursor="+*list.NextCursor, "sess-alice", nil, 400, problem.CodeInvalidCursor, "cursor", "", false)

	hdr := http.Header{}
	hdr.Set("Authorization", "Bearer not-a-token")
	assertErr("/api/v1/topics", "", hdr, 401, problem.CodeInvalidCredential, "", "", false)

	resp, body = f.get(t, "/api/v1/topics?limit=1", "missing-session", nil)
	if resp.StatusCode != 200 {
		t.Errorf("unknown session cookie: %d %s", resp.StatusCode, body)
	}

	f.failOA.Store(true)
	f.UserClient.Invalidate(v1UserAlice, v1UserBanned, v1UserBob)
	resp, body = f.get(t, "/api/v1/topics?limit=1", "", nil)
	if resp.StatusCode != 503 {
		t.Errorf("oauth 500: status %d %s", resp.StatusCode, body)
	}
	code, _, _, _, _ := decodeProblemBody(t, body)
	if code != problem.CodeServiceUnavailable {
		t.Errorf("oauth 500 code %s", code)
	}
	f.failOA.Store(false)

	err := f.db.Exec(`INSERT INTO topic (
		id, title, content, view, status, category, status_update_time, created, updated,
		user_id, is_nsfw, access_scope, cover_images, like_count, favorite_count, upvote_count, view_7d, view_30d, hidden_by
	) VALUES (910000299, 'bad-status', 'body', 0, 2, 'galgame', now(), now(), now(), ?, false, 'public', '', 0, 0, 0, 0, 0, '')`,
		v1UserAlice).Error
	if err == nil || !strings.Contains(err.Error(), "topic_status_check") {
		t.Errorf("a topic with status 2 was not refused by topic_status_check: %v", err)
	}
}

func TestV1TopicsStopsRefillingAfterFiveWindows(t *testing.T) {
	f := newTopicsFix(t)
	resp, body := f.get(t, "/api/v1/topics?sort=created_desc&limit=1", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	first := decodeList(t, body)
	if len(first.Items) != 0 || first.NextCursor == nil {
		t.Fatalf("six banned topics lead created_desc; five one-row windows should leave the page empty with a cursor: %s", body)
	}
	_, body = f.get(t, "/api/v1/topics?sort=created_desc&limit=1&cursor="+*first.NextCursor, "", nil)
	if second := decodeList(t, body); len(second.Items) != 1 || second.Items[0].ID != "910000217" {
		t.Fatalf("the page after the banned run = %s, want 910000217", body)
	}
}

func TestV1TopicsItemFields(t *testing.T) {
	f := newTopicsFix(t)
	resp, body := f.get(t, "/api/v1/topics?sort=created_asc&limit=100", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatal(err)
	}
	if keys := slices.Sorted(maps.Keys(envelope)); !slices.Equal(keys, []string{"items", "object"}) {
		t.Errorf("last page keys %v, want [items object]", keys)
	}
	var list struct {
		Items []map[string]json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatal(err)
	}
	itemKeys := []string{
		"author", "bumped_at", "category", "comment_count", "cover_images", "created_at", "edited_at", "has_best_answer", "id",
		"is_nsfw", "like_count", "mini_apps", "object", "reply_count", "sections", "state", "title", "upvoted_at", "view_count",
	}
	for _, it := range list.Items {
		if keys := slices.Sorted(maps.Keys(it)); !slices.Equal(keys, itemKeys) {
			t.Errorf("topic %s keys %v, want %v", it["id"], keys, itemKeys)
		}
	}
	byID := map[string]map[string]json.RawMessage{}
	for _, it := range list.Items {
		var id string
		if err := json.Unmarshal(it["id"], &id); err != nil {
			t.Fatal(err)
		}
		byID[id] = it
	}
	cover := `[{"url":"https://image.test.example/aa/aa/` + v1CoverHash + `.webp","hash":"` + v1CoverHash +
		`","width":null,"height":null,"thumbhash":null,"sexual":null}]`
	for id, want := range map[string]map[string]string{
		"910000201": {
			"object": `"topic"`, "title": `"v1-topic-910000201"`, "state": `"published"`, "category": `"galgame"`,
			"sections": `["g-walkthrough"]`, "cover_images": cover, "mini_apps": `["poll"]`,
			"author":     `{"object":"user","id":"910000001","name":"alice","avatar":null,"avatar_frame":null}`,
			"view_count": `10`, "like_count": `1`, "reply_count": `0`, "comment_count": `0`,
			"has_best_answer": `false`, "is_nsfw": `false`,
			"bumped_at": `"2026-01-15T15:00:00Z"`, "created_at": `"2026-01-15T12:00:00Z"`, "upvoted_at": `"2026-01-15T12:00:00Z"`,
			"edited_at": `"2026-01-15T14:00:00Z"`,
		},
		"910000202": {"sections": `[]`, "cover_images": `[]`, "mini_apps": `["lottery"]`, "upvoted_at": `null`, "edited_at": `null`},
		"910000203": {
			"has_best_answer": `true`,
			"cover_images": `[{"url":"https://image.test.example/bb/bb/` + v1CoverHash2 + `.webp","hash":"` + v1CoverHash2 +
				`","width":null,"height":null,"thumbhash":null,"sexual":null}]`,
		},
		"910000204": {"category": `"others"`, "view_count": `30`},
		"910000213": {"author": `{"object":"user","id":"910000003","name":"bob","avatar":null,"avatar_frame":null}`},
		"910000217": {"author": `{"object":"user","id":"910000004","name":null,"avatar":null,"avatar_frame":null}`},
	} {
		item, ok := byID[id]
		if !ok {
			t.Errorf("topic %s missing from the list", id)
			continue
		}
		for field, w := range want {
			if got := string(item[field]); got != w {
				t.Errorf("topic %s %s = %s, want %s", id, field, got, w)
			}
		}
	}
	if _, ok := byID["910000210"]; ok {
		t.Error("the banned author's topic was listed")
	}
}
