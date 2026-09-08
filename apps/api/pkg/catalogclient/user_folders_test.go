package catalogclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

type folderFace struct {
	mu   sync.Mutex
	hits []recordedCall
	// pages[path] is served in order, one per request to that path.
	pages map[string][]string
}

type recordedCall struct {
	Method string
	Path   string
	Query  string
	Auth   string
}

func (f *folderFace) server(t *testing.T) *httptest.Server {
	t.Helper()
	seen := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.hits = append(f.hits, recordedCall{
			Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery,
			Auth: r.Header.Get("Authorization"),
		})
		n := seen[r.URL.Path]
		seen[r.URL.Path]++
		f.mu.Unlock()

		bodies := f.pages[r.URL.Path]
		if len(bodies) == 0 {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":"NOT_FOUND","status":404,"detail":"no"}`))
			return
		}
		if n >= len(bodies) {
			n = len(bodies) - 1
		}
		_, _ = w.Write([]byte(bodies[n]))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func (f *folderFace) calls() []recordedCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]recordedCall, len(f.hits))
	copy(out, f.hits)
	return out
}

func folderPage(next string, ids ...int64) string {
	items := make([]string, 0, len(ids))
	for _, id := range ids {
		items = append(items, `{"object":"folder","id":"`+strconv.FormatInt(id, 10)+
			`","owner_uid":"7","name":"f","description":"","visibility":"public",`+
			`"is_default":false,"item_count":1,"created_at":"2026-01-01T00:00:00Z",`+
			`"updated_at":"2026-01-02T00:00:00Z"}`)
	}
	cursor := "null"
	if next != "" {
		cursor = `"` + next + `"`
	}
	return `{"items":[` + strings.Join(items, ",") + `],"next_cursor":` + cursor + `}`
}

func itemPage(next string, workIDs ...int64) string {
	items := make([]string, 0, len(workIDs))
	for _, id := range workIDs {
		items = append(items, `{"object":"folder_item","folder_id":"5","work_id":"`+
			strconv.FormatInt(id, 10)+`","created_at":"2026-01-01T00:00:00Z",`+
			`"updated_at":"2026-01-01T00:00:00Z"}`)
	}
	cursor := "null"
	if next != "" {
		cursor = `"` + next + `"`
	}
	return `{"items":[` + strings.Join(items, ",") + `],"next_cursor":` + cursor + `}`
}

// A public folder is public to anonymous readers, so it is read with the
// application key. Sending the user token instead would make the face demand
// catalog:read of a person's grant, which is the shape of the cover-vote
// outage: a scope missing from the authorize request, failing silently.
func TestPublicFolderReadsUseTheApplicationKey(t *testing.T) {
	face := &folderFace{pages: map[string][]string{
		"/v2/folders":         {folderPage("", 11, 12)},
		"/v2/folders/5":       {`{"object":"folder","id":"5","owner_uid":"7","name":"f","description":"","visibility":"public","is_default":false,"item_count":0,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`},
		"/v2/folders/5/items": {itemPage("", 900)},
	}}
	srv := face.server(t)
	c := New(Config{BaseURL: srv.URL, AppKey: "nmk_live_key"})
	ctx := context.Background()

	folders, err := c.PublicFolders(ctx, 7)
	if err != nil {
		t.Fatalf("PublicFolders: %v", err)
	}
	if len(folders) != 2 || folders[0].ID != 11 || folders[0].OwnerUID != 7 {
		t.Fatalf("decoded %+v", folders)
	}
	if _, err = c.PublicFolder(ctx, 5); err != nil {
		t.Fatalf("PublicFolder: %v", err)
	}
	if _, err = c.PublicFolderItems(ctx, 5); err != nil {
		t.Fatalf("PublicFolderItems: %v", err)
	}

	for _, call := range face.calls() {
		if !strings.Contains(call.Auth, "nmk_live_key") {
			t.Errorf("%s %s sent %q, want the application key", call.Method, call.Path, call.Auth)
		}
	}
	if q := face.calls()[0].Query; !strings.Contains(q, "owner_uid=7") {
		t.Errorf("list query %q carries no owner_uid", q)
	}
}

