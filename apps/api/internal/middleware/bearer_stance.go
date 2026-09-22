package middleware

import (
	"context"
	"log/slog"
	"time"

	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/pkg/content"

	"github.com/redis/go-redis/v9"
)

const (
	bearerStancePrefix = "kungal:bearer-stance:"
	bearerStanceTTL    = 5 * time.Minute
)

type UserInfoFetcher interface {
	FetchUserInfo(accessToken string) (*oauth.UserInfo, error)
}

// A Bearer request carries no content preference of its own, so its stance is
// the account's, read from userinfo. The cache is what makes that affordable:
// without it every public read the App makes costs an upstream round trip.
type BearerStance struct {
	verifier AccessTokenVerifier
	oauth    UserInfoFetcher
	rdb      *redis.Client
}

func NewBearerStance(
	verifier AccessTokenVerifier,
	userinfo UserInfoFetcher,
	rdb *redis.Client,
) *BearerStance {
	return &BearerStance{verifier: verifier, oauth: userinfo, rdb: rdb}
}

// Every failure resolves to hide — an unverifiable token, an OP that is down or
// slow, an unreadable cache. The App must not be shown adult content because a
// network call did not come back. A failed fetch is deliberately not cached, so
// one blip does not pin a reader to SFW for the whole TTL.
func (s *BearerStance) Resolve(ctx context.Context, token string) content.Stance {
	claims, err := s.verifier.Verify(ctx, token)
	if err != nil || claims.Subject == "" {
		return content.StanceHide
	}

	key := bearerStancePrefix + claims.Subject
	if cached, cerr := s.rdb.Get(ctx, key).Result(); cerr == nil {
		return content.ParseStance(cached)
	}

	info, ferr := s.oauth.FetchUserInfo(token)
	if ferr != nil {
		slog.Warn("Bearer 读取账号内容分级失败, 本次按 SFW 过滤", "error", ferr)
		return content.StanceHide
	}

	stance := content.Fold(info.AdultConfirmed, info.NSFWDisplay)
	if serr := s.rdb.Set(ctx, key, string(stance), bearerStanceTTL).Err(); serr != nil {
		slog.Warn("Bearer 内容分级缓存写入失败, 下次仍会回源", "error", serr)
	}
	return stance
}
