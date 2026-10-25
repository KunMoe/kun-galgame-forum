package app

import (
	"net/http"
	"strconv"
	"testing"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"
)

func TestV1CompanyFollowLifecycle(t *testing.T) {
	f := newG6Fix(t)
	f.user.followNotFound = 880002
	f.user.followCapped = 880003
	spec := "/me/followed-companies/{company_id}"
	one := "/api/v1/me/followed-companies/880001"

	resp, body := f.call(t, http.MethodGet, one, spec, "sess-alice", "", nil)
	geStatus(t, resp, body, http.StatusOK, "")
	assertCompanyFollowState(t, body, "880001", false)

	resp, body = f.call(t, http.MethodPut, one, spec, "sess-alice", "", nil)
	geStatus(t, resp, body, http.StatusOK, "")
	assertCompanyFollowState(t, body, "880001", true)

	resp, body = f.call(t, http.MethodGet, one, spec, "sess-alice", "", nil)
	geStatus(t, resp, body, http.StatusOK, "")
	assertCompanyFollowState(t, body, "880001", true)

	resp, body = f.call(t, http.MethodDelete, one, spec, "sess-alice", "", nil)
	geStatus(t, resp, body, http.StatusOK, "")
	assertCompanyFollowState(t, body, "880001", false)

	resp, body = f.call(t, http.MethodGet, one, spec, "", "", nil)
	wantCode(t, resp, body, http.StatusUnauthorized, problem.CodeMissingCredential)

	resp, body = f.call(t, http.MethodPut, "/api/v1/me/followed-companies/880002", spec, "sess-alice", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)

	resp, body = f.call(t, http.MethodPut, "/api/v1/me/followed-companies/880003", spec, "sess-alice", "", nil)
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
}

func TestV1ListMyFollowedCompanies(t *testing.T) {
	f := newG6Fix(t)
	f.user.companyFollows = []catalogclient.CompanyFollowRecord{
		{CompanyID: 880301, CreatedAt: "2026-09-04T00:00:00Z"},
		{CompanyID: 880302, CreatedAt: "2026-09-03T00:00:00Z"},
		{CompanyID: 880303, CreatedAt: "2026-09-02T00:00:00Z"},
		{CompanyID: 880304, CreatedAt: "2026-09-01T00:00:00Z"},
	}
	f.cat.taxonomy = map[string][]client.CatalogTaxonomyItem{
		"labels": {
			{ID: 880301, DisplayName: "Brand 880301", Kind: "game_brand", WorkCount: 1},
			{ID: 880303, DisplayName: "Brand 880303", Kind: "game_brand", WorkCount: 1},
			{ID: 880304, DisplayName: "Brand 880304", Kind: "game_brand", WorkCount: 1},
		},
	}
	spec := "/me/followed-companies"

	resp, body := f.call(t, http.MethodGet, "/api/v1/me/followed-companies?page=1&limit=2", spec, "sess-alice", "", nil)
	geStatus(t, resp, body, http.StatusOK, "")
	assertFollowedPage(t, body, 3, false)
	items := body["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("page 1 items %+v", items)
	}
	assertFollowedCompany(t, items[0], "880301", "2026-09-04T00:00:00Z")
	assertFollowedCompany(t, items[1], "880303", "2026-09-02T00:00:00Z")
	assertHydratedIDs(t, f, "880301,880302,880303,880304")

	resp, body = f.call(t, http.MethodGet, "/api/v1/me/followed-companies?page=2&limit=2", spec, "sess-alice", "", nil)
	geStatus(t, resp, body, http.StatusOK, "")
	assertFollowedPage(t, body, 3, false)
	items = body["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("page 2 items %+v", items)
	}
	assertFollowedCompany(t, items[0], "880304", "2026-09-01T00:00:00Z")
}

func TestV1CompanyDetailFollowerCount(t *testing.T) {
	f := newGEFix(t)
	resp, body := f.get(t, "/api/v1/companies/6101", "/companies/{company_id}")
	geStatus(t, resp, body, http.StatusOK, "")
	if body["follower_count"] != nil {
		t.Fatalf("count before catalog sent one: %+v", body["follower_count"])
	}
	n := 17
	f.cat.labels[strconv.Itoa(geCompany)].FollowerCount = &n
	resp, body = f.get(t, "/api/v1/companies/6101", "/companies/{company_id}")
	geStatus(t, resp, body, http.StatusOK, "")
	if asInt(body["follower_count"]) != 17 {
		t.Fatalf("follower_count %+v", body["follower_count"])
	}
	resp, list := f.get(t, "/api/v1/companies?ids=6101", "/companies")
	geStatus(t, resp, list, http.StatusOK, "")
	items, _ := list["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("ids list %+v", list)
	}
	if _, ok := items[0].(map[string]any)["follower_count"]; ok {
		t.Fatalf("company list carries follower_count: %+v", items[0])
	}
}

func assertCompanyFollowState(t *testing.T, body map[string]any, id string, following bool) {
	t.Helper()
	if body["object"] != "company_follow_state" || body["id"] != id || body["company_id"] != id || body["is_following"] != following {
		t.Fatalf("state %+v, want id %s following %v", body, id, following)
	}
}

func assertFollowedPage(t *testing.T, body map[string]any, total int, truncated bool) {
	t.Helper()
	if body["object"] != "list" || body["total_relation"] != "eq" || body["is_truncated"] != truncated || asInt(body["total"]) != total {
		t.Fatalf("list envelope %+v, want total %d truncated %v", body, total, truncated)
	}
}

func assertFollowedCompany(t *testing.T, raw any, id, followedAt string) {
	t.Helper()
	item, _ := raw.(map[string]any)
	company, _ := item["company"].(map[string]any)
	if item["object"] != "followed_company" || item["followed_at"] != followedAt || company["object"] != "company" || company["id"] != id {
		t.Fatalf("followed company %+v, want %s at %s", item, id, followedAt)
	}
}

func assertHydratedIDs(t *testing.T, f *g6Fix, ids string) {
	t.Helper()
	f.cat.mu.Lock()
	defer f.cat.mu.Unlock()
	if len(f.cat.taxonomyQ) != 1 || f.cat.taxonomyQ[0].Get("ids") != ids {
		t.Fatalf("hydrated queries %+v, want ids %s", f.cat.taxonomyQ, ids)
	}
}
