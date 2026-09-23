package app

import (
	"net/http"
	"testing"

	"kun-galgame-api/pkg/problem"
)

func TestV1PutToolsetPracticality(t *testing.T) {
	f := newToolsetFix(t, nil)
	resp, got := f.ts(t, http.MethodPut, "/api/v1/toolsets/"+idStr(g1TSMain)+"/practicality",
		"/toolsets/{toolset_id}/practicality", "sess-other", "", map[string]any{"rating": 3})
	if resp.StatusCode != http.StatusOK || got["object"] != "toolset_practicality" {
		t.Fatalf("put %d %+v", resp.StatusCode, got)
	}
	v, _ := got["viewer"].(map[string]any)
	if v == nil || asInt(v["practicality_rating"]) != 3 {
		t.Errorf("viewer %+v", got["viewer"])
	}
	if asInt(got["practicality_count"]) != 3 {
		t.Errorf("count %v", got["practicality_count"])
	}

	resp, got = f.ts(t, http.MethodPut, "/api/v1/toolsets/"+idStr(g1TSMain)+"/practicality",
		"/toolsets/{toolset_id}/practicality", "sess-other", "", map[string]any{"rating": 3})
	if resp.StatusCode != http.StatusOK || asInt(got["practicality_count"]) != 3 {
		t.Errorf("idempotent put %d %+v", resp.StatusCode, got)
	}

	before := f.scalar(t, `SELECT COUNT(*) FROM galgame_toolset_practicality`)
	resp, got = f.ts(t, http.MethodPut, "/api/v1/toolsets/"+idStr(g1TSGone)+"/practicality",
		"/toolsets/{toolset_id}/practicality", "sess-alice", "", map[string]any{"rating": 5})
	wantCode(t, resp, got, http.StatusNotFound, problem.CodeNotFound)
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_toolset_practicality`); n != before {
		t.Error("missing toolset wrote a rating")
	}
}
