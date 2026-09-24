package app

import (
	"net/http"
	"testing"

	"kun-galgame-api/pkg/problem"
)

func TestV1UserPlaneBannedAndMerged(t *testing.T) {
	f := newG6Fix(t)
	resp, body := f.call(t, http.MethodPut, g6Path(g6WorkLive)+"/covers/"+idStr(g6CoverA)+"/vote",
		"/works/{work_id}/covers/{cover_id}/vote", "sess-banned", "", nil)
	wantCode(t, resp, body, http.StatusForbidden, problem.CodeAccountBanned)

	f.cat.movedWorks[g6WorkExtra] = g6WorkLive
	resp, body = f.call(t, http.MethodGet, g6Path(g6WorkExtra)+"/covers/"+idStr(g6CoverA),
		"/works/{work_id}/covers/{cover_id}", "", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeEntityMerged)
}

func TestV1PatchOwnFolderForbiddenIsPermissionRequired(t *testing.T) {
	f := newG6Fix(t)
	f.user.ownPatch403 = true
	resp, body := f.call(t, http.MethodPatch, g6col(g6FolderAlicePub), "/collections/{collection_id}", "sess-alice", "",
		map[string]any{"title": "x"})
	wantCode(t, resp, body, http.StatusForbidden, problem.CodePermissionRequired)
}

func TestV1ListCollectionWorks(t *testing.T) {
	f := newG6Fix(t)
	for _, sess := range []string{"sess-alice", "", ""} {
		resp, body := f.call(t, http.MethodGet, g6col(g6FolderAliceDef)+"/works",
			"/collections/{collection_id}/works", sess, "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("works %q %d %+v", sess, resp.StatusCode, body)
		}
		items, _ := body["items"].([]any)
		if len(items) != 1 {
			t.Errorf("items %+v", items)
		}
	}
	f.user.mu.Lock()
	pub, my := f.user.pubReads, f.user.myReads
	f.user.mu.Unlock()
	if my != 0 {
		t.Errorf("the owner's public folder was read with their token %d times", my)
	}
	if pub != 1 {
		t.Errorf("public item reads %d, want 1 (the rest from cache)", pub)
	}
}

func TestV1CookieStaffCanPatchOthers(t *testing.T) {
	f := newG6Fix(t)
	resp, body := f.call(t, http.MethodPatch, g6col(g6FolderAlicePub), "/collections/{collection_id}", "sess-staff", "",
		map[string]any{"title": "staff-edit"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("staff patch %d %+v", resp.StatusCode, body)
	}
	if f.user.modPatch != 1 {
		t.Errorf("moderation calls %d", f.user.modPatch)
	}
}

func TestV1MembershipDoesNotTouchOtherFolders(t *testing.T) {
	f := newG6Fix(t)
	path := g6col(g6FolderAlicePub) + "/works/" + idStr(g6WorkExtra)
	spec := "/collections/{collection_id}/works/{work_id}"
	resp, body := f.call(t, http.MethodPut, path, spec, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put %d %+v", resp.StatusCode, body)
	}
	for _, call := range f.user.deleteItems {
		t.Errorf("unexpected delete %s", call)
	}
	if len(f.user.putItems) != 1 || f.user.putItems[0] != idStr(int(g6FolderAlicePub))+"/"+idStr(g6WorkExtra) {
		t.Errorf("putItems %v", f.user.putItems)
	}

	resp, body = f.call(t, http.MethodPut, g6col(g6FolderAlicePub)+"/works/"+idStr(g6WorkLive), spec, "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put held work %d %+v", resp.StatusCode, body)
	}
	if len(f.user.deleteItems) != 0 {
		t.Errorf("adding a work held in the default folder removed it elsewhere: %v", f.user.deleteItems)
	}
}
