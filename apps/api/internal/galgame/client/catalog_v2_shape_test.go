package client

import (
	"encoding/json"
	"net/url"
	"testing"
)

// The v2 read lane is dormant behind catalogReadsV1, so nothing exercises these
// mappings until the flip. Every case below was captured from a live catalog
// response; the last cutover shipped green because it rewrote its assertions
// alongside the code, and the pages just went blank.

func TestV2CatalogQuery_DetailFacesAskForEveryBlock(t *testing.T) {
	for path, want := range map[string]string{
		"/catalog/works/4":                 v2CatalogDetailInclude["works"],
		"/catalog/characters/7":            v2CatalogDetailInclude["characters"],
		"/catalog/labels/1":                v2CatalogDetailInclude["companies"],
		"/catalog/names/1183":              v2CatalogDetailInclude["credit-names"],
		"/catalog/tags/41":                 v2CatalogDetailInclude["tags"],
		"/catalog/series/1":                v2CatalogDetailInclude["series"],
		"/catalog/labels/1/relation-graph": v2CatalogDetailInclude["companies"],
	} {
		if got := v2CatalogQuery(path, url.Values{}).Get("include"); got != want {
			t.Errorf("%s include = %q, want %q — v2 answers a bare id+name row without it", path, got, want)
		}
	}
}

func TestV2CatalogQuery_ListFacesKeepTheirOwnVocabulary(t *testing.T) {
	if got := v2CatalogQuery("/catalog/labels", url.Values{}).Get("include"); got != "aliases,logo" {
		t.Errorf("companies list include = %q, want aliases,logo (the v1 browse row)", got)
	}
	// The works list must keep the caller's include, remapped, not a full detail set.
	q := v2CatalogQuery("/catalog/works/search", url.Values{"include": {"names,covers,labels"}})
	if got := q.Get("include"); got != "titles,covers,companies" {
		t.Errorf("works search include = %q, want the v1 tokens remapped", got)
	}
}

func TestV2CatalogQuery_SpoilerCeiling(t *testing.T) {
	for _, tc := range []struct{ path, spoilers, want string }{
		{"/catalog/works/4", "2", "major"},
		{"/catalog/works/4", "1", "minor"},
		{"/catalog/works/4", "0", "none"},
		{"/catalog/characters/7", "2", "major"},
	} {
		q := v2CatalogQuery(tc.path, url.Values{"spoilers": {tc.spoilers}})
		if got := q.Get("spoiler"); got != tc.want {
			t.Errorf("%s spoilers=%s -> spoiler=%q, want %q", tc.path, tc.spoilers, got, tc.want)
		}
		if q.Has("spoilers") {
			t.Errorf("%s still sends v1's spoilers=, which v2 drops silently", tc.path)
		}
	}
	// A face with no spoiler axis must not grow the parameter.
	if q := v2CatalogQuery("/catalog/labels/1", url.Values{"spoilers": {"2"}}); q.Has("spoiler") {
		t.Error("companies detail grew a spoiler parameter")
	}
}

func TestV2CatalogQuery_BooleanFlags(t *testing.T) {
	// v1 took 1/0; v2 types these as real booleans and 400s on "1".
	if got := v2CatalogQuery("/catalog/tags", url.Values{"has_works": {"1"}}).Get("has_works"); got != "true" {
		t.Errorf("has_works = %q, want true — \"1\" is a 400", got)
	}
	if q := v2CatalogQuery("/catalog/tags", url.Values{"has_works": {"0"}}); q.Has("has_works") {
		t.Error("has_works=0 should drop the filter, not send false")
	}
}

func rewriteOne(t *testing.T, raw string) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(rewriteV2JSON([]byte(raw), ""), &got); err != nil {
		t.Fatalf("rewrite produced invalid json: %v", err)
	}
	return got
}

func TestRewriteV2JSON_WorkTagCarriesCanonicalID(t *testing.T) {
	// catalog_detail.go drops a tag row whose canonical_id is 0, so v2 naming
	// the same column `id` empties the whole tag panel rather than degrading it.
	got := rewriteOne(t, `{"id":"638","display_name":"key","tag_kind":"meta","spoiler":"minor","is_sexual":false,"tier":"core","work_count":12}`)
	if got["canonical_id"] != float64(638) {
		t.Errorf("canonical_id = %v, want 638", got["canonical_id"])
	}
	if got["spoiler"] != float64(1) {
		t.Errorf("spoiler = %v, want 1 — v2's enum is none|minor|major", got["spoiler"])
	}
	if got["kind"] != "meta" || got["sexual"] != false {
		t.Errorf("tag_kind/is_sexual not folded: %v", got)
	}
	// A standalone tag list row has no spoiler and must not claim a canonical id.
	list := rewriteOne(t, `{"object":"tag","id":"638","display_name":"key","tag_kind":"meta","is_sexual":false}`)
	if _, has := list["canonical_id"]; has {
		t.Error("a tag browse row grew a canonical_id it never had on v1")
	}
}

