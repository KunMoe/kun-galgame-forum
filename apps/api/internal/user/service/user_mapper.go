package service

import (
	"kun-galgame-api/internal/user/repository"
)

func appendUniqueStr(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}

func emptyStrSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func groupResourceMeta(rows []repository.GalgameResourceMeta) (platforms, languages map[int][]string) {
	platforms = make(map[int][]string)
	languages = make(map[int][]string)
	for _, r := range rows {
		if r.Platform != "" {
			platforms[r.WorkID] = appendUniqueStr(platforms[r.WorkID], r.Platform)
		}
		if r.Language != "" {
			languages[r.WorkID] = appendUniqueStr(languages[r.WorkID], r.Language)
		}
	}
	return
}

func collectUniqueIDs[T any](rows []T, pick func(T) int) []int {
	out := make([]int, 0, len(rows))
	seen := make(map[int]bool, len(rows))
	for _, r := range rows {
		id := pick(r)
		if id > 0 && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}
