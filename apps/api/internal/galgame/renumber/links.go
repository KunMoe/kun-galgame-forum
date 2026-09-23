package renumber

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const galgamePath = "/galgame/"

var absPrefixes = []string{
	"https://www.kungal.com",
	"http://www.kungal.com",
	"https://www.kungal.org",
	"http://www.kungal.org",
	"https://kungal.com",
	"http://kungal.com",
}

var bareHosts = []string{
	"www.kungal.com",
	"www.kungal.org",
	"kungal.com",
}

func RewriteLinks(text string, lookup func(int64) (int64, bool)) (string, int) {
	if lookup == nil || !strings.Contains(text, galgamePath) {
		return text, 0
	}
	var b strings.Builder
	b.Grow(len(text))
	n := 0
	i := 0
	for {
		j := strings.Index(text[i:], galgamePath)
		if j < 0 {
			b.WriteString(text[i:])
			return b.String(), n
		}
		j += i
		digitStart := j + len(galgamePath)
		digitEnd := digitStart
		for digitEnd < len(text) && text[digitEnd] >= '0' && text[digitEnd] <= '9' {
			digitEnd++
		}
		if digitEnd == digitStart || !idBoundary(text, digitEnd) {
			b.WriteString(text[i : j+1])
			i = j + 1
			continue
		}
		locStart := localeStart(text, j)
		ok := false
		if _, absOK := absoluteStart(text, locStart); absOK {
			ok = true
		} else if bareHostOK(text, locStart) {
			ok = true
		} else if relativeOK(text, locStart) {
			ok = true
		}
		if !ok {
			b.WriteString(text[i : j+1])
			i = j + 1
			continue
		}
		oldID, err := strconv.ParseInt(text[digitStart:digitEnd], 10, 64)
		if err != nil {
			b.WriteString(text[i:digitEnd])
			i = digitEnd
			continue
		}
		newID, hit := lookup(oldID)
		if !hit || newID == oldID {
			b.WriteString(text[i:digitEnd])
			i = digitEnd
			continue
		}
		b.WriteString(text[i:digitStart])
		b.WriteString(strconv.FormatInt(newID, 10))
		i = digitEnd
		n++
	}
}

func localeStart(s string, galgameIdx int) int {
	if galgameIdx < 6 {
		return galgameIdx
	}
	p := s[galgameIdx-6 : galgameIdx]
	if p[0] == '/' && isLowerAZ(p[1]) && isLowerAZ(p[2]) && p[3] == '-' && isLowerAZ(p[4]) && isLowerAZ(p[5]) {
		return galgameIdx - 6
	}
	return galgameIdx
}

func absoluteStart(s string, locStart int) (int, bool) {
	for _, p := range absPrefixes {
		if start, ok := hostEndsAt(s, locStart, p); ok {
			return start, true
		}
	}
	return 0, false
}

// The editor escapes the dots and the colon of a URL it did not turn into a
// link, so production text holds "https\://www\.kungal.com/galgame/4603".
func hostEndsAt(s string, end int, host string) (int, bool) {
	k := end - 1
	for h := len(host) - 1; h >= 0; h-- {
		if k < 0 || !strings.EqualFold(s[k:k+1], host[h:h+1]) {
			return 0, false
		}
		k--
		if (host[h] == '.' || host[h] == ':') && k >= 0 && s[k] == '\\' {
			k--
		}
	}
	return k + 1, true
}

// A markdown label repeats the href without its scheme,
// "[www.kungal.com/galgame/6798](http://www.kungal.com/galgame/6798)", and the
// rehearsal on the production copy renumbered the href but left the label.
func bareHostOK(s string, locStart int) bool {
	for _, h := range bareHosts {
		if start, ok := hostEndsAt(s, locStart, h); ok && relativeOK(s, start) {
			return true
		}
	}
	return false
}

func relativeOK(s string, locStart int) bool {
	if locStart == 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(s[:locStart])
	if r == utf8.RuneError {
		return false
	}
	if unicode.IsSpace(r) {
		return true
	}
	switch r {
	case '(', '<', '[', '"', '\'', '>', '`', '+', '*':
		return true
	default:
		return false
	}
}

func idBoundary(s string, i int) bool {
	if i == len(s) {
		return true
	}
	c := s[i]
	if c >= '0' && c <= '9' {
		return false
	}
	if c >= 'A' && c <= 'Z' {
		return false
	}
	if c >= 'a' && c <= 'z' {
		return false
	}
	return c != '_'
}

func isLowerAZ(c byte) bool { return c >= 'a' && c <= 'z' }
