package app

import (
	"fmt"
	"net/http"
	"testing"
)

func TestV1WebsitesWalk(t *testing.T) {
	f := newWebsiteFix(t)
	for _, c := range []struct{ query, where string }{
		{"&include_nsfw=true", "WHERE true"},
		{"", "WHERE age_limit = 'all'"},
		{fmt.Sprintf("&include_nsfw=true&website_category_id=%d", wsCatMain), fmt.Sprintf("WHERE category_id = %d", wsCatMain)},
	} {
		want := f.sqlIDs(t, `SELECT id::text FROM galgame_website `+c.where+` ORDER BY created DESC, id DESC`)
		if len(want) < 4 {
			t.Fatalf("%q: seed too thin %v", c.query, want)
		}
		for _, limit := range []int{2, 3} {
			if got := f.walkWebsites(t, c.query, limit); fmt.Sprint(got) != fmt.Sprint(want) {
				t.Errorf("%q limit %d walked %v, want %v", c.query, limit, got, want)
			}
		}
	}
}

func TestV1WebsitesFilterAndTotal(t *testing.T) {
	f := newWebsiteFix(t)
	resp, body := f.ws(t, http.MethodGet, "/api/v1/websites?limit=100&include_total=true", "/websites", "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	for _, it := range body["items"].([]any) {
		if m := it.(map[string]any); m["is_nsfw"] != false {
			t.Errorf("default list carries an NSFW site: %+v", m)
		}
	}
	if want := f.scalar(t, `SELECT COUNT(*) FROM galgame_website WHERE age_limit = 'all'`); asInt(body["total"]) != want {
		t.Errorf("total %v, want %d", body["total"], want)
	}

	q := fmt.Sprintf("/api/v1/websites?limit=1&include_total=true&website_category_id=%d", wsCatMain)
	_, body = f.ws(t, http.MethodGet, q, "/websites", "", nil, nil)
	if want := f.scalar(t, `SELECT COUNT(*) FROM galgame_website WHERE age_limit = 'all' AND category_id = ?`, wsCatMain); asInt(body["total"]) != want {
		t.Errorf("category total %v, want %d", body["total"], want)
	}

	q = fmt.Sprintf("/api/v1/websites?limit=100&include_nsfw=true&website_tag_id=%d", wsTagB)
	_, body = f.ws(t, http.MethodGet, q, "/websites", "", nil, nil)
	if got := adminItemIDs(body); fmt.Sprint(got) != fmt.Sprint([]string{fmt.Sprint(wsSiteTie + 1), fmt.Sprint(wsSiteTie)}) {
		t.Errorf("tag filter %v", got)
	}

	_, first := f.ws(t, http.MethodGet, "/api/v1/websites?limit=1", "/websites", "", nil, nil)
	cur, _ := first["next_cursor"].(string)
	for _, c := range []struct{ url, code string }{
		{"/api/v1/websites?include_nsfw=true&cursor=" + cur, "INVALID_CURSOR"},
		{"/api/v1/websites?limit=101", "LIMIT_TOO_LARGE"},
		{"/api/v1/website-categories?cursor=cur_x", "INVALID_CURSOR"},
		{"/api/v1/website-tags?limit=101", "LIMIT_TOO_LARGE"},
		{"/api/v1/website-tag-groups?cursor=cur_x", "INVALID_CURSOR"},
	} {
		resp, body := f.ws(t, http.MethodGet, c.url, specOf(c.url), "", nil, nil)
		if resp.StatusCode != http.StatusBadRequest || body["code"] != c.code {
			t.Errorf("%s: %d %+v", c.url, resp.StatusCode, body)
		}
	}
}

func specOf(url string) string {
	for _, p := range []string{"/website-categories", "/website-tag-groups", "/website-tags", "/websites"} {
		if len(url) >= len("/api/v1"+p) && url[len("/api/v1"):len("/api/v1")+len(p)] == p {
			return p
		}
	}
	return url
}

