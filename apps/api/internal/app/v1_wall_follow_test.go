package app

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"kun-galgame-api/pkg/communityclient"
)

const wallStatePath = "/me/walls/{subject_type}/{subject_id}"

func (f *wallFix) wallOp(t *testing.T, method, session, subjectType string, subjectID int, suffix string) (*http.Response, map[string]any) {
	t.Helper()
	u := fmt.Sprintf("/api/v1/me/walls/%s/%d%s", subjectType, subjectID, suffix)
	return f.do(t, method, u, session, wallStatePath+suffix, "", nil)
}

func (f *wallFix) followedWalls(t *testing.T, session string, q url.Values) (*http.Response, map[string]any) {
	t.Helper()
	return f.do(t, http.MethodGet, "/api/v1/me/walls?"+q.Encode(), session, "/me/walls", "", nil)
}

func (f *wallFix) threadID(kind int32, anchor string) int64 {
	f.cm.mu.Lock()
	defer f.cm.mu.Unlock()
	return f.cm.threads[anchorKey(kind, anchor)].id
}

func (f *wallFix) writes() []levelWrite {
	f.cm.mu.Lock()
	defer f.cm.mu.Unlock()
	return append([]levelWrite(nil), f.cm.levelWrites...)
}

func (f *wallFix) readCalls() [][2]int64 {
	f.cm.mu.Lock()
	defer f.cm.mu.Unlock()
	return append([][2]int64(nil), f.cm.reads...)
}

func wantState(t *testing.T, resp *http.Response, body map[string]any, subjectType string, subjectID int, following bool) {
	t.Helper()
	if resp.StatusCode != http.StatusOK || body["object"] != "wall_state" || body["subject_type"] != subjectType ||
		body["subject_id"] != strconv.Itoa(subjectID) || body["id"] != strconv.Itoa(subjectID) || body["is_following"] != following {
		t.Fatalf("state %d %+v, want %s %d following=%v", resp.StatusCode, body, subjectType, subjectID, following)
	}
}

func TestV1WallFollowWritesBothLevels(t *testing.T) {
	f := newWallFix(t)
	anchor := strconv.Itoa(rcGalgame)
	f.cm.seed(communityclient.AnchorSiteGame, anchor, rcBob, "hello", communityclient.PostVisible, 0, time.Now())
	thread := strconv.FormatInt(f.threadID(communityclient.AnchorSiteGame, anchor), 10)

	resp, body := f.wallOp(t, http.MethodPut, "rc-alice", "galgame", rcGalgame, "/follow")
	wantState(t, resp, body, "galgame", rcGalgame, true)
	resp, body = f.wallOp(t, http.MethodDelete, "rc-alice", "galgame", rcGalgame, "/follow")
	wantState(t, resp, body, "galgame", rcGalgame, false)

	want := []levelWrite{
		{"anchor", rcAlice, anchorKey(communityclient.AnchorSiteGame, anchor), communityclient.NotificationWatching},
		{"thread", rcAlice, thread, communityclient.NotificationWatching},
		{"anchor", rcAlice, anchorKey(communityclient.AnchorSiteGame, anchor), communityclient.NotificationNormal},
		{"thread", rcAlice, thread, communityclient.NotificationNormal},
	}
	if got := f.writes(); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("writes %v, want %v", got, want)
	}

	resp, body = f.wallOp(t, http.MethodGet, "rc-alice", "galgame", rcGalgame, "")
	wantState(t, resp, body, "galgame", rcGalgame, false)
}

func TestV1WallFollowWithoutThread(t *testing.T) {
	f := newWallFix(t)
	resp, body := f.wallOp(t, http.MethodPut, "rc-alice", "galgame_resource", rcResource, "/follow")
	wantState(t, resp, body, "galgame_resource", rcResource, true)
	got := f.writes()
	if len(got) != 1 || got[0].scope != "anchor" || got[0].key != anchorKey(communityclient.AnchorSiteResource, "resource:"+strconv.Itoa(rcResource)) {
		t.Errorf("a wall without a thread gets only the anchor write: %v", got)
	}
	resp, body = f.wallOp(t, http.MethodGet, "rc-alice", "galgame_resource", rcResource, "")
	wantState(t, resp, body, "galgame_resource", rcResource, true)
}

