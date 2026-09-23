package app

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"kun-galgame-api/internal/trust/gate"
)

const (
	upLogMin  = 930000501
	upLogTie  = 930000510
	upLogMax  = 930000599
	upTodoMin = 930000601
	upTodoTie = 930000620
	upTodoMax = 930000699

	upTodoPending   = 930000601
	upTodoClaimed   = 930000602
	upTodoDone      = 930000603
	upTodoDiscarded = 930000604
	upTodoOrphan    = 930000605
	upTodoBobs      = 930000606
)

// newUpdateFix seeds both collections with seven rows that share one created
// instant, so a page boundary lands inside the tie and only the id
// tie-breaker keeps a cursor walk stable.
func newUpdateFix(t *testing.T, checker gate.Checker) *writeFix {
	t.Helper()
	f := newWriteFix(t, checker)
	f.alice(t)
	f.putSession(t, "sess-admin", w3UserGrant, "user", "admin")
	f.putSession(t, "sess-banned", w3UserBanned)
	f.cleanupUpdate(t)
	t.Cleanup(func() { f.cleanupUpdate(t) })

	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, q)
		}
	}
	base := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	tie := time.Date(2026, 5, 2, 9, 0, 0, 123456000, time.UTC)
	types := []string{"feat", "perf", "fix", "style", "mod", "chore", "sec", "refactor", "docs", "test"}
	for i := 0; i < 5; i++ {
		at := base.Add(time.Duration(i) * time.Hour)
		run(`INSERT INTO update_log (id, type, version, content, user_id, created, updated) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			upLogMin+i, types[i], fmt.Sprintf("4.0.%d", i), fmt.Sprintf("log %d\nsecond line", i), w3UserStaff, at, at)
	}
	for i := 0; i < 7; i++ {
		run(`INSERT INTO update_log (id, type, version, content, user_id, created, updated) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			upLogTie+i, types[5+i%5], "4.1.0", fmt.Sprintf("tied %d", i), w3UserStaff, tie, tie)
	}

	claimer := w3UserStaff
	todo := func(id, author, status int, typ string, claimedBy *int, completed *time.Time, at time.Time) {
		t.Helper()
		run(`INSERT INTO todo (id, type, status, content, completed_time, claimed_user_id, user_id, created, updated)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, typ, status, fmt.Sprintf("todo %d", id), completed, claimedBy, author, at, at)
	}
	done := base.Add(48 * time.Hour)
	todo(upTodoPending, w3UserAlice, 0, "forum", nil, nil, base)
	todo(upTodoClaimed, w3UserAlice, 1, "patch", &claimer, nil, base.Add(time.Hour))
	todo(upTodoDone, w3UserAlice, 2, "forum", &claimer, &done, base.Add(2*time.Hour))
	todo(upTodoDiscarded, w3UserAlice, 3, "forum", &claimer, nil, base.Add(3*time.Hour))
	todo(upTodoOrphan, w3UserAlice, 1, "forum", nil, nil, base.Add(4*time.Hour))
	todo(upTodoBobs, w3UserBob, 0, "patch", nil, nil, base.Add(5*time.Hour))
	for i := 0; i < 7; i++ {
		todo(upTodoTie+i, w3UserBob, i%4, "forum", nil, nil, tie)
	}
	return f
}

func (f *writeFix) cleanupUpdate(t *testing.T) {
	t.Helper()
	_ = f.db.Exec(`DELETE FROM todo WHERE user_id BETWEEN 930000001 AND 930000999 OR id BETWEEN ? AND ?`, upTodoMin, upTodoMax).Error
	_ = f.db.Exec(`DELETE FROM update_log WHERE user_id BETWEEN 930000001 AND 930000999 OR id BETWEEN ? AND ?`, upLogMin, upLogMax).Error
}

func (f *writeFix) up(t *testing.T, method, rawURL, specPath, session string, hdr http.Header, idem string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	resp, body := f.doJSON(t, method, rawURL, session, specPath, idem, hdr, payload)
	if len(body) == 0 {
		return resp, nil
	}
	return resp, problemMap(t, body)
}

func (f *writeFix) patchTodo(t *testing.T, id int, session string, payload any) (*http.Response, map[string]any) {
	t.Helper()
	return f.up(t, http.MethodPatch, fmt.Sprintf("/api/v1/todos/%d", id), "/todos/{todo_id}", session, nil, "", payload)
}

func (f *writeFix) getTodo(t *testing.T, id int, session string, hdr http.Header) map[string]any {
	t.Helper()
	resp, body := f.up(t, http.MethodGet, fmt.Sprintf("/api/v1/todos/%d", id), "/todos/{todo_id}", session, hdr, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get todo %d: %d %+v", id, resp.StatusCode, body)
	}
	return body
}

func (f *writeFix) todoStatus(t *testing.T, id int) int {
	t.Helper()
	return f.scalar(t, `SELECT status FROM todo WHERE id = ?`, id)
}

func (f *writeFix) todoClaimer(t *testing.T, id int) int {
	t.Helper()
	return f.scalar(t, `SELECT COALESCE(claimed_user_id, 0) FROM todo WHERE id = ?`, id)
}

func (f *writeFix) walkIDs(t *testing.T, base, spec, session string, limit int) []string {
	t.Helper()
	var got []string
	cursor := ""
	for page := 0; page < 200; page++ {
		url := fmt.Sprintf("%s%slimit=%d", base, sep(base), limit)
		if cursor != "" {
			url += "&cursor=" + cursor
		}
		resp, body := f.up(t, http.MethodGet, url, spec, session, nil, "", nil)
		if resp.StatusCode != http.StatusOK || body["object"] != "list" {
			t.Fatalf("walk %s page %d: %d %+v", url, page, resp.StatusCode, body)
		}
		got = append(got, adminItemIDs(body)...)
		next, _ := body["next_cursor"].(string)
		if next == "" {
			return got
		}
		cursor = next
	}
	t.Fatal("walk did not end")
	return nil
}

func sep(url string) string {
	for _, c := range url {
		if c == '?' {
			return "&"
		}
	}
	return "?"
}

func todoBody(project, text string) map[string]any {
	return map[string]any{"project": project, "text": text}
}

func stateBody(state string) map[string]any {
	return map[string]any{"state": state}
}
