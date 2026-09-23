package app

import (
	"fmt"
	"net/http"
	"testing"
)

func TestV1AdminWebsiteCreate(t *testing.T) {
	f := newWebsiteFix(t)
	body := websiteBody("WWW.Example-Site.COM", "New Site", map[string]any{
		"website_tag_ids": []string{fmt.Sprint(wsTagA), fmt.Sprint(wsTagMulti)},
		"urls":            []string{"https://www.example-site.com", "http://mirror.example"},
		"founded":         "  2020  ",
		"language":        "JA-JP",
	})
	resp, got := f.ws(t, http.MethodPost, "/api/v1/admin/websites", "/admin/websites", "sess-staff", nil, body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, got)
	}
	id := strID(got["id"])
	if resp.Header.Get("Location") != "/api/v1/admin/websites/"+id || got["object"] != "admin_website" || got["host"] != "www.example-site.com" ||
		got["founded"] != "2020" || got["language"] != "ja-jp" || got["state"] != "normal" {
		t.Errorf("created %s %+v", resp.Header.Get("Location"), got)
	}
	resp, detail := f.ws(t, http.MethodGet, "/api/v1/websites/www.example-site.com", "/websites/{website_host}", "", nil, nil)
	if resp.StatusCode != http.StatusOK || asInt(detail["score"]) != 13 {
		t.Errorf("lower-cased host reads back %d %+v", resp.StatusCode, detail)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM feed_activity WHERE type = 'GALGAME_WEBSITE_CREATION' AND source_id = ?`, asInt(id)); n != 1 {
		t.Errorf("feed card %d", n)
	}

	for _, c := range []struct {
		name    string
		body    map[string]any
		status  int
		code    string
		pointer string
		reason  string
	}{
		{"host taken", websiteBody(wsHost(wsSiteMain), "Other", nil), 409, "ALREADY_EXISTS", "/host", "NOT_ALLOWED_VALUE"},
		{"title taken", websiteBody("fresh.example", fmt.Sprintf("Site %d", wsSiteMain), nil), 409, "ALREADY_EXISTS", "/title", "NOT_ALLOWED_VALUE"},
		{"unknown category", websiteBody("fresh.example", "Fresh", map[string]any{"website_category_id": "930000709"}), 422, "VALIDATION_FAILED", "/website_category_id", "UNKNOWN_REFERENCE"},
		{"unknown tag", websiteBody("fresh.example", "Fresh", map[string]any{"website_tag_ids": []string{"930000729"}}), 422, "VALIDATION_FAILED", "/website_tag_ids/0", "UNKNOWN_REFERENCE"},
		{"duplicate tag", websiteBody("fresh.example", "Fresh", map[string]any{"website_tag_ids": []string{fmt.Sprint(wsTagLoose), fmt.Sprint(wsTagLoose)}}), 422, "VALIDATION_FAILED", "/website_tag_ids/1", "DUPLICATE_ITEM"},
		{"two tags of a single-select group", websiteBody("fresh.example", "Fresh", map[string]any{"website_tag_ids": []string{fmt.Sprint(wsTagA), fmt.Sprint(wsTagB)}}), 422, "VALIDATION_FAILED", "/website_tag_ids/1", "INCONSISTENT_WITH"},
		{"url without scheme", websiteBody("fresh.example", "Fresh", map[string]any{"urls": []string{"fresh.example"}}), 422, "VALIDATION_FAILED", "/urls/0", ""},
		{"blank title", websiteBody("fresh.example", "   ", nil), 422, "VALIDATION_FAILED", "/title", "TOO_SHORT"},
		{"short description once trimmed", websiteBody("fresh.example", "Fresh", map[string]any{"description": "   short    "}), 422, "VALIDATION_FAILED", "/description", "TOO_SHORT"},
		{"host with a path", websiteBody("fresh.example/x", "Fresh", nil), 422, "VALIDATION_FAILED", "/host", ""},
	} {
		resp, got := f.ws(t, http.MethodPost, "/api/v1/admin/websites", "/admin/websites", "sess-staff", nil, c.body)
		if resp.StatusCode != c.status || got["code"] != c.code {
			t.Errorf("%s: %d %+v", c.name, resp.StatusCode, got)
			continue
		}
		e := errorAt(got, c.pointer)
		if e == nil || (c.reason != "" && e["reason"] != c.reason) {
			t.Errorf("%s: errors %+v", c.name, got["errors"])
		}
	}
	resp, got = f.ws(t, http.MethodPost, "/api/v1/admin/websites", "/admin/websites", "sess-staff", nil,
		websiteBody("multi.example", "Multi", map[string]any{"website_tag_ids": []string{fmt.Sprint(wsTagMulti), fmt.Sprint(wsTagLoose)}}))
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("multi-select and ungrouped tags together %d %+v", resp.StatusCode, got)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_website WHERE url = 'fresh.example'`); n != 0 {
		t.Error("a refused create wrote a row")
	}
}

