package app

import (
	"net/http"
	"slices"
	"testing"

	"kun-galgame-api/pkg/problem"
)

const rlEd2kLink = "ed2k://|file|纯白交响曲 Remake.rar|1234567890|0123456789ABCDEF0123456789abcdef|h=QWERTYUIOPASDFGHJKLZXCVBNM234567|/"

func TestV1UpdateGalgameResourceVersionLabel(t *testing.T) {
	f := newResourceFix(t, nil)
	patch := func(body map[string]any) (*http.Response, map[string]any) {
		return f.rs(t, http.MethodPatch, "/api/v1/galgame-resources/"+idStr(g3ResMain),
			"/galgame-resources/{resource_id}", "sess-alice", "", body)
	}
	stored := func() string {
		var v string
		if err := f.db.Raw(`SELECT version_label FROM galgame_resource WHERE id = ?`, g3ResMain).Scan(&v).Error; err != nil {
			t.Fatal(err)
		}
		return v
	}

	resp, got := patch(map[string]any{"title": "Renamed"})
	if resp.StatusCode != http.StatusOK || got["version_label"] != "official_latest" || stored() != "官方最新" {
		t.Fatalf("absent version_label must keep it: %d %v %q", resp.StatusCode, got["version_label"], stored())
	}

	resp, got = patch(map[string]any{"version_label": nil})
	if resp.StatusCode != http.StatusOK || got["version_label"] != nil || stored() != "" {
		t.Fatalf("null must clear it: %d %v %q", resp.StatusCode, got["version_label"], stored())
	}

	resp, got = patch(map[string]any{"version_label": nil})
	if resp.StatusCode != http.StatusOK || got["version_label"] != nil || stored() != "" {
		t.Fatalf("clearing an empty label again: %d %v %q", resp.StatusCode, got["version_label"], stored())
	}

	resp, got = patch(map[string]any{"version_label": "stable"})
	if resp.StatusCode != http.StatusOK || got["version_label"] != "stable" || stored() != "稳定版" {
		t.Fatalf("set: %d %v %q", resp.StatusCode, got["version_label"], stored())
	}

	resp, got = patch(map[string]any{"version_label": "latest"})
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if stored() != "稳定版" {
		t.Errorf("an unknown token was written: %q", stored())
	}
}

func TestV1CreateWorkResourceEd2kLink(t *testing.T) {
	f := newResourceFix(t, nil)
	urls := []string{rlEd2kLink, "https://files.example.invalid/a(1).zip"}
	resp, got := f.rs(t, http.MethodPost, "/api/v1/works/"+idStr(g3WorkSFW)+"/resources",
		"/works/{work_id}/resources", "sess-alice", keyUUID(40), createResourceBody(map[string]any{"download_urls": urls}))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create with an ed2k link %d %+v", resp.StatusCode, got)
	}
	if names, _ := got["provider_names"].([]any); len(names) != 2 || names[0] != "电驴下载" || names[1] != "files.example.invalid" {
		t.Errorf("provider names %v", got["provider_names"])
	}
	id := strID(got["id"])
	resp, got = f.rs(t, http.MethodGet, "/api/v1/galgame-resources/"+id+"/source",
		"/galgame-resources/{resource_id}/source", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("source %d %+v", resp.StatusCode, got)
	}
	var stored []string
	for _, u := range got["download_urls"].([]any) {
		stored = append(stored, u.(string))
	}
	if !slices.Equal(stored, urls) {
		t.Errorf("stored links %q, want %q", stored, urls)
	}

	resp, got = f.rs(t, http.MethodPatch, "/api/v1/galgame-resources/"+id,
		"/galgame-resources/{resource_id}", "sess-alice", "",
		map[string]any{"download_urls": []string{"ed2k://|file|纯白交响曲.rar|1|NOTAHASH|/"}})
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if e := errorAt(got, "/download_urls/0"); e == nil || e["reason"] != problem.ReasonInvalidFormat {
		t.Errorf("malformed ed2k %+v", got["errors"])
	}
}

func TestV1CreateToolsetResourceEd2kLink(t *testing.T) {
	f := newToolsetFix(t, nil)
	resp, got := f.ts(t, http.MethodPost, "/api/v1/toolsets/"+idStr(g1TSMain)+"/resources",
		"/toolsets/{toolset_id}/resources", "sess-other", keyUUID(40), map[string]any{
			"toolset_resource_type": "link",
			"link_url":              rlEd2kLink,
			"size_label":            "12mb",
		})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create ed2k link %d %+v", resp.StatusCode, got)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_toolset_resource WHERE id = ? AND content = ?`, asInt(strID(got["id"])), rlEd2kLink); n != 1 {
		t.Error("ed2k link not stored as sent")
	}
}
