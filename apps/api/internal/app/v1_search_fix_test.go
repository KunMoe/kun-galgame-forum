package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"kun-galgame-api/internal/community/anchor"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/middleware"
	searchapiv1 "kun-galgame-api/internal/search/apiv1"
	searchRepo "kun-galgame-api/internal/search/repository"
	"kun-galgame-api/internal/testdb"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/userclient"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	srUserAlice  = 952000001
	srUserBob    = 952000002
	srUserBanned = 952000009

	srTopicTitle   = 952000101
	srTopicBody    = 952000102
	srTopicHidden  = 952000103
	srTopicNSFW    = 952000104
	srTopicLogin   = 952000105
	srTopicBanned  = 952000106
	srTopicTieMin  = 952000110
	srTopicTieMax  = 952000116
	srReplyMin     = 952000201
	srReplyTieMin  = 952000210
	srReplyTieMax  = 952000216
	srCommentMin   = 952000301
	srWallWork     = 61311
	srImageHash    = "abababababababababababababababababababababababababababababababab"
	srTopicPath    = "/search/topics"
	srRepliesPath  = "/search/replies"
	srCommentsPath = "/search/comments"
	srUsersPath    = "/search/users"
	srWorksPath    = "/search/works"
	srWallPath     = "/search/wall-comments"
)

type searchFix struct {
	app   *App
	db    *gorm.DB
	rdb   *redis.Client
	spec  *specConformance
	oauth struct {
		mu       sync.Mutex
		search   []map[string]any
		fail     atomic.Bool
		refuse   atomic.Bool
		searched atomic.Int32
	}
	catalog struct {
		mu    sync.Mutex
		query url.Values
		body  string
		fail  atomic.Bool
	}
	community struct {
		mu    sync.Mutex
		query url.Values
		pages map[string]string
		fail  atomic.Bool
	}
}

func srUsers() map[int]map[string]any {
	return map[int]map[string]any{
		srUserAlice:  {"name": "alice", "status": 0},
		srUserBob:    {"name": "bob", "status": 0},
		srUserBanned: {"name": "banned", "status": 1},
	}
}

