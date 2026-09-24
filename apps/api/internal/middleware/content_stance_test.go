package middleware

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func stanceHarness(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	return mr, redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: 0})
}

func storeSession(t *testing.T, rdb *redis.Client, token string, session SessionData) {
	t.Helper()
	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("marshal session: %v", err)
	}
	if err := rdb.Set(context.Background(), SessionKey(token), data, SessionTTL).Err(); err != nil {
		t.Fatalf("seed session: %v", err)
	}
}

func TestSetSessionContentStance(t *testing.T) {
	mr, rdb := stanceHarness(t)
	storeSession(t, rdb, "tok", SessionData{
		UserInfo:         UserInfo{ID: 7, Name: "alice", Roles: []string{"user"}, NSFWDisplay: "blur"},
		OAuthAccessToken: "access",
		OAuthExpiresAt:   time.Now().Add(time.Hour).Unix(),
	})
	before := mr.TTL(SessionKey("tok"))

	if err := SetSessionContentStance(context.Background(), rdb, "tok", true, "show"); err != nil {
		t.Fatalf("write-through: %v", err)
	}

	raw, err := rdb.Get(context.Background(), SessionKey("tok")).Result()
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	var got SessionData
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !got.AdultConfirmed || got.NSFWDisplay != "show" {
		t.Fatalf("claims not written through: %+v", got.UserInfo)
	}
	if got.Name != "alice" || got.OAuthAccessToken != "access" {
		t.Fatalf("write-through clobbered the rest of the session: %+v", got)
	}
	// KeepTTL, not SessionTTL: rewriting the whole 90-day window here would let
	// a preference change silently extend a session that was about to lapse.
	if after := mr.TTL(SessionKey("tok")); after != before {
		t.Fatalf("ttl moved from %v to %v", before, after)
	}
}

func TestSetSessionContentStanceMissingSession(t *testing.T) {
	_, rdb := stanceHarness(t)
	if err := SetSessionContentStance(context.Background(), rdb, "nope", true, "show"); err == nil {
		t.Fatal("expected an error when the session is gone")
	}
}
