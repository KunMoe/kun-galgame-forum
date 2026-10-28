package middleware

import (
	"context"
	stderrors "errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/pkg/role"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

const (
	bearerSeenPrefix = "kungal:bearer-seen:"
	bearerSeenTTL    = 24 * time.Hour
)

type AccessTokenVerifier interface {
	Verify(ctx context.Context, raw string) (*oauth.AccessClaims, error)
}

// FirstSeen stands in for what the web OAuth callback does at login, which a
// Bearer client never passes through: without the local user-state row every
// write that locks it (topic create, check-in) fails.
type FirstSeen func(userID int, roles []string) error

type Bearer struct {
	verifier  AccessTokenVerifier
	rdb       *redis.Client
	firstSeen FirstSeen
}

func NewBearer(verifier AccessTokenVerifier, rdb *redis.Client, firstSeen FirstSeen) *Bearer {
	return &Bearer{verifier: verifier, rdb: rdb, firstSeen: firstSeen}
}

func bearerToken(c fiber.Ctx) (string, bool) {
	scheme, token, _ := strings.Cut(strings.TrimSpace(c.Get(fiber.HeaderAuthorization)), " ")
	if !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	return strings.TrimSpace(token), true
}

func (b *Bearer) resolve(c fiber.Ctx, token string) Identity {
	if u := GetUser(c); u != nil && u.viaBearer {
		return Identity{
			Outcome:     IdentityBearerOK,
			User:        u,
			AccessToken: GetAccessToken(c),
		}
	}
	if b == nil || b.verifier == nil {
		return Identity{Outcome: IdentityBearerInvalid}
	}

	ctx := c.Context()
	claims, err := b.verifier.Verify(ctx, token)
	if err != nil {
		if stderrors.Is(err, oauth.ErrKeysUnavailable) {
			slog.Error("Bearer 校验拉取 JWKS 失败", "error", err)
			return Identity{Outcome: IdentityBearerKeysUnavailable, Err: err}
		}
		return Identity{Outcome: IdentityBearerInvalid, Err: err}
	}

	roles := role.Union(claims.Roles, claims.SiteRoles)
	if err := b.markSeen(ctx, claims.ID, roles); err != nil {
		slog.Error("Bearer 首见用户初始化失败", "user_id", claims.ID, "error", err)
		return Identity{Outcome: IdentityBearerProvisioningFailed, Err: err}
	}

	return Identity{
		Outcome:     IdentityBearerOK,
		User:        BearerUser(claims),
		AccessToken: token,
	}
}

func BearerUser(claims *oauth.AccessClaims) *UserInfo {
	return &UserInfo{
		ID:        claims.ID,
		Sub:       claims.Subject,
		Name:      claims.Name,
		Email:     claims.Email,
		Roles:     role.WithoutStaff(role.Union(claims.Roles, claims.SiteRoles)),
		viaBearer: true,
	}
}

// The trust boost gets the unstripped roles on purpose: it is declared once per
// user and guarded by SETNX, so a stripped declaration from the App would lock
// a moderator out of their boost when they later log in on the web.
func (b *Bearer) markSeen(ctx context.Context, userID int, roles []string) error {
	if b.firstSeen == nil {
		return nil
	}
	key := bearerSeenPrefix + strconv.Itoa(userID)
	fresh, err := b.rdb.SetNX(ctx, key, 1, bearerSeenTTL).Result()
	if err != nil || !fresh {
		return err
	}
	if err := b.firstSeen(userID, roles); err != nil {
		b.rdb.Del(ctx, key)
		return err
	}
	return nil
}
