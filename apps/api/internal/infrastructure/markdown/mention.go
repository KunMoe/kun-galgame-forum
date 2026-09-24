package markdown

import (
	"regexp"
	"strconv"
	"strings"
)

var mentionIDRe = regexp.MustCompile(`kungal-user:(\d+)`)

func ExtractMentionIDs(content string) []int {
	matches := mentionIDRe.FindAllStringSubmatch(content, -1)
	if matches == nil {
		return nil
	}
	seen := make(map[int]bool, len(matches))
	ids := make([]int, 0, len(matches))
	for _, m := range matches {
		id, err := strconv.Atoi(m[1])
		if err != nil || id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

var (
	mentionTokenRe = regexp.MustCompile(`\[@[^\]]*\]\(kungal-user:\d+\)`)
	quoteTokenRe   = regexp.MustCompile(`\[#[^\]]*\]\(kungal-reply:\d+\)`)
	replyHeaderRe  = regexp.MustCompile(`(?m)^\s*>\s*回复\s*`)
)

func StripReferenceTokens(content string) string {
	s := mentionTokenRe.ReplaceAllString(content, "")
	s = quoteTokenRe.ReplaceAllString(s, "")
	s = replyHeaderRe.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(s), " ")
}
