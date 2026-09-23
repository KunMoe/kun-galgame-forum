package app

import (
	"net/http"
	"strconv"
	"sync"
	"testing"

	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/problem"
)

func TestV1CreateWorkResource(t *testing.T) {
	f := newResourceFix(t, nil)
	resp, got := f.rs(t, http.MethodPost, "/api/v1/works/"+idStr(g3WorkUnpub)+"/resources",
		"/works/{work_id}/resources", "sess-alice", keyUUID(1), createResourceBody(nil))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, got)
	}
	id := strID(got["id"])
	if resp.Header.Get("Location") != "/api/v1/galgame-resources/"+id {
		t.Errorf("location %s", resp.Header.Get("Location"))
	}
	if f.claim.n.Load() != 1 {
		t.Errorf("first create claim calls %d, want 1", f.claim.n.Load())
	}
	aw := f.snapshotAwards()
	if len(aw) != 1 || aw[0].delta != 3 || aw[0].reason != moemoepoint.ReasonContentApproved ||
		aw[0].key != moemoepoint.Key("galgame_resource_create", id) {
		t.Errorf("award %+v", aw)
	}

	resp, got = f.rs(t, http.MethodPost, "/api/v1/works/"+idStr(g3WorkUnpub)+"/resources",
		"/works/{work_id}/resources", "sess-alice", keyUUID(2), createResourceBody(map[string]any{"title": "second"}))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("second create %d %+v", resp.StatusCode, got)
	}
	if f.claim.n.Load() != 1 {
		t.Errorf("second create claim calls %d, want 1", f.claim.n.Load())
	}

	resp, got = f.rs(t, http.MethodPost, "/api/v1/works/"+idStr(g3WorkUnpub)+"/resources",
		"/works/{work_id}/resources", "sess-alice", "", createResourceBody(nil))
	wantCode(t, resp, got, http.StatusBadRequest, problem.CodeInvalidParameter)
	if e := errorHeader(got, "Idempotency-Key"); e == nil || e["reason"] != "REQUIRED" {
		t.Errorf("missing idempotency %+v", got["errors"])
	}

	resp, got = f.rs(t, http.MethodPost, "/api/v1/works/"+idStr(g3WorkUnpub)+"/resources",
		"/works/{work_id}/resources", "sess-alice", keyUUID(1), createResourceBody(map[string]any{"title": "other"}))
	wantCode(t, resp, got, http.StatusConflict, problem.CodeIdempotencyKeyReused)

	resp, got = f.rs(t, http.MethodPost, "/api/v1/works/"+idStr(g3WorkUnpub)+"/resources",
		"/works/{work_id}/resources", "", keyUUID(3), createResourceBody(nil))
	wantCode(t, resp, got, http.StatusUnauthorized, problem.CodeMissingCredential)

	resp, got = f.rs(t, http.MethodPost, "/api/v1/works/"+idStr(g3WorkUnpub)+"/resources",
		"/works/{work_id}/resources", "sess-banned", keyUUID(4), createResourceBody(nil))
	wantCode(t, resp, got, http.StatusForbidden, problem.CodeAccountBanned)

	resp, raw := f.doJSON(t, http.MethodPost, "/api/v1/works/"+idStr(g3WorkUnpub)+"/resources", "",
		"/works/{work_id}/resources", keyUUID(5),
		http.Header{"Authorization": {"Bearer no-such"}}, createResourceBody(nil))
	wantCode(t, resp, problemMap(t, raw), http.StatusUnauthorized, problem.CodeInvalidCredential)

	resp, got = f.rs(t, http.MethodPost, "/api/v1/works/"+idStr(g3WorkBanned)+"/resources",
		"/works/{work_id}/resources", "sess-alice", keyUUID(6), createResourceBody(nil))
	wantCode(t, resp, got, http.StatusForbidden, problem.CodeResourcePublishBanned)

	resp, got = f.rs(t, http.MethodPost, "/api/v1/works/"+idStr(g3WorkMiss)+"/resources",
		"/works/{work_id}/resources", "sess-alice", keyUUID(7), createResourceBody(nil))
	wantCode(t, resp, got, http.StatusNotFound, problem.CodeNotFound)
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame WHERE id = ?`, g3WorkMiss); n != 0 {
		t.Error("unknown work seeded a local row")
	}

	resp, got = f.rs(t, http.MethodPost, "/api/v1/works/"+idStr(g3WorkHidden)+"/resources",
		"/works/{work_id}/resources", "sess-alice", keyUUID(8), createResourceBody(nil))
	wantCode(t, resp, got, http.StatusNotFound, problem.CodeNotFound)

	resp, got = f.rs(t, http.MethodPost, "/api/v1/works/"+idStr(g3WorkSFW)+"/resources",
		"/works/{work_id}/resources", "sess-alice", keyUUID(9),
		createResourceBody(map[string]any{"size": "∞GB"}))
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)

	resp, got = f.rs(t, http.MethodPost, "/api/v1/works/"+idStr(g3WorkSFW)+"/resources",
		"/works/{work_id}/resources", "sess-alice", keyUUID(10),
		createResourceBody(map[string]any{"resource_platforms": []string{}, "resource_runtimes": []string{}}))
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
}

func TestV1CreateWorkResourceIdempotencyInProgress(t *testing.T) {
	f := newResourceFix(t, nil)
	body := createResourceBody(map[string]any{"title": "concurrent"})
	var (
		wg  sync.WaitGroup
		mu  sync.Mutex
		got []string
	)
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			resp, out := f.rs(t, http.MethodPost, "/api/v1/works/"+idStr(g3WorkNoLocal)+"/resources",
				"/works/{work_id}/resources", "sess-alice", keyUUID(90), body)
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

func TestV1CreateWorkResourceContentRejected(t *testing.T) {
	f := newResourceFix(t, denyChecker{})
	resp, got := f.rs(t, http.MethodPost, "/api/v1/works/"+idStr(g3WorkSFW)+"/resources",
		"/works/{work_id}/resources", "sess-alice", keyUUID(11),
		createResourceBody(map[string]any{"content_markdown": "blocked"}))
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeContentRejected)
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_resource WHERE note = 'blocked'`); n != 0 {
		t.Error("denied create wrote a row")
	}
}

