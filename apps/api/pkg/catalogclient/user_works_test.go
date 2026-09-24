package catalogclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// meWorksFace answers GET /v2/me/works the way infra fac31d2a (spec 2.25.0)
// does: 1–100 ids, one item per distinct id in request order, unknown works
// answered with no folders, 400 TOO_MANY_IDS above the cap.
type meWorksFace struct {
	mu      sync.Mutex
	queries []string
	folders map[int64][]int64
	minutes map[int64]int
	states  map[int64][2]string
}

func (f *meWorksFace) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/me/works" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		raw := r.URL.Query().Get("work_ids")
		f.mu.Lock()
		f.queries = append(f.queries, raw)
		f.mu.Unlock()
		parts := strings.Split(raw, ",")
		if raw == "" {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":"INVALID_PARAMETER","status":400}`))
			return
		}
		if len(parts) > MyWorksMax {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":"TOO_MANY_IDS","status":400}`))
			return
		}
		items := []map[string]any{}
		seen := map[int64]bool{}
		for _, p := range parts {
			id, err := strconv.ParseInt(p, 10, 64)
			if err != nil || id <= 0 {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"code":"INVALID_PARAMETER","status":400}`))
				return
			}
			if seen[id] {
				continue
			}
			seen[id] = true
			folderIDs := []string{}
			for _, fid := range f.folders[id] {
				folderIDs = append(folderIDs, strconv.FormatInt(fid, 10))
			}
			item := map[string]any{"object": "user_work", "work_id": strconv.FormatInt(id, 10), "folder_ids": folderIDs, "playtime": nil, "work_state": nil}
			if m, ok := f.minutes[id]; ok {
				item["playtime"] = map[string]any{"object": "playtime", "work_id": strconv.FormatInt(id, 10), "minutes": m}
			}
			if st, ok := f.states[id]; ok {
				var completion any
				if st[1] != "" {
					completion = st[1]
				}
				item["work_state"] = map[string]any{"object": "work_state", "work_id": strconv.FormatInt(id, 10), "state": st[0], "completion": completion,
					"created_at": "2026-09-24T00:00:00Z", "updated_at": "2026-09-24T00:00:00Z"}
			}
			items = append(items, item)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "items": items, "next_cursor": nil})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestMyWorksDecodesAndFoldsDuplicates(t *testing.T) {
	face := &meWorksFace{
		folders: map[int64][]int64{7: {3, 9}},
		minutes: map[int64]int{7: 95},
		states:  map[int64][2]string{7: {"done", "main"}, 8: {"wish", ""}},
	}
	c := New(Config{BaseURL: face.server(t).URL, AppKey: "k"})
	got, err := c.MyWorks(context.Background(), "user-jwt", []int64{8, 7, 8, 404})
	if err != nil {
		t.Fatalf("MyWorks: %v", err)
	}
	if len(face.queries) != 1 || face.queries[0] != "8,7,404" {
		t.Fatalf("want one call with the duplicate dropped, got %q", face.queries)
	}
	if len(got) != 3 || got[0].WorkID != 8 || got[1].WorkID != 7 || got[2].WorkID != 404 {
		t.Fatalf("want every id once in request order, got %+v", got)
	}
	if fmt.Sprint(got[1].FolderIDs) != "[3 9]" || got[1].Playtime == nil || got[1].Playtime.Minutes != 95 ||
		got[1].WorkState == nil || got[1].WorkState.State != "done" || *got[1].WorkState.Completion != "main" {
		t.Errorf("work 7 %+v", got[1])
	}
	if got[0].WorkState == nil || got[0].WorkState.Completion != nil || got[0].Playtime != nil || len(got[0].FolderIDs) != 0 {
		t.Errorf("work 8 %+v", got[0])
	}
	if got[2].FolderIDs == nil || len(got[2].FolderIDs) != 0 || got[2].WorkState != nil {
		t.Errorf("an unknown work is answered empty, not left out: %+v", got[2])
	}
}

func TestMyWorksChunksAtTheCap(t *testing.T) {
	face := &meWorksFace{}
	c := New(Config{BaseURL: face.server(t).URL, AppKey: "k"})
	ids := make([]int64, MyWorksMax+1)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	got, err := c.MyWorks(context.Background(), "user-jwt", ids)
	if err != nil {
		t.Fatalf("MyWorks over the cap must chunk, not answer TOO_MANY_IDS: %v", err)
	}
	if len(face.queries) != 2 || len(got) != MyWorksMax+1 {
		t.Fatalf("want 2 calls and %d items, got %d calls and %d items", MyWorksMax+1, len(face.queries), len(got))
	}
}

func TestMyWorksMapsScopeAndThrottle(t *testing.T) {
	for _, c := range []struct {
		status int
		body   string
		check  func(error) bool
	}{
		{http.StatusForbidden, `{"code":"SCOPE_REQUIRED","status":403}`, func(err error) bool { return errors.Is(err, ErrInsufficientScope) }},
		{http.StatusTooManyRequests, `{"code":"RATE_LIMITED","status":429}`, func(err error) bool {
			var api *UserAPIError
			return errors.As(err, &api) && api.Throttled()
		}},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(c.status)
			_, _ = w.Write([]byte(c.body))
		}))
		cl := New(Config{BaseURL: srv.URL, AppKey: "k"})
		_, err := cl.MyWorks(context.Background(), "user-jwt", []int64{1})
		srv.Close()
		if !c.check(err) {
			t.Errorf("status %d: err %v", c.status, err)
		}
	}
}

func TestMyCoverVotes(t *testing.T) {
	face := &folderFace{pages: map[string][]string{
		"/v2/me/cover-votes": {`{"object":"list","items":[{"object":"cover_vote","cover_id":"11","work_id":"7","vote":"up"},{"object":"cover_vote","cover_id":"12","work_id":"8","vote":"up"}],"next_cursor":null}`},
	}}
	c := New(Config{BaseURL: face.server(t).URL, AppKey: "k"})
	got, err := c.MyCoverVotes(context.Background(), "user-jwt")
	if err != nil {
		t.Fatalf("MyCoverVotes: %v", err)
	}
	if len(got) != 2 || got[0] != (MyCoverVote{WorkID: 7, CoverID: 11}) || got[1] != (MyCoverVote{WorkID: 8, CoverID: 12}) {
		t.Errorf("got %+v", got)
	}
	if calls := face.calls(); len(calls) != 1 || calls[0].Auth != "Bearer user-jwt" {
		t.Errorf("calls %+v", calls)
	}
}
