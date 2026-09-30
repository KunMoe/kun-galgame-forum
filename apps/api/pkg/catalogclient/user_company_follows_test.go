package catalogclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCatalogClientCompanyFollows(t *testing.T) {
	const userToken = "user-jwt"
	type hit struct {
		method, path, auth, cursor, limit string
	}
	var hits []hit
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits = append(hits, hit{
			method: r.Method,
			path:   r.URL.Path,
			auth:   r.Header.Get("Authorization"),
			cursor: r.URL.Query().Get("cursor"),
			limit:  r.URL.Query().Get("limit"),
		})
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case http.MethodGet + " /v2/me/followed-companies/7":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":"NOT_FOUND","title":"Not found"}`))
		case http.MethodGet + " /v2/me/followed-companies/8":
			_, _ = w.Write([]byte(`{"object":"company_follow","company_id":"8","created_at":"2026-09-01T12:00:00Z"}`))
		case http.MethodPut + " /v2/me/followed-companies/8":
			_, _ = w.Write([]byte(`{"object":"company_follow","company_id":"8","created_at":"2026-09-01T12:00:00Z"}`))
		case http.MethodPut + " /v2/me/followed-companies/9":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":"ENTITY_MERGED","title":"Merged","current_id":"3"}`))
		case http.MethodPut + " /v2/me/followed-companies/10":
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"code":"VALIDATION_FAILED","title":"Validation failed","detail":"at most 1000 follows"}`))
		case http.MethodDelete + " /v2/me/followed-companies/8":
			w.WriteHeader(http.StatusNoContent)
		case http.MethodDelete + " /v2/me/followed-companies/11":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":"NOT_FOUND"}`))
		case http.MethodGet + " /v2/me/followed-companies":
			_, _ = w.Write([]byte(`{"object":"list","items":[{"object":"company_follow","company_id":"8","created_at":"2026-09-02T00:00:00Z"}],"next_cursor":"cur_next"}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.String())
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, AppKey: "app-key"})
	ctx := context.Background()

	following, err := c.MyCompanyFollow(ctx, userToken, 7)
	if err != nil || following {
		t.Fatalf("GET missing = %v, %v; want false, nil", following, err)
	}
	following, err = c.MyCompanyFollow(ctx, userToken, 8)
	if err != nil || !following {
		t.Fatalf("GET followed = %v, %v; want true, nil", following, err)
	}
	if err := c.FollowCompany(ctx, userToken, 8); err != nil {
		t.Fatalf("PUT follow: %v", err)
	}
	err = c.FollowCompany(ctx, userToken, 9)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("PUT unknown = %v, want ErrNotFound", err)
	}
	err = c.FollowCompany(ctx, userToken, 10)
	var api *UserAPIError
	if !errors.As(err, &api) || api.Status != http.StatusUnprocessableEntity {
		t.Fatalf("PUT cap = %v, want 422 UserAPIError", err)
	}
	if err := c.UnfollowCompany(ctx, userToken, 8); err != nil {
		t.Fatalf("DELETE followed: %v", err)
	}
	if err := c.UnfollowCompany(ctx, userToken, 11); err != nil {
		t.Fatalf("DELETE missing = %v, want nil", err)
	}
	rows, cursor, err := c.ListMyCompanyFollows(ctx, userToken, "cur_abc", 100)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 1 || rows[0].CompanyID != 8 || rows[0].CreatedAt != "2026-09-02T00:00:00Z" {
		t.Fatalf("rows = %+v", rows)
	}
	if cursor != "cur_next" {
		t.Fatalf("cursor = %q, want cur_next", cursor)
	}

	want := []hit{
		{http.MethodGet, "/v2/me/followed-companies/7", "Bearer " + userToken, "", ""},
		{http.MethodGet, "/v2/me/followed-companies/8", "Bearer " + userToken, "", ""},
		{http.MethodPut, "/v2/me/followed-companies/8", "Bearer " + userToken, "", ""},
		{http.MethodPut, "/v2/me/followed-companies/9", "Bearer " + userToken, "", ""},
		{http.MethodPut, "/v2/me/followed-companies/10", "Bearer " + userToken, "", ""},
		{http.MethodDelete, "/v2/me/followed-companies/8", "Bearer " + userToken, "", ""},
		{http.MethodDelete, "/v2/me/followed-companies/11", "Bearer " + userToken, "", ""},
		{http.MethodGet, "/v2/me/followed-companies", "Bearer " + userToken, "cur_abc", "100"},
	}
	if len(hits) != len(want) {
		t.Fatalf("hits = %+v, want %d", hits, len(want))
	}
	for i, got := range hits {
		if got != want[i] {
			t.Errorf("hit %d = %+v, want %+v", i, got, want[i])
		}
		if got.auth == "Bearer app-key" {
			t.Errorf("hit %d sent the app key", i)
		}
	}
}
