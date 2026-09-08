package utils

import "testing"

func TestDownloadLinkAcceptsMagnetURIs(t *testing.T) {
	for _, tc := range []struct {
		why  string
		in   string
		want bool
	}{
		{"production magnet with ASCII ?", "magnet:?xt=urn:btih:0ee024cfced3166973d5672471b223b7735604de&dn=x&tr=http%3A%2F%2Fnyaa.tracker.wf%3A7777%2Fannounce", true},
		{"magnet:// with query", "magnet://?xt=urn:btih:0ee024cfced3166973d5672471b223b7735604de", true},
		// 16 production rows carry a full-width ？; editing such a resource must not start failing.
		{"full-width ？ workaround", "magnet:？xt=urn:btih:0ee024cfced3166973d5672471b223b7735604de", true},
		{"https URL", "https://example.com/a?b=1", true},
		{"ftp URL", "ftp://files.example.com/path", true},
		{"empty", "", false},
		{"whitespace", "   ", false},
		{"not a url", "not-a-url", false},
		{"bare host", "example.com", false},
		{"hash only", "#anchor", false},
		{"javascript scheme", "javascript:alert(1)", false},
		{"data scheme", "data:text/html,<h1>x</h1>", false},
		{"https typo", "ttps://pan.example.com/s/abc", false},
	} {
		err := validate.Var(tc.in, "downloadlink")
		got := err == nil
		if got != tc.want {
			t.Errorf("%s: validate.Var(%q, \"downloadlink\") ok=%v, want %v",
				tc.why, tc.in, got, tc.want)
		}
	}
}
