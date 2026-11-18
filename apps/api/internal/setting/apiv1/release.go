package apiv1

import (
	"slices"
	"strconv"
	"strings"

	"kun-galgame-api/internal/setting"
	"kun-galgame-api/pkg/problem"
)

func mergeRelease(r setting.AppRelease, b SettingsPatch) setting.AppRelease {
	over := func(dst *string, v *string) {
		if v != nil {
			*dst = *v
		}
	}
	over(&r.MinVersion, b.AppMinVersion)
	over(&r.LatestVersion, b.AppLatestVersion)
	over(&r.AndroidPackageURL, b.AppAndroidPackageURL)
	over(&r.AndroidPackageSHA256, b.AppAndroidPackageSHA256)
	if b.AppAndroidPackageSize != nil {
		r.AndroidPackageSize = *b.AppAndroidPackageSize
	}
	return r
}

// checkRelease holds the rules the App relies on: it force-updates any install
// below min_version, so min above latest would send every user to a build that
// does not exist, and it installs the package only when size and hash match.
func checkRelease(r setting.AppRelease) *problem.Problem {
	var fields []problem.FieldError
	if slices.Compare(versionParts(r.MinVersion), versionParts(r.LatestVersion)) > 0 {
		fields = append(fields, problem.AtPointer("/app_min_version", problem.ReasonInconsistentWith, "/app_latest_version", nil))
	}
	set := []bool{r.AndroidPackageURL != "", r.AndroidPackageSize > 0, r.AndroidPackageSHA256 != ""}
	if slices.Contains(set, true) && slices.Contains(set, false) {
		for i, name := range []string{"/app_android_package_url", "/app_android_package_size", "/app_android_package_sha256"} {
			if !set[i] {
				fields = append(fields, problem.AtPointer(name, problem.ReasonRequired, "the Android package needs its url, size and sha256 together", nil))
			}
		}
	}
	if len(fields) == 0 {
		return nil
	}
	return problem.New(problem.CodeValidationFailed, "The App release settings contradict each other.", fields...)
}

func versionParts(v string) []int {
	out := make([]int, 3)
	for i, part := range strings.SplitN(v, ".", 3) {
		out[i], _ = strconv.Atoi(part)
	}
	return out
}
