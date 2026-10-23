package trustclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckNotConfigured(t *testing.T) {
	if _, err := New(Config{}).Check(context.Background(), CheckRequest{Text: "hi"}); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("want ErrNotConfigured, got %v", err)
	}
}

func TestScanNotConfigured(t *testing.T) {
	if _, err := New(Config{}).Scan(context.Background(), ScanRequest{SubjectKind: "forum_topic", SubjectID: "1"}); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("want ErrNotConfigured, got %v", err)
	}
}

func TestCheckWire(t *testing.T) {
	var gotAuth, gotPath, gotCT string
	var gotBody CheckRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotPath, gotCT = r.Header.Get("Authorization"), r.URL.Path, r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"decision":"deny","matched":["坏词"]}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
	author := int64(99)
	res, err := c.Check(context.Background(), CheckRequest{Text: "标题\n\n正文", AuthorID: &author})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Decision != "deny" || len(res.Matched) != 1 || res.Matched[0] != "坏词" {
		t.Fatalf("bad result: %+v", res)
	}
	if gotPath != "/api/v1/trust/check" {
		t.Fatalf("bad path %q", gotPath)
	}
	if len(gotAuth) < 6 || gotAuth[:6] != "Basic " {
		t.Fatalf("expected Basic auth, got %q", gotAuth)
	}
	if gotCT != "application/json" {
		t.Fatalf("bad content-type %q", gotCT)
	}
	if gotBody.Text != "标题\n\n正文" || gotBody.AuthorID == nil || *gotBody.AuthorID != 99 {
		t.Fatalf("bad forwarded body: %+v", gotBody)
	}
}

func TestScanWire(t *testing.T) {
	var gotPath string
	var gotBody ScanRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"scan_id":456,"truncated":true}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
	author := int64(7)
	res, err := c.Scan(context.Background(), ScanRequest{SubjectKind: "forum_reply", SubjectID: "8841", Text: "正文", AuthorID: &author})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ScanID != 456 || !res.Truncated {
		t.Fatalf("bad result: %+v", res)
	}
	if gotPath != "/api/v1/trust/scan" {
		t.Fatalf("bad path %q", gotPath)
	}
	if gotBody.SubjectKind != "forum_reply" || gotBody.SubjectID != "8841" || gotBody.Text != "正文" {
		t.Fatalf("bad forwarded body: %+v", gotBody)
	}
}

func TestModerationEnvelopeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"code":7,"message":"unregistered kind"}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
	if _, err := c.Scan(context.Background(), ScanRequest{SubjectKind: "nope", SubjectID: "1", Text: "x"}); err == nil {
		t.Fatal("expected error on non-zero envelope code")
	}
	if _, err := c.Check(context.Background(), CheckRequest{Text: "x"}); err == nil {
		t.Fatal("expected error on non-2xx status")
	}
}

func TestScanCheckSkippedWire(t *testing.T) {
	for _, skipped := range []bool{false, true} {
		var raw map[string]any
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&raw)
			_, _ = w.Write([]byte(`{"code":0,"message":"成功","data":{"scan_id":1}}`))
		}))
		c := New(Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
		if _, err := c.Scan(context.Background(), ScanRequest{SubjectKind: "forum_reply", SubjectID: "1", Text: "正文", CheckSkipped: skipped}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		srv.Close()
		got, present := raw["check_skipped"]
		if skipped && got != true {
			t.Fatalf("check_skipped = %v, want true", got)
		}
		if !skipped && present {
			t.Fatalf("an answered check sent check_skipped=%v; trust would queue it twice", got)
		}
	}
}
