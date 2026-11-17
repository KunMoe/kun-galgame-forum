package agentmcp

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

type apiLog struct {
	mu   sync.Mutex
	seen []*url.URL
}

func (l *apiLog) last(path string) url.Values {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i := len(l.seen) - 1; i >= 0; i-- {
		if l.seen[i].Path == path {
			return l.seen[i].Query()
		}
	}
	return nil
}

const workJSON = `{"object":"work","id":"15","display_name":"千恋＊万花","localized":{"zh-Hans":{"value":"千恋万花"}},
"cover":{"url":"https://img.test/a.webp","sexual":"explicit"},"cover_sfw":{"url":"https://img.test/safe.webp","sexual":"safe"},
"is_nsfw":false,"release_date":"2016-07-29","maker":{"id":"5147","display_name":"ゆずソフト","localized":{"zh-Hans":{"value":"柚子软件"}}},
"view_count":10,"rating_score":7.5,"rating_count":49,"resource_platforms":["win","and"],"resource_languages":["zh-cn"]}`

var forumPages = map[string]string{
	"/api/v1/works":              `{"items":[` + workJSON + `],"total":1}`,
	"/api/v1/search/works":       `{"items":[` + workJSON + `],"total":1}`,
	"/api/v1/works/15":           strings.TrimSuffix(workJSON, "}") + `,"engines":[{"id":"11","display_name":"KiriKiri","catalog_work_count":4729}],"tags":[{"id":"1","display_name":"拔作","is_sexual":true},{"id":"2","display_name":"废萌","is_sexual":false}],"intros":[{"locale":"zh-Hans","value":"穗织的故事"}],"is_resource_hosted_elsewhere":false}`,
	"/api/v1/works/15/resources": `{"items":[{"id":"60883","resource_type":"game","resource_languages":["zh-cn"],"resource_platforms":["win","and"],"resource_runtimes":["native-win","kirikiroid2"],"title":"PC+安卓","size":"13 GB","state":"expired","download_count":4,"updated_at":"2026-09-24T21:17:13Z"}],"total":1}`,
	"/api/v1/ratings":            `{"items":[{"id":"9","author":{"id":"1","name":"luv"},"recommend":"yes","overall":8,"play_status":"done_all","short_summary_text":"真结局有剧透","has_short_summary_spoiler":true}],"total":1}`,
	"/api/v1/engines/11":         `{"id":"11","display_name":"KiriKiri","aliases":["krkr"],"catalog_work_count":4729}`,
	"/api/v1/engines/11/works":   `{"items":[` + workJSON + `],"total":1}`,
	"/api/v1/toolsets":           `{"items":[{"id":"131","title":"Locale Emulator","toolset_type":"emulator","interface_language":"zh-cn","platform":"windows"}],"total":1}`,
	"/api/v1/search/users":       `{"items":[{"id":"1","name":"kun","bio":"KUN","roles":["admin"],"topic_count":3,"reply_count":3}],"total":1}`,
	"/api/v1/galgame-resources":  `{"items":[],"total":0}`,
	"/api/v1/engines":            `{"items":[],"total":0}`,
}

