package trustclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConfigured(t *testing.T) {
	if New(Config{BaseURL: "http://x", ClientID: "", ClientSecret: ""}).Configured() {
		t.Fatal("expected not configured without credentials")
	}
	if New(Config{BaseURL: "", ClientID: "a", ClientSecret: "b"}).Configured() {
		t.Fatal("expected not configured without base URL")
	}
	if !New(Config{BaseURL: "http://x", ClientID: "a", ClientSecret: "b"}).Configured() {
		t.Fatal("expected configured with base URL + credentials")
	}
}

func TestSubmitReportNotConfigured(t *testing.T) {
	_, err := New(Config{}).SubmitReport(context.Background(), ReportRequest{})
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("want ErrNotConfigured, got %v", err)
	}
}

func TestSubmitReportSuccess(t *testing.T) {
	var gotAuth, gotPath string
	var gotBody ReportRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"report_id":42,"review_item_id":7}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
	res, err := c.SubmitReport(context.Background(), ReportRequest{
		SubjectKind: "forum_topic", SubjectID: "1207", ReasonKey: "spam", ReporterID: 99,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ReportID != 42 || res.ReviewItemID != 7 {
		t.Fatalf("bad result: %+v", res)
	}
	if gotAuth == "" || gotAuth[:6] != "Basic " {
		t.Fatalf("expected Basic auth, got %q", gotAuth)
	}
	if gotPath != "/api/v1/trust/reports" {
		t.Fatalf("bad path %q", gotPath)
	}
	if gotBody.SubjectKind != "forum_topic" || gotBody.SubjectID != "1207" || gotBody.ReporterID != 99 {
		t.Fatalf("bad forwarded body: %+v", gotBody)
	}
}

func TestListReportReasons(t *testing.T) {
	var gotAuth, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"reasons":[{"id":1,"key":"spam","name_cn":"垃圾信息","severity":1,"is_deprecated":false}]}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
	reasons, err := c.ListReportReasons(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reasons) != 1 || reasons[0].Key != "spam" || reasons[0].NameCN != "垃圾信息" {
		t.Fatalf("bad reasons: %+v", reasons)
	}
	if gotPath != "/api/v1/trust/report-reasons" {
		t.Fatalf("bad path %q", gotPath)
	}
	if gotAuth[:6] != "Basic " {
		t.Fatalf("expected Basic auth, got %q", gotAuth)
	}

	if _, err := New(Config{}).ListReportReasons(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("want ErrNotConfigured, got %v", err)
	}
}

func TestSubmitReportErrorMapping(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   error
	}{
		{http.StatusTooManyRequests, `{"code":10,"message":"rate"}`, ErrRateLimited},
		{http.StatusUnprocessableEntity, `{"code":7,"message":"bad kind"}`, ErrValidation},
		{http.StatusForbidden, `{"code":5,"message":"no site"}`, ErrForbidden},
		{http.StatusUnauthorized, `{"code":10001,"message":"bad creds"}`, ErrUnauthorized},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(tc.body))
		}))
		c := New(Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
		_, err := c.SubmitReport(context.Background(), ReportRequest{SubjectKind: "x", SubjectID: "1", ReasonKey: "spam", ReporterID: 1})
		if !errors.Is(err, tc.want) {
			t.Fatalf("status %d: want %v, got %v", tc.status, tc.want, err)
		}
		srv.Close()
	}
}

// The trust service answers 422 to author_id 0 and fails the whole report.
func TestSubmitReportAuthorOnlyWhenKnown(t *testing.T) {
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		bodies = append(bodies, body)
		_, _ = w.Write([]byte(`{"code":0,"data":{"report_id":1}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
	author := int64(55)
	for _, req := range []ReportRequest{
		{SubjectKind: "forum_reply", SubjectID: "8", ReasonKey: "spam", ReporterID: 1},
		{SubjectKind: "forum_reply", SubjectID: "8", ReasonKey: "spam", ReporterID: 1, AuthorID: &author},
	} {
		if _, err := c.SubmitReport(context.Background(), req); err != nil {
			t.Fatal(err)
		}
	}
	if _, sent := bodies[0]["author_id"]; sent {
		t.Errorf("an unknown author is omitted: %v", bodies[0])
	}
	if bodies[1]["author_id"] != float64(55) {
		t.Errorf("a known author is sent: %v", bodies[1])
	}
}

func TestListReviewItemsAuthorFilter(t *testing.T) {
	var queries []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"code":0,"data":{"items":[{"id":3,"subject_author_id":55}],"total":1}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
	page, err := c.ListReviewItems(context.Background(), "tok", ReviewQuery{Site: "kungal", SubjectAuthorID: 55, Page: 1, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if page.Items[0].SubjectAuthorID == nil || *page.Items[0].SubjectAuthorID != 55 {
		t.Errorf("subject_author_id decoded: %+v", page.Items[0])
	}
	if _, err := c.ListReviewItems(context.Background(), "tok", ReviewQuery{Site: "kungal", Page: 1, Limit: 1}); err != nil {
		t.Fatal(err)
	}
	if queries[0] != "limit=1&page=1&site=kungal&subject_author_id=55" || queries[1] != "limit=1&page=1&site=kungal" {
		t.Errorf("queries = %q", queries)
	}
}
