package middleware

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/pkg/config"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

const (
	sessToken      = "sess-token"
	resolveBody    = "resolver-passthrough"
	oauthTokenOK   = `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600,"token_type":"Bearer"}`
	oauthUserOK    = `{"id":42,"sub":"u-42","name":"alice","email":"a@b.c","roles":["user"],"site_roles":["moderator"]}`
	oauthBanned    = `{"error":"access_denied","error_description":"account banned"}`
	oauthDead      = `{"error":"invalid_token","error_description":"refresh expired"}`
	oauthTransient = `{"error":"server_error","error_description":"blip"}`
)

type identHarness struct {
	mr        *miniredis.Miniredis
	rdb       *redis.Client
	oauth     *oauth.Client
	verifier  AccessTokenVerifier
	firstSeen FirstSeen
	authn     *Authenticator
}

func newIdentHarness(t *testing.T) *identHarness {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: 0})
	return &identHarness{mr: mr, rdb: rdb, verifier: stubVerifier{}}
}

func (h *identHarness) build() {
	h.authn = NewAuthenticator(h.rdb, h.oauth, NewBearer(h.verifier, h.rdb, h.firstSeen))
}

func cookieReq(sessionToken, authorization string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if sessionToken != "" {
		req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: sessionToken})
	}
	if authorization != "" {
		req.Header.Set(fiber.HeaderAuthorization, authorization)
	}
	return req
}

func sampleSession(expired bool) SessionData {
	exp := time.Now().Add(time.Hour).Unix()
	if expired {
		exp = time.Now().Add(-time.Minute).Unix()
	}
	return SessionData{
		UserInfo: UserInfo{
			ID: 42, Sub: "u-42", Name: "alice", Email: "a@b.c",
			Roles: []string{"user", "moderator"},
		},
		OAuthAccessToken:  "old-access",
		OAuthRefreshToken: "old-refresh",
		OAuthExpiresAt:    exp,
	}
}

func putSession(t *testing.T, rdb *redis.Client, token string, s SessionData) {
	t.Helper()
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if err := rdb.Set(context.Background(), SessionKey(token), data, SessionTTL).Err(); err != nil {
		t.Fatal(err)
	}
}

func newOAuthClient(t *testing.T, tokenStatus int, tokenBody string, userStatus int, userBody string) *oauth.Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(tokenStatus)
		if _, err := io.WriteString(w, tokenBody); err != nil {
			panic(err)
		}
	})
	mux.HandleFunc("/oauth/userinfo", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(userStatus)
		if _, err := io.WriteString(w, userBody); err != nil {
			panic(err)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return oauth.NewClient(config.OAuthConfig{
		ServerURL:    srv.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
	})
}

type resolveGot struct {
	id         Identity
	userAfter  *UserInfo
	resp       *http.Response
	bodyBefore []byte
}

func (h *identHarness) resolve(t *testing.T, req *http.Request) resolveGot {
	t.Helper()
	if h.authn == nil {
		h.build()
	}
	var got resolveGot
	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		got.id = h.authn.ResolveIdentity(c)
		got.bodyBefore = append([]byte(nil), c.Response().Body()...)
		got.userAfter = GetUser(c)
		return c.SendString(resolveBody)
	})
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	got.resp = resp
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.bodyBefore) != 0 {
		t.Errorf("resolver wrote body %q", got.bodyBefore)
	}
	if string(body) != resolveBody {
		t.Errorf("response body = %q, want %q (resolver must not write)", body, resolveBody)
	}
	return got
}

func sessionCookie(resp *http.Response) *http.Cookie {
	for _, c := range resp.Cookies() {
		if c.Name == SessionCookieName {
			return c
		}
	}
	return nil
}

