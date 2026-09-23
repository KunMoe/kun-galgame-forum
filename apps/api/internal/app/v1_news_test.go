package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	newsapiv1 "kun-galgame-api/internal/news/apiv1"
	"kun-galgame-api/pkg/newsclient"
)

type fakeNewsItem struct {
	id        int64
	source    string
	lane      string
	published time.Time
}

type fakeNews struct {
	mu        sync.Mutex
	items     []fakeNewsItem
	queries   []url.Values
	staleNext bool
}

var cst = time.FixedZone("CST", 8*60*60)

func newFakeNews() *fakeNews {
	at := func(y, m, d, h int) time.Time { return time.Date(y, time.Month(m), d, h, 0, 0, 0, cst) }
	items := []fakeNewsItem{
		{112, "ymgal", "news", at(2026, 9, 20, 10)},
		{111, "ymgal", "column", at(2026, 9, 5, 9)},
		{110, "galgame_hihyou", "news", at(2026, 9, 5, 9)},
		{109, "ymgal", "news", at(2026, 9, 1, 3)},
		{108, "ymgal", "news", at(2026, 9, 1, 3)},
		{107, "galgame_hihyou", "column", at(2026, 8, 31, 23)},
		{106, "ymgal", "news", at(2026, 8, 12, 12)},
		{105, "ymgal", "news", at(2026, 1, 1, 1)},
		{104, "galgame_hihyou", "news", at(2025, 12, 31, 23)},
		{103, "ymgal", "news", at(2025, 6, 6, 6)},
	}
	return &fakeNews{items: items}
}

func (n *fakeNews) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/news/sources", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "items": []map[string]any{
			{"name": "ymgal", "display_name": "月幕 Galgame", "homepage_url": "https://www.ymgal.games", "column_url": "", "attribution": "转载自月幕", "publisher_uid": w3UserAlice},
			{"name": "galgame_hihyou", "display_name": "Galgame 批评", "homepage_url": "", "column_url": "https://example.com/col", "attribution": "", "publisher_uid": 0},
		}})
	})
	mux.HandleFunc("/v2/news", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		n.mu.Lock()
		n.queries = append(n.queries, q)
		stale := n.staleNext
		n.mu.Unlock()
		cursor := q.Get("cursor")
		if strings.HasPrefix(cursor, "stale") {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":400,"message":"cursor expired"}`))
			return
		}
		var after, before time.Time
		if v := q.Get("published_after"); v != "" {
			after, _ = time.Parse(time.RFC3339Nano, v)
		}
		if v := q.Get("published_before"); v != "" {
			before, _ = time.Parse(time.RFC3339Nano, v)
		}
		var matched []fakeNewsItem
		for _, it := range n.items {
			if (q.Get("source") == "" || it.source == q.Get("source")) && (q.Get("lane") == "" || it.lane == q.Get("lane")) &&
				(after.IsZero() || !it.published.Before(after)) && (before.IsZero() || !it.published.After(before)) {
				matched = append(matched, it)
			}
		}
		offset, _ := strconv.Atoi(strings.TrimPrefix(cursor, "o"))
		limit, _ := strconv.Atoi(q.Get("limit"))
		if limit <= 0 {
			limit = 20
		}
		end := min(offset+limit, len(matched))
		page := []map[string]any{}
		for _, it := range matched[min(offset, len(matched)):end] {
			page = append(page, map[string]any{
				"id": strconv.FormatInt(it.id, 10), "source": map[string]any{"name": it.source}, "lane": it.lane,
				"source_url": fmt.Sprintf("https://partner.example/%d", it.id), "title": fmt.Sprintf("item %d", it.id),
				"summary": "lede", "published_at": it.published.UTC().Format(time.RFC3339),
			})
		}
		body := map[string]any{"object": "list", "items": page, "total": len(matched)}
		if end < len(matched) {
			next := "o" + strconv.Itoa(end)
			if stale {
				next = "stale" + next
			}
			body["next_cursor"] = next
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	})
	return mux
}

func (n *fakeNews) requestCount() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.queries)
}

func (n *fakeNews) lastQuery() url.Values {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.queries[len(n.queries)-1]
}

func newNewsFix(t *testing.T, configured bool) (*writeFix, *fakeNews) {
	t.Helper()
	f := newWriteFix(t, nil)
	news := newFakeNews()
	cfg := newsclient.Config{}
	if configured {
		srv := httptest.NewServer(news.handler())
		t.Cleanup(srv.Close)
		cfg = newsclient.Config{BaseURL: srv.URL, APIKey: "k"}
	}
	f.NewsV1 = newsapiv1.New(newsclient.New(cfg), f.UserClient, "https://image.test.example")
	f.Fiber = newFiber()
	f.setupRoutes()
	return f, news
}

