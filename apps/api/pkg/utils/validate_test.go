package utils

import "testing"

func TestIsDownloadLink(t *testing.T) {
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
		{"parentheses in a path", "https://x.example/a(1).zip", true},
		{"ed2k file link with a CJK name", "ed2k://|file|纯白交响曲.rar|123456|0123456789ABCDEF0123456789ABCDEF|/", true},
		{"ed2k with a space, AICH and part hashes", "ed2k://|file|Game Remake.iso|4294967296|0123456789abcdef0123456789abcdef|h=QWERTYUIOPASDFGHJKLZXCVBNM234567|p=0123456789abcdef0123456789abcdef|/", true},
		{"ed2k without the closing slash", "ed2k://|file|a.rar|1|0123456789abcdef0123456789abcdef|", true},
		{"ed2k in upper case", "ED2K://|FILE|a.rar|1|0123456789abcdef0123456789abcdef|/", true},
		{"ed2k cut at CJK", "ed2k://|file|", false},
		{"ed2k hash not hex", "ed2k://|file|a.rar|1|NOTAHASHNOTAHASHNOTAHASHNOTAHASH|/", false},
		{"ed2k hash too short", "ed2k://|file|a.rar|1|0123456789abcdef|/", false},
		{"ed2k size not a number", "ed2k://|file|a.rar|1GB|0123456789abcdef0123456789abcdef|/", false},
		{"ed2k tail after the link", "ed2k://|file|a.rar|1|0123456789abcdef0123456789abcdef|/<script>", false},
		{"ed2k name across a line break", "ed2k://|file|a\nb.rar|1|0123456789abcdef0123456789abcdef|/", false},
		{"ed2k server link", "ed2k://|server|1.2.3.4|4661|/", false},
	} {
		if got := IsDownloadLink(tc.in); got != tc.want {
			t.Errorf("%s: IsDownloadLink(%q) = %v, want %v", tc.why, tc.in, got, tc.want)
		}
	}
}
