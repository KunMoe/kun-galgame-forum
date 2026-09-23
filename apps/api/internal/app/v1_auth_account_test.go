package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/pkg/userclient"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

const (
	acctUser   = 950000001
	acctGone   = 950000002
	acctBearer = 950000003
	acctLegacy = 950000004

	acctAvatarHash = "abababababababababababababababababababababababababababababababab"
	acctCDN        = "https://image.test.example"
)

type acctVerifier struct{}

func (acctVerifier) Verify(_ context.Context, raw string) (*oauth.AccessClaims, error) {
	if raw == "acct-bearer" {
		return &oauth.AccessClaims{
			ID: acctBearer, Name: "bearer", Roles: []string{"user"},
			SiteRoles: []string{"moderator"}, ClientID: "kungal-app",
		}, nil
	}
	return nil, fmt.Errorf("bad token")
}

type acctFix struct {
	app    *App
	spec   *specConformance
	rdb    *redis.Client
	failOA atomic.Bool
}

func newAcctFix(t *testing.T) *acctFix {
	t.Helper()
	f := &acctFix{}
	records := map[int]map[string]any{
		acctUser:   {"name": "new-name", "status": 0, "roles": []string{"user"}, "avatar_image_hash": acctAvatarHash},
		acctBearer: {"name": "bearer", "status": 0, "roles": []string{"moderator"}},
		acctLegacy: {"name": "legacy", "status": 0, "roles": []string{"user"}},
	}
	oauthSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f.failOA.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		users := []map[string]any{}
		notFound := []int{}
		for _, raw := range strings.Split(r.URL.Query().Get("ids"), ",") {
			id, _ := strconv.Atoi(raw)
			rec, ok := records[id]
			if !ok {
				notFound = append(notFound, id)
				continue
			}
			u := map[string]any{"id": id}
			for k, v := range rec {
				u[k] = v
			}
			users = append(users, u)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"users": users, "not_found": notFound}})
	}))
	t.Cleanup(oauthSrv.Close)

	mr := miniredis.RunT(t)
	f.rdb = redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: 0})
	t.Cleanup(func() { _ = f.rdb.Close() })

	uc := userclient.New(userclient.Config{
		BaseURL: oauthSrv.URL, ClientID: "c", ClientSecret: "s",
		ImageCDNBase: acctCDN, HTTPTimeout: 2 * time.Second,
	})
	cfg := testConfig()
	cfg.NextMoeAPI.ImageCDNBase = acctCDN
	f.app = &App{
		Fiber:      newFiber(),
		Config:     cfg,
		Redis:      f.rdb,
		UserClient: uc,
		Authn:      middleware.NewAuthenticator(f.rdb, nil, middleware.NewBearer(acctVerifier{}, f.rdb, nil)),
	}
	f.app.setupRoutes()
	f.spec = newSpecConformance(t)

	f.putSession(t, "sess-acct", middleware.UserInfo{
		ID: acctUser, Name: "old-name", Roles: []string{"user", "moderator", "some-site-name"},
		AdultConfirmed: true, NSFWDisplay: "blur",
	})
	f.putSession(t, "sess-acct-gone", middleware.UserInfo{ID: acctGone, Name: "gone", Roles: []string{"user"}})
	f.putSession(t, "sess-acct-legacy", middleware.UserInfo{ID: acctLegacy, Name: "legacy", Roles: []string{"user"}})
	return f
}

func (f *acctFix) putSession(t *testing.T, token string, u middleware.UserInfo) {
	t.Helper()
	data, err := json.Marshal(middleware.SessionData{
		UserInfo:         u,
		OAuthAccessToken: "access",
		OAuthExpiresAt:   time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.rdb.Set(context.Background(), middleware.SessionKey(token), data, middleware.SessionTTL).Err(); err != nil {
		t.Fatal(err)
	}
}

func (f *acctFix) account(t *testing.T, session, bearer string) (*http.Response, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/account", nil)
	if session != "" {
		req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: session})
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := f.app.Fiber.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	f.spec.checkPath(t, http.MethodGet, "/me/account", resp, body)
	out := map[string]any{}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("body: %v\n%s", err, body)
	}
	return resp, out
}

