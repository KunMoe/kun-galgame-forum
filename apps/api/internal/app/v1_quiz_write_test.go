package app

import (
	"net/http"
	"strconv"
	"sync"
	"testing"

	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/problem"
)

func TestV1CreateQuiz(t *testing.T) {
	f := newQuizFix(t, nil)
	resp, got := f.qz(t, http.MethodPost, "/api/v1/quizzes", "/quizzes", "sess-alice", keyUUID(1), createQuizBody(nil))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, got)
	}
	id := strID(got["id"])
	if resp.Header.Get("Location") != "/api/v1/quizzes/"+id || got["quiz_type"] != "single" {
		t.Errorf("created %s %+v", resp.Header.Get("Location"), got)
	}
	aw := f.snapshotAwards()
	if len(aw) != 1 || aw[0].delta != 2 || aw[0].reason != moemoepoint.ReasonContentApproved ||
		aw[0].ref != moemoepoint.Ref("galgame_quiz", asInt(got["id"])) ||
		aw[0].key != moemoepoint.Key("quiz_create", id) {
		t.Errorf("award %+v", aw)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_quiz_answer WHERE quiz_id = ? AND role = 'author'`, asInt(id)); n != 1 {
		t.Errorf("author row %d", n)
	}

	resp, got = f.qz(t, http.MethodPost, "/api/v1/quizzes", "/quizzes", "sess-alice", "", createQuizBody(nil))
	wantCode(t, resp, got, http.StatusBadRequest, problem.CodeInvalidParameter)

	resp, got = f.qz(t, http.MethodPost, "/api/v1/quizzes", "/quizzes", "sess-alice", keyUUID(1), createQuizBody(map[string]any{"prompt_text": "other"}))
	wantCode(t, resp, got, http.StatusConflict, problem.CodeIdempotencyKeyReused)

	resp, got = f.qz(t, http.MethodPost, "/api/v1/quizzes", "/quizzes", "", keyUUID(2), createQuizBody(nil))
	wantCode(t, resp, got, http.StatusUnauthorized, problem.CodeMissingCredential)

	resp, got = f.qz(t, http.MethodPost, "/api/v1/quizzes", "/quizzes", "sess-banned", keyUUID(3), createQuizBody(nil))
	wantCode(t, resp, got, http.StatusForbidden, problem.CodeAccountBanned)

	resp, raw := f.doJSON(t, http.MethodPost, "/api/v1/quizzes", "", "/quizzes", keyUUID(4),
		http.Header{"Authorization": {"Bearer no-such"}}, createQuizBody(nil))
	wantCode(t, resp, problemMap(t, raw), http.StatusUnauthorized, problem.CodeInvalidCredential)

	resp, got = f.qz(t, http.MethodPost, "/api/v1/quizzes", "/quizzes", "sess-alice", keyUUID(5),
		createQuizBody(map[string]any{"prompt_text": "   "}))
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if e := errorAt(got, "/prompt_text"); e == nil || e["reason"] != "TOO_SHORT" {
		t.Errorf("blank prompt %+v", got["errors"])
	}

	resp, got = f.qz(t, http.MethodPost, "/api/v1/quizzes", "/quizzes", "sess-alice", keyUUID(6),
		createQuizBody(map[string]any{"work_ids": []string{idStr(g2WorkMiss)}}))
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if e := errorAt(got, "/work_ids/0"); e == nil || e["reason"] != "UNKNOWN_REFERENCE" {
		t.Errorf("missing work %+v", got["errors"])
	}

	resp, got = f.qz(t, http.MethodPost, "/api/v1/quizzes", "/quizzes", "sess-alice", keyUUID(7),
		createQuizBody(map[string]any{"work_ids": []string{idStr(g2WorkHidden)}}))
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)

	resp, got = f.qz(t, http.MethodPost, "/api/v1/quizzes", "/quizzes", "sess-alice", keyUUID(8),
		createQuizBody(map[string]any{"quiz_type": "fill"}))
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)

	resp, got = f.qz(t, http.MethodPost, "/api/v1/quizzes", "/quizzes", "sess-alice", keyUUID(9),
		createQuizBody(map[string]any{"quiz_type": "judge", "choices": []string{"a", "b"}, "judge_answer": true, "correct_choice_indexes": []int{}}))
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if e := errorAt(got, "/choices"); e == nil || e["reason"] != "INCONSISTENT_WITH" {
		t.Errorf("judge choices %+v", got["errors"])
	}
}

func TestV1CreateQuizIdempotencyInProgress(t *testing.T) {
	f := newQuizFix(t, nil)
	body := createQuizBody(map[string]any{"prompt_text": "concurrent"})
	var (
		wg  sync.WaitGroup
		mu  sync.Mutex
		got []string
	)
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			resp, out := f.qz(t, http.MethodPost, "/api/v1/quizzes", "/quizzes", "sess-alice", keyUUID(90), body)
			code, _ := out["code"].(string)
			mu.Lock()
			got = append(got, strconv.Itoa(resp.StatusCode)+":"+code)
			mu.Unlock()
		}()
	}
	wg.Wait()
	seen := map[string]int{}
	for _, g := range got {
		seen[g]++
	}
	if seen["201:"]+seen["409:IDEMPOTENCY_REQUEST_IN_PROGRESS"]+seen["409:IDEMPOTENCY_KEY_REUSED"] < 2 {
		t.Errorf("concurrent idempotency %v", got)
	}
}

func TestV1CreateQuizContentRejected(t *testing.T) {
	f := newQuizFix(t, denyChecker{})
	resp, got := f.qz(t, http.MethodPost, "/api/v1/quizzes", "/quizzes", "sess-alice", keyUUID(6),
		createQuizBody(map[string]any{"prompt_text": "blocked"}))
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeContentRejected)
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_quiz WHERE question = 'blocked'`); n != 0 {
		t.Error("denied create wrote a row")
	}
}

