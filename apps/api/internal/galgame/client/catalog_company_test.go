package client

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRewriteV2JSON_AttributionRoleBecomesRole(t *testing.T) {
	raw := []byte(`{"object":"company","id":"13","display_name":"FAVORITE","company_kind":"game_brand","attribution_role":"developer"}`)
	var got map[string]any
	if err := json.Unmarshal(rewriteV2JSON(raw, ""), &got); err != nil {
		t.Fatalf("rewrite produced invalid json: %v", err)
	}
	if got["role"] != "developer" {
		t.Fatalf("role = %v, want the attribution role, not the company kind", got["role"])
	}
	if got["kind"] != "game_brand" || got["label_kind"] != "game_brand" {
		t.Fatalf("company_kind mapping regressed: %v", got)
	}
}

func TestMakerName_PrefersTheRoleThatMadeTheGame(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name   string
		labels []catWorkLabel
		want   string
	}{
		{
			name: "developer wins over a publisher listed first",
			labels: []catWorkLabel{
				{DisplayName: "Dramatic Create", Role: "publisher"},
				{DisplayName: "FAVORITE", Role: "developer"},
			},
			want: "FAVORITE",
		},
		{
			name:   "a doujin work names its circle",
			labels: []catWorkLabel{{DisplayName: "たぬきそふと", Role: "circle"}},
			want:   "たぬきそふと",
		},
		{
			name: "no known role falls back to the first named company",
			labels: []catWorkLabel{
				{DisplayName: "", Role: "developer"},
				{DisplayName: "Kun", Role: "wat"},
			},
			want: "Kun",
		},
		{name: "no companies", labels: nil, want: ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := makerName(ctx, c.labels); got != c.want {
				t.Fatalf("makerName = %q, want %q", got, c.want)
			}
		})
	}
}
