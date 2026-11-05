package catalogclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestCurrentRatingAsksForTheOnePersonAndWork(t *testing.T) {
	var got url.Values
	body := `{"object":"list","items":[{"object":"rating","id":"77","work_id":"9","user_id":"5","overall":8}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	c := New(Config{BaseURL: srv.URL, AppKey: "nmk_test"})

	id, found, err := c.CurrentRating(t.Context(), 5, 9)
	if err != nil || !found || id != 77 {
		t.Fatalf("CurrentRating = %d %v %v", id, found, err)
	}
	if got.Get("user_id") != "5" || got.Get("work_id") != "9" || got.Get("nsfw") != "true" {
		t.Errorf("query = %v, want the person, the work and adult works included", got)
	}

	body = `{"object":"list","items":[]}`
	if _, found, err := c.CurrentRating(t.Context(), 5, 9); err != nil || found {
		t.Errorf("an empty list = found %v err %v, want none", found, err)
	}
}

func TestRatingCarriesTheCookedSummary(t *testing.T) {
	var v v2Rating
	raw := `{"id":"1","work_id":"2","user_id":"3","overall":7,"short_summary":"a ||b||",
		"short_summary_html":"<p>a <span class=\"kun-spoiler kun-spoiler-hidden\">b</span></p>",
		"short_summary_text":"a ███","short_summary_has_spoiler":true}`
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		t.Fatal(err)
	}
	r := v.rating()
	if r.ShortSummary != "a ||b||" || r.ShortSummaryText != "a ███" || !r.ShortSummaryHasSpoiler ||
		r.ShortSummaryHTML == "" {
		t.Fatalf("rating = %+v", r)
	}
}
