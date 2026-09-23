package app

import (
	"fmt"
	"net/http"
	"testing"
)

func (f *writeFix) walkVocabulary(t *testing.T, path string, limit int) []string {
	t.Helper()
	var got []string
	cursor := ""
	for page := 0; page < 100; page++ {
		url := fmt.Sprintf("/api/v1%s?limit=%d", path, limit)
		if cursor != "" {
			url += "&cursor=" + cursor
		}
		resp, body := f.ws(t, http.MethodGet, url, path, "", nil, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("walk %s: %d %+v", url, resp.StatusCode, body)
		}
		got = append(got, adminItemIDs(body)...)
		next, _ := body["next_cursor"].(string)
		if next == "" {
			return got
		}
		cursor = next
	}
	t.Fatal("walk did not end")
	return nil
}

func TestV1WebsiteVocabulariesWalk(t *testing.T) {
	f := newWebsiteFix(t)
	for _, c := range []struct{ path, sql string }{
		{"/website-categories", `SELECT id::text FROM galgame_website_category ORDER BY sort_order, id`},
		{"/website-tag-groups", `SELECT id::text FROM galgame_website_tag_group ORDER BY sort_order, id`},
		{"/website-tags", `SELECT id::text FROM galgame_website_tag ORDER BY id`},
	} {
		want := f.sqlIDs(t, c.sql)
		for _, limit := range []int{1, 2} {
			if got := f.walkVocabulary(t, c.path, limit); fmt.Sprint(got) != fmt.Sprint(want) {
				t.Errorf("%s limit %d walked %v, want %v", c.path, limit, got, want)
			}
		}
	}
}

