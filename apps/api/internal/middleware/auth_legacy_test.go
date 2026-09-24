package middleware

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"
	"testing"
	"time"

	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/pkg/errors"

	"github.com/gofiber/fiber/v3"
)

type legacyGot struct {
	status  int
	nextRan bool
	env     echoed
}

func (h *identHarness) callLegacy(t *testing.T, req *http.Request) legacyGot {
	t.Helper()
	if h.authn == nil {
		h.build()
	}
	var got legacyGot
	next := func(c fiber.Ctx) error {
		got.nextRan = true
		u := GetUser(c)
		if u == nil {
			return c.JSON(echoed{Anon: true})
		}
		return c.JSON(echoed{
			ID: u.ID, Roles: u.Roles, Bearer: u.ViaBearer(),
			Token: GetAccessToken(c), Moderate: u.CanModerate(),
		})
	}
	app := fiber.New()
	app.Get("/", h.authn.Auth(), next)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	got.status = resp.StatusCode
	if err := json.Unmarshal(body, &got.env); err != nil {
		t.Fatalf("body %s: %v", body, err)
	}
	return got
}

func TestLegacyAuthParity(t *testing.T) {
	expired := errors.ErrAuthExpired()
	banned := errors.ErrAccountBanned()
	internalSess := errors.ErrInternal("服务器内部错误")
	keysDown := errors.ErrInternal("认证服务暂不可用, 请稍后重试")
	provision := errors.ErrInternal("初始化用户状态失败")

	type want struct {
		status  int
		code    int
		message string
		next    bool
	}
	authFail := func(e *errors.AppError) want {
		return want{status: e.StatusCode, code: e.Code, message: e.Message, next: false}
	}
	okAuth := want{status: http.StatusOK, next: true}

	for _, tc := range []struct {
		name  string
		setup func(*testing.T) (*identHarness, *http.Request)
		auth  want
	}{
		{
			name: "anonymous",
			setup: func(t *testing.T) (*identHarness, *http.Request) {
				return newIdentHarness(t), cookieReq("", "")
			},
			auth: authFail(expired),
		},
		{
			name: "session missing",
			setup: func(t *testing.T) (*identHarness, *http.Request) {
				return newIdentHarness(t), cookieReq(sessToken, "")
			},
			auth: authFail(expired),
		},
		{
			name: "session store error",
			setup: func(t *testing.T) (*identHarness, *http.Request) {
				h := newIdentHarness(t)
				h.build()
				h.mr.Close()
				return h, cookieReq(sessToken, "")
			},
			auth: authFail(expired),
		},
		{
			name: "session refresh dead",
			setup: func(t *testing.T) (*identHarness, *http.Request) {
				h := newIdentHarness(t)
				h.oauth = newOAuthClient(t, http.StatusUnauthorized, oauthDead, http.StatusOK, oauthUserOK)
				putSession(t, h.rdb, sessToken, sampleSession(true))
				return h, cookieReq(sessToken, "")
			},
			auth: authFail(expired),
		},
		{
			name: "session refresh transient",
			setup: func(t *testing.T) (*identHarness, *http.Request) {
				h := newIdentHarness(t)
				h.oauth = newOAuthClient(t, http.StatusInternalServerError, oauthTransient, http.StatusOK, oauthUserOK)
				putSession(t, h.rdb, sessToken, sampleSession(true))
				return h, cookieReq(sessToken, "")
			},
			auth: authFail(expired),
		},
		{
			name: "banned",
			setup: func(t *testing.T) (*identHarness, *http.Request) {
				h := newIdentHarness(t)
				h.oauth = newOAuthClient(t, http.StatusForbidden, oauthBanned, http.StatusOK, oauthUserOK)
				putSession(t, h.rdb, sessToken, sampleSession(true))
				return h, cookieReq(sessToken, "")
			},
			auth: authFail(banned),
		},
		{
			name: "session internal error",
			setup: func(t *testing.T) (*identHarness, *http.Request) {
				h := newIdentHarness(t)
				h.oauth = newOAuthClient(t, http.StatusOK, oauthTokenOK, http.StatusOK, oauthUserOK)
				putSession(t, h.rdb, sessToken, sampleSession(true))
				h.build()
				h.authn.marshal = func(any) ([]byte, error) { return nil, stderrors.New("marshal boom") }
				return h, cookieReq(sessToken, "")
			},
			auth: authFail(internalSess),
		},
		{
			name: "session ok",
			setup: func(t *testing.T) (*identHarness, *http.Request) {
				h := newIdentHarness(t)
				putSession(t, h.rdb, sessToken, sampleSession(false))
				return h, cookieReq(sessToken, "")
			},
			auth: okAuth,
		},
		{
			name: "bearer ok",
			setup: func(t *testing.T) (*identHarness, *http.Request) {
				return newIdentHarness(t), cookieReq("", "Bearer "+goodToken)
			},
			auth: okAuth,
		},
		{
			name: "bearer invalid",
			setup: func(t *testing.T) (*identHarness, *http.Request) {
				return newIdentHarness(t), cookieReq("", "Bearer expired")
			},
			auth: authFail(expired),
		},
		{
			name: "bearer keys unavailable",
			setup: func(t *testing.T) (*identHarness, *http.Request) {
				h := newIdentHarness(t)
				h.verifier = stubVerifier{err: oauth.ErrKeysUnavailable}
				return h, cookieReq("", "Bearer "+goodToken)
			},
			auth: authFail(keysDown),
		},
		{
			name: "bearer provisioning failed",
			setup: func(t *testing.T) (*identHarness, *http.Request) {
				h := newIdentHarness(t)
				h.firstSeen = func(int, []string) error { return stderrors.New("db down") }
				return h, cookieReq("", "Bearer "+goodToken)
			},
			auth: authFail(provision),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			exp := tc.auth
			h, req := tc.setup(t)
			got := h.callLegacy(t, req)
			if got.status != exp.status || got.nextRan != exp.next {
				t.Errorf("status %d next %v, want %d / %v", got.status, got.nextRan, exp.status, exp.next)
			}
			if exp.message != "" && (got.env.Code != exp.code || got.env.Message != exp.message) {
				t.Errorf("envelope code=%d message=%q, want %d / %q", got.env.Code, got.env.Message, exp.code, exp.message)
			}
			if exp.next && exp.message == "" {
				switch tc.name {
				case "session ok":
					if got.env.Anon || got.env.ID != 42 || got.env.Bearer {
						t.Errorf("user %+v", got.env)
					}
				case "bearer ok":
					if got.env.Anon || got.env.ID != 1207 || !got.env.Bearer || got.env.Moderate {
						t.Errorf("user %+v", got.env)
					}
				default:
					if !got.env.Anon || got.env.ID != 0 {
						t.Errorf("want anonymous, got %+v", got.env)
					}
				}
			}
		})
	}
}

