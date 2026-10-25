package walkthroughapiv1

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"testing"

	"kun-galgame-api/internal/galgame/client"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/problem"
)

const catalogDetail = `{"object":"walkthrough","id":"7","work_id":"53","source":"seiya","source_url":"https://seiya-saiga.com/game/x.html",
 "source_author":null,"source_updated_on":"2023-12-07","lang":"ja",
 "characters":[{"key":"c1","name":"寧々","character_id":"2813","display_name":"綾地寧々","localized":{"zh-Hans":{"value":"绫地宁宁","is_machine":false}}},
               {"key":"c2","name":"モブ","character_id":null,"display_name":null,"localized":{}}],
 "ending_count":1,"save_count":1,"segment_count":1,"updated_at":"2026-10-24T00:00:00Z",
 "chart":{"sections":[],"entries":[{"key":"start","label":"START","unlocked_by_ending_key":null}],
  "saves":[{"key":"sv01","label":"セーブ01"}],
  "endings":[{"key":"e1","name":"寧々END1","localized":{"zh-Hans":{"value":"宁宁END1","is_machine":true}},"ending_type":"heroine","character_keys":["c1"]}],
  "segments":[{"key":"g1","section_key":null,"from_node":{"node_type":"entry","key":"start"},"condition":null,"condition_localized":{},
   "steps":[{"step_type":"choice","text":"はい","localized":{},"alternatives":[{"text":"いいえ","localized":{}}],"achievement":false,"save_key":null},
            {"step_type":"save","text":null,"localized":{},"alternatives":[],"achievement":false,"save_key":"sv01"}],
   "to_node":{"node_type":"ending","key":"e1"}}]}}`

func decode(err error) *legacyErrors.AppError {
	if err != nil {
		return legacyErrors.ErrInternal(err.Error())
	}
	return nil
}

type fakeCatalog struct {
	lastQuery url.Values
	lastNSFW  bool
	found     bool
}

func (f *fakeCatalog) CatalogWalkthroughs(_ context.Context, q url.Values) (*client.CatalogWalkthroughPage, *legacyErrors.AppError) {
	f.lastQuery = q
	var page client.CatalogWalkthroughPage
	return &page, decode(json.Unmarshal([]byte(`{"object":"list","items":[`+catalogDetail+`]}`), &page))
}

func (f *fakeCatalog) CatalogWalkthrough(_ context.Context, _ string, nsfw bool) (*client.CatalogWalkthrough, bool, *legacyErrors.AppError) {
	f.lastNSFW = nsfw
	if !f.found {
		return nil, false, nil
	}
	var w client.CatalogWalkthrough
	return &w, true, decode(json.Unmarshal([]byte(catalogDetail), &w))
}

func TestGetWalkthroughMapsCharactersAndChart(t *testing.T) {
	cat := &fakeCatalog{found: true}
	out, err := New(cat).getWalkthrough(context.Background(), &detailInput{WalkthroughID: "7", IncludeNSFW: true})
	if err != nil {
		t.Fatal(err)
	}
	if !cat.lastNSFW {
		t.Error("include_nsfw=true must reach catalog as nsfw=true")
	}
	w := out.Body
	if len(w.ChartCharacters) != 2 || w.ChartCharacters[0].CatalogCharacter == nil {
		t.Fatalf("characters = %+v", w.ChartCharacters)
	}
	if got := w.ChartCharacters[0].CatalogCharacter.CatalogName; got.DisplayName != "綾地寧々" || got.Localized["zh-Hans"].Value != "绫地宁宁" {
		t.Errorf("pinned character name = %+v", got)
	}
	if w.ChartCharacters[1].CatalogCharacter != nil {
		t.Error("an unpinned name carries no character")
	}
	if w.Chart == nil || w.Chart.ChartEndings[0].Localized["zh-Hans"].Value != "宁宁END1" {
		t.Fatalf("chart endings = %+v", w.Chart)
	}
	if k := w.Chart.ChartSegments[0].Steps[1].SaveKey; k == nil || *k != "sv01" {
		t.Errorf("save step key = %v", k)
	}
}

func TestGetWalkthroughNotFound(t *testing.T) {
	_, err := New(&fakeCatalog{}).getWalkthrough(context.Background(), &detailInput{WalkthroughID: "7"})
	var p *problem.Problem
	if !errors.As(err, &p) || p.Code != problem.CodeNotFound {
		t.Fatalf("err = %v, want NOT_FOUND", err)
	}
}

func TestListWalkthroughsDropsTheChartAndForwardsFilters(t *testing.T) {
	cat := &fakeCatalog{}
	out, err := New(cat).listWalkthroughs(context.Background(), &listInput{WorkID: "53", CharacterID: "2813", Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if cat.lastQuery.Get("work_id") != "53" || cat.lastQuery.Get("character_id") != "2813" {
		t.Errorf("query = %v", cat.lastQuery)
	}
	if cat.lastQuery.Has("nsfw") {
		t.Error("nsfw is sent only for include_nsfw=true")
	}
	if len(out.Body.Items) != 1 || out.Body.Items[0].Chart != nil {
		t.Errorf("items = %+v", out.Body.Items)
	}
}
