package client

import (
	"context"
	"encoding/json"
	"testing"

	"kun-galgame-api/pkg/namepref"
)

func originalNames() context.Context {
	return namepref.With(context.Background(), true)
}

// The preference has to reach every projection at once. A reader who asks for
// 原名 and gets a page where the title flipped but the roster, the credits and
// the 会社 did not has a page in two languages, which is worse than either.
func TestNamePreference_ReachesEveryProjection(t *testing.T) {
	const wire = `{"display_name":"ワルキューレロマンツェ","latin":"Walkure Romanze",` +
		`"localized":{"zh-Hans":{"value":"少女骑士物语","kind":"translation"}}}`

	renders := map[string]func(context.Context) string{
		"work list item": func(ctx context.Context) string {
			var v CatalogWorkListItem
			mustDecode(t, wire, &v)
			name, _ := v.Names(ctx)
			return name
		},
		"roster character": func(ctx context.Context) string {
			var v catWorkCharacter
			mustDecode(t, wire, &v)
			name, _ := CatalogEntityNames(ctx, v.Localized, v.DisplayName, v.Latin)
			return name
		},
		"credit": func(ctx context.Context) string {
			var v catCreditItem
			mustDecode(t, wire, &v)
			return v.Name(ctx)
		},
		"voice": func(ctx context.Context) string {
			var v CatalogPerson
			mustDecode(t, wire, &v)
			return v.Name(ctx)
		},
		"work label": func(ctx context.Context) string {
			var v catWorkLabel
			mustDecode(t, wire, &v)
			return v.Name(ctx)
		},
		"search hit": func(ctx context.Context) string {
			var v CatalogEntityHit
			mustDecode(t, wire, &v)
			return v.Name(ctx)
		},
	}

	for what, render := range renders {
		if got := render(context.Background()); got != "少女骑士物语" {
			t.Errorf("%s by default = %q, want the Chinese name", what, got)
		}
		if got := render(originalNames()); got != "ワルキューレロマンツェ" {
			t.Errorf("%s under 原名 = %q, want the record's own name", what, got)
		}
	}
}

// The secondary line is whichever name the reader did not pick, so the pair
// never repeats itself and never loses the other name.
func TestNamePreference_SecondaryLineIsTheOtherName(t *testing.T) {
	localized := map[string]catLocalizedName{"zh-Hans": {Value: "少女骑士物语"}}

	name, other := CatalogEntityNames(context.Background(), localized, "ワルキューレロマンツェ", "")
	if name != "少女骑士物语" || other != "ワルキューレロマンツェ" {
		t.Errorf("default = %q / %q, want 中文 over 原名", name, other)
	}

	name, other = CatalogEntityNames(originalNames(), localized, "ワルキューレロマンツェ", "")
	if name != "ワルキューレロマンツェ" || other != "少女骑士物语" {
		t.Errorf("under 原名 = %q / %q, want 原名 over 中文", name, other)
	}
}

func TestNamePreference_FallsBackWhenTheOtherNameIsMissing(t *testing.T) {
	zhOnly := map[string]catLocalizedName{"zh-Hans": {Value: "水野贵弘"}}

	if got := CatalogEntityName(originalNames(), zhOnly, "", ""); got != "水野贵弘" {
		t.Errorf("no display_name under 原名 = %q, want the Chinese name rather than a blank", got)
	}
	if got := CatalogEntityName(originalNames(), nil, "", "Maeda Jun"); got != "Maeda Jun" {
		t.Errorf("latin only under 原名 = %q, want the latin name", got)
	}
	if _, other := CatalogEntityNames(originalNames(), zhOnly, "水野贵弘", ""); other != "" {
		t.Errorf("secondary = %q, want empty when both names are the same string", other)
	}
}

func mustDecode(t *testing.T, raw string, into any) {
	t.Helper()
	if err := json.Unmarshal([]byte(raw), into); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
}