func TestV1WebsiteCategories(t *testing.T) {
	f := newWebsiteFix(t)
	resp, body := f.ws(t, http.MethodGet, "/api/v1/website-categories/wt-main", "/website-categories/{website_category_slug}", "", nil, nil)
	want := f.scalar(t, `SELECT COUNT(*) FROM galgame_website WHERE category_id = ?`, wsCatMain)
	if resp.StatusCode != http.StatusOK || body["object"] != "website_category" || asInt(body["website_count"]) != want || body["label"] != "label wt-main" {
		t.Errorf("by slug %d %+v (want count %d)", resp.StatusCode, body, want)
	}
	resp, body = f.ws(t, http.MethodGet, "/api/v1/website-categories/wt-none", "/website-categories/{website_category_slug}", "", nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown slug %d %+v", resp.StatusCode, body)
	}

	resp, body = f.ws(t, http.MethodPost, "/api/v1/admin/website-categories", "/admin/website-categories", "sess-staff", nil,
		map[string]any{"slug": "wt-new", "label": "New", "sort_order": 5})
	if resp.StatusCode != http.StatusCreated || body["object"] != "admin_website_category" || asInt(body["sort_order"]) != 5 {
		t.Fatalf("create %d %+v", resp.StatusCode, body)
	}
	id := strID(body["id"])
	if resp.Header.Get("Location") != "/api/v1/admin/website-categories/"+id {
		t.Errorf("location %s", resp.Header.Get("Location"))
	}
	for _, c := range []struct {
		method, url string
		payload     any
		status      int
		code        string
	}{
		{http.MethodPost, "/api/v1/admin/website-categories", map[string]any{"slug": "wt-main", "label": "x"}, 409, "ALREADY_EXISTS"},
		{http.MethodPatch, "/api/v1/admin/website-categories/" + id, map[string]any{"slug": "wt-main"}, 409, "ALREADY_EXISTS"},
		{http.MethodPost, "/api/v1/admin/website-categories", map[string]any{"slug": "WT Bad", "label": "x"}, 422, "VALIDATION_FAILED"},
		{http.MethodPost, "/api/v1/admin/website-categories", map[string]any{"slug": "wt-blank", "label": "  "}, 422, "VALIDATION_FAILED"},
		{http.MethodPatch, "/api/v1/admin/website-categories/930000709", map[string]any{"label": "x"}, 404, "NOT_FOUND"},
		{http.MethodDelete, "/api/v1/admin/website-categories/930000709", nil, 404, "NOT_FOUND"},
	} {
		spec := "/admin/website-categories"
		if c.method != http.MethodPost {
			spec = "/admin/website-categories/{website_category_id}"
		}
		resp, got := f.ws(t, c.method, c.url, spec, "sess-staff", nil, c.payload)
		if resp.StatusCode != c.status || got["code"] != c.code {
			t.Errorf("%s %s %v: %d %+v", c.method, c.url, c.payload, resp.StatusCode, got)
		}
	}
	resp, body = f.ws(t, http.MethodPatch, "/api/v1/admin/website-categories/"+id, "/admin/website-categories/{website_category_id}", "sess-staff", nil,
		map[string]any{"label": "Renamed", "description": "d"})
	if resp.StatusCode != http.StatusOK || body["label"] != "Renamed" || body["slug"] != "wt-new" || body["description"] != "d" {
		t.Errorf("patch %d %+v", resp.StatusCode, body)
	}

	resp, body = f.ws(t, http.MethodDelete, fmt.Sprintf("/api/v1/admin/website-categories/%d", wsCatMain), "/admin/website-categories/{website_category_id}", "sess-staff", nil, nil)
	if resp.StatusCode != http.StatusConflict || body["code"] != "WEBSITE_CATEGORY_NOT_EMPTY" || asInt(body["website_count"]) != want {
		t.Errorf("delete a used category %d %+v", resp.StatusCode, body)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_website_category WHERE id = ?`, wsCatMain); n != 1 {
		t.Error("the used category was deleted")
	}
	resp, _ = f.ws(t, http.MethodDelete, "/api/v1/admin/website-categories/"+id, "/admin/website-categories/{website_category_id}", "sess-staff", nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("delete an empty category %d", resp.StatusCode)
	}
}

func TestV1WebsiteTags(t *testing.T) {
	f := newWebsiteFix(t)
	resp, body := f.ws(t, http.MethodGet, "/api/v1/website-tags/wt-a", "/website-tags/{website_tag_slug}", "", nil, nil)
	if resp.StatusCode != http.StatusOK || body["object"] != "website_tag" || asInt(body["level"]) != 10 || strID(body["website_tag_group_id"]) != fmt.Sprint(wsGroupSingle) {
		t.Errorf("by slug %d %+v", resp.StatusCode, body)
	}
	_, body = f.ws(t, http.MethodGet, "/api/v1/website-tags/wt-loose", "/website-tags/{website_tag_slug}", "", nil, nil)
	if body["website_tag_group_id"] != nil {
		t.Errorf("ungrouped tag %+v", body)
	}

	resp, body = f.ws(t, http.MethodPost, "/api/v1/admin/website-tags", "/admin/website-tags", "sess-staff", nil,
		map[string]any{"slug": "wt-t", "label": "T", "level": 7, "website_tag_group_id": "930000719"})
	if resp.StatusCode != http.StatusUnprocessableEntity || errorAt(body, "/website_tag_group_id")["reason"] != "UNKNOWN_REFERENCE" {
		t.Errorf("unknown group %d %+v", resp.StatusCode, body)
	}
	resp, body = f.ws(t, http.MethodPost, "/api/v1/admin/website-tags", "/admin/website-tags", "sess-staff", nil,
		map[string]any{"slug": "wt-t", "label": "T", "level": 7, "website_tag_group_id": fmt.Sprint(wsGroupMulti)})
	if resp.StatusCode != http.StatusCreated || strID(body["website_tag_group_id"]) != fmt.Sprint(wsGroupMulti) {
		t.Fatalf("create %d %+v", resp.StatusCode, body)
	}
	id := strID(body["id"])
	item := "/api/v1/admin/website-tags/" + id
	spec := "/admin/website-tags/{website_tag_id}"
	resp, body = f.ws(t, http.MethodPatch, item, spec, "sess-staff", nil, map[string]any{"level": -3})
	if resp.StatusCode != http.StatusOK || asInt(body["level"]) != -3 || strID(body["website_tag_group_id"]) != fmt.Sprint(wsGroupMulti) {
		t.Errorf("patch keeps the group %d %+v", resp.StatusCode, body)
	}
	resp, body = f.ws(t, http.MethodPatch, item, spec, "sess-staff", nil, map[string]any{"website_tag_group_id": nil})
	if resp.StatusCode != http.StatusOK || body["website_tag_group_id"] != nil {
		t.Errorf("null ungroups %d %+v", resp.StatusCode, body)
	}
	resp, body = f.ws(t, http.MethodPost, "/api/v1/admin/website-tags", "/admin/website-tags", "sess-staff", nil,
		map[string]any{"slug": "wt-a", "label": "dup", "level": 1})
	if resp.StatusCode != http.StatusConflict || body["code"] != "ALREADY_EXISTS" {
		t.Errorf("slug taken %d %+v", resp.StatusCode, body)
	}
	resp, body = f.ws(t, http.MethodGet, item, spec, "sess-staff", nil, nil)
	if resp.StatusCode != http.StatusOK || body["object"] != "admin_website_tag" {
		t.Errorf("admin get %d %+v", resp.StatusCode, body)
	}

	resp, _ = f.ws(t, http.MethodDelete, fmt.Sprintf("/api/v1/admin/website-tags/%d", wsTagA), spec, "sess-staff", nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete %d", resp.StatusCode)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_website_tag_relation WHERE galgame_website_tag_id = ?`, wsTagA); n != 0 {
		t.Error("a deleted tag stayed on its websites")
	}
	resp, body = f.ws(t, http.MethodDelete, fmt.Sprintf("/api/v1/admin/website-tags/%d", wsTagA), spec, "sess-staff", nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("delete again %d %+v", resp.StatusCode, body)
	}
}

