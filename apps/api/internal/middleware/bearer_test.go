package middleware

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/pkg/perm"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

const goodToken = "good-token"

type stubVerifier struct{ err error }

func (s stubVerifier) Verify(_ context.Context, raw string) (*oauth.AccessClaims, error) {
	if s.err != nil {
		return nil, s.err
	}
	if raw != goodToken {
		return nil, stderrors.New("bad token")
	}
	return &oauth.AccessClaims{
		ID:        1207,
		Name:      "kun",
		Roles:     []string{"user", "creator"},
		SiteRoles: []string{"moderator"},
		ClientID:  "kungal-app",
	}, nil
}

type seenLog struct{ calls [][]string }

type echoed struct {
	Anon     bool     `json:"anon"`
	ID       int      `json:"id"`
	Roles    []string `json:"roles"`
	Bearer   bool     `json:"bearer"`
	Token    string   `json:"token"`
	Moderate bool     `json:"moderate"`
	Hide     bool     `json:"hide"`
	Code     int      `json:"code"`
	Message  string   `json:"message"`
}

func newAuthApp(t *testing.T, verifier AccessTokenVerifier, firstSeen FirstSeen) *fiber.App {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: miniredis.RunT(t).Addr()})
	authn := NewAuthenticator(rdb, nil, NewBearer(verifier, rdb, firstSeen))

	echo := func(c fiber.Ctx) error {
		u := GetUser(c)
		if u == nil {
			return c.JSON(echoed{Anon: true})
		}
		return c.JSON(echoed{
			ID: u.ID, Roles: u.Roles, Bearer: u.ViaBearer(),
			Token: GetAccessToken(c), Moderate: u.CanModerate(), Hide: u.Can(perm.TopicHide),
		})
	}
	app := fiber.New()
	app.Get("/optional", authn.OptionalAuth(), echo)
	app.Get("/required", authn.OptionalAuth(), authn.Auth(), echo)
	return app
}

func call(t *testing.T, app *fiber.App, path, authorization string) (int, echoed) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if authorization != "" {
		req.Header.Set(fiber.HeaderAuthorization, authorization)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	var out echoed
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("%s: %s", path, body)
	}
	return resp.StatusCode, out
}

func TestBearerAuthenticatesWithoutStaffPowers(t *testing.T) {
	var seen seenLog
	app := newAuthApp(t, stubVerifier{}, func(uid int, roles []string) error {
		seen.calls = append(seen.calls, roles)
		return nil
	})

	for range 2 {
		status, got := call(t, app, "/required", "Bearer "+goodToken)
		if status != http.StatusOK || !got.Bearer || got.ID != 1207 || got.Token != goodToken {
			t.Fatalf("status %d, %+v", status, got)
		}
		if slices.Contains(got.Roles, "moderator") || !slices.Contains(got.Roles, "creator") {
			t.Errorf("roles = %v, want staff roles stripped and the rest kept", got.Roles)
		}
		if got.Moderate {
			t.Error("a Bearer user must not moderate")
		}
	}

	if len(seen.calls) != 1 {
		t.Fatalf("first-seen ran %d times, want once", len(seen.calls))
	}
	if !slices.Contains(seen.calls[0], "moderator") {
		t.Errorf("first-seen got %v; the trust boost needs the unstripped roles", seen.calls[0])
	}
}

func TestBearerNeverReachesStaffGates(t *testing.T) {
	t.Cleanup(func() { perm.SetUserOverrides(nil) })
	perm.SetUserOverrides(map[int][]perm.Override{
		1207: {{Permission: perm.TopicHide, Effect: perm.EffectGrant}},
	})
	app := newAuthApp(t, stubVerifier{}, nil)

	status, got := call(t, app, "/required", "Bearer "+goodToken)
	if status != http.StatusOK || got.Hide || got.Moderate {
		t.Errorf("a Bearer user holds a staff power through a personal grant: status %d, %+v", status, got)
	}
	if !perm.CanUser(1207, nil, perm.TopicHide) {
		t.Fatal("fixture broken: the override should grant on the web path")
	}
}

func TestBearerFailures(t *testing.T) {
	for _, tc := range []struct {
		why        string
		verifier   AccessTokenVerifier
		firstSeen  FirstSeen
		path, auth string
		wantStatus int
		wantCode   int
	}{
		{"a bad token is refused even where login is optional", stubVerifier{}, nil,
			"/optional", "Bearer expired", 401, 205},
		{"an empty bearer is still a bearer", stubVerifier{}, nil,
			"/optional", "Bearer ", 401, 205},
		{"closed channel", nil, nil,
			"/optional", "Bearer " + goodToken, 401, 205},
		{"an unreachable OP is not a logout", stubVerifier{err: oauth.ErrKeysUnavailable}, nil,
			"/required", "Bearer " + goodToken, 500, 233},
		{"first-seen provisioning failed", stubVerifier{}, func(int, []string) error { return stderrors.New("db down") },
			"/required", "Bearer " + goodToken, 500, 233},
	} {
		app := newAuthApp(t, tc.verifier, tc.firstSeen)
		status, got := call(t, app, tc.path, tc.auth)
		if status != tc.wantStatus || got.Code != tc.wantCode {
			t.Errorf("%s: status %d code %d, want %d/%d", tc.why, status, got.Code, tc.wantStatus, tc.wantCode)
		}
	}
}

func TestFailedFirstSeenIsRetried(t *testing.T) {
	calls := 0
	app := newAuthApp(t, stubVerifier{}, func(int, []string) error {
		calls++
		if calls == 1 {
			return stderrors.New("db blip")
		}
		return nil
	})
	call(t, app, "/required", "Bearer "+goodToken)
	if status, _ := call(t, app, "/required", "Bearer "+goodToken); status != http.StatusOK {
		t.Fatalf("second request: status %d", status)
	}
	if calls != 2 {
		t.Errorf("first-seen ran %d times, want the failed run retried once", calls)
	}
}

func TestNonBearerAuthorizationStaysOnTheCookiePath(t *testing.T) {
	app := newAuthApp(t, stubVerifier{}, nil)
	for _, auth := range []string{"", "Basic Zm9vOmJhcg=="} {
		if status, got := call(t, app, "/optional", auth); status != http.StatusOK || !got.Anon {
			t.Errorf("Authorization %q: status %d, %+v", auth, status, got)
		}
	}
}

func TestBearerUserHasNoStaffPowers(t *testing.T) {
	u := &UserInfo{ID: 1207, Roles: []string{"user", "moderator", "admin"}, viaBearer: true}
	if u.CanModerate() || u.CanAdminister() || u.Can(perm.TopicHide) {
		t.Errorf("a Bearer UserInfo exercised a staff power: moderate=%v administer=%v hide=%v",
			u.CanModerate(), u.CanAdminister(), u.Can(perm.TopicHide))
	}
}
