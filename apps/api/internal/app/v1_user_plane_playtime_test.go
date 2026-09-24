package app

import (
	"net/http"
	"testing"

	"kun-galgame-api/pkg/problem"
)

func TestV1PutWorkPlaytime(t *testing.T) {
	f := newG6Fix(t)
	path := g6Path(g6WorkLive) + "/playtime"
	spec := "/works/{work_id}/playtime"
	doing := "doing"
	resp, body := f.call(t, http.MethodPut, path, spec, "sess-alice", "", map[string]any{"minutes": 90, "play_state": doing})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put %d %+v", resp.StatusCode, body)
	}
	if asInt(body["minutes"]) != 90 {
		t.Errorf("minutes %+v", body["minutes"])
	}

	resp, body = f.call(t, http.MethodPut, path, spec, "sess-alice", "", map[string]any{})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)

	resp, body = f.call(t, http.MethodPut, path, spec, "sess-alice", "", map[string]any{"minutes": 60001})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if e := firstError(t, body); e["reason"] != "OUT_OF_RANGE" {
		t.Errorf("range %+v", body["errors"])
	}

	resp, body = f.call(t, http.MethodPut, path, spec, "sess-alice", "", map[string]any{"play_state": "finished"})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if e := firstError(t, body); e["reason"] != problem.ReasonUnknownValue {
		t.Errorf("unknown play_state %+v", body["errors"])
	}

	resp, body = f.call(t, http.MethodPut, path, spec, "sess-alice", "", map[string]any{"play_state": "done"})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if e := firstError(t, body); e["reason"] != "NOT_ALLOWED_VALUE" {
		t.Errorf("done %+v", body["errors"])
	}
	f.user.mu.Lock()
	n := len(f.user.putStates)
	f.user.mu.Unlock()
	if n != 1 {
		t.Errorf("done must not PUT work-state, putStates=%d", n)
	}

	resp, body = f.call(t, http.MethodPut, g6Path(g6WorkHidden)+"/playtime", spec, "sess-alice", "", map[string]any{"minutes": 10})
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
}

func TestV1DeleteWorkPlaytime(t *testing.T) {
	f := newG6Fix(t)
	path := g6Path(g6WorkLive) + "/playtime"
	spec := "/works/{work_id}/playtime"
	resp, body := f.call(t, http.MethodDelete, path, spec, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete %d %+v", resp.StatusCode, body)
	}
	if asInt(body["minutes"]) != 0 || body["play_state"] != nil {
		t.Errorf("cleared %+v", body)
	}
	f.user.mu.Lock()
	got := append([]int64(nil), f.user.deletePlay...)
	f.user.mu.Unlock()
	if len(got) != 1 || got[0] != g6WorkLive {
		t.Errorf("upstream delete %v", got)
	}
	resp, body = f.call(t, http.MethodDelete, path, spec, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK || asInt(body["minutes"]) != 0 || body["play_state"] != nil {
		t.Errorf("repeat delete %d %+v", resp.StatusCode, body)
	}
}

func TestV1ListMyPlaytimesNSFWPredicate(t *testing.T) {
	f := newG6Fix(t)
	resp, body := f.call(t, http.MethodGet, "/api/v1/me/playtimes", "/me/playtimes", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	items, _ := body["items"].([]any)
	total := asInt(body["total"])
	if total != len(items) {
		t.Errorf("total %d items %d", total, len(items))
	}
	for _, raw := range items {
		it, _ := raw.(map[string]any)
		work, _ := it["work_summary"].(map[string]any)
		if work["is_nsfw"] == true {
			t.Errorf("nsfw leaked %+v", it)
		}
		if strID(work["id"]) == idStr(g6WorkNSFW) {
			t.Error("nsfw work in sfw list")
		}
	}
	minutes := asInt(body["total_minutes"])
	resp, all := f.call(t, http.MethodGet, "/api/v1/me/playtimes?include_nsfw=true", "/me/playtimes", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("nsfw list %d %+v", resp.StatusCode, all)
	}
	if asInt(all["total_minutes"]) < minutes {
		t.Errorf("nsfw minutes missing: sfw %d all %v", minutes, all["total_minutes"])
	}
}

func TestV1PlaytimeScopeAndLimit(t *testing.T) {
	f := newG6Fix(t)
	resp, body := f.call(t, http.MethodGet, "/api/v1/me/playtimes", "/me/playtimes", "sess-noscope", "", nil)
	wantCode(t, resp, body, http.StatusForbidden, problem.CodeScopeRequired)

	resp, body = f.call(t, http.MethodGet, "/api/v1/me/playtimes?limit=101", "/me/playtimes", "sess-alice", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeLimitTooLarge)

	resp, body = f.call(t, http.MethodGet, "/api/v1/me/playtimes?page=1000&limit=100", "/me/playtimes", "sess-alice", "", nil)
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeInvalidParameter)
}

func TestV1DeleteWorkPlaytimeAnswersWhatIsLeft(t *testing.T) {
	f := newG6Fix(t)
	f.user.mu.Lock()
	f.user.otherApp = map[int64]int{g6WorkLive: 300}
	f.user.mu.Unlock()
	resp, body := f.call(t, http.MethodDelete, g6Path(g6WorkLive)+"/playtime", "/works/{work_id}/playtime", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete %d %+v", resp.StatusCode, body)
	}
	if asInt(body["minutes"]) != 300 {
		t.Errorf("another app's minutes survive the forum's withdraw and must be what the response says: %+v", body)
	}
}

func TestV1ListMyPlaytimesReusesTheSweepUntilTheForumWrites(t *testing.T) {
	f := newG6Fix(t)
	sweeps := func() int {
		f.user.mu.Lock()
		defer f.user.mu.Unlock()
		return f.user.playSweeps
	}
	list := func(query string) map[string]any {
		t.Helper()
		resp, body := f.call(t, http.MethodGet, "/api/v1/me/playtimes"+query, "/me/playtimes", "sess-alice", "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("list %s: %d %+v", query, resp.StatusCode, body)
		}
		return body
	}
	list("")
	list("?page=2&limit=1")
	if n := sweeps(); n != 1 {
		t.Fatalf("paging re-swept catalog: %d sweeps, want 1", n)
	}

	resp, body := f.call(t, http.MethodPut, g6Path(g6WorkLive)+"/playtime", "/works/{work_id}/playtime", "sess-alice", "", map[string]any{"minutes": 4321})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put %d %+v", resp.StatusCode, body)
	}
	after := list("?include_nsfw=true&limit=100")
	if n := sweeps(); n != 2 {
		t.Fatalf("the forum's own write left the cached sweep in place: %d sweeps, want 2", n)
	}
	found := false
	for _, it := range after["items"].([]any) {
		row := it.(map[string]any)
		if asInt(row["minutes"]) == 4321 {
			found = true
		}
	}
	if !found {
		t.Errorf("the write is missing from the next read: %+v", after["items"])
	}
}
