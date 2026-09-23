package app

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"kun-galgame-api/pkg/problem"
)

func TestV1ToolsetsWalkTiedSort(t *testing.T) {
	f := newToolsetFix(t, nil)
	want := f.sqlIDs(t, `SELECT id::text FROM galgame_toolset
		WHERE id BETWEEN 930002201 AND 930002299 AND user_id IN (`+renderableAuthorsSQL()+`)
		ORDER BY resource_update_time DESC, id DESC`)
	if len(want) < 7 {
		t.Fatalf("seed too thin: %v", want)
	}
	for _, limit := range []int{2, 3} {
		if got := f.walkToolsets(t, "", limit); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("limit %d walked %v, want %v", limit, got, want)
		}
	}
	wantCreated := f.sqlIDs(t, `SELECT id::text FROM galgame_toolset
		WHERE id BETWEEN 930002201 AND 930002299 AND user_id IN (`+renderableAuthorsSQL()+`)
		ORDER BY created DESC, id DESC`)
	if got := f.walkToolsets(t, "&sort=created_desc", 2); fmt.Sprint(got) != fmt.Sprint(wantCreated) {
		t.Errorf("created_desc walked %v, want %v", got, wantCreated)
	}
}

func TestV1ToolsetsTotalExcludesUnrenderableAuthors(t *testing.T) {
	f := newToolsetFix(t, nil)
	resp, body := f.ts(t, http.MethodGet, "/api/v1/toolsets?limit=100", "/toolsets", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	want := f.scalar(t, `SELECT COUNT(*) FROM galgame_toolset
		WHERE id BETWEEN 930002201 AND 930002299 AND user_id IN (`+renderableAuthorsSQL()+`)`)
	if asInt(body["total"]) != want || body["total_relation"] != "eq" {
		t.Errorf("total %v %v, want %d eq", body["total"], body["total_relation"], want)
	}
	walked := f.walkToolsets(t, "", 2)
	if len(walked) != want {
		t.Errorf("walked %d items, want %d", len(walked), want)
	}
	for _, id := range walked {
		if id == idStr(g1TSTied+5) || id == idStr(g1TSTied+6) {
			t.Errorf("unrenderable author toolset %s in items", id)
		}
	}
}

func TestV1ToolsetsUnknownSortAndEnum(t *testing.T) {
	f := newToolsetFix(t, nil)
	resp, body := f.ts(t, http.MethodGet, "/api/v1/toolsets?sort=hot", "/toolsets", "", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeUnknownSort)
	resp, body = f.ts(t, http.MethodGet, "/api/v1/toolsets?toolset_type=engine", "/toolsets", "", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeUnknownEnumValue)
	resp, body = f.ts(t, http.MethodGet, "/api/v1/toolsets?limit=101", "/toolsets", "", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeLimitTooLarge)
	resp, body = f.ts(t, http.MethodGet, "/api/v1/toolsets?page=10000&limit=2", "/toolsets", "", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeInvalidParameter)
	if e := errorParam(body, "page"); e == nil || e["reason"] != "OUT_OF_RANGE" {
		t.Errorf("depth errors %+v", body["errors"])
	}
	resp, body = f.ts(t, http.MethodGet, "/api/v1/toolsets?q="+strings.Repeat("a", 101), "/toolsets", "", "", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("long q %d %+v", resp.StatusCode, body)
	}
}

func TestV1ToolsetsUserclientFailure(t *testing.T) {
	f := newToolsetFix(t, nil)
	f.failOA.Store(true)
	resp, body := f.ts(t, http.MethodGet, "/api/v1/toolsets", "/toolsets", "", "", nil)
	wantCode(t, resp, body, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
	resp, body = f.ts(t, http.MethodGet, "/api/v1/toolsets/"+idStr(g1TSMain), "/toolsets/{toolset_id}", "", "", nil)
	wantCode(t, resp, body, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
	resp, body = f.ts(t, http.MethodGet, "/api/v1/users/"+idStr(w3UserAlice)+"/toolsets", "/users/{user_id}/toolsets", "", "", nil)
	wantCode(t, resp, body, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
}

func TestV1ListUserToolsets(t *testing.T) {
	f := newToolsetFix(t, nil)
	resp, body := f.ts(t, http.MethodGet, "/api/v1/users/"+idStr(w3UserAlice)+"/toolsets?limit=100", "/users/{user_id}/toolsets", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("user list %d %+v", resp.StatusCode, body)
	}
	want := f.scalar(t, `SELECT COUNT(*) FROM galgame_toolset WHERE user_id = ? AND id BETWEEN 930002201 AND 930002299`, w3UserAlice)
	if asInt(body["total"]) != want {
		t.Errorf("alice total %v want %d", body["total"], want)
	}
	resp, body = f.ts(t, http.MethodGet, "/api/v1/users/"+idStr(g1HiddenA)+"/toolsets", "/users/{user_id}/toolsets", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	resp, body = f.ts(t, http.MethodGet, "/api/v1/users/930002198/toolsets", "/users/{user_id}/toolsets", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
}

func TestV1GetToolsetDetail(t *testing.T) {
	f := newToolsetFix(t, nil)
	updated := f.sqlIDs(t, `SELECT updated::text FROM galgame_toolset WHERE id = ?`, g1TSMain)
	resp, body := f.ts(t, http.MethodGet, "/api/v1/toolsets/"+idStr(g1TSMain), "/toolsets/{toolset_id}", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("detail %d %+v", resp.StatusCode, body)
	}
	if body["object"] != "toolset" || body["title"] != "Main tool" || asInt(body["view_count"]) != 4 || body["viewer"] != nil {
		t.Errorf("detail %+v", body)
	}
	if body["content"] == nil {
		t.Error("content missing")
	}
	if n := f.scalar(t, `SELECT view FROM galgame_toolset WHERE id = ?`, g1TSMain); n != 4 {
		t.Errorf("view %d", n)
	}
	if got := f.sqlIDs(t, `SELECT updated::text FROM galgame_toolset WHERE id = ?`, g1TSMain); fmt.Sprint(got) != fmt.Sprint(updated) {
		t.Errorf("a view moved updated: %v -> %v", updated, got)
	}
	aliases, _ := body["aliases"].([]any)
	if len(aliases) != 1 || aliases[0] != "alias-a" {
		t.Errorf("aliases %v", aliases)
	}
	homes, _ := body["homepage_urls"].([]any)
	if len(homes) != 1 || homes[0] != "https://example.com" {
		t.Errorf("homepage %v", homes)
	}
	dist, _ := body["practicality_distribution"].([]any)
	if len(dist) != 5 || asInt(dist[3]) != 1 || asInt(dist[4]) != 1 {
		t.Errorf("distribution %v", dist)
	}
	_, body = f.ts(t, http.MethodGet, "/api/v1/toolsets/"+idStr(g1TSMain), "/toolsets/{toolset_id}", "sess-alice", "", nil)
	v, _ := body["viewer"].(map[string]any)
	if v == nil || v["can_edit"] != true || v["can_delete"] != true || asInt(v["practicality_rating"]) != 5 {
		t.Errorf("alice viewer %+v", body["viewer"])
	}
	bearer := http.Header{"Authorization": {"Bearer staff-token"}}
	resp, raw := f.doJSON(t, http.MethodGet, "/api/v1/toolsets/"+idStr(g1TSMain), "", "/toolsets/{toolset_id}", "", bearer, nil)
	staff := problemMap(t, raw)
	sv, _ := staff["viewer"].(map[string]any)
	if resp.StatusCode != http.StatusOK || sv == nil || sv["can_edit"] != false || sv["can_delete"] != false {
		t.Errorf("Bearer moderator viewer %d %+v", resp.StatusCode, staff["viewer"])
	}
	resp, body = f.ts(t, http.MethodGet, "/api/v1/toolsets/"+idStr(g1TSTied+5), "/toolsets/{toolset_id}", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	resp, body = f.ts(t, http.MethodGet, "/api/v1/toolsets/"+idStr(g1TSGone), "/toolsets/{toolset_id}", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
}

func TestV1GetToolsetHomepageJSONError(t *testing.T) {
	f := newToolsetFix(t, nil)
	if err := f.db.Exec(`UPDATE galgame_toolset SET homepage = '{"url": 1}'::jsonb WHERE id = ?`, g1TSBad).Error; err != nil {
		t.Fatal(err)
	}
	resp, body := f.ts(t, http.MethodGet, "/api/v1/toolsets/"+idStr(g1TSBad), "/toolsets/{toolset_id}", "", "", nil)
	wantCode(t, resp, body, http.StatusInternalServerError, problem.CodeInternalError)
}

func TestV1GetToolsetResource(t *testing.T) {
	f := newToolsetFix(t, nil)
	resp, body := f.ts(t, http.MethodGet, "/api/v1/toolsets/"+idStr(g1TSMain)+"/resources/"+idStr(g1ResLink),
		"/toolsets/{toolset_id}/resources/{resource_id}", "", "", nil)
	if resp.StatusCode != http.StatusOK || body["object"] != "toolset_resource" || body["toolset_resource_type"] != "link" {
		t.Fatalf("resource %d %+v", resp.StatusCode, body)
	}
	if _, ok := body["download_url"]; ok {
		t.Errorf("GET leaked download_url: %+v", body)
	}
	if _, ok := body["extraction_code"]; ok {
		t.Errorf("GET leaked extraction_code: %+v", body)
	}
	if n := f.scalar(t, `SELECT download FROM galgame_toolset_resource WHERE id = ?`, g1ResLink); n != 4 {
		t.Errorf("GET counted a download: %d", n)
	}
	resp, body = f.ts(t, http.MethodGet, "/api/v1/toolsets/"+idStr(g1TSBob)+"/resources/"+idStr(g1ResLink),
		"/toolsets/{toolset_id}/resources/{resource_id}", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
}
