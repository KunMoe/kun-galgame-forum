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
	"sync/atomic"
	"testing"
	"time"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/testdb"
	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/userclient"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"gorm.io/gorm"
)

const (
	d1UserAlice  = 920000001
	d1UserBanned = 920000002
	d1UserBob    = 920000003
	d1UserGone   = 920000004
	d1UserCarol  = 920000005
	d1UserStaff  = 920000008
	d1TopicMin   = 920000201
	d1TopicMax   = 920000299
	d1ReplyMin   = 920000301
	d1ReplyMax   = 920000399
	d1CommentMin = 920000401
	d1CommentMax = 920000499
	d1CoverHash  = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"

	d1TopicPublic = 920000201
	d1TopicHidden = 920000220
	d1TopicLogin  = 920000221
	d1TopicRole   = 920000222
	d1TopicUsers  = 920000223
	d1TopicBanned = 920000224
	d1TopicGone   = 920000225
	d1TopicList   = 920000230
	d1TopicFill   = 920000231

	d1ReplyPinned = 920000301
	d1ReplyBest   = 920000302
	d1ReplyHidden = 920000360
	d1ReplyBanGet = 920000361
	d1ReplyOnHid  = 920000362
)

type detailVerifier struct{}

func (detailVerifier) Verify(_ context.Context, raw string) (*oauth.AccessClaims, error) {
	switch raw {
	case "staff-token":
		return &oauth.AccessClaims{
			ID: d1UserStaff, Name: "staff", Roles: []string{"user"},
			SiteRoles: []string{"moderator"}, ClientID: "kungal-app",
		}, nil
	case "bob-token":
		return &oauth.AccessClaims{
			ID: d1UserBob, Name: "bob", Roles: []string{"user"}, ClientID: "kungal-app",
		}, nil
	default:
		return nil, fmt.Errorf("bad token")
	}
}

type detailFix struct {
	*App
	db         *gorm.DB
	rdb        *redis.Client
	spec       *specConformance
	oauth      *httptest.Server
	batchCalls atomic.Int32
	failOA     atomic.Bool
}

