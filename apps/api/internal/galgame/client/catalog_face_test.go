package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

type catalogRecorder struct {
	mu    sync.Mutex
	paths []string
	query []url.Values
	body  []string
}

func (r *catalogRecorder) record(req *http.Request) {
	body := ""
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		body = string(b)
	}
	r.mu.Lock()
	r.paths = append(r.paths, req.URL.Path)
	r.query = append(r.query, req.URL.Query())
	r.body = append(r.body, body)
	r.mu.Unlock()
}

func (r *catalogRecorder) pathAt(i int) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if i >= len(r.paths) {
		return ""
	}
	return r.paths[i]
}

func (r *catalogRecorder) queryAt(i int) url.Values {
	r.mu.Lock()
	defer r.mu.Unlock()
	if i >= len(r.query) {
		return url.Values{}
	}
	return r.query[i]
}

func (r *catalogRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.paths)
}

func catalogStub(t *testing.T, rec *catalogRecorder, works map[int64]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		rec.record(req)
		w.Header().Set("Content-Type", "application/json")

		switch {
		case req.URL.Path == "/v2/catalog/works":
			var rows []string
			for _, raw := range strings.Split(req.URL.Query().Get("ids"), ",") {
				id := atoi64(raw)
				if frag, ok := works[id]; ok {
					rows = append(rows, frag)
				}
			}
			_, _ = w.Write([]byte(`{"items":[` + strings.Join(rows, ",") + `],"next_cursor":null}`))

		default:
			_, _ = w.Write([]byte(`{"items":[]}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func itoa(v int64) string {
	out := ""
	if v == 0 {
		return "0"
	}
	for v > 0 {
		out = string(rune('0'+v%10)) + out
		v /= 10
	}
	return out
}

func atoi64(s string) int64 {
	var out int64
	for _, r := range strings.TrimSpace(s) {
		if r < '0' || r > '9' {
			return 0
		}
		out = out*10 + int64(r-'0')
	}
	return out
}

func liveRow(id int64, name string) string {
	return `{"id":` + itoa(id) + `,"medium":"galgame","display_name":"` + name +
		`","content_rating":"all_ages","content_limit":"sfw","olang":"ja","release_date":"2024-06-14",` +
		`"claim":{"site":"kungal","site_work_id":` + itoa(id) + `,"state":"live"},` +
		`"updated":"2026-01-01T00:00:00Z","latin":"` + name + `Latin",` +
		`"localized":{"zh-Hans":{"value":"` + name + `CN","kind":"official","machine":true}},` +
		`"covers":{"portrait":{"url":"https://cdn.example/ab/cd/abcdef.webp","width":600,"height":800,"thumbhash":"TH"},"banner":null},` +
		`"refs":[{"source":"dlsite","external_id":"RJ01"},{"source":"vndb","external_id":"v19658"}]}`
}

// Wave 212 retired the four product-locale slots for the primitive every other
// catalog entity has carried since wave 209. The slots could not hold a Korean,
// Russian or untagged title at all, and 41,386 production rows have no lang.
func TestCatalogWorkListItem_NameComesFromTheLocalizedPrimitive(t *testing.T) {
	var it CatalogWorkListItem
	if err := json.Unmarshal([]byte(liveRow(4242, "Kun")), &it); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	name, original := it.Names(context.Background())
	if name != "KunCN" {
		t.Errorf("name = %q, want the Chinese title KunCN", name)
	}
	if original != "Kun" {
		t.Errorf("original = %q, want the work's own title Kun on the second line", original)
	}

	var bare CatalogWorkListItem
	if err := json.Unmarshal([]byte(`{"display_name":"Kun","latin":"KunLatin","localized":{}}`), &bare); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if name, original := bare.Names(context.Background()); name != "Kun" || original != "" {
		t.Errorf("no Chinese on file = (%q, %q), want the work's own title once and no "+
			"second line repeating it", name, original)
	}
}

func TestGetBatch_KeysByWorkID(t *testing.T) {
	rec := &catalogRecorder{}
	srv := catalogStub(t, rec, map[int64]string{4242: liveRow(4242, "Kun")})
	c := New(srv.URL, "nm_test_key", "")

	got, err := c.GetBatch(context.Background(), []int{4242})
	if err != nil {
		t.Fatalf("GetBatch: %v", err)
	}
	if rec.queryAt(0).Get("ids") != "4242" {
		t.Errorf("works ids = %q, want 4242", rec.queryAt(0).Get("ids"))
	}
	b, ok := got[4242]
	if !ok {
		t.Fatalf("result not keyed by work id 4242: %#v", got)
	}
	if b.ID != 4242 {
		t.Errorf("brief.ID = %d, want 4242", b.ID)
	}
	if b.Name != "KunCN" || b.NameOriginal != "Kun" {
		t.Errorf("names not projected: %+v", b)
	}
	if b.EffectiveBannerURL != "https://cdn.example/ab/cd/abcdef.webp" || b.EffectiveBannerThumbhash != "TH" {
		t.Errorf("cover slot not projected: %+v", b)
	}
	if b.Refs["dlsite"] != "RJ01" {
		t.Errorf("refs not projected (the DLsite purchase link reads this): %+v", b.Refs)
	}
	if b.VndbID != "v19658" {
		t.Errorf("vndb_id = %q, want it derived from refs", b.VndbID)
	}
	if b.ContentLimit != "sfw" || b.AgeLimit != "all" {
		t.Errorf("content rating projection wrong: %+v", b)
	}
	if b.OriginalLanguage != "ja-jp" {
		t.Errorf("olang = %q, want the ja-jp product key", b.OriginalLanguage)
	}
	if b.Status != GalgameStatusPublished {
		t.Errorf("status = %d, want published for a live claim", b.Status)
	}
}

func TestCatalogBatch_HiddenClaimNeverRenders(t *testing.T) {
	hidden := strings.Replace(liveRow(4242, "Banned"), `"state":"live"`, `"state":"hidden"`, 1)
	rec := &catalogRecorder{}
	srv := catalogStub(t, rec, map[int64]string{4242: hidden})
	c := New(srv.URL, "nm_test_key", "")

	got, err := c.GetBatch(context.Background(), []int{4242})
	if err != nil {
		t.Fatalf("GetBatch: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("a withdrawn (state=hidden) claim reached the caller: %#v — this republishes banned entries", got)
	}
}

func TestCatalogBatch_UnknownWorkIDIsAbsentNotAnError(t *testing.T) {
	rec := &catalogRecorder{}
	srv := catalogStub(t, rec, map[int64]string{})
	c := New(srv.URL, "nm_test_key", "")

	got, err := c.GetBatch(context.Background(), []int{999})
	if err != nil {
		t.Fatalf("an unknown work id must not be an error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %#v, want empty", got)
	}
	if rec.count() != 1 {
		t.Errorf("made %d calls, want 1", rec.count())
	}
}

func TestCatalogBatch_GatesAreParametersNotPostFilters(t *testing.T) {
	rec := &catalogRecorder{}
	srv := catalogStub(t, rec, map[int64]string{4242: liveRow(4242, "Kun")})
	c := New(srv.URL, "nm_test_key", "")

	if _, err := c.GetBatchPublic(context.Background(), []int{4242}, true); err != nil {
		t.Fatalf("GetBatchPublic sfw: %v", err)
	}
	if v := rec.queryAt(0).Get("nsfw"); v != "true" {
		t.Errorf("sfw caller sent nsfw=%q, want true — closing the age gate drops 94.5%% of the registry", v)
	}
	if v := rec.queryAt(0).Get("content_limit"); v != "sfw" {
		t.Errorf("sfw caller sent content_limit=%q, want sfw — the setting must reach the wire as the editorial gate", v)
	}

	before := rec.count()
	if _, err := c.GetBatchPublic(context.Background(), []int{4242}, false); err != nil {
		t.Fatalf("GetBatchPublic nsfw: %v", err)
	}
	if v := rec.queryAt(before).Get("nsfw"); v != "true" {
		t.Errorf("nsfw caller's works fetch sent nsfw=%q, want true", v)
	}
	if v := rec.queryAt(before).Get("content_limit"); v != "" {
		t.Errorf("nsfw caller sent content_limit=%q, want it absent (no editorial filter)", v)
	}
}

func TestCatalogDisplayLimit_ReadsTheEditorialAxis(t *testing.T) {
	r18SfwEntry := strings.Replace(liveRow(4242, "Kun"), `"content_rating":"all_ages"`, `"content_rating":"r18"`, 1)

	rec := &catalogRecorder{}
	srv := catalogStub(t, rec, map[int64]string{4242: r18SfwEntry})
	c := New(srv.URL, "nm_test_key", "")

	got, err := c.GetBatch(context.Background(), []int{4242})
	if err != nil {
		t.Fatalf("GetBatch: %v", err)
	}
	b, ok := got[4242]
	if !ok {
		t.Fatalf("row missing: %#v", got)
	}
	if b.ContentLimit != "sfw" {
		t.Errorf("content_limit = %q, want sfw — the editorial verdict wins over the age rating", b.ContentLimit)
	}
	if b.AgeLimit != "r18" {
		t.Errorf("age_limit = %q, want r18 — the two axes are independent", b.AgeLimit)
	}
}

func TestCatalogDisplayLimit_FailsClosed(t *testing.T) {
	sfwRow := liveRow(4242, "Kun")
	for name, body := range map[string]string{
		"verdict missing": strings.Replace(sfwRow, `"content_limit":"sfw",`, ``, 1),
		"verdict empty":   strings.Replace(sfwRow, `"content_limit":"sfw"`, `"content_limit":""`, 1),
		"verdict garbage": strings.Replace(sfwRow, `"content_limit":"sfw"`, `"content_limit":"ssfw"`, 1),
		"unclaimed, all cover art explicit": strings.Replace(
			strings.Replace(sfwRow, `"content_limit":"sfw"`, `"content_limit":"nsfw"`, 1),
			`"claim":{"site":"kungal","site_work_id":4242,"state":"live"},`, ``, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if body == sfwRow {
				t.Fatal("fixture did not change")
			}
			rec := &catalogRecorder{}
			srv := catalogStub(t, rec, map[int64]string{4242: body})
			c := New(srv.URL, "nm_test_key", "")

			got, err := c.GetBatch(context.Background(), []int{4242})
			if err != nil {
				t.Fatalf("GetBatch: %v", err)
			}
			if b := got[4242]; b.ContentLimit != "nsfw" || b.AgeLimit != "all" {
				t.Errorf("content_limit = %q on an all_ages row, want nsfw: only catalog's verdict can make a work sfw", b.ContentLimit)
			}
		})
	}
}

type faceRecorder struct {
	mu     sync.Mutex
	path   string
	apiKey string
	auth   string
}

func (r *faceRecorder) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.mu.Lock()
		r.path = req.URL.Path
		r.apiKey = strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
		r.auth = req.Header.Get("Authorization")
		r.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{}}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestCatalogFace_PathsAndCredentials(t *testing.T) {
	rec := &faceRecorder{}
	srv := rec.server(t)
	c := New(srv.URL, "nm_test_key", "")
	ctx := context.Background()

	t.Run("taxonomy list → /v2/catalog + key", func(t *testing.T) {
		if _, err := c.CatalogTaxonomyList(ctx, "tags", nil); err != nil {
			t.Fatalf("CatalogTaxonomyList: %v", err)
		}
		if rec.path != "/v2/catalog/tags" {
			t.Errorf("path = %q, want /v2/catalog/tags", rec.path)
		}
		if rec.apiKey != "nm_test_key" {
			t.Errorf("Authorization bearer = %q, want nm_test_key", rec.apiKey)
		}
	})

	t.Run("entity search → /v2/catalog/search", func(t *testing.T) {
		if _, _, err := c.CatalogEntitySearch(ctx, "labels", "kun", 1, 10); err != nil {
			t.Fatalf("CatalogEntitySearch: %v", err)
		}
		if rec.path != "/v2/catalog/search" {
			t.Errorf("path = %q, want /v2/catalog/search", rec.path)
		}
	})

	t.Run("works search → /v2/catalog/works", func(t *testing.T) {
		if _, err := c.CatalogWorksSearch(ctx, url.Values{"q": {"kun"}}); err != nil {
			t.Fatalf("CatalogWorksSearch: %v", err)
		}
		if rec.path != "/v2/catalog/works" {
			t.Errorf("path = %q, want /v2/catalog/works", rec.path)
		}
	})

	t.Run("calendar buckets → /v2/catalog/calendar*", func(t *testing.T) {
		for bucket, want := range map[string]string{
			"":         "/v2/catalog/calendar",
			"/pending": "/v2/catalog/calendar",
			"/tba":     "/v2/catalog/calendar",
		} {
			if _, err := c.CatalogCalendar(ctx, bucket, nil); err != nil {
				t.Fatalf("CatalogCalendar(%q): %v", bucket, err)
			}
			if rec.path != want {
				t.Errorf("bucket %q → path %q, want %q", bucket, rec.path, want)
			}
		}
	})

}

// An entity with no members must restrict the local list to nothing. ListIDs
// reads a nil RestrictIDs as "no restriction at all", so a nil here would list
// the whole forum on the page of a tag nobody uses.
func TestCatalogMemberWorkIDsIsNeverNil(t *testing.T) {
	rec := &catalogRecorder{}
	srv := catalogStub(t, rec, map[int64]string{})
	c := New(srv.URL, "nm_test_key", "")

	workIDs, err := c.CatalogMemberWorkIDs(context.Background(),
		url.Values{"tag_id": {"5"}}, true, 200)
	if err != nil {
		t.Fatalf("CatalogMemberWorkIDs: %v", err)
	}
	if workIDs == nil {
		t.Fatal("an empty membership walk returned nil, which restricts nothing")
	}
}

func TestCatalogMemberWorkIDsCarriesTheWalkSort(t *testing.T) {
	rec := &catalogRecorder{}
	srv := catalogStub(t, rec, map[int64]string{})
	c := New(srv.URL, "nm_test_key", "")

	if _, err := c.CatalogMemberWorkIDs(context.Background(),
		url.Values{"tag_id": {"5"}, "sort": {"released_desc"}}, true, 200); err != nil {
		t.Fatalf("CatalogMemberWorkIDs: %v", err)
	}
	if got := rec.queryAt(0).Get("sort"); got != "released_desc" {
		t.Errorf("sort = %q, want released_desc — the walk order is the page order", got)
	}
}

func TestCatalogMemberWorkIDs_DoesNotGateOnClaimState(t *testing.T) {
	// The forum's own vocabulary is v1's; only the wire is v2's, and the company
	// filter is the one that gets renamed on the way out.
	for family, filter := range map[string]struct{ in, wire string }{
		"tag":    {"tag_id", "tag_id"},
		"label":  {"label_id", "company_id"},
		"engine": {"engine_id", "engine_id"},
	} {
		t.Run(family, func(t *testing.T) {
			rec := &catalogRecorder{}
			srv := catalogStub(t, rec, map[int64]string{})
			c := New(srv.URL, "nm_test_key", "")

			if _, err := c.CatalogMemberWorkIDs(context.Background(),
				url.Values{filter.in: {"5"}}, true, 200); err != nil {
				t.Fatalf("CatalogMemberWorkIDs: %v", err)
			}
			if p := rec.pathAt(0); p != "/v2/catalog/works" {
				t.Fatalf("path = %q, want /v2/catalog/works", p)
			}
			q := rec.queryAt(0)
			if got := q.Get("claim_state"); got != "" {
				t.Errorf("claim_state = %q, want it absent — the %s page is catalog membership", got, family)
			}
			if got := q.Get("claimed"); got != "" {
				t.Errorf("claimed = %q, want it absent", got)
			}
			if got := q.Get(filter.wire); got != "5" {
				t.Errorf("%s = %q, want 5 — an unscoped walk lists the whole registry", filter.wire, got)
			}
			if filter.wire != filter.in && q.Has(filter.in) {
				t.Errorf("%s reached the wire; v2 ignores it silently and lists the whole registry", filter.in)
			}
			if got := q.Get("nsfw"); got != "true" {
				t.Errorf("nsfw = %q, want true — the age gate is never a population cut", got)
			}
			if got := q.Get("content_limit"); got != "sfw" {
				t.Errorf("content_limit = %q, want sfw for an SFW caller", got)
			}
		})
	}
}

// The portrait card reads its own slot, so it must not be filled from the
// banner here: the fallback belongs on the client, where a landscape banner is
// cropped rather than shown at the wrong ratio.
func TestCoverSlots_PortraitRidesSeparately(t *testing.T) {
	for name, tc := range map[string]struct {
		covers       string
		wantPortrait string
		wantW, wantH int
	}{
		"portrait present": {
			`{"portrait":{"url":"https://cdn.example/ab/cd/portrait.webp","width":600,"height":800,"thumbhash":"P"},` +
				`"banner":{"url":"https://cdn.example/ef/gh/banner.webp","width":1280,"height":720,"thumbhash":"B"}}`,
			"https://cdn.example/ab/cd/portrait.webp", 600, 800,
		},
		"banner only": {
			`{"portrait":null,"banner":{"url":"https://cdn.example/ef/gh/banner.webp","width":1280,"height":720,"thumbhash":"B"}}`,
			"", 0, 0,
		},
	} {
		t.Run(name, func(t *testing.T) {
			row := strings.Replace(liveRow(4242, "Kun"),
				`"covers":{"portrait":{"url":"https://cdn.example/ab/cd/abcdef.webp","width":600,"height":800,"thumbhash":"TH"},"banner":null}`,
				`"covers":`+tc.covers, 1)
			rec := &catalogRecorder{}
			srv := catalogStub(t, rec, map[int64]string{4242: row})
			c := New(srv.URL, "nm_test_key", "")

			got, err := c.GetBatch(context.Background(), []int{4242})
			if err != nil {
				t.Fatalf("GetBatch: %v", err)
			}
			b := got[4242]
			if b.EffectivePortraitURL != tc.wantPortrait {
				t.Errorf("effective portrait = %q, want %q", b.EffectivePortraitURL, tc.wantPortrait)
			}
			if b.EffectivePortraitWidth != tc.wantW || b.EffectivePortraitHeight != tc.wantH {
				t.Errorf("portrait dims = %dx%d, want %dx%d", b.EffectivePortraitWidth, b.EffectivePortraitHeight, tc.wantW, tc.wantH)
			}
		})
	}
}

func TestCoverSlots_BannerWinsPortraitFallsBack(t *testing.T) {
	const (
		portraitSlot = `"portrait":{"url":"https://cdn.example/ab/cd/portrait.webp","width":600,"height":800,"thumbhash":"P"}`
		bannerSlot   = `"banner":{"url":"https://cdn.example/ef/gh/banner.webp","width":1280,"height":720,"thumbhash":"B"}`
	)
	for name, tc := range map[string]struct {
		covers   string
		wantURL  string
		wantW    int
		wantH    int
		wantHash string
	}{
		"both slots filled → banner": {
			`{` + portraitSlot + `,` + bannerSlot + `}`,
			"https://cdn.example/ef/gh/banner.webp", 1280, 720, "B",
		},
		"portrait only → portrait": {
			`{` + portraitSlot + `,"banner":null}`,
			"https://cdn.example/ab/cd/portrait.webp", 600, 800, "P",
		},
	} {
		t.Run(name, func(t *testing.T) {
			row := strings.Replace(liveRow(4242, "Kun"),
				`"covers":{"portrait":{"url":"https://cdn.example/ab/cd/abcdef.webp","width":600,"height":800,"thumbhash":"TH"},"banner":null}`,
				`"covers":`+tc.covers, 1)
			rec := &catalogRecorder{}
			srv := catalogStub(t, rec, map[int64]string{4242: row})
			c := New(srv.URL, "nm_test_key", "")

			got, err := c.GetBatch(context.Background(), []int{4242})
			if err != nil {
				t.Fatalf("GetBatch: %v", err)
			}
			b := got[4242]
			if b.EffectiveBannerURL != tc.wantURL {
				t.Errorf("effective banner = %q, want %q", b.EffectiveBannerURL, tc.wantURL)
			}
			if b.EffectiveBannerWidth != tc.wantW || b.EffectiveBannerHeight != tc.wantH ||
				b.EffectiveBannerThumbhash != tc.wantHash {
				t.Errorf("dims/thumbhash = %dx%d %q, want %dx%d %q — they must ride with the chosen slot",
					b.EffectiveBannerWidth, b.EffectiveBannerHeight, b.EffectiveBannerThumbhash,
					tc.wantW, tc.wantH, tc.wantHash)
			}
		})
	}
}

func TestProductLocaleProjection(t *testing.T) {
	cases := map[string]string{
		"ja": "ja-jp", "ja-JP": "ja-jp",
		"zh": "zh-cn", "zh-Hans": "zh-cn",
		"zh-Hant": "zh-tw", "zh-TW": "zh-tw", "zh-HK": "zh-tw",
		"en": "en-us", "en-GB": "en-us",
		"ko": "ko", "": "",
	}
	for in, want := range cases {
		if got := productLocale(in); got != want {
			t.Errorf("productLocale(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCatalogLabelRollupMembers_AsksForTheHopAndKeepsTheAttribution(t *testing.T) {
	rec := &catalogRecorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		rec.record(req)
		w.Header().Set("Content-Type", "application/json")
		own := liveRow(4242, "Own")
		via := strings.TrimSuffix(liveRow(4243, "Imprinted"), "}") +
			`,"via_label":{"id":24,"display_name":"Key",` +
			`"localized":{"zh-Hans":{"value":"键社","kind":"translation"}}}}`
		_, _ = w.Write([]byte(`{"object":"list","items":[` +
			own + `,` + via + `],"next_cursor":null}`))
	}))
	t.Cleanup(srv.Close)
	c := New(srv.URL, "nm_test_key", "")

	members, err := c.CatalogLabelRollupMembers(context.Background(), "993", "", false, 5)
	if err != nil {
		t.Fatalf("CatalogLabelRollupMembers: %v", err)
	}

	q := rec.queryAt(0)
	if got := q.Get("company_rollup"); got != "true" {
		t.Errorf("company_rollup = %q, want true — without it a holding company's page is empty", got)
	}
	if got := q.Get("company_id"); got != "993" {
		t.Errorf("company_id = %q, want 993", got)
	}
	if got := q.Get("claim_state"); got != "" {
		t.Errorf("claim_state = %q, want it absent", got)
	}

	if len(members) != 2 {
		t.Fatalf("members = %d, want 2", len(members))
	}
	if members[0].WorkID != 4242 || members[0].Via != nil {
		t.Errorf("own work = %+v, want work 4242 with no via — a company's own game must not read as borrowed", members[0])
	}
	if members[1].WorkID != 4243 || members[1].Via == nil {
		t.Fatalf("rolled-up work = %+v, want work 4243 with a via", members[1])
	}
	if members[1].Via.ID != 24 || members[1].Via.Name(context.Background()) != "键社" {
		t.Errorf("via = %+v, want id 24 rendered as 键社", *members[1].Via)
	}
}

func TestCatalogWorkDetailCarriesTheVerdict(t *testing.T) {
	for _, limit := range []string{"sfw", "nsfw"} {
		var d CatalogWorkDetail
		raw := `{"id":4242,"display_name":"Kun","content_rating":"all_ages","content_limit":"` + limit + `"}`
		if err := json.Unmarshal([]byte(raw), &d); err != nil {
			t.Fatal(err)
		}
		item := d.ListItem()
		if got := CatalogItemToBrief(context.Background(), &item).ContentLimit; got != limit {
			t.Errorf("detail content_limit %q became %q on the list item", limit, got)
		}
	}
}
