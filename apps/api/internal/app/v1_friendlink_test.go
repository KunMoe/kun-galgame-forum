package app

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"kun-galgame-api/pkg/perm"
)

const linkSeedMin = 932000001

// Ties on sort_order inside the galgame shelf straddle page boundaries at limit
// 2 and 3, so only the id tie-breaker keeps a walk stable.
var linkSeeds = []struct {
	category string
	order    int
	status   string
}{
	{"galgame", 0, "normal"},
	{"others", 0, "normal"},
	{"galgame", 1, "normal"},
	{"official", 0, "normal"},
	{"galgame", 1, "down"},
	{"galgame", 1, "normal"},
	{"official", 1, "normal"},
	{"others", 0, "normal"},
}

func newFriendLinkFix(t *testing.T) *writeFix {
	t.Helper()
	f := newWriteFix(t, nil)
	f.alice(t)
	wipe := func() {
		if err := f.db.Exec(`DELETE FROM friend_link`).Error; err != nil {
			t.Fatal(err)
		}
	}
	wipe()
	t.Cleanup(wipe)
	for i, s := range linkSeeds {
		if err := f.db.Exec(`INSERT INTO friend_link (id, category, name, link, description, banner, banner_image_hash, status, sort_order)
			VALUES (?, ?, ?, ?, 'blurb', '', ?, ?, ?)`,
			linkSeedMin+i, s.category, fmt.Sprintf("site %d", i), fmt.Sprintf("https://site%d.example", i),
			strings.Repeat("ab", 32), s.status, s.order).Error; err != nil {
			t.Fatal(err)
		}
	}
	return f
}

const linkOrderSQL = `SELECT id::text FROM friend_link %s ORDER BY array_position(ARRAY['official','galgame','others']::text[], category), sort_order, id`

func (f *writeFix) walkLinks(t *testing.T, q url.Values) []string {
	t.Helper()
	var got []string
	for range 50 {
		resp, body := f.docCall(t, http.MethodGet, "/api/v1/friend-links?"+q.Encode(), "/friend-links", "", "", nil, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("list %v: %d %+v", q, resp.StatusCode, body)
		}
		got = append(got, adminItemIDs(body)...)
		next, _ := body["next_cursor"].(string)
		if next == "" {
			return got
		}
		q.Set("cursor", next)
	}
	t.Fatal("walk did not end")
	return nil
}

func (f *writeFix) linkCall(t *testing.T, method, id, session string, hdr http.Header, payload any) (*http.Response, map[string]any) {
	t.Helper()
	u, spec := "/api/v1/admin/friend-links", "/admin/friend-links"
	if id != "" {
		u, spec = u+"/"+id, spec+"/{friend_link_id}"
	}
	return f.docCall(t, method, u, spec, session, "", hdr, payload)
}

func (f *writeFix) linkOrder(t *testing.T, session, category string, ids []string) (*http.Response, map[string]any) {
	t.Helper()
	return f.docCall(t, http.MethodPut, "/api/v1/admin/friend-link-order", "/admin/friend-link-order", session, "", nil,
		map[string]any{"friend_link_category": category, "friend_link_ids": ids})
}

func TestV1FriendLinksWalk(t *testing.T) {
	f := newFriendLinkFix(t)
	want := f.sqlIDs(t, fmt.Sprintf(linkOrderSQL, ""))
	for _, limit := range []int{1, 2, 3} {
		if got := f.walkLinks(t, url.Values{"limit": {fmt.Sprint(limit)}}); joinIDs(got) != joinIDs(want) {
			t.Errorf("limit %d walked %v, want %v", limit, got, want)
		}
	}
	_, body := f.docCall(t, http.MethodGet, "/api/v1/friend-links", "/friend-links", "", "", nil, nil)
	items, _ := body["items"].([]any)
	first, _ := items[0].(map[string]any)
	banner, _ := first["banner"].(map[string]any)
	if first["friend_link_category"] != "official" || first["object"] != "friend_link" || banner["hash"] == nil ||
		!strings.HasPrefix(fmt.Sprint(first["url"]), "https://") {
		t.Errorf("first %+v", first)
	}
	for _, k := range []string{"sort_order", "status", "link", "name", "banner_url", "created"} {
		if _, has := first[k]; has {
			t.Errorf("link carries %s", k)
		}
	}
}

func TestV1FriendLinksShelfAndCursor(t *testing.T) {
	f := newFriendLinkFix(t)
	got := f.walkLinks(t, url.Values{"limit": {"2"}, "friend_link_category": {"galgame"}})
	want := f.sqlIDs(t, fmt.Sprintf(linkOrderSQL, "WHERE category = 'galgame'"))
	if joinIDs(got) != joinIDs(want) || len(want) != 4 {
		t.Errorf("galgame shelf %v, want %v", got, want)
	}
	resp, body := f.docCall(t, http.MethodGet, "/api/v1/friend-links?limit=1&friend_link_category=galgame", "/friend-links", "", "", nil, nil)
	cur, _ := body["next_cursor"].(string)
	resp, body = f.docCall(t, http.MethodGet, "/api/v1/friend-links?limit=1&friend_link_category=others&cursor="+cur, "/friend-links", "", "", nil, nil)
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_CURSOR" {
		t.Errorf("cursor reused on another shelf: %d %+v", resp.StatusCode, body)
	}
	resp, body = f.docCall(t, http.MethodGet, "/api/v1/friend-links?friend_link_category=blogs", "/friend-links", "", "", nil, nil)
	if resp.StatusCode != http.StatusBadRequest || body["code"] != "UNKNOWN_ENUM_VALUE" {
		t.Errorf("unknown shelf %d %+v", resp.StatusCode, body)
	}
}

