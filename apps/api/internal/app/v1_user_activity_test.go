package app

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"kun-galgame-api/pkg/communityclient"
)

const uaPath = "/users/{user_id}/activities"

func (f *followingFix) userActivities(t *testing.T, session string, userID int, q url.Values) (*http.Response, map[string]any) {
	t.Helper()
	return f.callJSON(t, http.MethodGet, "/api/v1/users/"+strconv.Itoa(userID)+"/activities?"+q.Encode(), uaPath, session, "", nil)
}

func TestV1UserActivitiesMapsGroupsWithoutActor(t *testing.T) {
	f := newFollowingFix(t)
	work := int64(77)
	f.catalog.items[77] = catalogItem(t, 77, "sfw")
	f.cm.groups = []communityclient.ActivityGroupView{
		faGroup(11, int64(w3UserOther), "kungal", "publish", "2026-09-25",
			faItem(101, int64(w3UserOther), "kungal", "publish", "Hello", "https://www.kungal.com/topic/42", "sfw", &work, "")),
		faGroup(12, int64(w3UserOther), "moyu", "rate", "2026-09-24",
			faItem(202, int64(w3UserOther), "moyu", "rate", "Patch", "https://www.moyu.moe/patch/9", "sfw", nil, "")),
	}

	resp, body := f.userActivities(t, "", w3UserOther, url.Values{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	if got := f.cm.last().Path; got != "/users/"+strconv.Itoa(w3UserOther)+"/activities" {
		t.Fatalf("community path %q", got)
	}
	q := f.cm.query()
	if q.Get("viewer_id") != "0" || q.Get("content_limit") != "sfw" {
		t.Fatalf("community query %q", f.cm.last().RawQuery)
	}
	groups := listItems(t, body)
	if len(groups) != 2 {
		t.Fatalf("groups %+v", groups)
	}
	g0 := groups[0]
	if g0["object"] != "user_activity_group" || fmt.Sprint(g0["id"]) != "11" || g0["site"] != "kungal" ||
		g0["verb"] != "publish" || g0["calendar_date"] != "2026-09-25" || asInt(g0["item_count"]) != 1 {
		t.Fatalf("group %+v", g0)
	}
	if _, has := g0["actor"]; has {
		t.Fatalf("group carries an actor: %+v", g0)
	}
	row, _ := g0["items"].([]any)
	it, _ := row[0].(map[string]any)
	if it["object"] != "following_activity_item" || it["in_site_path"] != "/topic/42" || fmt.Sprint(it["related_work_id"]) != "77" {
		t.Fatalf("item %+v", it)
	}
	w, _ := it["work"].(map[string]any)
	maker, _ := w["maker"].(map[string]any)
	if fmt.Sprint(w["id"]) != "77" || w["release_date"] != "2024-05-01" || w["release_date_precision"] != "month" ||
		maker["display_name"] != "Brand" || w["top_tags"] == nil {
		t.Fatalf("work %+v", it["work"])
	}
	if groups[1]["site"] != "moyu" {
		t.Fatalf("moyu group %+v", groups[1])
	}
	moyu, _ := groups[1]["items"].([]any)
	if m, _ := moyu[0].(map[string]any); m["work"] != nil {
		t.Fatalf("an item without a work has one: %+v", m)
	}
}

func TestV1UserActivitiesSurviveACatalogOutage(t *testing.T) {
	f := newFollowingFix(t)
	work := int64(78)
	f.catalog.items[78] = catalogItem(t, 78, "sfw")
	f.catalog.fail.Store(true)
	f.cm.groups = []communityclient.ActivityGroupView{
		faGroup(13, int64(w3UserOther), "kungal", "rate", "2026-09-25",
			faItem(103, int64(w3UserOther), "kungal", "rate", "Work 78", "https://www.kungal.com/galgame-rating/5", "sfw", &work, "")),
	}
	resp, body := f.userActivities(t, "", w3UserOther, url.Values{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	row, _ := listItems(t, body)[0]["items"].([]any)
	if it, _ := row[0].(map[string]any); it["work"] != nil || fmt.Sprint(it["related_work_id"]) != "78" {
		t.Fatalf("item %+v", it)
	}
}

func TestV1UserActivitiesPassesViewerAndFilters(t *testing.T) {
	f := newFollowingFix(t)
	resp, body := f.userActivities(t, "sess-bob", w3UserOther,
		url.Values{"include_nsfw": {"true"}, "verbs": {"publish,rate"}, "sites": {"moyu"}, "limit": {"5"}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	q := f.cm.query()
	if q.Get("viewer_id") != strconv.Itoa(w3UserBob) || q.Get("content_limit") != "all" ||
		q.Get("verbs") != "publish,rate" || q.Get("sites") != "moyu" || q.Get("limit") != "5" {
		t.Fatalf("community query %q", f.cm.last().RawQuery)
	}
}

func TestV1UserActivitiesUnknownUser(t *testing.T) {
	f := newFollowingFix(t)
	resp, body := f.userActivities(t, "", 987654321, url.Values{})
	mustCode(t, resp, body, http.StatusNotFound, "NOT_FOUND")
}

func TestV1UserActivitiesCursorBoundToUser(t *testing.T) {
	f := newFollowingFix(t)
	f.cm.next = "cm_page2"
	f.cm.groups = []communityclient.ActivityGroupView{
		faGroup(11, int64(w3UserOther), "kungal", "publish", "2026-09-25",
			faItem(101, int64(w3UserOther), "kungal", "publish", "x", "https://www.kungal.com/topic/1", "sfw", nil, "")),
	}
	resp, body := f.userActivities(t, "", w3UserOther, url.Values{"limit": {"1"}})
	own := nextCursor(body)
	if resp.StatusCode != http.StatusOK || own == "" {
		t.Fatalf("first page %d %+v", resp.StatusCode, body)
	}
	resp, body = f.userActivities(t, "", w3UserOther, url.Values{"limit": {"1"}, "cursor": {own}})
	if resp.StatusCode != http.StatusOK || f.cm.query().Get("cursor") != "cm_page2" {
		t.Fatalf("page 2 %d %+v (community %q)", resp.StatusCode, body, f.cm.last().RawQuery)
	}
	resp, body = f.userActivities(t, "", w3UserBob, url.Values{"limit": {"1"}, "cursor": {own}})
	mustCode(t, resp, body, http.StatusBadRequest, "INVALID_CURSOR")
}

func TestV1UserActivitiesCommunityDown(t *testing.T) {
	f := newFollowingFix(t)
	f.cm.down.Store(true)
	resp, body := f.userActivities(t, "", w3UserOther, url.Values{})
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1UserActivitiesSkipsAVerbItDoesNotKnow(t *testing.T) {
	f := newFollowingFix(t)
	f.cm.groups = []communityclient.ActivityGroupView{
		faGroup(11, int64(w3UserOther), "catalog", "favorite", "2026-09-25",
			faItem(101, int64(w3UserOther), "catalog", "favorite", "Work", "https://www.kungal.com/galgame/1", "sfw", nil, "")),
		faGroup(12, int64(w3UserOther), "kungal", "publish", "2026-09-24",
			faItem(102, int64(w3UserOther), "kungal", "publish", "Topic", "https://www.kungal.com/topic/1", "sfw", nil, "")),
	}
	resp, body := f.userActivities(t, "", w3UserOther, url.Values{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	groups := listItems(t, body)
	if len(groups) != 1 || fmt.Sprint(groups[0]["id"]) != "12" {
		t.Fatalf("groups %+v, want only the publish group", groups)
	}
}
