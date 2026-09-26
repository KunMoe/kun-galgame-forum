package app

import (
	"net/http"
	"testing"
)

func TestV1AccountPrices(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(*meFix)
		want    any
		session string
	}{
		{
			name:    "published",
			setup:   func(f *meFix) { f.setSettings(map[string]any{"auth.name_change_cost": 23}) },
			want:    float64(23),
			session: "sess-alice",
		},
		{
			name:    "key absent",
			setup:   func(*meFix) {},
			want:    nil,
			session: "sess-alice",
		},
		{
			name:    "upstream 500",
			setup:   func(f *meFix) { f.settingsHTTP.Store(500) },
			want:    nil,
			session: "sess-alice",
		},
		{
			name:    "zero",
			setup:   func(f *meFix) { f.setSettings(map[string]any{"auth.name_change_cost": 0}) },
			want:    float64(0),
			session: "sess-alice",
		},
		{
			name:    "negative",
			setup:   func(f *meFix) { f.setSettings(map[string]any{"auth.name_change_cost": -1}) },
			want:    nil,
			session: "sess-alice",
		},
		{
			name:    "fraction",
			setup:   func(f *meFix) { f.setSettings(map[string]any{"auth.name_change_cost": 17.5}) },
			want:    nil,
			session: "sess-alice",
		},
		{
			name:    "string",
			setup:   func(f *meFix) { f.setSettings(map[string]any{"auth.name_change_cost": "17"}) },
			want:    nil,
			session: "sess-alice",
		},
		{
			name:    "anonymous",
			setup:   func(f *meFix) { f.setSettings(map[string]any{"auth.name_change_cost": 23}) },
			want:    float64(23),
			session: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newMeFix(t)
			tc.setup(f)
			resp, body := f.call(t, http.MethodGet, "/api/v1/account-prices", "/account-prices", tc.session, "", nil, nil)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("%d %+v", resp.StatusCode, body)
			}
			if body["object"] != "account_prices" {
				t.Fatalf("object %v", body["object"])
			}
			if body["rename_cost"] != tc.want {
				t.Fatalf("rename_cost %v, want %v", body["rename_cost"], tc.want)
			}
		})
	}
}
