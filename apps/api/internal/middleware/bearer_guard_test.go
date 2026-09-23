package middleware

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// These read roles that did not come off the request, so the Bearer channel
// cannot reach them through a raw check.
var rawCapabilityCheckAllowed = map[string]string{
	"middleware/auth.go":             "the UserInfo methods themselves",
	"admin/service/purge_service.go": "the purge TARGET's roles, from the user client",
	"community/trust/trust.go":       "the trust boost declared once per user",
}

// A capability check written against u.Roles bypasses UserInfo.Can, so a
// staff member's App token would regain the powers the Bearer channel denies.
func TestCapabilityChecksGoThroughUserInfo(t *testing.T) {
	raw := regexp.MustCompile(`perm\.(CanUser|EffectiveForUser)\(|role\.Can(Moderate|Administer)\(`)
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		rel := filepath.ToSlash(strings.TrimPrefix(path, "../"))
		if _, ok := rawCapabilityCheckAllowed[rel]; ok {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(src), "\n") {
			if raw.MatchString(line) {
				t.Errorf("internal/%s:%d checks a capability on raw roles; use the "+
					"UserInfo methods (user.Can / user.CanModerate):\n\t%s",
					rel, i+1, strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
