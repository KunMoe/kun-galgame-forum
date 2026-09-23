package dlsite

import "strings"

const worknoPlaceholder = "{workno}"

var worknoPrefixes = []string{"RJ", "VJ"}

func Link(template, workno string) string {
	if template == "" || !ValidWorkno(workno) {
		return ""
	}
	return strings.ReplaceAll(template, worknoPlaceholder, workno)
}

func WorknoFor(workID int, refsWorkno string) string {
	if ValidWorkno(refsWorkno) {
		return refsWorkno
	}
	return VerifiedWorkno(workID)
}

func ValidWorkno(s string) bool {
	digits, ok := trimKnownPrefix(s)
	if !ok || digits == "" {
		return false
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func trimKnownPrefix(s string) (string, bool) {
	for _, p := range worknoPrefixes {
		if rest, found := strings.CutPrefix(s, p); found {
			return rest, true
		}
	}
	return "", false
}
