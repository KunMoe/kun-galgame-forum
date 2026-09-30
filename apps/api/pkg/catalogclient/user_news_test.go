package catalogclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

const sampleNewsSubmission = `{"object":"news_submission","id":"4700","source":{"object":"news_source","name":"community","display_name":"NextMoe 用户投稿"},"lane":"news","status":"pending","title":"T","summary":"S","source_url":"https://example.com/a","banner":null,"published_at":"2026-09-30T02:00:00Z","work_ids":[],"body":"md","submitter_uid":"12345"}`

func assertSampleSubmission(t *testing.T, got *NewsSubmission) {
	t.Helper()
	if got == nil {
		t.Fatal("nil submission")
	}
	wantAt := time.Date(2026, 9, 30, 2, 0, 0, 0, time.UTC)
	if got.ID != 4700 || got.SourceKey != "community" || got.Lane != "news" || got.Status != "pending" ||
		got.Title != "T" || got.Summary != "S" || got.Body != "md" || got.SourceURL != "https://example.com/a" ||
		got.SubmitterUID != 12345 || !got.PublishedAt.Equal(wantAt) {
		t.Fatalf("submission = %+v", got)
	}
}

func TestCreateMyNews_SendsOnlyTextFieldsAndForwardsKey(t *testing.T) {
	var gotPath, gotAuth, gotMethod, gotKey, gotMatch string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth, gotMethod = r.URL.Path, r.Header.Get("Authorization"), r.Method
		gotKey, gotMatch = r.Header.Get("Idempotency-Key"), r.Header.Get("If-Match")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(sampleNewsSubmission))
	}))
	defer srv.Close()

	got, err := New(Config{BaseURL: srv.URL}).CreateMyNews(context.Background(), "user-jwt",
		NewsSubmissionWrite{Title: "T", Summary: "S", Body: "md", SourceURL: "https://example.com/a"}, "idem-1")
	if err != nil {
		t.Fatalf("CreateMyNews: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/v2/me/news" {
		t.Errorf("path = %s, want /v2/me/news", gotPath)
	}
	if gotAuth != "Bearer user-jwt" {
		t.Errorf("auth = %q, want the user's bearer", gotAuth)
	}
	if gotKey != "idem-1" {
		t.Errorf("Idempotency-Key = %q", gotKey)
	}
	if gotMatch != "" {
		t.Errorf("If-Match = %q, create must not send one", gotMatch)
	}
	for _, k := range []string{"title", "summary", "body", "source_url"} {
		if _, ok := gotBody[k]; !ok {
			t.Errorf("body missing %s: %v", k, gotBody)
		}
	}
	if gotBody["title"] != "T" || gotBody["summary"] != "S" || gotBody["body"] != "md" || gotBody["source_url"] != "https://example.com/a" {
		t.Errorf("body = %v", gotBody)
	}
	for _, k := range []string{"source", "lane", "published_at", "banner_hash", "work_ids", "status"} {
		if _, ok := gotBody[k]; ok {
			t.Errorf("body must not send %s, got %v", k, gotBody)
		}
	}
	assertSampleSubmission(t, got)
}

func TestCreateMyNews_OmitsEmptyOptionals(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"object":"news_submission","id":"4700","source":{"name":"community"},"lane":"news","status":"pending","title":"T","summary":"S","source_url":"","published_at":"2026-09-30T02:00:00Z","body":"","submitter_uid":null}`))
	}))
	defer srv.Close()

	got, err := New(Config{BaseURL: srv.URL}).CreateMyNews(context.Background(), "user-jwt",
		NewsSubmissionWrite{Title: "T", Summary: "S"}, "idem-1")
	if err != nil {
		t.Fatalf("CreateMyNews: %v", err)
	}
	if _, ok := gotBody["body"]; ok {
		t.Errorf("empty body must be omitted, got %v", gotBody)
	}
	if _, ok := gotBody["source_url"]; ok {
		t.Errorf("empty source_url must be omitted, got %v", gotBody)
	}
	if gotBody["title"] != "T" || gotBody["summary"] != "S" {
		t.Errorf("body = %v", gotBody)
	}
	if got.SubmitterUID != 0 || got.SourceURL != "" || got.Body != "" {
		t.Errorf("record = %+v", got)
	}
}

func TestCreateMyNews_QuotaExceeded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.Header().Set("Retry-After", "3600")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"code":"QUOTA_EXCEEDED","detail":"daily quota"}`))
	}))
	defer srv.Close()

	_, err := New(Config{BaseURL: srv.URL}).CreateMyNews(context.Background(), "user-jwt",
		NewsSubmissionWrite{Title: "T", Summary: "S"}, "idem-1")
	var api *UserAPIError
	if !errors.As(err, &api) || api.Status != http.StatusTooManyRequests || api.ProblemCode != "QUOTA_EXCEEDED" || api.RetryAfter != "3600" {
		t.Fatalf("err = %#v, want 429 QUOTA_EXCEEDED with Retry-After", err)
	}
}