func TestLegacySessionMissingUnmarshal(t *testing.T) {
	h := newIdentHarness(t)
	if err := h.rdb.Set(context.Background(), SessionKey(sessToken), "{", SessionTTL).Err(); err != nil {
		t.Fatal(err)
	}
	req := cookieReq(sessToken, "")
	auth := h.callLegacy(t, req)
	if auth.status != http.StatusUnauthorized || auth.env.Code != errors.CodeAuth || auth.env.Message != "用户登录失效" || auth.nextRan {
		t.Errorf("Auth: %+v %+v", auth.status, auth.env)
	}
}

func TestLegacyWaitForRefreshGaveUp(t *testing.T) {
	h := newIdentHarness(t)
	putSession(t, h.rdb, sessToken, sampleSession(true))
	if err := h.rdb.Set(context.Background(), "refresh_lock:"+sessToken, "1", 15*time.Second).Err(); err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(20 * time.Millisecond)
		h.rdb.Del(context.Background(), SessionKey(sessToken))
	}()
	auth := h.callLegacy(t, cookieReq(sessToken, ""))
	if auth.status != http.StatusUnauthorized || auth.env.Code != errors.CodeAuth || auth.nextRan {
		t.Errorf("Auth wait-for-refresh: status %d code %d next %v msg %q", auth.status, auth.env.Code, auth.nextRan, auth.env.Message)
	}
}
