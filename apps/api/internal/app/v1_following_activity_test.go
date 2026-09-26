package app

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/imageclient"
)

const (
	faPath      = "/me/following-activities"
	faGroupPath = "/activity-groups/{group_id}/items"
	faCoverHash = "abababababababababababababababababababababababababababababababab"
)

type followingReq struct {
	Method, Path, RawQuery, Body string
}

type followingUpstream struct {
	mu         sync.Mutex
	reqs       []followingReq
	down       atomic.Bool
	groups     []communityclient.ActivityGroupView
	next       string
	items      map[int64][]communityclient.ActivityItemView
	unseen     int
	seenAt     *string
	seenReply  string
	badCursors map[string]bool
}

func (u *followingUpstream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if u.down.Load() {
		writeEnvelope(w, http.StatusBadGateway, 50000, "upstream down", nil)
		return
	}
	body, _ := io.ReadAll(r.Body)
	u.mu.Lock()
	u.reqs = append(u.reqs, followingReq{Method: r.Method, Path: r.URL.Path, RawQuery: r.URL.RawQuery, Body: string(body)})
	groups := append([]communityclient.ActivityGroupView(nil), u.groups...)
	next := u.next
	items := u.items
	unseen, seenAt, seenReply := u.unseen, u.seenAt, u.seenReply
	bad := u.badCursors[r.URL.Query().Get("cursor")]
	u.mu.Unlock()

	path := r.URL.Path
	switch {
	case r.Method == http.MethodGet && strings.HasSuffix(path, "/following/activities") && !strings.HasSuffix(path, "/unseen"):
		if bad {
			writeEnvelope(w, http.StatusBadRequest, 40000, "bad cursor", nil)
			return
		}
		writeEnvelope(w, 200, 0, "", communityclient.ActivityGroupListResponse{Groups: groups, NextCursor: next})
	case r.Method == http.MethodGet && strings.HasSuffix(path, "/following/activities/unseen"):
		writeEnvelope(w, 200, 0, "", communityclient.ActivityUnseenResponse{UnseenCount: unseen, SeenAt: seenAt})
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/following/activities/seen"):
		writeEnvelope(w, 200, 0, "", communityclient.ActivitySeenResponse{SeenAt: seenReply})
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/activity-groups/") && strings.HasSuffix(path, "/items"):
		if bad {
			writeEnvelope(w, http.StatusBadRequest, 40000, "bad cursor", nil)
			return
		}
		idStr := strings.TrimSuffix(strings.TrimPrefix(path, "/activity-groups/"), "/items")
		id, _ := strconv.ParseInt(idStr, 10, 64)
		page, ok := items[id]
		if !ok {
			writeEnvelope(w, http.StatusNotFound, 40400, "no such group", nil)
			return
		}
		writeEnvelope(w, 200, 0, "", communityclient.ActivityItemListResponse{Items: page, NextCursor: next})
	default:
		writeEnvelope(w, http.StatusNotFound, 40400, "no route "+path, nil)
	}
}

func (u *followingUpstream) last() followingReq {
	u.mu.Lock()
	defer u.mu.Unlock()
	if len(u.reqs) == 0 {
		return followingReq{}
	}
	return u.reqs[len(u.reqs)-1]
}

func (u *followingUpstream) query() url.Values {
	q, _ := url.ParseQuery(u.last().RawQuery)
	return q
}

type followingFix struct {
	*writeFix
	cm *followingUpstream
}

func newFollowingFix(t *testing.T) *followingFix {
	t.Helper()
	cm := &followingUpstream{items: map[int64][]communityclient.ActivityItemView{}, seenReply: "2026-09-26T00:00:00Z"}
	base := newWriteFixCommunity(t, nil, cm)
	base.alice(t)
	return &followingFix{writeFix: base, cm: cm}
}

func (f *followingFix) list(t *testing.T, session string, q url.Values) (*http.Response, map[string]any) {
	t.Helper()
	return f.callJSON(t, http.MethodGet, "/api/v1"+faPath+"?"+q.Encode(), faPath, session, "", nil)
}

func (f *followingFix) summary(t *testing.T, session string, q url.Values) (*http.Response, map[string]any) {
	t.Helper()
	return f.callJSON(t, http.MethodGet, "/api/v1"+faPath+"/summary?"+q.Encode(), faPath+"/summary", session, "", nil)
}

