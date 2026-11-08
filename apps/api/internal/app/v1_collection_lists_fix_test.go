package app

import (
	"context"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"testing"

	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"
)

func (u *g6User) FolderDirectory(_ context.Context, in catalogclient.DirectoryQuery) (*catalogclient.FolderPage, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	var out []catalogclient.Folder
	for _, f := range u.folders {
		if f.Visibility != "public" || f.IsDefault || f.Name == "" {
			continue
		}
		if !in.IncludeNSFW && f.ContentRating == "r18" {
			continue
		}
		if in.ContainsWorkID > 0 && !slices.ContainsFunc(u.items[f.ID], func(it catalogclient.FolderItem) bool {
			return it.WorkID == in.ContainsWorkID
		}) {
			continue
		}
		f.FollowerCount = u.followerCount(f.ID)
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].FollowerCount != out[j].FollowerCount {
			return out[i].FollowerCount > out[j].FollowerCount
		}
		return out[i].ID < out[j].ID
	})
	return &catalogclient.FolderPage{Folders: out}, nil
}

func (u *g6User) followerCount(folderID int64) int {
	n := 0
	for _, set := range u.folderFollows {
		if set[folderID] {
			n++
		}
	}
	return n
}

func (u *g6User) PatchFolderItem(_ context.Context, token string, folderID, workID int64, in catalogclient.FolderItemPatch) (*catalogclient.FolderItem, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	f, ok := u.folders[folderID]
	if !ok || f.OwnerUID != u.uid(token) {
		return nil, catalogclient.ErrNotFound
	}
	return u.patchItem(folderID, workID, in)
}

func (u *g6User) ModeratePatchFolderItem(_ context.Context, token string, folderID, workID int64, note string) (*catalogclient.FolderItem, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	u.modItemPatch++
	if _, ok := u.folders[folderID]; !ok {
		return nil, catalogclient.ErrNotFound
	}
	return u.patchItem(folderID, workID, catalogclient.FolderItemPatch{Note: &note})
}

func (u *g6User) patchItem(folderID, workID int64, in catalogclient.FolderItemPatch) (*catalogclient.FolderItem, error) {
	items := u.items[folderID]
	sort.SliceStable(items, func(i, j int) bool { return items[i].Position < items[j].Position })
	at := slices.IndexFunc(items, func(it catalogclient.FolderItem) bool { return it.WorkID == workID })
	if at < 0 {
		return nil, &catalogclient.UserAPIError{Status: http.StatusNotFound, Message: "work is not in this folder."}
	}
	moved := items[at]
	if in.Note != nil {
		moved.Note = *in.Note
	}
	if anchor := in.AfterWorkID + in.BeforeWorkID; anchor != "" {
		anchorID, _ := strconv.ParseInt(anchor, 10, 64)
		rest := slices.Delete(slices.Clone(items), at, at+1)
		to := slices.IndexFunc(rest, func(it catalogclient.FolderItem) bool { return it.WorkID == anchorID })
		if to < 0 {
			return nil, &catalogclient.UserAPIError{Status: http.StatusUnprocessableEntity, Message: "the move anchor must be another work in this folder."}
		}
		if in.AfterWorkID != "" {
			to++
		}
		items = slices.Insert(rest, to, moved)
	} else {
		items[at] = moved
	}
	for i := range items {
		items[i].Position = int64(i+1) * 1024
	}
	u.items[folderID] = items
	f := u.folders[folderID]
	f.UpdatedAt = u.touch()
	u.folders[folderID] = f
	for _, it := range items {
		if it.WorkID == workID {
			return &it, nil
		}
	}
	return nil, catalogclient.ErrNotFound
}

