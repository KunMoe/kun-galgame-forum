package app

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	adminModel "kun-galgame-api/internal/admin/model"
	updateRepo "kun-galgame-api/internal/update/repository"
)

func TestV1TodosWalk(t *testing.T) {
	f := newUpdateFix(t, nil)
	for _, c := range []struct{ query, where string }{
		{"", ""},
		{"?state=pending", "WHERE status = 0"},
		{"?state=in_progress", "WHERE status = 1"},
	} {
		want := f.sqlIDs(t, `SELECT id::text FROM todo `+c.where+` ORDER BY created DESC, id DESC`)
		for _, limit := range []int{2, 3} {
			got := f.walkIDs(t, "/api/v1/todos"+c.query, "/todos", "", limit)
			if fmt.Sprint(got) != fmt.Sprint(want) {
				t.Errorf("%q limit %d walked %v, want %v", c.query, limit, got, want)
			}
		}
	}
}

func TestV1TodosTotalAndFilter(t *testing.T) {
	f := newUpdateFix(t, nil)
	resp, body := f.up(t, http.MethodGet, "/api/v1/todos?state=done&include_total=true&limit=1", "/todos", "", nil, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	if want := f.scalar(t, `SELECT COUNT(*) FROM todo WHERE status = 2`); asInt(body["total"]) != want || want < 2 {
		t.Errorf("total %v, want %d", body["total"], want)
	}
	for _, it := range body["items"].([]any) {
		if m := it.(map[string]any); m["state"] != "done" {
			t.Errorf("filtered item %+v", m)
		}
	}
	_, body = f.up(t, http.MethodGet, "/api/v1/todos", "/todos", "", nil, "", nil)
	if _, has := body["total"]; has {
		t.Error("total without include_total")
	}

	_, first := f.up(t, http.MethodGet, "/api/v1/todos?state=pending&limit=1", "/todos", "", nil, "", nil)
	cur, _ := first["next_cursor"].(string)
	for _, c := range []struct{ url, code string }{
		{"/api/v1/todos?state=done&cursor=" + cur, "INVALID_CURSOR"},
		{"/api/v1/todos?state=claimed", "UNKNOWN_ENUM_VALUE"},
		{"/api/v1/todos?limit=101", "LIMIT_TOO_LARGE"},
	} {
		resp, body := f.up(t, http.MethodGet, c.url, "/todos", "", nil, "", nil)
		if resp.StatusCode != http.StatusBadRequest || body["code"] != c.code {
			t.Errorf("%s: %d %+v", c.url, resp.StatusCode, body)
		}
	}
}

func TestV1TodoShape(t *testing.T) {
	f := newUpdateFix(t, nil)
	body := f.getTodo(t, upTodoDone, "", nil)
	claimer, _ := body["claimer"].(map[string]any)
	author, _ := body["author"].(map[string]any)
	if body["object"] != "todo" || body["project"] != "forum" || body["state"] != "done" || body["viewer"] != nil ||
		strID(author["id"]) != fmt.Sprint(w3UserAlice) || author["name"] != "alice" ||
		claimer == nil || strID(claimer["id"]) != fmt.Sprint(w3UserStaff) || body["completed_at"] != "2026-05-03T09:00:00Z" {
		t.Errorf("done shape %+v", body)
	}
	body = f.getTodo(t, upTodoOrphan, "", nil)
	if body["state"] != "in_progress" || body["claimer"] != nil || body["completed_at"] != nil {
		t.Errorf("claimer-less in_progress %+v", body)
	}
	resp, got := f.up(t, http.MethodGet, "/api/v1/todos/930000698", "/todos/{todo_id}", "", nil, "", nil)
	if resp.StatusCode != http.StatusNotFound || got["code"] != "NOT_FOUND" {
		t.Errorf("missing %d %+v", resp.StatusCode, got)
	}
}

func TestV1TodoReadsSurviveOAuthOutage(t *testing.T) {
	f := newUpdateFix(t, nil)
	f.failOA.Store(true)
	body := f.getTodo(t, upTodoDone, "", nil)
	author, _ := body["author"].(map[string]any)
	claimer, _ := body["claimer"].(map[string]any)
	if author["name"] != nil || strID(author["id"]) != fmt.Sprint(w3UserAlice) || claimer["name"] != nil {
		t.Errorf("reads render name null when OAuth is down: %+v", body)
	}
}

func TestV1TodoCreate(t *testing.T) {
	f := newUpdateFix(t, nil)
	resp, body := f.up(t, http.MethodPost, "/api/v1/todos", "/todos", "sess-alice", nil, keyUUID(9201), todoBody("patch", "please add\na thing"))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, body)
	}
	id := strID(body["id"])
	viewer, _ := body["viewer"].(map[string]any)
	if resp.Header.Get("Location") != "/api/v1/todos/"+id || body["state"] != "pending" || body["claimer"] != nil ||
		body["project"] != "patch" || body["text"] != "please add\na thing" || viewer["can_edit"] != true || viewer["can_discard"] != true {
		t.Errorf("created %s %+v", resp.Header.Get("Location"), body)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM feed_activity WHERE type = 'TODO_CREATION' AND source_id = ?`, asInt(id)); n != 1 {
		t.Errorf("feed card %d", n)
	}

	resp, body = f.up(t, http.MethodPost, "/api/v1/todos", "/todos", "sess-alice", nil, "", todoBody("forum", "x"))
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_PARAMETER" {
		t.Errorf("no idempotency key %d %+v", resp.StatusCode, body)
	}
	resp, body = f.up(t, http.MethodPost, "/api/v1/todos", "/todos", "sess-alice", nil, keyUUID(9202), todoBody("forum", " \n "))
	if resp.StatusCode != http.StatusUnprocessableEntity || body["code"] != "VALIDATION_FAILED" {
		t.Errorf("blank %d %+v", resp.StatusCode, body)
	}
	resp, body = f.up(t, http.MethodPost, "/api/v1/todos", "/todos", "sess-alice", nil, keyUUID(9203), todoBody("forum", strings.Repeat("a", 1001)))
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("too long %d %+v", resp.StatusCode, body)
	}
	resp, body = f.up(t, http.MethodPost, "/api/v1/todos", "/todos", "sess-banned", nil, keyUUID(9204), todoBody("forum", "x"))
	if resp.StatusCode != http.StatusForbidden || body["code"] != "ACCOUNT_BANNED" {
		t.Errorf("banned %d %+v", resp.StatusCode, body)
	}
	resp, body = f.up(t, http.MethodPost, "/api/v1/todos", "/todos", "", nil, keyUUID(9205), todoBody("forum", "x"))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous %d %+v", resp.StatusCode, body)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM todo WHERE user_id = ?`, w3UserBanned); n != 0 {
		t.Error("a banned user's todo was written")
	}

	resp, body = f.up(t, http.MethodPost, "/api/v1/todos", "/todos", "sess-alice", nil, keyUUID(9201), todoBody("forum", "different"))
	if resp.StatusCode != http.StatusConflict || body["code"] != "IDEMPOTENCY_KEY_REUSED" {
		t.Errorf("key reuse %d %+v", resp.StatusCode, body)
	}
	resp, body = f.up(t, http.MethodPost, "/api/v1/todos", "/todos", "sess-alice", nil, keyUUID(9201), todoBody("patch", "please add\na thing"))
	if resp.StatusCode != http.StatusCreated || strID(body["id"]) != id || resp.Header.Get("Idempotency-Replayed") != "true" {
		t.Errorf("replay %d %+v", resp.StatusCode, body)
	}
}

