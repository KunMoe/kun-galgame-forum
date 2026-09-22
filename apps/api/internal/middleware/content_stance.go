package middleware

import (
	"context"
	"encoding/json"

	"kun-galgame-api/pkg/content"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

// Most of the handlers that call utils.IsSFW hang off public routes carrying no
// auth middleware at all, so the stance AttachIdentity records would never
// reach them and a signed-in reader would keep being filtered by the legacy
// cookie there. This runs on the whole /api group instead. It is deliberately a
// read-only Redis GET with no token refresh and no sliding renewal: it must not
// become a second copy of the session hot path in auth.go.
func ContentStance(rdb *redis.Client) fiber.Handler {
	return func(c fiber.Ctx) error {
		if _, ok := bearerToken(c); ok {
			return c.Next()
		}
		token := c.Cookies(SessionCookieName)
		if token == "" {
			return c.Next()
		}
		val, err := rdb.Get(c.Context(), SessionKey(token)).Result()
		if err != nil {
			return c.Next()
		}
		var session SessionData
		if err := json.Unmarshal([]byte(val), &session); err != nil {
			return c.Next()
		}
		content.Attach(c, session.ContentStance())
		return c.Next()
	}
}

// SetSessionContentStance puts a write that already succeeded upstream into the
// session copy of the claims. Without it the reader keeps being filtered by the
// old stance until the access token expires and refreshSession re-reads
// userinfo, which is up to fifteen minutes after they changed the setting.
func SetSessionContentStance(
	ctx context.Context,
	rdb *redis.Client,
	token string,
	adultConfirmed bool,
	nsfwDisplay string,
) error {
	key := SessionKey(token)
	val, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	var session SessionData
	if err := json.Unmarshal([]byte(val), &session); err != nil {
		return err
	}
	session.AdultConfirmed = adultConfirmed
	session.NSFWDisplay = nsfwDisplay
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return rdb.Set(ctx, key, data, redis.KeepTTL).Err()
}
