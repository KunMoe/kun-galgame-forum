package main

import (
	"reflect"
	"testing"

	"kun-galgame-api/pkg/config"
)

func TestPlanStoresSecretsAndNamedKeysOnly(t *testing.T) {
	env := []string{
		"PATH=/usr/bin",
		config.DatabaseURLEnv + "=postgresql://postgres:x@postgres:5432/kungalgame",
		"KUN_CONFIG_KEK=k",
		"REDIS_HOST=redis",
		"OAUTH_CLIENT_ID=4ed9",
		"OAUTH_CLIENT_SECRET=s",
		"KUN_NEWS_API_KEY=nmk",
		"KUN_TRUST_CALLBACK_SECRET=",
		"LINK_CHECKER_BASE_URL=https://checker",
	}
	got := Plan(env, "LINK_CHECKER_BASE_URL, REDIS_HOST")
	want := []pair{
		{"KUN_NEWS_API_KEY", "nmk"},
		{"LINK_CHECKER_BASE_URL", "https://checker"},
		{"OAUTH_CLIENT_SECRET", "s"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("plan = %v, want %v", got, want)
	}
}