func TestV1AccountSession(t *testing.T) {
	f := newAcctFix(t)
	resp, body := f.account(t, "sess-acct", "")
	if resp.StatusCode != http.StatusOK || body["object"] != "account" || body["id"] != strconv.Itoa(acctUser) {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	if body["name"] != "new-name" {
		t.Errorf("name %v, want the account center's current name", body["name"])
	}
	if fmt.Sprint(body["roles"]) != "[moderator]" {
		t.Errorf("roles %v, want only the ranked role the session carries", body["roles"])
	}
	avatar, _ := body["avatar"].(map[string]any)
	if avatar["hash"] != acctAvatarHash || !strings.HasPrefix(fmt.Sprint(avatar["url"]), acctCDN+"/ab/ab/"+acctAvatarHash) {
		t.Errorf("avatar %+v", avatar)
	}
	stance, _ := body["content_stance"].(map[string]any)
	if stance["is_adult_confirmed"] != true || stance["nsfw_display"] != "blur" {
		t.Errorf("content_stance %+v", stance)
	}
	if resp.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control %q", resp.Header.Get("Cache-Control"))
	}
}

func TestV1AccountBearer(t *testing.T) {
	f := newAcctFix(t)
	resp, body := f.account(t, "", "acct-bearer")
	if resp.StatusCode != http.StatusOK || body["id"] != strconv.Itoa(acctBearer) {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	if roles, _ := body["roles"].([]any); roles == nil || len(roles) != 0 {
		t.Errorf("roles %v: a Bearer request carries no staff role, whatever the account record says", body["roles"])
	}
	if _, has := body["content_stance"]; !has || body["content_stance"] != nil {
		t.Errorf("content_stance %v: a Bearer request carries no stance", body["content_stance"])
	}
}

func TestV1AccountLegacyStance(t *testing.T) {
	f := newAcctFix(t)
	resp, body := f.account(t, "sess-acct-legacy", "")
	stance, _ := body["content_stance"].(map[string]any)
	if resp.StatusCode != http.StatusOK || stance["is_adult_confirmed"] != false || stance["nsfw_display"] != "hide" {
		t.Errorf("a session without a stored display falls back to hide: %d %+v", resp.StatusCode, body)
	}
	if body["avatar"] != nil {
		t.Errorf("no hash, no avatar: %v", body["avatar"])
	}
}

func TestV1AccountGone(t *testing.T) {
	f := newAcctFix(t)
	resp, body := f.account(t, "sess-acct-gone", "")
	if resp.StatusCode != http.StatusOK || body["id"] != strconv.Itoa(acctGone) {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	if _, has := body["name"]; !has || body["name"] != nil || body["avatar"] != nil {
		t.Errorf("an account the account center no longer knows has name and avatar null: %+v", body)
	}
}

func TestV1AccountRejects(t *testing.T) {
	f := newAcctFix(t)
	resp, body := f.account(t, "", "")
	if resp.StatusCode != http.StatusUnauthorized || body["code"] != "MISSING_CREDENTIAL" {
		t.Errorf("anonymous %d %+v", resp.StatusCode, body)
	}
	resp, body = f.account(t, "no-such-session", "")
	if resp.StatusCode != http.StatusUnauthorized || body["code"] != "INVALID_CREDENTIAL" {
		t.Errorf("stale cookie %d %+v", resp.StatusCode, body)
	}
	resp, body = f.account(t, "", "forged")
	if resp.StatusCode != http.StatusUnauthorized || body["code"] != "INVALID_CREDENTIAL" {
		t.Errorf("bad bearer %d %+v", resp.StatusCode, body)
	}

	down := newAcctFix(t)
	down.failOA.Store(true)
	resp, body = down.account(t, "sess-acct", "")
	if resp.StatusCode != http.StatusServiceUnavailable || body["code"] != "SERVICE_UNAVAILABLE" {
		t.Errorf("account center down %d %+v", resp.StatusCode, body)
	}
}
