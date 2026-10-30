package walkthroughapiv1

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"

	"kun-galgame-api/internal/galgame/client"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/problem"
)

const catalogDetail = `{"object":"walkthrough","id":"7","work_id":"53","work_content_limit":"sfw","source":"seiya","source_url":"https://seiya-saiga.com/game/x.html",
 "source_author":null,"source_updated_on":"2023-12-07","lang":"ja",
 "characters":[{"key":"c1","name":"寧々","character_id":"2813","display_name":"綾地寧々","localized":{"zh-Hans":{"value":"绫地宁宁","is_machine":false}}},
               {"key":"c2","name":"モブ","character_id":null,"display_name":null,"localized":{}}],
 "ending_count":1,"save_count":1,"segment_count":1,"updated_at":"2026-10-24T00:00:00Z",
 "chart":{"sections":[],"legend":[{"text":"★の選択肢は推理ランクに影響のある選択肢です","localized":{"zh-Hans":{"value":"★选项会影响推理评级","is_machine":true}}}],"entries":[{"key":"start","label":"START","unlocked_by_ending_key":null}],
  "saves":[{"key":"sv01","label":"セーブ01"}],
  "endings":[{"key":"e1","name":"寧々END1","localized":{"zh-Hans":{"value":"宁宁END1","is_machine":true}},"ending_type":"heroine","character_keys":["c1"]}],
  "segments":[{"key":"g1","section_key":null,"from_node":{"node_type":"entry","key":"start"},"condition":null,"condition_localized":{},
   "steps":[{"step_type":"choice","text":"はい","localized":{},"alternatives":[{"text":"いいえ","localized":{}}],"remark":"推理正解","remark_localized":{"zh-Hans":{"value":"推理正确","is_machine":true}},"achievement":false,"save_key":null},
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
	found     bool
	limit     string
}

func (f *fakeCatalog) CatalogWalkthroughs(_ context.Context, q url.Values) (*client.CatalogWalkthroughPage, *legacyErrors.AppError) {
	f.lastQuery = q
	var page client.CatalogWalkthroughPage
	return &page, decode(json.Unmarshal([]byte(`{"object":"list","items":[`+catalogDetail+`]}`), &page))
}

func (f *fakeCatalog) CatalogWalkthrough(_ context.Context, _ string) (*client.CatalogWalkthrough, bool, *legacyErrors.AppError) {
	if !f.found {
		return nil, false, nil
	}
	var w client.CatalogWalkthrough
	if err := json.Unmarshal([]byte(catalogDetail), &w); err != nil {
		return nil, false, decode(err)
	}
	if f.limit != "" {
		w.WorkContentLimit = f.limit
	}
	return &w, true, nil
}

func TestGetWalkthroughMapsCharactersAndChart(t *testing.T) {
	cat := &fakeCatalog{found: true}
	out, err := New(cat).getWalkthrough(context.Background(), &detailInput{WalkthroughID: "7"})
	if err != nil {
		t.Fatal(err)
	}
	w := out.Body
	if w.IsNSFW {
		t.Error("work_content_limit=sfw is not is_nsfw")
	}
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
	choice, save := w.Chart.ChartSegments[0].Steps[0], w.Chart.ChartSegments[0].Steps[1]
	if choice.Remark == nil || *choice.Remark != "推理正解" || choice.RemarkLocalized["zh-Hans"].Value != "推理正确" {
		t.Errorf("choice remark = %v %+v", choice.Remark, choice.RemarkLocalized)
	}
	if save.Remark != nil || save.RemarkLocalized == nil {
		t.Errorf("a step without a remark = %v %+v", save.Remark, save.RemarkLocalized)
	}
	if l := w.Chart.ChartLegend; len(l) != 1 || l[0].Localized["zh-Hans"].Value != "★选项会影响推理评级" {
		t.Errorf("legend = %+v", l)
	}
}

func TestChartFromACatalogWithoutLegendIsAnEmptyArray(t *testing.T) {
	b, err := json.Marshal(chartFrom(client.CatalogWalkthroughChart{}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"chart_legend":[]`) {
		t.Errorf("chart = %s", b)
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
	if cat.lastQuery.Get("nsfw") != "true" || cat.lastQuery.Get("content_limit") != "sfw" {
		t.Errorf("an SFW list must open the age gate and filter on content_limit, query = %v", cat.lastQuery)
	}
	if len(out.Body.Items) != 1 || out.Body.Items[0].Chart != nil {
		t.Errorf("items = %+v", out.Body.Items)
	}
}

func TestListWalkthroughsWithNSFWAppliesNoDisplayGate(t *testing.T) {
	cat := &fakeCatalog{}
	if _, err := New(cat).listWalkthroughs(context.Background(), &listInput{Limit: 20, IncludeNSFW: true}); err != nil {
		t.Fatal(err)
	}
	if cat.lastQuery.Get("nsfw") != "true" || cat.lastQuery.Has("content_limit") {
		t.Errorf("query = %v", cat.lastQuery)
	}
}