func TestV1WebsiteSummaryShape(t *testing.T) {
	f := newWebsiteFix(t)
	_, body := f.ws(t, http.MethodGet, "/api/v1/websites?limit=100&include_nsfw=true", "/websites", "", nil, nil)
	items := map[string]map[string]any{}
	for _, it := range body["items"].([]any) {
		m := it.(map[string]any)
		items[strID(m["id"])] = m
	}
	main := items[fmt.Sprint(wsSiteMain)]
	cat, _ := main["website_category"].(map[string]any)
	if main["host"] != wsHost(wsSiteMain) || main["title"] != fmt.Sprintf("Site %d", wsSiteMain) || asInt(main["score"]) != 8 ||
		main["icon"] != nil || main["external_icon_url"] != "https://favicon.example/f.ico" || cat["slug"] != "wt-main" || main["state"] != "normal" {
		t.Errorf("main summary %+v", main)
	}
	icon := items[fmt.Sprint(wsSiteIcon)]
	img, _ := icon["icon"].(map[string]any)
	if img == nil || img["hash"] != wsIconHash || icon["external_icon_url"] != nil {
		t.Errorf("icon summary %+v", icon)
	}
	if items[fmt.Sprint(wsSiteNSFW)]["is_nsfw"] != true {
		t.Error("NSFW flag")
	}
}

func TestV1WebsiteDetail(t *testing.T) {
	f := newWebsiteFix(t)
	updated := f.sqlIDs(t, `SELECT updated::text FROM galgame_website WHERE id = ?`, wsSiteMain)
	url := "/api/v1/websites/" + wsHost(wsSiteMain)
	resp, body := f.ws(t, http.MethodGet, url, "/websites/{website_host}", "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("detail %d %+v", resp.StatusCode, body)
	}
	tags, _ := body["website_tags"].([]any)
	var order []string
	for _, tg := range tags {
		order = append(order, tg.(map[string]any)["slug"].(string))
	}
	urls, _ := body["urls"].([]any)
	if fmt.Sprint(order) != "[wt-a wt-multi-tag wt-loose]" || body["language"] != "zh-cn" || body["founded"] != "2014-05-01" ||
		len(urls) != 1 || urls[0] != "https://"+wsHost(wsSiteMain) || asInt(body["view_count"]) != 1 || body["viewer"] != nil {
		t.Errorf("detail %+v", body)
	}
	if n := f.scalar(t, `SELECT view FROM galgame_website WHERE id = ?`, wsSiteMain); n != 1 {
		t.Errorf("view %d", n)
	}
	if got := f.sqlIDs(t, `SELECT updated::text FROM galgame_website WHERE id = ?`, wsSiteMain); fmt.Sprint(got) != fmt.Sprint(updated) {
		t.Errorf("a view moved updated: %v -> %v", updated, got)
	}

	_, body = f.ws(t, http.MethodGet, url, "/websites/{website_host}", "sess-staff", nil, nil)
	if v, _ := body["viewer"].(map[string]any); v == nil || v["can_edit"] != true || v["can_delete"] != true || v["has_liked"] != false {
		t.Errorf("staff viewer %+v", body["viewer"])
	}
	bearer := http.Header{"Authorization": {"Bearer staff-token"}}
	_, body = f.ws(t, http.MethodGet, url, "/websites/{website_host}", "", bearer, nil)
	if v, _ := body["viewer"].(map[string]any); v == nil || v["can_edit"] != false || v["can_delete"] != false {
		t.Errorf("Bearer moderator viewer %+v", body["viewer"])
	}
	_, body = f.ws(t, http.MethodGet, "/api/v1/websites/"+wsHost(wsSiteNSFW), "/websites/{website_host}", "", nil, nil)
	if body["is_nsfw"] != true {
		t.Errorf("an NSFW detail is returned with its flag: %+v", body)
	}
	resp, body = f.ws(t, http.MethodGet, "/api/v1/websites/nowhere.example", "/websites/{website_host}", "", nil, nil)
	if resp.StatusCode != http.StatusNotFound || body["code"] != "NOT_FOUND" {
		t.Errorf("unknown host %d %+v", resp.StatusCode, body)
	}
}