func (u *g6User) FollowFolder(_ context.Context, token string, folderID int64) (*catalogclient.Folder, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	f, ok := u.folders[folderID]
	if !ok || f.Visibility == "private" {
		return nil, catalogclient.ErrNotFound
	}
	uid := u.uid(token)
	if f.OwnerUID == uid {
		return nil, &catalogclient.UserAPIError{Status: http.StatusUnprocessableEntity, ProblemCode: "VALIDATION_FAILED", Message: "a folder's owner cannot follow it."}
	}
	if u.folderFollows == nil {
		u.folderFollows = map[int64]map[int64]bool{}
	}
	if u.folderFollows[uid] == nil {
		u.folderFollows[uid] = map[int64]bool{}
	}
	u.folderFollows[uid][folderID] = true
	f.FollowerCount = u.followerCount(folderID)
	u.folders[folderID] = f
	return &f, nil
}

func (u *g6User) UnfollowFolder(_ context.Context, token string, folderID int64) error {
	if err := u.gate(token); err != nil {
		return err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	delete(u.folderFollows[u.uid(token)], folderID)
	if f, ok := u.folders[folderID]; ok {
		f.FollowerCount = u.followerCount(folderID)
		u.folders[folderID] = f
	}
	return nil
}

func (u *g6User) FollowedFolders(_ context.Context, token, _ string, _ int) (*catalogclient.FolderPage, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	var out []catalogclient.Folder
	for id := range u.folderFollows[u.uid(token)] {
		if f, ok := u.folders[id]; ok && f.Visibility != "private" {
			f.FollowerCount = u.followerCount(id)
			out = append(out, f)
		}
	}
	return &catalogclient.FolderPage{Folders: out}, nil
}

func (u *g6User) FollowedAmong(_ context.Context, token string, folderIDs []int64) (map[int64]bool, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	out := map[int64]bool{}
	for _, id := range folderIDs {
		if u.folderFollows[u.uid(token)][id] {
			out[id] = true
		}
	}
	return out, nil
}

func (u *g6User) ReportFolder(_ context.Context, token string, in catalogclient.FolderReport, _ string) error {
	if err := u.gate(token); err != nil {
		return err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	id, _ := strconv.ParseInt(in.FolderID, 10, 64)
	if f, ok := u.folders[id]; !ok || f.Visibility != "public" {
		return catalogclient.ErrNotFound
	}
	u.reports = append(u.reports, in)
	return nil
}

func (f *fakeCatalogUser) FolderDirectory(context.Context, catalogclient.DirectoryQuery) (*catalogclient.FolderPage, error) {
	return &catalogclient.FolderPage{}, nil
}
func (f *fakeCatalogUser) PatchFolderItem(context.Context, string, int64, int64, catalogclient.FolderItemPatch) (*catalogclient.FolderItem, error) {
	return nil, catalogclient.ErrNotFound
}
func (f *fakeCatalogUser) ModeratePatchFolderItem(context.Context, string, int64, int64, string) (*catalogclient.FolderItem, error) {
	return nil, catalogclient.ErrNotFound
}
func (f *fakeCatalogUser) FollowFolder(context.Context, string, int64) (*catalogclient.Folder, error) {
	return nil, catalogclient.ErrNotFound
}
func (f *fakeCatalogUser) UnfollowFolder(context.Context, string, int64) error { return nil }
func (f *fakeCatalogUser) FollowedFolders(context.Context, string, string, int) (*catalogclient.FolderPage, error) {
	return &catalogclient.FolderPage{}, nil
}
func (f *fakeCatalogUser) FollowedAmong(context.Context, string, []int64) (map[int64]bool, error) {
	return map[int64]bool{}, nil
}
func (f *fakeCatalogUser) ReportFolder(context.Context, string, catalogclient.FolderReport, string) error {
	return catalogclient.ErrNotFound
}

func g6seedList(f *g6Fix) {
	u := f.user
	u.mu.Lock()
	defer u.mu.Unlock()
	pub := u.folders[g6FolderAlicePub]
	pub.ItemCount, pub.ContentRating = 3, "all_ages"
	u.folders[g6FolderAlicePub] = pub
	u.items[g6FolderAlicePub] = []catalogclient.FolderItem{
		{FolderID: g6FolderAlicePub, WorkID: g6WorkLive, Position: 1024, Note: "first", CreatedAt: "2026-09-03T00:00:00Z"},
		{FolderID: g6FolderAlicePub, WorkID: g6WorkOwner, Position: 2048, CreatedAt: "2026-09-04T00:00:00Z"},
		{FolderID: g6FolderAlicePub, WorkID: g6WorkExtra, Position: 3072, CreatedAt: "2026-09-05T00:00:00Z"},
	}
	bob := u.folders[g6FolderBobPub]
	bob.ContentRating = "r18"
	u.folders[g6FolderBobPub] = bob
}

func entryWorkIDs(body map[string]any) []string {
	var out []string
	items, _ := body["items"].([]any)
	for _, raw := range items {
		e := raw.(map[string]any)
		out = append(out, strID(e["work_summary"].(map[string]any)["id"]))
	}
	return out
}

func TestV1CollectionDirectory(t *testing.T) {
	f := newG6Fix(t)
	g6seedList(f)
	const spec = "/collections"

	resp, body := f.call(t, http.MethodGet, "/api/v1/collections", spec, "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("directory %d %+v", resp.StatusCode, body)
	}
	ids := map[string]bool{}
	for _, raw := range body["items"].([]any) {
		c := raw.(map[string]any)
		ids[strID(c["id"])] = true
		if c["follower_count"] == nil || c["content_rating"] == nil {
			t.Errorf("summary lacks follower_count/content_rating: %+v", c)
		}
	}
	if !ids[idStr(g6FolderAlicePub)] || ids[idStr(g6FolderAliceDef)] || ids[idStr(g6FolderBobPub)] {
		t.Errorf("directory without include_nsfw = %v; want Alice's public list, not her default nor Bob's r18 list", ids)
	}

	_, body = f.call(t, http.MethodGet, "/api/v1/collections?include_nsfw=true", spec, "", "", nil)
	if n := len(body["items"].([]any)); n < 2 {
		t.Errorf("include_nsfw=true lists %d collections, want the r18 one too", n)
	}

	_, body = f.call(t, http.MethodGet, "/api/v1/collections?contains_work_id="+idStr(g6WorkExtra), spec, "", "", nil)
	if items := body["items"].([]any); len(items) != 1 || strID(items[0].(map[string]any)["id"]) != idStr(g6FolderAlicePub) {
		t.Errorf("contains_work_id = %+v", items)
	}

	resp, body = f.call(t, http.MethodGet, "/api/v1/collections?sort=popular", spec, "", "", nil)
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("unknown sort %d %+v", resp.StatusCode, body)
	}
}

func TestV1CollectionFollow(t *testing.T) {
	f := newG6Fix(t)
	g6seedList(f)
	path := g6col(g6FolderAlicePub) + "/follow"
	const spec = "/collections/{collection_id}/follow"

	resp, body := f.call(t, http.MethodPut, path, spec, "", "", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous follow %d", resp.StatusCode)
	}

	resp, body = f.call(t, http.MethodPut, path, spec, "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK || body["follower_count"] != float64(1) ||
		body["viewer"].(map[string]any)["is_following"] != true {
		t.Fatalf("follow %d %+v", resp.StatusCode, body)
	}

	_, body = f.call(t, http.MethodGet, g6col(g6FolderAlicePub), "/collections/{collection_id}", "sess-bob", "", nil)
	if body["viewer"].(map[string]any)["is_following"] != true || body["follower_count"] != float64(1) {
		t.Errorf("detail after follow %+v", body)
	}

	_, body = f.call(t, http.MethodGet, "/api/v1/me/followed-collections", "/me/followed-collections", "sess-bob", "", nil)
	if items := body["items"].([]any); len(items) != 1 || strID(items[0].(map[string]any)["id"]) != idStr(g6FolderAlicePub) {
		t.Errorf("followed list %+v", body)
	}

	resp, body = f.call(t, http.MethodPut, path, spec, "sess-alice", "", nil)
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)

	resp, body = f.call(t, http.MethodPut, g6col(g6FolderBobPriv)+"/follow", spec, "sess-alice", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)

	resp, body = f.call(t, http.MethodDelete, path, spec, "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK || body["follower_count"] != float64(0) ||
		body["viewer"].(map[string]any)["is_following"] != false {
		t.Errorf("unfollow %d %+v", resp.StatusCode, body)
	}
}

func TestV1CollectionEntries(t *testing.T) {
	f := newG6Fix(t)
	g6seedList(f)
	path := g6col(g6FolderAlicePub) + "/entries"
	const spec = "/collections/{collection_id}/entries"

	resp, body := f.call(t, http.MethodGet, path, spec, "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("entries %d %+v", resp.StatusCode, body)
	}
	want := []string{idStr(g6WorkLive), idStr(g6WorkOwner), idStr(g6WorkExtra)}
	if got := entryWorkIDs(body); !slices.Equal(got, want) {
		t.Errorf("position order %v, want %v", got, want)
	}
	first := body["items"].([]any)[0].(map[string]any)
	if first["note"] != "first" || body["items"].([]any)[1].(map[string]any)["note"] != nil {
		t.Errorf("notes %+v", body["items"])
	}

	_, body = f.call(t, http.MethodGet, path+"?sort=favorited_desc", spec, "", "", nil)
	if got := entryWorkIDs(body); !slices.Equal(got, []string{want[2], want[1], want[0]}) {
		t.Errorf("favorited_desc %v", got)
	}

	item := g6col(g6FolderAlicePub) + "/works/" + idStr(g6WorkExtra)
	const itemSpec = "/collections/{collection_id}/works/{work_id}"
	resp, body = f.call(t, http.MethodPatch, item, itemSpec, "sess-alice", "",
		map[string]any{"after_work_id": idStr(g6WorkLive), "before_work_id": idStr(g6WorkOwner)})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)

	resp, body = f.call(t, http.MethodPatch, item, itemSpec, "sess-alice", "", map[string]any{})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)

	resp, body = f.call(t, http.MethodPatch, item, itemSpec, "sess-alice", "",
		map[string]any{"before_work_id": idStr(g6WorkLive), "note": "  moved  "})
	if resp.StatusCode != http.StatusOK || body["note"] != "moved" || body["object"] != "collection_item" {
		t.Fatalf("patch %d %+v", resp.StatusCode, body)
	}
	_, body = f.call(t, http.MethodGet, path, spec, "", "", nil)
	if got := entryWorkIDs(body); !slices.Equal(got, []string{want[2], want[0], want[1]}) {
		t.Errorf("after move %v", got)
	}

	resp, body = f.call(t, http.MethodPatch, item, itemSpec, "sess-bob", "", map[string]any{"note": "x"})
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)

	resp, body = f.call(t, http.MethodPatch, item, itemSpec, "sess-staff", "", map[string]any{"note": ""})
	if resp.StatusCode != http.StatusOK || body["note"] != nil || f.user.modItemPatch != 1 {
		t.Errorf("staff note clear %d %+v (moderation calls %d)", resp.StatusCode, body, f.user.modItemPatch)
	}
	resp, body = f.call(t, http.MethodPatch, item, itemSpec, "sess-staff", "", map[string]any{"after_work_id": idStr(g6WorkLive)})
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
}

