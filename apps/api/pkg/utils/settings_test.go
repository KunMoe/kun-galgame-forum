package utils

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"kun-galgame-api/pkg/content"

	"github.com/gofiber/fiber/v3"
)

// The cookie is written by the frontend's persisted-settings store, so it
// arrives URL-encoded and carries whatever keys that store happens to hold —
// including ones this API has never heard of.
func TestSettingsCookie(t *testing.T) {
	for _, tc := range []struct {
		why        string
		cookie     string
		wantSFW    bool
		wantOrigin bool
	}{
		{"no cookie at all", "", true, false},
		{"unparseable", "not json", true, false},
		{"a store that predates both keys", `{"showKUNGalgameRounded":"md"}`, true, false},
		{"nsfw on, names default", `{"showKUNGalgameContentLimit":"nsfw"}`, false, false},
		{"原名 on", `{"showKUNGalgamePreferOriginalName":true}`, true, true},
		{"both on", `{"showKUNGalgameContentLimit":"all","showKUNGalgamePreferOriginalName":true}`, false, true},
	} {
		app := fiber.New()
		var gotSFW, gotOrigin bool
		app.Get("/", func(c fiber.Ctx) error {
			gotSFW, gotOrigin = IsSFW(c), PrefersOriginalName(c)
			return nil
		})
		req := httptest.NewRequest("GET", "/", nil)
		if tc.cookie != "" {
			req.Header.Set("Cookie", "KUNGalgameSettings="+url.QueryEscape(tc.cookie))
		}
		if _, err := app.Test(req); err != nil {
			t.Fatalf("%s: %v", tc.why, err)
		}
		if gotSFW != tc.wantSFW || gotOrigin != tc.wantOrigin {
			t.Errorf("%s: sfw=%v origin=%v, want %v/%v",
				tc.why, gotSFW, gotOrigin, tc.wantSFW, tc.wantOrigin)
		}
	}
}

// X-Kungal-Nsfw is gone: no shipped client ever sent it, and an identified
// reader — session or Bearer — now arrives with a stance attached. A request
// still carrying the retired header must be decided by the stance or the
// cookie, never by the header.
func TestRetiredNSFWHeaderIsNotAnInput(t *testing.T) {
	for _, tc := range []struct {
		why     string
		header  string
		cookie  string
		wantSFW bool
	}{
		{"a header that says yes no longer widens anything", "1", "", true},
		{"a header that says no does not narrow the cookie either", "false", `{"showKUNGalgameContentLimit":"nsfw"}`, false},
	} {
		app := fiber.New()
		var gotSFW bool
		app.Get("/", func(c fiber.Ctx) error {
			gotSFW = IsSFW(c)
			return nil
		})
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Kungal-Nsfw", tc.header)
		if tc.cookie != "" {
			req.Header.Set("Cookie", "KUNGalgameSettings="+url.QueryEscape(tc.cookie))
		}
		if _, err := app.Test(req); err != nil {
			t.Fatalf("%s: %v", tc.why, err)
		}
		if gotSFW != tc.wantSFW {
			t.Errorf("%s: IsSFW = %v, want %v", tc.why, gotSFW, tc.wantSFW)
		}
	}
}

// The account stance outranks the cookie, and it is attached only for an
// identified reader — so the logged-out cookie lane keeps answering exactly
// what it always answered.
func TestStanceOutranksTheCookie(t *testing.T) {
	for _, tc := range []struct {
		why     string
		stance  *content.Stance
		cookie  string
		wantSFW bool
	}{
		{"hide beats an nsfw cookie", stance(content.StanceHide), `{"showKUNGalgameContentLimit":"nsfw"}`, true},
		{"blur lets the row through; masking is the web's job", stance(content.StanceBlur), "", false},
		{"show lets the row through", stance(content.StanceShow), "", false},
		{"show is not overturned by an sfw cookie", stance(content.StanceShow), `{"showKUNGalgameContentLimit":"sfw"}`, false},
		{"no stance: the cookie still decides (logged out)", nil, `{"showKUNGalgameContentLimit":"nsfw"}`, false},
		{"no stance, no cookie", nil, "", true},
	} {
		app := fiber.New()
		var gotSFW bool
		app.Get("/", func(c fiber.Ctx) error {
			if tc.stance != nil {
				content.Attach(c, *tc.stance)
			}
			gotSFW = IsSFW(c)
			return nil
		})
		req := httptest.NewRequest("GET", "/", nil)
		if tc.cookie != "" {
			req.Header.Set("Cookie", "KUNGalgameSettings="+url.QueryEscape(tc.cookie))
		}
		if _, err := app.Test(req); err != nil {
			t.Fatalf("%s: %v", tc.why, err)
		}
		if gotSFW != tc.wantSFW {
			t.Errorf("%s: IsSFW = %v, want %v", tc.why, gotSFW, tc.wantSFW)
		}
	}
}

func stance(s content.Stance) *content.Stance { return &s }
