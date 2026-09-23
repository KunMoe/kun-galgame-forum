package apiv1

import (
	"strings"
	"testing"
)

func TestExcerptWindowsOnTheEarliestHit(t *testing.T) {
	body := strings.Repeat("前", 400) + "汉化补丁" + strings.Repeat("后", 400)
	got := excerpt(body, []string{"补丁", "汉化"})
	if !strings.Contains(got, "汉化补丁") || !strings.HasPrefix(got, "…") {
		t.Errorf("excerpt %q", got)
	}
	if n := len([]rune(got)); n > excerptLen+1 {
		t.Errorf("excerpt is %d runes, want at most %d", n, excerptLen+1)
	}
}

func TestExcerptStripsMarkdown(t *testing.T) {
	got := excerpt("看这张图 ![](/image/abc123) 和**汉化**说明", []string{"汉化"})
	if strings.Contains(got, "/image/") || strings.Contains(got, "**") {
		t.Errorf("excerpt leaked markup: %q", got)
	}
}

func TestExcerptLeavesShortBodiesAlone(t *testing.T) {
	if got := excerpt("这个汉化很棒", []string{"汉化"}); got != "这个汉化很棒" {
		t.Errorf("excerpt = %q", got)
	}
}

func TestExcerptMatchesCaseInsensitively(t *testing.T) {
	body := strings.Repeat("x", 300) + " KunGal " + strings.Repeat("y", 300)
	if got := excerpt(body, []string{"kungal"}); !strings.Contains(got, "KunGal") {
		t.Errorf("excerpt %q", got)
	}
}
