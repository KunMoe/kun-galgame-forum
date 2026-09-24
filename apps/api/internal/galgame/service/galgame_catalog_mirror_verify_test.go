package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/testdb"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const verifyBase = 2_000_500_000

type catalogFake struct {
	mu          sync.Mutex
	rated       map[int]string
	limits      map[int]string
	hidden      map[int]bool
	fates       map[int]string
	listStatus  int
	pages       map[string]string
	detailCalls int
}

func (c *catalogFake) handler(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch path := r.URL.Path; {
	case strings.HasSuffix(path, "/changes"):
		body, ok := c.pages[r.URL.Query().Get("cursor")]
		if !ok {
			body = `{"object":"list","items":[]}`
		}
		_, _ = w.Write([]byte(body))
	case strings.HasSuffix(path, "/works"):
		if c.listStatus != 0 {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(c.listStatus)
			_, _ = w.Write([]byte(`{"code":"SERVICE_UNAVAILABLE"}`))
			return
		}
		var items []string
		for _, raw := range strings.Split(r.URL.Query().Get("ids"), ",") {
			id, _ := strconv.Atoi(raw)
			switch {
			case c.hidden[id]:
				items = append(items, fmt.Sprintf(`{"id":"%d","content_rating":"r18","content_limit":"nsfw","claim":{"site":"kungal","site_work_id":"%d","state":"hidden","content_limit":"nsfw"}}`, id, id))
			case c.rated[id] != "":
				limit := ""
				if l, ok := c.limits[id]; ok {
					limit = fmt.Sprintf(`,"content_limit":%q`, l)
				}
				items = append(items, fmt.Sprintf(`{"id":"%d","content_rating":%q%s}`, id, c.rated[id], limit))
			}
		}
		_, _ = w.Write([]byte(`{"object":"list","items":[` + strings.Join(items, ",") + `]}`))
	case strings.Contains(path, "/works/"):
		c.detailCalls++
		id, _ := strconv.Atoi(path[strings.LastIndex(path, "/")+1:])
		fate := c.fates[id]
		w.Header().Set("Content-Type", "application/problem+json")
		switch {
		case strings.HasPrefix(fate, "merged:"):
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprintf(w, `{"code":"ENTITY_MERGED","current_id":%q}`, strings.TrimPrefix(fate, "merged:"))
		case fate == "down":
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"code":"SERVICE_UNAVAILABLE"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":"NOT_FOUND"}`))
		}
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

type mergeQueueFake struct {
	mu  sync.Mutex
	got map[int]int64
}

func (m *mergeQueueFake) Enqueue(_ context.Context, oldID int, survivor int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.got[oldID] = survivor
}

type verifyFix struct {
	mirror *GalgameCatalogMirror
	fake   *catalogFake
	merges *mergeQueueFake
	db     *gorm.DB
}

func newVerifyFix(t *testing.T) *verifyFix {
	t.Helper()
	db := testdb.Open(t)
	cleanup := func() {
		db.Exec("DELETE FROM galgame WHERE id >= ? AND id < ?", verifyBase, verifyBase+10_000)
	}
	cleanup()
	t.Cleanup(cleanup)

	fake := &catalogFake{rated: map[int]string{}, limits: map[int]string{}, hidden: map[int]bool{}, fates: map[int]string{}, pages: map[string]string{}}
	srv := httptest.NewServer(http.HandlerFunc(fake.handler))
	t.Cleanup(srv.Close)
	merges := &mergeQueueFake{got: map[int]int64{}}
	mirror := NewGalgameCatalogMirror(
		client.New(srv.URL, "nmk_test", ""),
		repository.NewGalgameRepository(db),
		redis.NewClient(&redis.Options{Addr: miniredis.RunT(t).Addr()}),
		merges,
	)
	return &verifyFix{mirror: mirror, fake: fake, merges: merges, db: db}
}

func (f *verifyFix) seed(t *testing.T, limit string, rendered bool, ids ...int) {
	t.Helper()
	for _, id := range ids {
		if err := f.db.Create(&model.GalgameLocal{ID: id, ContentLimit: ptr(limit)}).Error; err != nil {
			t.Fatalf("seed %d: %v", id, err)
		}
	}
	if !rendered {
		f.db.Exec("UPDATE galgame SET catalog_rendered = false WHERE id IN ?", ids)
	}
}

type rowState struct {
	Rendered bool    `gorm:"column:catalog_rendered"`
	Checked  bool    `gorm:"column:checked"`
	Limit    *string `gorm:"column:content_limit"`
}

func (f *verifyFix) state(t *testing.T, id int) rowState {
	t.Helper()
	var s rowState
	if err := f.db.Raw(`SELECT catalog_rendered, catalog_checked_at IS NOT NULL AS checked, content_limit
		FROM galgame WHERE id = ?`, id).Scan(&s).Error; err != nil {
		t.Fatal(err)
	}
	return s
}

func span(from, n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = from + i
	}
	return out
}

func TestMirrorVerifySettlesEveryKindOfAnswer(t *testing.T) {
	f := newVerifyFix(t)
	stale, hidden, gone, merged, survivor, mergedHidden, hiddenSurvivor, disagree :=
		verifyBase+1, verifyBase+2, verifyBase+3, verifyBase+4, verifyBase+5, verifyBase+6, verifyBase+7, verifyBase+8
	allExplicit, unanswered := verifyBase+9, verifyBase+10
	f.seed(t, "sfw", true, stale, hidden, gone, merged, mergedHidden, disagree, allExplicit, unanswered)
	f.fake.rated[stale], f.fake.limits[stale] = "r18", "nsfw"
	f.fake.rated[allExplicit], f.fake.limits[allExplicit] = "all_ages", "nsfw"
	f.fake.rated[unanswered] = "all_ages"
	f.fake.hidden[hidden] = true
	f.fake.fates[merged] = fmt.Sprintf("merged:%d", survivor)
	f.fake.rated[survivor] = "all_ages"
	f.fake.fates[mergedHidden] = fmt.Sprintf("merged:%d", hiddenSurvivor)
	f.fake.hidden[hiddenSurvivor] = true
	f.fake.fates[disagree] = "down"

	f.mirror.verify(t.Context(), []int{stale, hidden, gone, merged, mergedHidden, disagree, allExplicit, unanswered})

	if s := f.state(t, stale); !s.Rendered || !s.Checked || s.Limit == nil || *s.Limit != "nsfw" {
		t.Errorf("an unclaimed r18 row the mirror wrote sfw must now read nsfw: %+v", s)
	}
	if s := f.state(t, allExplicit); s.Limit == nil || *s.Limit != "nsfw" {
		t.Errorf("catalog's verdict, not the rating, decides an unclaimed all_ages row: %+v", s)
	}
	if s := f.state(t, unanswered); !s.Checked || s.Limit == nil || *s.Limit != "sfw" {
		t.Errorf("a row without a verdict keeps the one it had: %+v", s)
	}
	if s := f.state(t, hidden); s.Limit == nil || *s.Limit != "nsfw" {
		t.Errorf("a hidden row still takes catalog's verdict: %+v", s)
	}
	for name, id := range map[string]int{"hidden claim": hidden, "not found": gone, "merged": merged, "merged into a hidden survivor": mergedHidden} {
		if s := f.state(t, id); s.Rendered || !s.Checked {
			t.Errorf("%s: want unlisted and checked, got %+v", name, s)
		}
	}
	if s := f.state(t, disagree); !s.Rendered || s.Checked {
		t.Errorf("a failed detail lookup must leave the row alone: %+v", s)
	}
	if got := f.merges.got; len(got) != 1 || got[merged] != int64(survivor) {
		t.Errorf("merge-fold queue = %v, want only %d → %d", got, merged, survivor)
	}
}

func TestMirrorVerifyTouchesNothingOnAFailedAnswer(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusTooManyRequests, http.StatusUnauthorized} {
		f := newVerifyFix(t)
		ids := span(verifyBase+100, 3)
		f.seed(t, "sfw", true, ids...)
		f.fake.listStatus = status

		f.mirror.verify(t.Context(), ids)

		for _, id := range ids {
			if s := f.state(t, id); !s.Rendered || s.Checked {
				t.Errorf("batch face %d: row %d changed: %+v", status, id, s)
			}
		}
		if f.fake.detailCalls != 0 {
			t.Errorf("batch face %d: %d detail lookups after a failed batch", status, f.fake.detailCalls)
		}
	}
}

