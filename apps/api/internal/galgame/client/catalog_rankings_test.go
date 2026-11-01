package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

const rankingFixture = `{"object":"list","items":[{"object":"ranking_entry","rank":1,"value":4.86,` +
	`"work":{"object":"work","id":"211063","medium":"galgame","display_name":"妹!せいかつ～ファンタジー～","latin":null,` +
	`"localized":{"zh-Hans":{"value":"妹！相随 ~黑白世界的缤纷冒险~","is_machine":false}},"olang":"ja","content_rating":"r18","content_limit":"sfw",` +
	`"release_date":"2022-02-10","release_date_precision":"day","release_status":"released",` +
	`"cover":{"url":"https://cdn.example/83/bc/83bc.webp","hash":"83bc","width":600,"height":900,"thumbhash":"6/cJ","sexual":"suggestive","violence":null,"source":"bangumi","origin":"cover"},` +
	`"claim":{"site":"kungal","site_work_id":"211063","state":"live","content_limit":"sfw"},"created_at":"2026-07-08T04:41:15Z","updated_at":"2026-10-21T21:20:04Z",` +
	`"refs":[{"source":"bangumi","external_id":"348651"}],` +
	`"companies":[{"object":"company","id":"12002","display_name":"いぬすく","localized":{},"company_kind":"doujin_circle","attribution_role":"circle","work_count":9}]},` +
	`"consensus":{"score":72.67,"lower":68.27,"upper":77.06,"percentile":99.35,"source_count":5,"vote_count":36064,"divergence":null},` +
	`"per_source":[{"source":"dlsite","score":4.87,"vote_count":34881,"mean":4.87,"lower":4.86,"upper":4.88,"median":null,"spread":0.46,"percentile":99.95}]}],` +
	`"total":46383,"total_relation":"eq","by":"dlsite","metric":"lower","direction":"desc","unit":"native",` +
	`"model":{"object":"rating_model","id":"1","version":1,"fitted_at":"2026-11-01T09:11:35.837267Z","interval":0.9,` +
	`"sources":[{"source":"dlsite","scale_min":1,"scale_max":5,"prior_votes":8.49,"works":46388,"in_consensus":true}]}}`

func TestCatalogRankingsReadsTheContractShape(t *testing.T) {
	var asked *url.URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = r.URL
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(rankingFixture))
	}))
	t.Cleanup(srv.Close)
	c := New(srv.URL, "test-key", "")

	page, appErr := c.CatalogRankings(context.Background(), url.Values{
		"by": {"dlsite"}, "page": {"2"}, "include": {"names,covers,refs,labels"},
	})
	if appErr != nil {
		t.Fatal(appErr)
	}
	if asked.Path != "/v2/catalog/rankings" {
		t.Errorf("path %s", asked.Path)
	}
	if q := asked.Query(); q.Get("include") != "titles,covers,refs,companies" || q.Get("page") != "2" || q.Get("cursor") != "" {
		t.Errorf("rankings takes page= itself and v2 include tokens: %s", asked.RawQuery)
	}

	if page.Total != 46383 || len(page.Items) != 1 || page.Model.FittedAt != "2026-11-01T09:11:35.837267Z" {
		t.Fatalf("page %+v", page)
	}
	if sc := page.Model.Sources; len(sc) != 1 || sc[0].Source != "dlsite" || sc[0].ScaleMin != 1 || sc[0].ScaleMax != 5 {
		t.Errorf("scales %+v", sc)
	}
	e := page.Items[0]
	if e.Rank != 1 || e.Value != 4.86 {
		t.Errorf("entry %+v", e)
	}
	w := e.Work
	if w.ID != 211063 || w.Cover != "https://cdn.example/83/bc/83bc.webp" || w.Localized["zh-Hans"].Value == "" || len(w.Labels) != 1 || len(w.Refs) != 1 {
		t.Errorf("the work inside an entry must fold like a works row: %+v", w)
	}
	if !CatalogItemRenderable(&w) {
		t.Error("a live sfw work must render")
	}
	if c := e.Consensus; c == nil || c.Score != 72.67 || c.SourceCount != 5 || c.VoteCount != 36064 || c.Divergence != nil {
		t.Errorf("consensus %+v", e.Consensus)
	}
	if s := e.PerSource; len(s) != 1 || s[0].Score != 4.87 || s[0].Mean != 4.87 || s[0].Median != nil || s[0].Spread == nil || *s[0].Spread != 0.46 {
		t.Errorf("per_source %+v", e.PerSource)
	}
}
