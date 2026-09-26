package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	activitypush "kun-galgame-api/internal/activity/push"
	"kun-galgame-api/pkg/communityclient"
)

const (
	apTopicA = 940100101
	apTopicB = 940100102
	apTodo   = 940100201
	apMsgUp  = 940100301
	apBadRes = 940100150
)

type pushCommunity struct {
	*fakeCommunity
	mu      sync.Mutex
	posts   []communityclient.ActivityWriteRequest
	stored  map[string]communityclient.SiteActivityView
	nextID  int64
	bisect  bool
	badKey  string
	outcome map[string]string
	list    []communityclient.SiteActivityView
	short   bool
}

func newPushCommunity() *pushCommunity {
	return &pushCommunity{
		fakeCommunity: newFakeCommunity(),
		stored:        map[string]communityclient.SiteActivityView{},
		outcome:       map[string]string{},
		nextID:        1,
	}
}

func (c *pushCommunity) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/activities" {
		c.handleActivities(w, r)
		return
	}
	c.fakeCommunity.ServeHTTP(w, r)
}

func (c *pushCommunity) handleActivities(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if r.Method == http.MethodGet {
		writeEnvelope(w, 200, 0, "", communityclient.SiteActivityListResponse{Activities: append([]communityclient.SiteActivityView{}, c.list...)})
		return
	}
	if r.Method != http.MethodPost {
		writeEnvelope(w, http.StatusNotFound, 40400, "no route", nil)
		return
	}
	body, _ := io.ReadAll(r.Body)
	var req communityclient.ActivityWriteRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeEnvelope(w, http.StatusUnprocessableEntity, 4, "bad json", nil)
		return
	}
	c.posts = append(c.posts, req)
	if c.bisect {
		if len(req.Items) != 1 || (c.badKey != "" && req.Items[0].Key == c.badKey) {
			writeEnvelope(w, http.StatusUnprocessableEntity, 4, "malformed", nil)
			return
		}
	}
	results := make([]communityclient.ActivityWriteOutcome, 0, len(req.Items))
	for _, it := range req.Items {
		out := "created"
		if it.Removed {
			out = "removed"
		}
		if o, ok := c.outcome[it.Key]; ok {
			out = o
		}
		results = append(results, communityclient.ActivityWriteOutcome{Key: it.Key, Outcome: out})
		if out == "created" || out == "updated" || out == "removed" || out == "restored" {
			c.nextID++
			c.stored[it.Key] = viewFromItem(c.nextID, it)
		}
	}
	if c.short && len(results) > 0 {
		results = results[:len(results)-1]
	}
	writeEnvelope(w, 200, 0, "", communityclient.ActivityWriteResponse{Results: results})
}

func viewFromItem(id int64, it communityclient.ActivityWriteItem) communityclient.SiteActivityView {
	v := communityclient.SiteActivityView{
		ID: id, Key: it.Key, ActorID: it.ActorID, Revision: it.Revision,
		Verb: it.Verb, ObjectKind: it.ObjectKind, ObjectLabel: it.ObjectLabel,
		Title: it.Title, Excerpt: it.Excerpt, URL: it.URL, ContentLimit: it.ContentLimit,
		Notify: it.Notify, OccurredAt: it.OccurredAt, Removed: it.Removed,
	}
	if it.CoverImageHash != "" {
		h := it.CoverImageHash
		v.CoverImageHash = &h
	}
	if it.WorkID != 0 {
		w := it.WorkID
		v.WorkID = &w
	}
	return v
}

type pushFix struct {
	*writeFix
	cm     *pushCommunity
	pusher *activitypush.Pusher
}

func newPushFix(t *testing.T) *pushFix {
	t.Helper()
	cm := newPushCommunity()
	base := newWriteFixCommunity(t, nil, cm)
	base.Config.OAuth.RedirectURI = "https://www.kungal.com/api/auth/callback"
	origin := activitypush.Origin(base.Config.OAuth.RedirectURI)
	p := activitypush.New(base.db, base.Community, base.ActivityV1, origin)
	f := &pushFix{writeFix: base, cm: cm, pusher: p}
	f.resetQueue(t)
	t.Cleanup(func() { f.cleanupPush(t) })
	return f
}

