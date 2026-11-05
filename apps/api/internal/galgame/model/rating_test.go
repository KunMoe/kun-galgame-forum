package model

import "testing"

func TestSummaryShown(t *testing.T) {
	cases := []struct {
		level string
		mask  bool
		want  bool
	}{
		{"none", false, true},
		{"none", true, true},
		{"portion", true, true},
		{"portion", false, false},
		{"serious", true, false},
		{"serious", false, false},
	}
	for _, c := range cases {
		if got := SummaryShown(c.level, c.mask); got != c.want {
			t.Errorf("SummaryShown(%q, %v) = %v, want %v", c.level, c.mask, got, c.want)
		}
	}
}

func TestSummaryExcerptDropsBlankLines(t *testing.T) {
	in := "第一段\n\n- 一\n- 二\n\n███\n\n末尾"
	if got := SummaryExcerpt(in); got != "第一段\n- 一\n- 二\n███\n末尾" {
		t.Fatalf("SummaryExcerpt = %q", got)
	}
}