func TestV1CollectionReport(t *testing.T) {
	f := newG6Fix(t)
	const spec = "/collection-reports"
	resp, body := f.call(t, http.MethodPost, "/api/v1/collection-reports", spec, "sess-bob", keyUUID(41),
		map[string]any{"collection_id": idStr(g6FolderAlicePub), "report_category": "spam", "note": "ads"})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("report %d %+v", resp.StatusCode, body)
	}
	if len(f.user.reports) != 1 || f.user.reports[0].Reason != "spam" || f.user.reports[0].Note != "ads" {
		t.Errorf("forwarded %+v", f.user.reports)
	}

	resp, body = f.call(t, http.MethodPost, "/api/v1/collection-reports", spec, "sess-bob", keyUUID(42),
		map[string]any{"collection_id": idStr(g6FolderBobPriv), "report_category": "spam"})
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)

	resp, body = f.call(t, http.MethodPost, "/api/v1/collection-reports", spec, "sess-bob", keyUUID(43),
		map[string]any{"collection_id": idStr(g6FolderAlicePub), "report_category": "boring"})
	wantCode(t, resp, body, http.StatusUnprocessableEntity, problem.CodeValidationFailed)

	resp, body = f.call(t, http.MethodPost, "/api/v1/collection-reports", spec, "sess-429", keyUUID(44),
		map[string]any{"collection_id": idStr(g6FolderAlicePub), "report_category": "spam"})
	wantCode(t, resp, body, http.StatusTooManyRequests, problem.CodeRateLimited)
}

