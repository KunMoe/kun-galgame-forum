package app

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

const (
	docSeedMin = 931000001
	docSeedMax = 931000010

	docBannerHash = "9f3c9f3c9f3c9f3c9f3c9f3c9f3c9f3c9f3c9f3c9f3c9f3c9f3c9f3c9f3c9f3c"
)

type docSeed struct {
	id, sortOrder, view int
	category            string
	pinned              bool
	published           time.Time
}

// Ties sit on every sort key and straddle page boundaries at limit 2 and 3:
// sort_order 1 three times, view 5 three times, published_time shared by
// three docs. Without the id tie-breaker a walk skips or repeats one of them.
func docSeeds() []docSeed {
	t0 := time.Date(2026, 5, 1, 8, 0, 0, 123_000_000, time.UTC)
	tie := t0.Add(48 * time.Hour)
	return []docSeed{
		{docSeedMin + 0, 0, 5, "galgame", true, t0},
		{docSeedMin + 1, 1, 5, "notice", false, tie},
		{docSeedMin + 2, 1, 5, "notice", true, tie},
		{docSeedMin + 3, 1, 1, "kun", false, tie},
		{docSeedMin + 4, 2, 1, "other", false, t0.Add(time.Hour)},
		{docSeedMin + 5, 3, 9, "galgame", true, t0.Add(2 * time.Hour)},
		{docSeedMin + 6, 3, 0, "notice", false, t0.Add(3 * time.Hour)},
		{docSeedMin + 7, 4, 0, "galgame", false, t0.Add(4 * time.Hour)},
		{docSeedMin + 8, 5, 3, "kun", true, t0.Add(5 * time.Hour)},
		{docSeedMin + 9, 5, 3, "galgame", false, t0.Add(6 * time.Hour)},
	}
}

func docSlug(id int) string {
	return fmt.Sprintf("dtest-%d", id)
}

func newDocFix(t *testing.T) *writeFix {
	t.Helper()
	f := newWriteFix(t, nil)
	f.alice(t)
	f.putSession(t, "sess-admin", w3UserStaff, "user", "admin")
	wipe := func() {
		if err := f.db.Exec(`DELETE FROM doc_article`).Error; err != nil {
			t.Fatal(err)
		}
		if err := f.db.Exec(`DELETE FROM doc_category WHERE slug NOT IN ('galgame', 'notice', 'kun', 'other')`).Error; err != nil {
			t.Fatal(err)
		}
	}
	wipe()
	t.Cleanup(wipe)
	for _, d := range docSeeds() {
		banner := ""
		if d.id == docSeedMin {
			banner = docBannerHash
		}
		edited := any(nil)
		if d.id == docSeedMin+1 {
			edited = d.published.Add(time.Hour)
		}
		if err := f.db.Exec(`INSERT INTO doc_article (
			id, title, slug, path, description, banner, banner_image_hash, status, is_pin, view,
			published_time, edited_time, content_markdown, category_id, author_id, created, updated, sort_order
		) VALUES (?, ?, ?, ?, 'desc', '/content/legacy/banner.avif', ?, 1, ?, ?, ?, ?, ?,
			(SELECT id FROM doc_category WHERE slug = ?), ?, ?, ?, ?)`,
			d.id, fmt.Sprintf("doc %d", d.id), docSlug(d.id), "/doc/"+docSlug(d.id), banner, d.pinned, d.view,
			d.published, edited, docBody, d.category, w3UserStaff, d.published, d.published, d.sortOrder).Error; err != nil {
			t.Fatal(err)
		}
	}
	return f
}

const docBody = "# Intro\n\nHello **world**.\n\n## Part two\n\n![shot](/image/" + docBannerHash + ")\n"

func (f *writeFix) docCall(t *testing.T, method, rawURL, specPath, session, idem string, hdr http.Header, payload any) (*http.Response, map[string]any) {
	t.Helper()
	resp, body := f.doJSON(t, method, rawURL, session, specPath, idem, hdr, payload)
	if len(body) == 0 {
		return resp, nil
	}
	return resp, problemMap(t, body)
}

func (f *writeFix) listDocs(t *testing.T, q url.Values) (*http.Response, map[string]any) {
	t.Helper()
	return f.docCall(t, http.MethodGet, "/api/v1/docs?"+q.Encode(), "/docs", "", "", nil, nil)
}

func (f *writeFix) walkDocs(t *testing.T, q url.Values) []string {
	t.Helper()
	var got []string
	for range 50 {
		resp, body := f.listDocs(t, q)
		if resp.StatusCode != http.StatusOK || body["object"] != "list" {
			t.Fatalf("list %v: %d %+v", q, resp.StatusCode, body)
		}
		got = append(got, adminItemIDs(body)...)
		next, _ := body["next_cursor"].(string)
		if next == "" {
			return got
		}
		q.Set("cursor", next)
	}
	t.Fatalf("walk %v did not end", q)
	return nil
}

func (f *writeFix) adminDocPath(id string) (string, string) {
	return "/api/v1/admin/docs/" + id, "/admin/docs/{doc_id}"
}

func (f *writeFix) createDoc(t *testing.T, session string, payload map[string]any) (*http.Response, map[string]any) {
	t.Helper()
	return f.docCall(t, http.MethodPost, "/api/v1/admin/docs", "/admin/docs", session, "", nil, payload)
}

func (f *writeFix) patchDoc(t *testing.T, id, session string, payload map[string]any) (*http.Response, map[string]any) {
	t.Helper()
	u, spec := f.adminDocPath(id)
	return f.docCall(t, http.MethodPatch, u, spec, session, "", nil, payload)
}

func (f *writeFix) putDocOrder(t *testing.T, session string, ids []string) (*http.Response, map[string]any) {
	t.Helper()
	return f.docCall(t, http.MethodPut, "/api/v1/admin/doc-order", "/admin/doc-order", session, "", nil,
		map[string]any{"doc_ids": ids})
}

func newDocBody(slug string) map[string]any {
	return map[string]any{
		"slug":             slug,
		"title":            "  New doc  ",
		"doc_category":     "notice",
		"content_markdown": "# Hi\n\nbody",
	}
}

func firstFieldError(body map[string]any) map[string]any {
	errs, _ := body["errors"].([]any)
	if len(errs) == 0 {
		return nil
	}
	e, _ := errs[0].(map[string]any)
	return e
}

func docSQLOrder(t *testing.T, f *writeFix, order string) []string {
	t.Helper()
	return f.sqlIDs(t, `SELECT id::text FROM doc_article ORDER BY `+order)
}

func joinIDs(ids []string) string {
	return strings.Join(ids, ",")
}
