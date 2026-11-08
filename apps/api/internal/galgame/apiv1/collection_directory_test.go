package apiv1

import (
	"encoding/json"
	"reflect"
	"testing"

	"kun-galgame-api/pkg/catalogclient"
)

func TestDirectoryCacheRoundTripKeepsEveryField(t *testing.T) {
	in := &catalogclient.FolderPage{NextCursor: "cur_9", Folders: []catalogclient.Folder{{
		ID: 7, OwnerUID: 3, Name: "n", Description: "d", Visibility: "public", IsDefault: true,
		ItemCount: 4, FollowerCount: 2, ContentRating: "r18", RecentWorkIDs: []int64{11, 12},
		CoverWorkID: 11, CopiedFromID: 5, Tags: []string{"纯爱"},
		CreatedAt: "2026-11-01T00:00:00Z", UpdatedAt: "2026-11-02T00:00:00Z",
	}}}
	raw, err := json.Marshal(cacheDirectoryPage(in))
	if err != nil {
		t.Fatal(err)
	}
	var cached cachedDirectoryPage
	if err := json.Unmarshal(raw, &cached); err != nil {
		t.Fatal(err)
	}
	if got := cached.page(); !reflect.DeepEqual(got, in) {
		t.Fatalf("round trip\n got %+v\nwant %+v", got, in)
	}
}

func TestDirectoryCacheKeySeparatesQueries(t *testing.T) {
	a := directoryCacheKey(catalogclient.DirectoryQuery{Sort: "followers_desc", ContainsWorkID: 1, Limit: 8})
	b := directoryCacheKey(catalogclient.DirectoryQuery{Sort: "followers_desc", ContainsWorkID: 2, Limit: 8})
	c := directoryCacheKey(catalogclient.DirectoryQuery{Sort: "followers_desc", ContainsWorkID: 1, Limit: 8, IncludeNSFW: true})
	if a == b || a == c {
		t.Fatalf("keys collide: %s %s %s", a, b, c)
	}
}