func (u *g6User) MyFolderItem(_ context.Context, token string, folderID, workID int64) (*catalogclient.FolderItem, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if f, ok := u.folders[folderID]; !ok || f.OwnerUID != u.uid(token) {
		return nil, catalogclient.ErrNotFound
	}
	for _, it := range u.items[folderID] {
		if it.WorkID == workID {
			cp := it
			return &cp, nil
		}
	}
	return nil, catalogclient.ErrNotFound
}

func (f *fakeCatalogUser) MyFolderItem(context.Context, string, int64, int64) (*catalogclient.FolderItem, error) {
	return nil, catalogclient.ErrNotFound
}

func g6feedRows(t *testing.T, f *g6Fix, folderID int64) map[string]int {
	t.Helper()
	var rows []struct {
		Type   string
		WorkID int
	}
	if err := f.db.Raw(`SELECT type, work_id FROM feed_activity WHERE link = ?`,
		"/collection/"+strconv.FormatInt(folderID, 10)).Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	out := map[string]int{}
	for _, r := range rows {
		out[r.Type] = r.WorkID
	}
	return out
}

func TestV1CollectionViewsAndRecentWorks(t *testing.T) {
	f := newG6Fix(t)
	g6seedList(f)
	f.user.mu.Lock()
	pub := f.user.folders[g6FolderAlicePub]
	pub.RecentWorkIDs = []int64{g6WorkExtra, g6WorkNSFW, g6WorkOwner}
	f.user.folders[g6FolderAlicePub] = pub
	f.user.mu.Unlock()
	clearViews := func() {
		_ = f.db.Exec(`DELETE FROM galgame_collection_view WHERE catalog_folder_id = ?`, g6FolderAlicePub).Error
	}
	clearViews()
	t.Cleanup(clearViews)

	const spec = "/collections/{collection_id}"
	_, body := f.call(t, http.MethodGet, g6col(g6FolderAlicePub), spec, "", "", nil)
	_, body = f.call(t, http.MethodGet, g6col(g6FolderAlicePub), spec, "", "", nil)
	if body["view_count"] != float64(2) {
		t.Errorf("view_count after two reads = %v, want 2", body["view_count"])
	}
	var recent []string
	for _, w := range body["recent_works"].([]any) {
		recent = append(recent, strID(w.(map[string]any)["id"]))
	}
	if !slices.Equal(recent, []string{idStr(g6WorkExtra), idStr(g6WorkOwner)}) {
		t.Errorf("recent_works without include_nsfw = %v, want newest first and no adult work", recent)
	}

	_, body = f.call(t, http.MethodGet, "/api/v1/collections", "/collections", "", "", nil)
	for _, raw := range body["items"].([]any) {
		c := raw.(map[string]any)
		if strID(c["id"]) == idStr(g6FolderAlicePub) && c["view_count"] != float64(2) {
			t.Errorf("directory view_count = %v, want 2", c["view_count"])
		}
	}
}

