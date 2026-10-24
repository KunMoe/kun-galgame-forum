package app

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"kun-galgame-api/pkg/problem"
)

func newSectionAdminFix(t *testing.T) *writeFix {
	t.Helper()
	f := newWriteFix(t, nil)
	f.alice(t)
	f.putSession(t, "sess-admin", w3UserGrant, "user", "admin")
	type sectionRow struct {
		ID                              int
		Label, Description, Icon, Color string
		Position                        int
		IsArchived                      bool
		BannerImageHash                 string
		MascotImageHashes               string
	}
	var sections []sectionRow
	if err := f.db.Raw(`SELECT id, label, description, icon, color, position, is_archived, banner_image_hash, mascot_image_hashes::text AS mascot_image_hashes FROM topic_section`).Scan(&sections).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, s := range sections {
			if err := f.db.Exec(`UPDATE topic_section SET label = ?, description = ?, icon = ?, color = ?, position = ?, is_archived = ?, banner_image_hash = ?, mascot_image_hashes = ?::jsonb WHERE id = ?`,
				s.Label, s.Description, s.Icon, s.Color, s.Position, s.IsArchived, s.BannerImageHash, s.MascotImageHashes, s.ID).Error; err != nil {
				t.Errorf("restore section %d: %v", s.ID, err)
			}
		}
	})
	return f
}

func listedOrder(t *testing.T, body map[string]any) []string {
	t.Helper()
	var out []string
	items, _ := body["items"].([]any)
	for _, it := range items {
		m, _ := it.(map[string]any)
		out = append(out, fmt.Sprint(m["section"]))
	}
	return out
}

func TestV1SectionAdminNeedsManageSections(t *testing.T) {
	f := newSectionAdminFix(t)
	for _, c := range []struct {
		method, path, spec string
		body               any
	}{
		{http.MethodPatch, "/api/v1/admin/sections/news", "/admin/sections/{section}", map[string]any{"label": "x"}},
		{http.MethodPut, "/api/v1/admin/section-order", "/admin/section-order", map[string]any{"sections": []string{
			"galgame-discussion", "galgame-walkthrough", "review", "news", "resource-seeking", "resource-release",
			"tech-help", "game-running", "development", "acgn", "daily", "site",
		}}},
	} {
		resp, body := f.docCall(t, c.method, c.path, c.spec, "sess-staff", "", nil, c.body)
		if resp.StatusCode != http.StatusForbidden || body["code"] != problem.CodePermissionRequired {
			t.Errorf("moderator %s %s: %d %+v", c.method, c.path, resp.StatusCode, body)
		}
	}
	var label string
	f.db.Raw(`SELECT label FROM topic_section WHERE name = 'news'`).Scan(&label)
	if label == "x" {
		t.Error("a moderator's patch was written")
	}
}

func TestV1SectionAdminPatchShowsInList(t *testing.T) {
	f := newSectionAdminFix(t)
	if _, body := f.sectionList(t, url.Values{}); sectionByName(t, body, "review")["is_archived"] != false {
		t.Fatalf("review starts archived: %+v", sectionByName(t, body, "review"))
	}
	resp, body := f.docCall(t, http.MethodPatch, "/api/v1/admin/sections/review", "/admin/sections/{section}", "sess-admin", "", nil,
		map[string]any{
			"label": "  评测  ", "is_archived": true,
			"banner_image_hash":   strings.Repeat("ab", 32),
			"mascot_image_hashes": []string{strings.Repeat("cd", 32), strings.Repeat("ef", 32)},
		})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("patch %d %+v", resp.StatusCode, body)
	}
	_, body = f.sectionList(t, url.Values{})
	review := sectionByName(t, body, "review")
	if review["label"] != "评测" || review["is_archived"] != true {
		t.Errorf("review after patch %+v", review)
	}
	banner, _ := review["banner"].(map[string]any)
	mascots, _ := review["mascots"].([]any)
	if banner["hash"] != strings.Repeat("ab", 32) || len(mascots) != 2 {
		t.Errorf("review art banner %v mascots %v", review["banner"], review["mascots"])
	}
	resp, _ = f.docCall(t, http.MethodPatch, "/api/v1/admin/sections/review", "/admin/sections/{section}", "sess-admin", "", nil,
		map[string]any{"banner_image_hash": "", "mascot_image_hashes": []string{}})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("clear art %d", resp.StatusCode)
	}
	_, body = f.sectionList(t, url.Values{})
	if review = sectionByName(t, body, "review"); review["banner"] != nil || len(review["mascots"].([]any)) != 0 {
		t.Errorf("review art after clearing banner %v mascots %v", review["banner"], review["mascots"])
	}

	resp, raw := f.doJSON(t, http.MethodPost, "/api/v1/topics", "sess-alice", "/topics", keyUUID(900), nil, map[string]any{
		"title": "x", "content_markdown": "y", "sections": []string{"review"}, "is_nsfw": false, "access_scope": "public",
	})
	if errs := problemErrors(t, raw); resp.StatusCode != 422 || len(errs) == 0 ||
		errs[0]["pointer"] != "/sections/0" || errs[0]["reason"] != problem.ReasonNotAllowedValue {
		t.Errorf("create in a section archived by an admin: %d %s", resp.StatusCode, raw)
	}

	resp, body = f.docCall(t, http.MethodPatch, "/api/v1/admin/sections/review", "/admin/sections/{section}", "sess-admin", "", nil,
		map[string]any{"label": "   "})
	if resp.StatusCode != 422 || body["code"] != problem.CodeValidationFailed {
		t.Errorf("blank label %d %+v", resp.StatusCode, body)
	}
}

func TestV1SectionAdminReorder(t *testing.T) {
	f := newSectionAdminFix(t)
	_, body := f.sectionList(t, url.Values{})
	current := listedOrder(t, body)
	if len(current) != 12 {
		t.Fatalf("sections %v", current)
	}
	put := func(sections []string) (*http.Response, map[string]any) {
		t.Helper()
		return f.docCall(t, http.MethodPut, "/api/v1/admin/section-order", "/admin/section-order", "sess-admin", "", nil,
			map[string]any{"sections": sections})
	}
	if resp, body := put(current[1:]); resp.StatusCode != 422 || body["code"] != problem.CodeValidationFailed {
		t.Fatalf("a section left out %d %+v", resp.StatusCode, body)
	}
	if resp, body := put(append(slices.Clone(current[1:]), "g-news")); resp.StatusCode != 422 {
		t.Fatalf("a retired key %d %+v", resp.StatusCode, body)
	}
	want := slices.Clone(current)
	slices.Reverse(want)
	if resp, body := put(want); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("valid order %d %+v", resp.StatusCode, body)
	}
	_, body = f.sectionList(t, url.Values{})
	if got := listedOrder(t, body); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("order %v, want %v", got, want)
	}
}
