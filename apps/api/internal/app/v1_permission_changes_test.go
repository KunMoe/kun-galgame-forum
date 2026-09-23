package app

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func (f *permFix) seedChanges(t *testing.T) {
	t.Helper()
	tie := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	for i := 0; i < 7; i++ {
		subjectKind, subject := "role", "moderator"
		if i%2 == 1 {
			subjectKind, subject = "user", fmt.Sprint(pmModTarget)
		}
		if err := f.db.Exec(`INSERT INTO permission_audit_log (operator_id, subject_kind, subject, action, before_rows, after_rows, created_at)
			VALUES (?, ?, ?, 'replace', '[]', '[{"permission":"doc.edit","effect":"revoke"}]', ?)`,
			pmAdmin, subjectKind, subject, tie).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i := 1; i <= 3; i++ {
		if err := f.db.Exec(`INSERT INTO permission_audit_log (operator_id, subject_kind, subject, action, before_rows, after_rows, created_at)
			VALUES (?, 'role', 'creator', 'reset', '[{"permission":"topic.hide","effect":"grant"}]', '[]', ?)`,
			pmRen, tie.Add(time.Duration(i)*time.Hour)).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func (f *permFix) changes(t *testing.T, session string, q url.Values) (*http.Response, map[string]any) {
	t.Helper()
	return f.call(t, http.MethodGet, pmChangesPath+"?"+q.Encode(), "/admin/permission-changes", pmAuth{session: session}, nil)
}

func TestV1PermChangesWalk(t *testing.T) {
	f := newPermFix(t)
	f.seedChanges(t)
	var want []string
	rows, err := f.db.Raw(`SELECT id::text FROM permission_audit_log ORDER BY created_at DESC, id DESC`).Rows()
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		want = append(want, id)
	}
	_ = rows.Close()
	for _, limit := range []int{2, 3} {
		var got []string
		for page := 1; page <= 20; page++ {
			resp, body := f.changes(t, "sess-pm-admin", url.Values{"page": {fmt.Sprint(page)}, "limit": {fmt.Sprint(limit)}})
			if resp.StatusCode != http.StatusOK || asInt(body["total"]) != len(want) || body["total_relation"] != "eq" {
				t.Fatalf("page %d: %d %+v", page, resp.StatusCode, body)
			}
			items, _ := body["items"].([]any)
			if len(items) == 0 {
				break
			}
			for _, it := range items {
				m, _ := it.(map[string]any)
				got = append(got, strID(m["id"]))
			}
		}
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("limit %d walked %v, want %v", limit, got, want)
		}
	}
}

func TestV1PermChangesShape(t *testing.T) {
	f := newPermFix(t)
	f.seedChanges(t)
	resp, body := f.changes(t, "sess-pm-admin", url.Values{"limit": {"100"}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	items, _ := body["items"].([]any)
	var roles, users int
	for _, it := range items {
		m, _ := it.(map[string]any)
		actor, _ := m["actor"].(map[string]any)
		if m["object"] != "permission_change" || actor["name"] == nil {
			t.Errorf("item %+v", m)
		}
		switch {
		case m["target_role"] != nil && m["target_user"] == nil:
			roles++
		case m["target_user"] != nil && m["target_role"] == nil:
			target, _ := m["target_user"].(map[string]any)
			if strID(target["id"]) != fmt.Sprint(pmModTarget) {
				t.Errorf("target %+v", target)
			}
			users++
		default:
			t.Errorf("exactly one target: %+v", m)
		}
	}
	if roles != 7 || users != 3 {
		t.Errorf("%d role and %d user changes", roles, users)
	}

	resp, body = f.changes(t, "sess-pm-admin", url.Values{"page": {"101"}, "limit": {"100"}})
	mustCode(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
	resp, body = f.changes(t, "sess-pm-mod", url.Values{})
	mustCode(t, resp, body, http.StatusForbidden, "PERMISSION_REQUIRED")
}

func TestV1PermChangesFailClosed(t *testing.T) {
	f := newPermFix(t)
	f.seedChanges(t)
	f.oaDown.Store(true)
	resp, body := f.changes(t, "sess-pm-admin", url.Values{})
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}
