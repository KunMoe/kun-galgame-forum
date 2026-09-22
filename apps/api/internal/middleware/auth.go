package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/pkg/content"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/role"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

type contextKey string

const (
	UserInfoKey         contextKey = "userInfo"
	OAuthAccessTokenKey contextKey = "oauthAccessToken"
)

const (
	SessionCookieName  = "kungal_session"
	SessionPrefix      = "kungal:session:v2:"
	SessionTTL         = 90 * 24 * time.Hour
	sessionRenewPrefix = "kungal:session-renew:"
)

var SecureCookies = true

func SessionKey(token string) string { return SessionPrefix + token }

type UserInfo struct {
	ID    int      `json:"id"`
	Sub   string   `json:"sub"`
	Name  string   `json:"name"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`

	AdultConfirmed bool   `json:"adult_confirmed"`
	NSFWDisplay    string `json:"nsfw_display"`

	viaBearer bool
}

func (u *UserInfo) ContentStance() content.Stance {
	return content.Fold(u.AdultConfirmed, u.NSFWDisplay)
}

// Staff powers are never reachable through the Bearer channel, whatever the
// token's roles or the user's permission overrides say. Check capabilities
// through these methods, not perm.CanUser / role.Can* on u.Roles, or that
// guarantee is lost; bearer_guard_test.go enforces it.
func (u *UserInfo) ViaBearer() bool { return u.viaBearer }

func (u *UserInfo) Can(p perm.Permission) bool {
	return !u.viaBearer && perm.CanUser(u.ID, u.Roles, p)
}

func (u *UserInfo) CanModerate() bool {
	return !u.viaBearer && role.CanModerate(u.Roles)
}

func (u *UserInfo) CanAdminister() bool {
	return !u.viaBearer && role.CanAdminister(u.Roles)
}

type SessionData struct {
	UserInfo
	OAuthAccessToken  string `json:"oauth_access_token"`
	OAuthRefreshToken string `json:"oauth_refresh_token"`
	OAuthExpiresAt    int64  `json:"oauth_expires_at"`
}

type Authenticator struct {
	rdb         *redis.Client
	oauthClient *oauth.Client
	bearer      *Bearer
	marshal     func(any) ([]byte, error)
}

func NewAuthenticator(rdb *redis.Client, oauthClient *oauth.Client, bearer *Bearer) *Authenticator {
	return &Authenticator{rdb: rdb, oauthClient: oauthClient, bearer: bearer}
}

func (a *Authenticator) Auth() fiber.Handler {
	return func(c fiber.Ctx) error {
		id := a.ResolveIdentity(c)
		if id.OK() {
			AttachIdentity(c, id)
			return c.Next()
		}
		return response.Error(c, id.legacyError())
	}
}

// A Bearer that fails verification is refused even here: dropping to anonymous
// would hide the expiry from the App, which only refreshes on a 401.
func (a *Authenticator) OptionalAuth() fiber.Handler {
	return func(c fiber.Ctx) error {
		id := a.ResolveIdentity(c)
		switch id.Outcome {
		case IdentitySessionOK, IdentityBearerOK:
			AttachIdentity(c, id)
			return c.Next()
		case IdentityBearerInvalid, IdentityBearerKeysUnavailable, IdentityBearerProvisioningFailed:
			return response.Error(c, id.legacyError())
		default:
			return c.Next()
		}
	}
}

func GetUser(c fiber.Ctx) *UserInfo {
	info, ok := c.Locals(string(UserInfoKey)).(*UserInfo)
	if !ok {
		return nil
	}
	return info
}

func MustGetUser(c fiber.Ctx) (*UserInfo, *errors.AppError) {
	info := GetUser(c)
	if info == nil {
		return nil, errors.ErrAuthExpired()
	}
	return info, nil
}

func GetAccessToken(c fiber.Ctx) string {
	tok, _ := c.Locals(string(OAuthAccessTokenKey)).(string)
	return tok
}

