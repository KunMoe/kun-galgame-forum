package app

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"kun-galgame-api/pkg/perm"
)

func TestV1DocsWalkEverySort(t *testing.T) {
	f := newDocFix(t)
	for _, c := range []struct{ sort, order string }{
		{"", "sort_order ASC, id ASC"},
		{"position_asc", "sort_order ASC, id ASC"},
		{"published_desc", "published_time DESC, id DESC"},
		{"views_desc", "view DESC, id DESC"},
	} {
		want := docSQLOrder(t, f, c.order)
		for _, limit := range []int{1, 2, 3} {
			q := url.Values{"limit": {fmt.Sprint(limit)}}
			if c.sort != "" {
				q.Set("sort", c.sort)
			}
			if got := f.walkDocs(t, q); joinIDs(got) != joinIDs(want) {
				t.Errorf("sort %q limit %d walked %v, want %v", c.sort, limit, got, want)
			}
		}
	}
}

func TestV1DocsFilters(t *testing.T) {
	f := newDocFix(t)
	got := f.walkDocs(t, url.Values{"limit": {"2"}, "doc_category": {"notice"}})
	want := f.sqlIDs(t, `SELECT a.id::text FROM doc_article a JOIN doc_category c ON c.id = a.category_id
		WHERE c.slug = 'notice' ORDER BY a.sort_order, a.id`)
	if joinIDs(got) != joinIDs(want) || len(want) != 3 {
		t.Errorf("notice shelf %v, want %v", got, want)
	}
	got = f.walkDocs(t, url.Values{"limit": {"2"}, "is_pinned": {"true"}, "sort": {"views_desc"}})
	want = docSQLOrder(t, f, "is_pin DESC, view DESC, id DESC")[:4]
	if joinIDs(got) != joinIDs(want) {
		t.Errorf("pinned %v, want %v", got, want)
	}
	if got = f.walkDocs(t, url.Values{"is_pinned": {"false"}}); len(got) != 6 {
		t.Errorf("unpinned %v", got)
	}

	_, body := f.listDocs(t, url.Values{"limit": {"2"}, "doc_category": {"notice"}})
	cur, _ := body["next_cursor"].(string)
	for _, q := range []url.Values{
		{"cursor": {cur}, "limit": {"2"}, "doc_category": {"kun"}},
		{"cursor": {cur}, "limit": {"2"}},
		{"cursor": {cur}, "limit": {"2"}, "doc_category": {"notice"}, "is_pinned": {"true"}},
		{"cursor": {cur}, "limit": {"2"}, "doc_category": {"notice"}, "sort": {"views_desc"}},
	} {
		resp, body := f.listDocs(t, q)
		if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_CURSOR" {
			t.Errorf("cursor reused with %v: %d %+v", q, resp.StatusCode, body)
		}
	}

	for q, code := range map[string]string{
		"sort=title_asc":      "UNKNOWN_SORT",
		"doc_category=help":   "UNKNOWN_ENUM_VALUE",
		"limit=101":           "LIMIT_TOO_LARGE",
		"is_pinned=yes":       "INVALID_PARAMETER",
		"cursor=cur_not-real": "INVALID_CURSOR",
	} {
		resp, body := f.docCall(t, http.MethodGet, "/api/v1/docs?"+q, "/docs", "", "", nil, nil)
		if resp.StatusCode != http.StatusBadRequest || body["code"] != code {
			t.Errorf("%s: %d %+v, want %s", q, resp.StatusCode, body, code)
		}
	}
}

func TestV1DocsSummaryShape(t *testing.T) {
	f := newDocFix(t)
	_, body := f.listDocs(t, url.Values{"limit": {"2"}})
	items, _ := body["items"].([]any)
	first, _ := items[0].(map[string]any)
	banner, _ := first["banner"].(map[string]any)
	if first["object"] != "doc" || first["slug"] != docSlug(docSeedMin) || first["doc_category"] != "galgame" ||
		first["is_pinned"] != true || banner["hash"] != docBannerHash || first["edited_at"] != nil {
		t.Fatalf("first %+v", first)
	}
	second, _ := items[1].(map[string]any)
	if second["banner"] != nil {
		t.Errorf("a legacy banner path without a hash is no Image: %+v", second["banner"])
	}
	for _, k := range []string{"content", "content_markdown", "path", "status", "created_at", "updated_at", "sort_order"} {
		if _, has := first[k]; has {
			t.Errorf("summary carries %s", k)
		}
	}
}