func TestV1WallReadMarker(t *testing.T) {
	f := newWallFix(t)
	anchor := strconv.Itoa(rcGalgame)
	f.cm.seed(communityclient.AnchorSiteGame, anchor, rcBob, "one", communityclient.PostVisible, 0, time.Now())
	f.cm.seed(communityclient.AnchorSiteGame, anchor, rcBob, "two", communityclient.PostVisible, 0, time.Now())
	thread := f.threadID(communityclient.AnchorSiteGame, anchor)

	resp, body := f.wallOp(t, http.MethodPut, "rc-alice", "galgame", rcGalgame, "/read-marker")
	wantState(t, resp, body, "galgame", rcGalgame, false)
	if got := f.readCalls(); len(got) != 0 {
		t.Fatalf("a wall the caller has no standing on must not be marked upstream: %v", got)
	}

	if err := f.db.Exec(`INSERT INTO message (type, sender_id, receiver_id, status, updated, community_thread_id, community_post_number)
		VALUES ('replied', ?, ?, 'unread', now(), ?, 2), ('replied', ?, ?, 'unread', now(), ?, 2)`,
		rcBob, rcAlice, thread, rcBob, rcOther, thread).Error; err != nil {
		t.Fatal(err)
	}
	f.wallOp(t, http.MethodPut, "rc-alice", "galgame", rcGalgame, "/follow")
	resp, body = f.wallOp(t, http.MethodPut, "rc-alice", "galgame", rcGalgame, "/read-marker")
	wantState(t, resp, body, "galgame", rcGalgame, true)
	if got := f.readCalls(); len(got) != 1 || got[0] != [2]int64{thread, rcAlice} {
		t.Errorf("reads %v", got)
	}
	if n := f.count(t, `SELECT count(*) FROM message WHERE receiver_id = ? AND community_thread_id = ? AND status = 'read'`, rcAlice, thread); n != 1 {
		t.Errorf("the caller's mirrored notification is read: %d", n)
	}
	if n := f.count(t, `SELECT count(*) FROM message WHERE receiver_id = ? AND community_thread_id = ? AND status = 'unread'`, rcOther, thread); n != 1 {
		t.Errorf("someone else's notification stays unread: %d", n)
	}
}

func TestV1WallFollowRejects(t *testing.T) {
	f := newWallFix(t)
	resp, body := f.wallOp(t, http.MethodPut, "rc-alice", "website", 950000699, "/follow")
	if resp.StatusCode != http.StatusNotFound || body["code"] != "NOT_FOUND" {
		t.Errorf("missing website %d %+v", resp.StatusCode, body)
	}
	resp, body = f.wallOp(t, http.MethodPut, "rc-alice", "galgame", rcGalgameMissing, "/follow")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("missing work %d %+v", resp.StatusCode, body)
	}
	resp, body = f.wallOp(t, http.MethodPut, "rc-alice", "galgame_quiz", rcQuizSpoiler, "/follow")
	if resp.StatusCode != http.StatusForbidden || body["code"] != "QUIZ_ANSWER_REQUIRED" {
		t.Errorf("spoiler quiz %d %+v", resp.StatusCode, body)
	}
	resp, body = f.wallOp(t, http.MethodPut, "rc-carol", "galgame_quiz", rcQuizSpoiler, "/follow")
	wantState(t, resp, body, "galgame_quiz", rcQuizSpoiler, true)
	if got := f.writes(); len(got) != 1 {
		t.Errorf("only carol's follow reached upstream: %v", got)
	}

	resp, body = f.wallOp(t, http.MethodPut, "", "galgame", rcGalgame, "/follow")
	if resp.StatusCode != http.StatusUnauthorized || body["code"] != "MISSING_CREDENTIAL" {
		t.Errorf("anonymous %d %+v", resp.StatusCode, body)
	}
	resp, _ = f.do(t, http.MethodPut, "/api/v1/me/walls/topic/1/follow", "rc-alice", wallStatePath+"/follow", "", nil)
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("unknown subject_type %d", resp.StatusCode)
	}

	f.cm.down.Store(true)
	for _, suffix := range []string{"", "/read-marker", "/follow"} {
		method := http.MethodPut
		if suffix == "" {
			method = http.MethodGet
		}
		resp, body = f.wallOp(t, method, "rc-alice", "galgame_resource", rcResource, suffix)
		if resp.StatusCode != http.StatusServiceUnavailable || body["code"] != "SERVICE_UNAVAILABLE" {
			t.Errorf("%s %s upstream down: %d %+v", method, suffix, resp.StatusCode, body)
		}
	}
	resp, body = f.followedWalls(t, "rc-alice", nil)
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("list upstream down: %d %+v", resp.StatusCode, body)
	}
}

