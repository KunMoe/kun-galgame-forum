package apiv1

import (
	"reflect"
	"testing"
	"time"

	"kun-galgame-api/internal/envconfig"
	"kun-galgame-api/pkg/config"
)

func TestIssues(t *testing.T) {
	booted := config.LoadedAt()
	before, after := booted.Add(-time.Hour), booted.Add(time.Hour)
	run := func(src config.Source, v string) *config.Var {
		return &config.Var{Key: "KUN_X_API_KEY", Value: v, Source: src}
	}
	stored := func(v string, at time.Time) *envconfig.Stored {
		return &envconfig.Stored{Key: "KUN_X_API_KEY", Value: v, Updated: at}
	}
	for _, tc := range []struct {
		why     string
		running *config.Var
		stored  *envconfig.Stored
		want    []EnvVariableIssue
	}{
		{"env only", run(config.SourceEnv, "a"), nil, []EnvVariableIssue{}},
		{"panel copy of the stored value", run(config.SourceEnv, "a"), stored("a", after), []EnvVariableIssue{"panel_redundant"}},
		{"panel differs from the store", run(config.SourceEnv, "a"), stored("b", before), []EnvVariableIssue{"panel_overrides_store"}},
		{"in use from the store", run(config.SourceStore, "a"), stored("a", before), []EnvVariableIssue{}},
		{"stored after the start", run(config.SourceDefault, ""), stored("a", after), []EnvVariableIssue{"restart_pending"}},
		{"deleted after the start", run(config.SourceStore, "a"), nil, []EnvVariableIssue{"restart_pending"}},
		{"stored, never read", nil, stored("a", before), []EnvVariableIssue{}},
	} {
		if got := issues(tc.running, tc.stored); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: issues = %v, want %v", tc.why, got, tc.want)
		}
	}
}

func TestSecretsNeverShowTheirValue(t *testing.T) {
	v := variableOf(&config.Var{Key: "KUN_X_API_KEY", Value: "nmk_live_secret", Source: config.SourceEnv},
		&envconfig.Stored{Key: "KUN_X_API_KEY", Value: "nmk_live_secret"})
	if v.VariableKind != "secret" || v.Running.Plain != nil || v.Stored.Plain != nil || v.DefaultValue != nil ||
		v.Running.Fingerprint == nil || *v.Running.Hint != "…cret" {
		t.Fatalf("secret shown as %+v / %+v", v.Running, v.Stored)
	}
	if p := variableOf(&config.Var{Key: config.DatabaseURLEnv, Source: config.SourceEnv}, nil); p.IsStorable || p.VariableKind != "bootstrap" {
		t.Errorf("bootstrap variable %+v", p)
	}
}
