package app

import (
	"net/http"
	"slices"
	"testing"

	"kun-galgame-api/pkg/problem"
)

func (f *quizFix) setRendered(t *testing.T, workID int, rendered bool) {
	t.Helper()
	if err := f.db.Exec(`UPDATE galgame SET catalog_rendered = ? WHERE id = ?`, rendered, workID).Error; err != nil {
		t.Fatal(err)
	}
}

func (f *quizFix) listedIDs(t *testing.T, rawURL, spec, session string) ([]string, int) {
	t.Helper()
	resp, body := f.qz(t, http.MethodGet, rawURL, spec, session, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s: %d %+v", rawURL, resp.StatusCode, body)
	}
	return adminItemIDs(body), asInt(body["total"])
}

func TestV1QuizzesHideWhileALinkedWorkIsNotRendered(t *testing.T) {
	f := newQuizFix(t, nil)
	main := idStr(g2QMain)
	for _, q := range []string{
		`UPDATE galgame_quiz SET hide_galgame = false WHERE id = ?`,
		`INSERT INTO galgame_quiz_galgame (quiz_id, work_id) VALUES (?, ` + idStr(g2WorkOrphan) + `)`,
	} {
		if err := f.db.Exec(q, g2QMain).Error; err != nil {
			t.Fatal(err)
		}
	}
	all := "/api/v1/quizzes?limit=100"
	byWork := all + "&work_id=" + idStr(g2WorkSFW)
	answered := "/api/v1/me/answered-quizzes?limit=100"
	states := "/api/v1/me/quiz-states?quiz_ids=" + main

	listedAll, totalAll := f.listedIDs(t, all, "/quizzes", "")
	listedWork, _ := f.listedIDs(t, byWork, "/quizzes", "")
	listedAnswered, totalAnswered := f.listedIDs(t, answered, "/me/answered-quizzes", "sess-other")
	for name, ids := range map[string][]string{"list": listedAll, "work list": listedWork, "answered": listedAnswered} {
		if !slices.Contains(ids, main) {
			t.Fatalf("%s misses the quiz before its work is hidden: %v", name, ids)
		}
	}
	if !slices.Contains(listedAll, idStr(g2QOrphan)) {
		t.Fatal("a work with no local row counts as rendered")
	}

	f.setRendered(t, g2WorkSFW, false)
	ids, total := f.listedIDs(t, all, "/quizzes", "")
	if slices.Contains(ids, main) || total != totalAll-1 || len(ids) != total {
		t.Errorf("list while hidden: %d items, total %d (was %d), has quiz %v", len(ids), total, totalAll, slices.Contains(ids, main))
	}
	if ids, total := f.listedIDs(t, byWork, "/quizzes", ""); len(ids) != 0 || total != 0 {
		t.Errorf("work list while hidden: %v total %d", ids, total)
	}
	ids, total = f.listedIDs(t, answered, "/me/answered-quizzes", "sess-other")
	if slices.Contains(ids, main) || total != totalAnswered-1 || len(ids) != total {
		t.Errorf("answered while hidden: %v total %d (was %d)", ids, total, totalAnswered)
	}
	for _, session := range []string{"", "sess-alice", "sess-bob"} {
		resp, body := f.qz(t, http.MethodGet, "/api/v1/quizzes/"+main, "/quizzes/{quiz_id}", session, "", nil)
		wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	}
	resp, body := f.qz(t, http.MethodGet, "/api/v1/quizzes/"+main+"/answers", "/quizzes/{quiz_id}/answers", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	resp, body = f.qz(t, http.MethodGet, states, "/me/quiz-states", "sess-bob", "", nil)
	if items, _ := body["items"].([]any); resp.StatusCode != http.StatusOK || len(items) != 0 {
		t.Errorf("quiz state while hidden: %d %+v", resp.StatusCode, body)
	}
	if n := f.scalar(t, `SELECT view FROM galgame_quiz WHERE id = ?`, g2QMain); n != 3 {
		t.Errorf("a hidden quiz counted a view: %d", n)
	}

	f.setRendered(t, g2WorkSFW, true)
	if ids, total := f.listedIDs(t, all, "/quizzes", ""); !slices.Contains(ids, main) || total != totalAll {
		t.Errorf("list after restore: total %d, want %d", total, totalAll)
	}
	if ids, _ := f.listedIDs(t, byWork, "/quizzes", ""); !slices.Contains(ids, main) {
		t.Errorf("work list after restore: %v", ids)
	}
	if ids, total := f.listedIDs(t, answered, "/me/answered-quizzes", "sess-other"); !slices.Contains(ids, main) || total != totalAnswered {
		t.Errorf("answered after restore: %v total %d", ids, total)
	}
	resp, body = f.qz(t, http.MethodGet, "/api/v1/quizzes/"+main, "/quizzes/{quiz_id}", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("detail after restore: %d %+v", resp.StatusCode, body)
	}
	resp, body = f.qz(t, http.MethodGet, states, "/me/quiz-states", "sess-bob", "", nil)
	if items, _ := body["items"].([]any); resp.StatusCode != http.StatusOK || len(items) != 1 {
		t.Errorf("quiz state after restore: %d %+v", resp.StatusCode, body)
	}
}
