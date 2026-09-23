package app

import (
	"net/http"
	"strconv"
	"sync"
	"testing"

	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/problem"
)

func TestV1CreateQuizAnswerSelfForbidden(t *testing.T) {
	f := newQuizFix(t, nil)
	resp, got := f.qz(t, http.MethodPost, "/api/v1/quizzes/"+idStr(g2QJudge)+"/answers",
		"/quizzes/{quiz_id}/answers", "sess-grant", keyUUID(20),
		map[string]any{"choice_indexes": []int{}, "is_statement_true": true})
	wantCode(t, resp, got, http.StatusForbidden, problem.CodeSelfAnswerForbidden)
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_quiz_answer WHERE quiz_id = ? AND role = 'answerer'`, g2QJudge); n != 0 {
		t.Error("author answerer row written")
	}
}

func TestV1CreateQuizAnswerAlreadyExists(t *testing.T) {
	f := newQuizFix(t, nil)
	resp, got := f.qz(t, http.MethodPost, "/api/v1/quizzes/"+idStr(g2QMain)+"/answers",
		"/quizzes/{quiz_id}/answers", "sess-bob", keyUUID(21),
		map[string]any{"choice_indexes": []int{0}, "is_statement_true": nil})
	wantCode(t, resp, got, http.StatusConflict, problem.CodeAlreadyExists)
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_quiz_answer WHERE quiz_id = ? AND user_id = ? AND role = 'answerer'`,
		g2QMain, w3UserBob); n != 1 {
		t.Errorf("second row %d", n)
	}
}

func TestV1CreateQuizAnswerConcurrentUnique(t *testing.T) {
	f := newQuizFix(t, nil)
	body := map[string]any{"choice_indexes": []int{}, "is_statement_true": false}
	var (
		wg  sync.WaitGroup
		mu  sync.Mutex
		got []string
	)
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			resp, out := f.qz(t, http.MethodPost, "/api/v1/quizzes/"+idStr(g2QJudge)+"/answers",
				"/quizzes/{quiz_id}/answers", "sess-bob", keyUUID(30+i), body)
			code, _ := out["code"].(string)
			mu.Lock()
			got = append(got, strconv.Itoa(resp.StatusCode)+":"+code)
			mu.Unlock()
		}()
	}
	wg.Wait()
	ok, conflict := 0, 0
	for _, g := range got {
		switch g {
		case "201:":
			ok++
		case "409:ALREADY_EXISTS":
			conflict++
		}
	}
	if ok+conflict != 2 || ok == 0 {
		t.Errorf("concurrent answers %v", got)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_quiz_answer WHERE quiz_id = ? AND role = 'answerer'`, g2QJudge); n != 1 {
		t.Errorf("answerer rows %d", n)
	}
}

func TestV1CreateQuizAnswerOK(t *testing.T) {
	f := newQuizFix(t, nil)
	resp, got := f.qz(t, http.MethodPost, "/api/v1/quizzes/"+idStr(g2QJudge)+"/answers",
		"/quizzes/{quiz_id}/answers", "sess-bob", keyUUID(22),
		map[string]any{"choice_indexes": []int{}, "is_statement_true": true})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("answer %d %+v", resp.StatusCode, got)
	}
	if resp.Header.Get("Location") != "/api/v1/quizzes/"+idStr(g2QJudge) {
		t.Errorf("location %s", resp.Header.Get("Location"))
	}
	if got["object"] != "quiz_answer_result" {
		t.Errorf("object %v", got["object"])
	}
	ans, _ := got["answer"].(map[string]any)
	if ans == nil || ans["is_correct"] != true {
		t.Errorf("answer %+v", ans)
	}
	if got["solution"] == nil {
		t.Error("missing solution")
	}
	if n := f.scalar(t, `SELECT answer_count FROM galgame_quiz WHERE id = ?`, g2QJudge); n != 1 {
		t.Errorf("answer_count %d", n)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM message WHERE sender_id = ? AND receiver_id = ? AND type = 'quiz-answered'
		AND link = ? AND content = '选择「正确」，回答正确'`, w3UserBob, w3UserGrant, "/galgame-quiz/"+idStr(g2QJudge)); n != 1 {
		t.Errorf("the author was not told about the answer: %d quiz-answered rows", n)
	}
}