func TestV1WebsiteTagGroups(t *testing.T) {
	f := newWebsiteFix(t)
	resp, body := f.ws(t, http.MethodPost, "/api/v1/admin/website-tag-groups", "/admin/website-tag-groups", "sess-staff", nil,
		map[string]any{"slug": "wt-g", "label": "G", "is_multi_select": true})
	if resp.StatusCode != http.StatusCreated || body["is_multi_select"] != true || body["object"] != "admin_website_tag_group" {
		t.Fatalf("create %d %+v", resp.StatusCode, body)
	}
	item := "/api/v1/admin/website-tag-groups/" + strID(body["id"])
	spec := "/admin/website-tag-groups/{website_tag_group_id}"
	resp, body = f.ws(t, http.MethodPatch, item, spec, "sess-staff", nil, map[string]any{"is_multi_select": false, "sort_order": 99})
	if resp.StatusCode != http.StatusOK || body["is_multi_select"] != false || asInt(body["sort_order"]) != 99 {
		t.Errorf("patch %d %+v", resp.StatusCode, body)
	}
	resp, body = f.ws(t, http.MethodPatch, item, spec, "sess-staff", nil, map[string]any{"slug": "wt-single"})
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("slug taken %d %+v", resp.StatusCode, body)
	}

	resp, _ = f.ws(t, http.MethodDelete, fmt.Sprintf("/api/v1/admin/website-tag-groups/%d", wsGroupSingle), spec, "sess-staff", nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete %d", resp.StatusCode)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_website_tag WHERE id IN (?, ?) AND group_id IS NULL`, wsTagA, wsTagB); n != 2 {
		t.Errorf("tags of a deleted group are ungrouped: %d", n)
	}
	resp, body = f.ws(t, http.MethodGet, fmt.Sprintf("/api/v1/admin/website-tag-groups/%d", wsGroupSingle), spec, "sess-staff", nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("get a deleted group %d %+v", resp.StatusCode, body)
	}
}