func TestV1FriendLinkUnknownStatusIsNotNormal(t *testing.T) {
	f := newFriendLinkFix(t)
	if err := f.db.Exec(`UPDATE friend_link SET status = 'essential' WHERE id = ?`, linkSeedMin).Error; err != nil {
		t.Fatal(err)
	}
	if resp, body := f.docCall(t, http.MethodGet, "/api/v1/friend-links", "/friend-links", "", "", nil, nil); resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("essential row: %d %+v", resp.StatusCode, body)
	}
}

func TestV1FriendLinkWritesNeedTheirPermissions(t *testing.T) {
	f := newFriendLinkFix(t)
	id := fmt.Sprint(linkSeedMin)
	bearer := http.Header{"Authorization": {"Bearer staff-token"}}
	create := map[string]any{"friend_link_category": "others", "title": "x", "url": "https://x.example"}
	for _, c := range []struct {
		name, method, id, session string
		hdr                       http.Header
		payload                   any
		status                    int
	}{
		{"view anonymously", http.MethodGet, id, "", nil, nil, 401},
		{"view as a user", http.MethodGet, id, "sess-alice", nil, nil, 403},
		{"create as a user", http.MethodPost, "", "sess-alice", nil, create, 403},
		{"create as a Bearer moderator", http.MethodPost, "", "", bearer, create, 403},
		{"patch as a user", http.MethodPatch, id, "sess-alice", nil, map[string]any{"state": "down"}, 403},
		{"delete as a user", http.MethodDelete, id, "sess-alice", nil, nil, 403},
	} {
		if resp, body := f.linkCall(t, c.method, c.id, c.session, c.hdr, c.payload); resp.StatusCode != c.status {
			t.Errorf("%s: %d %+v", c.name, resp.StatusCode, body)
		}
	}
	if resp, body := f.linkOrder(t, "sess-alice", "official", []string{fmt.Sprint(linkSeedMin + 3)}); resp.StatusCode != http.StatusForbidden {
		t.Errorf("reorder as a user: %d %+v", resp.StatusCode, body)
	}
	if n := f.scalar(t, `SELECT COUNT(*) FROM friend_link`); n != len(linkSeeds) {
		t.Fatalf("a refused call changed the links: %d", n)
	}

	perm.SetUserOverrides(map[int][]perm.Override{w3UserStaff: {{Permission: perm.FriendLinkDelete, Effect: perm.EffectRevoke}}})
	t.Cleanup(func() { perm.SetUserOverrides(nil) })
	if resp, body := f.linkCall(t, http.MethodPatch, id, "sess-staff", nil, map[string]any{"description": "edited"}); resp.StatusCode != http.StatusOK {
		t.Errorf("patch without friend_link.delete: %d %+v", resp.StatusCode, body)
	}
	if resp, body := f.linkCall(t, http.MethodDelete, id, "sess-staff", nil, nil); resp.StatusCode != http.StatusForbidden {
		t.Errorf("delete without friend_link.delete: %d %+v", resp.StatusCode, body)
	}
}

