package dlsite

import (
	"strings"
	"testing"
)

const tmpl = "https://dlaf.jp/soft/dlaf/=/t/s/link/work/aid/kungal/locale/zh_CN/id/{workno}.html/?locale=zh_CN"

func TestValidWorkno(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"RJ297925", true},
		{"VJ013975", true},
		{"RJ01005286", true},
		{"VJ01005286", true},

		{"AJ001234", false},

		{"", false},
		{"RJ", false},
		{"297925", false},
		{"XX123456", false},
		{"rj297925", false},
		{"RJ12 34", false},
		{"RJ123456/", false},
		{"RJ12a456", false},
	}
	for _, tc := range cases {
		if got := ValidWorkno(tc.in); got != tc.want {
			t.Errorf("ValidWorkno(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestLink(t *testing.T) {
	t.Run("interpolates the workno verbatim", func(t *testing.T) {
		got := Link(tmpl, "VJ013975")
		want := "https://dlaf.jp/soft/dlaf/=/t/s/link/work/aid/kungal/locale/zh_CN/id/VJ013975.html/?locale=zh_CN"
		if got != want {
			t.Errorf("Link() = %q, want %q", got, want)
		}
	})

	t.Run("does not pad the digit width", func(t *testing.T) {
		if got := Link(tmpl, "VJ013975"); !strings.Contains(got, "id/VJ013975.html") {
			t.Errorf("Link() rewrote the workno: %q", got)
		}
	})

	t.Run("empty template disables the feature", func(t *testing.T) {
		if got := Link("", "RJ297925"); got != "" {
			t.Errorf("Link() with no template = %q, want \"\"", got)
		}
	})

	t.Run("an unrecognised workno yields no link", func(t *testing.T) {
		for _, bad := range []string{"", "AJ001234", "rj297925", "RJ123456/"} {
			if got := Link(tmpl, bad); got != "" {
				t.Errorf("Link(%q) = %q, want \"\"", bad, got)
			}
		}
	})
}

func TestVerifiedWhitelist(t *testing.T) {
	const want = 4317
	if got := VerifiedCount(); got != want {
		t.Errorf("VerifiedCount() = %d, want %d (verified.tsv mis-vendored?)", got, want)
	}

	t.Run("a conflicted game uses infra's pinned ruling", func(t *testing.T) {
		if got := VerifiedWorkno(4123); got != "RJ090411" {
			t.Errorf("VerifiedWorkno(4123) = %q, want RJ090411 (infra pinned)", got)
		}
	})

	t.Run("header row is not parsed as a pair", func(t *testing.T) {
		if got := VerifiedWorkno(0); got != "" {
			t.Errorf("VerifiedWorkno(0) = %q, want \"\"", got)
		}
	})

	t.Run("an unknown work has no entry", func(t *testing.T) {
		if got := VerifiedWorkno(999999999); got != "" {
			t.Errorf("VerifiedWorkno(999999999) = %q, want \"\"", got)
		}
	})

	t.Run("every entry is a well-formed workno", func(t *testing.T) {
		verifiedOnce.Do(loadVerified)
		for id, wn := range verifiedMap {
			if !ValidWorkno(wn) {
				t.Fatalf("work %d maps to malformed workno %q", id, wn)
			}
		}
	})
}

func TestWorknoForPrecedence(t *testing.T) {
	const whitelisted = 4

	t.Run("refs wins when present", func(t *testing.T) {
		if got := WorknoFor(whitelisted, "RJ297925"); got != "RJ297925" {
			t.Errorf("WorknoFor() = %q, want the refs workno to win", got)
		}
	})

	t.Run("whitelist fills the gap when refs is absent", func(t *testing.T) {
		if got := WorknoFor(whitelisted, ""); got != "VJ013550" {
			t.Errorf("WorknoFor() = %q, want the whitelisted workno", got)
		}
	})

	t.Run("neither source yields no workno", func(t *testing.T) {
		if got := WorknoFor(999999999, ""); got != "" {
			t.Errorf("WorknoFor() = %q, want \"\"", got)
		}
	})

	t.Run("unconfigured template disables the link", func(t *testing.T) {
		if got := Link("", WorknoFor(whitelisted, "RJ297925")); got != "" {
			t.Errorf("Link() with no template = %q, want \"\"", got)
		}
	})
}
