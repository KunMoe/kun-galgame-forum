package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/testdb"
	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/pkg/userclient"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	e1UserAlice  = 940000001
	e1UserBanned = 940000002
	e1UserBob    = 940000003
	e1UserGone   = 940000004
	e1UserCarol  = 940000005
	e1UserDave   = 940000006
	e1UserStaff  = 940000008
	e1UserPoor   = 940000009

	e1TopicPublic       = 940000201
	e1TopicHiddenAuthor = 940000202
	e1TopicHiddenMod    = 940000203
	e1TopicLogin        = 940000204
	e1TopicOther        = 940000205
	e1TopicOld          = 940000206
	e1TopicConcurrent   = 940000207
	e1TopicLists        = 940000208
	e1TopicMigrateA     = 940000209
	e1TopicMigrateB     = 940000210

	e1ReplyDave       = 940000301
	e1ReplyCarol      = 940000302
	e1ReplyAlice      = 940000303
	e1ReplyHidden     = 940000304
	e1ReplyOtherTopic = 940000305
	e1ReplyBanned     = 940000306
	e1ReplyOnHidden   = 940000307
	e1ReplyReact      = 940000308
	e1ReplyMigrate    = 940000309

	e1UserMin  = 940000001
	e1UserMax  = 940000099
	e1TopicMin = 940000201
	e1TopicMax = 940000299
	e1ReplyMin = 940000301
	e1ReplyMax = 940000399
)

type engageVerifier struct{}

func (engageVerifier) Verify(_ context.Context, raw string) (*oauth.AccessClaims, error) {
	switch raw {
	case "staff-token":
		return &oauth.AccessClaims{
			ID: e1UserStaff, Name: "staff", Roles: []string{"user"},
			SiteRoles: []string{"moderator"}, ClientID: "kungal-app",
		}, nil
	case "bob-token":
		return &oauth.AccessClaims{
			ID: e1UserBob, Name: "bob", Roles: []string{"user"}, ClientID: "kungal-app",
		}, nil
	case "alice-token":
		return &oauth.AccessClaims{
			ID: e1UserAlice, Name: "alice", Roles: []string{"user"}, ClientID: "kungal-app",
		}, nil
	default:
		return nil, fmt.Errorf("bad token")
	}
}

type engageAward struct {
	UserID int
	Delta  int
	Reason string
	Ref    string
	Key    string
}

type awardRecorder struct {
	mu    sync.Mutex
	calls []engageAward
}

func (r *awardRecorder) Award(userID, delta int, reason, ref, key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, engageAward{userID, delta, reason, ref, key})
}

func (r *awardRecorder) snapshot() []engageAward {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]engageAward, len(r.calls))
	copy(out, r.calls)
	return out
}

type engageFix struct {
	*App
	db     *gorm.DB
	rdb    *redis.Client
	spec   *specConformance
	oauth  *httptest.Server
	awards *awardRecorder
}

func newEngageFix(t *testing.T) *engageFix {
	t.Helper()
	db := testdb.Open(t)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: 0})
	t.Cleanup(func() { _ = rdb.Close() })

	rec := &awardRecorder{}

	f := &engageFix{db: db, rdb: rdb, awards: rec}
	mux := http.NewServeMux()
	mux.HandleFunc("/users/batch", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"users": []map[string]any{
					{"id": e1UserAlice, "name": "alice", "status": 0, "roles": []string{"user"}},
					{"id": e1UserBanned, "name": "banned", "status": 1, "roles": []string{"user"}},
					{"id": e1UserBob, "name": "bob", "status": 0, "roles": []string{"user"}},
					{"id": e1UserCarol, "name": "carol", "status": 0, "roles": []string{"user"}},
					{"id": e1UserDave, "name": "dave", "status": 0, "roles": []string{"user"}},
					{"id": e1UserStaff, "name": "staff", "status": 0, "roles": []string{"moderator"}},
					{"id": e1UserPoor, "name": "poor", "status": 0, "roles": []string{"user"}},
				},
				"not_found": []int{},
			},
		})
	})
	f.oauth = httptest.NewServer(mux)
	t.Cleanup(f.oauth.Close)

	uc := userclient.New(userclient.Config{
		BaseURL: f.oauth.URL, ClientID: "test-client", ClientSecret: "test-secret",
		ImageCDNBase: "https://image.test.example", HTTPTimeout: 2 * time.Second,
	})
	cfg := testConfig()
	cfg.NextMoeAPI.ImageCDNBase = "https://image.test.example"
	f.App = &App{
		Fiber:      newFiber(),
		Config:     cfg,
		DB:         db,
		Redis:      rdb,
		UserClient: uc,
		Authn:      middleware.NewAuthenticator(rdb, nil, middleware.NewBearer(engageVerifier{}, rdb, nil)),
		TopicAward: rec.Award,
	}
	f.setupRoutes()
	f.spec = newSpecConformance(t)
	f.seed(t)
	return f
}

