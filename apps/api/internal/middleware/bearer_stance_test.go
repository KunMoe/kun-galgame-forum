package middleware

import (
	"context"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/pkg/content"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

const stanceToken = "app-access-token"

type subjectVerifier struct {
	sub string
	err error
}

func (v subjectVerifier) Verify(_ context.Context, raw string) (*oauth.AccessClaims, error) {
	if v.err != nil {
		return nil, v.err
	}
	if raw != stanceToken {
		return nil, stderrors.New("bad token")
	}
	return &oauth.AccessClaims{
		ID:               1207,
		RegisteredClaims: jwt.RegisteredClaims{Subject: v.sub},
	}, nil
}

type countingUserInfo struct {
	info  *oauth.UserInfo
	err   error
	calls int
}

func (f *countingUserInfo) FetchUserInfo(string) (*oauth.UserInfo, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.info, nil
}

func userinfo(nsfwDisplay string) *oauth.UserInfo {
	return &oauth.UserInfo{ID: 1207, Sub: "u-1207", AdultConfirmed: true, NSFWDisplay: nsfwDisplay}
}

func TestBearerStanceResolve(t *testing.T) {
	for _, tc := range []struct {
		why      string
		verifier AccessTokenVerifier
		fetcher  *countingUserInfo
		want     content.Stance
	}{
		{
			why:      "show reaches the App as show",
			verifier: subjectVerifier{sub: "u-1207"},
			fetcher:  &countingUserInfo{info: userinfo("show")},
			want:     content.StanceShow,
		},
		{
			why:      "blur reaches the App as blur",
			verifier: subjectVerifier{sub: "u-1207"},
			fetcher:  &countingUserInfo{info: userinfo("blur")},
			want:     content.StanceBlur,
		},
		{
			why:      "hide is hide",
			verifier: subjectVerifier{sub: "u-1207"},
			fetcher:  &countingUserInfo{info: userinfo("hide")},
			want:     content.StanceHide,
		},
		{
			why:      "an OP that will not answer must not uncensor the App",
			verifier: subjectVerifier{sub: "u-1207"},
			fetcher:  &countingUserInfo{err: stderrors.New("dial tcp: i/o timeout")},
			want:     content.StanceHide,
		},
		{
			why:      "an unverifiable token never reaches upstream",
			verifier: subjectVerifier{err: stderrors.New("bad signature")},
			fetcher:  &countingUserInfo{info: userinfo("show")},
			want:     content.StanceHide,
		},
		{
			why:      "a token with no sub has nothing to key the cache on",
			verifier: subjectVerifier{sub: ""},
			fetcher:  &countingUserInfo{info: userinfo("show")},
			want:     content.StanceHide,
		},
	} {
		_, rdb := stanceHarness(t)
		s := NewBearerStance(tc.verifier, tc.fetcher, rdb)
		if got := s.Resolve(context.Background(), stanceToken); got != tc.want {
			t.Errorf("%s: stance = %q, want %q", tc.why, got, tc.want)
		}
	}
}

func TestBearerStanceCachesPerSubject(t *testing.T) {
	_, rdb := stanceHarness(t)
	fetcher := &countingUserInfo{info: userinfo("show")}
	s := NewBearerStance(subjectVerifier{sub: "u-1207"}, fetcher, rdb)

	for i := range 3 {
		if got := s.Resolve(context.Background(), stanceToken); got != content.StanceShow {
			t.Fatalf("call %d: stance = %q", i, got)
		}
	}
	if fetcher.calls != 1 {
		t.Fatalf("userinfo called %d times, want 1", fetcher.calls)
	}
}

// A failed fetch is not cached: pinning a reader to SFW for the whole TTL
// because one call timed out outlives the outage that caused it.
func TestBearerStanceDoesNotCacheAFailure(t *testing.T) {
	_, rdb := stanceHarness(t)
	fetcher := &countingUserInfo{err: stderrors.New("blip")}
	s := NewBearerStance(subjectVerifier{sub: "u-1207"}, fetcher, rdb)

	if got := s.Resolve(context.Background(), stanceToken); got != content.StanceHide {
		t.Fatalf("stance = %q", got)
	}
	fetcher.err, fetcher.info = nil, userinfo("show")
	if got := s.Resolve(context.Background(), stanceToken); got != content.StanceShow {
		t.Fatalf("the blip stuck: stance = %q", got)
	}
}

// X-Kungal-Nsfw was specified for the App but no client ever shipped it, so it
// is not an input any more. A request still sending it gets the account stance.
func TestContentStanceBearerLaneIgnoresTheRetiredHeader(t *testing.T) {
	for _, tc := range []struct {
		why     string
		display string
		header  string
		want    content.Stance
	}{
		{"hide is not widened by a header that says yes", "hide", "1", content.StanceHide},
		{"show is not narrowed by a header that says no", "show", "false", content.StanceShow},
		{"no header at all is the normal App request", "blur", "", content.StanceBlur},
	} {
		_, rdb := stanceHarness(t)
		bearer := NewBearerStance(
			subjectVerifier{sub: "u-1207"},
			&countingUserInfo{info: userinfo(tc.display)},
			rdb,
		)

		app := fiber.New()
		app.Use(ContentStance(rdb, bearer))
		var got content.Stance
		var ok bool
		app.Get("/", func(c fiber.Ctx) error {
			got, ok = content.FromCtx(c)
			return nil
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(fiber.HeaderAuthorization, "Bearer "+stanceToken)
		if tc.header != "" {
			req.Header.Set("X-Kungal-Nsfw", tc.header)
		}
		if _, err := app.Test(req); err != nil {
			t.Fatalf("%s: %v", tc.why, err)
		}
		if !ok || got != tc.want {
			t.Errorf("%s: stance=%q ok=%v, want %q", tc.why, got, ok, tc.want)
		}
	}
}
