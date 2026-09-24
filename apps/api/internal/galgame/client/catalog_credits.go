package client

import (
	"slices"
	"sort"
)

var staffRoleFold = map[string]string{
	"剧本":                 "scenario",
	"原画":                 "illustration",
	"音乐":                 "music",
	"director-direction": "director",
}

var staffRoleName = map[string]string{
	"scenario":     "脚本",
	"illustration": "原画",
	"music":        "音乐",
	"director":     "导演",
}

var staffRoleDisplayOrder = []string{
	"原作",
	"scenario",
	"illustration",
	"character-design",
	"music",
	"voice-actor",
	"director",
	"composer",
	"lyric",
	"arrange",
	"vocal",
	"theme-song-composition",
	"theme-song-lyrics",
	"theme-song-performance",
	"inserted-song-performance",
}

const staffRoleLast = "other-staff"

const StaffRoleOtherKey = staffRoleLast

func StaffRoleCanonicalKey(roleKey string) string {
	if folded, ok := staffRoleFold[roleKey]; ok {
		return folded
	}
	return roleKey
}

func StaffRoleLabel(roleKey, roleName string) string {
	key := StaffRoleCanonicalKey(roleKey)
	if pinned, ok := staffRoleName[key]; ok {
		return pinned
	}
	if roleName != "" {
		return roleName
	}
	return key
}

func SortStaffRoleKeys(keys []string) []string {
	rank := make(map[string]int, len(staffRoleDisplayOrder))
	for i, key := range staffRoleDisplayOrder {
		rank[key] = i
	}
	weight := func(key string) int {
		if key == staffRoleLast {
			return len(staffRoleDisplayOrder) + 1
		}
		if r, ok := rank[key]; ok {
			return r
		}
		return len(staffRoleDisplayOrder)
	}
	out := slices.Clone(keys)
	sort.SliceStable(out, func(i, j int) bool { return weight(out[i]) < weight(out[j]) })
	return out
}
