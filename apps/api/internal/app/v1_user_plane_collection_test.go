package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"

	"github.com/gofiber/fiber/v3"
)

func TestV1CreateCollection(t *testing.T) {
	f := newG6Fix(t)
	resp, body := f.call(t, http.MethodPost, "/api/v1/collections", "/collections", "sess-alice", "",
		map[string]any{"title": "New", "visibility": "private"})
	wantCode(t, resp, body, http.StatusBadRequest, problem.CodeInvalidParameter)

	resp, body = f.call(t, http.MethodPost, "/api/v1/collections", "/collections", "sess-alice", keyUUID(1),
		map[string]any{"title": "New", "visibility": "private"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, body)
	}
	if resp.Header.Get("Location") != "/api/v1/collections/"+strID(body["id"]) {
		t.Errorf("Location %q", resp.Header.Get("Location"))
	}
	if body["object"] != "collection" || body["visibility"] != "private" {
		t.Errorf("created %+v", body)
	}

	resp, body = f.call(t, http.MethodPost, "/api/v1/collections", "/collections", "sess-alice", keyUUID(1),
		map[string]any{"title": "Other", "visibility": "public"})
	wantCode(t, resp, body, http.StatusConflict, problem.CodeIdempotencyKeyReused)

	resp, body = f.call(t, http.MethodPost, "/api/v1/collections", "/collections", "sess-alice", keyUUID(2),
		map[string]any{"title": "New", "visibility": "restricted"})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	errs, _ := body["errors"].([]any)
	if len(errs) != 1 || errs[0].(map[string]any)["reason"] != problem.ReasonUnknownValue {
		t.Errorf("a body enum outside infra's set is one UNKNOWN_VALUE error: %+v", errs)
	}
}

func TestV1CreateCollectionRejected(t *testing.T) {
	f := newG6FixDeny(t)
	resp, body := f.call(t, http.MethodPost, "/api/v1/collections", "/collections", "sess-alice", keyUUID(3),
		map[string]any{"title": "bad", "visibility": "public"})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeContentRejected)
}

func TestV1GetCollectionVisibility(t *testing.T) {
	f := newG6Fix(t)
	resp, body := f.call(t, http.MethodGet, g6col(g6FolderBobPriv), "/collections/{collection_id}", "sess-alice", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	if hasPrivateLeak(body) {
		t.Errorf("private leak %+v", body)
	}

	resp, body = f.call(t, http.MethodGet, g6col(g6FolderBobPriv), "/collections/{collection_id}", "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("owner private %d %+v", resp.StatusCode, body)
	}

	resp, body = f.call(t, http.MethodGet, g6col(g6FolderBannedPub), "/collections/{collection_id}", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)

	resp, body = f.call(t, http.MethodGet, g6col(g6FolderAliceDef), "/collections/{collection_id}", "", "", nil)
	if resp.StatusCode != http.StatusOK || body["viewer"] != nil {
		t.Errorf("anon public %d %+v", resp.StatusCode, body)
	}
}

func TestV1PatchAndDeleteCollection(t *testing.T) {
	f := newG6Fix(t)
	resp, body := f.call(t, http.MethodPatch, g6col(g6FolderAlicePub), "/collections/{collection_id}", "sess-alice", "",
		map[string]any{})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)

	resp, body = f.call(t, http.MethodPatch, g6col(g6FolderAlicePub), "/collections/{collection_id}", "sess-alice", "",
		map[string]any{"title": "   "})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)
	if e := firstError(t, body); e["pointer"] != "/title" || e["reason"] != problem.ReasonTooShort {
		t.Errorf("blank title %+v", body["errors"])
	}

	resp, body = f.call(t, http.MethodPatch, g6col(g6FolderAlicePub), "/collections/{collection_id}", "sess-alice", "",
		map[string]any{"title": "Renamed"})
	if resp.StatusCode != http.StatusOK || body["title"] != "Renamed" {
		t.Fatalf("patch %d %+v", resp.StatusCode, body)
	}

	resp, body = f.call(t, http.MethodPatch, g6col(g6FolderAlicePub), "/collections/{collection_id}", "sess-alice", "",
		map[string]any{"is_default": false})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)

	resp, body = f.call(t, http.MethodDelete, g6col(g6FolderAliceDef), "/collections/{collection_id}", "sess-alice", "", nil)
	wantCode(t, resp, body, http.StatusConflict, problem.CodeInvalidStateTransition)
	if _, ok := f.user.folders[g6FolderAliceDef]; !ok {
		t.Error("default folder was deleted")
	}

	resp, body = f.call(t, http.MethodDelete, g6col(g6FolderAlicePub), "/collections/{collection_id}", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete %d %+v", resp.StatusCode, body)
	}
}

