package app

import (
	"fmt"
	"net/http"
	"testing"

	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/problem"
)

func TestV1CreateToolsetResourceLink(t *testing.T) {
	f := newToolsetFix(t, nil)
	body := map[string]any{
		"toolset_resource_type": "link",
		"link_url":      "https://cdn.example/tool.7z",
		"size_label":    "12mb",
		"note":          "n",
	}
	resp, got := f.ts(t, http.MethodPost, "/api/v1/toolsets/"+idStr(g1TSMain)+"/resources",
		"/toolsets/{toolset_id}/resources", "sess-other", keyUUID(10), body)
	if resp.StatusCode != http.StatusCreated || got["toolset_resource_type"] != "link" {
		t.Fatalf("create link %d %+v", resp.StatusCode, got)
	}
	id := strID(got["id"])
	if resp.Header.Get("Location") != "/api/v1/toolsets/"+idStr(g1TSMain)+"/resources/"+id {
		t.Errorf("Location %s", resp.Header.Get("Location"))
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_toolset_contributor WHERE toolset_id = ? AND user_id = ?`, g1TSMain, w3UserOther); n != 1 {
		t.Errorf("contributor %d", n)
	}
	aw := f.snapshotAwards()
	found := false
	for _, a := range aw {
		if a.ref == moemoepoint.Ref("toolset_resource", asInt(id)) && a.key == moemoepoint.Key("resource_create", id) {
			found = true
		}
	}
	if !found {
		t.Errorf("resource award %+v", aw)
	}
}

func TestV1CreateToolsetResourceFileUnknownReference(t *testing.T) {
	f := newToolsetFix(t, nil)
	post := func(artifact string) (*http.Response, map[string]any) {
		t.Helper()
		return f.ts(t, http.MethodPost, "/api/v1/toolsets/"+idStr(g1TSMain)+"/resources",
			"/toolsets/{toolset_id}/resources", "sess-alice", keyUUID(int(artifact[len(artifact)-1])), map[string]any{
				"toolset_resource_type": "file", "artifact_id": artifact,
			})
	}
	before := f.scalar(t, `SELECT COUNT(*) FROM galgame_toolset_resource WHERE toolset_id = ?`, g1TSMain)
	resp, got := post(g1UpBob)
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if e := errorAt(got, "/artifact_id"); e == nil || e["reason"] != "UNKNOWN_REFERENCE" {
		t.Errorf("other user %+v", got["errors"])
	}
	resp, got = post(g1UpOther)
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if e := errorAt(got, "/artifact_id"); e == nil || e["reason"] != "UNKNOWN_REFERENCE" {
		t.Errorf("other toolset %+v", got["errors"])
	}
	resp, got = post(g1UpPend)
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if e := errorAt(got, "/artifact_id"); e == nil || e["reason"] != "UNKNOWN_REFERENCE" {
		t.Errorf("pending %+v", got["errors"])
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_toolset_resource WHERE toolset_id = ?`, g1TSMain); n != before {
		t.Error("unknown reference wrote a row")
	}

	resp, got = f.ts(t, http.MethodPost, "/api/v1/toolsets/"+idStr(g1TSMain)+"/resources",
		"/toolsets/{toolset_id}/resources", "sess-alice", keyUUID(21), map[string]any{
			"toolset_resource_type": "file", "artifact_id": g1UpAliceA,
		})
	wantCode(t, resp, got, http.StatusConflict, problem.CodeAlreadyExists)

	resp, got = f.ts(t, http.MethodPost, "/api/v1/toolsets/"+idStr(g1TSMain)+"/resources",
		"/toolsets/{toolset_id}/resources", "sess-alice", keyUUID(22), map[string]any{
			"toolset_resource_type": "file", "link_url": "https://x.example",
		})
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
}

func TestV1CreateFileResourceFromCompletedUpload(t *testing.T) {
	f := newToolsetFix(t, nil)
	resp, got := f.ts(t, http.MethodPost, "/api/v1/toolsets/"+idStr(g1TSMain)+"/resources",
		"/toolsets/{toolset_id}/resources", "sess-alice", keyUUID(23), map[string]any{
			"toolset_resource_type": "file", "artifact_id": g1UpAliceB,
		})
	if resp.StatusCode != http.StatusCreated || got["toolset_resource_type"] != "file" || got["archive"] == nil {
		t.Fatalf("file resource %d %+v", resp.StatusCode, got)
	}
}