func (f *engageFix) seed(t *testing.T) {
	t.Helper()
	f.cleanup(t)
	t.Cleanup(func() { f.cleanup(t) })
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	old := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, q)
		}
	}
	for _, u := range []struct {
		id, moe int
	}{
		{e1UserAlice, 100}, {e1UserBanned, 100}, {e1UserBob, 100}, {e1UserCarol, 100},
		{e1UserDave, 100}, {e1UserStaff, 100}, {e1UserPoor, 9},
	} {
		run(`INSERT INTO kungal_user_state (user_id, moemoepoint, created, updated)
			VALUES (?, ?, ?, ?) ON CONFLICT (user_id) DO UPDATE SET moemoepoint = EXCLUDED.moemoepoint`,
			u.id, u.moe, base, base)
	}
	insTopic := func(id, user, status int, scope, hiddenBy, title string, created time.Time) {
		t.Helper()
		run(`INSERT INTO topic (
			id, title, content, view, status, category, status_update_time, created, updated,
			user_id, is_nsfw, access_scope, cover_images, like_count, dislike_count, reply_count, comment_count,
			favorite_count, upvote_count, view_7d, view_30d, upvote_time, hidden_by, edited
		) VALUES (?, ?, ?, 0, ?, 'galgame', ?, ?, ?, ?, false, ?, '', 0, 0, 0, 0, 0, 0, 0, 0, NULL, ?, NULL)`,
			id, title, title+"-body", status, created, created, created, user, scope, hiddenBy)
	}
	insTopic(e1TopicPublic, e1UserAlice, 0, "public", "", "e1-public", base)
	insTopic(e1TopicHiddenAuthor, e1UserAlice, 1, "public", "author", "e1-hidden-author", base)
	insTopic(e1TopicHiddenMod, e1UserAlice, 1, "public", "moderator", "e1-hidden-mod", base)
	insTopic(e1TopicLogin, e1UserAlice, 0, "login", "", "e1-login", base)
	insTopic(e1TopicOther, e1UserAlice, 0, "public", "", "e1-other", base)
	insTopic(e1TopicOld, e1UserAlice, 0, "public", "", "e1-old", old)
	insTopic(e1TopicConcurrent, e1UserAlice, 0, "public", "", "e1-concurrent", base)
	insTopic(e1TopicLists, e1UserAlice, 0, "public", "", "e1-lists", base)
	insTopic(e1TopicMigrateA, e1UserAlice, 0, "public", "", "e1-migrate-a", base)
	insTopic(e1TopicMigrateB, e1UserAlice, 0, "public", "", "e1-migrate-b", base)
	fileUnsectionedTopics(t, f.db, e1TopicMin, e1TopicMax)

	insReply := func(id, user, topic, floor, status int, body string) {
		t.Helper()
		run(`INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, like_count, dislike_count, created, updated)
			VALUES (?, ?, ?, ?, ?, ?, 0, 0, ?, ?)`, id, body, floor, user, topic, status, base, base)
	}
	insReply(e1ReplyDave, e1UserDave, e1TopicPublic, 1, 0, "dave-reply")
	insReply(e1ReplyCarol, e1UserCarol, e1TopicPublic, 2, 0, "carol-reply")
	insReply(e1ReplyAlice, e1UserAlice, e1TopicPublic, 3, 0, "alice-reply")
	insReply(e1ReplyHidden, e1UserDave, e1TopicPublic, 4, 1, "hidden-reply")
	insReply(e1ReplyOtherTopic, e1UserDave, e1TopicOther, 1, 0, "other-topic-reply")
	insReply(e1ReplyBanned, e1UserBanned, e1TopicPublic, 5, 0, "banned-reply")
	insReply(e1ReplyOnHidden, e1UserDave, e1TopicHiddenAuthor, 1, 0, "on-hidden")
	insReply(e1ReplyReact, e1UserDave, e1TopicPublic, 6, 0, "react-reply")
	insReply(e1ReplyMigrate, e1UserDave, e1TopicMigrateA, 1, 0, "migrate-reply")
}

