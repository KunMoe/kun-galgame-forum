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
		"resource_type": "link",
		"link_url":      "https://cdn.example/tool.7z",
		"size_label":    "12mb",
		"note":          "n",
	}
	resp, got := f.ts(t, http.MethodPost, "/api/v1/toolsets/"+idStr(g1TSMain)+"/resources",
		"/toolsets/{toolset_id}/resources", "sess-other", keyUUID(10), body)
	if resp.StatusCode != http.StatusCreated || got["resource_type"] != "link" {
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
				"resource_type": "file", "artifact_id": artifact,
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
			"resource_type": "file", "artifact_id": g1UpAliceA,
		})
	wantCode(t, resp, got, http.StatusConflict, problem.CodeAlreadyExists)

	resp, got = f.ts(t, http.MethodPost, "/api/v1/toolsets/"+idStr(g1TSMain)+"/resources",
		"/toolsets/{toolset_id}/resources", "sess-alice", keyUUID(22), map[string]any{
			"resource_type": "file", "link_url": "https://x.example",
		})
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
}

func TestV1CreateFileResourceFromCompletedUpload(t *testing.T) {
	f := newToolsetFix(t, nil)
	resp, got := f.ts(t, http.MethodPost, "/api/v1/toolsets/"+idStr(g1TSMain)+"/resources",
		"/toolsets/{toolset_id}/resources", "sess-alice", keyUUID(23), map[string]any{
			"resource_type": "file", "artifact_id": g1UpAliceB,
		})
	if resp.StatusCode != http.StatusCreated || got["resource_type"] != "file" || got["archive"] == nil {
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

	resp, got = f.ts(t, http.MethodPost, "/api/v1/toolsets/"+idStr(g1TSBob)+"/resources/"+idStr(g1ResLink)+"/downloads",
		spec, "", "", nil)
	wantCode(t, resp, got, http.StatusNotFound, problem.CodeNotFound)
}