func TestV1DocGetCountsViewsWithoutTouchingUpdated(t *testing.T) {
	f := newDocFix(t)
	slug := docSlug(docSeedMin + 1)
	updated := func() string {
		var s string
		if err := f.db.Raw(`SELECT updated::text FROM doc_article WHERE slug = ?`, slug).Scan(&s).Error; err != nil {
			t.Fatal(err)
		}
		return s
	}
	before := updated()
	for want := 6; want <= 7; want++ {
		resp, body := f.docCall(t, http.MethodGet, "/api/v1/docs/"+slug, "/docs/{doc_slug}", "", "", nil, nil)
		if resp.StatusCode != http.StatusOK || asInt(body["view_count"]) != want {
			t.Fatalf("read %d: %d %+v", want, resp.StatusCode, body)
		}
	}
	if after := updated(); after != before {
		t.Errorf("a view moved updated from %s to %s", before, after)
	}
	if n := f.scalar(t, `SELECT view FROM doc_article WHERE slug = ?`, slug); n != 7 {
		t.Errorf("stored view %d", n)
	}
}

func TestV1DocGetShape(t *testing.T) {
	f := newDocFix(t)
	resp, body := f.docCall(t, http.MethodGet, "/api/v1/docs/"+docSlug(docSeedMin+1), "/docs/{doc_slug}", "sess-alice", "", nil, nil)
	if resp.StatusCode != http.StatusOK || body["object"] != "doc" || body["edited_at"] == nil {
		t.Fatalf("get %d %+v", resp.StatusCode, body)
	}
	author, _ := body["author"].(map[string]any)
	if author["id"] != fmt.Sprint(w3UserStaff) || author["name"] != "staff" {
		t.Errorf("author %+v", author)
	}
	doc, _ := body["content"].(map[string]any)
	children, _ := doc["children"].([]any)
	var headings []string
	var image map[string]any
	for _, c := range children {
		n, _ := c.(map[string]any)
		switch n["object"] {
		case "heading":
			headings = append(headings, fmt.Sprint(n["anchor"]))
		case "paragraph":
			for _, in := range n["children"].([]any) {
				if m, _ := in.(map[string]any); m["object"] == "image" {
					image = m
				}
			}
		}
	}
	if len(headings) != 2 || headings[0] == "" || image == nil || image["image"] == nil {
		t.Errorf("content headings %v image %+v", headings, image)
	}
	for _, k := range []string{"content_html", "toc", "content_markdown"} {
		if _, has := body[k]; has {
			t.Errorf("doc carries %s", k)
		}
	}

	for path, status := range map[string]int{"no-such-doc": 404, "Bad_Slug": 400} {
		resp, body := f.docCall(t, http.MethodGet, "/api/v1/docs/"+path, "/docs/{doc_slug}", "", "", nil, nil)
		if resp.StatusCode != status {
			t.Errorf("%s: %d %+v", path, resp.StatusCode, body)
		}
	}
}

func TestV1DocUnknownCategoryIsNotFiledElsewhere(t *testing.T) {
	f := newDocFix(t)
	if err := f.db.Exec(`INSERT INTO doc_category (slug, title, updated) VALUES ('help', 'Help', now())`).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Exec(`UPDATE doc_article SET category_id = (SELECT id FROM doc_category WHERE slug = 'help') WHERE id = ?`, docSeedMin).Error; err != nil {
		t.Fatal(err)
	}
	if resp, body := f.listDocs(t, url.Values{}); resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("list %d %+v", resp.StatusCode, body)
	}
	resp, body := f.docCall(t, http.MethodGet, "/api/v1/docs/"+docSlug(docSeedMin), "/docs/{doc_slug}", "", "", nil, nil)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("get %d %+v", resp.StatusCode, body)
	}
}

