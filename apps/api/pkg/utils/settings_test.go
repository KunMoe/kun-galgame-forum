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

func TestNSFWHeaderBeatsCookie(t *testing.T) {
	for _, tc := range []struct {
		why     string
		header  string
		cookie  string
		wantSFW bool
	}{
		{"header on, no cookie", "1", "", false},
		{"header off overrides an nsfw cookie", "false", `{"showKUNGalgameContentLimit":"nsfw"}`, true},
		{"unparseable header falls back to the cookie", "maybe", `{"showKUNGalgameContentLimit":"nsfw"}`, false},
		{"neither", "", "", true},
	} {
		app := fiber.New()
		var gotSFW bool
		app.Get("/", func(c fiber.Ctx) error {
			gotSFW = IsSFW(c)
			return nil
		})
		req := httptest.NewRequest("GET", "/", nil)
		if tc.header != "" {
			req.Header.Set(NSFWHeader, tc.header)
		}
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

// The account stance outranks both other inputs, and it is attached only for a
// session identity — so the App's header lane and the logged-out cookie lane
// keep answering exactly what they answered before this wave.
func TestStanceOutranksHeaderAndCookie(t *testing.T) {
	for _, tc := range []struct {
		why     string
		stance  *content.Stance
		header  string
		cookie  string
		wantSFW bool
	}{
		{"hide beats an nsfw cookie", stance(content.StanceHide), "", `{"showKUNGalgameContentLimit":"nsfw"}`, true},
		{"hide beats a header that says yes", stance(content.StanceHide), "1", "", true},
		{"blur lets the row through; masking is the web's job", stance(content.StanceBlur), "", "", false},
		{"show lets the row through", stance(content.StanceShow), "", "", false},
		{"show is not overturned by an sfw cookie", stance(content.StanceShow), "", `{"showKUNGalgameContentLimit":"sfw"}`, false},
		{"no stance: the header still wins (App lane)", nil, "1", `{"showKUNGalgameContentLimit":"sfw"}`, false},
		{"no stance and no header: the cookie still decides (logged out)", nil, "", `{"showKUNGalgameContentLimit":"nsfw"}`, false},
		{"no stance, no header, no cookie", nil, "", "", true},
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
		if tc.header != "" {
			req.Header.Set(NSFWHeader, tc.header)
		}
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
