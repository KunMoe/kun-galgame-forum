package app

import (
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func geWant(t *testing.T, label string, got []string, want ...string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("%s: got %v, want %v", label, got, want)
	}
}

func geStatus(t *testing.T, resp *http.Response, body map[string]any, status int, code string) {
	t.Helper()
	if resp.StatusCode != status {
		t.Fatalf("status %d, want %d: %+v", resp.StatusCode, status, body)
	}
	if code != "" && body["code"] != code {
		t.Fatalf("code %v, want %s: %+v", body["code"], code, body)
	}
}

// geWalk pages a page-number collection to the end at the given limit.
func (f *geFix) geWalk(t *testing.T, path, specPath string, q url.Values, limit int) []string {
	t.Helper()
	var got []string
	total := -1
	for page := 1; page <= 50; page++ {
		q.Set("page", strconv.Itoa(page))
		q.Set("limit", strconv.Itoa(limit))
		resp, body := f.get(t, path+"?"+q.Encode(), specPath)
		geStatus(t, resp, body, http.StatusOK, "")
		if body["total_relation"] != "eq" {
			t.Fatalf("%s: total_relation %v", path, body["total_relation"])
		}
		if n := int(body["total"].(float64)); total == -1 {
			total = n
		} else if n != total {
			t.Fatalf("%s: total moved from %d to %d between pages", path, total, n)
		}
		ids := geItemIDs(body)
		got = append(got, ids...)
		if page*limit >= total {
			break
		}
	}
	if len(got) != total {
		t.Fatalf("%s %v: walked %d items, total says %d: %v", path, q, len(got), total, got)
	}
	return got
}

