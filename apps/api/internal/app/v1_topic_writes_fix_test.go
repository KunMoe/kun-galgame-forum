package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	msgRepo "kun-galgame-api/internal/message/repository"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/testdb"
	topicRepo "kun-galgame-api/internal/topic/repository"
	topicService "kun-galgame-api/internal/topic/service"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/internal/user/oauth"
	userRepo "kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/secretbox"
	"kun-galgame-api/pkg/trustclient"
	"kun-galgame-api/pkg/userclient"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	w3UserAlice  = 930000001
	w3UserBob    = 930000002
	w3UserStaff  = 930000003
	w3UserBanned = 930000004
	w3UserOther  = 930000005
	w3UserGrant  = 930000006
	w3MentionMin = 930000021
	w3MentionMax = 930000032

	w3SecNews  = 930000010
	w3SecSeek  = 930000011
	w3SecOther = 930000012
	w3SecHelp  = 930000013
	w3SecWeb   = 930000014
	w3SecDaily = 930000015
	w3SecWalk  = 930000016

	w3TopicMin = 930000201
	w3TopicMax = 930000299
	w3ReplyMin = 930000301
	w3ReplyMax = 930000399
	w3CommMin  = 930000401
	w3CommMax  = 930000499

	w3TopicPub     = 930000201
	w3TopicHidden  = 930000202
	w3TopicModHide = 930000203
	w3TopicOld     = 930000204
	w3TopicFloors  = 930000205
	w3TopicNSFW    = 930000206
	w3TopicRole    = 930000207
	w3TopicPaid    = 930000208

	w3CoverHash        = "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	gradedExplicitHash = "c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3"
	w3CoverAlt         = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
)

type w3Verifier struct{}

func (w3Verifier) Verify(_ context.Context, raw string) (*oauth.AccessClaims, error) {
	switch raw {
	case "staff-token":
		return &oauth.AccessClaims{
			ID: w3UserStaff, Name: "staff", Roles: []string{"user"},
			SiteRoles: []string{"moderator"}, ClientID: "kungal-app",
		}, nil
	case "bob-token":
		return &oauth.AccessClaims{
			ID: w3UserBob, Name: "bob", Roles: []string{"user"}, ClientID: "kungal-app",
		}, nil
	default:
		return nil, fmt.Errorf("bad token")
	}
}

type awardCall struct {
	userID int
	delta  int
	reason string
	ref    string
	key    string
}

type denyChecker struct{}

func (denyChecker) Check(_ context.Context, _ trustclient.CheckRequest) (*trustclient.CheckResult, error) {
	return &trustclient.CheckResult{Decision: gate.DecisionDeny, Matched: []string{"x"}}, nil
}

type writeFix struct {
	*App
	db         *gorm.DB
	rdb        *redis.Client
	spec       *specConformance
	oauth      *httptest.Server
	mux        *http.ServeMux
	awards     []awardCall
	mu         sync.Mutex
	nBatch     atomic.Int32
	failOA     atomic.Bool
	oauthExtra []map[string]any
}