func (f *pushFix) resetQueue(t *testing.T) {
	t.Helper()
	if err := f.db.Exec(`DELETE FROM activity_push_queue`).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Exec(`DELETE FROM activity_push_sent`).Error; err != nil {
		t.Fatal(err)
	}
	f.cm.mu.Lock()
	f.cm.posts = nil
	f.cm.mu.Unlock()
}

func (f *pushFix) cleanupPush(t *testing.T) {
	t.Helper()
	_ = f.db.Exec(`DELETE FROM topic WHERE id BETWEEN ? AND ?`, apTopicA, apTopicB).Error
	_ = f.db.Exec(`DELETE FROM galgame_resource WHERE id = ?`, apBadRes).Error
	_ = f.db.Exec(`DELETE FROM galgame WHERE id = ?`, 940100888).Error
	_ = f.db.Exec(`DELETE FROM feed_activity WHERE source_id BETWEEN ? AND ?`, 940100101, 940100399).Error
	_ = f.db.Exec(`DELETE FROM activity_push_queue WHERE source_id BETWEEN ? AND ?`, 940100101, 940100399).Error
	_ = f.db.Exec(`DELETE FROM activity_push_sent WHERE source_id BETWEEN ? AND ?`, 940100101, 940100399).Error
}

func (f *pushFix) topic(t *testing.T, id, user int, nsfw bool, created time.Time, title string) {
	t.Helper()
	if title == "" {
		title = fmt.Sprintf("t%d", id)
	}
	if err := f.db.Exec(`INSERT INTO topic (
		id, title, content, view, status, category, status_update_time, created, updated,
		user_id, is_nsfw, access_scope, cover_images, like_count, dislike_count, reply_count, comment_count,
		favorite_count, upvote_count, view_7d, view_30d, hidden_by, last_reply_floor
	) VALUES (?, ?, 'body', 0, 0, 'galgame', ?, ?, ?, ?, ?, 'public', '', 0, 0, 0, 0, 0, 0, 0, 0, '', 0)`,
		id, title, created, created, created, user, nsfw).Error; err != nil {
		t.Fatalf("topic %d: %v", id, err)
	}
}

func (f *pushFix) queueCount(t *testing.T) int {
	t.Helper()
	var n int
	if err := f.db.Raw(`SELECT count(*) FROM activity_push_queue`).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func (f *pushFix) queueRow(t *testing.T, typ string, source int) (backfill bool, found bool) {
	t.Helper()
	var rows []struct {
		Backfill bool `gorm:"column:backfill"`
	}
	if err := f.db.Raw(`SELECT backfill FROM activity_push_queue WHERE type = ? AND source_id = ?`, typ, source).Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		return false, false
	}
	return rows[0].Backfill, true
}

func (f *pushFix) sentRow(t *testing.T, typ string, source int) (actor int, rev int64, removed bool, found bool) {
	t.Helper()
	var rows []struct {
		ActorID  int   `gorm:"column:actor_id"`
		Revision int64 `gorm:"column:revision"`
		Removed  bool  `gorm:"column:removed"`
	}
	if err := f.db.Raw(`SELECT actor_id, revision, removed FROM activity_push_sent WHERE type = ? AND source_id = ?`,
		typ, source).Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		return 0, 0, false, false
	}
	return rows[0].ActorID, rows[0].Revision, rows[0].Removed, true
}

func (f *pushFix) writes() []communityclient.ActivityWriteRequest {
	f.cm.mu.Lock()
	defer f.cm.mu.Unlock()
	out := make([]communityclient.ActivityWriteRequest, len(f.cm.posts))
	copy(out, f.cm.posts)
	return out
}

func (f *pushFix) lastItems(t *testing.T) []communityclient.ActivityWriteItem {
	t.Helper()
	w := f.writes()
	if len(w) == 0 {
		t.Fatal("no POST /activities")
	}
	return w[len(w)-1].Items
}
