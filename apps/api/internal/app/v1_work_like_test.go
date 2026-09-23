package app

import (
	"net/http"
	"testing"

	"kun-galgame-api/pkg/problem"
)

func TestV1PutWorkLikeSelfForbidden(t *testing.T) {
	f := newWorkFix(t)
	before := f.snapshotAwards()
	resp, body := f.wk(t, http.MethodPut, g4WorkPath(g4WorkOwner)+"/like", "/works/{work_id}/like", "sess-alice", nil)
	wantCode(t, resp, body, http.StatusForbidden, problem.CodeSelfLikeForbidden)
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_like WHERE work_id = ? AND user_id = ?`, g4WorkOwner, w3UserAlice); n != 0 {
		t.Error("self like inserted a row")
	}
	if len(f.snapshotAwards()) != len(before) {
		t.Error("self like awarded moemoepoint")
	}
}

func TestV1PutWorkLikeNoOwnerSkipsAwardAndMessage(t *testing.T) {
	f := newWorkFix(t)
	beforeAwards := f.snapshotAwards()
	beforeMsg := f.scalar(t, `SELECT COUNT(*) FROM message WHERE type = 'liked' AND link = ?`, "/galgame/"+idStr(g4WorkNoOwner))
	resp, body := f.wk(t, http.MethodPut, g4WorkPath(g4WorkNoOwner)+"/like", "/works/{work_id}/like", "sess-alice", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("like %d %+v", resp.StatusCode, body)
	}
	v, _ := body["viewer"].(map[string]any)
	if v["has_liked"] != true || asInt(body["like_count"]) != 1 {
		t.Errorf("body %+v", body)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame_like WHERE work_id = ? AND user_id = ?`, g4WorkNoOwner, w3UserAlice); n != 1 {
		t.Error("like row missing")
	}
	if len(f.snapshotAwards()) != len(beforeAwards) {
		t.Error("no-owner like awarded moemoepoint")
	}
	afterMsg := f.scalar(t, `SELECT COUNT(*) FROM message WHERE type = 'liked' AND link = ?`, "/galgame/"+idStr(g4WorkNoOwner))
	if afterMsg != beforeMsg {
		t.Error("no-owner like wrote a liked message")
	}
}

func TestV1PutWorkLikeIdempotent(t *testing.T) {
	f := newWorkFix(t)
	resp, body := f.wk(t, http.MethodPut, g4WorkPath(g4WorkLive)+"/like", "/works/{work_id}/like", "sess-bob", nil)
	if resp.StatusCode != http.StatusOK || asInt(body["like_count"]) != 1 {
		t.Fatalf("like %d %+v", resp.StatusCode, body)
	}
	rows := f.scalar(t, `SELECT COUNT(*) FROM galgame_like WHERE work_id = ? AND user_id = ?`, g4WorkLive, w3UserBob)
	resp, body = f.wk(t, http.MethodPut, g4WorkPath(g4WorkLive)+"/like", "/works/{work_id}/like", "sess-bob", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("like again %d %+v", resp.StatusCode, body)
	}
	v, _ := body["viewer"].(map[string]any)
	if v["has_liked"] != true {
		t.Errorf("has_liked %+v", v)
	}
	if f.scalar(t, `SELECT COUNT(*) FROM galgame_like WHERE work_id = ? AND user_id = ?`, g4WorkLive, w3UserBob) != rows {
		t.Error("second PUT inserted another row")
	}
}

func TestV1DeleteWorkLikeIdempotentAndNonNegative(t *testing.T) {
	f := newWorkFix(t)
	resp, body := f.wk(t, http.MethodDelete, g4WorkPath(g4WorkLive)+"/like", "/works/{work_id}/like", "sess-bob", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unlike empty %d %+v", resp.StatusCode, body)
	}
	v, _ := body["viewer"].(map[string]any)
	if v["has_liked"] != false {
		t.Errorf("has_liked %+v", v)
	}
	if asInt(body["like_count"]) != 0 {
		t.Errorf("like_count %v", body["like_count"])
	}
	if n := f.scalar(t, `SELECT like_count FROM galgame WHERE id = ?`, g4WorkLive); n < 0 {
		t.Errorf("like_count column %d", n)
	}
	resp, body = f.wk(t, http.MethodPut, g4WorkPath(g4WorkLive)+"/like", "/works/{work_id}/like", "sess-bob", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("like %d %+v", resp.StatusCode, body)
	}
	resp, body = f.wk(t, http.MethodDelete, g4WorkPath(g4WorkLive)+"/like", "/works/{work_id}/like", "sess-bob", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unlike %d %+v", resp.StatusCode, body)
	}
	beforeMsg := f.scalar(t, `SELECT COUNT(*) FROM message WHERE type = 'liked' AND sender_id = ? AND link = ?`,
		w3UserBob, "/galgame/"+idStr(g4WorkLive))
	resp, body = f.wk(t, http.MethodDelete, g4WorkPath(g4WorkLive)+"/like", "/works/{work_id}/like", "sess-bob", nil)
	if resp.StatusCode != http.StatusOK || asInt(body["like_count"]) != 0 {
		t.Errorf("unlike again %d %+v", resp.StatusCode, body)
	}
	afterMsg := f.scalar(t, `SELECT COUNT(*) FROM message WHERE type = 'liked' AND sender_id = ? AND link = ?`,
		w3UserBob, "/galgame/"+idStr(g4WorkLive))
	if afterMsg != beforeMsg {
		t.Error("unlike wrote a liked message")
	}

	if _, body := f.wk(t, http.MethodPut, g4WorkPath(g4WorkLive)+"/like", "/works/{work_id}/like", "sess-bob", nil); body["code"] != nil {
		t.Fatalf("relike %+v", body)
	}
	if err := f.db.Exec(`UPDATE galgame SET like_count = 0 WHERE id = ?`, g4WorkLive).Error; err != nil {
		t.Fatal(err)
	}
	resp, body = f.wk(t, http.MethodDelete, g4WorkPath(g4WorkLive)+"/like", "/works/{work_id}/like", "sess-bob", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unlike at a drifted zero %d %+v", resp.StatusCode, body)
	}
	if n := f.scalar(t, `SELECT like_count FROM galgame WHERE id = ?`, g4WorkLive); n != 0 {
		t.Errorf("like_count went below zero: %d", n)
	}
}

func TestV1PutWorkLikeUnknownWritesNothing(t *testing.T) {
	f := newWorkFix(t)
	resp, body := f.wk(t, http.MethodPut, g4WorkPath(g4WorkUnknown)+"/like", "/works/{work_id}/like", "sess-alice", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	if n := f.scalar(t, `SELECT COUNT(*) FROM galgame WHERE id = ?`, g4WorkUnknown); n != 0 {
		t.Error("unknown like created a local row")
	}
	resp, body = f.wk(t, http.MethodPut, g4WorkPath(g4WorkHidden)+"/like", "/works/{work_id}/like", "sess-alice", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
}