func TestV1PatchAndDeleteResourcePath(t *testing.T) {
	f := newToolsetFix(t, nil)
	resp, got := f.ts(t, http.MethodPatch, "/api/v1/toolsets/"+idStr(g1TSBob)+"/resources/"+idStr(g1ResLink),
		"/toolsets/{toolset_id}/resources/{resource_id}", "sess-alice", "", map[string]any{"note": "x"})
	wantCode(t, resp, got, http.StatusNotFound, problem.CodeNotFound)
	resp, got = f.ts(t, http.MethodDelete, "/api/v1/toolsets/"+idStr(g1TSBob)+"/resources/"+idStr(g1ResLink),
		"/toolsets/{toolset_id}/resources/{resource_id}", "sess-alice", "", nil)
	wantCode(t, resp, got, http.StatusNotFound, problem.CodeNotFound)

	resp, got = f.ts(t, http.MethodPatch, "/api/v1/toolsets/"+idStr(g1TSMain)+"/resources/"+idStr(g1ResFile),
		"/toolsets/{toolset_id}/resources/{resource_id}", "sess-alice", "", map[string]any{"link_url": "https://x.example"})
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if e := errorAt(got, "/link_url"); e == nil || e["reason"] != "IMMUTABLE" {
		t.Errorf("file url %+v", got["errors"])
	}

	resp, got = f.ts(t, http.MethodPatch, "/api/v1/toolsets/"+idStr(g1TSMain)+"/resources/"+idStr(g1ResLink),
		"/toolsets/{toolset_id}/resources/{resource_id}", "sess-bob", "", map[string]any{"note": "nope"})
	wantCode(t, resp, got, http.StatusForbidden, problem.CodePermissionRequired)

	resp, got = f.ts(t, http.MethodPatch, "/api/v1/toolsets/"+idStr(g1TSMain)+"/resources/"+idStr(g1ResLink),
		"/toolsets/{toolset_id}/resources/{resource_id}", "sess-alice", "", map[string]any{"note": "edited"})
	if resp.StatusCode != http.StatusOK || got["note"] != "edited" {
		t.Errorf("patch note %d %+v", resp.StatusCode, got)
	}

	resp, _ = f.ts(t, http.MethodDelete, "/api/v1/toolsets/"+idStr(g1TSMain)+"/resources/"+idStr(g1ResBob),
		"/toolsets/{toolset_id}/resources/{resource_id}", "sess-bob", "", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete %d", resp.StatusCode)
	}
}

const g1ResBareLink = 930002306