func TestV1CollectionRelated(t *testing.T) {
	f := newG6Fix(t)
	g6seedList(f)
	f.user.mu.Lock()
	bob := f.user.folders[g6FolderBobPub]
	bob.ContentRating, bob.ItemCount = "all_ages", 1
	f.user.folders[g6FolderBobPub] = bob
	f.user.items[g6FolderBobPub] = []catalogclient.FolderItem{
		{FolderID: g6FolderBobPub, WorkID: g6WorkLive, Position: 1024, CreatedAt: "2026-09-06T00:00:00Z"},
	}
	f.user.mu.Unlock()

	resp, body := f.call(t, http.MethodGet, g6col(g6FolderAlicePub)+"/related", "/collections/{collection_id}/related", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("related %d %+v", resp.StatusCode, body)
	}
	ids := []string{}
	for _, raw := range body["items"].([]any) {
		ids = append(ids, strID(raw.(map[string]any)["id"]))
	}
	if slices.Contains(ids, idStr(g6FolderAlicePub)) || !slices.Contains(ids, idStr(g6FolderBobPub)) {
		t.Errorf("related = %v, want Bob's list sharing a work and never the folder itself", ids)
	}

	_, body = f.call(t, http.MethodGet, g6col(g6FolderBobPriv)+"/related", "/collections/{collection_id}/related", "sess-bob", "", nil)
	if n := len(body["items"].([]any)); n != 0 {
		t.Errorf("a private collection has %d related, want 0", n)
	}
}

