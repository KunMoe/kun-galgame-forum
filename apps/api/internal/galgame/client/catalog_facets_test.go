package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCatalogFacetsRankByCountAndCap(t *testing.T) {
	var f catFacets
	raw := `{"tag_id":[{"value":"30","display_name":"30","count":234},{"value":"872","count":564},{"value":"9","count":234},{"value":"x","count":999},{"value":"5","count":0}]}`
	if err := json.Unmarshal([]byte(raw), &f); err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprint(f.Tags()); got != "[{872 564} {9 234} {30 234}]" {
		t.Errorf("most works first, ties by id, junk dropped: %s", got)
	}
	if got := f.Companies(); len(got) != 0 {
		t.Errorf("absent facet: %v", got)
	}

	many := catFacets{"company_id": {}}
	for i := 1; i <= 50; i++ {
		many["company_id"] = append(many["company_id"], catFacetBucket{Value: fmt.Sprint(i), Count: i})
	}
	if got := many.Companies(); len(got) != entityFacetCap || got[0].ID != 50 {
		t.Errorf("capped at %d, largest first: %v", entityFacetCap, got[:3])
	}
}

func TestEntityWorkIDsIntersectsEveryFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q := r.URL.Query()
		switch {
		case r.URL.Path == "/v2/catalog/tag-exclusion" && q.Get("exclude_tag_id") == "1":
			_, _ = w.Write([]byte(`{"tag_ids":["1"],"work_ids":["10","11","12","13"]}`))
		case r.URL.Path == "/v2/catalog/tag-exclusion" && q.Get("exclude_tag_id") == "2":
			_, _ = w.Write([]byte(`{"tag_ids":["2"],"work_ids":["11","12","14"]}`))
		case r.URL.Path == "/v2/catalog/works" && q.Get("company_id") == "7":
			_, _ = w.Write([]byte(`{"object":"list","items":[` +
				`{"object":"work","id":"12","display_name":"a","content_limit":"sfw","claim":{"site":"kungal","site_work_id":"12","state":"live","content_limit":"sfw"}},` +
				`{"object":"work","id":"14","display_name":"b","content_limit":"sfw","claim":{"site":"kungal","site_work_id":"14","state":"live","content_limit":"sfw"}}]}`))
		case r.URL.Path == "/v2/catalog/works":
			_, _ = w.Write([]byte(`{"object":"list","items":[]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	c := New(srv.URL, "test-key", "")

	if _, ok, appErr := c.EntityWorkIDs(context.Background(), nil, 0); ok || appErr != nil {
		t.Errorf("no filter must not restrict: %v %v", ok, appErr)
	}
	ids, ok, appErr := c.EntityWorkIDs(context.Background(), []int64{1, 2}, 0)
	if appErr != nil || !ok || fmt.Sprint(ids) != "[11 12]" {
		t.Errorf("tags are ANDed: %v %v %v", ids, ok, appErr)
	}
	ids, ok, appErr = c.EntityWorkIDs(context.Background(), []int64{1, 2}, 7)
	if appErr != nil || !ok || fmt.Sprint(ids) != "[12]" {
		t.Errorf("company narrows the tags: %v %v %v", ids, ok, appErr)
	}
	ids, ok, appErr = c.EntityWorkIDs(context.Background(), []int64{1}, 99)
	if appErr != nil || !ok || len(ids) != 0 {
		t.Errorf("a company with no works matches nothing, not everything: %v %v", ids, ok)
	}
}
