package catalogclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

func TestDeleteWorkState_NotFoundIsSuccess(t *testing.T) {
	var gotMethod, gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotAuth = r.Method, r.URL.Path, r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	err := New(Config{BaseURL: srv.URL}).DeleteWorkState(context.Background(), "user-jwt", 7)
	if err != nil {
		t.Fatalf("DeleteWorkState: %v, want nil — deleting a state that is not there is success", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %s, want DELETE", gotMethod)
	}
	if gotPath != "/v2/me/work-states/7" {
		t.Errorf("path = %s, want /v2/me/work-states/7", gotPath)
	}
	if gotAuth != "Bearer user-jwt" {
		t.Errorf("auth = %q, want the user's bearer", gotAuth)
	}
}

func TestListMyWorkStates_TwoPageCursorSweep(t *testing.T) {
	var gotQuery url.Values
	var pages int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		pages++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("cursor") == "" {
			_, _ = w.Write([]byte(`{"object":"list","items":[{"object":"work_state","work_id":"7","state":"wish","completion":null}],"next_cursor":"page-2"}`))
			return
		}
		_, _ = w.Write([]byte(`{"object":"list","items":[{"object":"work_state","work_id":"9","state":"done","completion":"main"}]}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL})
	page1, cursor, err := c.ListMyWorkStates(context.Background(), "user-jwt", "", 100)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if gotQuery.Get("cursor") != "" {
		t.Errorf("first page cursor param = %q, want empty", gotQuery.Get("cursor"))
	}
	limit, err := strconv.Atoi(gotQuery.Get("limit"))
	if err != nil || limit <= 0 || limit > 100 {
		t.Errorf("limit = %q, want 1–100", gotQuery.Get("limit"))
	}
	if len(page1) != 1 || page1[0].WorkID != 7 || page1[0].State != WorkStateWish {
		t.Errorf("page1 = %+v", page1)
	}
	if cursor != "page-2" {
		t.Errorf("cursor = %q, want page-2", cursor)
	}

	page2, next, err := c.ListMyWorkStates(context.Background(), "user-jwt", cursor, 100)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}
	if gotQuery.Get("cursor") != "page-2" {
		t.Errorf("second page cursor param = %q", gotQuery.Get("cursor"))
	}
	if len(page2) != 1 || page2[0].WorkID != 9 || page2[0].State != WorkStateDone {
		t.Errorf("page2 = %+v", page2)
	}
	if page2[0].Completion == nil || *page2[0].Completion != "main" {
		t.Errorf("page2 completion = %v, want main", page2[0].Completion)
	}
	if next != "" {
		t.Errorf("next cursor = %q, want empty at the end of the sweep", next)
	}
	if pages != 2 {
		t.Errorf("pages = %d, want 2", pages)
	}
}
