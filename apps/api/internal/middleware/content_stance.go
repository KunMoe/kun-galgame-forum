package middleware

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

// SetSessionContentStance puts a write that already succeeded upstream into the
// session copy of the claims. Without it /account keeps reporting the old
// stance until the access token expires and refreshSession re-reads userinfo,
// which is up to fifteen minutes after they changed the setting.
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
