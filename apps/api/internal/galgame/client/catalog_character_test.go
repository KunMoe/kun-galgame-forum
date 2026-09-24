package client

import (
	"cmp"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// v2 moved the character's works onto their own cursor-paged sub-face, so the
// stub answers two routes and records whether the sub-face was reached at all —
// that call, not an include= token, is what "works were asked for" now means.
func characterStub(t *testing.T, id int64, status int, body string) (*httptest.Server, *characterCalls) {
	t.Helper()
	calls := &characterCalls{}
	detail := "/v2/catalog/characters/" + itoa(id)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case detail:
			calls.detail = req.URL.Query()
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		case detail + "/appearances":
			calls.appearances = req.URL.Query()
			_, _ = w.Write([]byte(calls.appearanceBody))
		default:
			t.Errorf("unexpected upstream call: %s", req.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, calls
}

type characterCalls struct {
	detail         url.Values
	appearances    url.Values
	appearanceBody string
}

func TestCatalogCharacter_WorksAreIncludeGatedOnTheWire(t *testing.T) {
	srv, calls := characterStub(t, 5, http.StatusOK,
		`{"object":"character","id":"5","display_name":"朝倉","traits":[],"intros":[],"refs":[]}`)
	calls.appearanceBody = `{"object":"list","items":[],"next_cursor":null}`
	c := New(srv.URL, "nm_test_key", "")

	if _, _, _, appErr := c.CatalogCharacterDetail(context.Background(), 5, 50, 0, false); appErr != nil {
		t.Fatalf("CatalogCharacterDetail: %v", appErr)
	}
	if calls.appearances != nil {
		t.Error("the appearances sub-face was fetched with withWorks=false")
	}
	// The detail include is unconditional and total: v2 answers a bare id+name
	// row for anything it was not asked for, so dropping a token blanks a panel.
	if got := calls.detail.Get("include"); got != v2CatalogDetailInclude["characters"] {
		t.Errorf("include = %q, want the full character vocabulary", got)
	}
	if got := calls.detail.Get("spoiler"); got != "major" {
		t.Errorf("spoiler = %q, want the full ceiling major", got)
	}
	if calls.detail.Has("spoilers") {
		t.Error("v1's numeric spoilers= leaked; v2 drops it silently and hides every spoilered trait")
	}
	if got := calls.detail.Get("nsfw"); got != "true" {
		t.Errorf("nsfw = %q, want the population open", got)
	}

	if _, _, _, appErr := c.CatalogCharacterDetail(context.Background(), 5, 50, 0, true); appErr != nil {
		t.Fatalf("CatalogCharacterDetail(withWorks): %v", appErr)
	}
	if calls.appearances == nil {
		t.Fatal("the appearances sub-face was not fetched with withWorks=true")
	}
	// The sub-face gates its own population, so it has to inherit the parent's.
	if got := calls.appearances.Get("nsfw"); got != "true" {
		t.Errorf("appearances nsfw = %q, want the parent's open population", got)
	}
}

func TestCatalogCharacter_BothArtsSurviveInTheirOwnFields(t *testing.T) {
	// Captured from character 7 on the running catalog: the two arts are objects
	// rather than URL strings, traits carry localized/is_* rather than *_zh, and
	// the works block arrives from the appearances sub-face with a cursor.
	srv, calls := characterStub(t, 7, http.StatusOK, `{
		"object":"character","id":"7","display_name":"雪村杏","latin":"Yukimura Anzu",
		"image":{"url":"https://cdn.test/aa/bb/bust.webp","hash":"aabb","width":250,"height":300,"thumbhash":"B","sexual":"safe","violence":null},
		"figure":{"url":"https://cdn.test/cc/dd/figure.webp","hash":"ccdd","width":400,"height":900,"thumbhash":"F","sexual":"safe","violence":null},
		"traits":[{"object":"character_trait","id":"1","display_name":"Blonde","group":"Hair",
		           "localized":{"zh-Hans":{"value":"金发","is_machine":false}},
		           "group_localized":{"zh-Hans":{"value":"发型","is_machine":false}},
		           "spoiler":"none","is_sexual":false,"is_lie":false},
		          {"object":"character_trait","id":"2","display_name":"Dead","group":"Role",
		           "localized":{},"group_localized":{},
		           "spoiler":"major","is_sexual":false,"is_lie":true}],
		"intros":[{"lang":"zh-Hans","text":"主人公的青梅竹马","source":"bangumi","is_machine":true}],
		"refs":[{"source":"vndb","external_id":"c1234"}]}`)
	calls.appearanceBody = `{"object":"list","items":[
		{"object":"appearance","roster_role":"main","spoiler":"none",
		 "work":{"object":"work","id":"900","display_name":"テスト","medium":"game","content_rating":"r18"},
		 "voices":[{"object":"credit_name","id":"11","display_name":"茶木ひかる","lang":"ja"}]}],
		"next_cursor":"cur_NTA"}`
	c := New(srv.URL, "nm_test_key", "")

	ch, found, movedTo, appErr := c.CatalogCharacterDetail(context.Background(), 7, 50, 0, true)
	if appErr != nil || !found || movedTo != 0 {
		t.Fatalf("CatalogCharacterDetail = (%v, %v, %d)", appErr, found, movedTo)
	}
	if ch.Image != "https://cdn.test/aa/bb/bust.webp" {
		t.Errorf("Image = %q", ch.Image)
	}
	if ch.Figure != "https://cdn.test/cc/dd/figure.webp" {
		t.Errorf("Figure = %q", ch.Figure)
	}
	if len(ch.Traits) != 2 || ch.Traits[1].Spoiler != 2 || !ch.Traits[1].Lie {
		t.Errorf("Traits = %+v, want the spoiler+lie row intact", ch.Traits)
	}
	if got, want := ch.Traits[0].Localized["zh-Hans"].Value, "金发"; got != want {
		t.Errorf("zh trait name = %q, want %q", got, want)
	}
	if got, want := ch.Traits[0].GroupLocalized["zh-Hans"].Value, "发型"; got != want {
		t.Errorf("zh trait group = %q, want %q", got, want)
	}
	if got, want := cmp.Or(ch.Traits[1].DisplayName, ch.Traits[1].Name), "Dead"; got != want {
		t.Errorf("untranslated trait = %q, want the vocabulary name %q", got, want)
	}
	if len(ch.Intros) != 1 || !ch.Intros[0].Machine {
		t.Errorf("Intros = %+v, want the machine flag carried", ch.Intros)
	}
	if len(ch.Works) != 1 || len(ch.Works[0].Voices) != 1 || ch.Works[0].Voices[0].ID != 11 {
		t.Errorf("Works = %+v, want the per-work voice name", ch.Works)
	}
	if ch.NextOffset == nil || *ch.NextOffset != 50 {
		t.Errorf("NextOffset = %v, want 50", ch.NextOffset)
	}
}

func TestCatalogCharacter_MergedIDRedirectsRatherThanRenders(t *testing.T) {
	// Captured from company 15845 on the running catalog, which is merged into
	// 818: v2 reports a merge as a 404 whose code is ENTITY_MERGED, where v1 sent
	// 301 + Location. A reader that checks the status alone sees only "absent".
	srv, _ := characterStub(t, 3, http.StatusNotFound, `{
		"$schema":"https://catalog/Problem.json",
		"type":"https://developer.nextmoe.dev/problems/catalog/entity-merged",
		"title":"Entity merged","status":404,
		"detail":"character 3 was merged into character 91.",
		"code":"ENTITY_MERGED","object":"character","current_id":"91"}`)
	c := New(srv.URL, "nm_test_key", "")

	ch, found, movedTo, appErr := c.CatalogCharacterDetail(context.Background(), 3, 50, 0, true)
	if appErr != nil {
		t.Fatalf("CatalogCharacterDetail: %v", appErr)
	}
	if found || ch != nil {
		t.Errorf("found = %v, record = %+v, want neither under a dead id", found, ch)
	}
	if movedTo != 91 {
		t.Errorf("movedTo = %d, want 91", movedTo)
	}
}

func TestCatalogCharacter_UnknownIDIsAMiss(t *testing.T) {
	srv, _ := characterStub(t, 4, http.StatusNotFound, `{"code":404,"message":"not found","data":null}`)
	c := New(srv.URL, "nm_test_key", "")

	_, found, movedTo, appErr := c.CatalogCharacterDetail(context.Background(), 4, 50, 0, true)
	if appErr != nil {
		t.Fatalf("CatalogCharacterDetail: %v", appErr)
	}
	if found || movedTo != 0 {
		t.Errorf("found = %v, movedTo = %d, want a clean miss", found, movedTo)
	}
}
