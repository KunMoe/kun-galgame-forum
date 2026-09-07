package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"kun-galgame-api/pkg/catalogclient"
)

func folder(id int64, isDefault bool, updated string) catalogclient.Folder {
	return catalogclient.Folder{ID: id, IsDefault: isDefault, UpdatedAt: updated, ItemCount: 1}
}

// The catalog lists folders id-ascending because that keyset is a sync
// watermark. This face has always shown the default folder first and the rest
// most-recently-touched first, so the order is re-imposed here.
func TestFoldersAreReorderedForThisSite(t *testing.T) {
	got := []catalogclient.Folder{
		folder(1, false, "2026-01-01T00:00:00Z"),
		folder(2, false, "2026-03-01T00:00:00Z"),
		folder(3, true, "2025-01-01T00:00:00Z"),
		folder(4, false, "2026-02-01T00:00:00Z"),
	}
	sortFolders(got)

	ids := []int64{}
	for _, f := range got {
		ids = append(ids, f.ID)
	}
	want := []int64{3, 2, 4, 1}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("order %v, want %v (default first, then newest updated)", ids, want)
		}
	}
}

func TestFolderPagingSlicesLocally(t *testing.T) {
	all := []catalogclient.Folder{folder(1, false, ""), folder(2, false, ""), folder(3, false, ""), folder(4, false, "")}

	page2 := pageOfFolders(all, 2, 2)
	if len(page2) != 2 || page2[0].ID != 3 {
		t.Fatalf("page 2 of 2 was %+v", page2)
	}
	if got := pageOfFolders(all, 3, 2); got != nil {
		t.Fatalf("a page past the end returned %+v, want nothing", got)
	}
	tail := pageOfFolders(all, 2, 3)
	if len(tail) != 1 || tail[0].ID != 4 {
		t.Fatalf("a short last page was %+v", tail)
	}
}

func TestItemPagingSlicesLocally(t *testing.T) {
	items := []catalogclient.FolderItem{{WorkID: 10}, {WorkID: 11}, {WorkID: 12}}
	if got := pageOfWorkIDs(items, 1, 2); len(got) != 2 || got[0] != 10 || got[1] != 11 {
		t.Fatalf("page 1 was %v", got)
	}
	if got := pageOfWorkIDs(items, 2, 2); len(got) != 1 || got[0] != 12 {
		t.Fatalf("page 2 was %v", got)
	}
	if got := pageOfWorkIDs(items, 9, 2); len(got) != 0 {
		t.Fatalf("a page past the end returned %v, want an empty slice not nil", got)
	}
}

// Deleting a folder lowers the local favourite counter only for the works it
// was the last folder to hold. A work the same person also keeps elsewhere is
// still favourited by them, and counting it out would drift the ranking down
// on every folder anyone deletes.
func TestWorksLeavingTheLibraryIgnoresWorksHeldElsewhere(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v2/me/folders":
			_, _ = w.Write([]byte(`{"items":[
				{"object":"folder","id":"5","owner_uid":"7","name":"a","description":"","visibility":"private","is_default":false,"item_count":2,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},
				{"object":"folder","id":"6","owner_uid":"7","name":"b","description":"","visibility":"private","is_default":false,"item_count":1,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}
			],"next_cursor":null}`))
		case strings.HasPrefix(r.URL.Path, "/v2/me/folders/5/items"):
			_, _ = w.Write([]byte(`{"items":[
				{"object":"folder_item","folder_id":"5","work_id":"100","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},
				{"object":"folder_item","folder_id":"5","work_id":"200","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}
			],"next_cursor":null}`))
		case strings.HasPrefix(r.URL.Path, "/v2/me/folders/6/items"):
			_, _ = w.Write([]byte(`{"items":[
				{"object":"folder_item","folder_id":"6","work_id":"200","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}
			],"next_cursor":null}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	s := &CollectionService{catalog: catalogclient.New(catalogclient.Config{BaseURL: srv.URL, AppKey: "k"})}
	mine, err := s.catalog.MyFolders(context.Background(), "tok")
	if err != nil {
		t.Fatalf("MyFolders: %v", err)
	}
	got := s.worksLeavingTheLibrary(context.Background(), "tok", 5, mine)
	sort.Ints(got)
	if len(got) != 1 || got[0] != 100 {
		t.Fatalf("got %v, want only work 100 — 200 is still in folder 6", got)
	}
}

// An unreadable sibling folder must not be read as "nothing else holds these":
// answering the difference from a partial view would decrement counters for
// works the person still keeps.
func TestWorksLeavingTheLibraryYieldsNothingWhenAFolderCannotBeRead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v2/me/folders":
			_, _ = w.Write([]byte(`{"items":[
				{"object":"folder","id":"5","owner_uid":"7","name":"a","description":"","visibility":"private","is_default":false,"item_count":1,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},
				{"object":"folder","id":"6","owner_uid":"7","name":"b","description":"","visibility":"private","is_default":false,"item_count":1,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}
			],"next_cursor":null}`))
		case strings.HasPrefix(r.URL.Path, "/v2/me/folders/5/items"):
			_, _ = w.Write([]byte(`{"items":[
				{"object":"folder_item","folder_id":"5","work_id":"100","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}
			],"next_cursor":null}`))
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)

	s := &CollectionService{catalog: catalogclient.New(catalogclient.Config{BaseURL: srv.URL, AppKey: "k"})}
	mine, err := s.catalog.MyFolders(context.Background(), "tok")
	if err != nil {
		t.Fatalf("MyFolders: %v", err)
	}
	if got := s.worksLeavingTheLibrary(context.Background(), "tok", 5, mine); len(got) != 0 {
		t.Fatalf("got %v from a partial read, want nothing", got)
	}
}