func TestV1AdminWebsitePatch(t *testing.T) {
	f := newWebsiteFix(t)
	url := fmt.Sprintf("/api/v1/admin/websites/%d", wsSiteMain)
	resp, got := f.ws(t, http.MethodGet, url, "/admin/websites/{website_id}", "sess-staff", nil, nil)
	if resp.StatusCode != http.StatusOK || fmt.Sprint(got["website_tag_ids"]) != fmt.Sprint([]any{fmt.Sprint(wsTagA), fmt.Sprint(wsTagMulti), fmt.Sprint(wsTagLoose)}) {
		t.Fatalf("edit source %d %+v", resp.StatusCode, got)
	}
	resp, got = f.ws(t, http.MethodPatch, url, "/admin/websites/{website_id}", "sess-staff", nil, map[string]any{"title": "Renamed", "is_nsfw": true})
	if resp.StatusCode != http.StatusOK || got["title"] != "Renamed" || got["is_nsfw"] != true || len(got["website_tag_ids"].([]any)) != 3 {
		t.Errorf("patch keeps the tags %d %+v", resp.StatusCode, got)
	}
	resp, got = f.ws(t, http.MethodPatch, url, "/admin/websites/{website_id}", "sess-staff", nil,
		map[string]any{"website_tag_ids": []string{}, "urls": []string{}, "host": "Renamed.Example", "icon_image_hash": wsIconHash})
	img, _ := got["icon"].(map[string]any)
	if resp.StatusCode != http.StatusOK || len(got["website_tag_ids"].([]any)) != 0 || len(got["urls"].([]any)) != 0 ||
		got["host"] != "renamed.example" || img == nil || got["external_icon_url"] != nil {
		t.Errorf("patch replaces tags and urls %d %+v", resp.StatusCode, got)
	}
	resp, got = f.ws(t, http.MethodPatch, url, "/admin/websites/{website_id}", "sess-staff", nil, map[string]any{})
	if resp.StatusCode != http.StatusOK || got["host"] != "renamed.example" {
		t.Errorf("empty patch %d %+v", resp.StatusCode, got)
	}
	resp, got = f.ws(t, http.MethodPatch, url, "/admin/websites/{website_id}", "sess-staff", nil, map[string]any{"host": wsHost(wsSiteIcon)})
	if resp.StatusCode != http.StatusConflict || got["code"] != "ALREADY_EXISTS" {
		t.Errorf("rename onto a taken host %d %+v", resp.StatusCode, got)
	}
	resp, got = f.ws(t, http.MethodPatch, "/api/v1/admin/websites/930000898", "/admin/websites/{website_id}", "sess-staff", nil, map[string]any{"title": "x"})
	if resp.StatusCode != http.StatusNotFound || got["code"] != "NOT_FOUND" {
		t.Errorf("patch a missing site %d %+v", resp.StatusCode, got)
	}
	resp, got = f.ws(t, http.MethodGet, "/api/v1/admin/websites/930000898", "/admin/websites/{website_id}", "sess-staff", nil, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("get a missing site %d %+v", resp.StatusCode, got)
	}
}

func TestV1AdminWebsiteDelete(t *testing.T) {
	f := newWebsiteFix(t)
	f.ws(t, http.MethodPut, "/api/v1/websites/"+wsHost(wsSiteMain)+"/like", "/websites/{website_host}/like", "sess-alice", nil, nil)
	url := fmt.Sprintf("/api/v1/admin/websites/%d", wsSiteMain)
	resp, _ := f.ws(t, http.MethodDelete, url, "/admin/websites/{website_id}", "sess-staff", nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete %d", resp.StatusCode)
	}
	for _, q := range []string{
		`SELECT COUNT(*) FROM galgame_website WHERE id = ?`,
		`SELECT COUNT(*) FROM galgame_website_like WHERE website_id = ?`,
		`SELECT COUNT(*) FROM galgame_website_tag_relation WHERE galgame_website_id = ?`,
		`SELECT COUNT(*) FROM feed_activity WHERE type = 'GALGAME_WEBSITE_CREATION' AND source_id = ?`,
	} {
		if n := f.scalar(t, q, wsSiteMain); n != 0 {
			t.Errorf("%s left %d", q, n)
		}
	}
	resp, got := f.ws(t, http.MethodDelete, url, "/admin/websites/{website_id}", "sess-staff", nil, nil)
	if resp.StatusCode != http.StatusNotFound || got["code"] != "NOT_FOUND" {
		t.Errorf("delete again %d %+v", resp.StatusCode, got)
	}
}

