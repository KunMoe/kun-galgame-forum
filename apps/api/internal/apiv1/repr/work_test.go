package repr

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewWorkRef(t *testing.T) {
	ref := NewWorkRef(19658, NewCatalogName("紅殻のパンドラ", "", nil), nil, true)
	raw, err := json.Marshal(ref)
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	for _, want := range []string{`"object":"work"`, `"id":"19658"`, `"display_name":"紅殻のパンドラ"`,
		`"latin":null`, `"localized":{}`, `"cover":null`, `"is_nsfw":true`} {
		if !strings.Contains(got, want) {
			t.Errorf("%s missing %s", got, want)
		}
	}
	if strings.Contains(got, "CatalogName") {
		t.Errorf("the name trio must be flattened: %s", got)
	}

	var zero CatalogName
	if NewWorkRef(1, zero, nil, false).Localized == nil {
		t.Error("localized must never be null, even from a zero CatalogName")
	}
	if n := NewCatalogName("x", "Koukaku", nil); n.Latin == nil || *n.Latin != "Koukaku" {
		t.Errorf("latin %v", n.Latin)
	}
}
