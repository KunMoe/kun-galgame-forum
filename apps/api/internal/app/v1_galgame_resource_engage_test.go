package app

import (
	"net/http"
	"testing"

	"kun-galgame-api/pkg/linkcheck"
	"kun-galgame-api/pkg/problem"
)

func TestV1PutDeleteGalgameResourceLike(t *testing.T) {
	f := newResourceFix(t, nil)
	before := f.scalar(t, `SELECT COUNT(*) FROM message WHERE sender_id = ? AND receiver_id = ? AND type = 'liked' AND link = ?`,
		w3UserBob, w3UserAlice, "/galgame/"+idStr(g3WorkSFW))
	resp, got := f.rs(t, http.MethodPut, "/api/v1/galgame-resources/"+idStr(g3ResMain)+"/like",
		"/galgame-resources/{resource_id}/like", "sess-alice", "", nil)
	wantCode(t, resp, got, http.StatusForbidden, problem.CodeSelfLikeForbidden)
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_resource_like WHERE galgame_resource_id = ? AND user_id = ?`, g3ResMain, w3UserAlice); n != 0 {
		t.Error("self like inserted a row")
	}

	resp, got = f.rs(t, http.MethodPut, "/api/v1/galgame-resources/"+idStr(g3ResMain)+"/like",
		"/galgame-resources/{resource_id}/like", "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("like %d %+v", resp.StatusCode, got)
	}
	v, _ := got["viewer"].(map[string]any)
	if v == nil || v["has_liked"] != true || asInt(got["like_count"]) != 1 {
		t.Errorf("like body %+v", got)
	}
	afterPut := f.scalar(t, `SELECT COUNT(*) FROM message WHERE sender_id = ? AND receiver_id = ? AND type = 'liked' AND link = ?`,
		w3UserBob, w3UserAlice, "/galgame/"+idStr(g3WorkSFW))
	if afterPut != before+1 {
		t.Errorf("liked message %d want %d", afterPut, before+1)
	}

	resp, got = f.rs(t, http.MethodPut, "/api/v1/galgame-resources/"+idStr(g3ResMain)+"/like",
		"/galgame-resources/{resource_id}/like", "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK || asInt(got["like_count"]) != 1 {
		t.Errorf("like again %d %+v", resp.StatusCode, got)
	}

	if err := f.db.Exec(`DELETE FROM message WHERE sender_id = ? AND receiver_id = ? AND type = 'liked' AND link = ?`,
		w3UserBob, w3UserAlice, "/galgame/"+idStr(g3WorkSFW)).Error; err != nil {
		t.Fatal(err)
	}
	resp, got = f.rs(t, http.MethodDelete, "/api/v1/galgame-resources/"+idStr(g3ResMain)+"/like",
		"/galgame-resources/{resource_id}/like", "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unlike %d %+v", resp.StatusCode, got)
	}
	v, _ = got["viewer"].(map[string]any)
	if v == nil || v["has_liked"] != false {
		t.Errorf("unlike viewer %+v", got)
	}
	afterDel := f.scalar(t, `SELECT COUNT(*) FROM message WHERE sender_id = ? AND receiver_id = ? AND type = 'liked' AND link = ?`,
		w3UserBob, w3UserAlice, "/galgame/"+idStr(g3WorkSFW))
	if afterDel != 0 {
		t.Errorf("unlike wrote a liked message, count %d", afterDel)
	}

	resp, got = f.rs(t, http.MethodDelete, "/api/v1/galgame-resources/"+idStr(g3ResMain)+"/like",
		"/galgame-resources/{resource_id}/like", "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK || asInt(got["like_count"]) != 0 {
		t.Errorf("unlike again %d %+v", resp.StatusCode, got)
	}
}

func TestV1CreateGalgameResourceExpiryReport(t *testing.T) {
	f := newResourceFix(t, nil)
	before := f.scalar(t, `SELECT COUNT(*) FROM message WHERE type = 'expired' AND link = ?`, "/galgame/"+idStr(g3WorkSFW))
	resp, got := f.rs(t, http.MethodPost, "/api/v1/galgame-resources/"+idStr(g3ResExpired)+"/expiry-reports",
		"/galgame-resources/{resource_id}/expiry-reports", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK || got["state"] != "expired" || got["verdict"] != "dead" {
		t.Fatalf("already expired %d %+v", resp.StatusCode, got)
	}
	after := f.scalar(t, `SELECT COUNT(*) FROM message WHERE type = 'expired' AND link = ?`, "/galgame/"+idStr(g3WorkSFW))
	if after != before {
		t.Errorf("already expired wrote a message %d -> %d", before, after)
	}

	f.shares.status = linkcheck.StatusAlive
	resp, got = f.rs(t, http.MethodPost, "/api/v1/galgame-resources/"+idStr(g3ResMain)+"/expiry-reports",
		"/galgame-resources/{resource_id}/expiry-reports", "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK || got["verdict"] != "alive" || got["state"] != "valid" {
		t.Errorf("alive %d %+v", resp.StatusCode, got)
	}

	f.shares.status = linkcheck.StatusUnknown
	resp, got = f.rs(t, http.MethodPost, "/api/v1/galgame-resources/"+idStr(g3ResMain)+"/expiry-reports",
		"/galgame-resources/{resource_id}/expiry-reports", "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK || got["verdict"] != "unchecked" || got["state"] != "expired" {
		t.Errorf("unchecked %d %+v", resp.StatusCode, got)
	}
}

func TestV1WorkResourcePublishBan(t *testing.T) {
	f := newResourceFix(t, nil)
	resp, got := f.rs(t, http.MethodPut, "/api/v1/works/"+idStr(g3WorkMiss)+"/resource-publish-ban",
		"/works/{work_id}/resource-publish-ban", "sess-staff", "", nil)
	wantCode(t, resp, got, http.StatusNotFound, problem.CodeNotFound)
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame WHERE id = ?`, g3WorkMiss); n != 0 {
		t.Error("ban PUT seeded an unknown work")
	}

	resp, got = f.rs(t, http.MethodPut, "/api/v1/works/"+idStr(g3WorkNoLocal)+"/resource-publish-ban",
		"/works/{work_id}/resource-publish-ban", "sess-staff", "", nil)
	if resp.StatusCode != http.StatusOK || got["is_resource_publish_banned"] != true {
		t.Fatalf("ban no-local %d %+v", resp.StatusCode, got)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame WHERE id = ? AND resource_publish_banned AND published = false`, g3WorkNoLocal); n != 1 {
		t.Error("ban insert published the work")
	}

	resp, got = f.rs(t, http.MethodDelete, "/api/v1/works/"+idStr(g3WorkNoLocal)+"/resource-publish-ban",
		"/works/{work_id}/resource-publish-ban", "sess-staff", "", nil)
	if resp.StatusCode != http.StatusOK || got["is_resource_publish_banned"] != false {
		t.Errorf("unban %d %+v", resp.StatusCode, got)
	}

	resp, raw := f.doJSON(t, http.MethodPut, "/api/v1/works/"+idStr(g3WorkSFW)+"/resource-publish-ban", "",
		"/works/{work_id}/resource-publish-ban", "",
		http.Header{"Authorization": {"Bearer staff-token"}}, nil)
	wantCode(t, resp, problemMap(t, raw), http.StatusForbidden, problem.CodePermissionRequired)

	resp, got = f.rs(t, http.MethodPut, "/api/v1/works/"+idStr(g3WorkSFW)+"/resource-publish-ban",
		"/works/{work_id}/resource-publish-ban", "sess-alice", "", nil)
	wantCode(t, resp, got, http.StatusForbidden, problem.CodePermissionRequired)

	resp, got = f.rs(t, http.MethodDelete, "/api/v1/works/"+idStr(g3WorkMiss)+"/resource-publish-ban",
		"/works/{work_id}/resource-publish-ban", "sess-staff", "", nil)
	wantCode(t, resp, got, http.StatusNotFound, problem.CodeNotFound)
}