func TestV1CreateToolsetDownload(t *testing.T) {
	f := newToolsetFix(t, nil)
	path := "/api/v1/toolsets/" + idStr(g1TSMain) + "/resources/" + idStr(g1ResLink) + "/downloads"
	spec := "/toolsets/{toolset_id}/resources/{resource_id}/downloads"
	resp, got := f.ts(t, http.MethodPost, path, spec, "", "", nil)
	if resp.StatusCode != http.StatusOK || got["object"] != "toolset_download" || got["download_url"] != "https://files.example/a.zip" {
		t.Fatalf("link download %d %+v", resp.StatusCode, got)
	}
	if got["extraction_code"] != "ex1" || got["archive_password"] != "pw1" {
		t.Errorf("secrets %+v", got)
	}
	if n := f.scalar(t, `SELECT download FROM galgame_toolset_resource WHERE id = ?`, g1ResLink); n != 5 {
		t.Errorf("download count %d", n)
	}
	updated := f.sqlIDs(t, `SELECT updated::text FROM galgame_toolset_resource WHERE id = ?`, g1ResFile)
	path = "/api/v1/toolsets/" + idStr(g1TSMain) + "/resources/" + idStr(g1ResFile) + "/downloads"
	resp, got = f.ts(t, http.MethodPost, path, spec, "", "", nil)
	if resp.StatusCode != http.StatusOK || got["download_url"] != "https://dl.test/"+g1UpAliceA || got["expires_at"] == nil {
		t.Errorf("file download %d %+v", resp.StatusCode, got)
	}
	if got := f.sqlIDs(t, `SELECT updated::text FROM galgame_toolset_resource WHERE id = ?`, g1ResFile); fmt.Sprint(got) != fmt.Sprint(updated) {
		t.Errorf("download moved updated: %v -> %v", updated, got)
	}
	f.art.failDownload[g1UpAliceA] = true
	resp, got = f.ts(t, http.MethodPost, path, spec, "", "", nil)
	wantCode(t, resp, got, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)

	if err := f.db.Exec(`INSERT INTO galgame_toolset_resource (id, content, type, artifact_uuid, code, password, size, note, download, toolset_id, user_id, created, updated)
		VALUES (?, '', 'user', '', 'link: https://pan.example/s/1', '', '1mb', '', 0, ?, ?, now(), now())`, g1ResBareLink, g1TSMain, w3UserAlice).Error; err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{g1ResEmpty, g1ResBareLink} {
		resp, got = f.ts(t, http.MethodPost, "/api/v1/toolsets/"+idStr(g1TSMain)+"/resources/"+idStr(id)+"/downloads", spec, "", "", nil)
		if v, ok := got["download_url"]; resp.StatusCode != http.StatusOK || !ok || v != nil {
			t.Errorf("resource %d with nothing on record: %d %+v", id, resp.StatusCode, got)
		}
		if n := f.scalar(t, `SELECT download FROM galgame_toolset_resource WHERE id = ?`, id); n != 0 {
			t.Errorf("resource %d with nothing on record counted a download: %d", id, n)
		}
	}
	if got["extraction_code"] != "link: https://pan.example/s/1" {
		t.Errorf("bare link lost its extraction code %+v", got)
	}

	resp, got = f.ts(t, http.MethodPost, "/api/v1/toolsets/"+idStr(g1TSBob)+"/resources/"+idStr(g1ResLink)+"/downloads",
		spec, "", "", nil)
	wantCode(t, resp, got, http.StatusNotFound, problem.CodeNotFound)
}

func TestV1GetToolsetResourceSource(t *testing.T) {
	f := newToolsetFix(t, nil)
	path := "/api/v1/toolsets/" + idStr(g1TSMain) + "/resources/" + idStr(g1ResLink) + "/source"
	spec := "/toolsets/{toolset_id}/resources/{resource_id}/source"
	before := f.scalar(t, `SELECT download FROM galgame_toolset_resource WHERE id = ?`, g1ResLink)
	resp, got := f.ts(t, http.MethodGet, path, spec, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK || got["object"] != "toolset_resource_source" || got["link_url"] == nil {
		t.Fatalf("source %d %+v", resp.StatusCode, got)
	}
	if n := f.scalar(t, `SELECT download FROM galgame_toolset_resource WHERE id = ?`, g1ResLink); n != before {
		t.Errorf("reading the source counted a download: %d -> %d", before, n)
	}
	resp, got = f.ts(t, http.MethodGet, path, spec, "sess-bob", "", nil)
	wantCode(t, resp, got, http.StatusForbidden, problem.CodePermissionRequired)
	resp, got = f.ts(t, http.MethodGet, path, spec, "", "", nil)
	wantCode(t, resp, got, http.StatusUnauthorized, problem.CodeMissingCredential)

	if err := f.db.Exec(`INSERT INTO galgame_toolset_resource (id, content, type, artifact_uuid, code, password, size, note, download, toolset_id, user_id, created, updated)
		VALUES (?, '', 'user', '', '', '', '1mb', '', 0, ?, ?, now(), now())`, g1ResBareLink, g1TSMain, w3UserAlice).Error; err != nil {
		t.Fatal(err)
	}
	resp, got = f.ts(t, http.MethodGet, "/api/v1/toolsets/"+idStr(g1TSMain)+"/resources/"+idStr(g1ResBareLink)+"/source", spec, "sess-alice", "", nil)
	if _, ok := got["link_url"]; resp.StatusCode != http.StatusOK || ok || got["size_label"] != "1mb" {
		t.Errorf("bare link source %d %+v", resp.StatusCode, got)
	}
}