func TestV1PutDeleteQuizFavorite(t *testing.T) {
	f := newQuizFix(t, nil)
	resp, got := f.qz(t, http.MethodPut, "/api/v1/quizzes/"+idStr(g2QMain)+"/favorite",
		"/quizzes/{quiz_id}/favorite", "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fav %d %+v", resp.StatusCode, got)
	}
	v, _ := got["viewer"].(map[string]any)
	if v == nil || v["has_favorited"] != true || asInt(got["favorite_count"]) != 1 {
		t.Errorf("fav body %+v", got)
	}
	aw := f.snapshotAwards()
	if len(aw) != 1 || aw[0].delta != 1 || aw[0].reason != moemoepoint.ReasonLiked {
		t.Errorf("fav award %+v", aw)
	}
	resp, got = f.qz(t, http.MethodPut, "/api/v1/quizzes/"+idStr(g2QMain)+"/favorite",
		"/quizzes/{quiz_id}/favorite", "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK || asInt(got["favorite_count"]) != 1 {
		t.Errorf("idempotent put %d %+v", resp.StatusCode, got)
	}
	if len(f.snapshotAwards()) != 1 {
		t.Errorf("second put awarded again %+v", f.snapshotAwards())
	}
	resp, got = f.qz(t, http.MethodDelete, "/api/v1/quizzes/"+idStr(g2QMain)+"/favorite",
		"/quizzes/{quiz_id}/favorite", "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK || asInt(got["favorite_count"]) != 0 {
		t.Errorf("unfav %d %+v", resp.StatusCode, got)
	}
	resp, got = f.qz(t, http.MethodDelete, "/api/v1/quizzes/"+idStr(g2QMain)+"/favorite",
		"/quizzes/{quiz_id}/favorite", "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK || asInt(got["favorite_count"]) != 0 {
		t.Errorf("idempotent delete %d %+v", resp.StatusCode, got)
	}
	before := len(f.snapshotAwards())
	resp, got = f.qz(t, http.MethodPut, "/api/v1/quizzes/"+idStr(g2QMain)+"/favorite",
		"/quizzes/{quiz_id}/favorite", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("self fav %d %+v", resp.StatusCode, got)
	}
	if len(f.snapshotAwards()) != before {
		t.Errorf("self-favorite awarded: %+v", f.snapshotAwards()[before:])
	}
}

func TestV1PutQuizQualityRating(t *testing.T) {
	f := newQuizFix(t, nil)
	resp, got := f.qz(t, http.MethodPut, "/api/v1/quizzes/"+idStr(g2QMain)+"/quality-rating",
		"/quizzes/{quiz_id}/quality-rating", "sess-alice", "", map[string]any{"rating": 8})
	wantCode(t, resp, got, http.StatusForbidden, problem.CodeQuizAnswerRequired)
	resp, got = f.qz(t, http.MethodPut, "/api/v1/quizzes/"+idStr(g2QJudge)+"/quality-rating",
		"/quizzes/{quiz_id}/quality-rating", "sess-bob", "", map[string]any{"rating": 8})
	wantCode(t, resp, got, http.StatusForbidden, problem.CodeQuizAnswerRequired)
	resp, got = f.qz(t, http.MethodPut, "/api/v1/quizzes/"+idStr(g2QMain)+"/quality-rating",
		"/quizzes/{quiz_id}/quality-rating", "sess-bob", "", map[string]any{"rating": 8})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("rate %d %+v", resp.StatusCode, got)
	}
	if asInt(got["quality_count"]) != 1 {
		t.Errorf("count %v", got["quality_count"])
	}
	v, _ := got["viewer"].(map[string]any)
	if v == nil || asInt(v["quality_rating"]) != 8 {
		t.Errorf("viewer %+v", v)
	}
	resp, got = f.qz(t, http.MethodPut, "/api/v1/quizzes/"+idStr(g2QMain)+"/quality-rating",
		"/quizzes/{quiz_id}/quality-rating", "sess-bob", "", map[string]any{"rating": 4})
	if resp.StatusCode != http.StatusOK || asInt(got["quality_count"]) != 1 {
		t.Errorf("replace %d %+v", resp.StatusCode, got)
	}
	sum := f.scalar(t, `SELECT quality_sum FROM galgame_quiz WHERE id = ?`, g2QMain)
	if sum != 4 {
		t.Errorf("sum %d", sum)
	}
}

func TestV1ListMyQuizStates(t *testing.T) {
	f := newQuizFix(t, nil)
	_, _ = f.qz(t, http.MethodPut, "/api/v1/quizzes/"+idStr(g2QMain)+"/favorite",
		"/quizzes/{quiz_id}/favorite", "sess-bob", "", nil)
	ids := idStr(g2QMain) + "," + idStr(g2QJudge) + "," + idStr(g2QTied+5) + "," + idStr(g2QGone)
	resp, body := f.qz(t, http.MethodGet, "/api/v1/me/quiz-states?quiz_ids="+ids,
		"/me/quiz-states", "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK || body["object"] != "list" {
		t.Fatalf("states %d %+v", resp.StatusCode, body)
	}
	items, _ := body["items"].([]any)
	missing, _ := body["missing"].([]any)
	if len(items) != 2 {
		t.Errorf("items %v", items)
	}
	if len(missing) != 2 {
		t.Errorf("missing %v", missing)
	}
	foundFav := false
	for _, raw := range items {
		it, _ := raw.(map[string]any)
		if strID(it["quiz_id"]) == idStr(g2QMain) && it["has_favorited"] == true {
			foundFav = true
		}
	}
	if !foundFav {
		t.Errorf("favorite not reported in items %+v", items)
	}
	resp, body = f.qz(t, http.MethodGet, "/api/v1/me/quiz-states", "/me/quiz-states", "sess-bob", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeInvalidParameter)
}
