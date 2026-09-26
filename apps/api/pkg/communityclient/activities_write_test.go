package communityclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kun-galgame-api/pkg/communityclient"
)

func TestWriteActivitiesPathBodyAndEnvelope(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "message": "成功",
			"data": map[string]any{"results": []any{
				map[string]any{"key": "topic_creation:1", "outcome": "created"},
			}},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).WriteActivities(context.Background(), []communityclient.ActivityWriteItem{{
		Key: "topic_creation:1", ActorID: 3, Revision: 1, Verb: "publish",
		ObjectKind: "topic", ObjectLabel: "话题", Title: "hi", URL: "https://www.kungal.com/topic/1",
		ContentLimit: "sfw", OccurredAt: "2026-09-26T00:00:00.000000Z",
	}})
	if err != nil {
		t.Fatalf("WriteActivities: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/activities" {
		t.Errorf("method/path = %s %q", gotMethod, gotPath)
	}
	if !strings.Contains(gotBody, `"items"`) || !strings.Contains(gotBody, `"key":"topic_creation:1"`) {
		t.Errorf("body = %q", gotBody)
	}
	for _, absent := range []string{"work_id", "cover_image_hash", "notify", "removed"} {
		if strings.Contains(gotBody, absent) {
			t.Errorf("optional %s was sent: %s", absent, gotBody)
		}
	}
	if len(out.Results) != 1 || out.Results[0].Outcome != "created" {
		t.Errorf("decoded = %+v", out)
	}
}

func TestWriteActivitiesNonZeroCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 40001, "message": "bad"})
	}))
	defer srv.Close()

	_, err := newTestClient(srv.URL).WriteActivities(context.Background(), []communityclient.ActivityWriteItem{{
		Key: "topic_creation:1", ActorID: 3, Revision: 1,
	}})
	var apiErr *communityclient.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != 40001 {
		t.Errorf("err = %v, want *APIError code=40001", err)
	}
}

func TestWriteActivitiesUnprocessable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 4, "message": "malformed"})
	}))
	defer srv.Close()

	_, err := newTestClient(srv.URL).WriteActivities(context.Background(), []communityclient.ActivityWriteItem{{
		Key: "topic_creation:1", ActorID: 3, Revision: 1,
	}})
	var apiErr *communityclient.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusUnprocessableEntity {
		t.Errorf("err = %v, want *APIError status=422", err)
	}
}

func TestListSiteActivitiesPathQueryAndEnvelope(t *testing.T) {
	var gotMethod, gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotQuery = r.Method, r.URL.Path, r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"activities": []any{
					map[string]any{"id": 9, "key": "topic_creation:1", "actor_id": 3, "removed": false},
				},
				"next_cursor": "9",
			},
		})
	}))
	defer srv.Close()

	out, err := newTestClient(srv.URL).ListSiteActivities(context.Background(), "cur", 1000)
	if err != nil {
		t.Fatalf("ListSiteActivities: %v", err)
	}
	if gotMethod != http.MethodGet || gotPath != "/activities" {
		t.Errorf("method/path = %s %q", gotMethod, gotPath)
	}
	if !strings.Contains(gotQuery, "cursor=cur") || !strings.Contains(gotQuery, "limit=1000") {
		t.Errorf("query = %q", gotQuery)
	}
	if len(out.Activities) != 1 || out.Activities[0].Key != "topic_creation:1" || out.NextCursor != "9" {
		t.Errorf("decoded = %+v", out)
	}
}

func TestListSiteActivitiesNonZeroCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 40001, "message": "bad"})
	}))
	defer srv.Close()

	_, err := newTestClient(srv.URL).ListSiteActivities(context.Background(), "", 10)
	var apiErr *communityclient.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != 40001 {
		t.Errorf("err = %v, want *APIError code=40001", err)
	}
}

func TestListSiteActivitiesUnprocessable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 4, "message": "malformed"})
	}))
	defer srv.Close()

	_, err := newTestClient(srv.URL).ListSiteActivities(context.Background(), "", 0)
	var apiErr *communityclient.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusUnprocessableEntity {
		t.Errorf("err = %v, want *APIError status=422", err)
	}
}
