package app

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"
)

func TestV1GetWorkCover(t *testing.T) {
	f := newG6Fix(t)
	resp, body := f.call(t, http.MethodGet, g6Path(g6WorkLive)+"/covers/"+idStr(g6CoverA),
		"/works/{work_id}/covers/{cover_id}", "", "", nil)
	if resp.StatusCode != http.StatusOK || body["object"] != "work_cover" {
		t.Fatalf("get cover %d %+v", resp.StatusCode, body)
	}
	if strID(body["id"]) != idStr(g6CoverA) {
		t.Errorf("id %+v", body["id"])
	}

	resp, body = f.call(t, http.MethodGet, g6Path(g6WorkLive)+"/covers/"+idStr(g6CoverOther),
		"/works/{work_id}/covers/{cover_id}", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)

	resp, body = f.call(t, http.MethodGet, g6Path(g6WorkHidden)+"/covers/"+idStr(g6CoverA),
		"/works/{work_id}/covers/{cover_id}", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)

	resp, body = f.call(t, http.MethodPut, g6Path(g6WorkExtra)+"/covers/"+idStr(g6CoverA)+"/vote",
		"/works/{work_id}/covers/{cover_id}/vote", "sess-alice", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	f.user.mu.Lock()
	voted := f.user.votes[g6WorkExtra]
	f.user.mu.Unlock()
	if voted != 0 {
		t.Errorf("a vote for another work's cover reached catalog: %d", voted)
	}
}

func TestV1GetWorkCoverTallyDegrades(t *testing.T) {
	f := newG6Fix(t)
	f.user.tallyErr = catalogclient.ErrUpstream
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	resp, body := f.call(t, http.MethodGet, g6Path(g6WorkLive)+"/covers/"+idStr(g6CoverA),
		"/works/{work_id}/covers/{cover_id}", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("degrade %d %+v", resp.StatusCode, body)
	}
	if asInt(body["vote_count"]) != 0 {
		t.Errorf("vote_count %+v", body["vote_count"])
	}
	if !strings.Contains(buf.String(), "vote tallies unavailable") {
		t.Errorf("log %q", buf.String())
	}
}

func TestV1PutWorkCoverVoteIdempotent(t *testing.T) {
	f := newG6Fix(t)
	path := g6Path(g6WorkLive) + "/covers/" + idStr(g6CoverA) + "/vote"
	spec := "/works/{work_id}/covers/{cover_id}/vote"
	resp, body := f.call(t, http.MethodPut, path, spec, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("vote %d %+v", resp.StatusCode, body)
	}
	v, _ := body["viewer"].(map[string]any)
	if v["has_voted"] != true {
		t.Errorf("has_voted %+v", v)
	}
	count := asInt(body["vote_count"])
	resp, body = f.call(t, http.MethodPut, path, spec, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("vote again %d %+v", resp.StatusCode, body)
	}
	v, _ = body["viewer"].(map[string]any)
	if v["has_voted"] != true || asInt(body["vote_count"]) != count {
		t.Errorf("second PUT must keep the vote: %+v", body)
	}
}

func TestV1DeleteWorkCoverVoteIdempotent(t *testing.T) {
	f := newG6Fix(t)
	path := g6Path(g6WorkLive) + "/covers/" + idStr(g6CoverA) + "/vote"
	spec := "/works/{work_id}/covers/{cover_id}/vote"
	resp, body := f.call(t, http.MethodDelete, path, spec, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unvote empty %d %+v", resp.StatusCode, body)
	}
	v, _ := body["viewer"].(map[string]any)
	if v["has_voted"] != false {
		t.Errorf("has_voted %+v", v)
	}
}

func TestV1CoverVoteScopeAndMissing(t *testing.T) {
	f := newG6Fix(t)
	path := g6Path(g6WorkLive) + "/covers/" + idStr(g6CoverA) + "/vote"
	spec := "/works/{work_id}/covers/{cover_id}/vote"
	resp, body := f.call(t, http.MethodPut, path, spec, "sess-noscope", "", nil)
	wantCode(t, resp, body, http.StatusForbidden, problem.CodeScopeRequired)

	resp, body = f.call(t, http.MethodPut, path, spec, "", "", nil)
	wantCode(t, resp, body, http.StatusUnauthorized, problem.CodeMissingCredential)

	resp, raw := f.doJSON(t, http.MethodPut, path, "", spec, "", http.Header{"Authorization": {"Bearer no-such"}}, nil)
	wantCode(t, resp, problemMap(t, raw), http.StatusUnauthorized, problem.CodeInvalidCredential)

	resp, body = f.call(t, http.MethodPut, g6Path(g6WorkUnknown)+"/covers/"+idStr(g6CoverA)+"/vote", spec, "sess-alice", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
}

func TestV1CoverVote429(t *testing.T) {
	f := newG6Fix(t)
	path := g6Path(g6WorkLive) + "/covers/" + idStr(g6CoverA) + "/vote"
	spec := "/works/{work_id}/covers/{cover_id}/vote"
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	resp, body := f.call(t, http.MethodPut, path, spec, "sess-429", "", nil)
	wantCode(t, resp, body, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
	if resp.Header.Get("Retry-After") != "30" {
		t.Errorf("Retry-After %q", resp.Header.Get("Retry-After"))
	}
	wantThrottleLoggedOnce(t, buf.String(), resp.Header.Get("X-Request-ID"))

	resp, body = f.call(t, http.MethodPut, path, spec, "sess-429q", "", nil)
	wantCode(t, resp, body, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
	if resp.Header.Get("Retry-After") != "3600" {
		t.Errorf("quota Retry-After %q", resp.Header.Get("Retry-After"))
	}
}