// Hot path: this runs on every authenticated request, so it must handle
// concurrent expiry without N parallel refresh round-trips (SETNX single-flight;
// the winner refreshes, losers poll for the published session) and it must
// survive transient OAuth failures without logging anyone out. On refresh
// failure THIS request gets 205 but the session is left intact so the next one
// retries — many 205s during an OAuth blip beat auto-logging-out every active
// user, and only a permanently-invalid refresh token keeps failing.
func refreshSession(
	ctx context.Context,
	rdb *redis.Client,
	oauthClient *oauth.Client,
	token string,
	session *SessionData,
	marshal func(any) ([]byte, error),
) (IdentityOutcome, error) {
	refreshed, err := oauthClient.RefreshOAuthToken(session.OAuthRefreshToken)
	if err != nil {
		switch {
		case oauth.IsBanned(err):
			slog.Warn("OAuth 刷新返回账号封禁", "error", err)
			rdb.Del(ctx, SessionKey(token))
			return IdentityBanned, err
		case oauth.IsRefreshTokenDead(err):
			slog.Warn("OAuth refresh_token 不可恢复, 清除 session", "error", err)
			rdb.Del(ctx, SessionKey(token))
			return IdentitySessionRefreshDead, err
		default:
			slog.Warn("OAuth token 刷新失败 (保留 session, 留给下次请求重试)",
				"error", err)
			return IdentitySessionRefreshTransient, err
		}
	}
	session.OAuthAccessToken = refreshed.AccessToken
	session.OAuthRefreshToken = refreshed.RefreshToken
	session.OAuthExpiresAt = time.Now().Unix() + int64(refreshed.ExpiresIn)

	if info, uErr := oauthClient.FetchUserInfo(refreshed.AccessToken); uErr == nil {
		session.Roles = role.Union(info.Roles, info.SiteRoles)
		session.AdultConfirmed = info.AdultConfirmed
		session.NSFWDisplay = info.NSFWDisplay
	} else if oauth.IsBanned(uErr) {
		slog.Warn("刷新后 userinfo 返回账号封禁, 清除 session", "error", uErr)
		rdb.Del(ctx, SessionKey(token))
		return IdentityBanned, uErr
	} else {
		slog.Warn("刷新后拉取 userinfo 失败, 保留旧 roles", "error", uErr)
	}

	if marshal == nil {
		marshal = json.Marshal
	}
	data, mErr := marshal(session)
	if mErr != nil {
		slog.Error("序列化 session 失败", "error", mErr)
		return IdentitySessionInternalError, mErr
	}
	rdb.Set(ctx, SessionKey(token), data, SessionTTL)
	return IdentitySessionOK, nil
}

func renewSlidingSession(c fiber.Ctx, rdb *redis.Client, token string) {
	ctx := c.Context()
	if ok, _ := rdb.SetNX(ctx, sessionRenewPrefix+token, "1", SessionTTL/2).Result(); !ok {
		return
	}
	rdb.Expire(ctx, SessionKey(token), SessionTTL)
	c.Cookie(&fiber.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		MaxAge:   int(SessionTTL.Seconds()),
		HTTPOnly: true,
		Secure:   SecureCookies,
		SameSite: "Lax",
		Path:     "/",
	})
}

func waitForRefresh(
	ctx context.Context,
	rdb *redis.Client,
	lockKey, token string,
	session *SessionData,
) *errors.AppError {
	deadline := time.Now().Add(12 * time.Second)
	for {
		time.Sleep(150 * time.Millisecond)

		val, err := rdb.Get(ctx, SessionKey(token)).Result()
		if err != nil {
			return errors.ErrAuthExpired()
		}
		if uErr := json.Unmarshal([]byte(val), session); uErr != nil {
			return errors.ErrAuthExpired()
		}

		if session.OAuthExpiresAt > time.Now().Unix() {
			return nil
		}

		exists, _ := rdb.Exists(ctx, lockKey).Result()
		if exists == 0 {
			return errors.ErrAuthExpired()
		}

		if time.Now().After(deadline) {
			return errors.ErrAuthExpired()
		}
	}
}