func TestV1EntityTagList(t *testing.T) {
	f := newGEFix(t)
	geWant(t, "sfw browse", f.geWalk(t, "/api/v1/tags", "/tags", url.Values{}, 2), "5101", "5105", "5106")
	geWant(t, "nsfw browse", f.geWalk(t, "/api/v1/tags", "/tags", url.Values{"include_nsfw": {"true"}}, 1),
		"5101", "5102", "5105", "5106")

	_, body := f.get(t, "/api/v1/tags?limit=1", "/tags")
	first := body["items"].([]any)[0].(map[string]any)
	if first["display_name"] != "Plot twist" || first["tag_kind"] != "content" || first["catalog_work_count"] != float64(40) {
		t.Fatalf("tag summary %+v", first)
	}
	if loc := first["localized"].(map[string]any)["zh-Hans"].(map[string]any); loc["value"] != "剧情反转" {
		t.Fatalf("localized %+v", first["localized"])
	}

	_, body = f.get(t, "/api/v1/tags?q=x", "/tags")
	geWant(t, "sfw search", geItemIDs(body), "5104", "5101")
	if body["total"] != float64(2) {
		t.Fatalf("search total %v", body["total"])
	}
	_, body = f.get(t, "/api/v1/tags?q=x&include_nsfw=true", "/tags")
	geWant(t, "nsfw search", geItemIDs(body), "5102", "5104", "5107", "5101")

	resp, body := f.get(t, "/api/v1/tags?q=x&page=2&limit=100", "/tags")
	geStatus(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
	resp, body = f.get(t, "/api/v1/tags?page=101&limit=100", "/tags")
	geStatus(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
}

func TestV1EntityTagDetail(t *testing.T) {
	f := newGEFix(t)
	resp, body := f.get(t, "/api/v1/tags/5101", "/tags/{tag_id}")
	geStatus(t, resp, body, http.StatusOK, "")
	intros := body["intros"].([]any)
	if len(intros) != 1 || intros[0].(map[string]any)["data_source"] != "vndb" || intros[0].(map[string]any)["locale"] != "zh-Hans" {
		t.Fatalf("intros %+v", intros)
	}
	if body["is_hidden"] != false || body["is_sexual"] != false {
		t.Fatalf("flags %+v", body)
	}

	resp, body = f.get(t, "/api/v1/tags/5102", "/tags/{tag_id}")
	geStatus(t, resp, body, http.StatusNotFound, "NOT_FOUND")
	resp, body = f.get(t, "/api/v1/tags/5102/works", "/tags/{tag_id}/works")
	geStatus(t, resp, body, http.StatusNotFound, "NOT_FOUND")
	resp, body = f.get(t, "/api/v1/tags/5102?include_nsfw=true", "/tags/{tag_id}")
	geStatus(t, resp, body, http.StatusOK, "")
	if body["is_sexual"] != true {
		t.Fatalf("adult tag %+v", body)
	}

	_, body = f.get(t, "/api/v1/tags/5103", "/tags/{tag_id}")
	if body["is_hidden"] != true {
		t.Fatalf("hidden tag %+v", body)
	}
	resp, body = f.get(t, "/api/v1/tags/5999", "/tags/{tag_id}")
	geStatus(t, resp, body, http.StatusNotFound, "NOT_FOUND")
}

func TestV1EntityTagWorksEntityLane(t *testing.T) {
	f := newGEFix(t)
	path, spec := "/api/v1/tags/5101/works", "/tags/{tag_id}/works"
	for _, limit := range []int{1, 2, 3, 24} {
		geWant(t, "default sort", f.geWalk(t, path, spec, url.Values{}, limit), wid(5), wid(3), wid(1), wid(0), wid(4))
		geWant(t, "view ties keep catalog order", f.geWalk(t, path, spec, url.Values{"sort": {"view_desc"}}, limit),
			wid(0), wid(1), wid(3), wid(5), wid(4))
		geWant(t, "nsfw view ties", f.geWalk(t, path, spec, url.Values{"sort": {"view_desc"}, "include_nsfw": {"true"}}, limit),
			wid(0), wid(1), wid(2), wid(3), wid(5), wid(4))
		geWant(t, "release pushed into the walk", f.geWalk(t, path, spec, url.Values{"sort": {"release_date_desc"}}, limit),
			wid(0), wid(1), wid(3), wid(5), wid(4))
	}
}

func TestV1EntityTagWorksLocalLane(t *testing.T) {
	f := newGEFix(t)
	path, spec := "/api/v1/tags/5101/works", "/tags/{tag_id}/works"
	for _, limit := range []int{1, 2, 24} {
		geWant(t, "platform axis", f.geWalk(t, path, spec, url.Values{"resource_platform": {"win"}}, limit), wid(5), wid(1), wid(0))
		geWant(t, "id breaks view ties", f.geWalk(t, path, spec, url.Values{"resource_language": {"zh-cn"}, "sort": {"view_desc"}}, limit),
			wid(1), wid(0), wid(5))
	}
	geWant(t, "axis, not the legacy scalar", f.geWalk(t, path, spec, url.Values{"resource_platform": {"mac"}}, 24), wid(3))
	geWant(t, "resource type", f.geWalk(t, path, spec, url.Values{"resource_type": {"game"}}, 24), wid(5), wid(3), wid(1), wid(0))
	geWant(t, "game type", f.geWalk(t, path, spec, url.Values{"game_type": {"plot"}}, 24), wid(0))
	geWant(t, "uncategorized", f.geWalk(t, path, spec, url.Values{"game_type": {"uncategorized"}}, 24), wid(3), wid(1))

	resp, body := f.get(t, path+"?resource_platform=zzz", spec)
	geStatus(t, resp, body, http.StatusBadRequest, "UNKNOWN_ENUM_VALUE")
	resp, body = f.get(t, path+"?sort=hot", spec)
	geStatus(t, resp, body, http.StatusBadRequest, "UNKNOWN_SORT")
}

func geWorkByID(t *testing.T, body map[string]any, id string) map[string]any {
	t.Helper()
	for _, raw := range body["items"].([]any) {
		if w := raw.(map[string]any); w["id"] == id {
			return w
		}
	}
	t.Fatalf("work %s not in %v", id, geItemIDs(body))
	return nil
}

func TestV1EntityWorkSummary(t *testing.T) {
	f := newGEFix(t)
	_, body := f.get(t, "/api/v1/tags/5101/works?include_nsfw=true", "/tags/{tag_id}/works")

	a := geWorkByID(t, body, wid(0))
	if a["object"] != "work" || a["display_name"] != "Alpha" || a["latin"] != "alpha" || a["is_published"] != true {
		t.Fatalf("alpha %+v", a)
	}
	if loc := a["localized"].(map[string]any)["zh-Hans"].(map[string]any); loc["value"] != "Alpha（中）" || loc["is_machine"] != true {
		t.Fatalf("localized %+v", a["localized"])
	}
	cover := a["cover"].(map[string]any)
	if cover["hash"] != geHash(geWorkMin) || !strings.HasPrefix(cover["url"].(string), geCDN) || cover["width"] != float64(256) || cover["sexual"] != nil {
		t.Fatalf("cover %+v", cover)
	}
	if banner := a["banner"].(map[string]any); banner["hash"] != geHash(geWorkMin+1000) || banner["width"] != float64(800) {
		t.Fatalf("banner %+v", banner)
	}
	if maker := a["maker"].(map[string]any); maker["id"] != strconv.Itoa(geCompany) || maker["object"] != "company" {
		t.Fatalf("maker %+v", maker)
	}
	if a["release_date"] != "2026-01-01" || a["release_date_precision"] != "day" {
		t.Fatalf("release %v %v", a["release_date"], a["release_date_precision"])
	}
	if a["view_count"] != float64(7) || a["like_count"] != float64(14) || a["rating_count"] != float64(2) || a["rating_score"] == nil {
		t.Fatalf("counts %+v", a)
	}
	geWant(t, "platforms", anyStrings(a["resource_platforms"]), "win", "and")
	if a["resource_updated_at"] != "2026-03-02T08:00:00Z" {
		t.Fatalf("resource_updated_at %v", a["resource_updated_at"])
	}

	b := geWorkByID(t, body, wid(1))
	if b["is_nsfw"] != false || b["rating_score"] != nil || b["rating_count"] != float64(0) {
		t.Fatalf("beta: claim says sfw, no ratings: %+v", b)
	}
	geWant(t, "languages in vocabulary order", anyStrings(b["resource_languages"]), "zh-cn", "ja-jp")

	if g := geWorkByID(t, body, wid(2)); g["is_nsfw"] != true || g["release_date"] != "2025-06-01" || g["release_date_precision"] != "month" {
		t.Fatalf("gamma %+v", g)
	}
	if d := geWorkByID(t, body, wid(3)); d["release_date"] != "2024-01-01" || d["release_date_precision"] != "year" {
		t.Fatalf("delta %+v", d)
	}
	e := geWorkByID(t, body, wid(4))
	if e["is_published"] != false || e["view_count"] != float64(0) || e["resource_updated_at"] != nil ||
		e["release_date"] != nil || e["release_date_precision"] != nil || len(e["resource_platforms"].([]any)) != 0 {
		t.Fatalf("catalog-only epsilon %+v", e)
	}
}

func anyStrings(v any) []string {
	out := []string{}
	for _, x := range v.([]any) {
		out = append(out, x.(string))
	}
	return out
}

func TestV1EntityTaggedWorks(t *testing.T) {
	f := newGEFix(t)
	geWant(t, "one tag", f.geWalk(t, "/api/v1/tagged-works", "/tagged-works", url.Values{"tag_ids": {"5101,5101"}}, 2),
		wid(0), wid(1), wid(3), wid(5), wid(4))
	_, body := f.get(t, "/api/v1/tagged-works?tag_ids=5101,5102", "/tagged-works")
	geWant(t, "every tag", geItemIDs(body), wid(1))

	ids := []string{}
	for i := range 11 {
		ids = append(ids, strconv.Itoa(5200+i))
	}
	resp, body := f.get(t, "/api/v1/tagged-works?tag_ids="+strings.Join(ids, ","), "/tagged-works")
	geStatus(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
	if errs := body["errors"].([]any); errs[0].(map[string]any)["reason"] != "TOO_MANY_ITEMS" {
		t.Fatalf("errors %+v", errs)
	}
}

func TestV1EntityCompanies(t *testing.T) {
	f := newGEFix(t)
	geWant(t, "browse", f.geWalk(t, "/api/v1/companies", "/companies", url.Values{}, 2), "6104", "6101", "6102")
	geWant(t, "kind", f.geWalk(t, "/api/v1/companies", "/companies", url.Values{"company_kind": {"game_brand"}}, 1), "6101", "6102")
	_, body := f.get(t, "/api/v1/companies?q=x", "/companies")
	geWant(t, "search", geItemIDs(body), "6106", "6101")
	_, body = f.get(t, "/api/v1/companies?q=x&company_kind=doujin_circle", "/companies")
	geWant(t, "search by kind", geItemIDs(body), "6106")
	resp, body := f.get(t, "/api/v1/companies?company_kind=conglomerate", "/companies")
	geStatus(t, resp, body, http.StatusBadRequest, "UNKNOWN_ENUM_VALUE")

	resp, body = f.get(t, "/api/v1/companies/6101", "/companies/{company_id}")
	geStatus(t, resp, body, http.StatusOK, "")
	links := body["links"].([]any)
	if len(links) != 1 || links[0].(map[string]any)["site"] != "official_site" {
		t.Fatalf("links %+v", links)
	}
	geWant(t, "aliases", anyStrings(body["aliases"]), "MB")
	if body["lang"] != "ja" || body["logo"].(map[string]any)["hash"] != geHash(61) {
		t.Fatalf("company %+v", body)
	}

	resp, body = f.get(t, "/api/v1/companies/6103", "/companies/{company_id}")
	geStatus(t, resp, body, http.StatusNotFound, "ENTITY_MERGED")
	if body["object"] != "company" || body["current_id"] != "6101" {
		t.Fatalf("merged %+v", body)
	}
	resp, body = f.get(t, "/api/v1/companies/6103/works", "/companies/{company_id}/works")
	geStatus(t, resp, body, http.StatusNotFound, "ENTITY_MERGED")
}

func TestV1EntityCompanyWorks(t *testing.T) {
	f := newGEFix(t)
	path, spec := "/api/v1/companies/6101/works", "/companies/{company_id}/works"
	geWant(t, "all", f.geWalk(t, path, spec, url.Values{}, 1), wid(5), wid(3), wid(1), wid(0))
	geWant(t, "own", f.geWalk(t, path, spec, url.Values{"via": {"own"}}, 1), wid(3), wid(0))
	geWant(t, "imprint", f.geWalk(t, path, spec, url.Values{"via": {"imprint"}}, 1), wid(5), wid(1))

	_, body := f.get(t, path, spec)
	for _, raw := range body["items"].([]any) {
		item := raw.(map[string]any)
		id := item["work_summary"].(map[string]any)["id"]
		via, _ := item["via_company"].(map[string]any)
		switch id {
		case wid(1), wid(5):
			if via == nil || via["id"] != strconv.Itoa(geImprint) || via["display_name"] != "Imprint Label" {
				t.Fatalf("%v via %+v", id, item)
			}
		default:
			if via != nil {
				t.Fatalf("%v is the company's own: %+v", id, item)
			}
		}
	}
}

func TestV1EntityCompanyGraphAndWiki(t *testing.T) {
	f := newGEFix(t)
	resp, body := f.get(t, "/api/v1/companies/6101/graph", "/companies/{company_id}/graph")
	geStatus(t, resp, body, http.StatusOK, "")
	if len(body["nodes"].([]any)) != 2 || len(body["edges"].([]any)) != 2 {
		t.Fatalf("graph %+v", body)
	}
	if n := body["nodes"].([]any)[1].(map[string]any); n["display_name"] != "Imprint Label" {
		t.Fatalf("node read from the legacy name %+v", n)
	}
	resp, body = f.get(t, "/api/v1/wiki-company-redirects/42", "/wiki-company-redirects/{wiki_company_id}")
	geStatus(t, resp, body, http.StatusOK, "")
	if body["company_id"] != "6101" || body["wiki_company_id"] != "42" {
		t.Fatalf("redirect %+v", body)
	}
	resp, body = f.get(t, "/api/v1/wiki-company-redirects/43", "/wiki-company-redirects/{wiki_company_id}")
	geStatus(t, resp, body, http.StatusNotFound, "NOT_FOUND")
}

func TestV1EntityEngines(t *testing.T) {
	f := newGEFix(t)
	geWant(t, "browse", f.geWalk(t, "/api/v1/engines", "/engines", url.Values{}, 1), "7101", "7102")
	geWant(t, "alias search", f.geWalk(t, "/api/v1/engines", "/engines", url.Values{"q": {"KRKR"}}, 5), "7101")
	resp, body := f.get(t, "/api/v1/engines/7101", "/engines/{engine_id}")
	geStatus(t, resp, body, http.StatusOK, "")
	if body["description"] != "A scripting engine." {
		t.Fatalf("engine %+v", body)
	}
	geWant(t, "works", f.geWalk(t, "/api/v1/engines/7101/works", "/engines/{engine_id}/works", url.Values{}, 1), wid(1), wid(0))
}

func TestV1EntitySeries(t *testing.T) {
	f := newGEFix(t)
	geWant(t, "sfw browse", f.geWalk(t, "/api/v1/series", "/series", url.Values{}, 1), "8101")
	geWant(t, "nsfw browse", f.geWalk(t, "/api/v1/series", "/series", url.Values{"include_nsfw": {"true"}}, 1), "8101", "8102")
	geWant(t, "sfw search", f.geWalk(t, "/api/v1/series", "/series", url.Values{"q": {"saga"}}, 1), "8101")
	geWant(t, "nsfw search", f.geWalk(t, "/api/v1/series", "/series", url.Values{"q": {"saga"}, "include_nsfw": {"true"}}, 2),
		"8101", "8102", "8103")

	resp, body := f.get(t, "/api/v1/series/8101", "/series/{series_id}")
	geStatus(t, resp, body, http.StatusOK, "")
	samples := body["sample_works"].([]any)
	if body["listed_work_count"] != float64(2) || len(samples) != 2 || samples[0].(map[string]any)["banner"] == nil {
		t.Fatalf("series %+v", body)
	}
	geWant(t, "works", f.geWalk(t, "/api/v1/series/8101/works", "/series/{series_id}/works", url.Values{}, 1), wid(1), wid(0))
}

func (f *geFix) geStream(t *testing.T, path, specPath string, q url.Values) []string {
	t.Helper()
	var got []string
	for range 20 {
		resp, body := f.get(t, path+"?"+q.Encode(), specPath)
		geStatus(t, resp, body, http.StatusOK, "")
		got = append(got, geItemIDs(body)...)
		next, _ := body["next_cursor"].(string)
		if next == "" {
			return got
		}
		q.Set("cursor", next)
	}
	t.Fatalf("%s did not end", path)
	return nil
}

func TestV1EntityCreditNames(t *testing.T) {
	f := newGEFix(t)
	resp, body := f.get(t, "/api/v1/credit-names", "/credit-names")
	geStatus(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
	resp, body = f.get(t, "/api/v1/credit-names?q=%20%20", "/credit-names")
	geStatus(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
	_, body = f.get(t, "/api/v1/credit-names?q=seto", "/credit-names")
	geWant(t, "search", geItemIDs(body), "9101")

	resp, body = f.get(t, "/api/v1/credit-names/9101", "/credit-names/{credit_name_id}")
	geStatus(t, resp, body, http.StatusOK, "")
	if body["gender"] != "female" || body["birth_year"] != nil || body["birth_month"] != float64(4) || len(body["siblings"].([]any)) != 1 {
		t.Fatalf("credit name %+v", body)
	}
	if links := body["links"].([]any); len(links) != 1 || links[0].(map[string]any)["url"] != "https://vndb.org/s2099" {
		t.Fatalf("links %+v", links)
	}
	resp, body = f.get(t, "/api/v1/credit-names/9102", "/credit-names/{credit_name_id}")
	geStatus(t, resp, body, http.StatusNotFound, "ENTITY_MERGED")
	if body["object"] != "credit_name" || body["current_id"] != "9101" {
		t.Fatalf("merged %+v", body)
	}

	path, spec := "/api/v1/credit-names/9101/credits", "/credit-names/{credit_name_id}/credits"
	geWant(t, "sfw credits", f.geStream(t, path, spec, url.Values{"limit": {"2"}}), wid(0), wid(1), wid(3))
	geWant(t, "nsfw credits", f.geStream(t, path, spec, url.Values{"limit": {"2"}, "include_nsfw": {"true"}}), wid(0), wid(1), wid(2), wid(3))

	_, body = f.get(t, path+"?limit=2", spec)
	items := body["items"].([]any)
	roles := items[0].(map[string]any)["credit_roles"].([]any)
	if len(roles) != 1 || roles[0].(map[string]any)["role_key"] != "scenario" {
		t.Fatalf("other-staff beside a real role must go: %+v", roles)
	}
	chars := items[1].(map[string]any)["characters"].([]any)
	if len(chars) != 2 || chars[0].(map[string]any)["character_id"] != "9201" || chars[1].(map[string]any)["character_id"] != nil {
		t.Fatalf("characters %+v", chars)
	}
	next := body["next_cursor"].(string)
	resp, body = f.get(t, path+"?limit=2&include_nsfw=true&cursor="+url.QueryEscape(next), spec)
	geStatus(t, resp, body, http.StatusBadRequest, "INVALID_CURSOR")
}

func TestV1EntityCharacters(t *testing.T) {
	f := newGEFix(t)
	_, body := f.get(t, "/api/v1/characters?q=kaho", "/characters")
	geWant(t, "search", geItemIDs(body), "9201")

	resp, body := f.get(t, "/api/v1/characters/9201", "/characters/{character_id}")
	geStatus(t, resp, body, http.StatusOK, "")
	traits := body["traits"].([]any)
	if len(traits) != 2 {
		t.Fatalf("adult trait must be left out: %+v", traits)
	}
	blonde := traits[0].(map[string]any)
	if blonde["localized"].(map[string]any)["zh-Hans"].(map[string]any)["value"] != "金发" ||
		blonde["trait_group"].(map[string]any)["localized"].(map[string]any)["zh-Hans"].(map[string]any)["value"] != "毛发" {
		t.Fatalf("trait names %+v", blonde)
	}
	if liar := traits[1].(map[string]any); liar["spoiler"] != "minor" || liar["is_lie"] != true {
		t.Fatalf("liar %+v", liar)
	}
	if body["image"].(map[string]any)["hash"] != geHash(9201) || body["figure"] != nil {
		t.Fatalf("images %+v", body)
	}
	_, body = f.get(t, "/api/v1/characters/9201?include_nsfw=true", "/characters/{character_id}")
	if len(body["traits"].([]any)) != 3 {
		t.Fatalf("include_nsfw traits %+v", body["traits"])
	}
	resp, body = f.get(t, "/api/v1/characters/9202", "/characters/{character_id}")
	geStatus(t, resp, body, http.StatusNotFound, "ENTITY_MERGED")

	path, spec := "/api/v1/characters/9201/appearances", "/characters/{character_id}/appearances"
	geWant(t, "appearances", f.geStream(t, path, spec, url.Values{"limit": {"2"}}), wid(0), wid(5))
	_, body = f.get(t, path+"?limit=1", spec)
	if v := body["items"].([]any)[0].(map[string]any)["voices"].([]any); len(v) != 1 || v[0].(map[string]any)["id"] != "9101" {
		t.Fatalf("voices %+v", v)
	}
}

func TestV1EntityUpstreamDown(t *testing.T) {
	f := newGEFix(t)
	f.cat.fail.Store(true)
	for _, c := range []struct{ path, spec string }{
		{"/api/v1/tags", "/tags"},
		{"/api/v1/tags/5101", "/tags/{tag_id}"},
		{"/api/v1/tagged-works?tag_ids=5101", "/tagged-works"},
		{"/api/v1/companies/6101/works", "/companies/{company_id}/works"},
		{"/api/v1/credit-names?q=x", "/credit-names"},
		{"/api/v1/characters/9201/appearances", "/characters/{character_id}/appearances"},
	} {
		resp, body := f.get(t, c.path, c.spec)
		geStatus(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
	}
}
