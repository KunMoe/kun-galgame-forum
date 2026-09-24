package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

const logoCDN = "https://image.example.test"

func logoCatalog(t *testing.T) *GalgameClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/catalog/companies/107":
			_, _ = w.Write([]byte(`{"id":"107",` +
				`"display_name":"Purple SOFTWARE","logo":{"url":"https://cdn.example/abcd1234ef.webp","hash":"abcd1234ef","width":200,"height":60}}`))
		case "/v2/catalog/companies/309":
			_, _ = w.Write([]byte(`{"id":"309","display_name":"无标社","logo":null}`))
		case "/v2/catalog/companies/409":
			_, _ = w.Write([]byte(`{"id":"409","display_name":"旧契约社"}`))
		case "/v2/catalog/companies":
			_, _ = w.Write([]byte(`{"items":[` +
				`{"id":"107","display_name":"Purple SOFTWARE","work_count":42,"logo":{"url":"https://cdn.example/abcd1234ef.webp","hash":"abcd1234ef","width":200,"height":60}},` +
				`{"id":"309","display_name":"无标社","work_count":3}],` +
				`"next_cursor":null,"total":2}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":4,"message":"资源不存在"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return New(srv.URL, "test-key", logoCDN)
}

func TestImageURLFromHashFansOut(t *testing.T) {
	c := logoCatalog(t)
	const want = logoCDN + "/ab/cd/abcd1234ef.webp"
	if got := c.ImageURLFromHash("abcd1234ef"); got != want {
		t.Fatalf("logo URL = %q, want %q", got, want)
	}
	if got := c.ImageURLFromHash(""); got != "" {
		t.Fatalf("empty hash resolved to %q, want empty", got)
	}
	if got := c.ImageURLFromHash("ab"); got != "" {
		t.Fatalf("short hash resolved to %q, want empty", got)
	}
	if got := New("http://unused", "k", "").ImageURLFromHash("abcd1234ef"); got != "" {
		t.Fatalf("unconfigured CDN resolved to %q, want empty", got)
	}
}

func TestCatalogLabelCarriesLogoHash(t *testing.T) {
	c := logoCatalog(t)
	for _, tc := range []struct {
		id   string
		want string
	}{
		{"107", "abcd1234ef"},
		{"309", ""},
		{"409", ""},
	} {
		rec, found, movedTo, appErr := c.CatalogLabel(context.Background(), tc.id)
		if appErr != nil || !found || movedTo != 0 {
			t.Fatalf("label %s: found=%v movedTo=%d err=%v", tc.id, found, movedTo, appErr)
		}
		if rec.LogoHash != tc.want {
			t.Fatalf("label %s logo_hash = %q, want %q", tc.id, rec.LogoHash, tc.want)
		}
	}
}

func TestCatalogLabelListCarriesLogoHash(t *testing.T) {
	page, appErr := logoCatalog(t).CatalogTaxonomyList(context.Background(), "labels", url.Values{})
	if appErr != nil {
		t.Fatalf("list lane: %v", appErr)
	}
	rows := page.Items
	if len(rows) != 2 {
		t.Fatalf("list lane returned %d rows, want 2", len(rows))
	}
	if rows[0].LogoHash != "abcd1234ef" {
		t.Fatalf("row 0 logo_hash = %q, want abcd1234ef", rows[0].LogoHash)
	}
	if rows[1].LogoHash != "" {
		t.Fatalf("row 1 logo_hash = %q, want empty", rows[1].LogoHash)
	}
}