func TestMirrorVerifyHoldsAMassUnlisting(t *testing.T) {
	f := newVerifyFix(t)
	ids := span(verifyBase+200, 100)
	f.seed(t, "sfw", true, ids...)

	f.mirror.verify(t.Context(), ids)

	for _, id := range ids {
		if s := f.state(t, id); !s.Rendered || s.Checked {
			t.Fatalf("catalog answered with none of 100 rows and row %d changed: %+v", id, s)
		}
	}
	if f.fake.detailCalls != 0 {
		t.Errorf("the breaker must trip before the per-row lookups, made %d", f.fake.detailCalls)
	}
}

func TestMirrorVerifyUnlistsAtTheBreakerFloor(t *testing.T) {
	f := newVerifyFix(t)
	ids := span(verifyBase+400, 100)
	f.seed(t, "sfw", true, ids...)
	for _, id := range ids[20:] {
		f.fake.rated[id] = "all_ages"
	}

	f.mirror.verify(t.Context(), ids)

	for _, id := range ids[:20] {
		if s := f.state(t, id); s.Rendered || !s.Checked {
			t.Fatalf("20 of 100 is at the floor, not over it: row %d = %+v", id, s)
		}
	}
}

func TestMirrorVerifyAlwaysRelists(t *testing.T) {
	f := newVerifyFix(t)
	ids := span(verifyBase+600, 30)
	f.seed(t, "sfw", false, ids...)
	for _, id := range ids {
		f.fake.rated[id] = "all_ages"
	}

	f.mirror.verify(t.Context(), ids)

	for _, id := range ids {
		if s := f.state(t, id); !s.Rendered || !s.Checked {
			t.Fatalf("row %d answered by catalog must be listed again: %+v", id, s)
		}
	}
}

func TestMirrorChannelUnlistsGoneRows(t *testing.T) {
	f := newVerifyFix(t)
	gone := verifyBase + 700
	f.seed(t, "sfw", true, gone)
	f.fake.pages[""] = changePage("", fmt.Sprintf("gone:%d", gone), "999999999")

	f.mirror.RunMirror()

	if s := f.state(t, gone); s.Rendered || !s.Checked {
		t.Errorf("a local row the feed marks gone must be unlisted: %+v", s)
	}
}

func TestMirrorChannelHoldsAMassUnlisting(t *testing.T) {
	f := newVerifyFix(t)
	ids := span(verifyBase+800, 30)
	f.seed(t, "sfw", true, ids...)
	gone := make([]string, len(ids))
	for i, id := range ids {
		gone[i] = fmt.Sprintf("gone:%d", id)
	}
	f.fake.pages[""] = changePage("", gone...)

	f.mirror.RunMirror()

	for _, id := range ids {
		if s := f.state(t, id); !s.Rendered {
			t.Fatalf("30 gone rows on one page are over the breaker and row %d was unlisted", id)
		}
	}
}
