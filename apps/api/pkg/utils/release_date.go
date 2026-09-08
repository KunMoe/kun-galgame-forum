package utils

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	reYear  = regexp.MustCompile(`^\d{4}$`)
	reMonth = regexp.MustCompile(`^(\d{4})-(\d{2})$`)
)

// Accepts "YYYY" and "YYYY-MM" and resolves each to an inclusive date bound.
// Anything else is REJECTED so a malformed filter surfaces as a 400 rather than
// silently returning the whole table. Returns date strings, not time.Time: the
// column is date-typed and tz-free, so day granularity is exact and cannot drift.
func ParseReleaseLowerBound(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	if reYear.MatchString(s) {
		return s + "-01-01", nil
	}
	if m := reMonth.FindStringSubmatch(s); m != nil {
		if err := validMonth(m[2]); err != nil {
			return "", err
		}
		return m[1] + "-" + m[2] + "-01", nil
	}
	return "", fmt.Errorf("非法的发售日期下限 %q（应为 YYYY 或 YYYY-MM）", s)
}

func ParseReleaseUpperBound(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	if reYear.MatchString(s) {
		return s + "-12-31", nil
	}
	if m := reMonth.FindStringSubmatch(s); m != nil {
		if err := validMonth(m[2]); err != nil {
			return "", err
		}
		year, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		last := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC)
		return last.Format("2006-01-02"), nil
	}
	return "", fmt.Errorf("非法的发售日期上限 %q（应为 YYYY 或 YYYY-MM）", s)
}

func validMonth(mm string) error {
	month, _ := strconv.Atoi(mm)
	if month < 1 || month > 12 {
		return fmt.Errorf("非法的月份 %q（应为 01-12）", mm)
	}
	return nil
}

func ParseMonthSet(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	seen := map[int]bool{}
	for _, tok := range strings.Split(s, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		m, err := strconv.Atoi(tok)
		if err != nil || m < 1 || m > 12 {
			return nil, fmt.Errorf("非法的月份 %q（应为 1-12 的逗号分隔列表）", tok)
		}
		seen[m] = true
	}
	if len(seen) == 0 {
		return nil, nil
	}
	out := make([]int, 0, len(seen))
	for m := range seen {
		out = append(out, m)
	}
	sort.Ints(out)
	return out, nil
}

// Catalog sends release_date at whatever precision it knows: "2026", "2026-08"
// or "2026-08-27". The local mirror column is a DATE, so a partial date is kept
// as the first day of the period it names — the same lower bound the year and
// month filters resolve to, which keeps a year-precision work inside its own
// year. The second return is false only for a string catalog should never send.
func NormalizeCatalogReleaseDate(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", true
	}
	switch len(s) {
	case 4:
		s += "-01-01"
	case 7:
		s += "-01"
	}
	if _, err := time.Parse(time.DateOnly, s); err != nil {
		return "", false
	}
	return s, true
}