func newAPIFixture(t *testing.T) (*fixture, *apiLog) {
	t.Helper()
	log := &apiLog{}
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.mu.Lock()
		log.seen = append(log.seen, r.URL)
		log.mu.Unlock()
		body, ok := forumPages[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(api.Close)
	h := Handler(Deps{
		Verifier: fakeVerifier{"alice": {ID: userAlice, Roles: []string{"user"}}, "bob": {ID: userBob, Roles: []string{"user"}}},
		Profiles: fakeProfiles{
			"alice": {AdultConfirmed: true, NSFWDisplay: "hide"},
			"bob":   {AdultConfirmed: true, NSFWDisplay: "show"},
		},
		API: NewForumAPI(api.URL + "/api/v1"),
	})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &fixture{url: srv.URL + Path}, log
}

func TestForumToolsRefuseAnonymousCallers(t *testing.T) {
	f, log := newAPIFixture(t)
	for _, tool := range []string{"search_galgames", "search_toolsets"} {
		text, _, isErr := f.call(t, "", tool, map[string]any{})
		if !isErr || !strings.Contains(text, "重新登录") {
			t.Errorf("%s: %v %q", tool, isErr, text)
		}
	}
	if len(log.seen) != 0 {
		t.Errorf("reached the forum API: %v", log.seen)
	}
}

func TestSearchGalgamesFollowsTheStanceAndTheMobileFilter(t *testing.T) {
	f, log := newAPIFixture(t)
	text, cards, isErr := f.call(t, "alice", "search_galgames", map[string]any{"mobile": true, "language": "zh-cn", "sort": "rating"})
	if isErr {
		t.Fatal(text)
	}
	q := log.last("/api/v1/works")
	if q.Get("include_nsfw") != "false" || q.Get("resource_platforms") != "and,ios" || q.Get("resource_runtimes") != "emulator" ||
		q.Get("resource_languages") != "zh-cn" || q.Get("sort") != "rating_desc" {
		t.Errorf("query: %v", q)
	}
	if len(cards) != 1 || cards[0].Kind != kindGalgame || cards[0].SameAs != "work:15" || cards[0].Title != "千恋万花" {
		t.Fatalf("cards: %+v", cards)
	}
	if cards[0].Cover != "https://img.test/safe.webp" || cards[0].ContentLimit != "sfw" {
		t.Errorf("an adult cover must give way to the safe one for a hiding reader: %+v", cards[0])
	}
	if !strings.Contains(text, "手机可玩") || !strings.Contains(text, "get_galgame") {
		t.Errorf("text: %s", text)
	}

	_, cards, _ = f.call(t, "bob", "search_galgames", map[string]any{"q": "千恋"})
	if log.last("/api/v1/search/works").Get("include_nsfw") != "true" || cards[0].ContentLimit != "nsfw" {
		t.Errorf("a showing reader sees the adult cover: %+v", cards)
	}
}

func TestSearchGalgamesSaysTheWorkIsHereWhenOnlyTheFilterMisses(t *testing.T) {
	f, _ := newAPIFixture(t)
	text, cards, isErr := f.call(t, "bob", "search_galgames", map[string]any{"q": "千恋", "platform": "ios"})
	if isErr || len(cards) != 1 || !strings.Contains(text, "本站有这些作品，但没有符合条件的资源") {
		t.Errorf("%v %+v %s", isErr, cards, text)
	}
}

func TestGetGalgameListsResourcesEnginesAndRatings(t *testing.T) {
	f, _ := newAPIFixture(t)
	text, cards, isErr := f.call(t, "alice", "get_galgame", map[string]any{"id": 15})
	if isErr {
		t.Fatal(text)
	}
	for _, want := range []string{"KiriKiri（引擎 #11）", "资源 #60883", "Kirikiroid2", "链接已失效", "穗织的故事", "短评含剧透", "废萌"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
	if strings.Contains(text, "拔作") {
		t.Errorf("a sexual tag reached a hiding reader:\n%s", text)
	}
	if len(cards) != 1 || cards[0].Summary == "" {
		t.Errorf("cards: %+v", cards)
	}

	text, _, isErr = f.call(t, "alice", "get_galgame", map[string]any{"id": 16})
	if !isErr || !strings.Contains(text, "没有 id 为 16 的作品") {
		t.Errorf("missing work: %v %s", isErr, text)
	}
}

func TestGetEngineNamesItsFiltersInTheSingular(t *testing.T) {
	f, log := newAPIFixture(t)
	_, cards, isErr := f.call(t, "bob", "get_engine", map[string]any{"id": 11, "mobile": true, "language": "zh-cn"})
	if isErr {
		t.Fatal(cards)
	}
	q := log.last("/api/v1/engines/11/works")
	if q.Get("resource_platform") != "and" || q.Get("resource_runtime") != "emulator" || q.Get("resource_language") != "zh-cn" ||
		q.Has("resource_platforms") || q.Has("resource_languages") {
		t.Errorf("query: %v", q)
	}
	if len(cards) != 2 || cards[0].Kind != kindEngine || cards[0].URL != "/galgame/engine/11" || cards[1].Kind != kindGalgame {
		t.Errorf("cards: %+v", cards)
	}
}

func TestSmallerListsCarryTheirOwnCards(t *testing.T) {
	f, _ := newAPIFixture(t)
	_, cards, _ := f.call(t, "bob", "search_toolsets", map[string]any{"q": "转区"})
	if len(cards) != 1 || cards[0].Kind != kindToolset || cards[0].URL != "/toolset/131" {
		t.Errorf("toolsets: %+v", cards)
	}
	_, cards, _ = f.call(t, "bob", "search_users", map[string]any{"q": "kun"})
	if len(cards) != 1 || cards[0].Kind != kindUser || cards[0].URL != "/user/1" {
		t.Errorf("users: %+v", cards)
	}
	_, cards, _ = f.call(t, "bob", "search_ratings", map[string]any{"work_id": 15, "sort": "highest"})
	if len(cards) != 1 || cards[0].Kind != kindRating || cards[0].Summary != "（短评含剧透，已省略）" {
		t.Errorf("ratings: %+v", cards)
	}
	text, _, isErr := f.call(t, "bob", "search_engines", map[string]any{"q": "吉里吉里"})
	if isErr || !strings.Contains(text, "KiriKiri") {
		t.Errorf("an engine miss should point at the English name: %s", text)
	}
}
