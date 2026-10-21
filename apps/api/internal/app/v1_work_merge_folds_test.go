package app

import (
	"net/http"
	"net/url"
	"testing"
)

const (
	wmfOld = 930160001
	wmfNew = 930160002
)

func TestV1WorkMergeFolds(t *testing.T) {
	f := newPermFix(t)
	clean := func() { f.db.Exec(`DELETE FROM galgame_merge_fold WHERE old_work_id = ?`, wmfOld) }
	clean()
	t.Cleanup(clean)
	var foldID int64
	if err := f.db.Raw(`INSERT INTO galgame_merge_fold (old_work_id, new_work_id, folded_at, undone_at, undo_problems)
		VALUES (?, ?, '2026-10-21T08:00:00Z', '2026-10-22T08:00:00Z',
			'[{"table":"galgame_resource","key":77,"reason":"elsewhere","work_id":70000},{"table":"galgame_like","key":5,"reason":"purged_user"}]')
		RETURNING id`, wmfOld, wmfNew).Scan(&foldID).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Exec(`INSERT INTO galgame_merge_fold_row (fold_id, table_name, action, row_key)
		VALUES (?, 'galgame_resource', 'moved', '{"id":77}'), (?, 'galgame_resource', 'moved', '{"id":78}'),
		       (?, 'galgame_like', 'dropped', '{"id":5}')`, foldID, foldID, foldID).Error; err != nil {
		t.Fatal(err)
	}

	get := func(session string) (*http.Response, map[string]any) {
		return f.call(t, http.MethodGet, "/api/v1/admin/work-merge-folds?"+url.Values{"limit": {"100"}}.Encode(),
			"/admin/work-merge-folds", pmAuth{session: session}, nil)
	}
	for _, session := range []string{"sess-pm-mod", "sess-pm-plain"} {
		if resp, body := get(session); resp.StatusCode != http.StatusForbidden || body["code"] != "PERMISSION_REQUIRED" {
			t.Errorf("%s: %d %+v, want PERMISSION_REQUIRED", session, resp.StatusCode, body)
		}
	}

	resp, body := get("sess-pm-admin")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin: %d %+v", resp.StatusCode, body)
	}
	var fold map[string]any
	items, _ := body["items"].([]any)
	for _, it := range items {
		if m, _ := it.(map[string]any); strID(m["old_work_id"]) == idStr(wmfOld) {
			fold = m
		}
	}
	if fold == nil {
		t.Fatalf("fold missing from %+v", body)
	}
	if asInt(fold["moved_count"]) != 2 || asInt(fold["dropped_count"]) != 1 || fold["undone_at"] == nil {
		t.Errorf("fold %+v", fold)
	}
	problems, _ := fold["problems"].([]any)
	if len(problems) != 2 {
		t.Fatalf("problems %+v", problems)
	}
	first, _ := problems[0].(map[string]any)
	second, _ := problems[1].(map[string]any)
	if first["table_name"] != "galgame_resource" || first["row_id"] != "77" || first["cause"] != "elsewhere" || first["current_work_id"] != "70000" {
		t.Errorf("first problem %+v", first)
	}
	if second["cause"] != "purged_user" || second["current_work_id"] != nil {
		t.Errorf("second problem %+v", second)
	}
}
