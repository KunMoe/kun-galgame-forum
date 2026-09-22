package middleware

import (
	"encoding/json"
	"fmt"
	"time"

	"kun-galgame-api/pkg/content"
	"kun-galgame-api/pkg/errors"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

type IdentityOutcome int

const (
	IdentityAnonymous IdentityOutcome = iota
	IdentitySessionMissing
	IdentitySessionStoreError
	IdentitySessionRefreshDead
	IdentitySessionRefreshTransient
	IdentityBanned
	IdentitySessionInternalError
	IdentitySessionOK
	IdentityBearerOK
	IdentityBearerInvalid
	IdentityBearerKeysUnavailable
	IdentityBearerProvisioningFailed
)

func (o IdentityOutcome) String() string {
	switch o {
	case IdentityAnonymous:
		return "anonymous"
	case IdentitySessionMissing:
		return "session missing"
	case IdentitySessionStoreError:
		return "session store error"
	case IdentitySessionRefreshDead:
		return "session refresh dead"
	case IdentitySessionRefreshTransient:
		return "session refresh transient"
	case IdentityBanned:
		return "banned"
	case IdentitySessionInternalError:
		return "session internal error"
	case IdentitySessionOK:
		return "session ok"
	case IdentityBearerOK:
		return "bearer ok"
	case IdentityBearerInvalid:
		return "bearer invalid"
	case IdentityBearerKeysUnavailable:
		return "bearer keys unavailable"
	case IdentityBearerProvisioningFailed:
		return "bearer provisioning failed"
	default:
		return fmt.Sprintf("IdentityOutcome(%d)", int(o))
	}
}

type Identity struct {
	Outcome     IdentityOutcome
	User        *UserInfo
	AccessToken string
	Err         error
}

func (id Identity) OK() bool {
	return id.Outcome == IdentitySessionOK || id.Outcome == IdentityBearerOK
}

func AttachIdentity(c fiber.Ctx, id Identity) {
	c.Locals(string(UserInfoKey), id.User)
	c.Locals(string(OAuthAccessTokenKey), id.AccessToken)
	if id.User != nil && !id.User.viaBearer {
		content.Attach(c, id.User.ContentStance())
	}
}

func (id Identity) legacyError() *errors.AppError {
	switch id.Outcome {
	case IdentityBanned:
		return errors.ErrAccountBanned()
	case IdentitySessionInternalError:
		return errors.ErrInternal("服务器内部错误")
	case IdentityBearerKeysUnavailable:
		return errors.ErrInternal("认证服务暂不可用, 请稍后重试")
	case IdentityBearerProvisioningFailed:
		return errors.ErrInternal("初始化用户状态失败")
	default:
		return errors.ErrAuthExpired()
	}
}

func (a *Authenticator) ResolveIdentity(c fiber.Ctx) Identity {
	if token, ok := bearerToken(c); ok {
		return a.bearer.resolve(c, token)
	}
	return a.resolveSession(c)
}

func (a *Authenticator) resolveSession(c fiber.Ctx) Identity {
	rdb, oauthClient := a.rdb, a.oauthClient

	token := c.Cookies(SessionCookieName)
	if token == "" {
		return Identity{Outcome: IdentityAnonymous}
	}

	ctx := c.Context()
	val, err := rdb.Get(ctx, SessionKey(token)).Result()
	if err != nil {
		if err == redis.Nil {
			return Identity{Outcome: IdentitySessionMissing, Err: err}
		}
		return Identity{Outcome: IdentitySessionStoreError, Err: err}
	}

	var session SessionData
	if err := json.Unmarshal([]byte(val), &session); err != nil {
		return Identity{Outcome: IdentitySessionMissing, Err: err}
	}

	const refreshSkew = 30 * time.Second
	needsRefresh := session.OAuthExpiresAt > 0 &&
		time.Now().Add(refreshSkew).Unix() > session.OAuthExpiresAt
	if needsRefresh {
		lockKey := "refresh_lock:" + token
		locked, _ := rdb.SetNX(ctx, lockKey, "1", 15*time.Second).Result()
		if locked {
			outcome, rerr := refreshSession(ctx, rdb, oauthClient, token, &session, a.marshal)
			rdb.Del(ctx, lockKey)
			if outcome != IdentitySessionOK {
				return Identity{Outcome: outcome, Err: rerr}
			}
		} else {
			if err := waitForRefresh(ctx, rdb, lockKey, token, &session); err != nil {
				return Identity{Outcome: IdentitySessionRefreshTransient, Err: err}
			}
		}
	}

	renewSlidingSession(c, rdb, token)
	return Identity{
		Outcome:     IdentitySessionOK,
		User:        &session.UserInfo,
		AccessToken: session.OAuthAccessToken,
	}
}