func newWriteFix(t *testing.T, checker gate.Checker) *writeFix {
	t.Helper()
	db := testdb.Open(t)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: 0})
	t.Cleanup(func() { _ = rdb.Close() })

	f := &writeFix{db: db, rdb: rdb}
	mux := http.NewServeMux()
	f.mux = mux
	mux.HandleFunc("/users/batch", func(w http.ResponseWriter, _ *http.Request) {
		f.nBatch.Add(1)
		if f.failOA.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		users := []map[string]any{
			{"id": w3UserAlice, "name": "alice", "status": 0, "roles": []string{"user"}},
			{"id": w3UserBob, "name": "bob", "status": 0, "roles": []string{"user"}},
			{"id": w3UserStaff, "name": "staff", "status": 0, "roles": []string{"moderator"}},
			{"id": w3UserBanned, "name": "banned", "status": 1, "roles": []string{"user"}},
			{"id": w3UserOther, "name": "other", "status": 0, "roles": []string{"user"}},
			{"id": w3UserGrant, "name": "grantee", "status": 0, "roles": []string{"user"}},
		}
		for id := w3MentionMin; id <= w3MentionMax; id++ {
			users = append(users, map[string]any{
				"id": id, "name": fmt.Sprintf("m%d", id), "status": 0, "roles": []string{"user"},
			})
		}
		f.mu.Lock()
		users = append(users, f.oauthExtra...)
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{"users": users, "not_found": []int{}},
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
	cfg.OAuth.ServerURL = f.oauth.URL
	cfg.OAuth.ClientID = "test-client"
	sexual := int16(0)
	var trust *gate.CheckService
	if checker != nil {
		trust = gate.NewCheckService(checker)
	}
	state := userRepo.NewStateRepository(db)
	f.App = &App{
		Fiber:      newFiber(),
		Config:     cfg,
		DB:         db,
		Redis:      rdb,
		UserClient: uc,
		UserState:  state,
		TopicAward: f.recordAward,
		TrustCheck: trust,
		Authn:      middleware.NewAuthenticator(rdb, nil, middleware.NewBearer(w3Verifier{}, rdb, nil)),
		ImageMeta: func(hashes []string) map[string]imageclient.ImageMeta {
			out := map[string]imageclient.ImageMeta{}
			explicit := imageclient.SexualExplicit
			for _, h := range hashes {
				out[h] = imageclient.ImageMeta{Width: 64, Height: 36, Thumbhash: "AbC+", Sexual: &sexual}
				if h == gradedExplicitHash {
					out[h] = imageclient.ImageMeta{Width: 64, Height: 36, Thumbhash: "AbC+", Sexual: &explicit}
				}
			}
			return out
		},
		LotteryService: topicService.NewLotteryService(
			topicRepo.NewLotteryRepository(db), state, uc,
			msgService.NewNotifier(msgRepo.NewMessageRepository(db)), testLotteryBox(t),
		),
	}
	f.setupRoutes()
	f.spec = newSpecConformance(t)
	f.seed(t)
	return f
}

func testLotteryBox(t *testing.T) *secretbox.Box {
	t.Helper()
	box, err := secretbox.New(strings.Repeat("ab", 32))
	if err != nil {
		t.Fatal(err)
	}
	return box
}

func (f *writeFix) recordAward(userID, delta int, reason, ref, key string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.awards = append(f.awards, awardCall{userID, delta, reason, ref, key})
}

func (f *writeFix) snapshotAwards() []awardCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]awardCall, len(f.awards))
	copy(out, f.awards)
	return out
}

func (f *writeFix) seed(t *testing.T) {
	t.Helper()
	f.cleanup(t)
	t.Cleanup(func() { f.cleanup(t) })
	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, q)
		}
	}
	now := time.Now()
	// The daily limit counts a user's topics created in the last 24 hours, so
	// seeded topics sit two days back: inside the 3-month bump cutoff, outside
	// the limit's window.
	seeded := now.AddDate(0, 0, -2)
	old := now.AddDate(0, -4, 0)
	for _, row := range []struct {
		id   int
		name string
	}{
		{w3SecNews, "g-news"}, {w3SecSeek, "g-seeking"}, {w3SecOther, "g-other"},
		{w3SecHelp, "t-help"}, {w3SecWeb, "t-web"}, {w3SecDaily, "o-daily"},
		{w3SecWalk, "g-walkthrough"},
	} {
		run(`INSERT INTO topic_section (id, name, created, updated) VALUES (?, ?, ?, ?) ON CONFLICT (id) DO NOTHING`,
			row.id, row.name, base, base)
	}
	for _, uid := range []int{w3UserAlice, w3UserBob, w3UserStaff, w3UserBanned, w3UserOther, w3UserGrant} {
		moe := 30
		if uid == w3UserBob {
			moe = 5
		}
		run(`INSERT INTO kungal_user_state (user_id, moemoepoint, created, updated)
			VALUES (?, ?, ?, ?) ON CONFLICT (user_id) DO UPDATE SET moemoepoint = EXCLUDED.moemoepoint`,
			uid, moe, base, base)
	}
	for id := w3MentionMin; id <= w3MentionMax; id++ {
		run(`INSERT INTO kungal_user_state (user_id, moemoepoint, created, updated)
			VALUES (?, 7, ?, ?) ON CONFLICT (user_id) DO NOTHING`, id, base, base)
	}

	ins := func(id, user, status int, cat, scope, hidden, title, body string, nsfw bool, created, bumped time.Time) {
		t.Helper()
		run(`INSERT INTO topic (
			id, title, content, view, status, category, status_update_time, created, updated,
			user_id, is_nsfw, access_scope, cover_images, like_count, dislike_count, reply_count, comment_count,
			favorite_count, upvote_count, view_7d, view_30d, hidden_by, last_reply_floor
		) VALUES (?, ?, ?, 0, ?, ?, ?, ?, ?, ?, ?, ?, '', 0, 0, 0, 0, 0, 0, 0, 0, ?, 0)`,
			id, title, body, status, cat, bumped, created, created, user, nsfw, scope, hidden)
	}
	ins(w3TopicPub, w3UserAlice, 0, "galgame", "public", "", "pub", "pub-body", false, seeded, seeded)
	ins(w3TopicHidden, w3UserAlice, 1, "galgame", "public", "author", "hid", "hid-body", false, seeded, seeded)
	ins(w3TopicModHide, w3UserAlice, 1, "galgame", "public", "moderator", "modhid", "modhid-body", false, seeded, seeded)
	ins(w3TopicOld, w3UserAlice, 0, "galgame", "public", "", "old", "old-body", false, old, old)
	ins(w3TopicFloors, w3UserAlice, 0, "galgame", "public", "", "floors", "floors-body", false, seeded, seeded)
	ins(w3TopicNSFW, w3UserAlice, 0, "galgame", "public", "", "nsfw", "nsfw-body", true, seeded, seeded)
	ins(w3TopicRole, w3UserAlice, 0, "galgame", "role", "", "role", "role-body", false, seeded, seeded)
	ins(w3TopicPaid, w3UserAlice, 0, "galgame", "public", "", "paid", "paid-body", false, seeded, seeded)

	run(`INSERT INTO topic_section_relation (topic_id, topic_section_id, created, updated) VALUES (?, ?, ?, ?)`,
		w3TopicPub, w3SecNews, base, base)
	run(`INSERT INTO topic_section_relation (topic_id, topic_section_id, created, updated) VALUES (?, ?, ?, ?)`,
		w3TopicNSFW, w3SecNews, base, base)
	run(`INSERT INTO topic_section_relation (topic_id, topic_section_id, created, updated) VALUES (?, ?, ?, ?)`,
		w3TopicPaid, w3SecSeek, base, base)
	run(`INSERT INTO topic_section_relation (topic_id, topic_section_id, created, updated) VALUES (?, ?, ?, ?)`,
		w3TopicOld, w3SecNews, base, base)
	run(`INSERT INTO topic_access_grant (topic_id, subject_type, subject_value) VALUES (?, 'role', 'creator')`, w3TopicRole)
	run(`UPDATE topic SET cover_images = ? WHERE id = ?`, `["/image/`+w3CoverHash+`"]`, w3TopicNSFW)

	run(`INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, like_count, created, updated)
		VALUES (?, 'r1', 1, ?, ?, 0, 2, ?, ?)`, w3ReplyMin, w3UserBob, w3TopicFloors, seeded, seeded)
	run(`INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, like_count, created, updated)
		VALUES (?, 'r2', 2, ?, ?, 0, 0, ?, ?)`, w3ReplyMin+1, w3UserAlice, w3TopicFloors, seeded, seeded)
	run(`INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, like_count, created, updated)
		VALUES (?, 'r3', 3, ?, ?, 0, 0, ?, ?)`, w3ReplyMin+2, w3UserBob, w3TopicFloors, seeded, seeded)
	run(`UPDATE topic SET last_reply_floor = 3, reply_count = 3 WHERE id = ?`, w3TopicFloors)
}

