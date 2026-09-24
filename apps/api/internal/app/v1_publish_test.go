package app

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"kun-galgame-api/internal/apiv1"
)

// Publishing may only rename: undo it and the spec must be exactly what huma
// wrote, up to enum order and the redundant maxLength on an enum.
func TestPublishedSpecMeansWhatHumaWrote(t *testing.T) {
	api := V1Spec()
	raw, err := api.OpenAPI().MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	published, err := apiv1.MarshalOpenAPI(api)
	if err != nil {
		t.Fatal(err)
	}
	var want, got map[string]any
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(published, &got); err != nil {
		t.Fatal(err)
	}
	schemas := got["components"].(map[string]any)["schemas"].(map[string]any)
	vocab := map[string][]any{}
	for _, name := range apiv1.PublishedVocabularies() {
		c, ok := schemas[name].(map[string]any)
		if !ok {
			t.Errorf("vocabulary %s is not in the published spec; delete it from publish_vocabularies.go", name)
			continue
		}
		vocab["#/components/schemas/"+name] = c["enum"].([]any)
		delete(schemas, name)
	}
	unpublish(got, vocab)
	normalizeEnums(want)
	normalizeEnums(got)
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("published spec differs from huma's: %s", firstDiff("", want, got))
	}
}

func unpublish(v any, vocab map[string][]any) {
	switch x := v.(type) {
	case map[string]any:
		if ref, ok := x["$ref"].(string); ok && vocab[ref] != nil {
			delete(x, "$ref")
			x["type"], x["enum"], x["x-vocabulary-closed"] = "string", slices.Clone(vocab[ref]), true
		}
		if alts, ok := x["anyOf"].([]any); ok && len(alts) == 2 {
			ref, _ := alts[0].(map[string]any)["$ref"].(string)
			if values := vocab[ref]; values != nil && reflect.DeepEqual(alts[1], map[string]any{"type": "null"}) {
				delete(x, "anyOf")
				x["type"] = []any{"string", "null"}
				x["enum"] = append(slices.Clone(values), nil)
				x["x-vocabulary-closed"] = true
			}
		}
		if c, ok := x["const"].(string); ok {
			delete(x, "const")
			x["enum"], x["x-vocabulary-closed"] = []any{c}, true
		}
		for _, c := range x {
			unpublish(c, vocab)
		}
	case []any:
		for _, c := range x {
			unpublish(c, vocab)
		}
	}
}

func normalizeEnums(v any) {
	switch x := v.(type) {
	case map[string]any:
		for _, c := range x {
			normalizeEnums(c)
		}
		if enum, ok := x["enum"].([]any); ok {
			delete(x, "maxLength")
			slices.SortFunc(enum, func(a, b any) int { return strings.Compare(fmt.Sprint(a), fmt.Sprint(b)) })
		}
	case []any:
		for _, c := range x {
			normalizeEnums(c)
		}
	}
}

func firstDiff(path string, a, b any) string {
	switch x := a.(type) {
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok {
			return path
		}
		keys := make([]string, 0, len(x)+len(y))
		for k := range x {
			keys = append(keys, k)
		}
		for k := range y {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range slices.Compact(keys) {
			if !reflect.DeepEqual(x[k], y[k]) {
				return firstDiff(path+"/"+k, x[k], y[k])
			}
		}
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return fmt.Sprintf("%s (%v vs %v)", path, a, b)
		}
		for i := range x {
			if !reflect.DeepEqual(x[i], y[i]) {
				return firstDiff(fmt.Sprintf("%s/%d", path, i), x[i], y[i])
			}
		}
	}
	return fmt.Sprintf("%s (%v vs %v)", path, a, b)
}