func newDetailFix(t *testing.T) *detailFix {
	t.Helper()
	db := testdb.Open(t)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: 0})
	t.Cleanup(func() { _ = rdb.Close() })

	f := &detailFix{db: db, rdb: rdb}
	mux := http.NewServeMux()
	mux.HandleFunc("/users/batch", func(w http.ResponseWriter, _ *http.Request) {
		f.batchCalls.Add(1)
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
					{"id": d1UserAlice, "name": "alice", "status": 0, "roles": []string{"user"}},
					{"id": d1UserBanned, "name": "banned", "status": 1, "roles": []string{"user"}},
					{"id": d1UserBob, "name": "bob", "status": 0, "roles": []string{"user"}},
					{"id": d1UserCarol, "name": "carol", "status": 0, "roles": []string{"user"}},
					{"id": d1UserStaff, "name": "staff", "status": 0, "roles": []string{"moderator"}},
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
	sexual := int16(0)
	f.App = &App{
		Fiber:      newFiber(),
		Config:     cfg,
		DB:         db,
		Redis:      rdb,
		UserClient: uc,
		Authn:      middleware.NewAuthenticator(rdb, nil, middleware.NewBearer(detailVerifier{}, rdb, nil)),
		ImageMeta: func(hashes []string) map[string]imageclient.ImageMeta {
			out := map[string]imageclient.ImageMeta{}
			for _, h := range hashes {
				if h == d1CoverHash {
					out[h] = imageclient.ImageMeta{Width: 640, Height: 360, Thumbhash: "AbC+", Sexual: &sexual}
				}
			}
			return out
		},
	}
	f.setupRoutes()
	f.spec = newSpecConformance(t)
	f.seed(t)
	return f
}

func (f *detailFix) seed(t *testing.T) {
	t.Helper()
	f.cleanup(t)
	t.Cleanup(func() { f.cleanup(t) })
	base := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, q)
		}
	}

	run(`INSERT INTO kungal_user_state (user_id, moemoepoint, created, updated)
		VALUES (?, -16, ?, ?) ON CONFLICT (user_id) DO UPDATE SET moemoepoint = EXCLUDED.moemoepoint`,
		d1UserAlice, base, base)

	insTopic := func(id, user, view, like, dislike, fav, up, replies, comments, status int, cat, scope, hiddenBy, cover, title, body string, edited *time.Time) {
		t.Helper()
		run(`INSERT INTO topic (
			id, title, content, view, status, category, status_update_time, created, updated,
			user_id, is_nsfw, access_scope, cover_images, like_count, dislike_count, reply_count, comment_count,
			favorite_count, upvote_count, view_7d, view_30d, upvote_time, hidden_by, edited
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, false, ?, ?, ?, ?, ?, ?, ?, ?, 0, 0, ?, ?, ?)`,
			id, title, body, view, status, cat, base.Add(3*time.Hour), base, base,
			user, scope, cover, like, dislike, replies, comments, fav, up, &base, hiddenBy, edited)
	}
	edited := base.Add(time.Hour)
	insTopic(d1TopicPublic, d1UserAlice, 10, 4, 0, 3, 4, 5, 6, 0, "galgame", "public", "",
		`["/image/`+d1CoverHash+`"]`, "d1-public", "hello", &edited)
	insTopic(d1TopicHidden, d1UserAlice, 1, 0, 0, 0, 0, 0, 0, 1, "galgame", "public", "author",
		"", "d1-hidden", "hidden-body", nil)
	insTopic(d1TopicLogin, d1UserAlice, 7, 0, 0, 0, 0, 0, 0, 0, "galgame", "login", "",
		"", "d1-login", "login-body", nil)
	insTopic(d1TopicRole, d1UserAlice, 1, 0, 0, 0, 0, 0, 0, 0, "galgame", "role", "",
		"", "d1-role", "role-body", nil)
	insTopic(d1TopicUsers, d1UserAlice, 1, 0, 0, 0, 0, 0, 0, 0, "galgame", "users", "",
		"", "d1-users", "users-body", nil)
	insTopic(d1TopicBanned, d1UserBanned, 1, 0, 0, 0, 0, 0, 0, 0, "galgame", "public", "",
		"", "d1-banned", "banned-body", nil)
	insTopic(d1TopicGone, d1UserGone, 1, 0, 0, 0, 0, 0, 0, 0, "galgame", "public", "",
		"", "d1-gone", "gone-body", nil)
	insTopic(d1TopicList, d1UserAlice, 1, 0, 0, 0, 0, 0, 0, 0, "galgame", "public", "",
		"", "d1-list", "list-body", nil)
	insTopic(d1TopicFill, d1UserAlice, 1, 0, 0, 0, 0, 0, 0, 0, "galgame", "public", "",
		"", "d1-fill", "fill-body", nil)

	run(`INSERT INTO topic_section_relation (topic_id, topic_section_id, created, updated) VALUES (?, ?, ?, ?)`,
		d1TopicPublic, sectionID(t, f.db, "news"), base, base)
	fileUnsectionedTopics(t, f.db, d1TopicMin, d1TopicMax)
	run(`INSERT INTO topic_poll (title, topic_id, user_id, created, updated) VALUES ('p', ?, ?, ?, ?)`,
		d1TopicPublic, d1UserAlice, base, base)
	run(`INSERT INTO topic_access_grant (topic_id, subject_type, subject_value) VALUES (?, 'role', 'creator')`, d1TopicRole)
	run(`INSERT INTO topic_access_grant (topic_id, subject_type, subject_value) VALUES (?, 'user', ?)`,
		d1TopicUsers, strconv.Itoa(d1UserBob))

	run(`INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, like_count, dislike_count, created, updated)
		VALUES (?, 'pin-hi', 2, ?, ?, 0, 1, 1, ?, ?)`, d1ReplyPinned, d1UserBob, d1TopicPublic, base, base)
	run(`INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, like_count, dislike_count, created, updated)
		VALUES (?, 'best-hi', 5, ?, ?, 0, 0, 0, ?, ?)`, d1ReplyBest, d1UserAlice, d1TopicPublic, base.Add(time.Second), base)
	run(`UPDATE topic SET pinned_reply_id = ?, best_answer_id = ? WHERE id = ?`, d1ReplyPinned, d1ReplyBest, d1TopicPublic)

	run(`INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, created, updated)
		VALUES (?, 'hidden-reply', 9, ?, ?, 1, ?, ?)`, d1ReplyHidden, d1UserAlice, d1TopicPublic, base, base)
	run(`INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, created, updated)
		VALUES (?, 'banned-reply', 1, ?, ?, 0, ?, ?)`, d1ReplyBanGet, d1UserBanned, d1TopicPublic, base, base)
	run(`INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, created, updated)
		VALUES (?, 'on-hidden', 1, ?, ?, 0, ?, ?)`, d1ReplyOnHid, d1UserAlice, d1TopicHidden, base, base)

	parent, bannedC, orphan, child := 920000401, 920000403, 920000404, 920000402
	run(`INSERT INTO topic_comment (id, content, topic_id, topic_reply_id, user_id, target_user_id, parent_comment_id, status, created, updated)
		VALUES (?, 'parent-text', ?, ?, ?, ?, NULL, 0, ?, ?)`,
		parent, d1TopicPublic, d1ReplyPinned, d1UserAlice, d1UserBob, base, base)
	run(`INSERT INTO topic_comment (id, content, topic_id, topic_reply_id, user_id, target_user_id, parent_comment_id, status, created, updated)
		VALUES (?, 'banned-text', ?, ?, ?, ?, NULL, 0, ?, ?)`,
		bannedC, d1TopicPublic, d1ReplyPinned, d1UserBanned, d1UserBob, base.Add(time.Second), base)
	run(`INSERT INTO topic_comment (id, content, topic_id, topic_reply_id, user_id, target_user_id, parent_comment_id, status, created, updated)
		VALUES (?, 'orphan-text', ?, ?, ?, ?, ?, 0, ?, ?)`,
		orphan, d1TopicPublic, d1ReplyPinned, d1UserCarol, d1UserBanned, bannedC, base.Add(2*time.Second), base)
	run(`INSERT INTO topic_comment (id, content, topic_id, topic_reply_id, user_id, target_user_id, parent_comment_id, status, created, updated)
		VALUES (?, 'child-text', ?, ?, ?, ?, ?, 0, ?, ?)`,
		child, d1TopicPublic, d1ReplyPinned, d1UserBob, d1UserAlice, parent, base.Add(3*time.Second), base)
	run(`INSERT INTO topic_comment (id, content, topic_id, topic_reply_id, user_id, target_user_id, parent_comment_id, status, created, updated)
		VALUES (?, 'gone-target', ?, ?, ?, ?, NULL, 0, ?, ?)`,
		920000405, d1TopicPublic, d1ReplyPinned, d1UserAlice, d1UserGone, base.Add(4*time.Second), base)
	run(`INSERT INTO topic_comment_like (topic_comment_id, user_id, created, updated) VALUES (?, ?, ?, ?), (?, ?, ?, ?)`,
		parent, d1UserAlice, base, base, parent, d1UserBob, base, base)

	tLike := base
	run(`INSERT INTO topic_reaction (topic_id, user_id, reaction, created) VALUES
		(?, ?, 'like', ?), (?, ?, 'like', ?), (?, ?, 'like', ?), (?, ?, 'like', ?), (?, ?, 'heart', ?)`,
		d1TopicPublic, d1UserAlice, tLike,
		d1TopicPublic, d1UserBanned, tLike.Add(time.Second),
		d1TopicPublic, d1UserBob, tLike.Add(2*time.Second),
		d1TopicPublic, d1UserCarol, tLike.Add(3*time.Second),
		d1TopicPublic, d1UserBob, tLike.Add(4*time.Second))
	run(`INSERT INTO topic_favorite (user_id, topic_id, created, updated) VALUES (?, ?, ?, ?)`,
		d1UserAlice, d1TopicPublic, base, base)
	run(`INSERT INTO topic_upvote (user_id, topic_id, description, created, updated) VALUES (?, ?, '', ?, ?)`,
		d1UserAlice, d1TopicPublic, base, base)
	run(`INSERT INTO topic_reply_reaction (topic_reply_id, user_id, reaction, created) VALUES
		(?, ?, 'like', ?), (?, ?, 'dislike', ?)`,
		d1ReplyPinned, d1UserAlice, tLike, d1ReplyPinned, d1UserBob, tLike.Add(time.Second))

	listReplies := []struct {
		id, floor, user, status int
		body                    string
	}{
		// Two replies shared floor 1 here until migration 100 made
		// (topic_id, floor) unique; the traversal they guard is the same.
		{920000370, 1, d1UserAlice, 0, "f1a"},
		{920000371, 2, d1UserBob, 0, "f1b"},
		{920000372, 3, d1UserAlice, 0, "f3"},
		{920000373, 4, d1UserAlice, 1, "hidden-floor"},
		{920000374, 6, d1UserBanned, 0, "banned-floor"},
		{920000375, 8, d1UserAlice, 0, "f8"},
		{920000376, 10, d1UserBob, 0, "f10"},
	}
	for i, r := range listReplies {
		run(`INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, created, updated)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			r.id, r.body, r.floor, r.user, d1TopicList, r.status, base.Add(time.Duration(i)*time.Second), base)
	}
	for i := range 6 {
		run(`INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, created, updated)
			VALUES (?, 'ban', ?, ?, ?, 0, ?, ?)`,
			920000380+i, i+1, d1UserBanned, d1TopicFill, base.Add(time.Duration(i)*time.Second), base)
	}
	run(`INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, created, updated)
		VALUES (?, 'keep', 7, ?, ?, 0, ?, ?)`, 920000386, d1UserAlice, d1TopicFill, base.Add(10*time.Second), base)

	run(`INSERT INTO topic_view_daily (entity_id, day, count) VALUES (?, CURRENT_DATE, 4)`, d1TopicPublic)
}

func (f *detailFix) cleanup(t *testing.T) {
	t.Helper()
	for _, q := range []string{
		"DELETE FROM topic_comment_like WHERE topic_comment_id BETWEEN ? AND ?",
		"DELETE FROM topic_comment WHERE id BETWEEN ? AND ?",
		"DELETE FROM topic_reply_reaction WHERE topic_reply_id BETWEEN ? AND ?",
		"DELETE FROM topic_reaction WHERE topic_id BETWEEN ? AND ?",
		"DELETE FROM topic_favorite WHERE topic_id BETWEEN ? AND ?",
		"DELETE FROM topic_upvote WHERE topic_id BETWEEN ? AND ?",
		"DELETE FROM topic_access_grant WHERE topic_id BETWEEN ? AND ?",
		"DELETE FROM topic_view_daily WHERE entity_id BETWEEN ? AND ?",
		"DELETE FROM topic_poll WHERE topic_id BETWEEN ? AND ?",
		"DELETE FROM topic_section_relation WHERE topic_id BETWEEN ? AND ?",
		"UPDATE topic SET pinned_reply_id = NULL, best_answer_id = NULL WHERE id BETWEEN ? AND ?",
		"DELETE FROM topic_reply WHERE id BETWEEN ? AND ?",
		"DELETE FROM topic WHERE id BETWEEN ? AND ?",
	} {
		a, b := d1TopicMin, d1TopicMax
		switch {
		case strings.Contains(q, "topic_comment"):
			a, b = d1CommentMin, d1CommentMax
		case strings.Contains(q, "topic_reply_reaction") || strings.Contains(q, "DELETE FROM topic_reply"):
			a, b = d1ReplyMin, d1ReplyMax
		}
		if err := f.db.Exec(q, a, b).Error; err != nil {
			t.Errorf("cleanup %q: %v", q, err)
		}
	}
	_ = f.db.Exec("DELETE FROM kungal_user_state WHERE user_id BETWEEN ? AND ?", d1UserAlice, d1UserStaff).Error
}

func (f *detailFix) putSession(t *testing.T, token string, userID int, roles ...string) {
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

func (f *detailFix) do(t *testing.T, method, rawURL, session, specPath string, hdr http.Header) (*http.Response, []byte) {
	t.Helper()
	req := httptest.NewRequest(method, rawURL, nil)
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
	f.spec.checkPath(t, method, specPath, resp, body)
	return resp, body
}

func (s *specConformance) checkPath(t *testing.T, method, specPath string, resp *http.Response, body []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	if status == "204" {
		if len(bytes.TrimSpace(body)) != 0 {
			t.Errorf("%s %s 204 body %q", method, specPath, body)
		}
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

func objectKeys(t *testing.T, raw json.RawMessage) []string {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return slices.Sorted(maps.Keys(m))
}

func problemCore(t *testing.T, body []byte) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	delete(m, "request_id")
	delete(m, "instance")
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}
