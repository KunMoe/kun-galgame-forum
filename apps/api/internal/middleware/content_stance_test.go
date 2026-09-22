package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kun-galgame-api/pkg/content"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
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

func TestContentStanceMiddleware(t *testing.T) {
	for _, tc := range []struct {
		why        string
		session    *SessionData
		cookie     string
		authHeader string
		wantOK     bool
		want       content.Stance
	}{
		{
			why:     "an attested reader who picked show",
			session: &SessionData{UserInfo: UserInfo{ID: 1, AdultConfirmed: true, NSFWDisplay: "show"}},
			cookie:  "tok", wantOK: true, want: content.StanceShow,
		},
		{
			why:     "the backfilled default folds to hide, not blur",
			session: &SessionData{UserInfo: UserInfo{ID: 1, NSFWDisplay: "blur"}},
			cookie:  "tok", wantOK: true, want: content.StanceHide,
		},
		{
			why:     "a session written before this wave carries no claims",
			session: &SessionData{UserInfo: UserInfo{ID: 1}},
			cookie:  "tok", wantOK: true, want: content.StanceHide,
		},
		{why: "anonymous", cookie: "", wantOK: false},
		{why: "a cookie with no session behind it", cookie: "ghost", wantOK: false},
		{
			why:     "a Bearer request skips the session lane even with a cookie",
			session: &SessionData{UserInfo: UserInfo{ID: 1, AdultConfirmed: true, NSFWDisplay: "show"}},
			cookie:  "tok", authHeader: "Bearer whatever", wantOK: false,
		},
	} {
		_, rdb := stanceHarness(t)
		if tc.session != nil {
			storeSession(t, rdb, "tok", *tc.session)
		}

		app := fiber.New()
		app.Use(ContentStance(rdb, nil))
		var got content.Stance
		var ok bool
		app.Get("/", func(c fiber.Ctx) error {
			got, ok = content.FromCtx(c)
			return nil
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if tc.cookie != "" {
			req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: tc.cookie})
		}
		if tc.authHeader != "" {
			req.Header.Set(fiber.HeaderAuthorization, tc.authHeader)
		}
		if _, err := app.Test(req); err != nil {
			t.Fatalf("%s: %v", tc.why, err)
		}
		if ok != tc.wantOK || (tc.wantOK && got != tc.want) {
			t.Errorf("%s: stance=%q ok=%v, want %q/%v", tc.why, got, ok, tc.want, tc.wantOK)
		}
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
