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

	activityapiv1 "kun-galgame-api/internal/activity/apiv1"
	activityRepo "kun-galgame-api/internal/activity/repository"
	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/testdb"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/userclient"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	acUserAlice  = 950000001
	acUserBob    = 950000002
	acUserBanned = 950000003

	acSecChat = 950000011
	acSecSeek = 950000012

	acTopicNormal = 950000101
	acTopicHelp   = 950000102
	acTopicNSFW   = 950000103
	acTopicBanned = 950000104
	acTopicMin    = 950000101
	acTopicMax    = 950000199

	acReplyLiked  = 950000201
	acReplyQuote  = 950000202
	acReplyHidden = 950000203
	acReplyBanned = 950000204

	acComment       = 950000301
	acUpvote        = 950000401
	acMsgUpvote     = 950000501
	acMsgSolved     = 950000502
	acWorkShown     = 950000601
	acWorkGone      = 950000602
	acWorkAdult     = 950000603
	acResShown      = 950000701
	acEditEngine    = 950000801
	acEditWiki      = 950000802
	acEditBare      = 950000803
	acRatingHold    = 950000901
	acRatingSpoiler = 950000902
	acActivities    = "/activities"
)

var acTie = time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)

type fakeActivityCatalog struct {
	items map[int]client.CatalogWorkListItem
	adult map[int]bool
	fail  atomic.Bool
}

func (c *fakeActivityCatalog) CatalogRowsByWorkIDs(_ context.Context, ids []int, _, contentLimit string) (map[int]client.CatalogWorkListItem, *legacyErrors.AppError) {
	if c.fail.Load() {
		return nil, legacyErrors.ErrInternal("catalog down")
	}
	out := map[int]client.CatalogWorkListItem{}
	for _, id := range ids {
		it, ok := c.items[id]
		if !ok || (contentLimit == "sfw" && c.adult[id]) {
			continue
		}
		out[id] = it
	}
	return out, nil
}

func catalogItem(t *testing.T, id int, limit string) client.CatalogWorkListItem {
	t.Helper()
	raw := fmt.Sprintf(`{"id":%d,"display_name":"Work %d","latin":"Waaku","localized":{"zh-Hans":{"value":"作品%d","machine":false}},
		"release_date":"2024-05","claim":{"site":"kungal","site_work_id":%d,"state":"live","content_limit":%q},
		"cover_slots":{"portrait":{"url":"https://image.test.example/ab/cd/%s.webp","width":600,"height":800,"thumbhash":"AbC+"}},
		"labels":[{"id":7,"display_name":"Brand","label_kind":"brand","kind":"brand","role":"developer"}],
		"intros":[{"lang":"zh-Hans","intro":"简介","source":"vndb","machine":false}]}`,
		id, id, id, id, limit, strings.Repeat("ab", 32))
	var it client.CatalogWorkListItem
	if err := json.Unmarshal([]byte(raw), &it); err != nil {
		t.Fatal(err)
	}
	return it
}

type activityFix struct {
	app     *App
	db      *gorm.DB
	spec    *specConformance
	catalog *fakeActivityCatalog
	failOA  atomic.Bool
}

func newActivityFix(t *testing.T) *activityFix {
	t.Helper()
	db := testdb.Open(t)
	f := &activityFix{db: db, catalog: &fakeActivityCatalog{
		items: map[int]client.CatalogWorkListItem{}, adult: map[int]bool{acWorkAdult: true},
	}}
	f.catalog.items[acWorkShown] = catalogItem(t, acWorkShown, "sfw")
	f.catalog.items[acWorkAdult] = catalogItem(t, acWorkAdult, "nsfw")

	oauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f.failOA.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		known := map[int]map[string]any{
			acUserAlice:  {"name": "alice", "status": 0},
			acUserBob:    {"name": "bob", "status": 0},
			acUserBanned: {"name": "banned", "status": 1},
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
	t.Cleanup(oauth.Close)
	uc := userclient.New(userclient.Config{
		BaseURL: oauth.URL, ClientID: "c", ClientSecret: "s",
		ImageCDNBase: "https://image.test.example", HTTPTimeout: 2 * time.Second,
	})
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: 0})
	t.Cleanup(func() { _ = rdb.Close() })

	images := func([]string) map[string]imageclient.ImageMeta { return map[string]imageclient.ImageMeta{} }
	convert := &content.Converter{CDNBase: "https://image.test.example", SiteBase: apiv1.SiteOrigin, Images: images, Users: uc.Users}
	f.app = &App{
		Fiber:      newFiber(),
		Config:     testConfig(),
		DB:         db,
		Redis:      rdb,
		UserClient: uc,
		Authn:      middleware.NewAuthenticator(rdb, nil, nil),
		ActivityV1: activityapiv1.New(activityRepo.NewActivityRepository(db), f.catalog, uc, convert, "https://image.test.example"),
	}
	f.app.setupRoutes()
	f.spec = newSpecConformance(t)
	f.seed(t)
	return f
}

