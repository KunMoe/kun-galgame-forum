package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"kun-galgame-api/internal/galgame/dto"
)

func detailStub(t *testing.T, workID int64, body string) (*httptest.Server, *url.Values) {
	t.Helper()
	var seen url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if req.URL.Path == "/v2/catalog/works/"+itoa(workID) {
			seen = req.URL.Query()
			_, _ = w.Write([]byte(body))
			return
		}
		t.Errorf("unexpected upstream call: %s", req.URL.Path)
		_, _ = w.Write([]byte(`{"object":"work","id":"0"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &seen
}

func fullOf(t *testing.T, body string) dto.NextMoeGalgameDetailFull {
	t.Helper()
	const workID int64 = 4242
	srv, _ := detailStub(t, workID, body)
	c := New(srv.URL, "nm_test_key", "")

	d, found, _, appErr := c.CatalogWorkDetail(context.Background(), int(workID))
	if appErr != nil {
		t.Fatalf("CatalogWorkDetail: %v", appErr)
	}
	if !found {
		t.Fatal("CatalogWorkDetail: not found, want the stubbed work")
	}
	return CatalogDetailToFull(context.Background(), d, int(workID))
}

func TestCatalogDetail_HeroPrefersTheLandscapeCover(t *testing.T) {
	t.Run("landscape present", func(t *testing.T) {
		body := `{"object":"work","id":"4242","display_name":"Kun","content_rating":"all_ages","olang":"ja","covers":[
			{"url":"https://cdn.example/ab/cd/portrait.webp","cover_kind":"main","width":600,"height":800,"thumbhash":"P"},
			{"url":"https://cdn.example/ef/gh/banner.webp","cover_kind":"dig","width":1280,"height":720,"thumbhash":"B"}
		]}`
		f := fullOf(t, body)
		if f.EffectiveBannerURL != "https://cdn.example/ef/gh/banner.webp" {
			t.Errorf("hero = %q, want the landscape cover", f.EffectiveBannerURL)
		}
		if f.EffectiveBannerWidth != 1280 || f.EffectiveBannerHeight != 720 || f.EffectiveBannerThumbhash != "B" {
			t.Errorf("dims/thumbhash = %dx%d %q, want 1280x720 B",
				f.EffectiveBannerWidth, f.EffectiveBannerHeight, f.EffectiveBannerThumbhash)
		}
		if len(f.Covers) != 2 || f.Covers[0].CDNURL != "https://cdn.example/ab/cd/portrait.webp" {
			t.Errorf("covers[] order changed: %+v", f.Covers)
		}
	})

	t.Run("portrait only falls back", func(t *testing.T) {
		body := `{"object":"work","id":"4242","display_name":"Kun","content_rating":"all_ages","olang":"ja","covers":[
			{"url":"https://cdn.example/ab/cd/portrait.webp","cover_kind":"main","width":600,"height":800,"thumbhash":"P"}
		]}`
		f := fullOf(t, body)
		if f.EffectiveBannerURL != "https://cdn.example/ab/cd/portrait.webp" {
			t.Errorf("hero = %q, want the portrait fallback rather than an empty hero", f.EffectiveBannerURL)
		}
	})

	t.Run("dims unknown falls back to the pin", func(t *testing.T) {
		body := `{"object":"work","id":"4242","display_name":"Kun","content_rating":"all_ages","olang":"ja","covers":[
			{"url":"https://cdn.example/ab/cd/pinned.webp","cover_kind":"main"},
			{"url":"https://cdn.example/ef/gh/other.webp","cover_kind":"dig"}
		]}`
		f := fullOf(t, body)
		if f.EffectiveBannerURL != "https://cdn.example/ab/cd/pinned.webp" {
			t.Errorf("hero = %q, want the pinned cover when no dims are known", f.EffectiveBannerURL)
		}
	})
}

func TestCatalogDetail_HeroReadsTheCatalogCoverSlots(t *testing.T) {
	t.Run("banner slot wins", func(t *testing.T) {
		body := `{"object":"work","id":"4242","display_name":"Kun","content_rating":"all_ages","olang":"ja","covers":[
			{"url":"https://cdn.example/ab/cd/portrait.webp","cover_kind":"main","width":600,"height":800,"thumbhash":"P"},
			{"url":"https://cdn.example/ef/gh/banner.webp","cover_kind":"dig","width":1280,"height":720,"thumbhash":"B"}
		],
		"cover":{"url":"https://cdn.example/ab/cd/portrait.webp","width":600,"height":800,"thumbhash":"P","sexual":"safe","violence":null},
		"banner":{"url":"https://cdn.example/ef/gh/banner.webp","width":1280,"height":720,"thumbhash":"B","sexual":"safe","violence":null}}`
		f := fullOf(t, body)
		if f.EffectiveBannerURL != "https://cdn.example/ef/gh/banner.webp" ||
			f.EffectiveBannerWidth != 1280 || f.EffectiveBannerThumbhash != "B" {
			t.Errorf("hero = %q %dx%d %q, want the banner slot",
				f.EffectiveBannerURL, f.EffectiveBannerWidth, f.EffectiveBannerHeight, f.EffectiveBannerThumbhash)
		}
	})

	t.Run("null banner falls back to the portrait slot, not to the wide disc face", func(t *testing.T) {
		body := `{"object":"work","id":"4242","display_name":"Kun","content_rating":"all_ages","olang":"ja","covers":[
			{"url":"https://cdn.example/11/22/disc.webp","cover_kind":"pkgmed","width":1084,"height":1080,"thumbhash":"D"},
			{"url":"https://cdn.example/33/44/dig.webp","cover_kind":"dig","width":720,"height":1080,"thumbhash":"G"}
		],
		"cover":{"url":"https://cdn.example/33/44/dig.webp","width":720,"height":1080,"thumbhash":"G","sexual":"safe","violence":null},
		"banner":null}`
		f := fullOf(t, body)
		if f.EffectiveBannerURL != "https://cdn.example/33/44/dig.webp" {
			t.Errorf("hero = %q, want the portrait slot — the disc face is not key art", f.EffectiveBannerURL)
		}
		if f.EffectiveBannerWidth != 720 || f.EffectiveBannerHeight != 1080 {
			t.Errorf("dims = %dx%d, want 720x1080", f.EffectiveBannerWidth, f.EffectiveBannerHeight)
		}
	})

	t.Run("both slots null still shows the pin", func(t *testing.T) {
		body := `{"object":"work","id":"4242","display_name":"Kun","content_rating":"all_ages","olang":"ja","covers":[
			{"url":"https://cdn.example/11/22/disc.webp","cover_kind":"pkgmed","width":1084,"height":1080,"thumbhash":"D"}
		],"cover":null,"banner":null}`
		f := fullOf(t, body)
		if f.EffectiveBannerURL != "https://cdn.example/11/22/disc.webp" {
			t.Errorf("hero = %q, want covers[0] rather than a blank frame", f.EffectiveBannerURL)
		}
	})

	t.Run("no covers at all leaves the hero empty", func(t *testing.T) {
		body := `{"object":"work","id":"4242","display_name":"Kun","content_rating":"all_ages","olang":"ja","covers":[],"cover":null,"banner":null}`
		f := fullOf(t, body)
		if f.EffectiveBannerURL != "" {
			t.Errorf("hero = %q, want empty", f.EffectiveBannerURL)
		}
	})
}

func TestCatalogDetail_LabelsDedupPerLabelID(t *testing.T) {
	body := `{"object":"work","id":"4242","display_name":"Kun","content_rating":"all_ages","olang":"ja","companies":[
		{"object":"company","id":"11","display_name":"戯画","company_kind":"game_brand","attribution_role":"developer","lang":"ja","work_count":42,
		 "localized":{"zh-Hans":{"value":"戏画","is_machine":false}}},
		{"object":"company","id":"11","display_name":"戯画","company_kind":"game_brand","attribution_role":"publisher","lang":"ja","work_count":42},
		{"object":"company","id":"11","display_name":"戯画","company_kind":"game_brand","attribution_role":"brand","lang":"ja","work_count":42},
		{"object":"company","id":"22","display_name":"Circle","company_kind":"doujin_circle","attribution_role":"circle","lang":"en","work_count":3,
		 "localized":{}}
	]}`
	f := fullOf(t, body)

	if f.Official[0].Official.Name != "戏画" {
		t.Errorf("制作方 chip = %q, want the localized rendering 戏画", f.Official[0].Official.Name)
	}
	if f.Official[1].Official.Name != "Circle" {
		t.Errorf("a label with no Chinese name = %q, want its own name, never a blank",
			f.Official[1].Official.Name)
	}

	if len(f.Official) != 2 {
		t.Fatalf("projected %d 制作方 rows, want 2 — one per LABEL, not one per attribution edge: %+v",
			len(f.Official), f.Official)
	}
	if f.Official[0].Official.ID != 11 || f.Official[1].Official.ID != 22 {
		t.Errorf("row order = [%d %d], want [11 22] (first occurrence wins)",
			f.Official[0].Official.ID, f.Official[1].Official.ID)
	}
	if got := f.Official[0].Official.Category; got != "game_brand" {
		t.Errorf("category = %q, want game_brand (label_kind, not the edge kind)", got)
	}
	if got := f.Official[1].Official.Category; got != "doujin_circle" {
		t.Errorf("category = %q, want doujin_circle", got)
	}
	if got := f.Official[0].Official.Lang; got != "ja" {
		t.Errorf("lang = %q, want ja (an empty lang renders an empty span)", got)
	}
	if got := f.Official[0].Official.GalgameCount; got != 42 {
		t.Errorf("galgame_count = %d, want 42", got)
	}
}

func TestCatalogDetail_WorkCountReachesAllThreeChipFamilies(t *testing.T) {
	body := `{"object":"work","id":"4242","display_name":"Kun","content_rating":"all_ages","olang":"ja",
		"labels":[{"id":11,"display_name":"戯画","label_kind":"game_brand","kind":"developer","lang":"ja","work_count":42}],
		"engines":[{"id":31,"name":"KiriKiri","work_count":1337}],
		"tags":[{"name":"純愛","canonical_id":51,"kind":"content","spoiler":0,"sexual":false,"work_count":9001}]}`
	f := fullOf(t, body)

	if len(f.Official) != 1 || f.Official[0].Official.GalgameCount != 42 {
		t.Errorf("label count not projected: %+v", f.Official)
	}
	if len(f.Engine) != 1 || f.Engine[0].Engine.GalgameCount != 1337 {
		t.Errorf("engine count not projected: %+v", f.Engine)
	}
	if len(f.Tag) != 1 || f.Tag[0].Tag.GalgameCount != 9001 {
		t.Errorf("tag count not projected: %+v", f.Tag)
	}
}

func TestCatalogDetail_TagsArriveAtTheFullSpoilerCeiling(t *testing.T) {
	const workID int64 = 4242
	body := `{"object":"work","id":"4242","display_name":"Kun","content_rating":"all_ages","olang":"ja",
		"tags":[{"name":"純愛","canonical_id":51,"kind":"content","spoiler":0,"sexual":false},
		        {"name":"ヒロイン死亡","canonical_id":52,"kind":"content","spoiler":2,"sexual":false}]}`
	srv, seen := detailStub(t, workID, body)
	c := New(srv.URL, "nm_test_key", "")

	d, found, _, appErr := c.CatalogWorkDetail(context.Background(), int(workID))
	if appErr != nil || !found {
		t.Fatalf("CatalogWorkDetail = (%v, %v)", appErr, found)
	}
	if got := seen.Get("spoiler"); got != "major" {
		t.Errorf("spoiler = %q, want the full ceiling major — the tag panel filters client-side", got)
	}
	if seen.Has("spoilers") {
		t.Error("v1's numeric spoilers= leaked; v2 drops it silently and answers the none ceiling")
	}

	f := CatalogDetailToFull(context.Background(), d, int(workID))
	if len(f.Tag) != 2 || f.Tag[1].SpoilerLevel != 2 {
		t.Errorf("Tag = %+v, want the spoiler row carried with its level", f.Tag)
	}
}

func TestCatalogDetail_MissingWorkCountKeyIsZero(t *testing.T) {
	body := `{"object":"work","id":"4242","display_name":"Kun","content_rating":"all_ages","olang":"ja",
		"labels":[{"id":11,"display_name":"戯画","label_kind":"game_brand","kind":"developer","lang":"ja"}],
		"engines":[{"id":31,"name":"KiriKiri"}],
		"tags":[{"name":"純愛","canonical_id":51,"kind":"content","spoiler":0,"sexual":false},
		        {"name":"unmapped","canonical_id":0,"kind":"content","spoiler":0,"sexual":false}]}`
	f := fullOf(t, body)

	if len(f.Official) != 1 || f.Official[0].Official.GalgameCount != 0 {
		t.Errorf("missing label work_count must decode to 0, got %+v", f.Official)
	}
	if len(f.Engine) != 1 || f.Engine[0].Engine.GalgameCount != 0 {
		t.Errorf("missing engine work_count must decode to 0, got %+v", f.Engine)
	}
	if len(f.Tag) != 1 {
		t.Fatalf("projected %d tag chips, want 1 (the unmapped row has no id to link to): %+v", len(f.Tag), f.Tag)
	}
	if f.Tag[0].Tag.GalgameCount != 0 {
		t.Errorf("missing tag work_count must decode to 0, got %+v", f.Tag[0])
	}
	if f.Official[0].Official.Name != "戯画" || f.Engine[0].Engine.Name != "KiriKiri" {
		t.Errorf("rows dropped along with their counts: %+v %+v", f.Official, f.Engine)
	}
}

func TestCatalogDetail_RosterKeepsBothArtsApart(t *testing.T) {
	body := `{"object":"work","id":"4242","display_name":"Kun","content_rating":"all_ages","olang":"ja","characters":[
		{"id":11,"display_name":"藤田 佳奈","latin":"Fujita Kana","kind":"main","spoiler":0,
		 "localized":{"zh-Hans":{"value":"藤田佳奈","kind":"translation","machine":true}},
		 "image":"https://cdn.example/ab/cd/bust.webp","figure":"https://cdn.example/ef/gh/figure.webp",
		 "voices":[{"id":7,"display_name":"五十嵐裕美",
		            "localized":{"zh-Hans":{"value":"五十岚裕美","kind":"translation"}}},
		           {"id":9,"display_name":"別名義","localized":{}}]},
		{"id":12,"display_name":"名前だけ","kind":"unknown","spoiler":2,"localized":{},"voices":[]}
	]}`
	f := fullOf(t, body)

	if len(f.Characters) != 2 {
		t.Fatalf("projected %d roster rows, want 2 (a pictureless character is still cast): %+v",
			len(f.Characters), f.Characters)
	}

	lead := f.Characters[0]
	if lead.Name != "藤田佳奈" || lead.NameOriginal != "藤田 佳奈" {
		t.Errorf("roster name/original = %q/%q, want the localized rendering over the record name — "+
			"the roster is where a reader actually meets a character, and before wave 209 it "+
			"carried no localized block at all", lead.Name, lead.NameOriginal)
	}
	if lead.Voices[0].Name != "五十岚裕美" {
		t.Errorf("voice name = %q, want the localized rendering", lead.Voices[0].Name)
	}
	if lead.Image != "https://cdn.example/ab/cd/bust.webp" {
		t.Errorf("bust = %q, want the image URL verbatim", lead.Image)
	}
	if lead.Figure != "https://cdn.example/ef/gh/figure.webp" {
		t.Errorf("figure = %q, want the figure URL verbatim — it is NOT a variant of the bust", lead.Figure)
	}
	if lead.Kind != "main" || lead.Latin != "Fujita Kana" {
		t.Errorf("billing/latin lost: %+v", lead)
	}
	if len(lead.Voices) != 2 || lead.Voices[0].ID != 7 || lead.Voices[1].Name != "別名義" {
		t.Errorf("voices = %+v, want both credited names with their staff-page ids", lead.Voices)
	}

	bare := f.Characters[1]
	if bare.Kind != "unknown" || bare.Spoiler != 2 {
		t.Errorf("kind/spoiler = %q/%d, want unknown/2", bare.Kind, bare.Spoiler)
	}
	if bare.Image != "" || bare.Figure != "" {
		t.Errorf("a character with no art must not acquire any: %+v", bare)
	}
	if bare.Voices == nil {
		t.Error("voices serialized as null, want []")
	}
}

func TestCatalogDetail_RatingsCarryTheirPerSourceDetail(t *testing.T) {
	body := `{"object":"work","id":"4242","display_name":"Kun","content_rating":"all_ages","olang":"ja","ratings":[
		{"source":"vndb","score":8.1,"vote_count":900,"rank":12,
		 "distribution":[{"score":8,"count":500},{"score":9,"count":300}],
		 "stats":{"average":8.44}},
		{"source":"bangumi","score":6.5,"vote_count":5,
		 "distribution":[{"score":3,"count":1},{"score":10,"count":4}]},
		{"source":"erogamescape","score":76,"vote_count":28,
		 "distribution":[{"score":0,"count":1},{"score":70,"count":9},{"score":100,"count":2}],
		 "stats":{"average":74.5,"stdev":10,"min":60,"max":100}},
		{"source":"dlsite","score":4.6,"vote_count":10}
	]}`
	f := fullOf(t, body)

	by := map[string]dto.GalgameExternalRating{}
	for _, r := range f.ExternalRatings {
		by[r.Source] = r
	}

	vndb := by["vndb"]
	if len(vndb.Distribution) != 2 || vndb.Stats == nil || vndb.Stats.Average == nil {
		t.Errorf("vndb = %+v, want the histogram and stats it gained in 2026-08", vndb)
	}

	bgm := by["bangumi"]
	if len(bgm.Distribution) != 2 || bgm.Distribution[1].Score != 10 || bgm.Distribution[1].Count != 4 {
		t.Errorf("bangumi distribution = %+v, want the sparse buckets verbatim", bgm.Distribution)
	}

	// erogamescape's buckets are deciles keyed by their lower bound, 0 included.
	// Anything that renumbers them onto a 1-100 point axis drops the 0 bucket.
	eg := by["erogamescape"]
	if len(eg.Distribution) != 3 || eg.Distribution[0].Score != 0 || eg.Distribution[2].Score != 100 {
		t.Errorf("erogamescape distribution = %+v, want the decile keys verbatim", eg.Distribution)
	}
	if eg.Stats == nil || eg.Stats.Average == nil || *eg.Stats.Average != 74.5 {
		t.Fatalf("erogamescape stats = %+v, want average 74.5 alongside the 76 median", eg.Stats)
	}
	if eg.Score == *eg.Stats.Average {
		t.Error("median and mean collapsed into one number")
	}

	if got := by["dlsite"]; len(got.Distribution) != 0 || got.Stats != nil {
		t.Errorf("dlsite = %+v, want a source that sent neither to stay empty", got)
	}
}

func TestCatalogDetail_ImagesCarryTheirSource(t *testing.T) {
	body := `{"object":"work","id":"4242","display_name":"Kun","content_rating":"r18","olang":"ja","covers":[
		{"url":"https://cdn.example/ab/cd/a.webp","cover_kind":"main","source":"vndb","sexual":1},
		{"url":"https://cdn.example/ef/gh/b.webp","cover_kind":"main","source":"upscale"}
	],"screenshots":[
		{"url":"https://cdn.example/11/22/s1.webp","source":"vndb","sexual":0},
		{"url":"https://cdn.example/33/44/s2.webp","source":"getchu","sexual":2}
	]}`
	f := fullOf(t, body)

	if len(f.Covers) != 2 || f.Covers[0].Source != "vndb" || f.Covers[1].Source != "upscale" {
		t.Errorf("cover sources = %+v, want vndb then upscale", f.Covers)
	}
	if len(f.Screenshots) != 2 || f.Screenshots[0].Source != "vndb" || f.Screenshots[1].Source != "getchu" {
		t.Errorf("screenshot sources = %+v, want vndb then getchu", f.Screenshots)
	}
}

func TestCatalogDetail_PlaytimesCarryTheirSource(t *testing.T) {
	body := `{"object":"work","id":"4242","display_name":"Kun","content_rating":"all_ages","olang":"ja","playtimes":[
		{"source":"vndb","minutes":384,"vote_count":39},
		{"source":"erogamescape","minutes":240,"vote_count":0}
	]}`
	f := fullOf(t, body)

	if len(f.Playtimes) != 2 {
		t.Fatalf("len = %d, want 2", len(f.Playtimes))
	}
	vndb := f.Playtimes[0]
	if vndb.Source != "vndb" || vndb.Minutes != 384 || vndb.VoteCount != 39 {
		t.Errorf("vndb playtime = %+v, want source vndb 384 minutes 39 votes", vndb)
	}
	egs := f.Playtimes[1]
	if egs.Source != "erogamescape" || egs.Minutes != 240 || egs.VoteCount != 0 {
		t.Errorf("erogamescape playtime = %+v, want source erogamescape 240 minutes 0 votes", egs)
	}
}