func (f *writeFix) newsGet(t *testing.T, path, spec string) (*http.Response, map[string]any) {
	t.Helper()
	return f.docCall(t, http.MethodGet, "/api/v1"+path, spec, "", "", nil, nil)
}

func (f *writeFix) walkNews(t *testing.T, q url.Values) []string {
	t.Helper()
	var got []string
	for range 20 {
		resp, body := f.newsGet(t, "/news-items?"+q.Encode(), "/news-items")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("news %v: %d %+v", q, resp.StatusCode, body)
		}
		got = append(got, adminItemIDs(body)...)
		next, _ := body["next_cursor"].(string)
		if next == "" {
			return got
		}
		q.Set("cursor", next)
	}
	t.Fatal("walk did not end")
	return nil
}

func TestV1NewsItemsCursorIsBoundToFilters(t *testing.T) {
	f, _ := newNewsFix(t, true)
	got := f.walkNews(t, url.Values{"news_source": {"ymgal"}, "limit": {"2"}})
	if joinIDs(got) != "112,111,109,108,106,105,103" {
		t.Errorf("ymgal walk %v", got)
	}
	resp, body := f.newsGet(t, "/news-items?news_source=ymgal&limit=2&include_total=true", "/news-items")
	if resp.StatusCode != http.StatusOK || asInt(body["total"]) != 7 {
		t.Fatalf("total %d %+v", resp.StatusCode, body)
	}
	items, _ := body["items"].([]any)
	first, _ := items[0].(map[string]any)
	if first["object"] != "news_item" || first["news_source"] != "ymgal" || first["preview"] != "lede" {
		t.Errorf("item %+v", first)
	}
	if _, has := first["banner_url"]; has {
		t.Error("banner_url is not modeled")
	}
	cur, _ := body["next_cursor"].(string)
	for _, q := range []string{
		"news_source=galgame_hihyou&limit=2", "limit=2", "news_source=ymgal&limit=3", "news_source=ymgal&limit=2&lane=news",
	} {
		resp, body := f.newsGet(t, "/news-items?"+q+"&cursor="+cur, "/news-items")
		if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_CURSOR" {
			t.Errorf("cursor reused with %s: %d %+v", q, resp.StatusCode, body)
		}
	}
	if resp, body := f.newsGet(t, "/news-items?limit=51", "/news-items"); resp.StatusCode != http.StatusBadRequest || body["code"] != "LIMIT_TOO_LARGE" {
		t.Errorf("limit 51: %d %+v", resp.StatusCode, body)
	}
}

func TestV1NewsItemsMonthWindow(t *testing.T) {
	f, news := newNewsFix(t, true)
	got := f.walkNews(t, url.Values{"year": {"2026"}, "month": {"9"}})
	if joinIDs(got) != "112,111,110,109,108" {
		t.Errorf("september %v", got)
	}
	q := news.lastQuery()
	if q.Get("published_after") != "2026-08-31T16:00:00Z" || q.Get("published_before") != "2026-09-30T15:59:59.999999Z" {
		t.Errorf("window sent upstream: after %q before %q", q.Get("published_after"), q.Get("published_before"))
	}
	resp, body := f.newsGet(t, "/news-items?month=9", "/news-items")
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_PARAMETER" {
		t.Errorf("month without year: %d %+v", resp.StatusCode, body)
	}
}

func TestV1NewsItemsUpstreamFailures(t *testing.T) {
	f, news := newNewsFix(t, true)
	news.staleNext = true
	_, body := f.newsGet(t, "/news-items?limit=2", "/news-items")
	cur, _ := body["next_cursor"].(string)
	resp, body := f.newsGet(t, "/news-items?limit=2&cursor="+cur, "/news-items")
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_CURSOR" {
		t.Errorf("upstream rejects the cursor: %d %+v", resp.StatusCode, body)
	}

	off, _ := newNewsFix(t, false)
	for path, spec := range map[string]string{
		"/news-items": "/news-items", "/news-sources": "/news-sources", "/news-archive": "/news-archive",
		"/news-archive/2026/9": "/news-archive/{year}/{month}", "/news-archive/2026/9/items": "/news-archive/{year}/{month}/items",
	} {
		if resp, body := off.newsGet(t, path, spec); resp.StatusCode != http.StatusServiceUnavailable || body["code"] != "SERVICE_UNAVAILABLE" {
			t.Errorf("%s unconfigured: %d %+v", path, resp.StatusCode, body)
		}
	}
}

