package userclient

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestUserBriefFoldsSiteRolesIntoRoles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"","data":{"users":[` +
			`{"id":7,"uuid":"u-7","name":"kun","roles":["creator"],"site_roles":["moderator"]}` +
			`],"not_found":[]}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "x", ClientSecret: "y"})

	u, ok, err := c.User(context.Background(), 7)
	if err != nil || !ok {
		t.Fatalf("User() ok=%v err=%v", ok, err)
	}
	if want := []string{"creator", "moderator"}; !slices.Equal(u.Roles, want) {
		t.Fatalf("brief Roles = %v, want the effective union %v", u.Roles, want)
	}
	if want := []string{"moderator"}; !slices.Equal(u.SiteRoles, want) {
		t.Fatalf("brief SiteRoles = %v, want %v", u.SiteRoles, want)
	}
}

func TestUserBriefNoSiteRoles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"","data":{"users":[` +
			`{"id":8,"uuid":"u-8","name":"ren","roles":["admin"]}` +
			`],"not_found":[]}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "x", ClientSecret: "y"})
	u, _, _ := c.User(context.Background(), 8)
	if want := []string{"admin"}; !slices.Equal(u.Roles, want) {
		t.Fatalf("brief Roles = %v, want %v (no site grant → unchanged)", u.Roles, want)
	}
}

func TestUserBriefDecodesAvatarFrame(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"","data":{"users":[` +
			`{"id":9,"name":"framed","cosmetics":{` +
			`"profile_background":{"item_id":9,"name":"stars","static_url":"https://img.example/decorations/bg.jpg"},` +
			`"avatar_frame":{"item_id":3,"name":"sakura","static_url":"https://img.example/decorations/f.png","animated_url":"https://img.example/decorations/f.webp"},` +
			`"nameplate":{"item_id":12,"static_url":"https://img.example/decorations/n.png"}}},` +
			`{"id":10,"name":"plain"}` +
			`],"not_found":[]}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "x", ClientSecret: "y"})
	users, err := c.Users(context.Background(), []int{9, 10})
	if err != nil {
		t.Fatalf("Users() err=%v", err)
	}
	want := Decoration{StaticURL: "https://img.example/decorations/f.png", AnimatedURL: "https://img.example/decorations/f.webp"}
	if f := users[9].Cosmetics.AvatarFrame; f == nil || *f != want {
		t.Fatalf("framed AvatarFrame = %+v, want %+v", f, want)
	}
	if f := users[10].Cosmetics.AvatarFrame; f != nil {
		t.Fatalf("plain AvatarFrame = %+v, want nil", f)
	}
}

func TestPublicSettings(t *testing.T) {
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"","data":{"site_id":null,"etag":"x","settings":{"auth.name_change_cost":23}}}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "x", ClientSecret: "y"})
	settings, err := c.PublicSettings(context.Background())
	if err != nil {
		t.Fatalf("PublicSettings() err=%v", err)
	}
	if gotPath != "/settings" {
		t.Fatalf("path %q, want /settings", gotPath)
	}
	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("x:y"))
	if gotAuth != wantAuth {
		t.Fatalf("Authorization %q, want %q", gotAuth, wantAuth)
	}
	if settings["auth.name_change_cost"] != float64(23) {
		t.Fatalf("settings = %#v", settings)
	}
}

func TestPublicSettingsNonZeroCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":9,"message":"nope"}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "x", ClientSecret: "y"})
	_, err := c.PublicSettings(context.Background())
	if err == nil {
		t.Fatal("PublicSettings() err=nil, want non-zero code error")
	}
}

func TestPublicSettingsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, ClientID: "x", ClientSecret: "y"})
	_, err := c.PublicSettings(context.Background())
	if err == nil {
		t.Fatal("PublicSettings() err=nil, want non-2xx error")
	}
}
