package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"kun-galgame-api/pkg/problem"
)

func TestV1ListGalgameResourcesWalkTiedCreated(t *testing.T) {
	f := newResourceFix(t, nil)
	want := f.sqlIDs(t, `SELECT r.id::text FROM galgame_resource r
		JOIN galgame g ON g.id = r.work_id
		WHERE r.id BETWEEN 930004201 AND 930004299
		  AND g.published = true
		  AND g.content_limit = 'sfw'
		  AND r.user_id IN (`+renderableResourceAuthorsSQL()+`)
		ORDER BY r.created DESC, r.id DESC`)
	if len(want) < 7 {
		t.Fatalf("seed too thin: %v", want)
	}
	for _, limit := range []int{2, 3} {
		if got := f.walkResources(t, "", limit); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("limit %d walked %v, want %v", limit, got, want)
		}
	}
}

func TestV1ListGalgameResourcesTotalExcludesUnrenderableAndNSFW(t *testing.T) {
	f := newResourceFix(t, nil)
	resp, body := f.rs(t, http.MethodGet, "/api/v1/galgame-resources?limit=100", "/galgame-resources", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	want := f.scalar(t, `SELECT COUNT(*) FROM galgame_resource r
		JOIN galgame g ON g.id = r.work_id
		WHERE r.id BETWEEN 930004201 AND 930004299
		  AND g.published = true
		  AND g.content_limit = 'sfw'
		  AND r.user_id IN (`+renderableResourceAuthorsSQL()+`)`)
	if asInt(body["total"]) != want || body["total_relation"] != "eq" {
		t.Errorf("total %v %v, want %d eq", body["total"], body["total_relation"], want)
	}
	walked := f.walkResources(t, "", 2)
	if len(walked) != want {
		t.Errorf("walked %d items, want %d", len(walked), want)
	}
	for _, id := range walked {
		if id == idStr(g3ResTied+5) || id == idStr(g3ResTied+6) || id == idStr(g3ResNSFW) {
			t.Errorf("hidden item %s in list", id)
		}
	}
}

func TestV1ListGalgameResourcesOmitsSecrets(t *testing.T) {
	f := newResourceFix(t, nil)
	resp, body := f.rs(t, http.MethodGet, "/api/v1/galgame-resources?limit=100", "/galgame-resources", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	raw, _ := json.Marshal(body)
	if strings.Contains(string(raw), g3SecretURL) {
		t.Error("list leaked a download URL")
	}
	items, _ := body["items"].([]any)
	if len(items) == 0 {
		t.Fatal("empty list")
	}
	for _, it := range items {
		m, _ := it.(map[string]any)
		if _, ok := m["download_urls"]; ok {
			t.Errorf("list item has download_urls: %+v", m)
		}
		if _, ok := m["extraction_code"]; ok {
			t.Errorf("list item has extraction_code: %+v", m)
		}
		if _, ok := m["archive_password"]; ok {
			t.Errorf("list item has archive_password: %+v", m)
		}
	}
}

func TestV1ListGalgameResourcesUnknownSortAndEnum(t *testing.T) {
	f := newResourceFix(t, nil)
	resp, body := f.rs(t, http.MethodGet, "/api/v1/galgame-resources?sort=hot", "/galgame-resources", "", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeUnknownSort)
	resp, body = f.rs(t, http.MethodGet, "/api/v1/galgame-resources?state=alive", "/galgame-resources", "", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeUnknownEnumValue)
	resp, body = f.rs(t, http.MethodGet, "/api/v1/galgame-resources?limit=101", "/galgame-resources", "", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeLimitTooLarge)
	resp, body = f.rs(t, http.MethodGet, "/api/v1/galgame-resources?page=1000&limit=100", "/galgame-resources", "", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeInvalidParameter)
}

func TestV1ListGalgameResourcesQAndCatalogDown(t *testing.T) {
	f := newResourceFix(t, nil)
	resp, body := f.rs(t, http.MethodGet, "/api/v1/galgame-resources?q=AlphaRes", "/galgame-resources", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("q %d %+v", resp.StatusCode, body)
	}
	if asInt(body["total"]) < 1 {
		t.Errorf("q total %v", body["total"])
	}
	f.cat.fail.Store(true)
	resp, body = f.rs(t, http.MethodGet, "/api/v1/galgame-resources?q=AlphaRes", "/galgame-resources", "", "", nil)
	wantCode(t, resp, body, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
}

func TestV1GetGalgameResourceOmitsSecretsAndCountsView(t *testing.T) {
	f := newResourceFix(t, nil)
	before := f.scalar(t, `SELECT view FROM galgame_resource WHERE id = ?`, g3ResMain)
	resp, body := f.rs(t, http.MethodGet, "/api/v1/galgame-resources/"+idStr(g3ResMain), "/galgame-resources/{resource_id}", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get %d %+v", resp.StatusCode, body)
	}
	if _, ok := body["download_urls"]; ok {
		t.Error("detail leaked download_urls")
	}
	if _, ok := body["extraction_code"]; ok {
		t.Error("detail leaked extraction_code")
	}
	if _, ok := body["archive_password"]; ok {
		t.Error("detail leaked archive_password")
	}
	if asInt(body["view_count"]) != before+1 {
		t.Errorf("view_count %v want %d", body["view_count"], before+1)
	}
	updated := f.scalar(t, `SELECT EXTRACT(EPOCH FROM updated)::int FROM galgame_resource WHERE id = ?`, g3ResMain)
	created := f.scalar(t, `SELECT EXTRACT(EPOCH FROM created)::int FROM galgame_resource WHERE id = ?`, g3ResMain)
	_ = updated
	_ = created
	resp, body = f.rs(t, http.MethodGet, "/api/v1/galgame-resources/"+idStr(g3ResGone), "/galgame-resources/{resource_id}", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	resp, body = f.rs(t, http.MethodGet, "/api/v1/galgame-resources/"+idStr(g3ResTied+5), "/galgame-resources/{resource_id}", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
}

func TestV1CreateGalgameResourceDownload(t *testing.T) {
	f := newResourceFix(t, nil)
	before := f.scalar(t, `SELECT download FROM galgame_resource WHERE id = ?`, g3ResMain)
	resp, body := f.rs(t, http.MethodPost, "/api/v1/galgame-resources/"+idStr(g3ResMain)+"/downloads",
		"/galgame-resources/{resource_id}/downloads", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("downloads %d %+v", resp.StatusCode, body)
	}
	urls, _ := body["download_urls"].([]any)
	if len(urls) != 2 || urls[0] != g3SecretURL {
		t.Errorf("urls %+v", body["download_urls"])
	}
	if body["extraction_code"] != "abcd" || body["archive_password"] != "zip" {
		t.Errorf("secrets %+v", body)
	}
	after := f.scalar(t, `SELECT download FROM galgame_resource WHERE id = ?`, g3ResMain)
	if after != before+1 {
		t.Errorf("download %d want %d", after, before+1)
	}
}

func TestV1GetGalgameResourceSource(t *testing.T) {
	f := newResourceFix(t, nil)
	resp, body := f.rs(t, http.MethodGet, "/api/v1/galgame-resources/"+idStr(g3ResMain)+"/source",
		"/galgame-resources/{resource_id}/source", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("source %d %+v", resp.StatusCode, body)
	}
	if body["extraction_code"] != "abcd" {
		t.Errorf("source secrets %+v", body)
	}
	dl := f.scalar(t, `SELECT download FROM galgame_resource WHERE id = ?`, g3ResMain)
	_ = dl
	resp, body = f.rs(t, http.MethodGet, "/api/v1/galgame-resources/"+idStr(g3ResMain)+"/source",
		"/galgame-resources/{resource_id}/source", "sess-bob", "", nil)
	wantCode(t, resp, body, http.StatusForbidden, problem.CodePermissionRequired)
	resp, body = f.rs(t, http.MethodGet, "/api/v1/galgame-resources/"+idStr(g3ResMain)+"/source",
		"/galgame-resources/{resource_id}/source", "", "", nil)
	wantCode(t, resp, body, http.StatusUnauthorized, problem.CodeMissingCredential)
}

func TestV1ListWorkResourcesAndGetWork(t *testing.T) {
	f := newResourceFix(t, nil)
	resp, body := f.rs(t, http.MethodGet, "/api/v1/works/"+idStr(g3WorkSFW)+"/resources?limit=100",
		"/works/{work_id}/resources", "", "", nil)
	if resp.StatusCode != http.StatusOK || asInt(body["total"]) < 1 {
		t.Fatalf("work list %d %+v", resp.StatusCode, body)
	}
	resp, body = f.rs(t, http.MethodGet, "/api/v1/works/"+idStr(g3WorkUnpub)+"/resources",
		"/works/{work_id}/resources", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	resp, body = f.rs(t, http.MethodGet, "/api/v1/works/"+idStr(g3WorkMiss),
		"/works/{work_id}", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	resp, body = f.rs(t, http.MethodGet, "/api/v1/works/"+idStr(g3WorkHidden),
		"/works/{work_id}", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	resp, body = f.rs(t, http.MethodGet, "/api/v1/works/"+idStr(g3WorkUnpub),
		"/works/{work_id}", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("getWork unpublished %d %+v", resp.StatusCode, body)
	}
	if strID(body["id"]) != idStr(g3WorkUnpub) {
		t.Errorf("getWork %+v", body)
	}
}

func TestV1ListGalgameResourcesIncludeNSFW(t *testing.T) {
	f := newResourceFix(t, nil)
	resp, body := f.rs(t, http.MethodGet, "/api/v1/galgame-resources?include_nsfw=true&limit=100",
		"/galgame-resources", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("include_nsfw %d %+v", resp.StatusCode, body)
	}
	found := false
	for _, id := range adminItemIDs(body) {
		if id == idStr(g3ResNSFW) {
			found = true
		}
	}
	if !found {
		t.Error("NSFW resource missing when include_nsfw=true")
	}
}

func TestV1GetGalgameResourceBearerViewerOmitsStaff(t *testing.T) {
	f := newResourceFix(t, nil)
	resp, raw := f.doJSON(t, http.MethodGet, "/api/v1/galgame-resources/"+idStr(g3ResMain), "",
		"/galgame-resources/{resource_id}", "",
		http.Header{"Authorization": {"Bearer staff-token"}}, nil)
	body := problemMap(t, raw)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bearer get %d %+v", resp.StatusCode, body)
	}
	v, _ := body["viewer"].(map[string]any)
	if v == nil || v["can_edit"] != false || v["can_delete"] != false {
		t.Errorf("bearer staff viewer %+v", body["viewer"])
	}
}

func TestV1GalgameResourceUserclientDown(t *testing.T) {
	f := newResourceFix(t, nil)
	f.failOA.Store(true)
	resp, body := f.rs(t, http.MethodGet, "/api/v1/galgame-resources?limit=10", "/galgame-resources", "", "", nil)
	wantCode(t, resp, body, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
	resp, body = f.rs(t, http.MethodGet, "/api/v1/galgame-resources/"+idStr(g3ResMain),
		"/galgame-resources/{resource_id}", "", "", nil)
	wantCode(t, resp, body, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
	resp, body = f.rs(t, http.MethodPost, "/api/v1/galgame-resources/"+idStr(g3ResMain)+"/downloads",
		"/galgame-resources/{resource_id}/downloads", "", "", nil)
	wantCode(t, resp, body, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
	resp, body = f.rs(t, http.MethodGet, "/api/v1/galgame-resources/"+idStr(g3ResMain)+"/source",
		"/galgame-resources/{resource_id}/source", "sess-alice", "", nil)
	wantCode(t, resp, body, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
}
