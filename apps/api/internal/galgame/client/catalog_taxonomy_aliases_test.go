package client

import (
	"cmp"
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

// One CatalogTaxonomyItem reads four vocabularies whose aliases[] are mid-
// migration: the label browse row sends wave 209's object rows, engines still
// send bare strings. Decoding only one of the two returns 解析 Catalog 词表响应失败
// for every row of the other, which is what killed /galgame/official.
func TestCatalogTaxonomyList_DecodesBothAliasShapes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/catalog/companies":
			_, _ = w.Write([]byte(`{"items":[
				{"id":"13","display_name":"ねこねこソフト","company_kind":"game_brand","work_count":9,
				 "localized":{"zh-Hans":{"value":"猫猫社","kind":"translation"}},
				 "aliases":[{"value":"猫猫社","lang":"zh-Hans","kind":"translation","is_machine":true},
				            {"value":"NekoNeko-soft","lang":"en","kind":"spelling_variant"}]}],
				"next_cursor":null,"total":1}`))
		case "/v2/catalog/engines":
			_, _ = w.Write([]byte(`{"items":[
				{"id":"4","name":"KiriKiri","work_count":88,"description":"","aliases":["KRKR",""]}],
				"next_cursor":null,"total":1}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	c := New(srv.URL, "test-key", "")
	ctx := context.Background()

	labels, appErr := c.CatalogTaxonomyList(ctx, "labels", nil)
	if appErr != nil {
		t.Fatalf("labels: %v", appErr)
	}
	label := labels.Items[0]
	if !label.Aliases[0].Machine || label.Aliases[0].Lang != "zh-Hans" {
		t.Errorf("alias row = %+v, want the language and machine flag kept", label.Aliases[0])
	}
	if got := label.Aliases.Values(CatalogEntityName(ctx, label.Localized, label.DisplayName, "")); !slices.Equal(got, []string{"NekoNeko-soft"}) {
		t.Errorf("label aliases = %v, want the rendered 猫猫社 dropped from its own alias list", got)
	}

	engines, appErr := c.CatalogTaxonomyList(ctx, "engines", nil)
	if appErr != nil {
		t.Fatalf("engines: %v", appErr)
	}
	engine := engines.Items[0]
	if got := engine.Aliases.Values(cmp.Or(engine.DisplayName, engine.Name)); !slices.Equal(got, []string{"KRKR"}) {
		t.Errorf("engine aliases = %v, want the bare strings decoded and the blank dropped", got)
	}
}
