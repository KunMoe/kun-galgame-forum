package utils

import "testing"

// The mirror stores catalog's release_date in a DATE column, so a partial date
// has to become a whole one. Dropping the row instead would have left exactly
// the hole 092 was written to close.
func TestNormalizeCatalogReleaseDate(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"2026-08-27", "2026-08-27", true},
		{"2026-08", "2026-08-01", true},
		{"2026", "2026-01-01", true},
		{" 2026-08-27 ", "2026-08-27", true},
		{"", "", true},
		{"2026-13", "", false},
		{"2026-02-30", "", false},
		{"august", "", false},
	}
	for _, c := range cases {
		got, ok := NormalizeCatalogReleaseDate(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("NormalizeCatalogReleaseDate(%q) = %q, %v; want %q, %v",
				c.in, got, ok, c.want, c.ok)
		}
	}
}
