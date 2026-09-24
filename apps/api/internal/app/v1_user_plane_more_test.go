package app

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/catalogclient"
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

func TestV1ListCollectionWorksCountsWhatItShows(t *testing.T) {
	f := newG6Fix(t)
	f.cat.sfwHidden = map[int]bool{g6WorkExtra: true}
	f.user.mu.Lock()
	f.user.items[g6FolderAliceDef] = append(f.user.items[g6FolderAliceDef], catalogclient.FolderItem{
		FolderID: g6FolderAliceDef, WorkID: g6WorkExtra, CreatedAt: "2026-09-03T00:00:00Z", UpdatedAt: "2026-09-03T00:00:00Z",
	})
	folder := f.user.folders[g6FolderAliceDef]
	folder.ItemCount = 2
	folder.UpdatedAt = "2026-09-03T00:00:00Z"
	f.user.folders[g6FolderAliceDef] = folder
	f.user.mu.Unlock()
	for _, tc := range []struct {
		query string
		want  int
	}{{"", 1}, {"?include_nsfw=true", 2}} {
		resp, body := f.call(t, http.MethodGet, g6col(g6FolderAliceDef)+"/works"+tc.query,
			"/collections/{collection_id}/works", "", "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("works%s %d %+v", tc.query, resp.StatusCode, body)
		}
		items, _ := body["items"].([]any)
		if asInt(body["total"]) != tc.want || len(items) != tc.want {
			t.Errorf("works%s: total %v with %d items, want %d of each", tc.query, body["total"], len(items), tc.want)
		}
	}
}

func TestV1ListCollectionWorksCachesPopulation(t *testing.T) {
	f := newG6Fix(t)
	reads := func() int {
		f.cat.mu.Lock()
		defer f.cat.mu.Unlock()
		return len(f.cat.gotLimits)
	}
	view := func() {
		resp, body := f.call(t, http.MethodGet, g6col(g6FolderAliceDef)+"/works",
			"/collections/{collection_id}/works", "", "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("works %d %+v", resp.StatusCode, body)
		}
	}
	before := reads()
	view()
	cold := reads() - before
	view()
	warm := reads() - before - cold
	if cold != 2 || warm != 1 {
		t.Errorf("catalog row reads: cold %d (want population + page = 2), warm %d (want the page only = 1)", cold, warm)
	}
}

func TestV1ListCollectionWorksBuildsPopulationOnce(t *testing.T) {
	f := newG6Fix(t)
	gate := make(chan struct{})
	f.cat.mu.Lock()
	f.cat.rowsGate, f.cat.rowsIn = gate, make(chan struct{}, 1)
	before := len(f.cat.gotLimits)
	f.cat.mu.Unlock()
	view := func(done chan<- int) {
		resp, _ := f.call(t, http.MethodGet, g6col(g6FolderAliceDef)+"/works",
			"/collections/{collection_id}/works", "", "", nil)
		done <- resp.StatusCode
	}
	done := make(chan int, 2)
	go view(done)
	<-f.cat.rowsIn
	go view(done)
	time.Sleep(300 * time.Millisecond)
	close(gate)
	for range 2 {
		if code := <-done; code != http.StatusOK {
			t.Fatalf("status %d", code)
		}
	}
	f.cat.mu.Lock()
	reads := len(f.cat.gotLimits) - before
	f.cat.mu.Unlock()
	if reads != 3 {
		t.Errorf("catalog row reads %d, want 3: one shared population build and one page read per view", reads)
	}
}

func TestV1CollectionPreviewKeepsNSFWFromSFWReaders(t *testing.T) {
	f := newG6Fix(t)
	f.user.mu.Lock()
	f.user.items[g6FolderAlicePub] = []catalogclient.FolderItem{
		{FolderID: g6FolderAlicePub, WorkID: g6WorkNSFW, CreatedAt: "2026-09-04T00:00:00Z", UpdatedAt: "2026-09-04T00:00:00Z"},
		{FolderID: g6FolderAlicePub, WorkID: g6WorkLive, CreatedAt: "2026-09-03T00:00:00Z", UpdatedAt: "2026-09-03T00:00:00Z"},
	}
	folder := f.user.folders[g6FolderAlicePub]
	folder.ItemCount = 2
	f.user.folders[g6FolderAlicePub] = folder
	f.user.mu.Unlock()
	covers := func(query string) []string {
		resp, body := f.call(t, http.MethodGet, g6col(g6FolderAlicePub)+query, "/collections/{collection_id}", "", "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("get%s %d %+v", query, resp.StatusCode, body)
		}
		var out []string
		list, _ := body["preview_covers"].([]any)
		for _, c := range list {
			out = append(out, c.(map[string]any)["url"].(string))
		}
		return out
	}

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	sfw := covers("")
	slog.SetDefault(prev)
	all := covers("?include_nsfw=true")

	if len(all) != len(sfw)+1 {
		t.Fatalf("preview covers: sfw %v, include_nsfw %v; want the NSFW work's cover only in the second", sfw, all)
	}
	for _, u := range sfw {
		if u == all[0] {
			t.Errorf("an SFW reader got the NSFW work's cover %s", u)
		}
	}
	if strings.Contains(buf.String(), "did not render work") {
		t.Errorf("SFW preview logged a drop WARN: %s", buf.String())
	}
}