func TestV1AdminDocFacesNeedTheirPermissions(t *testing.T) {
	f := newDocFix(t)
	id := fmt.Sprint(docSeedMin)
	u, spec := f.adminDocPath(id)
	bearer := http.Header{"Authorization": {"Bearer staff-token"}}
	patch := map[string]any{"is_pinned": false}
	for _, c := range []struct {
		name, method, url, spec, session string
		hdr                              http.Header
		payload                          any
		status                           int
	}{
		{"view anonymously", http.MethodGet, u, spec, "", nil, nil, 401},
		{"view as a user", http.MethodGet, u, spec, "sess-alice", nil, nil, 403},
		{"view as a Bearer moderator", http.MethodGet, u, spec, "", bearer, nil, 403},
		{"create as a user", http.MethodPost, "/api/v1/admin/docs", "/admin/docs", "sess-alice", nil, newDocBody("x-user"), 403},
		{"create as a Bearer moderator", http.MethodPost, "/api/v1/admin/docs", "/admin/docs", "", bearer, newDocBody("x-bearer"), 403},
		{"patch as a user", http.MethodPatch, u, spec, "sess-alice", nil, patch, 403},
		{"delete as a user", http.MethodDelete, u, spec, "sess-alice", nil, nil, 403},
		{"delete as a Bearer moderator", http.MethodDelete, u, spec, "", bearer, nil, 403},
		{"reorder as a user", http.MethodPut, "/api/v1/admin/doc-order", "/admin/doc-order", "sess-alice", nil, map[string]any{"doc_ids": []string{id}}, 403},
	} {
		resp, body := f.docCall(t, c.method, c.url, c.spec, c.session, "", c.hdr, c.payload)
		if resp.StatusCode != c.status {
			t.Errorf("%s: %d %+v", c.name, resp.StatusCode, body)
		}
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM doc_article`); n != 10 {
		t.Fatalf("a refused call changed the docs: %d", n)
	}

	perm.SetUserOverrides(map[int][]perm.Override{w3UserStaff: {{Permission: perm.DocDelete, Effect: perm.EffectRevoke}}})
	t.Cleanup(func() { perm.SetUserOverrides(nil) })
	if resp, body := f.patchDoc(t, id, "sess-staff", map[string]any{"title": "still editable"}); resp.StatusCode != http.StatusOK {
		t.Errorf("patch without doc.delete: %d %+v", resp.StatusCode, body)
	}
	if resp, body := f.docCall(t, http.MethodDelete, u, spec, "sess-staff", "", nil, nil); resp.StatusCode != http.StatusForbidden ||
		body["code"] != "PERMISSION_REQUIRED" {
		t.Errorf("delete without doc.delete: %d %+v", resp.StatusCode, body)
	}
}

func TestV1AdminDocCreate(t *testing.T) {
	f := newDocFix(t)
	payload := newDocBody("fresh-doc")
	payload["banner_image_hash"] = docBannerHash
	resp, body := f.docCall(t, http.MethodPost, "/api/v1/admin/docs", "/admin/docs", "sess-staff", "01J8Z5W6N7Q8R9S0T1V2W3X4Y5", nil, payload)
	if resp.StatusCode != http.StatusCreated || body["object"] != "admin_doc" || body["title"] != "New doc" ||
		body["description"] != "" || body["is_pinned"] != false || body["edited_at"] != nil {
		t.Fatalf("create %d %+v", resp.StatusCode, body)
	}
	id := strID(body["id"])
	if loc := resp.Header.Get("Location"); loc != "/api/v1/admin/docs/"+id {
		t.Errorf("Location %q", loc)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM doc_article WHERE id = ? AND path = '/doc/fresh-doc' AND author_id = ? AND status = 1`, id, w3UserStaff); n != 1 {
		t.Errorf("stored row is off")
	}
	walk := f.walkDocs(t, url.Values{"limit": {"4"}})
	if walk[len(walk)-1] != id {
		t.Errorf("a new doc goes last in position order: %v", walk)
	}

	replay, again := f.docCall(t, http.MethodPost, "/api/v1/admin/docs", "/admin/docs", "sess-staff", "01J8Z5W6N7Q8R9S0T1V2W3X4Y5", nil, payload)
	if replay.StatusCode != http.StatusCreated || strID(again["id"]) != id || replay.Header.Get("Idempotency-Replayed") != "true" {
		t.Errorf("replay %d %+v", replay.StatusCode, again)
	}

	resp, body = f.createDoc(t, "sess-staff", newDocBody(docSlug(docSeedMin)))
	if resp.StatusCode != http.StatusConflict || body["code"] != "ALREADY_EXISTS" {
		t.Errorf("taken slug %d %+v", resp.StatusCode, body)
	}

	blank := newDocBody("blank-doc")
	blank["title"] = "   "
	resp, body = f.createDoc(t, "sess-staff", blank)
	if e := firstFieldError(body); resp.StatusCode != http.StatusUnprocessableEntity || e["pointer"] != "/title" || e["reason"] != "TOO_SHORT" {
		t.Errorf("blank title %d %+v", resp.StatusCode, body)
	}
	for field, value := range map[string]any{"slug": "Has Caps", "doc_category": "help", "banner_image_hash": "abc"} {
		bad := newDocBody("bad-doc")
		bad[field] = value
		if resp, body := f.createDoc(t, "sess-staff", bad); resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("bad %s: %d %+v", field, resp.StatusCode, body)
		}
	}
}