func TestGetWalkthroughIsNSFWFailsClosed(t *testing.T) {
	for _, limit := range []string{"nsfw", "bogus"} {
		out, err := New(&fakeCatalog{found: true, limit: limit}).getWalkthrough(context.Background(), &detailInput{WalkthroughID: "7"})
		if err != nil {
			t.Fatal(err)
		}
		if !out.Body.IsNSFW {
			t.Errorf("work_content_limit=%q must be is_nsfw", limit)
		}
	}
}

func TestGetWalkthroughNativeHasNoSourcePage(t *testing.T) {
	native := strings.Replace(catalogDetail, `"source":"seiya","source_url":"https://seiya-saiga.com/game/x.html"`, `"source":"nextmoe","source_url":null`, 1)
	native = strings.Replace(native, `"lang":"ja"`, `"lang":"zh-Hans"`, 1)
	var w client.CatalogWalkthrough
	if err := json.Unmarshal([]byte(native), &w); err != nil {
		t.Fatal(err)
	}
	out := walkthroughFrom(w)
	if out.SourceSite != "nextmoe" || out.SourcePageURL != nil || out.TextLang != "zh-Hans" {
		t.Errorf("native walkthrough = site %q, url %v, lang %q", out.SourceSite, out.SourcePageURL, out.TextLang)
	}
}

func TestGetWalkthroughCarriesRouteHintsAndRelease(t *testing.T) {
	withRelease := strings.Replace(catalogDetail, `"work_content_limit":"sfw",`, `"work_content_limit":"sfw","work_route_hints":true,
 "release":{"id":"901","title":null,"release_kind":"patch","lang":"zh-Hans","companies":["某汉化组"]},`, 1)
	var w client.CatalogWalkthrough
	if err := json.Unmarshal([]byte(withRelease), &w); err != nil {
		t.Fatal(err)
	}
	out := walkthroughFrom(w)
	if !out.HasInGameRouteHints {
		t.Error("work_route_hints=true must be has_in_game_route_hints")
	}
	r := out.FollowedRelease
	if r == nil || r.ID != "901" || r.ReleaseKind != "patch" || r.ReleaseTitle != nil || len(r.CompanyNames) != 1 || r.CompanyNames[0] != CompanyName("某汉化组") {
		t.Fatalf("release = %+v", r)
	}

	var plain client.CatalogWalkthrough
	if err := json.Unmarshal([]byte(catalogDetail), &plain); err != nil {
		t.Fatal(err)
	}
	if out := walkthroughFrom(plain); out.HasInGameRouteHints || out.FollowedRelease != nil {
		t.Errorf("a walkthrough without the fields = hints %v, release %+v", out.HasInGameRouteHints, out.FollowedRelease)
	}
}

func TestGetWalkthroughCarriesPointTranslationsAndPrerequisites(t *testing.T) {
	spec276 := strings.Replace(catalogDetail, `"saves":[{"key":"sv01","label":"セーブ01"}]`,
		`"saves":[{"key":"sv01","label":"セーブ01","localized":{"zh-Hans":{"value":"存档01","is_machine":true}}}]`, 1)
	spec276 = strings.Replace(spec276, `"condition":null,"condition_localized":{},`,
		`"condition":"END後に","condition_localized":{},"requires_ending_keys":["e1"],`, 1)
	spec276 = strings.Replace(spec276, `"step_type":"choice","text":"はい"`, `"step_type":"heading","text":"５月１４日"`, 1)
	var w client.CatalogWalkthrough
	if err := json.Unmarshal([]byte(spec276), &w); err != nil {
		t.Fatal(err)
	}
	c := walkthroughFrom(w).Chart
	if got := c.SavePoints[0].Localized["zh-Hans"].Value; got != "存档01" {
		t.Errorf("save localized = %q", got)
	}
	if got := c.ChartSegments[0].RequiresEndingKeys; len(got) != 1 || got[0] != "e1" {
		t.Errorf("requires_ending_keys = %v", got)
	}
	if got := c.ChartSegments[0].Steps[0].StepType; got != "heading" {
		t.Errorf("step_type = %q", got)
	}

	var old client.CatalogWalkthrough
	if err := json.Unmarshal([]byte(catalogDetail), &old); err != nil {
		t.Fatal(err)
	}
	oc := walkthroughFrom(old).Chart
	if oc.ChartSegments[0].RequiresEndingKeys == nil || oc.SavePoints[0].Localized == nil || oc.EntryPoints[0].Localized == nil {
		t.Error("a catalog before spec 2.76.0 must still answer [] and {}, never null")
	}
}
