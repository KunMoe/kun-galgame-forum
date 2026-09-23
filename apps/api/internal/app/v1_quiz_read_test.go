package app

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"kun-galgame-api/pkg/problem"
)

func TestV1QuizzesWalkTiedSort(t *testing.T) {
	f := newQuizFix(t, nil)
	want := f.sqlIDs(t, `SELECT q.id::text FROM galgame_quiz q
		WHERE q.id BETWEEN 930003201 AND 930003299 AND q.user_id IN (`+renderableQuizAuthorsSQL()+`)
		AND NOT EXISTS (
			SELECT 1 FROM galgame_quiz_galgame gg
			JOIN galgame g ON g.id = gg.work_id
			WHERE gg.quiz_id = q.id AND g.content_limit = 'nsfw')
		ORDER BY q.status_update_time DESC, q.id DESC`)
	if len(want) < 7 {
		t.Fatalf("seed too thin: %v", want)
	}
	for _, limit := range []int{2, 3} {
		if got := f.walkQuizzes(t, "", limit); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("limit %d walked %v, want %v", limit, got, want)
		}
	}
}

func TestV1QuizzesTotalExcludesUnrenderableAuthors(t *testing.T) {
	f := newQuizFix(t, nil)
	resp, body := f.qz(t, http.MethodGet, "/api/v1/quizzes?limit=100", "/quizzes", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	want := f.scalar(t, `SELECT COUNT(*) FROM galgame_quiz q
		WHERE q.id BETWEEN 930003201 AND 930003299 AND q.user_id IN (`+renderableQuizAuthorsSQL()+`)
		AND NOT EXISTS (
			SELECT 1 FROM galgame_quiz_galgame gg
			JOIN galgame g ON g.id = gg.work_id
			WHERE gg.quiz_id = q.id AND g.content_limit = 'nsfw')`)
	if asInt(body["total"]) != want || body["total_relation"] != "eq" {
		t.Errorf("total %v %v, want %d eq", body["total"], body["total_relation"], want)
	}
	walked := f.walkQuizzes(t, "", 2)
	if len(walked) != want {
		t.Errorf("walked %d items, want %d", len(walked), want)
	}
	for _, id := range walked {
		if id == idStr(g2QTied+5) || id == idStr(g2QTied+6) {
			t.Errorf("unrenderable author quiz %s in items", id)
		}
	}
}

func TestV1QuizzesSummaryHasNoAnswerKey(t *testing.T) {
	f := newQuizFix(t, nil)
	resp, body := f.qz(t, http.MethodGet, "/api/v1/quizzes?limit=100", "/quizzes", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	items, _ := body["items"].([]any)
	if len(items) == 0 {
		t.Fatal("empty list")
	}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if _, ok := item["solution"]; ok {
			t.Errorf("list item leaked solution: %+v", item)
		}
		if _, ok := item["correct_choice_indexes"]; ok {
			t.Errorf("list item leaked correct_choice_indexes: %+v", item)
		}
	}
}

func TestV1QuizzesUnknownSortAndEnum(t *testing.T) {
	f := newQuizFix(t, nil)
	resp, body := f.qz(t, http.MethodGet, "/api/v1/quizzes?sort=hot", "/quizzes", "", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeUnknownSort)
	resp, body = f.qz(t, http.MethodGet, "/api/v1/quizzes?quiz_type=fill", "/quizzes", "", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeUnknownEnumValue)
	resp, body = f.qz(t, http.MethodGet, "/api/v1/quizzes?limit=101", "/quizzes", "", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeLimitTooLarge)
	resp, body = f.qz(t, http.MethodGet, "/api/v1/quizzes?page=10000&limit=2", "/quizzes", "", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeInvalidParameter)
	if e := errorParam(body, "page"); e == nil || e["reason"] != "OUT_OF_RANGE" {
		t.Errorf("depth errors %+v", body["errors"])
	}
	resp, body = f.qz(t, http.MethodGet, "/api/v1/quizzes?difficulty=0", "/quizzes", "", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeInvalidParameter)
	resp, body = f.qz(t, http.MethodGet, "/api/v1/quizzes?include_nsfw=yes", "/quizzes", "", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeInvalidParameter)
}

func TestV1QuizzesUserclientFailure(t *testing.T) {
	f := newQuizFix(t, nil)
	f.failOA.Store(true)
	resp, body := f.qz(t, http.MethodGet, "/api/v1/quizzes", "/quizzes", "", "", nil)
	wantCode(t, resp, body, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
	resp, body = f.qz(t, http.MethodGet, "/api/v1/quizzes/"+idStr(g2QMain), "/quizzes/{quiz_id}", "", "", nil)
	wantCode(t, resp, body, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
	resp, body = f.qz(t, http.MethodGet, "/api/v1/quizzes/"+idStr(g2QMain)+"/answers", "/quizzes/{quiz_id}/answers", "", "", nil)
	wantCode(t, resp, body, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
	resp, body = f.qz(t, http.MethodGet, "/api/v1/quizzes/"+idStr(g2QMain)+"/source", "/quizzes/{quiz_id}/source", "sess-alice", "", nil)
	wantCode(t, resp, body, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
}

func TestV1GetQuizUnansweredHidesKeyAndWorks(t *testing.T) {
	f := newQuizFix(t, nil)
	updated := f.sqlIDs(t, `SELECT updated::text FROM galgame_quiz WHERE id = ?`, g2QMain)
	resp, body := f.qz(t, http.MethodGet, "/api/v1/quizzes/"+idStr(g2QMain), "/quizzes/{quiz_id}", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("detail %d %+v", resp.StatusCode, body)
	}
	if body["object"] != "quiz" || body["solution"] != nil {
		t.Errorf("unanswered solution %+v", body["solution"])
	}
	if body["viewer"] != nil {
		t.Errorf("anon viewer %+v", body["viewer"])
	}
	works, _ := body["works"].([]any)
	if len(works) != 0 {
		t.Errorf("hidden works leaked to unanswered: %v", works)
	}
	choices, _ := body["choices"].([]any)
	if len(choices) != 3 {
		t.Errorf("choices %v", choices)
	}
	if asInt(body["view_count"]) != 4 {
		t.Errorf("view_count %v", body["view_count"])
	}
	if n := f.scalar(t, `SELECT view FROM galgame_quiz WHERE id = ?`, g2QMain); n != 4 {
		t.Errorf("view %d", n)
	}
	if got := f.sqlIDs(t, `SELECT updated::text FROM galgame_quiz WHERE id = ?`, g2QMain); fmt.Sprint(got) != fmt.Sprint(updated) {
		t.Errorf("a view moved updated: %v -> %v", updated, got)
	}
	resp, body = f.qz(t, http.MethodGet, "/api/v1/quizzes/"+idStr(g2QTied+5), "/quizzes/{quiz_id}", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	resp, body = f.qz(t, http.MethodGet, "/api/v1/quizzes/"+idStr(g2QGone), "/quizzes/{quiz_id}", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
}

func TestV1GetQuizAnsweredRevealsWorks(t *testing.T) {
	f := newQuizFix(t, nil)
	resp, body := f.qz(t, http.MethodGet, "/api/v1/quizzes/"+idStr(g2QMain), "/quizzes/{quiz_id}", "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bob detail %d %+v", resp.StatusCode, body)
	}
	if body["solution"] == nil {
		t.Fatal("answered caller missing solution")
	}
	works, _ := body["works"].([]any)
	if len(works) == 0 {
		t.Fatal("answered caller missing works")
	}
	v, _ := body["viewer"].(map[string]any)
	if v == nil || v["has_answered"] != true || v["has_favorited"] != false {
		t.Errorf("bob viewer %+v", v)
	}
	bearer := http.Header{"Authorization": {"Bearer staff-token"}}
	resp, raw := f.doJSON(t, http.MethodGet, "/api/v1/quizzes/"+idStr(g2QMain), "", "/quizzes/{quiz_id}", "", bearer, nil)
	staff := problemMap(t, raw)
	sv, _ := staff["viewer"].(map[string]any)
	if resp.StatusCode != http.StatusOK || sv == nil || sv["can_edit"] != false || sv["can_delete"] != false {
		t.Errorf("Bearer moderator viewer %d %+v", resp.StatusCode, staff["viewer"])
	}
}

func TestV1GetQuizAuthorSeesSolutionWithoutAnswering(t *testing.T) {
	f := newQuizFix(t, nil)
	resp, body := f.qz(t, http.MethodGet, "/api/v1/quizzes/"+idStr(g2QMain), "/quizzes/{quiz_id}", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK || body["solution"] == nil {
		t.Fatalf("author %d %+v", resp.StatusCode, body)
	}
	v, _ := body["viewer"].(map[string]any)
	if v == nil || v["has_answered"] != false || v["answer"] != nil || v["can_edit"] != true {
		t.Errorf("author viewer %+v", v)
	}
}

func TestV1ListQuizAnswersHidesKey(t *testing.T) {
	f := newQuizFix(t, nil)
	resp, body := f.qz(t, http.MethodGet, "/api/v1/quizzes/"+idStr(g2QMain)+"/answers?limit=2&include_total=true",
		"/quizzes/{quiz_id}/answers", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("answers %d %+v", resp.StatusCode, body)
	}
	if asInt(body["total"]) != 4 {
		t.Errorf("total %v", body["total"])
	}
	items, _ := body["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("page %v", items)
	}
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if item["is_correct"] != nil || item["submission"] != nil {
			t.Errorf("unanswered answers leaked key fields: %+v", item)
		}
		if _, ok := item["solution"]; ok {
			t.Errorf("answers collection leaked solution: %+v", item)
		}
	}
	resp, body = f.qz(t, http.MethodGet, "/api/v1/quizzes/"+idStr(g2QMain)+"/answers?limit=2",
		"/quizzes/{quiz_id}/answers", "sess-bob", "", nil)
	items, _ = body["items"].([]any)
	if resp.StatusCode != http.StatusOK || len(items) == 0 {
		t.Fatalf("bob answers %d %+v", resp.StatusCode, body)
	}
	first, _ := items[0].(map[string]any)
	if first["is_correct"] == nil || first["submission"] == nil {
		t.Errorf("answered caller missing key fields: %+v", first)
	}
}

func TestV1ListQuizAnswersWalkTiedCreated(t *testing.T) {
	f := newQuizFix(t, nil)
	want := f.sqlIDs(t, `SELECT id::text FROM galgame_quiz_answer
		WHERE quiz_id = ? AND role = 'answerer'
		ORDER BY created DESC, id DESC`, g2QMain)
	if len(want) != 4 {
		t.Fatalf("answer seed %v", want)
	}
	got := f.walkAnswers(t, g2QMain, 2)
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("walked %v, want %v", got, want)
	}
}

func TestV1GetQuizSource(t *testing.T) {
	f := newQuizFix(t, nil)
	resp, body := f.qz(t, http.MethodGet, "/api/v1/quizzes/"+idStr(g2QMain)+"/source",
		"/quizzes/{quiz_id}/source", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK || body["object"] != "quiz_source" {
		t.Fatalf("source %d %+v", resp.StatusCode, body)
	}
	if !strings.Contains(fmt.Sprint(body["prompt_text"]), "secret") {
		t.Errorf("prompt_text %v", body["prompt_text"])
	}
	resp, body = f.qz(t, http.MethodGet, "/api/v1/quizzes/"+idStr(g2QMain)+"/source",
		"/quizzes/{quiz_id}/source", "sess-bob", "", nil)
	wantCode(t, resp, body, http.StatusForbidden, problem.CodePermissionRequired)
}

func TestV1QuizzesNSFWFilter(t *testing.T) {
	f := newQuizFix(t, nil)
	resp, body := f.qz(t, http.MethodGet, "/api/v1/quizzes?limit=100", "/quizzes", "", "", nil)
	ids := map[string]bool{}
	for _, id := range adminItemIDs(body) {
		ids[id] = true
	}
	if ids[idStr(g2QNSFW)] {
		t.Error("NSFW-linked quiz in default list")
	}
	if !ids[idStr(g2QOrphan)] {
		t.Error("orphan-link quiz missing from SFW list")
	}
	resp, body = f.qz(t, http.MethodGet, "/api/v1/quizzes?limit=100&include_nsfw=true", "/quizzes", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("nsfw list %d %+v", resp.StatusCode, body)
	}
	found := false
	for _, id := range adminItemIDs(body) {
		if id == idStr(g2QNSFW) {
			found = true
		}
	}
	if !found {
		t.Error("NSFW-linked quiz missing when include_nsfw=true")
	}
}

func TestV1ListMyAnsweredQuizzes(t *testing.T) {
	f := newQuizFix(t, nil)
	resp, body := f.qz(t, http.MethodGet, "/api/v1/me/answered-quizzes?limit=100",
		"/me/answered-quizzes", "sess-other", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("answered %d %+v", resp.StatusCode, body)
	}
	want := f.scalar(t, `SELECT COUNT(*) FROM galgame_quiz q
		JOIN galgame_quiz_answer a ON a.quiz_id = q.id
		WHERE a.user_id = ? AND a.role = 'answerer' AND q.user_id IN (`+renderableQuizAuthorsSQL()+`)
		AND q.id BETWEEN 930003201 AND 930003299`, w3UserOther)
	if asInt(body["total"]) != want {
		t.Errorf("total %v want %d", body["total"], want)
	}
	resp, body = f.qz(t, http.MethodGet, "/api/v1/me/answered-quizzes", "/me/answered-quizzes", "", "", nil)
	wantCode(t, resp, body, http.StatusUnauthorized, problem.CodeMissingCredential)
}
