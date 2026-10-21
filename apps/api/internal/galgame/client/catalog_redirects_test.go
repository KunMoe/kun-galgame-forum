package client

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func redirectsServer(t *testing.T, body string) (*GalgameClient, *url.Values) {
	t.Helper()
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return New(srv.URL, "nmk_test", ""), &got
}

func TestCatalogRedirectsReadsTheStringIDs(t *testing.T) {
	// old_id and current_id arrive as strings and stay strings: rewriteV2Object
	// numbers id/work_id/canonical_id/character_id/person_id/from/to, and these
	// two are not on that list. Declaring them int64 fails the WHOLE page with
	// "cannot unmarshal string", so every merge would be invisible while the
	// drain reported success.
	c, _ := redirectsServer(t, `{"object":"list","items":[
		{"object":"redirect","target_object":"work","old_id":"216649","current_id":"2797","merged_at":"2026-08-29T13:23:54Z"},
		{"object":"redirect","target_object":"company","old_id":"31","current_id":"32","merged_at":"2026-08-29T13:23:55Z"}
	],"next_cursor":"cur_next"}`)

	page, appErr := c.CatalogRedirects(t.Context(), "", 100)
	if appErr != nil {
		t.Fatalf("CatalogRedirects: %v", appErr.Message)
	}
	if len(page.Items) != 1 {
		t.Fatalf("kept %d rows, want only the work: %+v", len(page.Items), page.Items)
	}
	if page.Items[0].OldID != 216649 || page.Items[0].CurrentID != 2797 {
		t.Errorf("row = %+v, want 216649 -> 2797", page.Items[0])
	}
	if page.NextCursor != "cur_next" {
		t.Errorf("next_cursor = %q", page.NextCursor)
	}
}

func TestCatalogRedirectsAsksForWorksOnly(t *testing.T) {
	c, got := redirectsServer(t, `{"object":"list","items":[]}`)

	if _, appErr := c.CatalogRedirects(t.Context(), "cur_abc", 1000); appErr != nil {
		t.Fatalf("CatalogRedirects: %v", appErr.Message)
	}
	if v := got.Get("object"); v != "work" {
		t.Errorf("object = %q, want work — the feed carries all eight families", v)
	}
	// Over 100 is 400 LIMIT_TOO_LARGE, not a clamp.
	if v := got.Get("limit"); v != "100" {
		t.Errorf("limit = %q, want the collection ceiling 100", v)
	}
	if v := got.Get("cursor"); v != "cur_abc" {
		t.Errorf("cursor = %q", v)
	}

	if _, appErr := c.CatalogRedirects(t.Context(), "", 100); appErr != nil {
		t.Fatalf("CatalogRedirects: %v", appErr.Message)
	}
	if got.Has("cursor") {
		t.Error("an empty cursor must be omitted, not sent as cursor=")
	}
}

func TestCatalogRedirectsCarriesRevoked(t *testing.T) {
	c, _ := redirectsServer(t, `{"object":"list","items":[
		{"object":"redirect","target_object":"work","old_id":"5904","current_id":"5904","revoked":true,"merged_at":"2026-10-21T09:00:00Z"},
		{"object":"redirect","target_object":"work","old_id":"7001","current_id":"7002","merged_at":"2026-10-21T09:00:01Z"}
	]}`)

	page, appErr := c.CatalogRedirects(t.Context(), "", 100)
	if appErr != nil {
		t.Fatalf("CatalogRedirects: %v", appErr.Message)
	}
	if len(page.Items) != 2 || !page.Items[0].Revoked || page.Items[1].Revoked {
		t.Errorf("rows = %+v, want only 5904 revoked", page.Items)
	}
}