func (f *activityFix) run(t *testing.T, q string, args ...any) {
	t.Helper()
	if err := f.db.Exec(q, args...).Error; err != nil {
		t.Fatalf("seed: %v\n%s", err, q)
	}
}

func (f *activityFix) cleanup() {
	lo, hi := 950000001, 950000999
	_ = f.db.Exec(`DELETE FROM topic_reaction WHERE topic_id BETWEEN ? AND ?`, acTopicMin, acTopicMax).Error
	_ = f.db.Exec(`DELETE FROM topic_upvote WHERE topic_id BETWEEN ? AND ?`, acTopicMin, acTopicMax).Error
	_ = f.db.Exec(`DELETE FROM topic_comment WHERE topic_id BETWEEN ? AND ?`, acTopicMin, acTopicMax).Error
	_ = f.db.Exec(`UPDATE topic SET best_answer_id = NULL WHERE id BETWEEN ? AND ?`, acTopicMin, acTopicMax).Error
	_ = f.db.Exec(`DELETE FROM topic_reply WHERE topic_id BETWEEN ? AND ?`, acTopicMin, acTopicMax).Error
	_ = f.db.Exec(`DELETE FROM topic_section_relation WHERE topic_id BETWEEN ? AND ?`, acTopicMin, acTopicMax).Error
	_ = f.db.Exec(`DELETE FROM topic WHERE id BETWEEN ? AND ?`, acTopicMin, acTopicMax).Error
	_ = f.db.Exec(`DELETE FROM topic_section WHERE id BETWEEN ? AND ?`, acSecChat, acSecSeek).Error
	_ = f.db.Exec(`DELETE FROM galgame_resource WHERE work_id BETWEEN ? AND ?`, lo, hi).Error
	_ = f.db.Exec(`DELETE FROM galgame_activity WHERE work_id BETWEEN ? AND ?`, lo, hi).Error
	_ = f.db.Exec(`DELETE FROM galgame_rating WHERE work_id BETWEEN ? AND ?`, lo, hi).Error
	_ = f.db.Exec(`DELETE FROM galgame WHERE id BETWEEN ? AND ?`, lo, hi).Error
	_ = f.db.Exec(`DELETE FROM feed_activity WHERE user_id BETWEEN ? AND ? OR source_id BETWEEN ? AND ? OR work_id BETWEEN ? AND ?`,
		lo, hi, lo, hi, lo, hi).Error
}