// The owner's own folders, private ones included, are only reachable with the
// person's access token.
func TestMyFolderReadsUseTheUserToken(t *testing.T) {
	face := &folderFace{pages: map[string][]string{
		"/v2/me/folders":         {folderPage("", 11)},
		"/v2/me/folders/5/items": {itemPage("", 900)},
	}}
	srv := face.server(t)
	c := New(Config{BaseURL: srv.URL, AppKey: "nmk_live_key"})
	ctx := context.Background()

	if _, err := c.MyFolders(ctx, "user-jwt"); err != nil {
		t.Fatalf("MyFolders: %v", err)
	}
	if _, err := c.MyFolderItems(ctx, "user-jwt", 5); err != nil {
		t.Fatalf("MyFolderItems: %v", err)
	}
	for _, call := range face.calls() {
		if call.Auth != "Bearer user-jwt" {
			t.Errorf("%s %s sent %q, want the user token", call.Method, call.Path, call.Auth)
		}
	}
}

func TestMyFoldersContainingSendsTheFilter(t *testing.T) {
	face := &folderFace{pages: map[string][]string{
		"/v2/me/folders": {folderPage("", 11)},
	}}
	srv := face.server(t)
	c := New(Config{BaseURL: srv.URL, AppKey: "k"})

	got, err := c.MyFoldersContaining(context.Background(), "user-jwt", 4242)
	if err != nil {
		t.Fatalf("MyFoldersContaining: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d folders", len(got))
	}
	if q := face.calls()[0].Query; !strings.Contains(q, "contains_work_id=4242") {
		t.Fatalf("query %q carries no contains_work_id — the picker would read every folder instead", q)
	}
}

// The whole collection is read, not the first page: the forum orders and
// paginates locally because catalog's keyset is a sync watermark, and a walk
// that stopped early would silently drop the tail of a big folder.
func TestFolderWalksEveryPage(t *testing.T) {
	face := &folderFace{pages: map[string][]string{
		"/v2/me/folders/5/items": {
			itemPage("cur_a", 901, 902),
			itemPage("cur_b", 903),
			itemPage("", 904),
		},
	}}
	srv := face.server(t)
	c := New(Config{BaseURL: srv.URL, AppKey: "k"})

	items, err := c.MyFolderItems(context.Background(), "user-jwt", 5)
	if err != nil {
		t.Fatalf("MyFolderItems: %v", err)
	}
	if len(items) != 4 {
		t.Fatalf("walked %d items, want 4 across three pages", len(items))
	}
	calls := face.calls()
	if len(calls) != 3 {
		t.Fatalf("made %d requests, want 3", len(calls))
	}
	if strings.Contains(calls[0].Query, "cursor=") {
		t.Errorf("first page sent a cursor: %q", calls[0].Query)
	}
	if !strings.Contains(calls[1].Query, "cursor=cur_a") || !strings.Contains(calls[2].Query, "cursor=cur_b") {
		t.Errorf("cursors not threaded: %q then %q", calls[1].Query, calls[2].Query)
	}
}

// The preview reads one page and stops. Walking is what the detail page does;
// a cover mosaic that walked a 3,303-item folder would make a profile page
// thirty-four requests deep.
func TestFolderPreviewReadsOnePage(t *testing.T) {
	face := &folderFace{pages: map[string][]string{
		"/v2/me/folders/5/items": {itemPage("cur_a", 901, 902, 903, 904), itemPage("", 905)},
	}}
	srv := face.server(t)
	c := New(Config{BaseURL: srv.URL, AppKey: "k"})

	items, err := c.FolderPreviewItems(context.Background(), "user-jwt", 5, 4)
	if err != nil {
		t.Fatalf("FolderPreviewItems: %v", err)
	}
	if len(items) != 4 {
		t.Fatalf("got %d items, want the 4 of one page", len(items))
	}
	calls := face.calls()
	if len(calls) != 1 {
		t.Fatalf("made %d requests, want 1", len(calls))
	}
	if !strings.Contains(calls[0].Query, "limit=4") {
		t.Errorf("query %q does not bound the page", calls[0].Query)
	}
}

func TestFolderPreviewFallsBackToTheAppKeyWithoutAToken(t *testing.T) {
	face := &folderFace{pages: map[string][]string{
		"/v2/folders/5/items": {itemPage("", 901)},
	}}
	srv := face.server(t)
	c := New(Config{BaseURL: srv.URL, AppKey: "nmk_live_key"})

	if _, err := c.FolderPreviewItems(context.Background(), "", 5, 4); err != nil {
		t.Fatalf("FolderPreviewItems: %v", err)
	}
	call := face.calls()[0]
	if call.Path != "/v2/folders/5/items" {
		t.Fatalf("anonymous preview went to %s, want the public lane", call.Path)
	}
	if !strings.Contains(call.Auth, "nmk_live_key") {
		t.Fatalf("anonymous preview sent %q", call.Auth)
	}
}