func TestV1TodoWritesFailClosedWhenOAuthIsDown(t *testing.T) {
	f := newUpdateFix(t, nil)
	f.failOA.Store(true)
	resp, body := f.up(t, http.MethodPost, "/api/v1/todos", "/todos", "sess-alice", nil, keyUUID(9211), todoBody("forum", "x"))
	if resp.StatusCode != http.StatusServiceUnavailable || body["code"] != "SERVICE_UNAVAILABLE" {
		t.Errorf("create %d %+v", resp.StatusCode, body)
	}
	resp, body = f.patchTodo(t, upTodoPending, "sess-staff", stateBody("in_progress"))
	if resp.StatusCode != http.StatusServiceUnavailable || f.todoStatus(t, upTodoPending) != 0 {
		t.Errorf("claim %d %+v", resp.StatusCode, body)
	}
}

func TestV1TodoTrustCheck(t *testing.T) {
	f := newUpdateFix(t, denyChecker{})
	resp, body := f.up(t, http.MethodPost, "/api/v1/todos", "/todos", "sess-alice", nil, keyUUID(9221), todoBody("forum", "bad words"))
	if resp.StatusCode != http.StatusUnprocessableEntity || body["code"] != "CONTENT_REJECTED" {
		t.Errorf("create %d %+v", resp.StatusCode, body)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM todo WHERE user_id = ? AND content = 'bad words'`, w3UserAlice); n != 0 {
		t.Error("a refused todo was written")
	}
	resp, body = f.patchTodo(t, upTodoPending, "sess-alice", map[string]any{"text": "bad words"})
	if resp.StatusCode != http.StatusUnprocessableEntity || body["code"] != "CONTENT_REJECTED" {
		t.Errorf("edit %d %+v", resp.StatusCode, body)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM todo WHERE id = ? AND content = 'todo ' || id`, upTodoPending); n != 1 {
		t.Error("a refused edit changed the text")
	}
	// K18: no new text, nothing to judge.
	resp, body = f.patchTodo(t, upTodoPending, "sess-alice", map[string]any{"project": "patch", "text": fmt.Sprintf("todo %d", upTodoPending)})
	if resp.StatusCode != http.StatusOK || body["project"] != "patch" {
		t.Errorf("unchanged text is not rechecked: %d %+v", resp.StatusCode, body)
	}
}