func TestV1NewsSourcesForumAccount(t *testing.T) {
	f, _ := newNewsFix(t, true)
	resp, body := f.newsGet(t, "/news-sources", "/news-sources")
	items, _ := body["items"].([]any)
	if resp.StatusCode != http.StatusOK || len(items) != 2 {
		t.Fatalf("sources %d %+v", resp.StatusCode, body)
	}
	ym, _ := items[0].(map[string]any)
	account, _ := ym["forum_account"].(map[string]any)
	if ym["key"] != "ymgal" || ym["display_name"] != "月幕 Galgame" || account["id"] != fmt.Sprint(w3UserAlice) {
		t.Errorf("ymgal %+v", ym)
	}
	if hi, _ := items[1].(map[string]any); hi["forum_account"] != nil {
		t.Errorf("no publisher → null: %+v", hi)
	}

	down, _ := newNewsFix(t, true)
	down.failOA.Store(true)
	resp, body = down.newsGet(t, "/news-sources", "/news-sources")
	items, _ = body["items"].([]any)
	if resp.StatusCode != http.StatusOK || len(items) != 2 {
		t.Fatalf("accounts down: %d %+v", resp.StatusCode, body)
	}
	if ym, _ := items[0].(map[string]any); ym["forum_account"] != nil {
		t.Errorf("accounts down leaves forum_account null: %+v", ym)
	}
}

func TestV1NewsArchive(t *testing.T) {
	f, news := newNewsFix(t, true)
	resp, body := f.newsGet(t, "/news-archive?news_source=ymgal", "/news-archive")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("archive %d %+v", resp.StatusCode, body)
	}
	years, _ := json.Marshal(body["years"])
	if string(years) != `[{"count":6,"year":2026},{"count":1,"year":2025}]` {
		t.Errorf("years %s", years)
	}
	if months, _ := body["months"].([]any); len(months) != 0 {
		t.Errorf("no year → no months: %v", months)
	}
	_, body = f.newsGet(t, "/news-archive?news_source=ymgal&year=2026", "/news-archive")
	months, _ := json.Marshal(body["months"])
	if string(months) != `[{"count":1,"month":1},{"count":1,"month":8},{"count":4,"month":9}]` {
		t.Errorf("months %s", months)
	}
	before := news.requestCount()
	_, body = f.newsGet(t, "/news-archive?news_source=ymgal&year=2019", "/news-archive")
	if m, _ := body["months"].([]any); len(m) != 0 || news.requestCount() != before {
		t.Errorf("a year with no items asks nothing more: months %v, %d new requests", m, news.requestCount()-before)
	}
}

func TestV1NewsMonth(t *testing.T) {
	f, _ := newNewsFix(t, true)
	resp, body := f.newsGet(t, "/news-archive/2026/9", "/news-archive/{year}/{month}")
	days, _ := body["days"].([]any)
	if resp.StatusCode != http.StatusOK || body["object"] != "news_month" || asInt(body["item_count"]) != 5 || len(days) != 30 {
		t.Fatalf("month %d %+v", resp.StatusCode, body)
	}
	if d1, _ := days[0].(map[string]any); asInt(d1["day"]) != 1 || asInt(d1["count"]) != 2 {
		t.Errorf("day 1 %+v", d1)
	}

	var walked []string
	for page := 1; page <= 5; page++ {
		resp, body := f.newsGet(t, fmt.Sprintf("/news-archive/2026/9/items?limit=2&page=%d", page), "/news-archive/{year}/{month}/items")
		if resp.StatusCode != http.StatusOK || asInt(body["total"]) != 5 || body["total_relation"] != "eq" {
			t.Fatalf("page %d: %d %+v", page, resp.StatusCode, body)
		}
		walked = append(walked, adminItemIDs(body)...)
	}
	if joinIDs(walked) != "112,111,110,109,108" {
		t.Errorf("pages %v", walked)
	}
	resp, body = f.newsGet(t, "/news-archive/2026/9/items?day=5", "/news-archive/{year}/{month}/items")
	if joinIDs(adminItemIDs(body)) != "111,110" || asInt(body["total"]) != 2 {
		t.Errorf("day 5: %d %+v", resp.StatusCode, body)
	}
	if resp, _ := f.newsGet(t, "/news-archive/2026/13", "/news-archive/{year}/{month}"); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("month 13: %d", resp.StatusCode)
	}
}
