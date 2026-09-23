package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/pkg/communityclient"
)

type fakeCommentMap struct {
	row *model.GalgameCommentCommunityMap
}

func (f fakeCommentMap) FindMapByLegacyID(int) (*model.GalgameCommentCommunityMap, error) {
	return f.row, nil
}

func TestGalgameCommentEnforcerTombstoneMapHit(t *testing.T) {
	var resolveHit, deleteHit bool
	var deletePath, deleteQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/posts/resolve":
			resolveHit = true
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0, "data": map[string]any{
					"posts": []any{map[string]any{"post": map[string]any{"id": 900, "author_id": 77}}},
				},
			})
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/posts/"):
			deleteHit = true
			deletePath = r.URL.Path
			deleteQuery = r.URL.RawQuery
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	cli := communityclient.New(communityclient.Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
	enf := NewGalgameCommentEnforcer(cli, fakeCommentMap{row: &model.GalgameCommentCommunityMap{
		OldCommentID: 5, ThreadID: 7, PostID: 900, WorkID: 42,
	}})

	if err := enf.Tombstone(context.Background(), 5); err != nil {
		t.Fatalf("Tombstone: %v", err)
	}
	if !resolveHit {
		t.Error("expected ResolvePosts to be called for the author lookup")
	}
	if !deleteHit {
		t.Fatal("expected DeletePost (tombstone) to be called")
	}
	if deletePath != "/posts/900" {
		t.Errorf("delete path = %q, want /posts/900", deletePath)
	}
	if !strings.Contains(deleteQuery, "as_moderator=true") {
		t.Errorf("delete query = %q, want as_moderator=true", deleteQuery)
	}
	if !strings.Contains(deleteQuery, "author_id=77") {
		t.Errorf("delete query = %q, want author_id=77 (resolved)", deleteQuery)
	}
}

func TestGalgameCommentEnforcerTombstoneMapMiss(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no community request expected on a map miss, got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	cli := communityclient.New(communityclient.Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
	enf := NewGalgameCommentEnforcer(cli, fakeCommentMap{row: nil})

	if err := enf.Tombstone(context.Background(), 999); err != nil {
		t.Fatalf("map-miss Tombstone must succeed as a no-op, got %v", err)
	}
}

func TestGalgameCommentEnforcerAuthorID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{
				"posts": []any{map[string]any{"post": map[string]any{"id": 900, "author_id": 77}}},
			},
		})
	}))
	defer srv.Close()

	cli := communityclient.New(communityclient.Config{BaseURL: srv.URL, ClientID: "cid", ClientSecret: "sec"})
	hit := NewGalgameCommentEnforcer(cli, fakeCommentMap{row: &model.GalgameCommentCommunityMap{PostID: 900}})
	if id, err := hit.AuthorID(context.Background(), 5); err != nil || id != 77 {
		t.Fatalf("AuthorID map-hit = (%d, %v), want (77, nil)", id, err)
	}

	miss := NewGalgameCommentEnforcer(cli, fakeCommentMap{row: nil})
	if id, err := miss.AuthorID(context.Background(), 5); err != nil || id != 0 {
		t.Fatalf("AuthorID map-miss = (%d, %v), want (0, nil)", id, err)
	}
}