func (f *writeFix) cleanup(t *testing.T) {
	t.Helper()
	ours := []any{930000001, 930000999}
	_ = f.db.Exec(`DELETE FROM message WHERE sender_id BETWEEN ? AND ? OR receiver_id BETWEEN ? AND ?`,
		ours[0], ours[1], ours[0], ours[1]).Error
	_ = f.db.Exec(`DELETE FROM topic_comment_like WHERE topic_comment_id IN (
		SELECT id FROM topic_comment WHERE user_id BETWEEN ? AND ? OR topic_id IN (SELECT id FROM topic WHERE user_id BETWEEN ? AND ?))`,
		ours[0], ours[1], ours[0], ours[1]).Error
	_ = f.db.Exec(`DELETE FROM topic_comment WHERE user_id BETWEEN ? AND ? OR topic_id IN (SELECT id FROM topic WHERE user_id BETWEEN ? AND ?)`,
		ours[0], ours[1], ours[0], ours[1]).Error
	_ = f.db.Exec(`DELETE FROM topic_draft WHERE user_id BETWEEN ? AND ?`, ours[0], ours[1]).Error
	_ = f.db.Exec(`DELETE FROM topic_lottery WHERE user_id BETWEEN ? AND ? OR topic_id BETWEEN ? AND ?`,
		ours[0], ours[1], w3TopicMin, w3TopicMax).Error
	_ = f.db.Exec(`DELETE FROM topic_poll_vote WHERE poll_id IN (
		SELECT id FROM topic_poll WHERE topic_id IN (SELECT id FROM topic WHERE user_id BETWEEN ? AND ?))`,
		ours[0], ours[1]).Error
	_ = f.db.Exec(`DELETE FROM topic_poll_option WHERE poll_id IN (
		SELECT id FROM topic_poll WHERE topic_id IN (SELECT id FROM topic WHERE user_id BETWEEN ? AND ?))`,
		ours[0], ours[1]).Error
	_ = f.db.Exec(`DELETE FROM topic_poll WHERE topic_id IN (SELECT id FROM topic WHERE user_id BETWEEN ? AND ?)`,
		ours[0], ours[1]).Error
	_ = f.db.Exec(`DELETE FROM topic_reply_reaction WHERE topic_reply_id IN (
		SELECT id FROM topic_reply WHERE user_id BETWEEN ? AND ? OR topic_id IN (SELECT id FROM topic WHERE user_id BETWEEN ? AND ?))`,
		ours[0], ours[1], ours[0], ours[1]).Error
	_ = f.db.Exec(`DELETE FROM topic_reply_like WHERE topic_reply_id IN (SELECT id FROM topic_reply WHERE user_id BETWEEN ? AND ?)`,
		ours[0], ours[1]).Error
	_ = f.db.Exec(`DELETE FROM topic_reply_dislike WHERE topic_reply_id IN (SELECT id FROM topic_reply WHERE user_id BETWEEN ? AND ?)`,
		ours[0], ours[1]).Error
	_ = f.db.Exec(`UPDATE topic SET pinned_reply_id = NULL, best_answer_id = NULL WHERE user_id BETWEEN ? AND ? OR id BETWEEN ? AND ?`,
		ours[0], ours[1], w3TopicMin, w3TopicMax).Error
	_ = f.db.Exec(`DELETE FROM topic_reply WHERE user_id BETWEEN ? AND ? OR topic_id IN (SELECT id FROM topic WHERE user_id BETWEEN ? AND ?) OR id BETWEEN ? AND ?`,
		ours[0], ours[1], ours[0], ours[1], w3ReplyMin, w3ReplyMax).Error
	_ = f.db.Exec(`DELETE FROM topic_access_grant WHERE topic_id IN (SELECT id FROM topic WHERE user_id BETWEEN ? AND ?) OR topic_id BETWEEN ? AND ?`,
		ours[0], ours[1], w3TopicMin, w3TopicMax).Error
	_ = f.db.Exec(`DELETE FROM topic_section_relation WHERE topic_id IN (SELECT id FROM topic WHERE user_id BETWEEN ? AND ?) OR topic_id BETWEEN ? AND ?`,
		ours[0], ours[1], w3TopicMin, w3TopicMax).Error
	_ = f.db.Exec(`DELETE FROM topic WHERE user_id BETWEEN ? AND ? OR id BETWEEN ? AND ?`,
		ours[0], ours[1], w3TopicMin, w3TopicMax).Error
	_ = f.db.Exec(`DELETE FROM kungal_user_state WHERE user_id BETWEEN ? AND ?`, 930000001, 930000999).Error
	_ = f.db.Exec(`DELETE FROM topic_section WHERE id BETWEEN ? AND ?`, w3SecNews, w3SecDaily).Error
}

