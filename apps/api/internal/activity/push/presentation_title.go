package push

import (
	"strings"
	"unicode"
)

func presentationTitle(raw, fallback string) string {
	mapped := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, raw)
	var b strings.Builder
	prevSpace := false
	for _, r := range mapped {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	s := strings.TrimSpace(b.String())
	runes := []rune(s)
	if len(runes) > titleRunes {
		s = strings.TrimSpace(string(runes[:titleRunes]))
	}
	if s == "" {
		return fallback
	}
	return s
}
