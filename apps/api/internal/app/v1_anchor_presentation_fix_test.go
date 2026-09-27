package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	activitypush "kun-galgame-api/internal/activity/push"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/communityclient"
)

const (
	apnWork     = 960200001
	apnWorkB    = 960200002
	apnWorkC    = 960200003
	apnRes      = 960200101
	apnRating   = 960200201
	apnQuiz     = 960200301
	apnQuizB    = 960200302
	apnQuizC    = 960200303
	apnQuizD    = 960200304
	apnQuizE    = 960200305
	apnQuizF    = 960200306
	apnTool     = 960200401
	apnSite     = 960200501
	apnCat      = 960200601
	apnFeedSID  = 960200701
	apnFeedWork = 960200009
	apnGoneUser = 960209998
	apnCover    = "abababababababababababababababababababababababababababababababab"
)

type apnCommunity struct {
	*fakeCommunity
	mu       sync.Mutex
	puts     []communityclient.AnchorPresentationWriteRequest
	stored   map[string]communityclient.AnchorPresentationView
	list     []communityclient.AnchorPresentationView
	outcome  map[string]string
	bisect   bool
	badKey   string
	short    bool
	fail503  bool
	putAt    time.Time
	listedAt time.Time
}

func newApnCommunity() *apnCommunity {
	return &apnCommunity{
		fakeCommunity: newFakeCommunity(),
		stored:        map[string]communityclient.AnchorPresentationView{},
		outcome:       map[string]string{},
	}
}

func apnKey(kind int32, id string) string {
	return strconv.Itoa(int(kind)) + ":" + id
}

func (c *apnCommunity) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/anchor-presentations" {
		c.handle(w, r)
		return
	}
	c.fakeCommunity.ServeHTTP(w, r)
}

func (c *apnCommunity) handle(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if r.Method == http.MethodGet {
		c.listedAt = time.Now()
		writeEnvelope(w, 200, 0, "", communityclient.AnchorPresentationListResponse{
			Presentations: append([]communityclient.AnchorPresentationView{}, c.list...),
		})
		return
	}
	if r.Method != http.MethodPut {
		writeEnvelope(w, http.StatusNotFound, 40400, "no route", nil)
		return
	}
	if c.fail503 {
		writeEnvelope(w, http.StatusServiceUnavailable, 5, "unavailable", nil)
		return
	}
	body, _ := io.ReadAll(r.Body)
	var req communityclient.AnchorPresentationWriteRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeEnvelope(w, http.StatusUnprocessableEntity, 4, "bad json", nil)
		return
	}
	c.puts = append(c.puts, req)
	c.putAt = time.Now()
	if c.bisect {
		if len(req.Items) != 1 || (c.badKey != "" && apnKey(req.Items[0].AnchorKind, req.Items[0].AnchorID) == c.badKey) {
			writeEnvelope(w, http.StatusUnprocessableEntity, 4, "malformed", nil)
			return
		}
	}
	results := make([]communityclient.AnchorPresentationOutcome, 0, len(req.Items))
	for _, it := range req.Items {
		key := apnKey(it.AnchorKind, it.AnchorID)
		out := "created"
		if it.Removed {
			out = "removed"
		}
		if prev, ok := c.stored[key]; ok && it.Revision <= prev.Revision {
			out = "stale"
		}
		if o, ok := c.outcome[key]; ok {
			out = o
		}
		results = append(results, communityclient.AnchorPresentationOutcome{
			AnchorKind: it.AnchorKind, AnchorID: it.AnchorID, Outcome: out,
		})
		if out == "created" || out == "updated" || out == "removed" || out == "restored" {
			c.stored[key] = apnViewFromItem(it)
		}
	}
	if c.short && len(results) > 0 {
		results = results[:len(results)-1]
	}
	writeEnvelope(w, 200, 0, "", communityclient.AnchorPresentationWriteResponse{Results: results})
}