func (f *engageFix) cleanup(t *testing.T) {
	t.Helper()
	for _, q := range []string{
		"DELETE FROM message WHERE sender_id BETWEEN ? AND ? OR receiver_id BETWEEN ? AND ?",
		"DELETE FROM topic_upvote WHERE topic_id BETWEEN ? AND ?",
		"DELETE FROM topic_favorite WHERE topic_id BETWEEN ? AND ?",
		"DELETE FROM topic_reaction WHERE topic_id BETWEEN ? AND ?",
		"DELETE FROM topic_reply_reaction WHERE topic_reply_id BETWEEN ? AND ?",
		"DELETE FROM topic_like WHERE topic_id BETWEEN ? AND ?",
		"DELETE FROM topic_dislike WHERE topic_id BETWEEN ? AND ?",
		"DELETE FROM topic_reply_like WHERE topic_reply_id BETWEEN ? AND ?",
		"DELETE FROM topic_reply_dislike WHERE topic_reply_id BETWEEN ? AND ?",
		"UPDATE topic SET best_answer_id = NULL, pinned_reply_id = NULL WHERE id BETWEEN ? AND ?",
		"DELETE FROM topic_reply WHERE id BETWEEN ? AND ?",
		"DELETE FROM topic WHERE id BETWEEN ? AND ?",
	} {
		a, b := e1TopicMin, e1TopicMax
		switch {
		case strings.Contains(q, "message"):
			a, b = e1UserMin, e1UserMax
			if err := f.db.Exec(q, a, b, a, b).Error; err != nil {
				t.Errorf("cleanup %q: %v", q, err)
			}
			continue
		case strings.Contains(q, "topic_reply_reaction") || strings.Contains(q, "topic_reply_like") ||
			strings.Contains(q, "topic_reply_dislike") || strings.Contains(q, "DELETE FROM topic_reply"):
			a, b = e1ReplyMin, e1ReplyMax
		}
		if err := f.db.Exec(q, a, b).Error; err != nil {
			t.Errorf("cleanup %q: %v", q, err)
		}
	}
	_ = f.db.Exec("DELETE FROM kungal_user_state WHERE user_id BETWEEN ? AND ?", e1UserMin, e1UserMax).Error
}