func TestV1CollectionStaffBearer(t *testing.T) {
	f := newG6Fix(t)
	resp, body := f.call(t, http.MethodGet, g6col(g6FolderAlicePub), "/collections/{collection_id}", "sess-staff", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("staff get %d %+v", resp.StatusCode, body)
	}
	v, _ := body["viewer"].(map[string]any)
	if v["can_edit"] != true || v["can_delete"] != true {
		t.Errorf("cookie staff can_* %+v", v)
	}

	hdr := http.Header{"Authorization": {"Bearer staff-token"}}
	resp, raw := f.doJSON(t, http.MethodGet, g6col(g6FolderAlicePub), "", "/collections/{collection_id}", "", hdr, nil)
	got := problemMap(t, raw)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bearer get %d %+v", resp.StatusCode, got)
	}
	v, _ = got["viewer"].(map[string]any)
	if v["can_edit"] != false || v["can_delete"] != false {
		t.Errorf("bearer can_* %+v", v)
	}

	resp, raw = f.doJSON(t, http.MethodPatch, g6col(g6FolderAlicePub), "", "/collections/{collection_id}", "",
		hdr, map[string]any{"title": "nope"})
	wantCode(t, resp, problemMap(t, raw), http.StatusNotFound, problem.CodeNotFound)
	if f.user.modPatch != 0 {
		t.Errorf("moderation patch calls %d", f.user.modPatch)
	}
}

