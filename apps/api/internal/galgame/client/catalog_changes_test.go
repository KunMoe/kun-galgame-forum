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

func TestMirrorByCatalogIDsKeysByCatalogID(t *testing.T) {
	var asked url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = r.URL.Query()
		_, _ = w.Write([]byte(`{"object":"list","items":[
			{"id":"923","content_rating":"r18","content_limit":"sfw","claim":{"site":"kungal","site_work_id":"923","state":"live","content_limit":"sfw"}},
			{"id":"924","content_rating":"all_ages","content_limit":"nsfw","claim":{"site":"kungal","site_work_id":"924","state":"live","content_limit":"nsfw"}},
			{"id":"925","content_rating":"r18","content_limit":"nsfw"},
			{"id":"926","content_rating":"r18","content_limit":"sfw","claim":{"site":"moyu","site_work_id":"77","state":"live","content_limit":"sfw"}},
			{"id":"927","content_rating":"r18","content_limit":"nsfw","claim":{"site":"kungal","site_work_id":"927","state":"hidden","content_limit":"nsfw"}},
			{"id":"928","content_rating":"all_ages","content_limit":"nsfw"},
			{"id":"929","content_rating":"all_ages"}
		]}`))
	}))
	defer srv.Close()

	got, hidden, appErr := New(srv.URL, "nmk_test", "").
		MirrorByCatalogIDs(t.Context(), []int64{923, 924, 925, 926, 927, 928, 929})
	if appErr != nil {
		t.Fatalf("MirrorByCatalogIDs: %v", appErr.Message)
	}

	want := map[int]string{923: "sfw", 924: "nsfw", 925: "nsfw", 926: "sfw", 928: "nsfw", 929: ""}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for id, limit := range want {
		if got[id].ContentLimit != limit {
			t.Errorf("work %d = %q, want %q", id, got[id].ContentLimit, limit)
		}
	}
	if _, ok := got[927]; ok {
		t.Errorf("hidden claim leaked: %v", got)
	}
	if len(hidden) != 1 || hidden[927].ContentLimit != "nsfw" {
		t.Errorf("hidden = %v, want 927 with its verdict, so a row hidden before its first verdict still gets one", hidden)
	}

	if got[923].ContentLimit == "nsfw" || got[924].ContentLimit == "sfw" {
		t.Error("the age rating won over the editorial display axis")
	}

	if asked.Get("nsfw") != "true" || asked.Has("content_limit") {
		t.Errorf("hydration gates = %v, want both open (nsfw=true, no content_limit)", asked)
	}
}
