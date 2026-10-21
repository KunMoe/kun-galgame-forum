package app

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"kun-galgame-api/pkg/problem"
)

func (f *resourceFix) setRendered(t *testing.T, workID int, rendered bool) {
	t.Helper()
	if err := f.db.Exec(`UPDATE galgame SET catalog_rendered = ? WHERE id = ?`, rendered, workID).Error; err != nil {
		t.Fatal(err)
	}
}

func TestV1GalgameResourceFacesHideOnUnrenderedWork(t *testing.T) {
	f := newResourceFix(t, nil)
	res := "/api/v1/galgame-resources/" + idStr(g3ResMain)
	views := f.scalar(t, `SELECT view FROM galgame_resource WHERE id = ?`, g3ResMain)
	downloads := f.scalar(t, `SELECT download FROM galgame_resource WHERE id = ?`, g3ResMain)
	f.setRendered(t, g3WorkSFW, false)

	resp, body := f.rs(t, http.MethodGet, res, "/galgame-resources/{resource_id}", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	resp, body = f.rs(t, http.MethodPost, res+"/downloads", "/galgame-resources/{resource_id}/downloads", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	if raw, _ := json.Marshal(body); strings.Contains(string(raw), g3SecretURL) || strings.Contains(string(raw), "abcd") {
		t.Errorf("a hidden work's resource served its secrets: %s", raw)
	}
	resp, body = f.rs(t, http.MethodGet, res+"/source", "/galgame-resources/{resource_id}/source", "sess-alice", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	resp, body = f.rs(t, http.MethodPut, res+"/like", "/galgame-resources/{resource_id}/like", "sess-bob", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	resp, body = f.rs(t, http.MethodDelete, res+"/like", "/galgame-resources/{resource_id}/like", "sess-bob", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	resp, body = f.rs(t, http.MethodPost, res+"/expiry-reports", "/galgame-resources/{resource_id}/expiry-reports", "sess-bob", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	resp, body = f.rs(t, http.MethodGet, "/api/v1/works/"+idStr(g3WorkSFW)+"/resources", "/works/{work_id}/resources", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)

	if n := f.scalar(t, `SELECT view FROM galgame_resource WHERE id = ?`, g3ResMain); n != views {
		t.Errorf("view %d, want %d untouched", n, views)
	}
	if n := f.scalar(t, `SELECT download FROM galgame_resource WHERE id = ?`, g3ResMain); n != downloads {
		t.Errorf("download %d, want %d untouched", n, downloads)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_resource_like WHERE galgame_resource_id = ?`, g3ResMain); n != 0 {
		t.Errorf("a like landed on a hidden work's resource: %d", n)
	}
	if n := f.scalar(t, `SELECT status FROM galgame_resource WHERE id = ?`, g3ResMain); n != 0 {
		t.Errorf("an expiry report changed a hidden work's resource: status %d", n)
	}

	f.setRendered(t, g3WorkSFW, true)
	resp, body = f.rs(t, http.MethodGet, res, "/galgame-resources/{resource_id}", "", "", nil)
	if resp.StatusCode != http.StatusOK || asInt(body["view_count"]) != views+1 {
		t.Fatalf("restored get %d %+v", resp.StatusCode, body)
	}
	resp, body = f.rs(t, http.MethodPost, res+"/downloads", "/galgame-resources/{resource_id}/downloads", "", "", nil)
	if resp.StatusCode != http.StatusOK || body["extraction_code"] != "abcd" {
		t.Fatalf("restored downloads %d %+v", resp.StatusCode, body)
	}
	resp, body = f.rs(t, http.MethodPut, res+"/like", "/galgame-resources/{resource_id}/like", "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK || asInt(body["like_count"]) != 1 {
		t.Fatalf("restored like %d %+v", resp.StatusCode, body)
	}
}

func TestV1GetGalgameResourceCountsNoViewItDoesNotServe(t *testing.T) {
	f := newResourceFix(t, nil)
	views := f.scalar(t, `SELECT view FROM galgame_resource WHERE id = ?`, g3ResMain)
	f.cat.omitIDs = map[int]bool{g3WorkSFW: true}
	resp, body := f.rs(t, http.MethodGet, "/api/v1/galgame-resources/"+idStr(g3ResMain), "/galgame-resources/{resource_id}", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	if n := f.scalar(t, `SELECT view FROM galgame_resource WHERE id = ?`, g3ResMain); n != views {
		t.Errorf("view %d after a 404, want %d", n, views)
	}
}
