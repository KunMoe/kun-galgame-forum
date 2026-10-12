package config

import "testing"

func TestLoadNotificationRetentionDays(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want int
	}{
		{"", 365},
		{"30", 30},
		{"0", 0},
		{"-1", -1},
	} {
		t.Setenv("KUN_NOTIFICATION_RETENTION_DAYS", tc.raw)
		if got := loadNotification().RetentionDays; got != tc.want {
			t.Errorf("%q: retention %d, want %d", tc.raw, got, tc.want)
		}
	}
}