func TestListMyNews_PassesCursorAndReturnsNext(t *testing.T) {
	var gotQuery url.Values
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		gotPath, gotMethod = r.URL.Path, r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","items":[` + sampleNewsSubmission + `],"next_cursor":"4700"}`))
	}))
	defer srv.Close()

	items, cursor, err := New(Config{BaseURL: srv.URL}).ListMyNews(context.Background(), "user-jwt", "4699", 20)
	if err != nil {
		t.Fatalf("ListMyNews: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %s, want GET", gotMethod)
	}
	if gotPath != "/v2/me/news" {
		t.Errorf("path = %s, want /v2/me/news", gotPath)
	}
	if gotQuery.Get("cursor") != "4699" {
		t.Errorf("cursor param = %q", gotQuery.Get("cursor"))
	}
	if gotQuery.Get("limit") != "20" {
		t.Errorf("limit = %q", gotQuery.Get("limit"))
	}
	if len(gotQuery) != 2 {
		t.Errorf("query = %v, want only cursor and limit", gotQuery)
	}
	if len(items) != 1 {
		t.Fatalf("items = %+v", items)
	}
	assertSampleSubmission(t, &items[0])
	if cursor != "4700" {
		t.Errorf("cursor = %q", cursor)
	}
}

func TestListMyNews_AbsentNextCursorIsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","items":[` + sampleNewsSubmission + `]}`))
	}))
	defer srv.Close()

	items, cursor, err := New(Config{BaseURL: srv.URL}).ListMyNews(context.Background(), "user-jwt", "", 20)
	if err != nil {
		t.Fatalf("ListMyNews: %v", err)
	}
	if len(items) != 1 || cursor != "" {
		t.Fatalf("items = %d, cursor = %q, want one row and an empty next cursor", len(items), cursor)
	}
}

func TestGetMyNews_ReadsOne(t *testing.T) {
	var gotPath, gotAuth, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth, gotMethod = r.URL.Path, r.Header.Get("Authorization"), r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleNewsSubmission))
	}))
	defer srv.Close()

	got, err := New(Config{BaseURL: srv.URL}).GetMyNews(context.Background(), "user-jwt", 4700)
	if err != nil {
		t.Fatalf("GetMyNews: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %s, want GET", gotMethod)
	}
	if gotPath != "/v2/me/news/4700" {
		t.Errorf("path = %s", gotPath)
	}
	if gotAuth != "Bearer user-jwt" {
		t.Errorf("auth = %q", gotAuth)
	}
	assertSampleSubmission(t, got)
}

func TestPatchMyNews_EditSendsOnlySetMembers(t *testing.T) {
	var gotPath, gotMethod, gotMatch, gotKey string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		gotMatch, gotKey = r.Header.Get("If-Match"), r.Header.Get("Idempotency-Key")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleNewsSubmission))
	}))
	defer srv.Close()

	empty := ""
	title := "Next"
	got, err := New(Config{BaseURL: srv.URL}).PatchMyNews(context.Background(), "user-jwt", 4700,
		NewsSubmissionPatch{Title: &title, Body: &empty})
	if err != nil {
		t.Fatalf("PatchMyNews: %v", err)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("method = %s, want PATCH", gotMethod)
	}
	if gotPath != "/v2/me/news/4700" {
		t.Errorf("path = %s", gotPath)
	}
	if gotMatch != "" {
		t.Errorf("If-Match = %q, an edit must not send one", gotMatch)
	}
	if gotKey != "" {
		t.Errorf("Idempotency-Key = %q, an edit must not send one", gotKey)
	}
	if gotBody["title"] != "Next" {
		t.Errorf("body = %v", gotBody)
	}
	body, ok := gotBody["body"]
	if !ok || body != "" {
		t.Errorf("empty body must be sent to clear, got %v", gotBody)
	}
	for _, k := range []string{"status", "summary", "source_url"} {
		if _, present := gotBody[k]; present {
			t.Errorf("unset %s must be omitted, got %v", k, gotBody)
		}
	}
	assertSampleSubmission(t, got)
}

func TestPatchMyNews_WithdrawalIsStatusOnlyWithIfMatchStar(t *testing.T) {
	var gotMatch string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMatch = r.Header.Get("If-Match")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"news_submission","id":"4700","source":{"name":"community"},"lane":"news","status":"withdrawn","title":"T","summary":"S","source_url":"","published_at":"2026-09-30T02:00:00Z","body":"md","submitter_uid":"12345"}`))
	}))
	defer srv.Close()

	title := "ignored"
	got, err := New(Config{BaseURL: srv.URL}).PatchMyNews(context.Background(), "user-jwt", 4700,
		NewsSubmissionPatch{Withdraw: true, Title: &title})
	if err != nil {
		t.Fatalf("PatchMyNews: %v", err)
	}
	if gotMatch != "*" {
		t.Errorf("If-Match = %q, want *", gotMatch)
	}
	if len(gotBody) != 1 || gotBody["status"] != "withdrawn" {
		t.Errorf("body = %v, want exactly status withdrawn", gotBody)
	}
	if got.Status != "withdrawn" {
		t.Errorf("record = %+v", got)
	}
}
