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

func TestV1ListMyCoverVotes(t *testing.T) {
	f := newG6Fix(t)
	list := "/api/v1/me/cover-votes?work_ids=" + idStr(g6WorkLive) + "," + idStr(g6WorkExtra) + "," + idStr(g6WorkHidden)
	spec := "/me/cover-votes"
	votePath := func(cover int) string { return g6Path(g6WorkLive) + "/covers/" + idStr(cover) + "/vote" }
	voteSpec := "/works/{work_id}/covers/{cover_id}/vote"
	voted := func(body map[string]any) map[string]any {
		out := map[string]any{}
		for _, raw := range body["items"].([]any) {
			it := raw.(map[string]any)
			out[strID(it["work_id"])] = it["voted_cover_id"]
		}
		return out
	}
	reads := func() int {
		f.user.mu.Lock()
		defer f.user.mu.Unlock()
		return f.user.coverVoteReads
	}

	if resp, body := f.call(t, http.MethodPut, votePath(g6CoverA), voteSpec, "sess-alice", "", nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("vote %d %+v", resp.StatusCode, body)
	}
	resp, body := f.call(t, http.MethodGet, list, spec, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	got := voted(body)
	if strID(got[idStr(g6WorkLive)]) != idStr(g6CoverA) || got[idStr(g6WorkExtra)] != nil {
		t.Errorf("votes %+v", got)
	}
	if _, ok := got[idStr(g6WorkHidden)]; ok {
		t.Errorf("a hidden work answered: %+v", got)
	}
	if missing, _ := body["missing"].([]any); len(missing) != 1 || strID(missing[0]) != idStr(g6WorkHidden) {
		t.Errorf("missing %+v", body["missing"])
	}

	f.call(t, http.MethodGet, list, spec, "sess-alice", "", nil)
	if n := reads(); n != 1 {
		t.Errorf("two reads cost %d catalog calls, want 1 (cached per user)", n)
	}

	if resp, body := f.call(t, http.MethodPut, votePath(g6CoverB), voteSpec, "sess-alice", "", nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("revote %d %+v", resp.StatusCode, body)
	}
	resp, body = f.call(t, http.MethodGet, list, spec, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK || strID(voted(body)[idStr(g6WorkLive)]) != idStr(g6CoverB) {
		t.Errorf("the forum's own vote left the cached list in place: %d %+v", resp.StatusCode, body)
	}

	resp, body = f.call(t, http.MethodGet, list, spec, "sess-noscope", "", nil)
	wantCode(t, resp, body, http.StatusForbidden, problem.CodeScopeRequired)
	resp, body = f.call(t, http.MethodGet, list, spec, "sess-429", "", nil)
	wantCode(t, resp, body, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
}

func TestV1CoverVoteDropsTheCachedTallies(t *testing.T) {
	f := newG6Fix(t)
	count := func() int {
		resp, body := f.call(t, http.MethodGet, g6Path(g6WorkLive), "/works/{work_id}", "", "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("get %d %+v", resp.StatusCode, body)
		}
		for _, raw := range body["covers"].([]any) {
			c := raw.(map[string]any)
			if strID(c["id"]) == idStr(g6CoverA) {
				return asInt(c["vote_count"])
			}
		}
		t.Fatalf("cover A missing %+v", body["covers"])
		return 0
	}
	before := count()
	if resp, body := f.call(t, http.MethodPut, g6Path(g6WorkLive)+"/covers/"+idStr(g6CoverA)+"/vote", "/works/{work_id}/covers/{cover_id}/vote", "sess-alice", "", nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("vote %d %+v", resp.StatusCode, body)
	}
	if after := count(); after != before+1 {
		t.Errorf("vote_count %d after the forum's own vote, want %d: the cached tallies were not dropped", after, before+1)
	}
}