func TestV1CollectionMembership(t *testing.T) {
	f := newG6Fix(t)
	path := g6col(g6FolderAlicePub) + "/works/" + idStr(g6WorkLive)
	spec := "/collections/{collection_id}/works/{work_id}"
	before := f.scalar(t, `SELECT favorite_count FROM galgame WHERE id = ?`, g6WorkLive)
	resp, body := f.call(t, http.MethodPut, path, spec, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put member %d %+v", resp.StatusCode, body)
	}
	v, _ := body["viewer"].(map[string]any)
	if v["has_work"] != true {
		t.Errorf("has_work %+v", v)
	}
	if f.scalar(t, `SELECT favorite_count FROM galgame WHERE id = ?`, g6WorkLive) != before {
		t.Error("second-folder add moved favorite_count")
	}

	resp, body = f.call(t, http.MethodDelete, g6col(g6FolderBobPriv)+"/works/"+idStr(g6WorkLive), spec, "sess-alice", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)

	resp, body = f.call(t, http.MethodGet, g6col(g6FolderAliceDef)+"/works/"+idStr(g6WorkLive), spec, "", "", nil)
	if resp.StatusCode != http.StatusOK || strID(body["id"]) != idStr(g6WorkLive) {
		t.Errorf("get member %d %+v", resp.StatusCode, body)
	}
	resp, body = f.call(t, http.MethodGet, g6col(g6FolderAliceDef)+"/works/"+idStr(g6WorkExtra), spec, "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
}

func TestV1ListMyCollectionsNoDefault(t *testing.T) {
	f := newG6Fix(t)
	before := f.user.folderCount()
	beforeAlias := f.scalar(t, `SELECT COUNT(*) FROM galgame_collection WHERE id BETWEEN 947100001 AND 947100099`)
	resp, body := f.call(t, http.MethodGet, "/api/v1/me/collections", "/me/collections", "sess-other", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("empty %d %+v", resp.StatusCode, body)
	}
	items, _ := body["items"].([]any)
	if len(items) != 0 {
		t.Errorf("items %+v", items)
	}
	if f.user.creates != 0 || f.user.folderCount() != before {
		t.Errorf("created a default folder: creates=%d", f.user.creates)
	}
	if f.scalar(t, `SELECT COUNT(*) FROM galgame_collection WHERE id BETWEEN 947100001 AND 947100099`) != beforeAlias {
		t.Error("minted an alias")
	}

	resp, body = f.call(t, http.MethodGet, "/api/v1/me/collections?work_id="+idStr(g6WorkLive), "/me/collections", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("mine %d %+v", resp.StatusCode, body)
	}
}

func TestV1ListUserCollections(t *testing.T) {
	f := newG6Fix(t)
	resp, body := f.call(t, http.MethodGet, "/api/v1/users/"+idStr(w3UserBanned)+"/collections",
		"/users/{user_id}/collections", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)

	resp, body = f.call(t, http.MethodGet, "/api/v1/users/"+idStr(w3UserBob)+"/collections",
		"/users/{user_id}/collections", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bob public %d %+v", resp.StatusCode, body)
	}
	for _, raw := range body["items"].([]any) {
		it, _ := raw.(map[string]any)
		if it["visibility"] == "private" {
			t.Errorf("private in public list %+v", it)
		}
	}
}

func TestV1CollectionAlias(t *testing.T) {
	f := newG6Fix(t)
	resp, body := f.call(t, http.MethodGet, "/api/v1/collection-aliases/"+idStr(g6AliasBob),
		"/collection-aliases/{alias_id}", "sess-alice", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)

	resp, body = f.call(t, http.MethodGet, "/api/v1/collection-aliases/"+idStr(g6AliasBob),
		"/collection-aliases/{alias_id}", "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK || strID(body["collection_id"]) != idStr(int(g6FolderBobPriv)) {
		t.Fatalf("owner alias %d %+v", resp.StatusCode, body)
	}

	before := f.scalar(t, `SELECT COUNT(*) FROM galgame_collection WHERE id BETWEEN 947100001 AND 947100099`)
	resp, body = f.call(t, http.MethodGet, "/api/v1/collection-aliases/"+idStr(g6AliasGone),
		"/collection-aliases/{alias_id}", "sess-alice", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	if f.scalar(t, `SELECT COUNT(*) FROM galgame_collection WHERE id BETWEEN 947100001 AND 947100099`) != before {
		t.Error("unknown cid minted a row")
	}
}

func TestV1NoBulkMembershipRoute(t *testing.T) {
	f := newG6Fix(t)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/works/"+idStr(g6WorkLive)+"/collections",
		strings.NewReader(`{"collection_ids":[]}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "sess-alice"})
	resp, err := f.Fiber.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatal("bulk membership route exists")
	}
	if len(f.user.putItems)+len(f.user.deleteItems) != 0 {
		t.Errorf("bulk wrote items put=%v del=%v", f.user.putItems, f.user.deleteItems)
	}
}

func TestV1DeleteCollectionRefusesWhenMembershipUnread(t *testing.T) {
	f := newG6Fix(t)
	f.user.mu.Lock()
	f.user.itemsErr = catalogclient.ErrUpstream
	f.user.mu.Unlock()
	resp, body := f.call(t, http.MethodDelete, g6col(g6FolderAlicePub), "/collections/{collection_id}", "sess-alice", "", nil)
	wantCode(t, resp, body, http.StatusServiceUnavailable, problem.CodeServiceUnavailable)
	f.user.mu.Lock()
	_, kept := f.user.folders[g6FolderAlicePub]
	f.user.mu.Unlock()
	if !kept {
		t.Error("the folder was deleted without knowing which works leave the library")
	}
}
