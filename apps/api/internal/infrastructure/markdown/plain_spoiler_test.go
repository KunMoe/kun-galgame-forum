package markdown

import "testing"

func TestMaskPlainSpoilers(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
	}{
		{"plain text", "**普通** <文本>", "**普通** <文本>"},
		{"multiple passages", "前||秘密一||中||秘密二||后", "前███中███后"},
		{"multiline passage", "前||第一行\n第二行||后", "前███后"},
		{"unclosed marker", "前||未闭合", "前||未闭合"},
		{"empty passage", "前||||后", "前███后"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := MaskPlainSpoilers(tc.source); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