func (f *engageFix) putSession(t *testing.T, token string, userID int, roles ...string) {
	t.Helper()
	if len(roles) == 0 {
		roles = []string{"user"}
	}
	data, err := json.Marshal(middleware.SessionData{
		UserInfo:         middleware.UserInfo{ID: userID, Name: "n", Roles: roles},
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

func (f *engageFix) do(t *testing.T, method, rawURL, session, specPath string, body []byte, hdr http.Header) (*http.Response, []byte) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, rawURL, rdr)
	if session != "" {
		req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: session})
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, vs := range hdr {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	resp, err := f.Fiber.Test(req, fiber.TestConfig{Timeout: 15 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("%s Cache-Control %q", rawURL, resp.Header.Get("Cache-Control"))
	}
	f.spec.checkPath(t, method, specPath, resp, out)
	return resp, out
}

func (f *engageFix) asBob(t *testing.T) string {
	t.Helper()
	f.putSession(t, "sess-bob", e1UserBob)
	return "sess-bob"
}

func (f *engageFix) asAlice(t *testing.T) string {
	t.Helper()
	f.putSession(t, "sess-alice", e1UserAlice)
	return "sess-alice"
}

func (f *engageFix) asStaff(t *testing.T) string {
	t.Helper()
	f.putSession(t, "sess-staff", e1UserStaff, "moderator")
	return "sess-staff"
}

func (f *engageFix) asPoor(t *testing.T) string {
	t.Helper()
	f.putSession(t, "sess-poor", e1UserPoor)
	return "sess-poor"
}

func (f *engageFix) asDave(t *testing.T) string {
	t.Helper()
	f.putSession(t, "sess-dave", e1UserDave)
	return "sess-dave"
}

func (f *engageFix) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	if err := f.db.Raw(q, args...).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func (f *engageFix) runSQL(t *testing.T, q string, args ...any) {
	t.Helper()
	if err := f.db.Exec(q, args...).Error; err != nil {
		t.Fatalf("sql: %v\n%s", err, q)
	}
}

func (f *engageFix) topicInt(t *testing.T, id int, col string) int {
	t.Helper()
	var n int
	if err := f.db.Raw("SELECT "+col+" FROM topic WHERE id = ?", id).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func (f *engageFix) topicNullInt(t *testing.T, id int, col string) *int {
	t.Helper()
	var n *int
	if err := f.db.Raw("SELECT "+col+" FROM topic WHERE id = ?", id).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func (f *engageFix) topicTime(t *testing.T, id int, col string) time.Time {
	t.Helper()
	var ts time.Time
	if err := f.db.Raw("SELECT "+col+" FROM topic WHERE id = ?", id).Scan(&ts).Error; err != nil {
		t.Fatal(err)
	}
	return ts
}

func (f *engageFix) xmin(t *testing.T, table string, id int) string {
	t.Helper()
	var s string
	if err := f.db.Raw("SELECT xmin::text FROM "+table+" WHERE id = ?", id).Scan(&s).Error; err != nil {
		t.Fatal(err)
	}
	return s
}

func (f *engageFix) sqlIDs(t *testing.T, q string, args ...any) []string {
	t.Helper()
	var ids []int
	if err := f.db.Raw(q, args...).Scan(&ids).Error; err != nil {
		t.Fatal(err)
	}
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = strID(id)
	}
	return out
}

func (f *engageFix) replyInt(t *testing.T, id int, col string) int {
	t.Helper()
	var n int
	if err := f.db.Raw("SELECT "+col+" FROM topic_reply WHERE id = ?", id).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func (f *engageFix) reactionRowID(t *testing.T, replyID, userID int, tok string) int {
	t.Helper()
	var id int
	if err := f.db.Raw(
		`SELECT id FROM topic_reply_reaction WHERE topic_reply_id = ? AND user_id = ? AND reaction = ?`,
		replyID, userID, tok,
	).Scan(&id).Error; err != nil {
		t.Fatal(err)
	}
	return id
}

func (f *engageFix) favoriteRowID(t *testing.T, topicID, userID int) int {
	t.Helper()
	var id int
	if err := f.db.Raw(
		`SELECT id FROM topic_favorite WHERE topic_id = ? AND user_id = ?`,
		topicID, userID,
	).Scan(&id).Error; err != nil {
		t.Fatal(err)
	}
	return id
}

func (f *engageFix) messages(t *testing.T, receiver int, typ string) []map[string]any {
	t.Helper()
	var rows []struct {
		Type    string
		Sender  int
		Content string
		Link    string
	}
	if err := f.db.Raw(`SELECT type, sender_id AS sender, content, link FROM message
		WHERE receiver_id = ? AND type = ? ORDER BY id`, receiver, typ).Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{"type": r.Type, "sender": r.Sender, "content": r.Content, "link": r.Link})
	}
	return out
}

func jsonObj(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("json: %v\n%s", err, body)
	}
	return m
}

func authBearer(token string) http.Header {
	h := http.Header{}
	h.Set("Authorization", "Bearer "+token)
	return h
}

func idemKey(k string) http.Header {
	h := http.Header{}
	h.Set("Idempotency-Key", k)
	return h
}