func TestV1CollectionPreviewFallsBackToCover(t *testing.T) {
	f := newG6Fix(t)
	f.cat.mu.Lock()
	f.cat.rows[g6WorkExtra].CoverSlots.Banner = nil
	bare := f.cat.rows[g6WorkOwner]
	bare.CoverSlots = nil
	f.cat.rows[g6WorkOwner] = bare
	f.cat.mu.Unlock()
	f.user.mu.Lock()
	f.user.items[g6FolderAlicePub] = []catalogclient.FolderItem{
		{FolderID: g6FolderAlicePub, WorkID: g6WorkExtra, CreatedAt: "2026-09-02T00:00:00Z", UpdatedAt: "2026-09-02T00:00:00Z"},
		{FolderID: g6FolderAlicePub, WorkID: g6WorkOwner, CreatedAt: "2026-09-03T00:00:00Z", UpdatedAt: "2026-09-03T00:00:00Z"},
		{FolderID: g6FolderAlicePub, WorkID: g6WorkLive, CreatedAt: "2026-09-04T00:00:00Z", UpdatedAt: "2026-09-04T00:00:00Z"},
	}
	folder := f.user.folders[g6FolderAlicePub]
	folder.ItemCount = 3
	f.user.folders[g6FolderAlicePub] = folder
	f.user.mu.Unlock()

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	resp, body := f.call(t, http.MethodGet, g6col(g6FolderAlicePub), "/collections/{collection_id}", "", "", nil)
	slog.SetDefault(prev)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get %d %+v", resp.StatusCode, body)
	}
	var got []string
	list, _ := body["preview_covers"].([]any)
	for _, c := range list {
		got = append(got, c.(map[string]any)["hash"].(string))
	}
	want := []string{geHash(g6WorkExtra), geHash(g6WorkLive + 1000)}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("preview covers %v, want %v: the portrait for the bannerless work, the banner for the other, nothing for the bare one", got, want)
	}
	if buf.Len() > 0 {
		t.Errorf("preview logged for works that simply lack art: %s", buf.String())
	}
}

func TestV1CollectionPreviewKeepsExplicitArtFromSFWReaders(t *testing.T) {
	const allExplicit = 947000006
	f := newG6Fix(t)
	var row client.CatalogWorkListItem
	decodeInto(t, geRowJSON(geWork{id: allExplicit, name: "ExplicitG6", rating: "all_ages"}), &row)
	explicit := 2
	row.ContentLimit = "nsfw"
	row.CoverSlots.Banner = nil
	row.CoverSlots.Portrait.Sexual = &explicit
	f.cat.mu.Lock()
	f.cat.rows[allExplicit] = row
	f.cat.mu.Unlock()
	f.user.mu.Lock()
	f.user.items[g6FolderAlicePub] = []catalogclient.FolderItem{
		{FolderID: g6FolderAlicePub, WorkID: allExplicit, CreatedAt: "2026-09-02T00:00:00Z", UpdatedAt: "2026-09-02T00:00:00Z"},
		{FolderID: g6FolderAlicePub, WorkID: g6WorkLive, CreatedAt: "2026-09-03T00:00:00Z", UpdatedAt: "2026-09-03T00:00:00Z"},
	}
	folder := f.user.folders[g6FolderAlicePub]
	folder.ItemCount = 2
	f.user.folders[g6FolderAlicePub] = folder
	f.user.mu.Unlock()
	hashes := func(query string) string {
		resp, body := f.call(t, http.MethodGet, g6col(g6FolderAlicePub)+query, "/collections/{collection_id}", "", "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("get%s %d %+v", query, resp.StatusCode, body)
		}
		var out []string
		list, _ := body["preview_covers"].([]any)
		for _, c := range list {
			out = append(out, c.(map[string]any)["hash"].(string))
		}
		return strings.Join(out, ",")
	}

	if got, want := hashes(""), geHash(g6WorkLive+1000); got != want {
		t.Errorf("sfw preview %s, want only %s: an unclaimed work's explicit portrait reached an SFW reader", got, want)
	}
	if got, want := hashes("?include_nsfw=true"), geHash(allExplicit)+","+geHash(g6WorkLive+1000); got != want {
		t.Errorf("include_nsfw preview %s, want %s", got, want)
	}
}