func TestResolveIdentity(t *testing.T) {
	t.Run("anonymous", func(t *testing.T) {
		h := newIdentHarness(t)
		got := h.resolve(t, cookieReq("", ""))
		if got.id.Outcome != IdentityAnonymous {
			t.Fatalf("outcome = %s, want %s", got.id.Outcome, IdentityAnonymous)
		}
		if got.id.User != nil || got.id.AccessToken != "" || got.id.Err != nil {
			t.Errorf("identity = %+v", got.id)
		}
		if got.userAfter != nil {
			t.Error("resolver attached Locals")
		}
		if sessionCookie(got.resp) != nil {
			t.Error("unexpected Set-Cookie")
		}
	})

	t.Run("session missing", func(t *testing.T) {
		h := newIdentHarness(t)
		got := h.resolve(t, cookieReq(sessToken, ""))
		if got.id.Outcome != IdentitySessionMissing {
			t.Fatalf("outcome = %s, want %s", got.id.Outcome, IdentitySessionMissing)
		}
		if got.id.Err != redis.Nil {
			t.Errorf("Err = %v, want redis.Nil", got.id.Err)
		}
		if got.userAfter != nil {
			t.Error("resolver attached Locals")
		}
		if h.mr.Exists(SessionKey(sessToken)) || h.mr.Exists("refresh_lock:"+sessToken) {
			t.Error("missing session must not write redis keys")
		}
	})

	t.Run("session missing bad json", func(t *testing.T) {
		h := newIdentHarness(t)
		if err := h.rdb.Set(context.Background(), SessionKey(sessToken), "not-json", SessionTTL).Err(); err != nil {
			t.Fatal(err)
		}
		got := h.resolve(t, cookieReq(sessToken, ""))
		if got.id.Outcome != IdentitySessionMissing {
			t.Fatalf("outcome = %s, want %s", got.id.Outcome, IdentitySessionMissing)
		}
		if got.id.Err == nil {
			t.Error("want unmarshal error")
		}
		if !h.mr.Exists(SessionKey(sessToken)) {
			t.Error("corrupt session must not be deleted")
		}
	})

	t.Run("session store error", func(t *testing.T) {
		h := newIdentHarness(t)
		h.build()
		h.mr.Close()
		got := h.resolve(t, cookieReq(sessToken, ""))
		if got.id.Outcome != IdentitySessionStoreError {
			t.Fatalf("outcome = %s, want %s (not %s)", got.id.Outcome, IdentitySessionStoreError, IdentitySessionMissing)
		}
		if got.id.Err == nil || got.id.Err == redis.Nil {
			t.Errorf("Err = %v, want a non-Nil redis error", got.id.Err)
		}
		if got.userAfter != nil {
			t.Error("resolver attached Locals")
		}
	})

	t.Run("session refresh dead", func(t *testing.T) {
		h := newIdentHarness(t)
		h.oauth = newOAuthClient(t, http.StatusUnauthorized, oauthDead, http.StatusOK, oauthUserOK)
		putSession(t, h.rdb, sessToken, sampleSession(true))
		got := h.resolve(t, cookieReq(sessToken, ""))
		if got.id.Outcome != IdentitySessionRefreshDead {
			t.Fatalf("outcome = %s, want %s", got.id.Outcome, IdentitySessionRefreshDead)
		}
		if got.id.Err == nil {
			t.Error("want underlying oauth error")
		}
		if h.mr.Exists(SessionKey(sessToken)) {
			t.Error("dead refresh must delete the session")
		}
		if h.mr.Exists("refresh_lock:" + sessToken) {
			t.Error("lock must be deleted")
		}
		if got.userAfter != nil {
			t.Error("resolver attached Locals")
		}
	})

	t.Run("session refresh transient", func(t *testing.T) {
		h := newIdentHarness(t)
		h.oauth = newOAuthClient(t, http.StatusInternalServerError, oauthTransient, http.StatusOK, oauthUserOK)
		putSession(t, h.rdb, sessToken, sampleSession(true))
		got := h.resolve(t, cookieReq(sessToken, ""))
		if got.id.Outcome != IdentitySessionRefreshTransient {
			t.Fatalf("outcome = %s, want %s", got.id.Outcome, IdentitySessionRefreshTransient)
		}
		if got.id.Err == nil {
			t.Error("want underlying oauth error")
		}
		if !h.mr.Exists(SessionKey(sessToken)) {
			t.Error("transient refresh must keep the session")
		}
		raw, _ := h.mr.Get(SessionKey(sessToken))
		if !json.Valid([]byte(raw)) || !strings.Contains(raw, `"old-access"`) {
			t.Errorf("session rewritten: %s", raw)
		}
		if h.mr.Exists("refresh_lock:" + sessToken) {
			t.Error("lock must be deleted")
		}
	})

	t.Run("waiter sees the refresher delete a dead session", func(t *testing.T) {
		h := newIdentHarness(t)
		putSession(t, h.rdb, sessToken, sampleSession(true))
		if err := h.rdb.Set(context.Background(), "refresh_lock:"+sessToken, "1", 15*time.Second).Err(); err != nil {
			t.Fatal(err)
		}
		go func() {
			time.Sleep(20 * time.Millisecond)
			h.rdb.Del(context.Background(), SessionKey(sessToken))
			h.rdb.Del(context.Background(), "refresh_lock:"+sessToken)
		}()
		got := h.resolve(t, cookieReq(sessToken, ""))
		if got.id.Outcome != IdentitySessionMissing {
			t.Fatalf("outcome = %s, want %s", got.id.Outcome, IdentitySessionMissing)
		}
		if got.userAfter != nil {
			t.Error("resolver attached Locals")
		}
	})

	t.Run("session refresh transient waitForRefresh", func(t *testing.T) {
		h := newIdentHarness(t)
		putSession(t, h.rdb, sessToken, sampleSession(true))
		if err := h.rdb.Set(context.Background(), "refresh_lock:"+sessToken, "1", 15*time.Second).Err(); err != nil {
			t.Fatal(err)
		}
		go func() {
			time.Sleep(20 * time.Millisecond)
			h.rdb.Del(context.Background(), "refresh_lock:"+sessToken)
		}()
		got := h.resolve(t, cookieReq(sessToken, ""))
		if got.id.Outcome != IdentitySessionRefreshTransient {
			t.Fatalf("outcome = %s, want %s", got.id.Outcome, IdentitySessionRefreshTransient)
		}
		if !h.mr.Exists(SessionKey(sessToken)) {
			t.Error("a waiter must not delete the session")
		}
	})

	t.Run("banned", func(t *testing.T) {
		h := newIdentHarness(t)
		h.oauth = newOAuthClient(t, http.StatusForbidden, oauthBanned, http.StatusOK, oauthUserOK)
		putSession(t, h.rdb, sessToken, sampleSession(true))
		got := h.resolve(t, cookieReq(sessToken, ""))
		if got.id.Outcome != IdentityBanned {
			t.Fatalf("outcome = %s, want %s", got.id.Outcome, IdentityBanned)
		}
		if h.mr.Exists(SessionKey(sessToken)) {
			t.Error("banned refresh must delete the session")
		}
		if h.mr.Exists("refresh_lock:" + sessToken) {
			t.Error("lock must be deleted")
		}
	})

	t.Run("banned userinfo", func(t *testing.T) {
		h := newIdentHarness(t)
		h.oauth = newOAuthClient(t, http.StatusOK, oauthTokenOK, http.StatusForbidden, oauthBanned)
		putSession(t, h.rdb, sessToken, sampleSession(true))
		got := h.resolve(t, cookieReq(sessToken, ""))
		if got.id.Outcome != IdentityBanned {
			t.Fatalf("outcome = %s, want %s", got.id.Outcome, IdentityBanned)
		}
		if h.mr.Exists(SessionKey(sessToken)) {
			t.Error("banned userinfo must delete the session")
		}
	})

	t.Run("session internal error", func(t *testing.T) {
		h := newIdentHarness(t)
		h.oauth = newOAuthClient(t, http.StatusOK, oauthTokenOK, http.StatusOK, oauthUserOK)
		putSession(t, h.rdb, sessToken, sampleSession(true))
		h.build()
		h.authn.marshal = func(any) ([]byte, error) { return nil, stderrors.New("marshal boom") }
		got := h.resolve(t, cookieReq(sessToken, ""))
		if got.id.Outcome != IdentitySessionInternalError {
			t.Fatalf("outcome = %s, want %s", got.id.Outcome, IdentitySessionInternalError)
		}
		if got.id.Err == nil {
			t.Error("want marshal error")
		}
		raw, err := h.mr.Get(SessionKey(sessToken))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(raw, `"old-access"`) {
			t.Errorf("session rewritten on marshal failure: %s", raw)
		}
		if h.mr.Exists("refresh_lock:" + sessToken) {
			t.Error("lock must be deleted")
		}
	})

	t.Run("session ok", func(t *testing.T) {
		h := newIdentHarness(t)
		putSession(t, h.rdb, sessToken, sampleSession(false))
		got := h.resolve(t, cookieReq(sessToken, ""))
		if got.id.Outcome != IdentitySessionOK {
			t.Fatalf("outcome = %s, want %s", got.id.Outcome, IdentitySessionOK)
		}
		if got.userAfter != nil {
			t.Error("resolver attached Locals")
		}
		u := got.id.User
		if u == nil || u.ID != 42 || u.ViaBearer() {
			t.Fatalf("user = %+v", u)
		}
		if !slices.Contains(u.Roles, "moderator") {
			t.Errorf("session roles = %v, staff must not be stripped", u.Roles)
		}
		if got.id.AccessToken != "old-access" {
			t.Errorf("access token = %q", got.id.AccessToken)
		}
		if !h.mr.Exists(sessionRenewPrefix + sessToken) {
			t.Error("want kungal:session-renew: marker")
		}
		if h.mr.Exists("refresh_lock:" + sessToken) {
			t.Error("lock must not remain")
		}
		ck := sessionCookie(got.resp)
		if ck == nil || ck.Value != sessToken || !ck.HttpOnly || ck.Path != "/" {
			t.Errorf("Set-Cookie = %+v", ck)
		}
		if ck != nil && ck.MaxAge != int(SessionTTL.Seconds()) {
			t.Errorf("MaxAge = %d", ck.MaxAge)
		}
	})

	t.Run("session ok refreshed", func(t *testing.T) {
		h := newIdentHarness(t)
		h.oauth = newOAuthClient(t, http.StatusOK, oauthTokenOK, http.StatusOK, oauthUserOK)
		putSession(t, h.rdb, sessToken, sampleSession(true))
		got := h.resolve(t, cookieReq(sessToken, ""))
		if got.id.Outcome != IdentitySessionOK {
			t.Fatalf("outcome = %s, want %s", got.id.Outcome, IdentitySessionOK)
		}
		if got.id.AccessToken != "new-access" {
			t.Errorf("access token = %q, want new-access", got.id.AccessToken)
		}
		if got.id.User == nil || !slices.Contains(got.id.User.Roles, "moderator") {
			t.Errorf("refreshed roles = %v", got.id.User)
		}
		raw, err := h.mr.Get(SessionKey(sessToken))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(raw, `"new-access"`) {
			t.Errorf("redis session = %s", raw)
		}
		if h.mr.Exists("refresh_lock:" + sessToken) {
			t.Error("lock must be deleted after the winner refresh")
		}
		if sessionCookie(got.resp) == nil {
			t.Error("want Set-Cookie on sliding renewal")
		}
	})

	t.Run("bearer ok", func(t *testing.T) {
		h := newIdentHarness(t)
		var seen seenLog
		h.firstSeen = func(uid int, roles []string) error {
			seen.calls = append(seen.calls, roles)
			return nil
		}
		got := h.resolve(t, cookieReq(sessToken, "Bearer "+goodToken))
		if got.id.Outcome != IdentityBearerOK {
			t.Fatalf("outcome = %s, want %s", got.id.Outcome, IdentityBearerOK)
		}
		if got.userAfter != nil {
			t.Error("resolver attached Locals")
		}
		u := got.id.User
		if u == nil || u.ID != 1207 || !u.ViaBearer() || got.id.AccessToken != goodToken {
			t.Fatalf("user = %+v token = %q", u, got.id.AccessToken)
		}
		if slices.Contains(u.Roles, "moderator") || !slices.Contains(u.Roles, "creator") {
			t.Errorf("roles = %v, want staff stripped", u.Roles)
		}
		if u.CanModerate() {
			t.Error("Bearer user must not moderate")
		}
		if !h.mr.Exists(bearerSeenPrefix + "1207") {
			t.Error("want kungal:bearer-seen:1207")
		}
		if h.mr.Exists(SessionKey(sessToken)) {
			t.Error("Bearer path must ignore the cookie")
		}
		if len(seen.calls) != 1 || !slices.Contains(seen.calls[0], "moderator") {
			t.Errorf("first-seen = %v, want unstripped roles once", seen.calls)
		}
		if sessionCookie(got.resp) != nil {
			t.Error("Bearer path must not set the session cookie")
		}
	})

	t.Run("bearer invalid", func(t *testing.T) {
		h := newIdentHarness(t)
		putSession(t, h.rdb, sessToken, sampleSession(false))
		got := h.resolve(t, cookieReq(sessToken, "Bearer expired"))
		if got.id.Outcome != IdentityBearerInvalid {
			t.Fatalf("outcome = %s, want %s", got.id.Outcome, IdentityBearerInvalid)
		}
		if got.id.Err == nil {
			t.Error("want verify error")
		}
		if !h.mr.Exists(SessionKey(sessToken)) {
			t.Error("Bearer path must not touch the cookie session")
		}
		if h.mr.Exists(bearerSeenPrefix + "1207") {
			t.Error("invalid bearer must not mark seen")
		}
	})

	t.Run("bearer invalid closed channel", func(t *testing.T) {
		h := newIdentHarness(t)
		h.verifier = nil
		got := h.resolve(t, cookieReq("", "Bearer "+goodToken))
		if got.id.Outcome != IdentityBearerInvalid {
			t.Fatalf("outcome = %s, want %s", got.id.Outcome, IdentityBearerInvalid)
		}
	})

	t.Run("bearer keys unavailable", func(t *testing.T) {
		h := newIdentHarness(t)
		h.verifier = stubVerifier{err: oauth.ErrKeysUnavailable}
		got := h.resolve(t, cookieReq("", "Bearer "+goodToken))
		if got.id.Outcome != IdentityBearerKeysUnavailable {
			t.Fatalf("outcome = %s, want %s", got.id.Outcome, IdentityBearerKeysUnavailable)
		}
		if !stderrors.Is(got.id.Err, oauth.ErrKeysUnavailable) {
			t.Errorf("Err = %v", got.id.Err)
		}
	})

	t.Run("bearer provisioning failed", func(t *testing.T) {
		h := newIdentHarness(t)
		h.firstSeen = func(int, []string) error { return stderrors.New("db down") }
		got := h.resolve(t, cookieReq("", "Bearer "+goodToken))
		if got.id.Outcome != IdentityBearerProvisioningFailed {
			t.Fatalf("outcome = %s, want %s", got.id.Outcome, IdentityBearerProvisioningFailed)
		}
		if got.id.Err == nil {
			t.Error("want firstSeen error")
		}
		if h.mr.Exists(bearerSeenPrefix + "1207") {
			t.Error("failed first-seen must not leave the seen key")
		}
	})
}