func TestV1UpdateGalgameResourceTrustOnlyChangedText(t *testing.T) {
	f := newResourceFix(t, denyChecker{})
	resp, got := f.rs(t, http.MethodPatch, "/api/v1/galgame-resources/"+idStr(g3ResMain),
		"/galgame-resources/{resource_id}", "sess-alice", "",
		map[string]any{"resource_type": "collection"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch type %d %+v", resp.StatusCode, got)
	}
	if got["resource_type"] != "collection" {
		t.Errorf("type %+v", got["resource_type"])
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_resource WHERE id = ? AND type = 'collection'`, g3ResMain); n != 1 {
		t.Error("type not written")
	}

	resp, got = f.rs(t, http.MethodPatch, "/api/v1/galgame-resources/"+idStr(g3ResMain),
		"/galgame-resources/{resource_id}", "sess-alice", "",
		map[string]any{"content_markdown": "blocked"})
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeContentRejected)

	resp, got = f.rs(t, http.MethodPatch, "/api/v1/galgame-resources/"+idStr(g3ResMain),
		"/galgame-resources/{resource_id}", "sess-bob", "",
		map[string]any{"title": "nope"})
	wantCode(t, resp, got, http.StatusForbidden, problem.CodePermissionRequired)

	resp, got = f.rs(t, http.MethodPatch, "/api/v1/galgame-resources/"+idStr(g3ResBanned),
		"/galgame-resources/{resource_id}", "sess-grant", "",
		map[string]any{"title": "x"})
	wantCode(t, resp, got, http.StatusForbidden, problem.CodeResourcePublishBanned)

	resp, got = f.rs(t, http.MethodPatch, "/api/v1/galgame-resources/"+idStr(g3ResExpired),
		"/galgame-resources/{resource_id}", "sess-bob", "",
		map[string]any{"state": "expired"})
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if e := errorAt(got, "/state"); e == nil || e["reason"] != "NOT_ALLOWED_VALUE" {
		t.Errorf("expired state %+v", got["errors"])
	}

	resp, got = f.rs(t, http.MethodPatch, "/api/v1/galgame-resources/"+idStr(g3ResExpired),
		"/galgame-resources/{resource_id}", "sess-bob", "",
		map[string]any{"state": "valid"})
	if resp.StatusCode != http.StatusOK || got["state"] != "valid" {
		t.Errorf("mark valid %d %+v", resp.StatusCode, got)
	}
}

func TestV1DeleteGalgameResource(t *testing.T) {
	f := newResourceFix(t, nil)
	if err := f.db.Exec(`UPDATE galgame SET resource_count = 0 WHERE id = ?`, g3WorkCount0).Error; err != nil {
		t.Fatal(err)
	}
	resp, got := f.rs(t, http.MethodDelete, "/api/v1/galgame-resources/"+idStr(g3ResCount0),
		"/galgame-resources/{resource_id}", "sess-other", "", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete %d %+v", resp.StatusCode, got)
	}
	n := f.scalar(t, `SELECT resource_count FROM galgame WHERE id = ?`, g3WorkCount0)
	if n < 0 {
		t.Errorf("resource_count %d", n)
	}
	aw := f.snapshotAwards()
	found := false
	for _, a := range aw {
		if a.delta == -3 && a.reason == moemoepoint.ReasonContentRemoved {
			found = true
		}
	}
	if !found {
		t.Errorf("delete award %+v", aw)
	}
	resp, got = f.rs(t, http.MethodDelete, "/api/v1/galgame-resources/"+idStr(g3ResMain),
		"/galgame-resources/{resource_id}", "sess-bob", "", nil)
	wantCode(t, resp, got, http.StatusForbidden, problem.CodePermissionRequired)
}
