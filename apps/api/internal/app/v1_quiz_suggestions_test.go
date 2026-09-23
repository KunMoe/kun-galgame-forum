package app

import (
	"net/http"
	"net/url"
	"testing"

	"kun-galgame-api/pkg/problem"
)

func TestV1ListWorkSuggestions(t *testing.T) {
	f := newQuizFix(t, nil)
	resp, body := f.qz(t, http.MethodGet, "/api/v1/work-suggestions?q=Alpha", "/work-suggestions", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("suggest %d %+v", resp.StatusCode, body)
	}
	if body["object"] != "list" {
		t.Errorf("object %v", body["object"])
	}
	if _, ok := body["total"]; ok {
		t.Errorf("suggestions sent total: %+v", body)
	}
	items, _ := body["items"].([]any)
	if len(items) == 0 {
		t.Fatal("no suggestions")
	}
	first, _ := items[0].(map[string]any)
	if first["object"] != "work" || strID(first["id"]) != idStr(g2WorkSFW) {
		t.Errorf("first %+v", first)
	}
	lastSearch := func() url.Values {
		f.cat.mu.Lock()
		defer f.cat.mu.Unlock()
		return f.cat.searched[len(f.cat.searched)-1]
	}
	q := lastSearch()
	if q.Get("content_limit") != "sfw" {
		t.Errorf("default suggestions must be SFW-gated: %v", q)
	}
	if _, ok := q["claim_state"]; ok {
		t.Errorf("the picker is the catalog, not claimed works: %v", q)
	}
	if _, ok := q["claimed"]; ok {
		t.Errorf("the picker is the catalog, not claimed works: %v", q)
	}
	resp, body = f.qz(t, http.MethodGet, "/api/v1/work-suggestions?q=Alpha&include_nsfw=true", "/work-suggestions", "", "", nil)
	if q := lastSearch(); resp.StatusCode != http.StatusOK || q.Get("nsfw") != "true" || q.Has("content_limit") {
		t.Errorf("include_nsfw=true must open the catalog's NSFW population: %d %v", resp.StatusCode, lastSearch())
	}

	resp, body = f.qz(t, http.MethodGet, "/api/v1/work-suggestions?q=%20%20", "/work-suggestions", "", "", nil)
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)

	resp, body = f.qz(t, http.MethodGet, "/api/v1/work-suggestions", "/work-suggestions", "", "", nil)
	if resp.StatusCode != http.StatusUnprocessableEntity && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("missing q %d %+v", resp.StatusCode, body)
	}

	f.cat.fail.Store(true)
	resp, body = f.qz(t, http.MethodGet, "/api/v1/work-suggestions?q=Alpha", "/work-suggestions", "", "", nil)
	wantCode(t, resp, body, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
}

func TestV1GetQuizCatalogDown(t *testing.T) {
	f := newQuizFix(t, nil)
	f.cat.fail.Store(true)
	resp, body := f.qz(t, http.MethodGet, "/api/v1/quizzes/"+idStr(g2QMain), "/quizzes/{quiz_id}", "sess-bob", "", nil)
	wantCode(t, resp, body, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
}

func TestV1ListQuizAnswersInvalidCursor(t *testing.T) {
	f := newQuizFix(t, nil)
	resp, body := f.qz(t, http.MethodGet, "/api/v1/quizzes/"+idStr(g2QMain)+"/answers?cursor=cur_xxx",
		"/quizzes/{quiz_id}/answers", "", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeInvalidCursor)
}