func TestV1TodoEdit(t *testing.T) {
	f := newUpdateFix(t, nil)
	resp, body := f.patchTodo(t, upTodoClaimed, "sess-alice", map[string]any{"text": "rewritten"})
	if resp.StatusCode != http.StatusOK || body["text"] != "rewritten" || body["state"] != "in_progress" || body["project"] != "patch" {
		t.Errorf("author edits an in_progress todo: %d %+v", resp.StatusCode, body)
	}
	for _, c := range []struct {
		name    string
		id      int
		session string
		payload map[string]any
		status  int
		code    string
	}{
		{"moderator edits someone else's todo", upTodoPending, "sess-staff", map[string]any{"text": "mine now"}, 403, "PERMISSION_REQUIRED"},
		{"admin edits someone else's todo", upTodoPending, "sess-admin", map[string]any{"project": "patch"}, 403, "PERMISSION_REQUIRED"},
		{"author edits a done todo", upTodoDone, "sess-alice", map[string]any{"text": "late"}, 409, "INVALID_STATE_TRANSITION"},
		{"author edits a discarded todo", upTodoDiscarded, "sess-alice", map[string]any{"text": "late"}, 409, "INVALID_STATE_TRANSITION"},
		{"stranger edits a done todo", upTodoDone, "sess-bob", map[string]any{"text": "late"}, 409, "INVALID_STATE_TRANSITION"},
		{"blank text", upTodoPending, "sess-alice", map[string]any{"text": "  "}, 422, "VALIDATION_FAILED"},
		{"state with text", upTodoPending, "sess-alice", map[string]any{"state": "discarded", "text": "x"}, 422, "VALIDATION_FAILED"},
		{"missing todo", 930000698, "sess-alice", map[string]any{"text": "x"}, 404, "NOT_FOUND"},
	} {
		resp, body := f.patchTodo(t, c.id, c.session, c.payload)
		if resp.StatusCode != c.status || body["code"] != c.code {
			t.Errorf("%s: %d %+v", c.name, resp.StatusCode, body)
		}
	}
	_, body = f.patchTodo(t, upTodoPending, "sess-alice", map[string]any{"state": "discarded", "project": "patch"})
	if e := problemErrorsOf(body); len(e) != 1 || e[0]["pointer"] != "/state" || e[0]["reason"] != "INCONSISTENT_WITH" {
		t.Errorf("state with another field %+v", e)
	}
	if f.todoStatus(t, upTodoPending) != 0 {
		t.Error("a refused mixed patch moved the todo")
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM todo WHERE id = ? AND content = 'todo ' || id AND type = 'forum'`, upTodoPending); n != 1 {
		t.Error("a refused edit wrote")
	}
}

func TestV1TodoTransitions(t *testing.T) {
	f := newUpdateFix(t, nil)

	resp, body := f.patchTodo(t, upTodoPending, "sess-alice", stateBody("in_progress"))
	if resp.StatusCode != http.StatusForbidden || body["code"] != "PERMISSION_REQUIRED" || f.todoStatus(t, upTodoPending) != 0 {
		t.Errorf("a plain user claims: %d %+v", resp.StatusCode, body)
	}
	bearer := http.Header{"Authorization": {"Bearer staff-token"}}
	resp, body = f.up(t, http.MethodPatch, fmt.Sprintf("/api/v1/todos/%d", upTodoPending), "/todos/{todo_id}", "", bearer, "", stateBody("in_progress"))
	if resp.StatusCode != http.StatusForbidden || f.todoStatus(t, upTodoPending) != 0 {
		t.Errorf("a Bearer moderator claims: %d %+v", resp.StatusCode, body)
	}

	resp, body = f.patchTodo(t, upTodoPending, "sess-staff", stateBody("in_progress"))
	claimer, _ := body["claimer"].(map[string]any)
	if resp.StatusCode != http.StatusOK || body["state"] != "in_progress" || claimer == nil || strID(claimer["id"]) != fmt.Sprint(w3UserStaff) {
		t.Fatalf("claim %d %+v", resp.StatusCode, body)
	}
	if f.todoClaimer(t, upTodoPending) != w3UserStaff {
		t.Error("claim did not record the claimer")
	}
	resp, body = f.patchTodo(t, upTodoPending, "sess-admin", stateBody("in_progress"))
	if resp.StatusCode != http.StatusConflict || body["code"] != "INVALID_STATE_TRANSITION" || f.todoClaimer(t, upTodoPending) != w3UserStaff {
		t.Errorf("claiming a claimed todo %d %+v", resp.StatusCode, body)
	}

	resp, body = f.patchTodo(t, upTodoPending, "sess-admin", stateBody("pending"))
	if resp.StatusCode != http.StatusForbidden || f.todoStatus(t, upTodoPending) != 1 {
		t.Errorf("a non-claimer editor releases: %d %+v", resp.StatusCode, body)
	}
	resp, body = f.patchTodo(t, upTodoPending, "sess-staff", stateBody("pending"))
	if resp.StatusCode != http.StatusOK || body["state"] != "pending" || body["claimer"] != nil || f.todoClaimer(t, upTodoPending) != 0 {
		t.Errorf("the claimer releases: %d %+v", resp.StatusCode, body)
	}

	resp, body = f.patchTodo(t, upTodoClaimed, "sess-bob", stateBody("done"))
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("a stranger completes: %d %+v", resp.StatusCode, body)
	}
	resp, body = f.patchTodo(t, upTodoClaimed, "sess-admin", stateBody("done"))
	if resp.StatusCode != http.StatusOK || body["state"] != "done" || body["completed_at"] == nil ||
		f.todoClaimer(t, upTodoClaimed) != w3UserStaff {
		t.Errorf("a non-claimer editor completes: %d %+v", resp.StatusCode, body)
	}
	for _, target := range []string{"pending", "discarded", "in_progress", "done"} {
		resp, body = f.patchTodo(t, upTodoClaimed, "sess-admin", stateBody(target))
		if resp.StatusCode != http.StatusConflict || body["code"] != "INVALID_STATE_TRANSITION" || f.todoStatus(t, upTodoClaimed) != 2 {
			t.Errorf("done is final (%s): %d %+v", target, resp.StatusCode, body)
		}
	}

	resp, body = f.patchTodo(t, upTodoDiscarded, "sess-staff", stateBody("pending"))
	if resp.StatusCode != http.StatusForbidden || f.todoStatus(t, upTodoDiscarded) != 3 {
		t.Errorf("a moderator reopens: %d %+v", resp.StatusCode, body)
	}
	resp, body = f.patchTodo(t, upTodoDiscarded, "sess-admin", stateBody("pending"))
	if resp.StatusCode != http.StatusOK || body["state"] != "pending" || body["claimer"] != nil || f.todoClaimer(t, upTodoDiscarded) != 0 {
		t.Errorf("an admin reopens: %d %+v", resp.StatusCode, body)
	}

	resp, body = f.patchTodo(t, upTodoBobs, "sess-alice", stateBody("discarded"))
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("a stranger discards a pending todo: %d %+v", resp.StatusCode, body)
	}
	resp, body = f.patchTodo(t, upTodoBobs, "sess-staff", stateBody("discarded"))
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("an editor discards someone's pending todo: %d %+v", resp.StatusCode, body)
	}
	resp, body = f.patchTodo(t, upTodoBobs, "sess-bob", stateBody("discarded"))
	if resp.StatusCode != http.StatusOK || body["state"] != "discarded" {
		t.Errorf("the author discards: %d %+v", resp.StatusCode, body)
	}
	resp, body = f.patchTodo(t, upTodoBobs, "sess-bob", stateBody("discarded"))
	if resp.StatusCode != http.StatusConflict || body["code"] != "INVALID_STATE_TRANSITION" {
		t.Errorf("moving to the current state: %d %+v", resp.StatusCode, body)
	}

	resp, body = f.patchTodo(t, upTodoOrphan, "sess-staff", stateBody("discarded"))
	if resp.StatusCode != http.StatusOK || body["state"] != "discarded" {
		t.Errorf("an editor discards a claimer-less in_progress todo: %d %+v", resp.StatusCode, body)
	}
}

func TestV1TodoTransitionIsGuarded(t *testing.T) {
	f := newUpdateFix(t, nil)
	store := updateRepo.NewStore(f.db)
	moved, err := store.TransitionTodo(upTodoClaimed, adminModel.TodoStatusPending, adminModel.TodoStatusClaimed,
		map[string]any{"claimed_user_id": w3UserGrant})
	if err != nil || moved {
		t.Fatalf("a transition from a stale state moved the row: moved=%v err=%v", moved, err)
	}
	if f.todoStatus(t, upTodoClaimed) != 1 || f.todoClaimer(t, upTodoClaimed) != w3UserStaff {
		t.Error("the stale transition rewrote the row")
	}
}

// Every viewer flag must be exactly the gate updateTodo applies: for each
// caller and each todo, a flag is true iff the PATCH it stands for succeeds.
func TestV1TodoViewerMatchesTheGates(t *testing.T) {
	sessions := []string{"sess-alice", "sess-bob", "sess-staff", "sess-admin"}
	flags := []struct {
		flag    string
		payload map[string]any
	}{
		{"can_claim", stateBody("in_progress")},
		{"can_complete", stateBody("done")},
		{"can_discard", stateBody("discarded")},
		{"can_release", stateBody("pending")},
		{"can_reopen", stateBody("pending")},
		{"can_edit", map[string]any{"text": "edited"}},
	}
	f := newUpdateFix(t, nil)
	var seeded []adminModel.Todo
	if err := f.db.Where("id BETWEEN ? AND ?", upTodoMin, upTodoMax).Find(&seeded).Error; err != nil {
		t.Fatal(err)
	}
	restore := func(id int) {
		t.Helper()
		for _, s := range seeded {
			if s.ID != id {
				continue
			}
			if err := f.db.Exec(`UPDATE todo SET status = ?, claimed_user_id = ?, completed_time = ?, content = ?, type = ? WHERE id = ?`,
				s.Status, s.ClaimedUserID, s.CompletedTime, s.Content, s.Type, id).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, id := range []int{upTodoPending, upTodoClaimed, upTodoDone, upTodoDiscarded, upTodoOrphan} {
		for _, session := range sessions {
			viewer, _ := f.getTodo(t, id, session, nil)["viewer"].(map[string]any)
			status := f.todoStatus(t, id)
			for _, fl := range flags {
				if fl.flag == "can_release" && status != adminModel.TodoStatusClaimed ||
					fl.flag == "can_reopen" && status != adminModel.TodoStatusDiscarded {
					if viewer[fl.flag] != false {
						t.Errorf("todo %d %s: %s outside its state", id, session, fl.flag)
					}
					continue
				}
				resp, _ := f.patchTodo(t, id, session, fl.payload)
				restore(id)
				if viewer[fl.flag] != (resp.StatusCode == http.StatusOK) {
					t.Errorf("todo %d %s: %s=%v but PATCH answered %d", id, session, fl.flag, viewer[fl.flag], resp.StatusCode)
				}
			}
		}
	}
}

func TestV1TodoDelete(t *testing.T) {
	f := newUpdateFix(t, nil)
	url := fmt.Sprintf("/api/v1/todos/%d", upTodoDone)
	resp, body := f.up(t, http.MethodDelete, url, "/todos/{todo_id}", "sess-alice", nil, "", nil)
	if resp.StatusCode != http.StatusForbidden || body["code"] != "PERMISSION_REQUIRED" {
		t.Errorf("the author deletes: %d %+v", resp.StatusCode, body)
	}
	bearer := http.Header{"Authorization": {"Bearer staff-token"}}
	resp, _ = f.up(t, http.MethodDelete, url, "/todos/{todo_id}", "", bearer, "", nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("a Bearer moderator deletes: %d", resp.StatusCode)
	}
	viewer, _ := f.getTodo(t, upTodoDone, "sess-staff", nil)["viewer"].(map[string]any)
	if viewer["can_delete"] != true {
		t.Errorf("staff viewer %+v", viewer)
	}
	resp, _ = f.up(t, http.MethodDelete, url, "/todos/{todo_id}", "sess-staff", nil, "", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete %d", resp.StatusCode)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM todo WHERE id = ?`, upTodoDone); n != 0 {
		t.Error("the row is still there")
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM feed_activity WHERE type = 'TODO_CREATION' AND source_id = ?`, upTodoDone); n != 0 {
		t.Error("the feed card is still there")
	}
	resp, body = f.up(t, http.MethodDelete, url, "/todos/{todo_id}", "sess-staff", nil, "", nil)
	if resp.StatusCode != http.StatusNotFound || body["code"] != "NOT_FOUND" {
		t.Errorf("delete again %d %+v", resp.StatusCode, body)
	}
}
