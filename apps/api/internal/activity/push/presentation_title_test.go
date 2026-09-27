package push

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestPresentationTitle(t *testing.T) {
	if got := presentationTitle("a\nb\tc\r  d", "fb"); got != "a b c d" {
		t.Fatalf("controls = %q", got)
	}
	if got := presentationTitle("   \n\t  ", "Galgame"); got != "Galgame" {
		t.Fatalf("whitespace = %q", got)
	}
	long := strings.Repeat("字", 200) + strings.Repeat("🎮", 50)
	got := presentationTitle(long, "fb")
	if n := len([]rune(got)); n != 200 {
		t.Fatalf("runes = %d %q", n, got)
	}
	if !utf8.ValidString(got) {
		t.Fatal("cut title is not valid UTF-8")
	}
	if got := presentationTitle("pre\x00mid", "fb"); got != "pre mid" {
		t.Fatalf("mid control = %q", got)
	}
}