func TestResolveIdentityBearerShortcut(t *testing.T) {
	h := newIdentHarness(t)
	h.verifier = stubVerifier{err: stderrors.New("verify must not run")}
	h.build()
	pre := &UserInfo{ID: 9, Roles: []string{"admin"}, viaBearer: true}
	var id Identity
	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		c.Locals(string(UserInfoKey), pre)
		c.Locals(string(OAuthAccessTokenKey), "pre-set")
		id = h.authn.ResolveIdentity(c)
		return c.SendString(resolveBody)
	})
	resp, err := app.Test(cookieReq("", "Bearer ignored"))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != resolveBody {
		t.Fatalf("body = %q", body)
	}
	if id.Outcome != IdentityBearerOK || id.User != pre || id.AccessToken != "pre-set" {
		t.Fatalf("identity = %+v", id)
	}
	if !slices.Contains(id.User.Roles, "admin") {
		t.Error("shortcut must keep the already-attached roles")
	}
}

func TestNonBearerAuthorizationUsesCookiePath(t *testing.T) {
	h := newIdentHarness(t)
	got := h.resolve(t, cookieReq("", "Basic Zm9vOmJhcg=="))
	if got.id.Outcome != IdentityAnonymous {
		t.Fatalf("outcome = %s, want %s", got.id.Outcome, IdentityAnonymous)
	}
}