func TestRewriteV2JSON_ImageGradingIsNumeric(t *testing.T) {
	// The cover block is typed int, so one "safe" made the WHOLE work detail
	// fail to decode rather than just that field.
	for wire, want := range map[string]float64{"safe": 0, "suggestive": 1, "explicit": 2} {
		got := rewriteOne(t, `{"id":"1","url":"u","sexual":"`+wire+`","violence":null}`)
		if got["sexual"] != want {
			t.Errorf("sexual %q -> %v, want %v", wire, got["sexual"], want)
		}
	}
	// A tag's boolean `sexual` must survive untouched.
	got := rewriteOne(t, `{"object":"tag","id":"1","is_sexual":true}`)
	if got["sexual"] != true {
		t.Errorf("tag sexual = %v, want true", got["sexual"])
	}
}

func TestRewriteV2JSON_StaffAndCompanyShapes(t *testing.T) {
	got := rewriteOne(t, `{"object":"credit_name","id":"1183","display_name":"高森奈津美","gender":"female","birth_year":1987,"birth_month":2,"birth_day":14,"photo":{"url":"u","hash":"ph"},"aliases":[{"value":"a","alias_kind":"alias"}]}`)
	if got["gender"] != float64(2) {
		t.Errorf("gender = %v, want 2 (male=1, female=2)", got["gender"])
	}
	if got["birth_y"] != float64(1987) || got["birth_m"] != float64(2) || got["birth_d"] != float64(14) {
		t.Errorf("birth_* not folded: %v", got)
	}
	if got["photo_hash"] != "ph" {
		t.Errorf("photo_hash = %v", got["photo_hash"])
	}
	alias := got["aliases"].([]any)[0].(map[string]any)
	if alias["kind"] != "alias" {
		t.Errorf("alias_kind not folded: %v", alias)
	}
	company := rewriteOne(t, `{"object":"company","id":"1","logo":{"url":"u","hash":"lg"}}`)
	if company["logo_hash"] != "lg" {
		t.Errorf("logo_hash = %v", company["logo_hash"])
	}
}

func TestRewriteV2JSON_TraitLieAndCreditCharacter(t *testing.T) {
	trait := rewriteOne(t, `{"id":"1","display_name":"金发","is_lie":true,"is_sexual":false,"spoiler":"major"}`)
	if trait["lie"] != true {
		t.Errorf("is_lie not folded: %v", trait)
	}
	if trait["spoiler"] != float64(2) {
		t.Errorf("spoiler = %v, want 2", trait["spoiler"])
	}
	role := rewriteOne(t, `{"role_key":"voice-actor","character_id":"7936","character_name":"空門蒼"}`)
	if role["character"] != "空門蒼" {
		t.Errorf("character_name not folded: %v", role)
	}
}

func TestOffsetCursorRoundTrip(t *testing.T) {
	// The sub-faces drop `offset` silently and page by an opaque cursor that is
	// the next offset in base64 — cur_Mw skips exactly three rows on a live
	// catalog, and a limit=50 page answers cur_NTA where v1 said next_offset=50.
	if got := encodeOffsetCursor(3); got != "cur_Mw" {
		t.Errorf("encodeOffsetCursor(3) = %q, want cur_Mw", got)
	}
	if got, ok := decodeOffsetCursor("cur_NTA"); !ok || got != 50 {
		t.Errorf("decodeOffsetCursor(cur_NTA) = %d,%v, want 50,true", got, ok)
	}
	for _, bad := range []string{"", "50", "cur_", "cur_!!!", "cur_" + "eyJzIjoiaWQifQ"} {
		if _, ok := decodeOffsetCursor(bad); ok {
			t.Errorf("decodeOffsetCursor(%q) claimed an offset", bad)
		}
	}
}

func TestIncludeHasToken(t *testing.T) {
	q := url.Values{"include": {"names, credits ,covers"}}
	if !includeHasToken(q, "credits") {
		t.Error("credits not found in a spaced include list")
	}
	if includeHasToken(q, "cred") {
		t.Error("a prefix matched a token")
	}
	if includeHasToken(url.Values{}, "credits") {
		t.Error("an absent include matched")
	}
}

func TestRewriteV2JSON_CoverKindIsTheCoverGrouping(t *testing.T) {
	// Captured from work 3 on the running catalog: v2 spec gate G8 forbids a
	// bare `kind`, so #86 shipped the grouping as cover_kind and a cover with
	// no grouping sends "" where v1 omitted the field entirely.
	got := rewriteOne(t, `{"id":"6689","cover_kind":"main","portrait_pinned":false,"url":"u","hash":"h","sexual":"safe","violence":null,"source":"vndb"}`)
	if got["kind"] != "main" {
		t.Errorf("kind = %v, want main — the gallery groups on it", got["kind"])
	}
	blank := rewriteOne(t, `{"id":"6688","cover_kind":"","url":"u","sexual":"safe"}`)
	if blank["kind"] != "" {
		t.Errorf("kind = %v, want the empty grouping v1 sent as absent", blank["kind"])
	}
}

