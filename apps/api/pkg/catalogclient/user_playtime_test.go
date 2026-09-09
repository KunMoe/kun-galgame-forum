package catalogclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

func TestReportPlaytime_ForwardsBearerAndBody(t *testing.T) {
	var gotPath, gotAuth, gotMethod string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth, gotMethod = r.URL.Path, r.Header.Get("Authorization"), r.Method
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"work_id":7,"minutes":720}}`))
	}))
	defer srv.Close()

	// No client_id/secret: the playtime face is a user-plane call and must not
	// need the s2s credentials the rest of catalogclient carries.
	c := New(Config{BaseURL: srv.URL})
	got, err := c.ReportPlaytime(context.Background(), "user-jwt", 7,
		PlaytimeReport{Minutes: 720})
	if err != nil {
		t.Fatalf("ReportPlaytime: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %s, want PUT", gotMethod)
	}
	if gotPath != "/v2/me/playtimes/7" {
		t.Errorf("path = %s, want /v2/me/playtimes/7", gotPath)
	}
	if gotAuth != "Bearer user-jwt" {
		t.Errorf("auth = %q, want the user's bearer", gotAuth)
	}
	if gotBody["minutes"] != float64(720) {
		t.Errorf("body = %v, want the absolute minutes", gotBody)
	}
	if _, ok := gotBody["status"]; ok {
		t.Errorf("playtime PUT must not send status, got %v", gotBody)
	}
	if got.Minutes != 720 {
		t.Errorf("record = %+v", got)
	}
}

func TestMyPlaytime_NullPayloadIsNotAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":null}`))
	}))
	defer srv.Close()

	got, err := New(Config{BaseURL: srv.URL}).MyPlaytime(context.Background(), "user-jwt", 7)
	if err != nil {
		t.Fatalf("MyPlaytime: %v", err)
	}
	if got != nil {
		t.Fatalf("got %+v, want nil — never having reported is a 200 with a null payload", got)
	}
}

func TestPlaytime_ScopeDenialIsDistinctFromForbidden(t *testing.T) {
	cases := []struct {
		name string
		body string
		want error
	}{
		{"missing scope", `{"code":233,"message":"the access token is missing the playtime:write scope"}`, ErrInsufficientScope},
		{"plain forbidden", `{"code":233,"message":"banned"}`, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			_, err := New(Config{BaseURL: srv.URL}).ReportPlaytime(
				context.Background(), "user-jwt", 7, PlaytimeReport{Minutes: 60})
			if tc.want != nil {
				if !errors.Is(err, tc.want) {
					t.Fatalf("err = %v, want ErrInsufficientScope", err)
				}
				return
			}
			if errors.Is(err, ErrInsufficientScope) {
				t.Fatalf("a plain 403 must not read as a scope denial: %v", err)
			}
			var apiErr *UserAPIError
			if !errors.As(err, &apiErr) || apiErr.Status != http.StatusForbidden {
				t.Fatalf("err = %v, want a UserAPIError carrying 403", err)
			}
		})
	}
}

func TestListMyPlaytime_PassesCursorAndReturnsNext(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"items":[{"work_id":7,"minutes":120}],"next_cursor":"abc"}}`))
	}))
	defer srv.Close()

	items, cursor, err := New(Config{BaseURL: srv.URL}).ListMyPlaytime(
		context.Background(), "user-jwt", "abc-prev", 100)
	if err != nil {
		t.Fatalf("ListMyPlaytime: %v", err)
	}
	if gotQuery.Get("cursor") != "abc-prev" {
		t.Errorf("cursor param = %q", gotQuery.Get("cursor"))
	}
	if len(gotQuery) != 2 {
		t.Errorf("query = %v, want only cursor and limit", gotQuery)
	}
	limit, err := strconv.Atoi(gotQuery.Get("limit"))
	if err != nil || limit <= 0 || limit > 100 {
		t.Errorf("limit = %q, want 1–100", gotQuery.Get("limit"))
	}
	if len(items) != 1 || items[0].WorkID != 7 || items[0].Minutes != 120 {
		t.Errorf("items = %+v", items)
	}
	if cursor != "abc" {
		t.Errorf("cursor = %q", cursor)
	}
}

func TestMyWorkState_NotFoundIsNone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	got, err := New(Config{BaseURL: srv.URL}).MyWorkState(context.Background(), "user-jwt", 7)
	if err != nil {
		t.Fatalf("MyWorkState: %v", err)
	}
	if got != nil {
		t.Fatalf("got %+v, want nil — never having set a state is a 404", got)
	}
}

func TestPutWorkState_OmitsNilCompletionAndEchoes(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"work_state","work_id":"7","state":"done","completion":null,"created_at":"2026-08-20T00:00:00Z","updated_at":"2026-08-20T00:00:00Z"}`))
	}))
	defer srv.Close()

	got, err := New(Config{BaseURL: srv.URL}).PutWorkState(
		context.Background(), "user-jwt", 7, WorkStateDone, nil)
	if err != nil {
		t.Fatalf("PutWorkState: %v", err)
	}
	if gotBody["state"] != WorkStateDone {
		t.Errorf("body = %v", gotBody)
	}
	if _, ok := gotBody["completion"]; ok {
		t.Errorf("nil completion must be omitted, got %v", gotBody)
	}
	if got == nil || got.WorkID != 7 || got.State != WorkStateDone || got.Completion != nil {
		t.Errorf("record = %+v", got)
	}
}

func TestPutWorkState_SendsCompletionWhenSet(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"work_state","work_id":"7","state":"done","completion":"all","created_at":"2026-08-20T00:00:00Z","updated_at":"2026-08-20T00:00:00Z"}`))
	}))
	defer srv.Close()

	all := "all"
	got, err := New(Config{BaseURL: srv.URL}).PutWorkState(
		context.Background(), "user-jwt", 7, WorkStateDone, &all)
	if err != nil {
		t.Fatalf("PutWorkState: %v", err)
	}
	if gotBody["completion"] != "all" {
		t.Errorf("body = %v, want completion echoed", gotBody)
	}
	if got == nil || got.Completion == nil || *got.Completion != "all" {
		t.Errorf("record = %+v", got)
	}
}

func TestMyWorkStates_ParsesItemsAndIgnoresMissing(t *testing.T) {
	var gotIDs string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotIDs = r.URL.Query().Get("work_ids")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","items":[{"object":"work_state","work_id":"7","state":"done","completion":null}],"missing":["9"]}`))
	}))
	defer srv.Close()

	got, err := New(Config{BaseURL: srv.URL}).MyWorkStates(
		context.Background(), "user-jwt", []int64{7, 9})
	if err != nil {
		t.Fatalf("MyWorkStates: %v", err)
	}
	if gotIDs != "7,9" {
		t.Errorf("work_ids = %q, want 7,9", gotIDs)
	}
	if rec, ok := got[7]; !ok || rec.State != WorkStateDone {
		t.Errorf("got[7] = %+v", got[7])
	}
	if _, ok := got[9]; ok {
		t.Errorf("missing id 9 must not appear, got %+v", got)
	}
}