// A grant that never asked for folder:write comes back 403 SCOPE_REQUIRED, and
// the caller's own cure is signing in again — a refresh mints from the grant as
// recorded. It has to reach the caller as its own error, not a generic 403.
func TestFolderWriteMapsScopeRefusalToItsOwnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":"SCOPE_REQUIRED","status":403,` +
			`"detail":"this operation requires the folder:write scope."}`))
	}))
	t.Cleanup(srv.Close)
	c := New(Config{BaseURL: srv.URL, AppKey: "k"})

	name := "x"
	_, err := c.CreateFolder(context.Background(), "user-jwt", FolderWrite{Name: &name})
	if err == nil {
		t.Fatal("a scope refusal returned no error")
	}
	if !isScopeError(err) {
		t.Fatalf("got %v, want ErrInsufficientScope", err)
	}
}

func isScopeError(err error) bool {
	for e := err; e != nil; {
		if e == ErrInsufficientScope {
			return true
		}
		u, ok := e.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		e = u.Unwrap()
	}
	return false
}

func TestFolderWritesAddressTheRightRoutes(t *testing.T) {
	body := `{"object":"folder","id":"5","owner_uid":"7","name":"f","description":"","visibility":"private","is_default":false,"item_count":0,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`
	face := &folderFace{pages: map[string][]string{
		"/v2/me/folders":                 {body},
		"/v2/me/folders/5":               {body},
		"/v2/me/folders/5/items/900":     {`{}`},
		"/v2/moderation/folders/5":       {body},
		"/v2/moderation/users/7/folders": {`{"object":"folder_purge","owner_uid":"7","folders_deleted":2,"items_deleted":9}`},
	}}
	srv := face.server(t)
	c := New(Config{BaseURL: srv.URL, AppKey: "k"})
	ctx := context.Background()
	name := "n"

	if _, err := c.CreateFolder(ctx, "tok", FolderWrite{Name: &name}); err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	if _, err := c.PatchFolder(ctx, "tok", 5, FolderWrite{Name: &name}); err != nil {
		t.Fatalf("PatchFolder: %v", err)
	}
	if err := c.PutFolderItem(ctx, "tok", 5, 900); err != nil {
		t.Fatalf("PutFolderItem: %v", err)
	}
	if err := c.DeleteFolderItem(ctx, "tok", 5, 900); err != nil {
		t.Fatalf("DeleteFolderItem: %v", err)
	}
	if _, err := c.ModeratePatchFolder(ctx, "tok", 5, FolderWrite{Name: &name}); err != nil {
		t.Fatalf("ModeratePatchFolder: %v", err)
	}
	receipt, err := c.PurgeUserFolders(ctx, "tok", 7)
	if err != nil {
		t.Fatalf("PurgeUserFolders: %v", err)
	}
	if receipt.FoldersDeleted != 2 || receipt.ItemsDeleted != 9 {
		t.Fatalf("receipt %+v", receipt)
	}

	want := []struct{ method, path string }{
		{"POST", "/v2/me/folders"},
		{"PATCH", "/v2/me/folders/5"},
		{"PUT", "/v2/me/folders/5/items/900"},
		{"DELETE", "/v2/me/folders/5/items/900"},
		{"PATCH", "/v2/moderation/folders/5"},
		{"DELETE", "/v2/moderation/users/7/folders"},
	}
	calls := face.calls()
	if len(calls) != len(want) {
		t.Fatalf("made %d calls, want %d: %+v", len(calls), len(want), calls)
	}
	for i, w := range want {
		if calls[i].Method != w.method || calls[i].Path != w.path {
			t.Errorf("call %d was %s %s, want %s %s", i, calls[i].Method, calls[i].Path, w.method, w.path)
		}
	}
}

func TestFolderCreateSendsOnlyWhatWasSet(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"object":"folder","id":"5","owner_uid":"7","name":"n","description":"","visibility":"private","is_default":false,"item_count":0,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`))
	}))
	t.Cleanup(srv.Close)
	c := New(Config{BaseURL: srv.URL, AppKey: "k"})

	name := "n"
	if _, err := c.CreateFolder(context.Background(), "tok", FolderWrite{Name: &name}); err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	if _, ok := got["visibility"]; ok {
		t.Errorf("an unset field was sent anyway: %+v", got)
	}
	if got["name"] != "n" {
		t.Errorf("name did not travel: %+v", got)
	}
}