func TestV1FollowedWalls(t *testing.T) {
	f := newWallFix(t)
	for _, w := range []struct {
		typ string
		id  int
	}{{"galgame", rcGalgame}, {"galgame_resource", rcResource}, {"toolset", rcToolsetA}, {"website", rcWebsite}} {
		if resp, body := f.wallOp(t, http.MethodPut, "rc-alice", w.typ, w.id, "/follow"); resp.StatusCode != http.StatusOK {
			t.Fatalf("follow %s: %d %+v", w.typ, resp.StatusCode, body)
		}
	}
	f.wallOp(t, http.MethodDelete, "rc-alice", "toolset", rcToolsetA, "/follow")
	f.wallOp(t, http.MethodPut, "rc-bob", "galgame", rcGalgame, "/follow")

	var got []string
	var galgameItem map[string]any
	cursor := ""
	for range 10 {
		q := url.Values{"limit": {"1"}}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		resp, body := f.followedWalls(t, "rc-alice", q)
		if resp.StatusCode != http.StatusOK || body["object"] != "list" {
			t.Fatalf("page %d %+v", resp.StatusCode, body)
		}
		items, _ := body["items"].([]any)
		for _, it := range items {
			m, _ := it.(map[string]any)
			if m["object"] != "followed_wall" {
				t.Errorf("item %+v", m)
			}
			got = append(got, fmt.Sprint(m["subject_type"], ":", m["subject_id"]))
			switch m["subject_type"] {
			case "galgame":
				galgameItem = m
			case "website":
				site, _ := m["website"].(map[string]any)
				if site["object"] != "website" || site["host"] != rcWebsiteSlug || m["work"] != nil {
					t.Errorf("a website wall carries its site: %+v", m)
				}
			default:
				if m["work"] != nil || m["website"] != nil {
					t.Errorf("only galgame and website walls carry a ref: %+v", m)
				}
			}
		}
		next, _ := body["next_cursor"].(string)
		if next == "" {
			break
		}
		cursor = next
		if resp, body := f.followedWalls(t, "rc-bob", url.Values{"cursor": {next}}); resp.StatusCode != http.StatusBadRequest || body["code"] != "INVALID_CURSOR" {
			t.Errorf("another caller's cursor: %d %+v", resp.StatusCode, body)
		}
	}
	want := []string{
		"galgame:" + strconv.Itoa(rcGalgame),
		"galgame_resource:" + strconv.Itoa(rcResource),
		"website:" + strconv.Itoa(rcWebsite),
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("walked %v, want %v (the unfollowed toolset wall stays out)", got, want)
	}
	work, _ := galgameItem["work"].(map[string]any)
	if work["object"] != "work" || work["id"] != strconv.Itoa(rcGalgame) || work["display_name"] != "RC Game" || work["latin"] != "RC Geemu" ||
		galgameItem["website"] != nil {
		t.Errorf("galgame wall work %+v", galgameItem)
	}

	f.failGC.Store(true)
	if resp, body := f.followedWalls(t, "rc-alice", url.Values{"limit": {"100"}}); resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("catalog down: %d %+v", resp.StatusCode, body)
	}
	if resp, body := f.followedWalls(t, "", nil); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous: %d %+v", resp.StatusCode, body)
	}
}

func TestV1WallFollowBearer(t *testing.T) {
	f := newWallFix(t)
	resp, body := f.wallOp(t, http.MethodPut, "bearer:rc-staff-token", "galgame_resource", rcResource, "/follow")
	wantState(t, resp, body, "galgame_resource", rcResource, true)
}