func TestV1FriendLinkCreateAndPatch(t *testing.T) {
	f := newFriendLinkFix(t)
	resp, body := f.linkCall(t, http.MethodPost, "", "sess-staff", nil, map[string]any{
		"friend_link_category": "galgame", "title": "  New site ", "url": "https://new.example/", "description": " hi ",
	})
	if resp.StatusCode != http.StatusCreated || body["title"] != "New site" || body["description"] != "hi" ||
		body["state"] != "normal" || body["banner"] != nil {
		t.Fatalf("create %d %+v", resp.StatusCode, body)
	}
	id := strID(body["id"])
	if resp.Header.Get("Location") != "/api/v1/admin/friend-links/"+id {
		t.Errorf("Location %q", resp.Header.Get("Location"))
	}
	shelf := f.walkLinks(t, url.Values{"friend_link_category": {"galgame"}})
	if shelf[len(shelf)-1] != id {
		t.Errorf("a new link goes last on its shelf: %v", shelf)
	}

	for field, value := range map[string]any{"url": "javascript:alert(1)", "friend_link_category": "blogs", "banner_image_hash": "zz"} {
		bad := map[string]any{"friend_link_category": "others", "title": "x", "url": "https://x.example"}
		bad[field] = value
		if resp, body := f.linkCall(t, http.MethodPost, "", "sess-staff", nil, bad); resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("bad %s: %d %+v", field, resp.StatusCode, body)
		}
	}
	resp, body = f.linkCall(t, http.MethodPost, "", "sess-staff", nil, map[string]any{"friend_link_category": "others", "title": "  ", "url": "https://x.example"})
	if e := firstFieldError(body); resp.StatusCode != http.StatusUnprocessableEntity || e["pointer"] != "/title" || e["reason"] != "TOO_SHORT" {
		t.Errorf("blank title %d %+v", resp.StatusCode, body)
	}

	moved := fmt.Sprint(linkSeedMin + 3)
	resp, body = f.linkCall(t, http.MethodPatch, moved, "sess-staff", nil, map[string]any{"friend_link_category": "others", "state": "down"})
	if resp.StatusCode != http.StatusOK || body["friend_link_category"] != "others" || body["state"] != "down" {
		t.Fatalf("move %d %+v", resp.StatusCode, body)
	}
	others := f.walkLinks(t, url.Values{"friend_link_category": {"others"}})
	if others[len(others)-1] != moved {
		t.Errorf("a link that changes shelf goes last on it: %v", others)
	}
	resp, body = f.linkCall(t, http.MethodPatch, moved, "sess-staff", nil, map[string]any{"banner_image_hash": ""})
	if resp.StatusCode != http.StatusOK || body["banner"] != nil {
		t.Errorf("clear banner %d %+v", resp.StatusCode, body)
	}
	if resp, body := f.linkCall(t, http.MethodPatch, moved, "sess-staff", nil, map[string]any{}); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("empty patch %d %+v", resp.StatusCode, body)
	}
	for _, method := range []string{http.MethodPatch, http.MethodDelete, http.MethodGet} {
		var payload any
		if method == http.MethodPatch {
			payload = map[string]any{"state": "down"}
		}
		if resp, body := f.linkCall(t, method, "999999999", "sess-staff", nil, payload); resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s missing: %d %+v", method, resp.StatusCode, body)
		}
	}
	if resp, _ := f.linkCall(t, http.MethodDelete, id, "sess-staff", nil, nil); resp.StatusCode != http.StatusNoContent {
		t.Errorf("delete %d", resp.StatusCode)
	}
	if resp, _ := f.linkCall(t, http.MethodGet, id, "sess-staff", nil, nil); resp.StatusCode != http.StatusNotFound {
		t.Errorf("get after delete %d", resp.StatusCode)
	}
}

func TestV1FriendLinkOrderReplacesTheShelf(t *testing.T) {
	f := newFriendLinkFix(t)
	shelf := f.sqlIDs(t, `SELECT id::text FROM friend_link WHERE category = 'galgame' ORDER BY id DESC`)
	if resp, body := f.linkOrder(t, "sess-staff", "galgame", shelf); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("reorder %d %+v", resp.StatusCode, body)
	}
	if got := f.walkLinks(t, url.Values{"friend_link_category": {"galgame"}}); joinIDs(got) != joinIDs(shelf) {
		t.Errorf("order %v, want %v", got, shelf)
	}
	stamp := func() string { return joinIDs(f.sqlIDs(t, `SELECT id::text FROM friend_link ORDER BY category, sort_order, id`)) }
	before := stamp()

	resp, body := f.linkOrder(t, "sess-staff", "galgame", shelf[1:])
	e := firstFieldError(body)
	params, _ := e["params"].(map[string]any)
	if resp.StatusCode != http.StatusUnprocessableEntity || e["reason"] != "TOO_FEW_ITEMS" || asInt(params["min_items"]) != len(shelf) {
		t.Errorf("partial shelf %d %+v", resp.StatusCode, body)
	}
	other := fmt.Sprint(linkSeedMin + 3)
	resp, body = f.linkOrder(t, "sess-staff", "galgame", append([]string{other}, shelf...))
	if e := firstFieldError(body); resp.StatusCode != http.StatusUnprocessableEntity || e["reason"] != "UNKNOWN_REFERENCE" || e["pointer"] != "/friend_link_ids/0" {
		t.Errorf("link from another shelf %d %+v", resp.StatusCode, body)
	}
	if after := stamp(); after != before {
		t.Errorf("a refused reorder wrote: %s → %s", before, after)
	}
}

func TestV1AppVersion(t *testing.T) {
	f := newWriteFix(t, nil)
	f.Config.AppRelease.MinVersion = "1.2.0"
	f.Config.AppRelease.LatestVersion = "1.3.4"
	f.Config.AppRelease.Downloads.Android = "https://www.kungal.com/app"
	f.Config.AppRelease.Downloads.IOS = "https://www.kungal.com/app"
	f.Config.AppRelease.Downloads.Windows = "https://dl.example/kungal.exe"
	f.Config.AppRelease.Downloads.Linux = "https://www.kungal.com/app"
	resp, body := f.docCall(t, http.MethodGet, "/api/v1/app/version", "/app/version", "", "", nil, nil)
	downloads, _ := body["downloads"].(map[string]any)
	if resp.StatusCode != http.StatusOK || body["object"] != "app_version" || body["min_version"] != "1.2.0" ||
		body["latest_version"] != "1.3.4" || downloads["windows"] != "https://dl.example/kungal.exe" {
		t.Errorf("app version %d %+v", resp.StatusCode, body)
	}
}
