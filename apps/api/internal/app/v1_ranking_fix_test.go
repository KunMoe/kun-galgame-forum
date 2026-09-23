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
	"sync/atomic"
	"testing"
	"time"

	"kun-galgame-api/internal/galgame/client"
	rankingapiv1 "kun-galgame-api/internal/ranking/apiv1"
	rankingRepo "kun-galgame-api/internal/ranking/repository"
	"kun-galgame-api/internal/testdb"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	rkUserA       = 960000001
	rkUserB       = 960000002
	rkUserC       = 960000003
	rkUserBanned  = 960000004
	rkUserMissing = 960000005

	rkTopicMin = 960000101
	rkTopicMax = 960000199
	rkWorkMin  = 960000201
	rkWorkMax  = 960000299
	rkReplyMin = 960000301

	rkTopicTieLow  = 960000101
	rkTopicTieMid  = 960000102
	rkTopicTieHigh = 960000103
	rkTopicHidden  = 960000104
	rkTopicLogin   = 960000105
	rkTopicNSFW    = 960000106
	rkTopicBanned  = 960000107
	rkTopicMissing = 960000108

	rkWorkTop         = 960000201
	rkWorkUnpublished = 960000202
	rkWorkBare        = 960000203
	rkWorkNSFW        = 960000204
	rkWorkNoCreator   = 960000205
	rkWorkBannedMaker = 960000206
)

type rkWorks struct {
	nsfw map[int]bool
	fail atomic.Bool
}

func (w *rkWorks) CatalogRowsByWorkIDs(_ context.Context, ids []int, _, contentLimit string) (map[int]client.CatalogWorkListItem, *legacyErrors.AppError) {
	if w.fail.Load() {
		return nil, legacyErrors.ErrInternal("catalog down")
	}
	out := map[int]client.CatalogWorkListItem{}
	for _, id := range ids {
		if contentLimit == "sfw" && w.nsfw[id] {
			continue
		}
		limit := "sfw"
		if w.nsfw[id] {
			limit = "nsfw"
		}
		raw := fmt.Sprintf(`{"id": %d, "display_name": "作品%d", "latin": "Work %d",
			"localized": {"zh-Hans": {"value": "作品 %d", "machine": false}},
			"claim": {"site": "kungal", "state": "live", "content_limit": %q}}`, id, id, id, id, limit)
		var it client.CatalogWorkListItem
		if err := json.Unmarshal([]byte(raw), &it); err != nil {
			panic(err)
		}
		out[id] = it
	}
	return out, nil
}

type rkFix struct {
	app    *App
	db     *gorm.DB
	spec   *specConformance
	works  *rkWorks
	failOA atomic.Bool
}

func newRankingFix(t *testing.T) *rkFix {
	t.Helper()
	db := testdb.Open(t)
	f := &rkFix{db: db, works: &rkWorks{nsfw: map[int]bool{rkWorkNSFW: true}}}

	oauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f.failOA.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		status := map[int]int{rkUserA: 0, rkUserB: 0, rkUserC: 0, rkUserBanned: 1}
		var users []map[string]any
		for _, raw := range strings.Split(r.URL.Query().Get("ids"), ",") {
			id, _ := strconv.Atoi(raw)
			if st, ok := status[id]; ok {
				users = append(users, map[string]any{
					"id": id, "name": fmt.Sprintf("u%d", id), "status": st, "bio": fmt.Sprintf("bio of %d", id), "roles": []string{"user"},
				})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"users": users, "not_found": []int{}}})
	}))
	t.Cleanup(oauth.Close)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: 0})
	t.Cleanup(func() { _ = rdb.Close() })

	uc := userclient.New(userclient.Config{
		BaseURL: oauth.URL, ClientID: "c", ClientSecret: "s",
		ImageCDNBase: "https://image.test.example", HTTPTimeout: 2 * time.Second,
	})
	cfg := testConfig()
	cfg.NextMoeAPI.ImageCDNBase = "https://image.test.example"
	f.app = &App{
		Fiber:      newFiber(),
		Config:     cfg,
		DB:         db,
		Redis:      rdb,
		UserClient: uc,
		RankingV1:  rankingapiv1.New(rankingRepo.NewRankingRepository(db), uc, f.works, "https://image.test.example"),
	}
	f.app.setupRoutes()
	f.spec = newSpecConformance(t)
	f.seed(t)
	return f
}

func (f *rkFix) sql(t *testing.T, q string, args ...any) {
	t.Helper()
	if err := f.db.Exec(q, args...).Error; err != nil {
		t.Fatalf("seed: %v\n%s", err, q)
	}
}

func (f *rkFix) cleanup() {
	for _, q := range []string{
		`DELETE FROM galgame_rating WHERE work_id BETWEEN ? AND ?`,
		`DELETE FROM galgame_resource WHERE work_id BETWEEN ? AND ?`,
		`DELETE FROM galgame WHERE id BETWEEN ? AND ?`,
	} {
		_ = f.db.Exec(q, rkWorkMin, rkWorkMax).Error
	}
	for _, q := range []string{
		`DELETE FROM topic_comment WHERE topic_id BETWEEN ? AND ?`,
		`DELETE FROM topic_reply WHERE topic_id BETWEEN ? AND ?`,
		`DELETE FROM topic WHERE id BETWEEN ? AND ?`,
	} {
		_ = f.db.Exec(q, rkTopicMin, rkTopicMax).Error
	}
	_ = f.db.Exec(`DELETE FROM kungal_user_state WHERE user_id BETWEEN ? AND ?`, rkUserA, rkUserMissing).Error
}