func TestRewriteV2JSON_RollupViaCompanyIsViaLabel(t *testing.T) {
	// Captured from company 1's rollup: 13 of its rows arrive one hop away, and
	// without the alias every one of them reads as directly attributed — the
	// 39/13 own/imprint split the company page prints becomes 52/0.
	got := rewriteOne(t, `{"id":"671","via_company":{"object":"company","id":"713","display_name":"Sweets","localized":{}}}`)
	via, ok := got["via_label"].(map[string]any)
	if !ok {
		t.Fatalf("via_label = %v, want the via_company object", got["via_label"])
	}
	if via["id"] != float64(713) || via["display_name"] != "Sweets" {
		t.Errorf("via_label = %v, want id 713 / Sweets with the id coerced off v2's string", via)
	}
	// A directly attributed row must not grow one.
	if direct := rewriteOne(t, `{"id":"672"}`); direct["via_label"] != nil {
		t.Error("a directly attributed rollup row grew a via_label")
	}
}

// Captured from api.nextmoe.dev/v2/catalog/works/198?include=relations on
// 2026-10-13. Relation ends carried no cover then; the cover on 2679 is the
// CoverSlot shape the work's own cover already has, which infra 2.48.0 fills in.
func TestRewriteV2JSON_RelationWorksDecode(t *testing.T) {
	raw := `{"object":"work","id":"198","relations":[
		{"relation_type":"fandisc_of","phrase":"有 Fandisc","work":{"object":"work","id":"2679","medium":"galgame","display_name":"Rewrite: Cradles Tale","latin":null,"localized":{"zh-Hans":{"value":"Rewrite 特典后日谈","is_machine":false}},"olang":"ja","content_rating":"all_ages","content_limit":"sfw","release_date":"2011-12-01","release_date_precision":"month","release_status":"released","cover":{"url":"https://image.kungal.iloveren.link/64/83/6483076809b395c9688b3726b147cc0b25d432448edde674e4fb1d7f481b5507.webp","hash":"6483076809b395c9688b3726b147cc0b25d432448edde674e4fb1d7f481b5507","width":318,"height":450,"thumbhash":"6jgKDQLImYd3+Xe4iIaJl7AAlfoD","sexual":"suggestive","violence":null,"source":"dlsite","origin":"cover"},"banner":null,"claim":{"site":"kungal","site_work_id":"2679","state":"live","content_limit":"sfw"}}},
		{"relation_type":"adaptation_of","phrase":"被改编为","work":{"object":"work","id":"213225","medium":"manga","display_name":"Rewrite: SIDE-B","latin":null,"localized":{},"olang":"ja","content_rating":"all_ages","content_limit":"sfw","release_date":null,"release_date_precision":null,"release_status":"unknown","cover":null,"banner":null,"claim":null}}
	]}`
	var d CatalogWorkDetail
	if err := json.Unmarshal(rewriteV2JSON([]byte(raw), ""), &d); err != nil {
		t.Fatal(err)
	}
	if len(d.Relations) != 2 {
		t.Fatalf("relations = %d, want 2", len(d.Relations))
	}
	fd := d.Relations[0]
	if fd.RelationType != "fandisc_of" || fd.Phrase != "有 Fandisc" || fd.Work.ID != 2679 || fd.Work.Medium != "galgame" {
		t.Errorf("fandisc edge = %+v", fd)
	}
	if fd.Work.Claim == nil || fd.Work.Claim.State != "live" || fd.Work.Claim.SiteWorkID != 2679 {
		t.Errorf("claim = %+v", fd.Work.Claim)
	}
	if fd.Work.ReleaseDatePrecision == nil || *fd.Work.ReleaseDatePrecision != "month" {
		t.Errorf("release_date_precision = %v, want month", fd.Work.ReleaseDatePrecision)
	}
	slot := fd.Work.CoverSlots
	if slot == nil || slot.Portrait == nil || slot.Portrait.Width != 318 || slot.Portrait.Thumbhash == "" || slot.Portrait.Sexual == nil || *slot.Portrait.Sexual != 1 {
		t.Errorf("the cover object must survive as a portrait slot with its size, thumbhash and grade: %+v", slot)
	}
	if manga := d.Relations[1].Work; manga.ID != 213225 || manga.Medium != "manga" || manga.Claim != nil || manga.CoverSlots != nil {
		t.Errorf("manga end = %+v", manga)
	}
}
