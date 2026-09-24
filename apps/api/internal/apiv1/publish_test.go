package apiv1

import (
	"strings"
	"testing"
)

const publishFixture = `{"components":{"schemas":{"Thing":{"properties":{` +
	`"kind":{"description":"k","enum":["b","a"],"maxLength":1,"type":"string","x-vocabulary-closed":true},` +
	`"maybe":{"description":"m","enum":["a","b",null],"maxLength":1,"type":["string","null"],"x-vocabulary-closed":true},` +
	`"object":{"description":"o","enum":["thing"],"maxLength":5,"type":"string","x-vocabulary-closed":true}},"type":"object"}}},` +
	`"paths":{"/things":{"get":{"parameters":[{"in":"query","name":"k","schema":` +
	`{"default":"a","description":"q","enum":["a","b"],"maxLength":1,"type":"string","x-vocabulary-closed":true}}]}}}}`

func TestPublishNamesVocabulariesAndConstsDiscriminants(t *testing.T) {
	out, err := publishWith([]byte(publishFixture), []vocabulary{{"Kind", []string{"a", "b"}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"Kind":{"enum":["a","b"],"maxLength":1,"type":"string","x-vocabulary-closed":true}`,
		`"kind":{"$ref":"#/components/schemas/Kind","description":"k"}`,
		`"maybe":{"anyOf":[{"$ref":"#/components/schemas/Kind"},{"type":"null"}],"description":"m"}`,
		`"object":{"const":"thing","description":"o","maxLength":5,"type":"string"}`,
		`"schema":{"$ref":"#/components/schemas/Kind","default":"a","description":"q"}`,
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("published spec lacks %s\n%s", want, out)
		}
	}
}

func TestPublishRefusesAnUnnamedVocabulary(t *testing.T) {
	_, err := publishWith([]byte(publishFixture), nil)
	if err == nil || !strings.Contains(err.Error(), "{a,b} at Thing.kind") {
		t.Fatalf("err = %v", err)
	}
}

func TestPublishRefusesAVocabularyNamedLikeAComponent(t *testing.T) {
	_, err := publishWith([]byte(publishFixture), []vocabulary{{"Thing", []string{"a", "b"}}})
	if err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("err = %v", err)
	}
}
