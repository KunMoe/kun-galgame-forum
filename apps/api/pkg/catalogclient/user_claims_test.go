package catalogclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeleteMyClaim_TravelsAsTheUserWithAPrecondition(t *testing.T) {
	var method, path, auth, ifMatch string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		auth = r.Header.Get("Authorization")
		ifMatch = r.Header.Get("If-Match")
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	if err := New(Config{BaseURL: srv.URL}).DeleteMyClaim(context.Background(), "user-jwt", 4649, ""); err != nil {
		t.Fatalf("DeleteMyClaim: %v — 204 carries no body and must not read as a decode failure", err)
	}
	if method != http.MethodDelete || path != "/v2/me/claims/4649" {
		t.Fatalf("hit %s %s, want DELETE /v2/me/claims/4649", method, path)
	}
	if auth != "Bearer user-jwt" {
		t.Errorf("auth = %q, want the user's bearer and no Basic credential", auth)
	}
	if ifMatch != "*" {
		t.Errorf("If-Match = %q; the operation lists 412 and 428 among its responses", ifMatch)
	}
}

func TestDeleteMyClaim_SurfacesTheUpstreamRefusal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"code":"INVALID_STATE_TRANSITION","detail":"claim is live"}`))
	}))
	t.Cleanup(srv.Close)

	err := New(Config{BaseURL: srv.URL}).DeleteMyClaim(context.Background(), "user-jwt", 4649, "")
	var apiErr *UserAPIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusConflict {
		t.Fatalf("err = %v, want the 409 catalog raises for anything but a draft", err)
	}
}

func TestDeleteMyClaim_NeedsATokenAndABaseURL(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	t.Cleanup(srv.Close)

	if err := New(Config{BaseURL: srv.URL}).DeleteMyClaim(context.Background(), "", 1, ""); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("err = %v, want ErrUnauthorized", err)
	}
	if called {
		t.Error("a tokenless user-plane call must not reach the catalog")
	}
	if err := New(Config{}).DeleteMyClaim(context.Background(), "user-jwt", 1, ""); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("err = %v, want ErrNotConfigured", err)
	}
}

func TestPatchMyClaim_ForwardsIfMatchAndDecodesTheLastEvent(t *testing.T) {
	var ifMatch string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ifMatch = r.Header.Get("If-Match")
		w.Header().Set("ETag", `"v8"`)
		_, _ = w.Write([]byte(`{"object":"claim","id":"4649","state":"pending","display_name":"CLANNAD","site":"kungal",` +
			`"product_work_id":"4649","last_event":{"object":"claim_event","id":"77","from_state":"declined","to_state":"pending",` +
			`"reason":"fixed the titles","actor_uid":"12","created_at":"2026-09-20T01:02:03Z"},` +
			`"first_acted_at":"2026-09-01T00:00:00Z","acted_count":3}`))
	}))
	t.Cleanup(srv.Close)

	item, etag, err := New(Config{BaseURL: srv.URL}).PatchMyClaim(context.Background(), "user-jwt", 4649, "pending", `"v7"`)
	if err != nil {
		t.Fatalf("PatchMyClaim: %v", err)
	}
	if ifMatch != `"v7"` {
		t.Errorf("If-Match = %q, want the caller's validator forwarded", ifMatch)
	}
	if etag != `"v8"` {
		t.Errorf("etag = %q, want the response ETag", etag)
	}
	if item.WorkID != 4649 || item.ClaimState != "pending" || item.Site != "kungal" || item.ActedCount != 3 {
		t.Fatalf("item = %+v", item)
	}
	if item.LastEventID != 77 || item.LastToState != "pending" || item.LastFromState == nil || *item.LastFromState != "declined" {
		t.Errorf("last event = %d %v→%s", item.LastEventID, item.LastFromState, item.LastToState)
	}
	if item.LastReason == nil || *item.LastReason != "fixed the titles" || item.LastActorUID != 12 {
		t.Errorf("reason/actor = %v/%d; the decline reason lives in last_event", item.LastReason, item.LastActorUID)
	}
	if item.LastEventAt.IsZero() || item.FirstActedAt.IsZero() {
		t.Errorf("times = %v / %v", item.LastEventAt, item.FirstActedAt)
	}
}

func TestDecideClaim_WithoutAValidatorSendsStar(t *testing.T) {
	var path, ifMatch string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, ifMatch = r.URL.Path, r.Header.Get("If-Match")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"object":"decision","id":"78","decision":"unban","note":"","from_state":"hidden","to_state":"draft"}`))
	}))
	t.Cleanup(srv.Close)

	if err := New(Config{BaseURL: srv.URL}).DecideClaim(context.Background(), "user-jwt", 4649, "unban", "", ""); err != nil {
		t.Fatalf("DecideClaim: %v", err)
	}
	if path != "/v2/moderation/claims/4649/decisions" || ifMatch != "*" {
		t.Errorf("hit %s with If-Match %q", path, ifMatch)
	}
}

func TestSubmitWorkUser_SendsReleasedAndTheIdempotencyKey(t *testing.T) {
	var body map[string]any
	var key string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key = r.Header.Get("Idempotency-Key")
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"object":"claim","id":"9001","state":"pending","display_name":"x","site":"kungal","product_work_id":null}`))
	}))
	t.Cleanup(srv.Close)

	res, err := New(Config{BaseURL: srv.URL}).SubmitWorkUser(context.Background(), "user-jwt", UserWorkSubmitRequest{
		Fields:         map[string]any{"catalog.work.display_name": "x"},
		Released:       &WorkSubmitDate{Y: 2004, M: 4},
		IdempotencyKey: "k-1",
	})
	if err != nil {
		t.Fatalf("SubmitWorkUser: %v", err)
	}
	if res.WorkID != 9001 || res.ClaimState != "pending" {
		t.Errorf("result = %+v", res)
	}
	if key != "k-1" {
		t.Errorf("Idempotency-Key = %q", key)
	}
	released, _ := body["released"].(map[string]any)
	if released["y"] != float64(2004) || released["m"] != float64(4) || released["d"] != nil {
		t.Errorf("released = %v, want {y:2004, m:4} with no day", body["released"])
	}
	if _, ok := body["field_values"].(map[string]any); !ok {
		t.Errorf("field_values missing: %v", body)
	}
}

func TestListModerationClaims_PassesTheCursorBothWays(t *testing.T) {
	var path, state, cursor string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		state, cursor = r.URL.Query().Get("claim_state"), r.URL.Query().Get("cursor")
		_, _ = w.Write([]byte(`{"object":"list","items":[{"object":"claim","id":"5","state":"pending","display_name":"a","site":"kungal"}],"next_cursor":"cur_next"}`))
	}))
	t.Cleanup(srv.Close)

	page, err := New(Config{BaseURL: srv.URL}).ListModerationClaims(context.Background(), "user-jwt",
		UserClaimFilter{ClaimStates: []string{"pending", "hidden"}, Cursor: "cur_prev", Limit: 20})
	if err != nil {
		t.Fatalf("ListModerationClaims: %v", err)
	}
	if path != "/v2/moderation/claims" || state != "pending,hidden" || cursor != "cur_prev" {
		t.Errorf("hit %s claim_state=%q cursor=%q", path, state, cursor)
	}
	if page.NextCursor != "cur_next" || len(page.Items) != 1 || page.Items[0].WorkID != 5 {
		t.Errorf("page = %+v", page)
	}
}