func TestV1WebsiteSlots(t *testing.T) {
	f := newWebsiteFix(t)
	updated := f.sqlIDs(t, `SELECT updated::text FROM galgame_website WHERE id = ?`, wsSiteMain)
	for _, slot := range []struct{ path, spec, counter, table, flag string }{
		{"/like", "/websites/{website_host}/like", "like_count", "galgame_website_like", "has_liked"},
		{"/favorite", "/websites/{website_host}/favorite", "favorite_count", "galgame_website_favorite", "has_favorited"},
	} {
		url := "/api/v1/websites/" + wsHost(wsSiteMain) + slot.path
		rows := func() int {
			return f.scalar(t, `SELECT COUNT(*) FROM `+slot.table+` WHERE website_id = ?`, wsSiteMain)
		}
		counter := func() int {
			return f.scalar(t, `SELECT `+slot.counter+` FROM galgame_website WHERE id = ?`, wsSiteMain)
		}
		for i := 0; i < 2; i++ {
			resp, body := f.ws(t, http.MethodPut, url, slot.spec, "sess-alice", nil, nil)
			v, _ := body["viewer"].(map[string]any)
			if resp.StatusCode != http.StatusOK || body["object"] != "website_engagement" || v[slot.flag] != true ||
				asInt(body[slot.counter]) != 1 || counter() != 1 || rows() != 1 {
				t.Errorf("%s PUT #%d: %d %+v", slot.path, i, resp.StatusCode, body)
			}
		}
		_, body := f.ws(t, http.MethodPut, url, slot.spec, "sess-bob", nil, nil)
		if asInt(body[slot.counter]) != 2 || counter() != 2 {
			t.Errorf("%s second user %+v", slot.path, body)
		}
		for i := 0; i < 2; i++ {
			resp, body := f.ws(t, http.MethodDelete, url, slot.spec, "sess-alice", nil, nil)
			v, _ := body["viewer"].(map[string]any)
			if resp.StatusCode != http.StatusOK || v[slot.flag] != false || asInt(body[slot.counter]) != 1 || counter() != 1 || rows() != 1 {
				t.Errorf("%s DELETE #%d: %d %+v", slot.path, i, resp.StatusCode, body)
			}
		}
		resp, body := f.ws(t, http.MethodDelete, url, slot.spec, "sess-other", nil, nil)
		if resp.StatusCode != http.StatusOK || asInt(body[slot.counter]) != 1 || counter() != 1 {
			t.Errorf("%s DELETE of nothing moved the counter: %+v", slot.path, body)
		}
		for _, c := range []struct {
			session string
			url     string
			status  int
			code    string
		}{
			{"", url, 401, "MISSING_CREDENTIAL"},
			{"sess-banned", url, 403, "ACCOUNT_BANNED"},
			{"sess-alice", "/api/v1/websites/nowhere.example" + slot.path, 404, "NOT_FOUND"},
		} {
			resp, body := f.ws(t, http.MethodPut, c.url, slot.spec, c.session, nil, nil)
			if resp.StatusCode != c.status || body["code"] != c.code {
				t.Errorf("%s %q: %d %+v", slot.path, c.session, resp.StatusCode, body)
			}
		}
	}
	if got := f.sqlIDs(t, `SELECT updated::text FROM galgame_website WHERE id = ?`, wsSiteMain); fmt.Sprint(got) != fmt.Sprint(updated) {
		t.Errorf("likes and favorites moved updated: %v -> %v", updated, got)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_website_like WHERE user_id = ?`, w3UserBanned); n != 0 {
		t.Error("a banned user's like was written")
	}
}

func TestV1WebsiteReadsSurviveOAuthOutage(t *testing.T) {
	f := newWebsiteFix(t)
	f.failOA.Store(true)
	resp, _ := f.ws(t, http.MethodGet, "/api/v1/websites/"+wsHost(wsSiteMain), "/websites/{website_host}", "sess-alice", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("detail with OAuth down %d", resp.StatusCode)
	}
	resp, body := f.ws(t, http.MethodPut, "/api/v1/websites/"+wsHost(wsSiteMain)+"/like", "/websites/{website_host}/like", "sess-alice", nil, nil)
	if resp.StatusCode != http.StatusServiceUnavailable || body["code"] != "SERVICE_UNAVAILABLE" {
		t.Errorf("like with OAuth down %d %+v", resp.StatusCode, body)
	}
}