func (f *activityFix) seed(t *testing.T) {
	t.Helper()
	f.cleanup()
	t.Cleanup(f.cleanup)
	f.run(t, `INSERT INTO topic_section (id, name, created, updated) VALUES (?, 'g-chatting', ?, ?) ON CONFLICT DO NOTHING`, acSecChat, acTie, acTie)
	f.run(t, `INSERT INTO topic_section (id, name, created, updated) VALUES (?, 'g-seeking', ?, ?) ON CONFLICT DO NOTHING`, acSecSeek, acTie, acTie)
	var chatID, seekID int
	if err := f.db.Raw(`SELECT id FROM topic_section WHERE name = 'g-chatting'`).Scan(&chatID).Error; err != nil || chatID == 0 {
		t.Fatalf("g-chatting section: %v", err)
	}
	if err := f.db.Raw(`SELECT id FROM topic_section WHERE name = 'g-seeking'`).Scan(&seekID).Error; err != nil || seekID == 0 {
		t.Fatalf("g-seeking section: %v", err)
	}
	topic := func(id, user int, title string, nsfw bool, created time.Time) {
		f.run(t, `INSERT INTO topic (
			id, title, content, view, status, category, status_update_time, created, updated,
			user_id, is_nsfw, access_scope, cover_images, like_count, dislike_count, reply_count, comment_count,
			favorite_count, upvote_count, view_7d, view_30d, hidden_by, last_reply_floor
		) VALUES (?, ?, ?, 0, 0, 'galgame', ?, ?, ?, ?, ?, 'public', '', 0, 0, 3, 1, 2, 1, 0, 0, '', 0)`,
			id, title, title+" body", acTie, created, created, user, nsfw)
	}
	topic(acTopicNormal, acUserAlice, "normal", false, acTie)
	topic(acTopicHelp, acUserBob, "help", false, acTie)
	topic(acTopicNSFW, acUserAlice, "adult", true, acTie.Add(time.Hour))
	topic(acTopicBanned, acUserBanned, "banned", false, acTie)
	f.run(t, `INSERT INTO topic_section_relation (topic_id, topic_section_id, created, updated) VALUES (?, ?, ?, ?), (?, ?, ?, ?)`,
		acTopicNormal, chatID, acTie, acTie, acTopicHelp, seekID, acTie, acTie)

	reply := func(id, user, floor, status, likes int, body string) {
		f.run(t, `INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, like_count, created, updated)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, body, floor, user, acTopicNormal, status, likes, acTie, acTie)
	}
	reply(acReplyLiked, acUserBob, 1, 0, 5, "the liked reply")
	reply(acReplyQuote, acUserAlice, 2, 0, 0, fmt.Sprintf("[@bob](kungal-user:%d) [#1](kungal-reply:%d) agreed", acUserBob, acReplyLiked))
	reply(acReplyHidden, acUserBob, 3, 1, 99, "hidden but loved")
	reply(acReplyBanned, acUserBanned, 4, 0, 0, "from a banned account")
	f.run(t, `INSERT INTO topic_comment (id, topic_id, topic_reply_id, user_id, target_user_id, content, status, created, updated)
		VALUES (?, ?, ?, ?, ?, 'nice', 0, ?, ?)`, acComment, acTopicNormal, acReplyLiked, acUserBob, acUserBob, acTie, acTie)
	f.run(t, `INSERT INTO topic_upvote (id, topic_id, user_id, description, created, updated) VALUES (?, ?, ?, 'push', ?, ?)`,
		acUpvote, acTopicNormal, acUserBob, acTie.Add(2*time.Hour), acTie.Add(2*time.Hour))
	f.run(t, `INSERT INTO topic_reaction (topic_id, user_id, reaction) VALUES (?, ?, 'heart'), (?, ?, 'heart')`,
		acTopicNormal, acUserBanned, acTopicNormal, acUserBob)
	f.run(t, `UPDATE topic SET best_answer_id = ? WHERE id = ?`, acReplyLiked, acTopicNormal)
	feed := func(typ string, source, user int, body, link string, created time.Time) {
		f.run(t, `INSERT INTO feed_activity (type, source_id, user_id, work_id, content, link, is_nsfw, created)
			VALUES (?, ?, ?, 0, ?, ?, false, ?) ON CONFLICT (type, source_id) DO NOTHING`, typ, source, user, body, link, created)
	}
	feed("MESSAGE_UPVOTE", acMsgUpvote, acUserBob, "normal", fmt.Sprintf("/topic/%d", acTopicNormal), acTie.Add(2*time.Hour))
	feed("MESSAGE_SOLUTION", acMsgSolved, acUserAlice, "the liked reply", fmt.Sprintf("/topic/%d?reply=1", acTopicNormal), acTie.Add(3*time.Hour))

	for _, w := range []int{acWorkShown, acWorkGone, acWorkAdult} {
		f.run(t, `INSERT INTO galgame (id, published, creator_user_id, resource_count, like_count, favorite_count, created, updated)
			VALUES (?, true, ?, 1, 4, 5, ?, ?)`, w, acUserAlice, acTie.Add(4*time.Hour), acTie.Add(4*time.Hour))
	}
	for i, w := range []int{acWorkShown, acWorkGone, acWorkAdult} {
		f.run(t, `INSERT INTO galgame_resource (id, work_id, user_id, type, language, platform, size, note, created, updated)
			VALUES (?, ?, ?, 'game', 'zh-cn', 'windows', '1.7GB', 'note', ?, ?)`,
			acResShown+i, w, acUserBob, acTie.Add(5*time.Hour), acTie.Add(5*time.Hour))
	}
	f.run(t, `INSERT INTO galgame_rating (id, recommend, overall, play_status, short_summary, spoiler_level, user_id, work_id, created, updated)
		VALUES (?, 'yes', 7, 'on_hold', 'worth a second try', 'none', ?, ?, ?, ?), (?, 'no', 4, 'done_all', 'the twist is', 'serious', ?, ?, ?, ?)`,
		acRatingHold, acUserAlice, acWorkShown, acTie.Add(7*time.Hour), acTie.Add(7*time.Hour),
		acRatingSpoiler, acUserBob, acWorkShown, acTie.Add(7*time.Hour), acTie.Add(7*time.Hour))
	f.run(t, `INSERT INTO galgame_activity (id, wiki_revision_id, work_id, user_id, type, created, wiki_revision_number, edit_revision_id)
		VALUES (?, NULL, ?, ?, 'GALGAME_EDIT', ?, 3, ?), (?, 950000877, ?, ?, 'GALGAME_EDIT', ?, NULL, NULL), (?, NULL, ?, ?, 'GALGAME_EDIT', ?, NULL, NULL)`,
		acEditEngine, acWorkShown, acUserBob, acTie.Add(6*time.Hour), acEditEngine,
		acEditWiki, acWorkShown, acUserBob, acTie.Add(6*time.Hour),
		acEditBare, acWorkShown, acUserBob, acTie.Add(6*time.Hour))
}

func (f *activityFix) call(t *testing.T, q url.Values) (*http.Response, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/activities?"+q.Encode(), nil)
	resp, err := f.app.Fiber.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	f.spec.checkPath(t, http.MethodGet, acActivities, resp, body)
	out := map[string]any{}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("body: %v\n%s", err, body)
	}
	return resp, out
}

func (f *activityFix) walk(t *testing.T, q url.Values, limit int) []map[string]any {
	t.Helper()
	var all []map[string]any
	q.Set("limit", strconv.Itoa(limit))
	for page := 0; page < 2000; page++ {
		resp, body := f.call(t, q)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("page %d: %d %+v", page, resp.StatusCode, body)
		}
		items, _ := body["items"].([]any)
		for _, it := range items {
			m, _ := it.(map[string]any)
			all = append(all, m)
		}
		next, _ := body["next_cursor"].(string)
		if next == "" {
			return all
		}
		q.Set("cursor", next)
	}
	t.Fatal("walk did not end")
	return nil
}

func (f *activityFix) rowID(t *testing.T, typ string, source int) string {
	t.Helper()
	var id int64
	if err := f.db.Raw(`SELECT id FROM feed_activity WHERE type = ? AND source_id = ?`, typ, source).Scan(&id).Error; err != nil || id == 0 {
		t.Fatalf("no feed row %s/%d: %v", typ, source, err)
	}
	return strconv.FormatInt(id, 10)
}

func (f *activityFix) ourRows(t *testing.T) map[string]bool {
	t.Helper()
	var ids []int64
	if err := f.db.Raw(`SELECT id FROM feed_activity WHERE user_id BETWEEN 950000001 AND 950000999
		OR source_id BETWEEN 950000001 AND 950000999 OR work_id BETWEEN 950000001 AND 950000999`).Scan(&ids).Error; err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, id := range ids {
		out[strconv.FormatInt(id, 10)] = true
	}
	return out
}

func activityIDs(items []map[string]any, keep map[string]bool) []string {
	var out []string
	for _, it := range items {
		if id := fmt.Sprint(it["id"]); keep == nil || keep[id] {
			out = append(out, id)
		}
	}
	return out
}

func activityByID(items []map[string]any, id string) map[string]any {
	for _, it := range items {
		if fmt.Sprint(it["id"]) == id {
			return it
		}
	}
	return nil
}
