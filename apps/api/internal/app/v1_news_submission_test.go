package app

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"kun-galgame-api/internal/apiv1/content"
	newsapiv1 "kun-galgame-api/internal/news/apiv1"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/newsclient"
	"kun-galgame-api/pkg/problem"
)

const (
	newsSubList = "/me/news-submissions"
	newsSubItem = "/me/news-submissions/{news_submission_id}"
)

type meNewsCall struct {
	Method string
	Path   string
	Raw    []byte
	Header http.Header
	Query  url.Values
}

type meNewsFault struct {
	status int
	body   []byte
	retry  string
}

type meNewsItem struct {
	id          string
	title       string
	summary     string
	body        string
	sourceURL   string
	status      string
	publishedAt string
}

func (it meNewsItem) payload() map[string]any {
	return map[string]any{
		"object":        "news_submission",
		"id":            it.id,
		"source":        map[string]any{"object": "news_source", "name": "community"},
		"lane":          "news",
		"status":        it.status,
		"title":         it.title,
		"summary":       it.summary,
		"source_url":    it.sourceURL,
		"published_at":  it.publishedAt,
		"body":          it.body,
		"submitter_uid": strconv.Itoa(w3UserAlice),
	}
}

type fakeMeNews struct {
	mu       sync.Mutex
	items    []meNewsItem
	recorded []meNewsCall
	faults   map[string]meNewsFault
	nextID   int
}

func newFakeMeNews() *fakeMeNews {
	return &fakeMeNews{
		items: []meNewsItem{
			{"4802", "newer", "lede-new", "body-new", "", "pending", "2026-09-30T03:00:00Z"},
			{"4801", "older", "lede-old", "body-old", "", "pending", "2026-09-30T02:00:00Z"},
		},
		nextID: 4900,
	}
}

func (n *fakeMeNews) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if fault, ok := n.note(r, raw); ok {
			if fault.retry != "" {
				w.Header().Set("Retry-After", fault.retry)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(fault.status)
			_, _ = w.Write(fault.body)
			return
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/me/news":
			n.create(w, raw)
		case r.Method == http.MethodGet && r.URL.Path == "/v2/me/news":
			n.list(w, r)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/me/news/"):
			n.getOne(w, strings.TrimPrefix(r.URL.Path, "/v2/me/news/"))
		case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/v2/me/news/"):
			n.patch(w, strings.TrimPrefix(r.URL.Path, "/v2/me/news/"), raw)
		default:
			http.NotFound(w, r)
		}
	})
}

func (n *fakeMeNews) note(r *http.Request, raw []byte) (meNewsFault, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.recorded = append(n.recorded, meNewsCall{
		Method: r.Method,
		Path:   r.URL.Path,
		Raw:    append([]byte(nil), raw...),
		Header: r.Header.Clone(),
		Query:  r.URL.Query(),
	})
	fault, ok := n.faults[r.Method]
	return fault, ok
}

func (n *fakeMeNews) calls() []meNewsCall {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]meNewsCall(nil), n.recorded...)
}

func (n *fakeMeNews) fail(method string, status int, body, retry string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.faults == nil {
		n.faults = map[string]meNewsFault{}
	}
	n.faults[method] = meNewsFault{status: status, body: []byte(body), retry: retry}
}

func (n *fakeMeNews) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	n.mu.Lock()
	items := append([]meNewsItem(nil), n.items...)
	n.mu.Unlock()
	start := 0
	if cur := q.Get("cursor"); cur != "" {
		start = len(items)
		for i, it := range items {
			if it.id == cur {
				start = i + 1
				break
			}
		}
	}
	end := min(start+limit, len(items))
	page := make([]any, 0, end-start)
	for _, it := range items[start:end] {
		page = append(page, it.payload())
	}
	body := map[string]any{"object": "list", "items": page}
	if end < len(items) {
		body["next_cursor"] = items[end-1].id
	}
	writeMeJSON(w, http.StatusOK, body)
}

