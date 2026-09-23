package app

import (
	"net/http"
	"strconv"
	"sync"
	"testing"

	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/problem"
)

func TestV1CreateToolset(t *testing.T) {
	f := newToolsetFix(t, nil)
	body := createToolsetBody("  New Tool  ", map[string]any{
		"content_markdown": "desc",
		"aliases":          []string{" alt ", "other"},
		"homepage_urls":    []string{"https://home.example"},
	})
	resp, got := f.ts(t, http.MethodPost, "/api/v1/toolsets", "/toolsets", "sess-alice", keyUUID(1), body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, got)
	}
	id := strID(got["id"])
	if resp.Header.Get("Location") != "/api/v1/toolsets/"+id || got["title"] != "New Tool" {
		t.Errorf("created %s %+v", resp.Header.Get("Location"), got)
	}
	aw := f.snapshotAwards()
	if len(aw) != 1 || aw[0].delta != 3 || aw[0].reason != moemoepoint.ReasonContentApproved ||
		aw[0].ref != moemoepoint.Ref("toolset", asInt(got["id"])) || aw[0].key != moemoepoint.Key("toolset_create", id) {
		t.Errorf("award %+v", aw)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_toolset_contributor WHERE toolset_id = ? AND user_id = ?`, asInt(id), w3UserAlice); n != 1 {
		t.Errorf("contributor %d", n)
	}

	resp, got = f.ts(t, http.MethodPost, "/api/v1/toolsets", "/toolsets", "sess-alice", "", createToolsetBody("x", nil))
	wantCode(t, resp, got, http.StatusBadRequest, problem.CodeInvalidParameter)
	if e := errorHeader(got, "Idempotency-Key"); e == nil || e["reason"] != "REQUIRED" {
		t.Errorf("missing key errors %+v", got["errors"])
	}

	resp, got = f.ts(t, http.MethodPost, "/api/v1/toolsets", "/toolsets", "sess-alice", keyUUID(1), createToolsetBody("other", nil))
	wantCode(t, resp, got, http.StatusConflict, problem.CodeIdempotencyKeyReused)

	resp, got = f.ts(t, http.MethodPost, "/api/v1/toolsets", "/toolsets", "", keyUUID(2), createToolsetBody("x", nil))
	wantCode(t, resp, got, http.StatusUnauthorized, problem.CodeMissingCredential)

	resp, got = f.ts(t, http.MethodPost, "/api/v1/toolsets", "/toolsets", "sess-banned", keyUUID(3), createToolsetBody("x", nil))
	wantCode(t, resp, got, http.StatusForbidden, problem.CodeAccountBanned)

	resp, raw := f.doJSON(t, http.MethodPost, "/api/v1/toolsets", "", "/toolsets", keyUUID(4),
		http.Header{"Authorization": {"Bearer no-such"}}, createToolsetBody("x", nil))
	wantCode(t, resp, problemMap(t, raw), http.StatusUnauthorized, problem.CodeInvalidCredential)

	resp, got = f.ts(t, http.MethodPost, "/api/v1/toolsets", "/toolsets", "sess-alice", keyUUID(5), createToolsetBody("   ", nil))
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if e := errorAt(got, "/title"); e == nil || e["reason"] != "TOO_SHORT" {
		t.Errorf("blank name %+v", got["errors"])
	}
}

func TestV1CreateToolsetIdempotencyInProgress(t *testing.T) {
	f := newToolsetFix(t, nil)
	body := createToolsetBody("concurrent", nil)
	var (
		wg  sync.WaitGroup
		mu  sync.Mutex
		got []string
	)
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			resp, out := f.ts(t, http.MethodPost, "/api/v1/toolsets", "/toolsets", "sess-alice", keyUUID(90), body)
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

func TestV1CreateToolsetContentRejected(t *testing.T) {
	f := newToolsetFix(t, denyChecker{})
	resp, got := f.ts(t, http.MethodPost, "/api/v1/toolsets", "/toolsets", "sess-alice", keyUUID(6), createToolsetBody("blocked", nil))
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeContentRejected)
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_toolset WHERE name = 'blocked'`); n != 0 {
		t.Error("denied create wrote a row")
	}
}

func TestV1UpdateToolsetTrustOnlyChangedText(t *testing.T) {
	f := newToolsetFix(t, denyChecker{})
	resp, got := f.ts(t, http.MethodPatch, "/api/v1/toolsets/"+idStr(g1TSMain), "/toolsets/{toolset_id}", "sess-alice", "",
		map[string]any{"toolset_type": "launcher"})
	if resp.StatusCode != http.StatusOK || got["toolset_type"] != "launcher" {
		t.Fatalf("type-only patch %d %+v", resp.StatusCode, got)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_toolset WHERE id = ? AND type = 'launcher'`, g1TSMain); n != 1 {
		t.Error("type was not written")
	}
	resp, got = f.ts(t, http.MethodPatch, "/api/v1/toolsets/"+idStr(g1TSMain), "/toolsets/{toolset_id}", "sess-alice", "",
		map[string]any{"title": "blocked"})
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeContentRejected)
}

func TestV1UpdateToolsetPermissionAndSource(t *testing.T) {
	f := newToolsetFix(t, nil)
	resp, got := f.ts(t, http.MethodGet, "/api/v1/toolsets/"+idStr(g1TSMain)+"/source", "/toolsets/{toolset_id}/source", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK || got["content_markdown"] != "hello world" {
		t.Errorf("source %d %+v", resp.StatusCode, got)
	}
	resp, got = f.ts(t, http.MethodGet, "/api/v1/toolsets/"+idStr(g1TSMain)+"/source", "/toolsets/{toolset_id}/source", "sess-bob", "", nil)
	wantCode(t, resp, got, http.StatusForbidden, problem.CodePermissionRequired)
	resp, got = f.ts(t, http.MethodPatch, "/api/v1/toolsets/"+idStr(g1TSMain), "/toolsets/{toolset_id}", "sess-bob", "",
		map[string]any{"title": "nope"})
	wantCode(t, resp, got, http.StatusForbidden, problem.CodePermissionRequired)
	resp, got = f.ts(t, http.MethodPatch, "/api/v1/toolsets/"+idStr(g1TSMain), "/toolsets/{toolset_id}", "sess-staff", "",
		map[string]any{"title": "Staff edit"})
	if resp.StatusCode != http.StatusOK || got["title"] != "Staff edit" {
		t.Errorf("staff patch %d %+v", resp.StatusCode, got)
	}
}

func TestV1DeleteToolset(t *testing.T) {
	f := newToolsetFix(t, nil)
	resp, got := f.ts(t, http.MethodDelete, "/api/v1/toolsets/"+idStr(g1TSBob), "/toolsets/{toolset_id}", "sess-alice", "", nil)
	wantCode(t, resp, got, http.StatusForbidden, problem.CodePermissionRequired)
	resp, _ = f.ts(t, http.MethodDelete, "/api/v1/toolsets/"+idStr(g1TSBob), "/toolsets/{toolset_id}", "sess-bob", "", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete %d", resp.StatusCode)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_toolset WHERE id = ?`, g1TSBob); n != 0 {
		t.Error("row remained")
	}
	aw := f.snapshotAwards()
	found := false
	for _, a := range aw {
		if a.key == moemoepoint.Key("toolset_delete", idStr(g1TSBob)) && a.delta == -3 {
			found = true
		}
	}
	if !found {
		t.Errorf("delete award %+v", aw)
	}
}
