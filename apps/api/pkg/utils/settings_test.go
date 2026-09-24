package utils

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// The cookie is written by the frontend's persisted-settings store, so it
// arrives URL-encoded and carries whatever keys that store happens to hold —
// including ones this API has never heard of.
func TestSettingsCookie(t *testing.T) {
	for _, tc := range []struct {
		why        string
		cookie     string
		wantOrigin bool
	}{
		{"no cookie at all", "", false},
		{"unparseable", "not json", false},
		{"a store that predates the key", `{"showKUNGalgameRounded":"md"}`, false},
		{"原名 on", `{"showKUNGalgamePreferOriginalName":true}`, true},
		{"原名 on beside the retired content limit", `{"showKUNGalgameContentLimit":"all","showKUNGalgamePreferOriginalName":true}`, true},
	} {
		app := fiber.New()
		var gotOrigin bool
		app.Get("/", func(c fiber.Ctx) error {
			gotOrigin = PrefersOriginalName(c)
			return nil
		})
		req := httptest.NewRequest("GET", "/", nil)
		if tc.cookie != "" {
			req.Header.Set("Cookie", "KUNGalgameSettings="+url.QueryEscape(tc.cookie))
		}
		if _, err := app.Test(req); err != nil {
			t.Fatalf("%s: %v", tc.why, err)
		}
		if gotOrigin != tc.wantOrigin {
			t.Errorf("%s: origin=%v, want %v", tc.why, gotOrigin, tc.wantOrigin)
		}
	}
}
