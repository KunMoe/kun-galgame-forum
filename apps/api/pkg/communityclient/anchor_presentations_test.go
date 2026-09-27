package communityclient_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kun-galgame-api/pkg/communityclient"
)

func TestWriteAnchorPresentationsPathBodyAndEnvelope(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "message": "成功",
			"data": map[string]any{"results": []any{
				map[string]any{"anchor_kind": 1, "anchor_id": "42", "outcome": "created"},
			}},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).WriteAnchorPresentations(context.Background(), []communityclient.AnchorPresentationItem{{
		AnchorKind: 1, AnchorID: "42", Revision: 1, Title: "Work",
		URL: "https://www.kungal.com/galgame/42", ContentLimit: "sfw",
	}})
	if err != nil {
		t.Fatalf("WriteAnchorPresentations: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/anchor-presentations" {
		t.Errorf("method/path = %s %q", gotMethod, gotPath)
	}
	if !strings.Contains(gotBody, `"items"`) || !strings.Contains(gotBody, `"anchor_id":"42"`) {
		t.Errorf("body = %q", gotBody)
	}
	for _, absent := range []string{"work_id", "cover_image_hash", "removed"} {
		if strings.Contains(gotBody, absent) {
			t.Errorf("optional %s was sent: %s", absent, gotBody)
		}
	}
	if len(out.Results) != 1 || out.Results[0].Outcome != "created" || out.Results[0].AnchorID != "42" {
		t.Errorf("decoded = %+v", out)
	}
}

func TestWriteAnchorPresentationsTombstoneFourKeys(t *testing.T) {
	b, err := json.Marshal(communityclient.AnchorPresentationItem{
		AnchorKind: 2, AnchorID: "resource:9", Revision: 7, Removed: true,
		Title: "x", URL: "https://x", WorkID: 1, CoverImageHash: "h", ContentLimit: "sfw",
	})
	if err != nil {
		t.Fatal(err)
	}
	tomb, err := json.Marshal(communityclient.AnchorPresentationItem{
		AnchorKind: 2, AnchorID: "resource:9", Revision: 7, Removed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(tomb, &m); err != nil {
		t.Fatal(err)
	}
	if len(m) != 4 {
		t.Fatalf("tombstone keys = %v from %s", m, tomb)
	}
	for _, k := range []string{"anchor_kind", "anchor_id", "revision", "removed"} {
		if _, ok := m[k]; !ok {
			t.Errorf("missing %s in %s", k, tomb)
		}
	}
	if strings.Contains(string(tomb), "title") || strings.Contains(string(tomb), "url") {
		t.Errorf("tombstone carried live fields: %s", tomb)
	}
	var live map[string]any
	if err := json.Unmarshal(b, &live); err != nil {
		t.Fatal(err)
	}
	if live["removed"] != true {
		t.Errorf("removed live marshal = %s", b)
	}
}

func TestListAnchorPresentationsPathQueryAndNulls(t *testing.T) {
	var gotMethod, gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotQuery = r.Method, r.URL.Path, r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"presentations": []any{
					map[string]any{
						"anchor_kind": 1, "anchor_id": "42", "title": "Work",
						"url":     "https://www.kungal.com/galgame/42",
						"work_id": nil, "cover_image_hash": nil, "content_limit": "sfw",
						"revision": 3, "removed": false, "updated_at": "2026-09-27T00:00:00Z",
						"removed_at": nil,
					},
				},
				"next_cursor": "n1",
			},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).ListAnchorPresentations(context.Background(), "cur", 1000)
	if err != nil {
		t.Fatalf("ListAnchorPresentations: %v", err)
	}
	if gotMethod != http.MethodGet || gotPath != "/anchor-presentations" {
		t.Errorf("method/path = %s %q", gotMethod, gotPath)
	}
	if !strings.Contains(gotQuery, "cursor=cur") || !strings.Contains(gotQuery, "limit=1000") {
		t.Errorf("query = %q", gotQuery)
	}
	if len(out.Presentations) != 1 || out.NextCursor != "n1" {
		t.Errorf("decoded = %+v", out)
	}
	v := out.Presentations[0]
	if v.WorkID != nil || v.CoverImageHash != nil || v.RemovedAt != nil {
		t.Errorf("nulls = work=%v cover=%v removed_at=%v", v.WorkID, v.CoverImageHash, v.RemovedAt)
	}
}