func (f *followingFix) mark(t *testing.T, session string, payload map[string]any) (*http.Response, map[string]any) {
	t.Helper()
	return f.callJSON(t, http.MethodPut, "/api/v1"+faPath+"/read-marker", faPath+"/read-marker", session, "", payload)
}

func (f *followingFix) items(t *testing.T, session, groupID string, q url.Values) (*http.Response, map[string]any) {
	t.Helper()
	return f.callJSON(t, http.MethodGet, "/api/v1/activity-groups/"+groupID+"/items?"+q.Encode(), faGroupPath, session, "", nil)
}

func faItem(id, actor int64, site, verb, title, rawURL, limit string, workID *int64, hash string) communityclient.ActivityItemView {
	return communityclient.ActivityItemView{
		ID: id, Site: site, Key: "key-" + strconv.FormatInt(id, 10), ActorID: actor, Verb: verb,
		ObjectKind: "topic", ObjectLabel: "Topic", Title: title, Excerpt: "lede", URL: rawURL,
		CoverImageHash: hash, WorkID: workID, ContentLimit: limit, OccurredAt: "2026-09-25T12:00:00Z",
	}
}

func faGroup(id, actor int64, site, verb, day string, items ...communityclient.ActivityItemView) communityclient.ActivityGroupView {
	return communityclient.ActivityGroupView{
		ID: id, Site: site, ActorID: actor, Verb: verb, ObjectKind: "topic", ObjectLabel: "Topic",
		Day: day, ItemCount: max(len(items), 1), LatestAt: "2026-09-25T12:00:00Z", Items: items,
	}
}