func TestV1AdminDocPatch(t *testing.T) {
	f := newDocFix(t)
	id := fmt.Sprint(docSeedMin)
	if resp, body := f.patchDoc(t, id, "sess-staff", map[string]any{"is_pinned": false}); resp.StatusCode != http.StatusOK ||
		body["is_pinned"] != false || body["edited_at"] != nil {
		t.Fatalf("pin only %d %+v", resp.StatusCode, body)
	}
	if resp, body := f.patchDoc(t, id, "sess-staff", map[string]any{"title": "doc 931000001"}); resp.StatusCode != http.StatusOK || body["edited_at"] != nil {
		t.Errorf("same title is no edit: %d %+v", resp.StatusCode, body)
	}
	resp, body := f.patchDoc(t, id, "sess-staff", map[string]any{"title": " Renamed ", "slug": "renamed-doc", "banner_image_hash": ""})
	if resp.StatusCode != http.StatusOK || body["title"] != "Renamed" || body["slug"] != "renamed-doc" ||
		body["banner"] != nil || body["edited_at"] == nil {
		t.Fatalf("edit %d %+v", resp.StatusCode, body)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM doc_article WHERE id = ? AND path = '/doc/renamed-doc'`, id); n != 1 {
		t.Errorf("path did not follow the slug")
	}
	if resp, _ := f.docCall(t, http.MethodGet, "/api/v1/docs/renamed-doc", "/docs/{doc_slug}", "", "", nil, nil); resp.StatusCode != http.StatusOK {
		t.Errorf("new slug %d", resp.StatusCode)
	}

	resp, body = f.patchDoc(t, id, "sess-staff", map[string]any{"slug": docSlug(docSeedMin + 1)})
	if resp.StatusCode != http.StatusConflict || body["code"] != "ALREADY_EXISTS" {
		t.Errorf("taken slug %d %+v", resp.StatusCode, body)
	}
	resp, body = f.patchDoc(t, id, "sess-staff", map[string]any{})
	if e := firstFieldError(body); resp.StatusCode != http.StatusUnprocessableEntity || e["reason"] != "REQUIRED" {
		t.Errorf("empty patch %d %+v", resp.StatusCode, body)
	}
	resp, body = f.patchDoc(t, "999999999", "sess-staff", map[string]any{"is_pinned": true})
	if resp.StatusCode != http.StatusNotFound || body["code"] != "NOT_FOUND" {
		t.Errorf("missing doc %d %+v", resp.StatusCode, body)
	}
}

func TestV1AdminDocGetAndDelete(t *testing.T) {
	f := newDocFix(t)
	id := fmt.Sprint(docSeedMin + 2)
	u, spec := f.adminDocPath(id)
	resp, body := f.docCall(t, http.MethodGet, u, spec, "sess-staff", "", nil, nil)
	if resp.StatusCode != http.StatusOK || body["object"] != "admin_doc" || body["content_markdown"] != docBody {
		t.Fatalf("staff view %d %+v", resp.StatusCode, body)
	}
	if resp, body := f.docCall(t, http.MethodDelete, u, spec, "sess-staff", "", nil, nil); resp.StatusCode != http.StatusNoContent || body != nil {
		t.Fatalf("delete %d %+v", resp.StatusCode, body)
	}
	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		if resp, body := f.docCall(t, method, u, spec, "sess-staff", "", nil, nil); resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s after delete %d %+v", method, resp.StatusCode, body)
		}
	}
}

func TestV1DocOrderReplacesTheWholeSequence(t *testing.T) {
	f := newDocFix(t)
	all := docSQLOrder(t, f, "id DESC")
	if resp, body := f.putDocOrder(t, "sess-staff", all); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("reorder %d %+v", resp.StatusCode, body)
	}
	if got := f.walkDocs(t, url.Values{"limit": {"3"}}); joinIDs(got) != joinIDs(all) {
		t.Errorf("order %v, want %v", got, all)
	}
	stamp := func() string { return joinIDs(docSQLOrder(t, f, "sort_order, id")) }
	before := stamp()

	resp, body := f.putDocOrder(t, "sess-staff", all[1:])
	e := firstFieldError(body)
	params, _ := e["params"].(map[string]any)
	if resp.StatusCode != http.StatusUnprocessableEntity || e["pointer"] != "/doc_ids" || e["reason"] != "TOO_FEW_ITEMS" ||
		asInt(params["min_items"]) != len(all) {
		t.Errorf("partial list %d %+v", resp.StatusCode, body)
	}
	withUnknown := append([]string{all[0], "999999999"}, all[1:]...)
	resp, body = f.putDocOrder(t, "sess-staff", withUnknown)
	if e := firstFieldError(body); resp.StatusCode != http.StatusUnprocessableEntity || e["pointer"] != "/doc_ids/1" || e["reason"] != "UNKNOWN_REFERENCE" {
		t.Errorf("unknown id %d %+v", resp.StatusCode, body)
	}
	if resp, body := f.putDocOrder(t, "sess-staff", append(all, all[0])); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("duplicate id %d %+v", resp.StatusCode, body)
	}
	if after := stamp(); after != before {
		t.Errorf("a refused reorder wrote: %s → %s", before, after)
	}
}

func TestV1DocEditedAtIsUTCSeconds(t *testing.T) {
	f := newDocFix(t)
	_, body := f.patchDoc(t, fmt.Sprint(docSeedMin), "sess-staff", map[string]any{"description": "changed"})
	at, _ := body["edited_at"].(string)
	if _, err := time.Parse("2006-01-02T15:04:05Z", at); err != nil {
		t.Errorf("edited_at %q: %v", at, err)
	}
}

func TestV1DocGetIsUnavailableWhenAccountsAre(t *testing.T) {
	f := newDocFix(t)
	f.failOA.Store(true)
	resp, body := f.docCall(t, http.MethodGet, "/api/v1/docs/"+docSlug(docSeedMin), "/docs/{doc_slug}", "", "", nil, nil)
	if resp.StatusCode != http.StatusServiceUnavailable || body["code"] != "SERVICE_UNAVAILABLE" {
		t.Errorf("accounts down: %d %+v", resp.StatusCode, body)
	}
	if resp, body := f.listDocs(t, url.Values{}); resp.StatusCode != http.StatusOK {
		t.Errorf("the list needs no accounts: %d %+v", resp.StatusCode, body)
	}
}
