package client

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func changesServer(t *testing.T, body string) (*GalgameClient, *url.Values) {
	t.Helper()
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return New(srv.URL, "nmk_test", ""), &got
}

func TestCatalogChangesKeepsOnlyWorkRows(t *testing.T) {
	// The feed's target_object vocabulary is eight families wide. A tag id and a
	// work id are both small integers, so a company row taken for a work is a
	// hydration of somebody else's id, not an error.
	c, _ := changesServer(t, `{"object":"list","items":[
		{"object":"change","target_object":"work","id":"11","updated_at":"2026-08-01T00:00:00Z"},
		{"object":"change","target_object":"company","id":"12","updated_at":"2026-08-01T00:00:01Z"},
		{"object":"change","target_object":"tag","id":"13","updated_at":"2026-08-01T00:00:02Z"},
		{"object":"change","target_object":"work","id":"14","gone":true,"updated_at":"2026-08-01T00:00:03Z"}
	],"next_cursor":"cur_next"}`)

	page, appErr := c.CatalogChanges(t.Context(), "", 100)
	if appErr != nil {
		t.Fatalf("CatalogChanges: %v", appErr.Message)
	}
	if len(page.Items) != 2 {
		t.Fatalf("kept %d rows, want only the two works: %+v", len(page.Items), page.Items)
	}
	if page.Items[0].ID != 11 || page.Items[0].Gone {
		t.Errorf("first row = %+v, want the live work 11", page.Items[0])
	}
	if page.Items[1].ID != 14 || !page.Items[1].Gone {
		t.Errorf("second row = %+v, want work 14 carrying gone", page.Items[1])
	}
	if page.NextCursor != "cur_next" {
		t.Errorf("next_cursor = %q", page.NextCursor)
	}
}

func TestCatalogChangesAsksWithinTheFeedsOwnVocabulary(t *testing.T) {
	c, got := changesServer(t, `{"object":"list","items":[]}`)

	if _, appErr := c.CatalogChanges(t.Context(), "cur_abc", 1000); appErr != nil {
		t.Fatalf("CatalogChanges: %v", appErr.Message)
	}
	// Over 100 is 400 LIMIT_TOO_LARGE, not a clamp: asking for more stops the
	// mirror dead rather than returning fewer rows.
	if v := got.Get("limit"); v != "100" {
		t.Errorf("limit = %q, want the feed ceiling 100", v)
	}
	if v := got.Get("cursor"); v != "cur_abc" {
		t.Errorf("cursor = %q", v)
	}
	if v := got.Get("nsfw"); v != "true" {
		t.Errorf("nsfw = %q, want true — the feed documents absence as hiding r18, "+
			"and r18 is 99%% of the population the forum mirrors", v)
	}

	if _, appErr := c.CatalogChanges(t.Context(), "", 100); appErr != nil {
		t.Fatalf("CatalogChanges: %v", appErr.Message)
	}
	if got.Has("cursor") {
		t.Error("an empty cursor must be omitted, not sent as cursor= — the feed requires a cur_ prefix")
	}
}

func TestMirrorByCatalogIDsKeysByTheClaimNeverTheCatalogID(t *testing.T) {
	// 10,289 of the forum's 11,562 local ids are also the catalog id of some
	// other work. Keying a hydrated row by its catalog id — which is what gid()
	// falls back to when there is no claim — therefore writes one game's display
	// verdict onto another game's row on nearly every page of this feed.
	var asked url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = r.URL.Query()
		_, _ = w.Write([]byte(`{"object":"list","items":[
			{"id":"923","content_rating":"r18","claim":{"site":"kungal","site_work_id":"8471","state":"live","content_limit":"sfw"}},
			{"id":"924","content_rating":"all_ages","claim":{"site":"kungal","site_work_id":"8472","state":"live","content_limit":"nsfw"}},
			{"id":"925","content_rating":"r18"},
			{"id":"926","content_rating":"r18","claim":{"site":"moyu","site_work_id":"77","state":"live","content_limit":"sfw"}},
			{"id":"927","content_rating":"r18","claim":{"site":"kungal","site_work_id":"8473","state":"hidden","content_limit":"sfw"}}
		]}`))
	}))
	defer srv.Close()

	got, appErr := New(srv.URL, "nmk_test", "").
		MirrorByCatalogIDs(t.Context(), []int64{923, 924, 925, 926, 927})
	if appErr != nil {
		t.Fatalf("MirrorByCatalogIDs: %v", appErr.Message)
	}

	want := map[int]string{8471: "sfw", 8472: "nsfw"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for gid, limit := range want {
		if got[gid].ContentLimit != limit {
			t.Errorf("gid %d = %q, want %q", gid, got[gid].ContentLimit, limit)
		}
	}
	for _, catalogID := range []int{923, 924, 925, 926, 927} {
		if _, ok := got[catalogID]; ok {
			t.Errorf("catalog id %d leaked in as a gid: %v", catalogID, got)
		}
	}

	// The verdict is the editorial axis, so the r18 row is sfw and the all-ages
	// row is nsfw. Reading content_rating instead inverts both.
	if got[8471].ContentLimit == "nsfw" || got[8472].ContentLimit == "sfw" {
		t.Error("the age rating won over the editorial display axis")
	}

	if asked.Get("nsfw") != "true" || asked.Has("content_limit") {
		t.Errorf("hydration gates = %v, want both open (nsfw=true, no content_limit)", asked)
	}
}