func TestV1FollowingActivitiesMapsGroupAndItems(t *testing.T) {
	f := newFollowingFix(t)
	work := int64(77)
	kungal := faItem(101, int64(w3UserOther), "kungal", "publish", "Hello",
		"https://www.kungal.com/topic/42?reply=3", "sfw", &work, faCoverHash)
	moyu := faItem(202, int64(w3UserGrant), "moyu", "like", "Patch",
		"https://www.moyu.moe/patch/9", "all", nil, "")
	f.cm.groups = []communityclient.ActivityGroupView{
		faGroup(11, int64(w3UserOther), "kungal", "publish", "2026-09-25", kungal),
		faGroup(12, int64(w3UserGrant), "moyu", "like", "2026-09-26", moyu),
	}

	resp, body := f.list(t, "sess-bob", url.Values{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	items := listItems(t, body)
	if len(items) != 2 {
		t.Fatalf("groups %d, want 2: %+v", len(items), items)
	}
	g0 := items[0]
	if g0["object"] != "following_activity_group" || fmt.Sprint(g0["id"]) != "11" || g0["site"] != "kungal" ||
		g0["verb"] != "publish" || g0["object_kind"] != "topic" || g0["object_label"] != "Topic" ||
		g0["calendar_date"] != "2026-09-25" || asInt(g0["item_count"]) != 1 || g0["latest_at"] != "2026-09-25T12:00:00Z" {
		t.Fatalf("kungal group %+v", g0)
	}
	perf, _ := g0["actor"].(map[string]any)
	if fmt.Sprint(perf["id"]) != strconv.Itoa(w3UserOther) || perf["name"] != "other" {
		t.Fatalf("actor %+v", perf)
	}
	row, _ := g0["items"].([]any)
	if len(row) != 1 {
		t.Fatalf("items %+v", row)
	}
	it, _ := row[0].(map[string]any)
	if it["object"] != "following_activity_item" || fmt.Sprint(it["id"]) != "101" || it["title"] != "Hello" ||
		it["excerpt"] != "lede" || it["url"] != "https://www.kungal.com/topic/42?reply=3" ||
		it["in_site_path"] != "/topic/42?reply=3" || fmt.Sprint(it["related_work_id"]) != "77" ||
		it["is_nsfw"] != false || it["occurred_at"] != "2026-09-25T12:00:00Z" {
		t.Fatalf("kungal item %+v", it)
	}
	cover, _ := it["cover"].(map[string]any)
	wantURL := imageclient.MainURL("https://image.test.example", faCoverHash, "webp")
	if cover["hash"] != faCoverHash || cover["url"] != wantURL {
		t.Fatalf("cover %+v, want hash %s url %s", cover, faCoverHash, wantURL)
	}
	g1 := items[1]
	if g1["site"] != "moyu" {
		t.Fatalf("moyu group %+v", g1)
	}
	row, _ = g1["items"].([]any)
	it, _ = row[0].(map[string]any)
	if it["in_site_path"] != nil || it["is_nsfw"] != true || it["related_work_id"] != nil || it["cover"] != nil {
		t.Fatalf("moyu item %+v", it)
	}
}

func TestV1FollowingActivitiesFiltersReachCommunity(t *testing.T) {
	f := newFollowingFix(t)
	resp, body := f.list(t, "sess-bob", url.Values{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("sfw %d %+v", resp.StatusCode, body)
	}
	if f.cm.query().Get("content_limit") != "sfw" {
		t.Fatalf("default content_limit %q", f.cm.last().RawQuery)
	}
	resp, body = f.list(t, "sess-bob", url.Values{"include_nsfw": {"true"}, "verbs": {"publish,like"}, "sites": {"kungal,moyu"}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("filtered %d %+v", resp.StatusCode, body)
	}
	q := f.cm.query()
	if q.Get("content_limit") != "all" || q.Get("verbs") != "publish,like" || q.Get("sites") != "kungal,moyu" {
		t.Fatalf("filters %q", f.cm.last().RawQuery)
	}
}

func TestV1FollowingActivitiesDropsBannedActorKeepsCursor(t *testing.T) {
	f := newFollowingFix(t)
	f.cm.next = "cm_page2"
	f.cm.groups = []communityclient.ActivityGroupView{
		faGroup(11, int64(w3UserBanned), "kungal", "publish", "2026-09-25",
			faItem(101, int64(w3UserBanned), "kungal", "publish", "x", "https://www.kungal.com/topic/1", "sfw", nil, "")),
		faGroup(12, int64(w3UserOther), "kungal", "reply", "2026-09-25",
			faItem(102, int64(w3UserOther), "kungal", "reply", "y", "https://www.kungal.com/topic/2", "sfw", nil, "")),
	}
	resp, body := f.list(t, "sess-bob", url.Values{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	items := listItems(t, body)
	if len(items) != 1 || fmt.Sprint(items[0]["id"]) != "12" {
		t.Fatalf("items %+v, want only group 12", items)
	}
	if nextCursor(body) == "" {
		t.Fatal("dropped the banned group and also dropped next_cursor")
	}
}

func TestV1FollowingActivitiesCursorRoundTrip(t *testing.T) {
	f := newFollowingFix(t)
	f.cm.next = "cm_page2"
	f.cm.groups = []communityclient.ActivityGroupView{
		faGroup(11, int64(w3UserOther), "kungal", "publish", "2026-09-25",
			faItem(101, int64(w3UserOther), "kungal", "publish", "x", "https://www.kungal.com/topic/1", "sfw", nil, "")),
	}
	resp, body := f.list(t, "sess-bob", url.Values{"limit": {"1"}})
	own := nextCursor(body)
	if resp.StatusCode != http.StatusOK || own == "" {
		t.Fatalf("first page %d %+v", resp.StatusCode, body)
	}
	f.cm.next = ""
	resp, body = f.list(t, "sess-bob", url.Values{"limit": {"1"}, "cursor": {own}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("page 2 %d %+v", resp.StatusCode, body)
	}
	if f.cm.query().Get("cursor") != "cm_page2" {
		t.Fatalf("community cursor %q, want cm_page2", f.cm.last().RawQuery)
	}
	resp, body = f.list(t, "sess-bob", url.Values{"limit": {"1"}, "cursor": {own}, "include_nsfw": {"true"}})
	mustCode(t, resp, body, http.StatusBadRequest, "INVALID_CURSOR")
}

func TestV1FollowingActivitiesCommunityCursor400(t *testing.T) {
	f := newFollowingFix(t)
	fp := collect.Fingerprint("sfw", "", "")
	cur := collect.EncodeCursor("following", fp, "nope")
	f.cm.badCursors = map[string]bool{"nope": true}
	resp, body := f.list(t, "sess-bob", url.Values{"cursor": {cur}})
	mustCode(t, resp, body, http.StatusBadRequest, "INVALID_CURSOR")
}

func TestV1FollowingActivitiesCommunityDown(t *testing.T) {
	f := newFollowingFix(t)
	f.cm.down.Store(true)
	resp, body := f.list(t, "sess-bob", url.Values{})
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
	resp, body = f.summary(t, "sess-bob", url.Values{})
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
	resp, body = f.mark(t, "sess-bob", map[string]any{})
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
	resp, body = f.items(t, "", "11", url.Values{})
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1FollowingActivitiesUnauthenticated(t *testing.T) {
	f := newFollowingFix(t)
	resp, body := f.list(t, "", url.Values{})
	mustCode(t, resp, body, http.StatusUnauthorized, "MISSING_CREDENTIAL")
}

func TestV1ActivityGroupItemsPublic(t *testing.T) {
	f := newFollowingFix(t)
	f.cm.items[11] = []communityclient.ActivityItemView{
		faItem(101, int64(w3UserOther), "kungal", "publish", "Hello", "https://www.kungal.com/topic/42", "sfw", nil, ""),
	}
	resp, body := f.items(t, "", "11", url.Values{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("items %d %+v", resp.StatusCode, body)
	}
	got := listItems(t, body)
	if len(got) != 1 || fmt.Sprint(got[0]["id"]) != "101" || got[0]["in_site_path"] != "/topic/42" {
		t.Fatalf("items %+v", got)
	}
}

func TestV1ActivityGroupItemsUnknown(t *testing.T) {
	f := newFollowingFix(t)
	resp, body := f.items(t, "", "99", url.Values{})
	mustCode(t, resp, body, http.StatusNotFound, "NOT_FOUND")
}

func TestV1ActivityGroupItemsBannedActor(t *testing.T) {
	f := newFollowingFix(t)
	f.cm.items[11] = []communityclient.ActivityItemView{
		faItem(101, int64(w3UserBanned), "kungal", "publish", "x", "https://www.kungal.com/topic/1", "sfw", nil, ""),
	}
	resp, body := f.items(t, "", "11", url.Values{})
	mustCode(t, resp, body, http.StatusNotFound, "NOT_FOUND")
}

func TestV1FollowingActivitySummary(t *testing.T) {
	f := newFollowingFix(t)
	f.cm.unseen = 7
	resp, body := f.summary(t, "sess-bob", url.Values{"include_nsfw": {"true"}, "verbs": {"publish"}, "sites": {"kungal"}})
	if resp.StatusCode != http.StatusOK || asInt(body["unseen_count"]) != 7 || body["last_seen_at"] != nil {
		t.Fatalf("summary %d %+v", resp.StatusCode, body)
	}
	q := f.cm.query()
	if q.Get("content_limit") != "all" || q.Get("verbs") != "publish" || q.Get("sites") != "kungal" {
		t.Fatalf("unseen query %q", f.cm.last().RawQuery)
	}
	at := "2026-09-25T12:00:00Z"
	f.cm.seenAt = &at
	resp, body = f.summary(t, "sess-bob", url.Values{})
	if resp.StatusCode != http.StatusOK || body["last_seen_at"] != at {
		t.Fatalf("seen summary %d %+v", resp.StatusCode, body)
	}
}

func TestV1FollowingActivityReadMarker(t *testing.T) {
	f := newFollowingFix(t)
	f.cm.seenReply = "2026-09-26T01:02:03Z"
	resp, body := f.mark(t, "sess-bob", map[string]any{})
	if resp.StatusCode != http.StatusOK || body["object"] != "following_activity_read_marker" ||
		body["seen_at"] != "2026-09-26T01:02:03Z" {
		t.Fatalf("empty mark %d %+v", resp.StatusCode, body)
	}
	if f.cm.last().Body != "{}" {
		t.Fatalf("empty mark sent %q", f.cm.last().Body)
	}
	at := "2026-09-25T12:00:00Z"
	f.cm.seenReply = at
	resp, body = f.mark(t, "sess-bob", map[string]any{"seen_at": at})
	if resp.StatusCode != http.StatusOK || body["seen_at"] != at {
		t.Fatalf("timed mark %d %+v", resp.StatusCode, body)
	}
	if f.cm.last().Body != `{"at":"2026-09-25T12:00:00Z"}` {
		t.Fatalf("timed mark sent %q", f.cm.last().Body)
	}
}

func TestV1FollowingActivitiesLimitTooLarge(t *testing.T) {
	f := newFollowingFix(t)
	resp, body := f.list(t, "sess-bob", url.Values{"limit": {"51"}})
	mustCode(t, resp, body, http.StatusBadRequest, "LIMIT_TOO_LARGE")
}