func TestV1WebsiteAdminNeedsPermissions(t *testing.T) {
	f := newWebsiteFix(t)
	bearer := http.Header{"Authorization": {"Bearer staff-token"}}
	site := fmt.Sprintf("/api/v1/admin/websites/%d", wsSiteMain)
	cat := fmt.Sprintf("/api/v1/admin/website-categories/%d", wsCatEmpty)
	tag := fmt.Sprintf("/api/v1/admin/website-tags/%d", wsTagLoose)
	group := fmt.Sprintf("/api/v1/admin/website-tag-groups/%d", wsGroupMulti)
	ops := []struct {
		method, url, spec string
		body              any
	}{
		{http.MethodPost, "/api/v1/admin/websites", "/admin/websites", websiteBody("fresh.example", "Fresh", nil)},
		{http.MethodGet, site, "/admin/websites/{website_id}", nil},
		{http.MethodPatch, site, "/admin/websites/{website_id}", map[string]any{"title": "x"}},
		{http.MethodDelete, site, "/admin/websites/{website_id}", nil},
		{http.MethodPost, "/api/v1/admin/website-categories", "/admin/website-categories", map[string]any{"slug": "wt-new", "label": "n"}},
		{http.MethodGet, cat, "/admin/website-categories/{website_category_id}", nil},
		{http.MethodPatch, cat, "/admin/website-categories/{website_category_id}", map[string]any{"label": "x"}},
		{http.MethodDelete, cat, "/admin/website-categories/{website_category_id}", nil},
		{http.MethodPost, "/api/v1/admin/website-tags", "/admin/website-tags", map[string]any{"slug": "wt-new", "label": "n", "level": 1}},
		{http.MethodPatch, tag, "/admin/website-tags/{website_tag_id}", map[string]any{"label": "x"}},
		{http.MethodDelete, tag, "/admin/website-tags/{website_tag_id}", nil},
		{http.MethodPost, "/api/v1/admin/website-tag-groups", "/admin/website-tag-groups", map[string]any{"slug": "wt-new", "label": "n"}},
		{http.MethodPatch, group, "/admin/website-tag-groups/{website_tag_group_id}", map[string]any{"label": "x"}},
		{http.MethodDelete, group, "/admin/website-tag-groups/{website_tag_group_id}", nil},
	}
	for _, op := range ops {
		for _, who := range []struct {
			session string
			hdr     http.Header
			status  int
			code    string
		}{
			{"sess-alice", nil, 403, "PERMISSION_REQUIRED"},
			{"", bearer, 403, "PERMISSION_REQUIRED"},
			{"", nil, 401, "MISSING_CREDENTIAL"},
			{"sess-banned", nil, 403, "ACCOUNT_BANNED"},
		} {
			resp, got := f.ws(t, op.method, op.url, op.spec, who.session, who.hdr, op.body)
			if resp.StatusCode != who.status || got["code"] != who.code {
				t.Errorf("%s %s as %q: %d %+v", op.method, op.url, who.session, resp.StatusCode, got)
			}
		}
	}
	for _, q := range []string{
		`SELECT COUNT(*) FROM galgame_website WHERE id = 930000801 AND name = 'Site 930000801'`,
		`SELECT COUNT(*) FROM galgame_website_category WHERE id = 930000704`,
		`SELECT COUNT(*) FROM galgame_website_tag WHERE id = 930000724 AND label = 'label wt-loose'`,
		`SELECT COUNT(*) FROM galgame_website_tag_group WHERE id = 930000712`,
	} {
		if n := f.scalar(t, q); n != 1 {
			t.Errorf("a refused write changed state: %s", q)
		}
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_website_category WHERE name = 'wt-new'`); n != 0 {
		t.Error("a refused create wrote")
	}
}

func TestV1AdminWebsiteWritesFailClosedWhenOAuthIsDown(t *testing.T) {
	f := newWebsiteFix(t)
	f.failOA.Store(true)
	resp, got := f.ws(t, http.MethodPatch, fmt.Sprintf("/api/v1/admin/websites/%d", wsSiteMain), "/admin/websites/{website_id}", "sess-staff", nil, map[string]any{"title": "x"})
	if resp.StatusCode != http.StatusServiceUnavailable || got["code"] != "SERVICE_UNAVAILABLE" {
		t.Errorf("OAuth down %d %+v", resp.StatusCode, got)
	}
}
