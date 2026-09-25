package config

import (
	"strings"
	"testing"
)

func TestLoadAppRelease(t *testing.T) {
	for _, tc := range []struct {
		why     string
		min     string
		latest  string
		wantErr bool
	}{
		{"defaults", "", "", false},
		{"min below latest", "0.9.12", "0.10.0", false},
		{"min above latest", "0.2.0", "0.1.9", true},
		{"prefixed", "v0.1.0", "0.1.0", true},
		{"two parts", "0.1", "0.1.0", true},
	} {
		t.Setenv("KUN_APP_MIN_VERSION", tc.min)
		t.Setenv("KUN_APP_LATEST_VERSION", tc.latest)
		cfg, err := loadAppRelease()
		if (err != nil) != tc.wantErr {
			t.Errorf("%s: err = %v, wantErr %v", tc.why, err, tc.wantErr)
		}
		if err == nil && cfg.Downloads.Android == "" {
			t.Errorf("%s: android download must default to the download page", tc.why)
		}
	}
}

func TestLoadAndroidPackage(t *testing.T) {
	const sum = "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"
	for _, tc := range []struct {
		why             string
		url, size, hash string
		want            bool
		wantErr         bool
	}{
		{"unset", "", "", "", false, false},
		{"complete", "https://dl.example/kungal-1.3.4.apk", "52428800", sum, true, false},
		{"partial", "https://dl.example/kungal.apk", "", "", false, true},
		{"http", "http://dl.example/kungal.apk", "52428800", sum, false, true},
		{"relative", "/kungal.apk", "52428800", sum, false, true},
		{"zero size", "https://dl.example/kungal.apk", "0", sum, false, true},
		{"uppercase hash", "https://dl.example/kungal.apk", "52428800", strings.ToUpper(sum), false, true},
		{"short hash", "https://dl.example/kungal.apk", "52428800", sum[:63], false, true},
	} {
		t.Setenv("KUN_APP_ANDROID_PACKAGE_URL", tc.url)
		t.Setenv("KUN_APP_ANDROID_PACKAGE_SIZE", tc.size)
		t.Setenv("KUN_APP_ANDROID_PACKAGE_SHA256", tc.hash)
		cfg, err := loadAppRelease()
		if (err != nil) != tc.wantErr {
			t.Errorf("%s: err = %v, wantErr %v", tc.why, err, tc.wantErr)
		}
		if err == nil && (cfg.AndroidPackage != nil) != tc.want {
			t.Errorf("%s: package = %+v, want present %v", tc.why, cfg.AndroidPackage, tc.want)
		}
	}
}