func (f *writeFix) putSession(t *testing.T, token string, userID int, roles ...string) {
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

func (f *writeFix) doJSON(t *testing.T, method, rawURL, session, specPath, idem string, hdr http.Header, payload any) (*http.Response, []byte) {
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
	if idem != "" {
		req.Header.Set("Idempotency-Key", idem)
	}
	for k, vs := range hdr {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	resp, err := f.Fiber.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	f.spec.checkPath(t, method, specPath, resp, body)
	return resp, body
}

func (f *writeFix) alice(t *testing.T) {
	t.Helper()
	f.putSession(t, "sess-alice", w3UserAlice)
	f.putSession(t, "sess-bob", w3UserBob)
	f.putSession(t, "sess-staff", w3UserStaff, "user", "moderator")
	f.putSession(t, "sess-other", w3UserOther)
}

func (f *writeFix) addOAuthUser(id int, name string, status int, extra map[string]any) {
	u := map[string]any{
		"id": id, "name": name, "status": status, "roles": []string{"user"},
	}
	for k, v := range extra {
		u[k] = v
	}
	f.mu.Lock()
	f.oauthExtra = append(f.oauthExtra, u)
	f.mu.Unlock()
}

func problemMap(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("problem json: %v\n%s", err, body)
	}
	return m
}

func problemErrors(t *testing.T, body []byte) []map[string]any {
	t.Helper()
	m := problemMap(t, body)
	raw, _ := m["errors"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, e := range raw {
		if em, ok := e.(map[string]any); ok {
			out = append(out, em)
		}
	}
	return out
}

func keyUUID(n int) string {
	return fmt.Sprintf("11111111-1111-4111-8111-%012d", n)
}

func asInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	case string:
		i, _ := strconv.Atoi(n)
		return i
	default:
		return 0
	}
}

func strID(v any) string {
	switch s := v.(type) {
	case string:
		return s
	default:
		return fmt.Sprint(v)
	}
}

func mentionBody(ids ...int) string {
	var b strings.Builder
	b.WriteString("hello ")
	for _, id := range ids {
		fmt.Fprintf(&b, "[@u](kungal-user:%d) ", id)
	}
	return b.String()
}

func (f *writeFix) scalar(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	if err := f.db.Raw(query, args...).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}
