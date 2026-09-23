package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"kun-galgame-api/internal/galgame/client"
)

type worksQueryRecorder struct {
	mu    sync.Mutex
	path  string
	query url.Values
}

func (r *worksQueryRecorder) client(t *testing.T) *client.GalgameClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.mu.Lock()
		r.path = req.URL.Path
		r.query = req.URL.Query()
		r.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"total":0}`))
	}))
	t.Cleanup(srv.Close)
	return client.New(srv.URL, "nm_test_key", "")
}

func (r *worksQueryRecorder) get(key string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.query.Get(key)
}

func TestQuizPicker_OffersTheCatalogWithoutAClaimGate(t *testing.T) {
	rec := &worksQueryRecorder{}
	svc := NewQuizService(nil, rec.client(t), nil, nil, nil)

	if got := svc.SearchGalgameOptions(context.Background(), "恋爱", true); len(got) != 0 {
		t.Fatalf("stubbed empty upstream returned %d options", len(got))
	}
	if rec.path != "/v2/catalog/works" {
		t.Errorf("path = %q, want /v2/catalog/works", rec.path)
	}
	if got := rec.get("claim_state"); got != "" {
		t.Errorf("claim_state = %q, want it absent — the picker is the catalog", got)
	}
	if got := rec.get("claimed"); got != "" {
		t.Errorf("claimed = %q, want it absent", got)
	}
}
