package catalogclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCatalogClientPlaytimeSources(t *testing.T) {
	var deleted []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer user-jwt" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.EscapedPath() {
		case "GET /v2/me/playtimes/7/sources":
			_, _ = w.Write([]byte(`{"object":"list","items":[
				{"object":"playtime_source","client_id":"timer-app","app_name":null,"minutes":600,"last_played_at":"2026-11-01T10:00:00Z","updated_at":"2026-11-02T10:00:00Z","self":false},
				{"object":"playtime_source","client_id":"4ed9","app_name":"鲲 Galgame 论坛","minutes":90,"last_played_at":null,"updated_at":"2026-10-01T10:00:00Z","self":true}]}`))
		case "DELETE /v2/me/playtimes/7/sources/timer%2Fapp", "DELETE /v2/me/playtimes/7/sources":
			deleted = append(deleted, r.URL.EscapedPath())
			w.WriteHeader(http.StatusNoContent)
		case "DELETE /v2/me/playtimes/7/sources/gone":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":"NOT_FOUND","title":"Not found"}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.EscapedPath())
			w.WriteHeader(http.StatusTeapot)
		}
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL, AppKey: "app-key"})
	ctx := context.Background()

	got, err := c.MyPlaytimeSources(ctx, "user-jwt", 7)
	if err != nil || len(got) != 2 {
		t.Fatalf("sources = %+v, %v", got, err)
	}
	if got[0].ClientID != "timer-app" || got[0].AppName != nil || got[0].Minutes != 600 || got[0].Self || got[0].LastPlayedAt == nil {
		t.Errorf("first source %+v", got[0])
	}
	if !got[1].Self || got[1].AppName == nil || *got[1].AppName != "鲲 Galgame 论坛" || got[1].LastPlayedAt != nil {
		t.Errorf("self source %+v", got[1])
	}

	if err := c.DeleteMyPlaytimeSource(ctx, "user-jwt", 7, "timer/app"); err != nil {
		t.Fatalf("delete one: %v", err)
	}
	if err := c.DeleteMyPlaytimeSource(ctx, "user-jwt", 7, "gone"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete a missing source = %v, want ErrNotFound", err)
	}
	if err := c.DeleteMyPlaytimeSources(ctx, "user-jwt", 7); err != nil {
		t.Fatalf("delete all: %v", err)
	}
	if len(deleted) != 2 {
		t.Errorf("deleted %v", deleted)
	}
}
