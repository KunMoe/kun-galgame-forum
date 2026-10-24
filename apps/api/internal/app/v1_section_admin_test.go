package app

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
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
	}
	type groupRow struct {
		ID       int
		Label    string
		Position int
	}
	var sections []sectionRow
	var groups []groupRow
	if err := f.db.Raw(`SELECT id, label, description, icon, color, position, is_archived FROM topic_section`).Scan(&sections).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Raw(`SELECT id, label, position FROM topic_section_group`).Scan(&groups).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, s := range sections {
			if err := f.db.Exec(`UPDATE topic_section SET label = ?, description = ?, icon = ?, color = ?, position = ?, is_archived = ? WHERE id = ?`,
				s.Label, s.Description, s.Icon, s.Color, s.Position, s.IsArchived, s.ID).Error; err != nil {
				t.Errorf("restore section %d: %v", s.ID, err)
			}
		}
		for _, g := range groups {
			if err := f.db.Exec(`UPDATE topic_section_group SET label = ?, position = ? WHERE id = ?`, g.Label, g.Position, g.ID).Error; err != nil {
				t.Errorf("restore group %d: %v", g.ID, err)
			}
		}
	})
	return f
}

type sectionGroupOrder struct {
	Group    string   `json:"group"`
	Sections []string `json:"sections"`
}

func listedOrder(t *testing.T, body map[string]any) []sectionGroupOrder {
	t.Helper()
	var out []sectionGroupOrder
	items, _ := body["items"].([]any)
	for _, it := range items {
		m, _ := it.(map[string]any)
		g, _ := m["group"].(string)
		if len(out) == 0 || out[len(out)-1].Group != g {
			out = append(out, sectionGroupOrder{Group: g})
		}
		out[len(out)-1].Sections = append(out[len(out)-1].Sections, fmt.Sprint(m["section"]))
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
		{http.MethodPatch, "/api/v1/admin/section-groups/galgame", "/admin/section-groups/{group}", map[string]any{"label": "x"}},
		{http.MethodPut, "/api/v1/admin/section-order", "/admin/section-order", map[string]any{"groups": []sectionGroupOrder{
			{"galgame", []string{"galgame-discussion", "galgame-walkthrough", "review", "news"}},
			{"resource", []string{"resource-seeking", "resource-release"}},
			{"technique", []string{"tech-help", "game-running", "development"}},
			{"community", []string{"acgn", "daily", "site"}},
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
		map[string]any{"label": "  评测  ", "is_archived": true})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("patch %d %+v", resp.StatusCode, body)
	}
	resp, body = f.docCall(t, http.MethodPatch, "/api/v1/admin/section-groups/community", "/admin/section-groups/{group}", "sess-admin", "", nil,
		map[string]any{"label": "社区"})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("group patch %d %+v", resp.StatusCode, body)
	}
	_, body = f.sectionList(t, url.Values{})
	review := sectionByName(t, body, "review")
	if review["label"] != "评测" || review["is_archived"] != true {
		t.Errorf("review after patch %+v", review)
	}
	if acgn := sectionByName(t, body, "acgn"); acgn["group_label"] != "社区" {
		t.Errorf("acgn group_label %v", acgn["group_label"])
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
	if len(current) != 4 {
		t.Fatalf("groups %+v", current)
	}
	put := func(groups []sectionGroupOrder) (*http.Response, map[string]any) {
		t.Helper()
		return f.docCall(t, http.MethodPut, "/api/v1/admin/section-order", "/admin/section-order", "sess-admin", "", nil,
			map[string]any{"groups": groups})
	}

	moved := make([]sectionGroupOrder, len(current))
	for i, g := range current {
		moved[i] = sectionGroupOrder{g.Group, slices.Clone(g.Sections)}
	}
	var community, technique int
	for i, g := range moved {
		switch g.Group {
		case "community":
			community = i
		case "technique":
			technique = i
		}
	}
	moved[community].Sections = slices.DeleteFunc(moved[community].Sections, func(s string) bool { return s == "acgn" })
	moved[technique].Sections = append(moved[technique].Sections, "acgn")
	resp, body := put(moved)
	if resp.StatusCode != 422 || body["code"] != problem.CodeValidationFailed {
		t.Fatalf("section moved to another group %d %+v", resp.StatusCode, body)
	}
	pointers := map[string]bool{}
	for _, e := range problemErrorsOf(body) {
		pointers[fmt.Sprint(e["pointer"])] = true
	}
	if !pointers[fmt.Sprintf("/groups/%d/sections", technique)] || !pointers[fmt.Sprintf("/groups/%d/sections", community)] {
		t.Errorf("errors %v", problemErrorsOf(body))
	}

	want := make([]sectionGroupOrder, 0, len(current))
	for _, g := range slices.Backward(current) {
		secs := slices.Clone(g.Sections)
		slices.Reverse(secs)
		want = append(want, sectionGroupOrder{g.Group, secs})
	}
	if resp, body := put(want); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("valid order %d %+v", resp.StatusCode, body)
	}
	_, body = f.sectionList(t, url.Values{})
	if got := listedOrder(t, body); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("order %v, want %v", got, want)
	}
}
