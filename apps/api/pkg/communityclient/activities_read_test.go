package communityclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"kun-galgame-api/pkg/communityclient"
)

func TestListFollowingActivitiesQueryAndDecode(t *testing.T) {
	var gotMethod, gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotQuery = r.Method, r.URL.Path, r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "message": "成功",
			"data": map[string]any{
				"groups": []any{
					map[string]any{
						"id": 11, "site": "kungal", "actor_id": 7, "verb": "publish",
						"object_kind": "topic", "object_label": "Topic", "day": "2026-09-25",
						"item_count": 2, "latest_at": "2026-09-25T12:00:00Z",
						"items": []any{
							map[string]any{
								"id": 101, "site": "kungal", "key": "kungal:topic:1", "actor_id": 7,
								"verb": "publish", "object_kind": "topic", "object_label": "Topic",
								"title": "Hello", "excerpt": "lede", "url": "https://www.kungal.com/topic/1",
								"cover_image_hash": "", "work_id": 9, "content_limit": "sfw",
								"occurred_at": "2026-09-25T12:00:00Z",
							},
						},
					},
				},
				"next_cursor": "cm_next",
			},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).ListFollowingActivities(context.Background(), 42, "cm_prev", 20, "sfw",
		[]string{"kungal", "moyu"}, []string{"publish", "like"})
	if err != nil {
		t.Fatalf("ListFollowingActivities: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q", gotMethod)
	}
	if gotPath != "/users/42/following/activities" {
		t.Errorf("path = %q", gotPath)
	}
	q, _ := http.NewRequest(http.MethodGet, "http://x/?"+gotQuery, nil)
	vals := q.URL.Query()
	if vals.Get("cursor") != "cm_prev" || vals.Get("limit") != "20" || vals.Get("content_limit") != "sfw" {
		t.Errorf("query %q missing pagination or content_limit", gotQuery)
	}
	if vals.Get("sites") != "kungal,moyu" || vals.Get("verbs") != "publish,like" {
		t.Errorf("query %q missing comma lists", gotQuery)
	}
	if len(out.Groups) != 1 || out.Groups[0].ID != 11 || out.Groups[0].Verb != "publish" ||
		len(out.Groups[0].Items) != 1 || out.Groups[0].Items[0].WorkID == nil || *out.Groups[0].Items[0].WorkID != 9 ||
		out.NextCursor != "cm_next" {
		t.Errorf("decoded = %+v", out)
	}
}

func TestListFollowingActivitiesOmitsEmptyParams(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"groups": []any{}}})
	}))
	defer srv.Close()

	_, err := newTestClient(srv.URL).ListFollowingActivities(context.Background(), 1, "", 0, "", nil, nil)
	if err != nil {
		t.Fatalf("ListFollowingActivities: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query %q, want empty", gotQuery)
	}
}

func TestListActivityGroupItems(t *testing.T) {
	var gotMethod, gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotQuery = r.Method, r.URL.Path, r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"items": []any{
					map[string]any{"id": 101, "site": "kungal", "key": "k", "actor_id": 7, "verb": "publish",
						"object_kind": "topic", "object_label": "Topic", "title": "t", "excerpt": "",
						"url": "https://www.kungal.com/topic/1", "content_limit": "all", "occurred_at": "2026-09-25T12:00:00Z"},
				},
				"next_cursor": "more",
			},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).ListActivityGroupItems(context.Background(), 11, "cm_prev", 20, "all")
	if err != nil {
		t.Fatalf("ListActivityGroupItems: %v", err)
	}
	if gotMethod != http.MethodGet || gotPath != "/activity-groups/11/items" {
		t.Errorf("method %q path %q", gotMethod, gotPath)
	}
	q, _ := http.NewRequest(http.MethodGet, "http://x/?"+gotQuery, nil)
	if q.URL.Query().Get("cursor") != "cm_prev" || q.URL.Query().Get("limit") != "20" || q.URL.Query().Get("content_limit") != "all" {
		t.Errorf("query %q", gotQuery)
	}
	if len(out.Items) != 1 || out.Items[0].ID != 101 || out.NextCursor != "more" {
		t.Errorf("decoded = %+v", out)
	}
}

func TestListActivityGroupItemsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 40400, "message": "no such group"})
	}))
	defer srv.Close()

	_, err := newTestClient(srv.URL).ListActivityGroupItems(context.Background(), 99, "", 0, "")
	var apiErr *communityclient.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusNotFound || apiErr.Code != 40400 {
		t.Errorf("err = %v, want *APIError 404", err)
	}
}

func TestGetFollowingActivitiesUnseen(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "data": map[string]any{"unseen_count": 4, "seen_at": nil},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).GetFollowingActivitiesUnseen(context.Background(), 42, "sfw",
		[]string{"kungal"}, []string{"publish"})
	if err != nil {
		t.Fatalf("GetFollowingActivitiesUnseen: %v", err)
	}
	if gotPath != "/users/42/following/activities/unseen" {
		t.Errorf("path = %q", gotPath)
	}
	q, _ := http.NewRequest(http.MethodGet, "http://x/?"+gotQuery, nil)
	if q.URL.Query().Get("content_limit") != "sfw" || q.URL.Query().Get("sites") != "kungal" || q.URL.Query().Get("verbs") != "publish" {
		t.Errorf("query %q", gotQuery)
	}
	if out.UnseenCount != 4 || out.SeenAt != nil {
		t.Errorf("decoded = %+v", out)
	}
}

func TestMarkFollowingActivitiesSeenOmitsAt(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"seen_at": "2026-09-26T00:00:00Z"}})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).MarkFollowingActivitiesSeen(context.Background(), 42, nil)
	if err != nil {
		t.Fatalf("MarkFollowingActivitiesSeen: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/users/42/following/activities/seen" {
		t.Errorf("method %q path %q", gotMethod, gotPath)
	}
	if gotBody != "{}" {
		t.Errorf("body %q, want {}", gotBody)
	}
	if out.SeenAt != "2026-09-26T00:00:00Z" {
		t.Errorf("decoded = %+v", out)
	}
}

func TestMarkFollowingActivitiesSeenSendsAt(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"seen_at": "2026-09-25T12:00:00Z"}})
	}))
	defer srv.Close()

	at := "2026-09-25T12:00:00Z"
	out, err := newTestClient(srv.URL).MarkFollowingActivitiesSeen(context.Background(), 42, &at)
	if err != nil {
		t.Fatalf("MarkFollowingActivitiesSeen: %v", err)
	}
	if gotBody != `{"at":"2026-09-25T12:00:00Z"}` {
		t.Errorf("body %q", gotBody)
	}
	if out.SeenAt != at {
		t.Errorf("decoded = %+v", out)
	}
}