func (n *fakeMeNews) create(w http.ResponseWriter, raw []byte) {
	var in map[string]any
	_ = json.Unmarshal(raw, &in)
	n.mu.Lock()
	it := meNewsItem{
		id:          strconv.Itoa(n.nextID),
		title:       meString(in["title"]),
		summary:     meString(in["summary"]),
		body:        meString(in["body"]),
		sourceURL:   meString(in["source_url"]),
		status:      "pending",
		publishedAt: "2026-09-30T04:00:00Z",
	}
	n.nextID++
	n.items = append([]meNewsItem{it}, n.items...)
	n.mu.Unlock()
	writeMeJSON(w, http.StatusCreated, it.payload())
}

func (n *fakeMeNews) getOne(w http.ResponseWriter, id string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, it := range n.items {
		if it.id != id {
			continue
		}
		writeMeJSON(w, http.StatusOK, it.payload())
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func (n *fakeMeNews) patch(w http.ResponseWriter, id string, raw []byte) {
	var in map[string]any
	_ = json.Unmarshal(raw, &in)
	n.mu.Lock()
	defer n.mu.Unlock()
	for i := range n.items {
		if n.items[i].id != id {
			continue
		}
		if status, ok := in["status"].(string); ok && status == "withdrawn" {
			n.items[i].status = "withdrawn"
		} else {
			if v, ok := in["title"].(string); ok {
				n.items[i].title = v
			}
			if v, ok := in["summary"].(string); ok {
				n.items[i].summary = v
			}
			if v, ok := in["body"].(string); ok {
				n.items[i].body = v
			}
			if v, ok := in["source_url"].(string); ok {
				n.items[i].sourceURL = v
			}
			if n.items[i].status == "published" {
				n.items[i].status = "pending"
			}
		}
		writeMeJSON(w, http.StatusOK, n.items[i].payload())
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func writeMeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func meString(v any) string {
	s, _ := v.(string)
	return s
}

func newNewsSubmissionFix(t *testing.T, checker gate.Checker) (*writeFix, *fakeMeNews) {
	t.Helper()
	f := newWriteFix(t, checker)
	f.alice(t)
	f.putSession(t, "sess-banned", w3UserBanned)
	up := newFakeMeNews()
	srv := httptest.NewServer(up.handler())
	t.Cleanup(srv.Close)
	cat := catalogclient.New(catalogclient.Config{BaseURL: srv.URL, HTTPClient: srv.Client()})
	f.NewsV1 = newsapiv1.New(newsclient.New(newsclient.Config{}), f.UserClient, "https://image.test.example").
		WithContent(&content.Converter{SiteBase: "https://www.kungal.com"}).
		WithSubmissions(cat, f.TrustCheck)
	f.Fiber = newFiber()
	f.setupRoutes()
	return f, up
}

func (f *writeFix) subCall(t *testing.T, method, raw, spec, session, idem string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	return f.docCall(t, method, raw, spec, session, idem, nil, payload)
}

func newsCreateBody() map[string]any {
	return map[string]any{"title": "Hello", "preview": "Lede", "content_markdown": "Body text", "source_url": "https://example.com/a"}
}

func wantNewsField(t *testing.T, body map[string]any, pointer, reason, detail string) {
	t.Helper()
	e := firstError(t, body)
	if e["pointer"] != pointer || e["reason"] != reason || e["detail"] != detail {
		t.Fatalf("field %+v, want %s %s %s", e, pointer, reason, detail)
	}
}

func TestV1NewsSubmissionCreate(t *testing.T) {
	f, up := newNewsSubmissionFix(t, nil)
	resp, body := f.subCall(t, http.MethodPost, "/api/v1/me/news-submissions", newsSubList, "sess-alice", keyUUID(1), newsCreateBody())
	if resp.StatusCode != http.StatusCreated || body["object"] != "news_submission" || body["state"] != "pending" || body["id"] != "4900" {
		t.Fatalf("create %d %+v", resp.StatusCode, body)
	}
	if resp.Header.Get("Location") != "/api/v1/me/news-submissions/4900" {
		t.Fatalf("Location %q", resp.Header.Get("Location"))
	}
	calls := up.calls()
	if len(calls) != 1 || calls[0].Method != http.MethodPost || calls[0].Path != "/v2/me/news" {
		t.Fatalf("upstream %+v", calls)
	}
	if calls[0].Header.Get("Idempotency-Key") != keyUUID(1) {
		t.Fatalf("Idempotency-Key %q", calls[0].Header.Get("Idempotency-Key"))
	}
	var got map[string]any
	if err := json.Unmarshal(calls[0].Raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["title"] != "Hello" || got["summary"] != "Lede" || got["body"] != "Body text" || got["source_url"] != "https://example.com/a" {
		t.Fatalf("upstream body %s", calls[0].Raw)
	}
	if _, ok := got["source"]; ok {
		t.Fatalf("upstream sent source %s", calls[0].Raw)
	}
	if _, ok := got["lane"]; ok {
		t.Fatalf("upstream sent lane %s", calls[0].Raw)
	}
}

func TestV1NewsSubmissionCreateRequiresIdempotencyKey(t *testing.T) {
	f, up := newNewsSubmissionFix(t, nil)
	resp, body := f.subCall(t, http.MethodPost, "/api/v1/me/news-submissions", newsSubList, "sess-alice", "", newsCreateBody())
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeInvalidParameter)
	if len(up.calls()) != 0 {
		t.Fatal("upstream was called")
	}
}

func TestV1NewsSubmissionCreateRequiresBodyOrSource(t *testing.T) {
	f, up := newNewsSubmissionFix(t, nil)
	resp, body := f.subCall(t, http.MethodPost, "/api/v1/me/news-submissions", newsSubList, "sess-alice", keyUUID(2), map[string]any{"title": "Hello", "preview": "Lede"})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	wantNewsField(t, body, "/content_markdown", problem.ReasonRequired, "send content_markdown or source_url")
	if len(up.calls()) != 0 {
		t.Fatal("upstream was called")
	}
}

func TestV1NewsSubmissionCreateRejectsBlankTitle(t *testing.T) {
	f, up := newNewsSubmissionFix(t, nil)
	resp, body := f.subCall(t, http.MethodPost, "/api/v1/me/news-submissions", newsSubList, "sess-alice", keyUUID(3), map[string]any{"title": "   ", "preview": "Lede", "content_markdown": "text"})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	wantNewsField(t, body, "/title", problem.ReasonRequired, "title is blank")
	if len(up.calls()) != 0 {
		t.Fatal("upstream was called")
	}
}

func TestV1NewsSubmissionCreateRejectedByTrust(t *testing.T) {
	f, up := newNewsSubmissionFix(t, denyChecker{})
	resp, body := f.subCall(t, http.MethodPost, "/api/v1/me/news-submissions", newsSubList, "sess-alice", keyUUID(4), newsCreateBody())
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeContentRejected)
	if len(up.calls()) != 0 {
		t.Fatal("upstream was called")
	}
}

func TestV1NewsSubmissionCreateBanned(t *testing.T) {
	f, up := newNewsSubmissionFix(t, nil)
	resp, body := f.subCall(t, http.MethodPost, "/api/v1/me/news-submissions", newsSubList, "sess-banned", keyUUID(5), newsCreateBody())
	wantCode(t, resp, body, http.StatusForbidden, problem.CodeAccountBanned)
	if len(up.calls()) != 0 {
		t.Fatal("upstream was called")
	}
}

func TestV1NewsSubmissionCreateQuota(t *testing.T) {
	f, up := newNewsSubmissionFix(t, nil)
	up.fail(http.MethodPost, http.StatusTooManyRequests, `{"code":"QUOTA_EXCEEDED","detail":"daily quota"}`, "3600")
	resp, body := f.subCall(t, http.MethodPost, "/api/v1/me/news-submissions", newsSubList, "sess-alice", keyUUID(6), newsCreateBody())
	wantCode(t, resp, body, http.StatusTooManyRequests, problem.CodeQuotaExceeded)
	if resp.Header.Get("Retry-After") != "3600" {
		t.Fatalf("Retry-After %q", resp.Header.Get("Retry-After"))
	}
}

func TestV1NewsSubmissionCreateRemapsUpstreamField(t *testing.T) {
	f, up := newNewsSubmissionFix(t, nil)
	up.fail(http.MethodPost, http.StatusUnprocessableEntity, `{"code":"VALIDATION_FAILED","errors":[{"pointer":"/body","reason":"TOO_LONG"}]}`, "")
	resp, body := f.subCall(t, http.MethodPost, "/api/v1/me/news-submissions", newsSubList, "sess-alice", keyUUID(7), newsCreateBody())
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if firstError(t, body)["pointer"] != "/content_markdown" {
		t.Fatalf("pointer %+v", body["errors"])
	}
}

func TestV1NewsSubmissionPatch(t *testing.T) {
	f, up := newNewsSubmissionFix(t, nil)
	resp, body := f.subCall(t, http.MethodPatch, "/api/v1/me/news-submissions/4802", newsSubItem, "sess-alice", "", map[string]any{"state": "withdrawn"})
	if resp.StatusCode != http.StatusOK || body["state"] != "withdrawn" {
		t.Fatalf("withdraw %d %+v", resp.StatusCode, body)
	}
	calls := up.calls()
	if len(calls) != 1 || calls[0].Method != http.MethodPatch || string(calls[0].Raw) != `{"status":"withdrawn"}` || calls[0].Header.Get("If-Match") != "*" {
		t.Fatalf("upstream %s if-match %q", calls[0].Raw, calls[0].Header.Get("If-Match"))
	}
	resp, body = f.subCall(t, http.MethodPatch, "/api/v1/me/news-submissions/4802", newsSubItem, "sess-alice", "", map[string]any{"state": "withdrawn", "title": "x"})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	wantNewsField(t, body, "/state", problem.ReasonNotAllowedValue, "withdraw on its own")
	resp, body = f.subCall(t, http.MethodPatch, "/api/v1/me/news-submissions/4802", newsSubItem, "sess-alice", "", map[string]any{})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	wantNewsField(t, body, "", problem.ReasonRequired, "send state=withdrawn or at least one field to edit")
	if len(up.calls()) != 1 {
		t.Fatalf("upstream calls %d", len(up.calls()))
	}
}

func TestV1NewsSubmissionPatchEdits(t *testing.T) {
	f, up := newNewsSubmissionFix(t, nil)
	resp, body := f.subCall(t, http.MethodPatch, "/api/v1/me/news-submissions/4802", newsSubItem, "sess-alice", "", map[string]any{"title": "  New  ", "preview": "New lede", "content_markdown": "New body"})
	if resp.StatusCode != http.StatusOK || body["title"] != "New" || body["preview"] != "New lede" || body["content_markdown"] != "New body" {
		t.Fatalf("edit %d %+v", resp.StatusCode, body)
	}
	calls := up.calls()
	if len(calls) != 2 || calls[1].Method != http.MethodPatch || calls[1].Header.Get("If-Match") != "" {
		t.Fatalf("upstream %+v", calls)
	}
	var got map[string]any
	if err := json.Unmarshal(calls[1].Raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got["title"] != "New" || got["summary"] != "New lede" || got["body"] != "New body" {
		t.Fatalf("upstream body %s", calls[1].Raw)
	}

	resp, body = f.subCall(t, http.MethodPatch, "/api/v1/me/news-submissions/4802", newsSubItem, "sess-alice", "", map[string]any{"title": "   "})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	wantNewsField(t, body, "/title", problem.ReasonRequired, "title is blank")

	up.fail(http.MethodPatch, http.StatusUnprocessableEntity, `{"code":"VALIDATION_FAILED","errors":[{"pointer":"/summary","reason":"TOO_LONG"}]}`, "")
	resp, body = f.subCall(t, http.MethodPatch, "/api/v1/me/news-submissions/4802", newsSubItem, "sess-alice", "", map[string]any{"preview": "Too long upstream"})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if firstError(t, body)["pointer"] != "/preview" {
		t.Fatalf("pointer %+v", body["errors"])
	}
}

func TestV1NewsSubmissionPatchRejectedByTrust(t *testing.T) {
	f, up := newNewsSubmissionFix(t, denyChecker{})
	resp, body := f.subCall(t, http.MethodPatch, "/api/v1/me/news-submissions/4802", newsSubItem, "sess-alice", "", map[string]any{"title": "Changed"})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeContentRejected)
	if len(up.calls()) != 0 {
		t.Fatal("upstream was called")
	}
}

func TestV1NewsSubmissionPatchKeepsSomethingToOpen(t *testing.T) {
	f, up := newNewsSubmissionFix(t, nil)
	resp, body := f.subCall(t, http.MethodPatch, "/api/v1/me/news-submissions/4802", newsSubItem, "sess-alice", "", map[string]any{"content_markdown": ""})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	wantNewsField(t, body, "/content_markdown", problem.ReasonRequired, "send content_markdown or source_url")
	calls := up.calls()
	if len(calls) != 1 || calls[0].Method != http.MethodGet || calls[0].Path != "/v2/me/news/4802" {
		t.Fatalf("upstream %+v", calls)
	}
}

func TestV1NewsSubmissionPatchConflict(t *testing.T) {
	f, up := newNewsSubmissionFix(t, nil)
	up.fail(http.MethodPatch, http.StatusConflict, `{"code":"INVALID_STATE_TRANSITION"}`, "")
	resp, body := f.subCall(t, http.MethodPatch, "/api/v1/me/news-submissions/4802", newsSubItem, "sess-alice", "", map[string]any{"title": "Kept"})
	wantCode(t, resp, body, http.StatusConflict, problem.CodeInvalidStateTransition)
}

func TestV1NewsSubmissionListPages(t *testing.T) {
	f, up := newNewsSubmissionFix(t, nil)
	q := url.Values{}
	q.Set("limit", "1")
	resp, body := f.subCall(t, http.MethodGet, "/api/v1/me/news-submissions?"+q.Encode(), newsSubList, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("page1 %d %+v", resp.StatusCode, body)
	}
	if got := adminItemIDs(body); len(got) != 1 || got[0] != "4802" {
		t.Fatalf("page1 ids %v", got)
	}
	next, _ := body["next_cursor"].(string)
	if !strings.HasPrefix(next, "cur_") || next == "4802" {
		t.Fatalf("next_cursor %q", next)
	}
	q.Set("cursor", next)
	resp, body = f.subCall(t, http.MethodGet, "/api/v1/me/news-submissions?"+q.Encode(), newsSubList, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("page2 %d %+v", resp.StatusCode, body)
	}
	if got := adminItemIDs(body); len(got) != 1 || got[0] != "4801" {
		t.Fatalf("page2 ids %v", got)
	}
	if _, ok := body["next_cursor"]; ok {
		t.Fatalf("last page cursor %v", body["next_cursor"])
	}
	calls := up.calls()
	if len(calls) != 2 || calls[0].Query.Get("cursor") != "" || calls[0].Query.Get("limit") != "1" || calls[1].Query.Get("cursor") != "4802" {
		t.Fatalf("upstream cursors %q %q", calls[0].Query.Get("cursor"), calls[1].Query.Get("cursor"))
	}
}

func TestV1NewsSubmissionAnonymous(t *testing.T) {
	f, up := newNewsSubmissionFix(t, nil)
	f.anonNews(t, http.MethodGet, "/api/v1/me/news-submissions", newsSubList, "", nil)
	f.anonNews(t, http.MethodGet, "/api/v1/me/news-submissions/4802", newsSubItem, "", nil)
	f.anonNews(t, http.MethodPost, "/api/v1/me/news-submissions", newsSubList, keyUUID(17), newsCreateBody())
	f.anonNews(t, http.MethodPatch, "/api/v1/me/news-submissions/4802", newsSubItem, "", map[string]any{"title": "x"})
	if len(up.calls()) != 0 {
		t.Fatal("upstream was called")
	}
}

func (f *writeFix) anonNews(t *testing.T, method, raw, spec, idem string, payload any) {
	t.Helper()
	resp, body := f.subCall(t, method, raw, spec, "", idem, payload)
	wantCode(t, resp, body, http.StatusUnauthorized, problem.CodeMissingCredential)
}