func TestV1UpdateQuizTrustOnlyChangedText(t *testing.T) {
	f := newQuizFix(t, denyChecker{})
	resp, got := f.qz(t, http.MethodPatch, "/api/v1/quizzes/"+idStr(g2QMain), "/quizzes/{quiz_id}", "sess-alice", "",
		map[string]any{"quiz_category": "trivia"})
	if resp.StatusCode != http.StatusOK || got["quiz_category"] != "trivia" {
		t.Fatalf("category-only patch %d %+v", resp.StatusCode, got)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_quiz WHERE id = ? AND category = 'trivia'`, g2QMain); n != 1 {
		t.Error("category was not written")
	}
	resp, got = f.qz(t, http.MethodPatch, "/api/v1/quizzes/"+idStr(g2QMain), "/quizzes/{quiz_id}", "sess-alice", "",
		map[string]any{"prompt_text": "blocked"})
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeContentRejected)
}

func TestV1UpdateQuizTypeImmutable(t *testing.T) {
	f := newQuizFix(t, nil)
	resp, got := f.qz(t, http.MethodPatch, "/api/v1/quizzes/"+idStr(g2QMain), "/quizzes/{quiz_id}", "sess-alice", "",
		map[string]any{"quiz_type": "judge"})
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if e := errorAt(got, "/quiz_type"); e == nil || e["reason"] != "IMMUTABLE" {
		t.Errorf("immutable %+v", got["errors"])
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_quiz WHERE id = ? AND type = 'single'`, g2QMain); n != 1 {
		t.Error("type changed")
	}
}

func TestV1UpdateQuizRegradeBothDirections(t *testing.T) {
	f := newQuizFix(t, nil)
	resp, got := f.qz(t, http.MethodPatch, "/api/v1/quizzes/"+idStr(g2QRegrade), "/quizzes/{quiz_id}", "sess-alice", "",
		map[string]any{"correct_choice_indexes": []int{0}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("regrade to 0 %d %+v", resp.StatusCode, got)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_quiz_answer WHERE id = ? AND is_correct`, g2AnsRegrade); n != 1 {
		t.Error("wrong→right did not flip")
	}
	resp, got = f.qz(t, http.MethodPatch, "/api/v1/quizzes/"+idStr(g2QRegrade), "/quizzes/{quiz_id}", "sess-alice", "",
		map[string]any{"correct_choice_indexes": []int{2}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("regrade away %d %+v", resp.StatusCode, got)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_quiz_answer WHERE id = ? AND is_correct`, g2AnsRegrade); n != 0 {
		t.Error("right→wrong did not flip")
	}
	correct := f.scalar(t, `SELECT COUNT(*) FROM galgame_quiz_answer WHERE quiz_id = ? AND role = 'answerer' AND is_correct`, g2QRegrade)
	stored := f.scalar(t, `SELECT correct_count FROM galgame_quiz WHERE id = ?`, g2QRegrade)
	if stored != correct {
		t.Errorf("correct_count %d, sql %d", stored, correct)
	}
}

func TestV1UpdateQuizPermission(t *testing.T) {
	f := newQuizFix(t, nil)
	resp, got := f.qz(t, http.MethodPatch, "/api/v1/quizzes/"+idStr(g2QMain), "/quizzes/{quiz_id}", "sess-bob", "",
		map[string]any{"quiz_category": "music"})
	wantCode(t, resp, got, http.StatusForbidden, problem.CodePermissionRequired)
	resp, got = f.qz(t, http.MethodPatch, "/api/v1/quizzes/"+idStr(g2QMain), "/quizzes/{quiz_id}", "sess-staff", "",
		map[string]any{"quiz_category": "music"})
	if resp.StatusCode != http.StatusOK || got["quiz_category"] != "music" {
		t.Errorf("staff patch %d %+v", resp.StatusCode, got)
	}
}

func TestV1DeleteQuiz(t *testing.T) {
	f := newQuizFix(t, nil)
	resp, got := f.qz(t, http.MethodDelete, "/api/v1/quizzes/"+idStr(g2QJudge), "/quizzes/{quiz_id}", "sess-alice", "", nil)
	wantCode(t, resp, got, http.StatusForbidden, problem.CodePermissionRequired)
	resp, _ = f.qz(t, http.MethodDelete, "/api/v1/quizzes/"+idStr(g2QJudge), "/quizzes/{quiz_id}", "sess-grant", "", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete %d", resp.StatusCode)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_quiz WHERE id = ?`, g2QJudge); n != 0 {
		t.Error("row remained")
	}
	aw := f.snapshotAwards()
	found := false
	for _, a := range aw {
		if a.key == moemoepoint.Key("quiz_delete", idStr(g2QJudge)) && a.delta == -2 {
			found = true
		}
	}
	if !found {
		t.Errorf("delete award %+v", aw)
	}
}

func TestV1CreateQuizCatalogDown(t *testing.T) {
	f := newQuizFix(t, nil)
	f.cat.fail.Store(true)
	resp, got := f.qz(t, http.MethodPost, "/api/v1/quizzes", "/quizzes", "sess-alice", keyUUID(11),
		createQuizBody(map[string]any{"work_ids": []string{idStr(g2WorkSFW)}}))
	wantCode(t, resp, got, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
}
