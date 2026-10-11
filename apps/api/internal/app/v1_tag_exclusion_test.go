package app

import (
	"net/http"
	"slices"
	"testing"
)

func TestWorksListDropsExcludedWorks(t *testing.T) {
	f := newG5Fix(t)
	_, body := f.get(t, "/api/v1/works", "/works")
	if !g5HasID(body, g5SFW0) {
		t.Fatal("positive control: the work is listed without an exclusion")
	}

	f.cat.excluded = []int{g5SFW0}
	resp, body := f.get(t, "/api/v1/works?exclude_tag_trees=6,6&exclude_tag_ids=9", "/works")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if g5HasID(body, g5SFW0) {
		t.Fatal("an excluded work is still listed")
	}
	if !g5HasID(body, g5SFW0+1) {
		t.Fatal("a work the exclusion does not name went missing")
	}
	got := f.cat.exclusions[len(f.cat.exclusions)-1]
	if !slices.Equal(got.Trees, []int64{6}) || !slices.Equal(got.Exact, []int64{9}) {
		t.Fatalf("exclusion sent to catalog = %+v, want trees [6] exact [9]", got)
	}
}

func TestCatalogLanesForwardTheExclusion(t *testing.T) {
	f := newG5Fix(t)
	f.get(t, "/api/v1/library-works?exclude_tag_trees=6&exclude_tag_ids=9", "/library-works")
	q := f.cat.searched[len(f.cat.searched)-1]
	if q.Get("exclude_tag_tree") != "6" || q.Get("exclude_tag_id") != "9" {
		t.Fatalf("library-works sent %v", q)
	}

	f.get(t, "/api/v1/release-calendar?exclude_tag_trees=6", "/release-calendar")
	cq := f.cat.calQ[len(f.cat.calQ)-1]
	if cq.Get("exclude_tag_tree") != "6" {
		t.Fatalf("release-calendar sent %v", cq)
	}
	if cq.Has("exclude_tag_id") {
		t.Fatalf("release-calendar invented exclude_tag_id: %v", cq)
	}
}