func (f *rkFix) seed(t *testing.T) {
	t.Helper()
	f.cleanup()
	t.Cleanup(f.cleanup)
	now := time.Now().Add(-48 * time.Hour)

	topic := func(id, author, status int, scope string, nsfw bool, view int) {
		f.sql(t, `INSERT INTO topic (
			id, title, content, view, status, category, status_update_time, created, updated,
			user_id, is_nsfw, access_scope, cover_images, like_count, dislike_count, reply_count, comment_count,
			favorite_count, upvote_count, view_7d, view_30d, hidden_by, last_reply_floor
		) VALUES (?, ?, 'body', ?, ?, 'galgame', ?, ?, ?, ?, ?, ?, '', 0, 0, 0, 0, 0, 0, 0, 0, '', 0)`,
			id, fmt.Sprintf("topic %d", id), view, status, now, now, now, author, nsfw, scope)
	}
	// Three topics share one view count so the id tie-break decides their order.
	topic(rkTopicTieLow, rkUserA, 0, "public", false, 1900000100)
	topic(rkTopicTieMid, rkUserB, 0, "public", false, 1900000100)
	topic(rkTopicTieHigh, rkUserC, 0, "public", false, 1900000100)
	topic(rkTopicHidden, rkUserA, 1, "public", false, 1900000900)
	topic(rkTopicLogin, rkUserA, 0, "login", false, 1900000800)
	topic(rkTopicNSFW, rkUserA, 0, "public", true, 1900000700)
	topic(rkTopicBanned, rkUserBanned, 0, "public", false, 1900000600)
	topic(rkTopicMissing, rkUserMissing, 0, "public", false, 1900000500)

	replyID, floor := rkReplyMin, 0
	reply := func(topicID, author, status int) {
		floor++
		f.sql(t, `INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, like_count, created, updated)
			VALUES (?, 'r', ?, ?, ?, ?, 0, ?, ?)`, replyID, floor, author, topicID, status, now, now)
		replyID++
	}
	for range 3 {
		reply(rkTopicTieLow, rkUserA, 0)
	}
	for range 2 {
		reply(rkTopicTieLow, rkUserB, 0)
		reply(rkTopicTieLow, rkUserB, 1)
	}
	reply(rkTopicTieLow, rkUserC, 0)
	for range 3 {
		reply(rkTopicHidden, rkUserC, 0)
	}

	for _, s := range []struct{ uid, moe int }{
		{rkUserA, 2000000003}, {rkUserB, 2000000002}, {rkUserC, 2000000001},
		{rkUserBanned, 2000000010}, {rkUserMissing, 2000000009},
	} {
		f.sql(t, `INSERT INTO kungal_user_state (user_id, moemoepoint, created, updated) VALUES (?, ?, ?, ?)`, s.uid, s.moe, now, now)
	}

	work := func(id int, published bool, limit string, creator *int, view int, resources int) {
		f.sql(t, `INSERT INTO galgame (id, view, updated, published, content_limit, creator_user_id, resource_count)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, id, view, now, published, limit, creator, resources)
		for range resources {
			f.sql(t, `INSERT INTO galgame_resource (work_id, user_id, updated) VALUES (?, ?, ?)`, id, rkUserA, now)
		}
	}
	a, banned := rkUserA, rkUserBanned
	work(rkWorkTop, true, "sfw", &a, 1900000050, 1)
	work(rkWorkUnpublished, false, "sfw", &a, 1900000900, 1)
	work(rkWorkBare, true, "sfw", &a, 1900000800, 0)
	work(rkWorkNSFW, true, "nsfw", &a, 1900000700, 1)
	work(rkWorkNoCreator, true, "sfw", nil, 1900000040, 1)
	work(rkWorkBannedMaker, true, "sfw", &banned, 1900000030, 1)

	// Identical ratings on two works tie on the weighted score.
	for _, w := range []int{rkWorkTop, rkWorkNoCreator} {
		f.sql(t, `INSERT INTO galgame_rating (recommend, overall, user_id, work_id, updated) VALUES ('yes', 10, ?, ?, ?)`, rkUserA, w, now)
	}
}

func (f *rkFix) get(t *testing.T, path string, q url.Values) (*http.Response, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1"+path+"?"+q.Encode(), nil)
	resp, err := f.app.Fiber.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	f.spec.checkPath(t, http.MethodGet, path, resp, body)
	out := map[string]any{}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("body: %v\n%s", err, body)
	}
	return resp, out
}

type rkEntry struct {
	rank  int
	id    string
	value float64
	raw   map[string]any
}

func rkEntries(body map[string]any, ref string) []rkEntry {
	items, _ := body["items"].([]any)
	out := make([]rkEntry, 0, len(items))
	for _, it := range items {
		m, _ := it.(map[string]any)
		inner, _ := m[ref].(map[string]any)
		v, _ := m["metric_value"].(float64)
		out = append(out, rkEntry{rank: asInt(m["rank"]), id: strID(inner["id"]), value: v, raw: m})
	}
	return out
}

func rkIDs(es []rkEntry) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.id
	}
	return out
}

func rkWant(ids ...int) string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = strconv.Itoa(id)
	}
	return fmt.Sprint(out)
}