func TestV1CollectionFeed(t *testing.T) {
	f := newG6Fix(t)
	const spec = "/collections"
	resp, body := f.call(t, http.MethodPost, "/api/v1/collections", spec, "sess-alice", keyUUID(61),
		map[string]any{"title": "Feed me", "visibility": "public"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %+v", resp.StatusCode, body)
	}
	id, _ := strconv.ParseInt(strID(body["id"]), 10, 64)
	t.Cleanup(func() {
		_ = f.db.Exec(`DELETE FROM feed_activity WHERE link = ?`, "/collection/"+strconv.FormatInt(id, 10)).Error
	})
	if _, ok := g6feedRows(t, f, id)["COLLECTION_CREATION"]; !ok {
		t.Fatalf("no creation row for a public collection")
	}

	itemPath := g6col(id) + "/works/" + idStr(g6WorkLive)
	resp, body = f.call(t, http.MethodPut, itemPath, "/collections/{collection_id}/works/{work_id}", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put work %d %+v", resp.StatusCode, body)
	}
	if got := g6feedRows(t, f, id)["COLLECTION_WORK_ADDITION"]; got != g6WorkLive {
		t.Errorf("addition row work_id = %d, want %d", got, g6WorkLive)
	}

	resp, body = f.call(t, http.MethodPatch, g6col(id), "/collections/{collection_id}", "sess-alice", "",
		map[string]any{"visibility": "private"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("make private %d %+v", resp.StatusCode, body)
	}
	if rows := g6feedRows(t, f, id); len(rows) != 0 {
		t.Errorf("a collection made private keeps feed rows %v", rows)
	}

	resp, body = f.call(t, http.MethodPost, "/api/v1/collections", spec, "sess-alice", keyUUID(62),
		map[string]any{"title": "Secret", "visibility": "private"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create private %d %+v", resp.StatusCode, body)
	}
	secret, _ := strconv.ParseInt(strID(body["id"]), 10, 64)
	if rows := g6feedRows(t, f, secret); len(rows) != 0 {
		t.Errorf("a private collection wrote feed rows %v", rows)
	}
}

func TestV1CollectionPickerNotes(t *testing.T) {
	f := newG6Fix(t)
	f.user.mu.Lock()
	f.user.items[g6FolderAliceDef][0].Note = "keep"
	f.user.mu.Unlock()
	resp, body := f.call(t, http.MethodGet, "/api/v1/me/works/"+idStr(g6WorkLive)+"/collections",
		"/me/works/{work_id}/collections", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("picker %d %+v", resp.StatusCode, body)
	}
	for _, raw := range body["items"].([]any) {
		c := raw.(map[string]any)
		want := any(nil)
		if strID(c["id"]) == idStr(g6FolderAliceDef) {
			want = "keep"
		}
		if c["work_note"] != want {
			t.Errorf("collection %s work_note = %v, want %v", strID(c["id"]), c["work_note"], want)
		}
	}
}

func (u *g6User) FolderTags(context.Context, int, bool) ([]catalogclient.FolderTagCount, error) {
	return []catalogclient.FolderTagCount{{Tag: "纯爱", FolderCount: 2}}, nil
}

func (u *g6User) CopyFolder(_ context.Context, token string, folderID int64, name, _ string) (*catalogclient.Folder, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	src, ok := u.folders[folderID]
	if !ok || (src.Visibility == "private" && src.OwnerUID != u.uid(token)) {
		return nil, catalogclient.ErrNotFound
	}
	id := u.nextFolder
	u.nextFolder++
	if name == "" {
		name = src.Name
	}
	cp := catalogclient.Folder{
		ID: id, OwnerUID: u.uid(token), Name: name, Description: "源自「" + src.Name + "」",
		Visibility: "private", ItemCount: src.ItemCount, Tags: src.Tags, CopiedFromID: folderID,
		CreatedAt: "2026-09-20T00:00:00Z", UpdatedAt: "2026-09-20T00:00:00Z",
	}
	u.folders[id] = cp
	u.items[id] = append([]catalogclient.FolderItem(nil), u.items[folderID]...)
	return &cp, nil
}

func (u *g6User) FollowedUpdates(_ context.Context, token string) (int, error) {
	if err := u.gate(token); err != nil {
		return 0, err
	}
	return 2, nil
}

func (u *g6User) MarkFollowedSeen(_ context.Context, token string, folderID int64) error {
	if err := u.gate(token); err != nil {
		return err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if !u.folderFollows[u.uid(token)][folderID] {
		return catalogclient.ErrNotFound
	}
	return nil
}

func (u *g6User) FolderProgress(_ context.Context, token string, ids []int64) ([]catalogclient.FolderProgress, error) {
	if err := u.gate(token); err != nil {
		return nil, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	var out []catalogclient.FolderProgress
	for _, id := range ids {
		if f, ok := u.folders[id]; ok && (f.Visibility != "private" || f.OwnerUID == u.uid(token)) {
			out = append(out, catalogclient.FolderProgress{FolderID: id, ItemCount: f.ItemCount, Done: 1})
		}
	}
	return out, nil
}

func (f *fakeCatalogUser) FolderTags(context.Context, int, bool) ([]catalogclient.FolderTagCount, error) {
	return nil, nil
}
func (f *fakeCatalogUser) CopyFolder(context.Context, string, int64, string, string) (*catalogclient.Folder, error) {
	return nil, catalogclient.ErrNotFound
}
func (f *fakeCatalogUser) FollowedUpdates(context.Context, string) (int, error) { return 0, nil }
func (f *fakeCatalogUser) MarkFollowedSeen(context.Context, string, int64) error {
	return catalogclient.ErrNotFound
}
func (f *fakeCatalogUser) FolderProgress(context.Context, string, []int64) ([]catalogclient.FolderProgress, error) {
	return nil, nil
}

func TestV1CollectionRoundThree(t *testing.T) {
	f := newG6Fix(t)
	g6seedList(f)
	f.user.mu.Lock()
	pub := f.user.folders[g6FolderAlicePub]
	pub.Visibility, pub.Tags, pub.CoverWorkID = "unlisted", []string{"纯爱"}, g6WorkOwner
	f.user.folders[g6FolderAlicePub] = pub
	f.user.mu.Unlock()

	resp, body := f.call(t, http.MethodGet, g6col(g6FolderAlicePub), "/collections/{collection_id}", "", "", nil)
	if resp.StatusCode != http.StatusOK || body["visibility"] != "unlisted" {
		t.Fatalf("unlisted read %d %+v", resp.StatusCode, body)
	}
	if tags := body["tag_names"].([]any); len(tags) != 1 || tags[0] != "纯爱" {
		t.Errorf("tag_names %+v", body["tag_names"])
	}
	if cw, _ := body["cover_work"].(map[string]any); cw == nil || strID(cw["id"]) != idStr(g6WorkOwner) {
		t.Errorf("cover_work %+v", body["cover_work"])
	}

	resp, body = f.call(t, http.MethodPost, "/api/v1/collection-copies", "/collection-copies", "sess-bob", keyUUID(71),
		map[string]any{"collection_id": idStr(g6FolderAlicePub)})
	if resp.StatusCode != http.StatusCreated || body["visibility"] != "private" || body["owner"].(map[string]any)["id"] != idStr(w3UserBob) {
		t.Fatalf("copy %d %+v", resp.StatusCode, body)
	}
	if strID(body["copied_from_collection_id"]) != idStr(g6FolderAlicePub) {
		t.Errorf("copy copied_from_collection_id %+v", body["copied_from_collection_id"])
	}
	copyID := strID(body["id"])
	resp, body = f.call(t, http.MethodGet, "/api/v1/collections/"+copyID, "/collections/{collection_id}", "sess-bob", "", nil)
	if src, _ := body["copied_from"].(map[string]any); resp.StatusCode != http.StatusOK || src == nil ||
		strID(src["id"]) != idStr(g6FolderAlicePub) || src["title"] != pub.Name {
		t.Errorf("copy detail copied_from %d %+v", resp.StatusCode, body["copied_from"])
	}
	resp, body = f.call(t, http.MethodGet, "/api/v1/me/collections?limit=100", "/me/collections", "sess-bob", "", nil)
	found := false
	for _, it := range body["items"].([]any) {
		m := it.(map[string]any)
		found = found || (strID(m["id"]) == copyID && strID(m["copied_from_collection_id"]) == idStr(g6FolderAlicePub))
	}
	if resp.StatusCode != http.StatusOK || !found {
		t.Errorf("my collections do not carry the copy source %d %+v", resp.StatusCode, body)
	}
	f.user.mu.Lock()
	gone := f.user.folders[g6FolderAlicePub]
	gone.Visibility = "private"
	f.user.folders[g6FolderAlicePub] = gone
	f.user.mu.Unlock()
	resp, body = f.call(t, http.MethodGet, "/api/v1/collections/"+copyID, "/collections/{collection_id}", "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK || body["copied_from"] != nil || strID(body["copied_from_collection_id"]) != idStr(g6FolderAlicePub) {
		t.Errorf("private source should drop the link only %d %+v", resp.StatusCode, body)
	}
	f.user.mu.Lock()
	gone.Visibility = "unlisted"
	f.user.folders[g6FolderAlicePub] = gone
	f.user.mu.Unlock()
	resp, body = f.call(t, http.MethodPost, "/api/v1/collection-copies", "/collection-copies", "sess-alice", keyUUID(72),
		map[string]any{"collection_id": idStr(g6FolderBobPriv)})
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)

	resp, body = f.call(t, http.MethodGet, "/api/v1/collection-tags", "/collection-tags", "", "", nil)
	if items := body["items"].([]any); resp.StatusCode != http.StatusOK || len(items) != 1 ||
		items[0].(map[string]any)["collection_count"] != float64(2) {
		t.Errorf("tags %d %+v", resp.StatusCode, body)
	}

	resp, body = f.call(t, http.MethodGet, "/api/v1/me/collection-progress?collection_ids="+idStr(g6FolderAlicePub)+","+idStr(g6FolderBobPriv),
		"/me/collection-progress", "sess-alice", "", nil)
	if resp.StatusCode != http.StatusOK || len(body["items"].([]any)) != 1 || len(body["missing"].([]any)) != 1 {
		t.Errorf("progress %d %+v", resp.StatusCode, body)
	}

	resp, body = f.call(t, http.MethodGet, "/api/v1/me/followed-collections/updates", "/me/followed-collections/updates", "sess-bob", "", nil)
	if resp.StatusCode != http.StatusOK || body["updated_count"] != float64(2) {
		t.Errorf("updates %d %+v", resp.StatusCode, body)
	}
	marker := g6col(g6FolderAlicePub) + "/read-marker"
	resp, body = f.call(t, http.MethodPut, marker, "/collections/{collection_id}/read-marker", "sess-bob", "", nil)
	wantCode(t, resp, body, http.StatusNotFound, problem.CodeNotFound)
	f.call(t, http.MethodPut, g6col(g6FolderAlicePub)+"/follow", "/collections/{collection_id}/follow", "sess-bob", "", nil)
	resp, _ = f.call(t, http.MethodPut, marker, "/collections/{collection_id}/read-marker", "sess-bob", "", nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("read-marker after follow %d", resp.StatusCode)
	}
}