func apnViewFromItem(it communityclient.AnchorPresentationItem) communityclient.AnchorPresentationView {
	v := communityclient.AnchorPresentationView{
		AnchorKind: it.AnchorKind, AnchorID: it.AnchorID, Revision: it.Revision,
		Title: it.Title, URL: it.URL, ContentLimit: it.ContentLimit, Removed: it.Removed,
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

type apnFix struct {
	*writeFix
	cm        *apnCommunity
	cat       *fakeCatalog
	presenter *activitypush.Presenter
}

func newApnFix(t *testing.T) *apnFix {
	t.Helper()
	cm := newApnCommunity()
	base := newWriteFixCommunity(t, nil, cm)
	base.Config.OAuth.RedirectURI = "https://www.kungal.com/api/auth/callback"
	cat := &fakeCatalog{rows: map[int]client.CatalogWorkListItem{}, omitIDs: map[int]bool{}}
	origin := activitypush.Origin(base.Config.OAuth.RedirectURI)
	p := activitypush.NewPresenter(base.db, base.Community, base.UserClient, cat, "https://image.test.example", origin)
	f := &apnFix{writeFix: base, cm: cm, cat: cat, presenter: p}
	f.resetQueue(t)
	t.Cleanup(func() { f.cleanupApn(t) })
	return f
}

func (f *apnFix) resetQueue(t *testing.T) {
	t.Helper()
	if err := f.db.Exec(`DELETE FROM anchor_presentation_queue`).Error; err != nil {
		t.Fatal(err)
	}
	f.cm.mu.Lock()
	f.cm.puts = nil
	f.cm.mu.Unlock()
}

func (f *apnFix) cleanupApn(t *testing.T) {
	t.Helper()
	qs := []string{
		`DELETE FROM galgame_quiz_galgame WHERE quiz_id BETWEEN 960200301 AND 960200399 OR work_id BETWEEN 960200001 AND 960200099`,
		`DELETE FROM galgame_quiz WHERE id BETWEEN 960200301 AND 960200399`,
		`DELETE FROM galgame_resource WHERE id BETWEEN 960200101 AND 960200199`,
		`DELETE FROM galgame_rating WHERE id BETWEEN 960200201 AND 960200299`,
		`DELETE FROM galgame_toolset WHERE id BETWEEN 960200401 AND 960200499`,
		`DELETE FROM galgame_website WHERE id BETWEEN 960200501 AND 960200599`,
		`DELETE FROM galgame_website_category WHERE id BETWEEN 960200601 AND 960200699`,
		`DELETE FROM galgame WHERE id BETWEEN 960200001 AND 960200099`,
		`DELETE FROM feed_activity WHERE source_id BETWEEN 960200701 AND 960200799`,
		`DELETE FROM anchor_presentation_queue WHERE anchor_kind = 1 AND anchor_id LIKE '9602%'`,
		`DELETE FROM anchor_presentation_sent WHERE anchor_kind = 1 AND anchor_id LIKE '9602%'`,
		`DELETE FROM activity_push_queue WHERE source_id BETWEEN 960200001 AND 960209999`,
		`DELETE FROM activity_push_sent WHERE source_id BETWEEN 960200001 AND 960209999`,
		`DELETE FROM anchor_presentation_queue WHERE anchor_id LIKE 'resource:9602001%' OR anchor_id LIKE 'rating:9602002%' OR anchor_id LIKE 'quiz:9602003%' OR anchor_id LIKE 'toolset:9602004%' OR anchor_id LIKE 'website:9602005%'`,
		`DELETE FROM anchor_presentation_sent WHERE anchor_id LIKE 'resource:9602001%' OR anchor_id LIKE 'rating:9602002%' OR anchor_id LIKE 'quiz:9602003%' OR anchor_id LIKE 'toolset:9602004%' OR anchor_id LIKE 'website:9602005%'`,
	}
	for _, q := range qs {
		if err := f.db.Exec(q).Error; err != nil {
			t.Fatalf("cleanup: %v\n%s", err, q)
		}
	}
}

func (f *apnFix) runSQL(t *testing.T, q string, args ...any) {
	t.Helper()
	if err := f.db.Exec(q, args...).Error; err != nil {
		t.Fatalf("%v\n%s", err, q)
	}
}

func (f *apnFix) seedWork(t *testing.T, id int, limit string) {
	t.Helper()
	f.cat.mu.Lock()
	f.cat.rows[id] = catalogItem(t, id, limit)
	f.cat.mu.Unlock()
}

func (f *apnFix) insertGalgame(t *testing.T, id int, published bool) {
	t.Helper()
	now := time.Now()
	f.runSQL(t, `INSERT INTO galgame (id, created, updated, published) VALUES (?, ?, ?, ?)`, id, now, now, published)
}

func (f *apnFix) insertResource(t *testing.T, id, workID, userID int) {
	t.Helper()
	f.runSQL(t, `INSERT INTO galgame_resource (id, work_id, user_id, updated) VALUES (?, ?, ?, ?)`, id, workID, userID, time.Now())
}

func (f *apnFix) insertRating(t *testing.T, id, workID, userID int) {
	t.Helper()
	f.runSQL(t, `INSERT INTO galgame_rating (id, work_id, user_id, recommend, overall, updated) VALUES (?, ?, ?, 'yes', 8, ?)`,
		id, workID, userID, time.Now())
}

func (f *apnFix) insertQuiz(t *testing.T, id int64, userID int, question string, hide bool) {
	t.Helper()
	f.runSQL(t, `INSERT INTO galgame_quiz (id, user_id, type, question, spoiler_level, hide_galgame) VALUES (?, ?, 'single', ?, 'none', ?)`,
		id, userID, question, hide)
}

func (f *apnFix) linkQuiz(t *testing.T, quizID int64, workID int) {
	t.Helper()
	f.runSQL(t, `INSERT INTO galgame_quiz_galgame (quiz_id, work_id) VALUES (?, ?)`, quizID, workID)
}

func (f *apnFix) insertToolset(t *testing.T, id, userID, status int, name string) {
	t.Helper()
	f.runSQL(t, `INSERT INTO galgame_toolset (id, name, status, user_id, updated) VALUES (?, ?, ?, ?, ?)`,
		id, name, status, userID, time.Now())
}

func (f *apnFix) insertWebsite(t *testing.T, id int, name, url, age string) {
	t.Helper()
	now := time.Now()
	var n int
	if err := f.db.Raw(`SELECT count(*) FROM galgame_website_category WHERE id = ?`, apnCat).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		f.runSQL(t, `INSERT INTO galgame_website_category (id, name, updated) VALUES (?, 'apn', ?)`, apnCat, now)
	}
	f.runSQL(t, `INSERT INTO galgame_website (id, name, url, create_time, category_id, age_limit, updated) VALUES (?, ?, ?, '2020', ?, ?, ?)`,
		id, name, url, apnCat, age, now)
}

func (f *apnFix) seedSent(t *testing.T, kind int16, id string, rev int64, removed bool) {
	t.Helper()
	f.runSQL(t, `INSERT INTO anchor_presentation_sent (anchor_kind, anchor_id, revision, removed)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (anchor_kind, anchor_id) DO UPDATE SET revision = EXCLUDED.revision, removed = EXCLUDED.removed`,
		kind, id, rev, removed)
}

func (f *apnFix) queueHas(t *testing.T, kind int16, id string) bool {
	t.Helper()
	var n int
	if err := f.db.Raw(`SELECT count(*) FROM anchor_presentation_queue WHERE anchor_kind = ? AND anchor_id = ?`, kind, id).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n > 0
}

func (f *apnFix) queueCount(t *testing.T) int {
	t.Helper()
	var n int
	if err := f.db.Raw(`SELECT count(*) FROM anchor_presentation_queue`).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func (f *apnFix) sent(t *testing.T, kind int16, id string) (rev int64, removed bool, found bool) {
	t.Helper()
	var rows []struct {
		Revision int64 `gorm:"column:revision"`
		Removed  bool  `gorm:"column:removed"`
	}
	if err := f.db.Raw(`SELECT revision, removed FROM anchor_presentation_sent WHERE anchor_kind = ? AND anchor_id = ?`,
		kind, id).Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		return 0, false, false
	}
	return rows[0].Revision, rows[0].Removed, true
}

func (f *apnFix) writes() []communityclient.AnchorPresentationWriteRequest {
	f.cm.mu.Lock()
	defer f.cm.mu.Unlock()
	out := make([]communityclient.AnchorPresentationWriteRequest, len(f.cm.puts))
	copy(out, f.cm.puts)
	return out
}

func (f *apnFix) allItems() []communityclient.AnchorPresentationItem {
	var out []communityclient.AnchorPresentationItem
	for _, w := range f.writes() {
		out = append(out, w.Items...)
	}
	return out
}

func (f *apnFix) lastItems(t *testing.T) []communityclient.AnchorPresentationItem {
	t.Helper()
	w := f.writes()
	if len(w) == 0 {
		t.Fatal("no PUT /anchor-presentations")
	}
	return w[len(w)-1].Items
}

func (f *apnFix) findItem(kind int32, id string) (communityclient.AnchorPresentationItem, bool) {
	items := f.allItems()
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].AnchorKind == kind && items[i].AnchorID == id {
			return items[i], true
		}
	}
	return communityclient.AnchorPresentationItem{}, false
}

func apnItemKeys(t *testing.T, it communityclient.AnchorPresentationItem) map[string]any {
	t.Helper()
	b, err := json.Marshal(it)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func apnWorkTitle(id int) string { return fmt.Sprintf("作品%d", id) }
