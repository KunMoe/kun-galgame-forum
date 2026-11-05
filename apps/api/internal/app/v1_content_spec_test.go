package app

import (
	"encoding/json"
	"slices"
	"testing"

	"kun-galgame-api/internal/apiv1"
)

type publishedSpec struct {
	Components struct {
		Schemas map[string]struct {
			Properties map[string]struct {
				Ref string `json:"$ref"`
			} `json:"properties"`
			OneOf []struct {
				Ref string `json:"$ref"`
			} `json:"oneOf"`
			Discriminator struct {
				PropertyName string            `json:"propertyName"`
				Mapping      map[string]string `json:"mapping"`
			} `json:"discriminator"`
		} `json:"schemas"`
	} `json:"components"`
}

func TestV1SpecPublishesTheContentNodeVocabulary(t *testing.T) {
	raw, err := apiv1.MarshalOpenAPI(V1Spec())
	if err != nil {
		t.Fatal(err)
	}
	var spec publishedSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatal(err)
	}
	topic, ok := spec.Components.Schemas["Topic"]
	if !ok {
		t.Fatal("Topic is not in the published spec")
	}
	if got := topic.Properties["content"].Ref; got != "#/components/schemas/ContentDocument" {
		t.Fatalf("Topic.content $ref = %q", got)
	}
	if _, ok := spec.Components.Schemas["ContentDocument"]; !ok {
		t.Fatal("ContentDocument is not in the published spec")
	}
	unions := map[string][]string{
		"BlockNode": {"blockquote", "code", "entity_card", "heading", "list", "math", "paragraph", "spoiler", "table", "thematic_break"},
		"InlineNode": {"break", "emphasis", "entity_link", "image", "inline_code", "inline_math", "inline_spoiler", "link",
			"mention", "reply_reference", "strikethrough", "strong", "text", "video"},
		"EntityTarget": {"character", "company", "credit_name", "galgame_resource", "rating", "series", "tag", "topic", "unavailable_entity", "user", "work"},
	}
	for name, want := range unions {
		u, ok := spec.Components.Schemas[name]
		if !ok {
			t.Fatalf("%s is not in the published spec", name)
		}
		if u.Discriminator.PropertyName != "object" {
			t.Errorf("%s discriminates on %q", name, u.Discriminator.PropertyName)
		}
		var tokens, refs []string
		for tok, ref := range u.Discriminator.Mapping {
			tokens = append(tokens, tok)
			refs = append(refs, ref)
			if _, ok := spec.Components.Schemas[ref[len("#/components/schemas/"):]]; !ok {
				t.Errorf("%s maps %s to %s, which is not published", name, tok, ref)
			}
		}
		slices.Sort(tokens)
		if !slices.Equal(tokens, want) {
			t.Errorf("%s tokens = %v, want %v", name, tokens, want)
		}
		var oneOf []string
		for _, o := range u.OneOf {
			oneOf = append(oneOf, o.Ref)
		}
		slices.Sort(oneOf)
		slices.Sort(refs)
		if !slices.Equal(oneOf, refs) {
			t.Errorf("%s oneOf %v differs from its mapping %v", name, oneOf, refs)
		}
	}
}
