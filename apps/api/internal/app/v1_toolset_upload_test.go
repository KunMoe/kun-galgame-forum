package app

import (
	"net/http"
	"testing"

	"kun-galgame-api/pkg/problem"
)

func TestV1CreateToolsetUpload(t *testing.T) {
	f := newToolsetFix(t, nil)
	resp, got := f.ts(t, http.MethodPost, "/api/v1/toolsets/"+idStr(g1TSMain)+"/uploads",
		"/toolsets/{toolset_id}/uploads", "sess-alice", keyUUID(40), map[string]any{
			"filename": "pack.zip", "file_size": 2048,
		})
	if resp.StatusCode != http.StatusCreated || got["object"] != "toolset_upload" || got["state"] != "pending" {
		t.Fatalf("init %d %+v", resp.StatusCode, got)
	}
	id := strID(got["id"])
	if resp.Header.Get("Location") != "/api/v1/toolsets/"+idStr(g1TSMain)+"/uploads/"+id {
		t.Errorf("Location %s", resp.Header.Get("Location"))
	}

	resp, got = f.ts(t, http.MethodPost, "/api/v1/toolsets/"+idStr(g1TSMain)+"/uploads",
		"/toolsets/{toolset_id}/uploads", "sess-alice", keyUUID(41), map[string]any{
			"filename": "pack.exe", "file_size": 2048,
		})
	wantCode(t, resp, got, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if e := errorAt(got, "/filename"); e == nil || e["reason"] != "INVALID_FORMAT" {
		t.Errorf("ext %+v", got["errors"])
	}

	resp, got = f.ts(t, http.MethodPost, "/api/v1/toolsets/"+idStr(g1TSGone)+"/uploads",
		"/toolsets/{toolset_id}/uploads", "sess-alice", keyUUID(42), map[string]any{
			"filename": "pack.zip", "file_size": 2048,
		})
	wantCode(t, resp, got, http.StatusNotFound, problem.CodeNotFound)
}

func TestV1UploadQuotaExceeded(t *testing.T) {
	f := newToolsetFix(t, nil)
	resp, got := f.ts(t, http.MethodPost, "/api/v1/toolsets/"+idStr(g1TSMain)+"/uploads",
		"/toolsets/{toolset_id}/uploads", "sess-alice", keyUUID(43), map[string]any{
			"filename": "huge.zip", "file_size": 200 * 1024 * 1024,
		})
	wantCode(t, resp, got, http.StatusTooManyRequests, problem.CodeQuotaExceeded)
	if resp.Header.Get("Retry-After") == "" {
		t.Error("missing Retry-After")
	}
	f.art.quota = true
	resp, got = f.ts(t, http.MethodPost, "/api/v1/toolsets/"+idStr(g1TSMain)+"/uploads",
		"/toolsets/{toolset_id}/uploads", "sess-staff", keyUUID(44), map[string]any{
			"filename": "x.zip", "file_size": 2048,
		})
	wantCode(t, resp, got, http.StatusTooManyRequests, problem.CodeQuotaExceeded)
}

func TestV1CompleteUploadOwnershipAndQuotaOnce(t *testing.T) {
	f := newToolsetFix(t, nil)
	before := f.scalar(t, `SELECT daily_toolset_upload_bytes FROM kungal_user_state WHERE user_id = ?`, w3UserAlice)
	path := "/api/v1/toolsets/" + idStr(g1TSMain) + "/uploads/" + g1UpPend
	spec := "/toolsets/{toolset_id}/uploads/{upload_id}"
	resp, got := f.ts(t, http.MethodPatch, path, spec, "sess-bob", "", map[string]any{"state": "completed"})
	wantCode(t, resp, got, http.StatusNotFound, problem.CodeNotFound)
	if n := f.scalar(t, `SELECT daily_toolset_upload_bytes FROM kungal_user_state WHERE user_id = ?`, w3UserAlice); n != before {
		t.Errorf("bob complete moved quota %d -> %d", before, n)
	}
	if n := f.scalar(t, `SELECT daily_toolset_upload_bytes FROM kungal_user_state WHERE user_id = ?`, w3UserBob); n != 0 {
		t.Errorf("bob quota %d", n)
	}

	resp, got = f.ts(t, http.MethodPatch, path, spec, "sess-alice", "", map[string]any{"state": "pending"})
	wantCode(t, resp, got, http.StatusConflict, problem.CodeInvalidStateTransition)

	resp, got = f.ts(t, http.MethodPatch, path, spec, "sess-alice", "", map[string]any{"state": "completed"})
	if resp.StatusCode != http.StatusOK || got["state"] != "completed" {
		t.Fatalf("complete %d %+v", resp.StatusCode, got)
	}
	mid := f.scalar(t, `SELECT daily_toolset_upload_bytes FROM kungal_user_state WHERE user_id = ?`, w3UserAlice)
	if mid != before+1024 {
		t.Errorf("quota after first complete %d, want %d", mid, before+1024)
	}
	resp, got = f.ts(t, http.MethodPatch, path, spec, "sess-alice", "", map[string]any{"state": "completed"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("second complete %d %+v", resp.StatusCode, got)
	}
	if n := f.scalar(t, `SELECT daily_toolset_upload_bytes FROM kungal_user_state WHERE user_id = ?`, w3UserAlice); n != mid {
		t.Errorf("second complete counted quota again: %d", n)
	}
}

func TestV1AbortUpload(t *testing.T) {
	f := newToolsetFix(t, nil)
	path := "/api/v1/toolsets/" + idStr(g1TSMain) + "/uploads/" + g1UpAliceA
	spec := "/toolsets/{toolset_id}/uploads/{upload_id}"
	resp, got := f.ts(t, http.MethodDelete, path, spec, "sess-alice", "", nil)
	wantCode(t, resp, got, http.StatusConflict, problem.CodeInvalidStateTransition)

	f.art.failDelete[g1UpPend] = true
	path = "/api/v1/toolsets/" + idStr(g1TSMain) + "/uploads/" + g1UpPend
	resp, got = f.ts(t, http.MethodDelete, path, spec, "sess-alice", "", nil)
	wantCode(t, resp, got, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
	if n := f.scalar(t, `SELECT COUNT(*) FROM toolset_upload WHERE artifact_uuid = ?`, g1UpPend); n != 1 {
		t.Error("abort deleted the row after artifact failure")
	}

	f.art.failDelete[g1UpPend] = false
	resp, _ = f.ts(t, http.MethodDelete, path, spec, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("abort %d", resp.StatusCode)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM toolset_upload WHERE artifact_uuid = ?`, g1UpPend); n != 0 {
		t.Error("abort left the row")
	}
}

func TestV1GetToolsetUpload(t *testing.T) {
	f := newToolsetFix(t, nil)
	resp, got := f.ts(t, http.MethodGet, "/api/v1/toolsets/"+idStr(g1TSMain)+"/uploads/"+g1UpPend,
		"/toolsets/{toolset_id}/uploads/{upload_id}", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK || got["state"] != "pending" {
		t.Errorf("get pending %d %+v", resp.StatusCode, got)
	}
	resp, got = f.ts(t, http.MethodGet, "/api/v1/toolsets/"+idStr(g1TSMain)+"/uploads/"+g1UpPend,
		"/toolsets/{toolset_id}/uploads/{upload_id}", "sess-bob", "", nil)
	wantCode(t, resp, got, http.StatusNotFound, problem.CodeNotFound)
}
