package client

import (
	"strings"
	"testing"
)

func TestStaleGIDsForCatalogIDs_NamesNonCanonicalAnchorRefs(t *testing.T) {
	row := strings.Replace(liveRow(61101, 61295, "Yan"),
		`"refs":[{"source":"dlsite","external_id":"RJ01"},{"source":"vndb","external_id":"v19658"}]`,
		`"refs":[`+
			`{"source":"curated","external_id":"5904"},`+
			`{"source":"curated","external_id":"61295"},`+
			`{"source":"galgame_wiki","external_id":"5904"},`+
			`{"source":"galgame_wiki","external_id":"7001"},`+
			`{"source":"dlsite","external_id":"RJ01"}]`,
		1)
	rec := &catalogRecorder{}
	srv := catalogStub(t, rec,
		map[string]int64{"61295": 61101},
		map[int64]string{61101: row},
	)
	c := New(srv.URL, "nm_test_key", "")

	got, appErr := c.StaleGIDsForCatalogIDs(t.Context(), []int64{61101})
	if appErr != nil {
		t.Fatalf("StaleGIDsForCatalogIDs: %v", appErr)
	}
	stale := got[61101]
	if len(stale) != 2 || stale[0] != 5904 || stale[1] != 7001 {
		t.Fatalf("stale = %v, want [5904 7001] (canonical 61295 and dlsite dropped, wiki 5904 deduped)", stale)
	}

	if rec.count() != 1 {
		t.Fatalf("made %d calls, want 1 (ids= fetch with include=refs)", rec.count())
	}
	if rec.pathAt(0) != "/v2/catalog/works" {
		t.Errorf("path = %q, want /v2/catalog/works", rec.pathAt(0))
	}
	q := rec.queryAt(0)
	if q.Get("ids") != "61101" {
		t.Errorf("ids = %q, want 61101", q.Get("ids"))
	}
	if inc := q.Get("include"); inc != "refs" {
		t.Errorf("include = %q, want refs", inc)
	}
	if q.Get("nsfw") != "true" {
		t.Errorf("nsfw = %q, want true", q.Get("nsfw"))
	}
}

func TestStaleGIDsForCatalogIDs_EmptyIDs(t *testing.T) {
	c := New("http://127.0.0.1:1", "nm_test_key", "")
	got, appErr := c.StaleGIDsForCatalogIDs(t.Context(), nil)
	if appErr != nil {
		t.Fatalf("empty ids: %v", appErr)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("got %#v, want an empty map", got)
	}
}
