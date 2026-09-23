package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
)

type libraryRecorder struct {
	mu    sync.Mutex
	path  string
	query map[string][]string
}

func (r *libraryRecorder) get(key string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.query == nil {
		return ""
	}
	v := r.query[key]
	if len(v) == 0 {
		return ""
	}
	return v[0]
}

// /galgame is the resource list and never reaches catalog; only library=true does.
func TestCatalogLibrary_OnlyTheLibraryFlagLeavesTheForum(t *testing.T) {
	rec := &libraryRecorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		rec.mu.Lock()
		rec.path = req.URL.Path
		rec.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"total":0}`))
	}))
	t.Cleanup(srv.Close)

	svc := NewGalgameService(nil, nil, nil, nil, nil, nil, nil,
		client.New(srv.URL, "nm_test_key", ""), nil, nil, nil)
	func() {
		// The local branch dereferences a repository this unit test has no
		// database for. Panicking there IS the assertion: it can only be reached
		// by not taking the catalog branch.
		defer func() { _ = recover() }()
		_, _ = svc.GetList(context.Background(), &dto.GalgameListRequest{Page: 1, Limit: 24}, true)
	}()
	if rec.path != "" {
		t.Errorf("plain /galgame called catalog at %q", rec.path)
	}
}

func TestCatalogLibrary_DoesNotSendClaimState(t *testing.T) {
	rec := &libraryRecorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		rec.mu.Lock()
		rec.path = req.URL.Path
		rec.query = req.URL.Query()
		rec.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"total":0}`))
	}))
	t.Cleanup(srv.Close)

	svc := NewGalgameService(nil, nil, nil, nil, nil, nil, nil,
		client.New(srv.URL, "nm_test_key", ""), nil, nil, nil)
	page, appErr := svc.GetList(context.Background(), &dto.GalgameListRequest{
		Page: 1, Limit: 24, SortField: "popularity", SortOrder: "desc", Library: true,
	}, true)
	if appErr != nil {
		t.Fatalf("GetList: %v", appErr)
	}
	if page.Total != 0 {
		t.Errorf("total = %d, want 0 from empty upstream", page.Total)
	}
	if rec.path != "/v2/catalog/works" {
		t.Errorf("path = %q, want /v2/catalog/works", rec.path)
	}
	if got := rec.get("claim_state"); got != "" {
		t.Errorf("claim_state = %q, want it absent", got)
	}
	if got := rec.get("sort"); got != "popularity" {
		t.Errorf("sort = %q, want popularity", got)
	}
	if got := rec.get("nsfw"); got != "true" {
		t.Errorf("nsfw = %q, want true — the SFW library still spans the whole age range", got)
	}
	if got := rec.get("content_limit"); got != "sfw" {
		t.Errorf("content_limit = %q, want sfw", got)
	}
}