func newSearchFix(t *testing.T) *searchFix {
	t.Helper()
	f := &searchFix{db: testdb.Open(t)}
	mr := miniredis.RunT(t)
	f.rdb = redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: 0})
	t.Cleanup(func() { _ = f.rdb.Close() })

	oauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f.oauth.fail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users/batch":
			known := srUsers()
			users := []map[string]any{}
			for _, raw := range strings.Split(r.URL.Query().Get("ids"), ",") {
				id, _ := strconv.Atoi(raw)
				if u, ok := known[id]; ok {
					users = append(users, map[string]any{"id": id, "name": u["name"], "status": u["status"], "roles": []string{"user"}})
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"users": users, "not_found": []int{}}})
		case "/users/search":
			f.oauth.searched.Add(1)
			if f.oauth.refuse.Load() || utf8.RuneCountInString(strings.TrimSpace(r.URL.Query().Get("q"))) > 50 {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]any{"code": 9, "message": "q: max 50 chars"})
				return
			}
			f.oauth.mu.Lock()
			users := f.oauth.search
			f.oauth.mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"users": users}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(oauth.Close)

	catalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f.catalog.fail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		f.catalog.mu.Lock()
		f.catalog.query = r.URL.Query()
		body := f.catalog.body
		f.catalog.mu.Unlock()
		if body == "" {
			body = `{"items":[],"total":0}`
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(catalog.Close)

	community := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f.community.fail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		f.community.mu.Lock()
		f.community.query = r.URL.Query()
		data, ok := f.community.pages[r.URL.Query().Get("cursor")]
		f.community.mu.Unlock()
		if !ok {
			data = `{"posts":[],"next_cursor":""}`
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"message":"","data":`+data+`}`)
	}))
	t.Cleanup(community.Close)

	uc := userclient.New(userclient.Config{
		BaseURL: oauth.URL, ClientID: "c", ClientSecret: "s",
		ImageCDNBase: "https://image.test.example", HTTPTimeout: 2 * time.Second,
	})
	gc := client.New(catalog.URL, "nm_test_key", "https://image.test.example")
	cc := communityclient.New(communityclient.Config{BaseURL: community.URL, ClientID: "c", ClientSecret: "s"})
	cfg := testConfig()
	cfg.NextMoeAPI.ImageCDNBase = "https://image.test.example"
	f.app = &App{
		Fiber:      newFiber(),
		Config:     cfg,
		DB:         f.db,
		Redis:      f.rdb,
		UserClient: uc,
		Authn:      middleware.NewAuthenticator(f.rdb, nil, middleware.NewBearer(w3Verifier{}, f.rdb, nil)),
	}
	f.app.SearchV1 = searchapiv1.New(searchapiv1.Deps{
		Repo: searchRepo.NewSearchRepository(f.db), Topics: f.app.newTopicV1(), Users: uc, Galgame: gc,
		Community: cc, Anchors: anchor.New(f.db, gc), CDN: "https://image.test.example",
	})
	f.app.setupRoutes()
	f.spec = newSpecConformance(t)
	f.seed(t)
	f.putSession(t, "sess-sr-alice", srUserAlice)
	return f
}

func (f *searchFix) putSession(t *testing.T, token string, uid int) {
	t.Helper()
	data, err := json.Marshal(middleware.SessionData{
		UserInfo:         middleware.UserInfo{ID: uid, Name: "n", Roles: []string{"user"}},
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

func (f *searchFix) cleanup() {
	_ = f.db.Exec(`DELETE FROM topic_comment WHERE id BETWEEN 952000001 AND 952000999`).Error
	_ = f.db.Exec(`DELETE FROM topic_reply WHERE id BETWEEN 952000001 AND 952000999`).Error
	_ = f.db.Exec(`DELETE FROM topic WHERE id BETWEEN 952000001 AND 952000999`).Error
}

func (f *searchFix) seed(t *testing.T) {
	t.Helper()
	f.cleanup()
	t.Cleanup(f.cleanup)
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, q)
		}
	}
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	topic := func(id, user, status int, scope string, nsfw bool, title, body string, bumped time.Time) {
		t.Helper()
		run(`INSERT INTO topic (
			id, title, content, view, status, category, status_update_time, created, updated,
			user_id, is_nsfw, access_scope, cover_images, like_count, dislike_count, reply_count, comment_count,
			favorite_count, upvote_count, view_7d, view_30d, hidden_by, last_reply_floor
		) VALUES (?, ?, ?, 0, ?, 'galgame', ?, ?, ?, ?, ?, ?, '', 0, 0, 0, 0, 0, 0, 0, 0, ?, 0)`,
			id, title, body, status, bumped, bumped, bumped, user, nsfw, scope, map[bool]string{true: "author", false: ""}[status == 1])
	}
	topic(srTopicTitle, srUserAlice, 0, "public", false, "xsrchalpha in the title", "body", base)
	topic(srTopicBody, srUserBob, 0, "public", false, "plain title", "only the body has xsrchalpha", base.Add(time.Hour))
	topic(srTopicHidden, srUserAlice, 1, "public", false, "xsrchalpha hidden", "body", base)
	topic(srTopicNSFW, srUserAlice, 0, "public", true, "xsrchalpha nsfw", "body", base)
	topic(srTopicLogin, srUserAlice, 0, "login", false, "xsrchalpha login only", "body", base)
	topic(srTopicBanned, srUserBanned, 0, "public", false, "xsrchalpha by banned", "body", base)
	// Seven topics with the same title and bump time, so a page boundary lands
	// inside the tie and only the id tie-breaker keeps the walk stable.
	for id := srTopicTieMin; id <= srTopicTieMax; id++ {
		topic(id, srUserBob, 0, "public", false, "xsrchtie", "body", base)
	}

	reply := func(id, topicID, user, floor int, body string, at time.Time) {
		t.Helper()
		run(`INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, like_count, created, updated)
			VALUES (?, ?, ?, ?, ?, 0, 0, ?, ?)`, id, body, floor, user, topicID, at, at)
	}
	long := strings.Repeat("填充文字", 80)
	reply(srReplyMin, srTopicTitle, srUserBob, 1, "**xsrchreply** bold ![](/image/"+srImageHash+") "+long, base)
	reply(srReplyMin+1, srTopicHidden, srUserBob, 1, "xsrchreply under a hidden topic", base)
	reply(srReplyMin+2, srTopicNSFW, srUserBob, 1, "xsrchreply under an nsfw topic", base)
	reply(srReplyMin+3, srTopicTitle, srUserBanned, 2, "xsrchreply by a banned author", base)
	reply(srReplyMin+4, srTopicTitle, srUserBob, 3, long+" deep xsrchreply hit "+long, base.Add(-time.Hour))
	for id := srReplyTieMin; id <= srReplyTieMax; id++ {
		reply(id, srTopicBody, srUserAlice, id-srReplyTieMin+10, "zzreplytie", base)
	}

	comment := func(id, topicID, user int, body string) {
		t.Helper()
		run(`INSERT INTO topic_comment (id, content, topic_id, topic_reply_id, user_id, target_user_id, parent_comment_id, status, created, updated)
			VALUES (?, ?, ?, ?, ?, ?, NULL, 0, ?, ?)`, id, body, topicID, srReplyMin, user, srUserBob, base, base)
	}
	comment(srCommentMin, srTopicTitle, srUserBob, "a xsrchcomm comment")
	comment(srCommentMin+1, srTopicHidden, srUserBob, "xsrchcomm under hidden")
	comment(srCommentMin+2, srTopicLogin, srUserBob, "xsrchcomm under login-only")
}

func (f *searchFix) get(t *testing.T, specPath, rawURL, session string) (*http.Response, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1"+rawURL, nil)
	if session != "" {
		req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: session})
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
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("not a JSON object: %v\n%s", err, body)
	}
	return resp, out
}

func itemsOf(body map[string]any) []map[string]any {
	raw, _ := body["items"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, it := range raw {
		m, _ := it.(map[string]any)
		out = append(out, m)
	}
	return out
}

func idsOf(body map[string]any) []string {
	var out []string
	for _, it := range itemsOf(body) {
		out = append(out, fmt.Sprint(it["id"]))
	}
	return out
}

func wantProblemCode(t *testing.T, resp *http.Response, body map[string]any, status int, code string) {
	t.Helper()
	if resp.StatusCode != status || body["code"] != code {
		t.Fatalf("got %d %v, want %d %s: %+v", resp.StatusCode, body["code"], status, code, body)
	}
}

func wantErrorAt(t *testing.T, body map[string]any, where, reason string) {
	t.Helper()
	errs, _ := body["errors"].([]any)
	for _, e := range errs {
		m, _ := e.(map[string]any)
		if (m["parameter"] == where || m["pointer"] == where) && m["reason"] == reason {
			return
		}
	}
	t.Fatalf("no error at %s with %s: %+v", where, reason, body)
}
