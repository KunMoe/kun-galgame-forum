package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/catalogclient"
)

type coverPlaneRecorder struct {
	mu         sync.Mutex
	path       string
	query      string
	auth       string
	staleToken bool
}

func (r *coverPlaneRecorder) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		r.mu.Lock()
		r.path, r.query, r.auth = req.URL.Path, req.URL.RawQuery, req.Header.Get("Authorization")
		stale := r.staleToken
		r.mu.Unlock()

		if stale && req.Header.Get("Authorization") == "Bearer pre-scope-jwt" {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"code":"SCOPE_REQUIRED","title":"missing required scope: catalog:edit"}`))
			return
		}
		if req.URL.Path == "/v2/catalog/works/1000" {
			_, _ = w.Write([]byte(`{"id":"1000","covers":[{"id":"88","hash":"abc","vote_count":3}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"object":"list","items":[` +
			`{"id":"88","hash":"abc","vote_count":3,"voted":true}]}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func (r *coverPlaneRecorder) service(t *testing.T) *GalgameService {
	srv := r.server(t)
	return &GalgameService{
		galgameClient: client.New(srv.URL, "nm_test_key", ""),
		catalog:       catalogclient.New(catalogclient.Config{BaseURL: srv.URL, AppKey: "nmk_test"}),
	}
}

func TestCoverVotesReadAsTheViewerWhenTheyHaveAToken(t *testing.T) {
	rec := &coverPlaneRecorder{}
	covers := []dto.GalgameCover{{ImageHash: "abc"}}

	rec.service(t).hydrateCoverVotes(t.Context(), 1000, "user-jwt", covers)

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.path != "/v2/catalog/works/1000/covers" {
		t.Errorf("covers read hit %q, want the user plane's covers face", rec.path)
	}
	if rec.auth != "Bearer user-jwt" {
		t.Errorf("auth = %q, want the viewer's own bearer", rec.auth)
	}
	if strings.Contains(rec.query, "uid=") {
		t.Errorf("the viewer is the token, not a query parameter: %q", rec.query)
	}
	if covers[0].ID != 88 || covers[0].VoteCount != 3 || !covers[0].Voted {
		t.Errorf("hydration lost the tally: %+v", covers[0])
	}
}

func TestCoverVotesFallBackToPublicCountsOnAStaleToken(t *testing.T) {
	rec := &coverPlaneRecorder{staleToken: true}
	covers := []dto.GalgameCover{{ImageHash: "abc"}}

	rec.service(t).hydrateCoverVotes(t.Context(), 1000, "pre-scope-jwt", covers)

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.path != "/v2/catalog/works/1000" {
		t.Errorf("stale-token covers read landed on %q, want the public v2 work", rec.path)
	}
	if rec.auth != "Bearer nmk_test" {
		t.Errorf("auth = %q, want the application key after the fallback", rec.auth)
	}
	if covers[0].ID != 88 || covers[0].VoteCount != 3 {
		t.Errorf("fallback lost the tally: %+v", covers[0])
	}
}

func TestCoverVotesStayAnonymousWithoutAToken(t *testing.T) {
	rec := &coverPlaneRecorder{}
	covers := []dto.GalgameCover{{ImageHash: "abc"}}

	rec.service(t).hydrateCoverVotes(t.Context(), 1000, "", covers)

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if rec.path != "/v2/catalog/works/1000" {
		t.Errorf("anonymous covers read hit %q, want the public v2 work", rec.path)
	}
	if rec.auth != "Bearer nmk_test" {
		t.Errorf("auth = %q, want the application key", rec.auth)
	}
	if strings.Contains(rec.query, "uid") || strings.Contains(rec.query, "user") {
		t.Errorf("an anonymous read sent an identity: %q", rec.query)
	}
	if !strings.Contains(rec.query, "include=covers") || !strings.Contains(rec.query, "nsfw=true") {
		t.Errorf("an anonymous read sent %q, want include=covers with the population gate open", rec.query)
	}
	if covers[0].ID != 88 || covers[0].VoteCount != 3 {
		t.Errorf("hydration lost the tally: %+v", covers[0])
	}
}
