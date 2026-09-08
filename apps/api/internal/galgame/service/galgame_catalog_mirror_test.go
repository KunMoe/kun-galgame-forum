package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"kun-galgame-api/internal/galgame/client"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

type mirrorStub struct {
	mu sync.Mutex
	// pages maps the cursor a request arrives with to the body it answers.
	pages       map[string]string
	cursorsSeen []string
	hydrated    []string
	failAfter   int
	feedCalls   int
}

func newMirror(t *testing.T, stub *mirrorStub) (*GalgameCatalogMirror, *miniredis.Miniredis) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stub.mu.Lock()
		defer stub.mu.Unlock()
		switch {
		case strings.HasSuffix(r.URL.Path, "/changes"):
			cursor := r.URL.Query().Get("cursor")
			stub.cursorsSeen = append(stub.cursorsSeen, cursor)
			stub.feedCalls++
			if stub.failAfter > 0 && stub.feedCalls > stub.failAfter {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"code":"INTERNAL"}`))
				return
			}
			body, ok := stub.pages[cursor]
			if !ok {
				body = `{"object":"list","items":[]}`
			}
			_, _ = w.Write([]byte(body))
		case strings.HasSuffix(r.URL.Path, "/works"):
			stub.hydrated = append(stub.hydrated, r.URL.Query().Get("ids"))
			_, _ = w.Write([]byte(`{"object":"list","items":[]}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	mr := miniredis.RunT(t)
	s := NewGalgameCatalogMirror(
		client.New(srv.URL, "nmk_test", ""),
		nil,
		redis.NewClient(&redis.Options{Addr: mr.Addr()}),
	)
	return s, mr
}

func changePage(cursor string, ids ...string) string {
	rows := make([]string, len(ids))
	for i, id := range ids {
		gone := ""
		if rest, ok := strings.CutPrefix(id, "gone:"); ok {
			id, gone = rest, `,"gone":true`
		}
		rows[i] = fmt.Sprintf(`{"object":"change","target_object":"work","id":%q,"updated_at":"2026-08-01T00:00:00Z"%s}`, id, gone)
	}
	next := ""
	if cursor != "" {
		next = fmt.Sprintf(`,"next_cursor":%q`, cursor)
	}
	return fmt.Sprintf(`{"object":"list","items":[%s]%s}`, strings.Join(rows, ","), next)
}

func storedCursor(t *testing.T, mr *miniredis.Miniredis) string {
	t.Helper()
	got, err := mr.Get(mirrorCursorKey)
	if err != nil {
		return ""
	}
	return got
}

func TestMirrorWalksTheFeedAndStopsOnTheLastPage(t *testing.T) {
	stub := &mirrorStub{pages: map[string]string{
		"":      changePage("cur_1", "101"),
		"cur_1": changePage("cur_2", "102"),
		"cur_2": changePage("", "103"),
	}}
	s, mr := newMirror(t, stub)
	s.RunMirror()

	if want := []string{"", "cur_1", "cur_2"}; !slices.Equal(stub.cursorsSeen, want) {
		t.Fatalf("cursors sent = %v, want %v", stub.cursorsSeen, want)
	}
	// The last page carries no next_cursor, so the stored cursor stays on the
	// page before it. Advancing past the tail is not possible, and re-reading
	// that page is the only way new rows are ever noticed.
	if got := storedCursor(t, mr); got != "cur_2" {
		t.Errorf("stored cursor = %q, want cur_2", got)
	}
}

func TestMirrorResumesFromTheLastPageThatLanded(t *testing.T) {
	// The old sweep aborted the whole round on its first error and started over
	// from the beginning next time. The cursor makes the failure cost one page.
	stub := &mirrorStub{
		pages: map[string]string{
			"":      changePage("cur_1", "101"),
			"cur_1": changePage("cur_2", "102"),
		},
		failAfter: 1,
	}
	s, mr := newMirror(t, stub)
	s.RunMirror()

	if got := storedCursor(t, mr); got != "cur_1" {
		t.Fatalf("stored cursor = %q, want cur_1 — the page that failed must not be skipped", got)
	}

	stub.failAfter = 0
	stub.cursorsSeen = nil
	s.RunMirror()
	if len(stub.cursorsSeen) == 0 || stub.cursorsSeen[0] != "cur_1" {
		t.Errorf("second run started at %v, want it to resume at cur_1", stub.cursorsSeen)
	}
}

