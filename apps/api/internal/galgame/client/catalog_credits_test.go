package client

import (
	"strings"
	"testing"
)

func TestSortStaffRoleKeys_RanksUnpinnedRolesToo(t *testing.T) {
	got := SortStaffRoleKeys([]string{
		"other-staff", "企画", "theme-song-lyrics", "director", "lyric", "scenario",
	})
	want := "scenario,director,lyric,theme-song-lyrics,企画,other-staff"
	if strings.Join(got, ",") != want {
		t.Errorf("role order = %s, want %s", strings.Join(got, ","), want)
	}
}

func TestStaffRoleLabel_AgreesWithThePanel(t *testing.T) {
	for _, c := range []struct{ key, upstream, want string }{
		{"原画", "原画", "原画"},
		{"illustration", "插画", "原画"},
		{"剧本", "剧本", "脚本"},
		{"director-direction", "导演", "导演"},
		{"qa", "QA", "QA"},
	} {
		if got := StaffRoleLabel(c.key, c.upstream); got != c.want {
			t.Errorf("StaffRoleLabel(%q, %q) = %q, want %q", c.key, c.upstream, got, c.want)
		}
	}
}