func TestMirrorNeverHydratesAGoneID(t *testing.T) {
	stub := &mirrorStub{pages: map[string]string{
		"":      changePage("cur_1", "101", "gone:102", "103"),
		"cur_1": changePage("", "gone:201", "gone:202"),
	}}
	s, _ := newMirror(t, stub)
	s.RunMirror()

	if len(stub.hydrated) != 1 {
		t.Fatalf("hydration calls = %v, want one — a page of only gone ids has nothing to ask about", stub.hydrated)
	}
	ids := strings.Split(stub.hydrated[0], ",")
	if !slices.Equal(ids, []string{"101", "103"}) {
		t.Errorf("hydrated ids = %v, want the live ones only", ids)
	}
}

func TestMirrorStopsAtThePageCap(t *testing.T) {
	// Without a cap one tick would follow the feed forever, and the 110k-work
	// night of a machine re-grade would hold the lock past the next tick.
	stub := &mirrorStub{pages: map[string]string{}}
	for i := range 10 {
		stub.pages[cursorAt(i)] = changePage(cursorAt(i+1), "1")
	}
	s, mr := newMirror(t, stub)
	s.maxPages = 3
	s.RunMirror()

	if len(stub.cursorsSeen) != 3 {
		t.Fatalf("read %d pages, want the cap of 3", len(stub.cursorsSeen))
	}
	if got := storedCursor(t, mr); got != cursorAt(3) {
		t.Errorf("stored cursor = %q, want %q so the next tick continues", got, cursorAt(3))
	}
}

func cursorAt(i int) string {
	if i == 0 {
		return ""
	}
	return fmt.Sprintf("cur_%d", i)
}

func TestMirrorHoldsTheCursorWhenRedisIsDown(t *testing.T) {
	stub := &mirrorStub{pages: map[string]string{"": changePage("cur_1", "101")}}
	s, mr := newMirror(t, stub)
	mr.Close()
	s.RunMirror()

	if len(stub.cursorsSeen) != 0 {
		t.Errorf("drained %v with no readable cursor — an unreadable cursor is not an empty one, "+
			"and treating it as empty replays the whole inventory every tick", stub.cursorsSeen)
	}
}

// A local row catalog has no work for is never confirmed, so without the memo
// the ten-minute pass re-asks about the same orphans on every tick — and now
// that the pass is capped, orphans at the front of the window would starve every
// row behind them. The nightly full sweep used to be what cleared the memo;
// nothing does now, so it has to expire.
func TestOrphanMemoExpires(t *testing.T) {
	s := NewGalgameCatalogMirror(nil, nil, nil)
	answered := func(gids ...int) map[int]client.CatalogMirror {
		out := map[int]client.CatalogMirror{}
		for _, gid := range gids {
			out[gid] = client.CatalogMirror{ContentLimit: "sfw"}
		}
		return out
	}
	memoised := func() []int {
		got := s.memoisedOrphans()
		slices.Sort(got)
		return got
	}
	asked := []int{1, 2, 3}

	s.rememberUnresolved(asked, answered(1, 3))
	if got := memoised(); !slices.Equal(got, []int{2}) {
		t.Fatalf("memoised = %v, want [2]", got)
	}

	s.rememberUnresolved(asked, answered())
	if got := memoised(); !slices.Equal(got, asked) {
		t.Fatalf("memoised = %v, want every id", got)
	}

	s.rememberUnresolved([]int{2}, answered(2))
	if got := memoised(); !slices.Equal(got, []int{1, 3}) {
		t.Fatalf("an adopted orphan stays skipped: memoised = %v, want [1 3]", got)
	}

	s.unresolved[1] = time.Now().Add(-time.Minute)
	if got := memoised(); !slices.Equal(got, []int{3}) {
		t.Fatalf("memoised = %v, want the expired entry dropped", got)
	}
}

func TestGroupByContentLimitDropsUnknownVerdicts(t *testing.T) {
	got := groupByContentLimit(map[int]string{1: "sfw", 2: "nsfw", 3: "", 4: "all"})
	if !slices.Equal(got["sfw"], []int{1}) || !slices.Equal(got["nsfw"], []int{2}) {
		t.Fatalf("groupByContentLimit = %v", got)
	}
	if len(got) != 2 {
		t.Fatalf("groupByContentLimit kept %d buckets, want only sfw and nsfw", len(got))
	}
}
