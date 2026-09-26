package app

import (
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestV1FollowUserRoundTrip(t *testing.T) {
	f := newFollowFix(t)
	bob := strconv.Itoa(w3UserBob)

	resp, body := f.followOp(t, http.MethodPut, "sess-alice", bob)
	wantFollowState(t, resp, body, bob, true, false, "all")

	resp, body = f.followOp(t, http.MethodPut, "sess-alice", bob)
	wantFollowState(t, resp, body, bob, true, false, "all")

	resp, body = f.followOp(t, http.MethodGet, "sess-alice", bob)
	wantFollowState(t, resp, body, bob, true, false, "all")

	resp, body = f.followOp(t, http.MethodPut, "sess-bob", strconv.Itoa(w3UserAlice))
	wantFollowState(t, resp, body, strconv.Itoa(w3UserAlice), true, true, "all")

	resp, body = f.followOp(t, http.MethodGet, "sess-alice", bob)
	wantFollowState(t, resp, body, bob, true, true, "all")

	resp, body = f.followOp(t, http.MethodDelete, "sess-alice", bob)
	wantFollowState(t, resp, body, bob, false, true, nil)

	resp, body = f.followOp(t, http.MethodDelete, "sess-alice", bob)
	wantFollowState(t, resp, body, bob, false, true, nil)
}

func TestV1FollowUserRejects(t *testing.T) {
	f := newFollowFix(t)
	alice := strconv.Itoa(w3UserAlice)

	resp, body := f.followOp(t, http.MethodPut, "sess-alice", alice)
	mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	fieldErr(t, body, "parameter", "user_id", "NOT_PERMITTED")

	resp, body = f.followOp(t, http.MethodPut, "sess-alice", strconv.Itoa(u2UserGone))
	mustCode(t, resp, body, http.StatusNotFound, "NOT_FOUND")

	resp, body = f.followOp(t, http.MethodPut, "sess-alice", strconv.Itoa(w3UserBanned))
	mustCode(t, resp, body, http.StatusNotFound, "NOT_FOUND")

	f.cm.down.Store(true)
	resp, body = f.followOp(t, http.MethodPut, "sess-alice", strconv.Itoa(w3UserBob))
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
	f.cm.down.Store(false)

	f.cm.followCap.Store(true)
	resp, body = f.followOp(t, http.MethodPut, "sess-alice", strconv.Itoa(w3UserBob))
	mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if body["detail"] != "The following limit is reached." {
		t.Fatalf("detail %v", body["detail"])
	}

	resp, body = f.followOp(t, http.MethodPut, "", strconv.Itoa(w3UserBob))
	mustCode(t, resp, body, http.StatusUnauthorized, "MISSING_CREDENTIAL")
}

func TestV1ListUserFollowersAndFollowing(t *testing.T) {
	f := newFollowFix(t)
	alice := strconv.Itoa(w3UserAlice)
	f.cm.seedFollow(int64(w3UserBob), int64(w3UserAlice), "2026-09-25T02:00:00Z")
	f.cm.seedFollow(int64(w3UserOther), int64(w3UserAlice), "2026-09-25T01:00:00Z")
	f.cm.seedFollow(int64(u2UserGone), int64(w3UserAlice), "2026-09-25T00:00:00Z")
	f.cm.seedFollow(int64(w3UserAlice), int64(w3UserBob), "2026-09-24T00:00:00Z")

	resp, body := f.listFollows(t, "", alice, "followers", "limit=2")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("followers %d %+v", resp.StatusCode, body)
	}
	items := listItems(t, body)
	if len(items) != 2 {
		t.Fatalf("page1 %v", body["items"])
	}
	if items[0]["object"] != "user_follower" {
		t.Fatalf("object %v", items[0]["object"])
	}
	first := items[0]["follower"].(map[string]any)
	if first["id"] != strconv.Itoa(w3UserBob) || first["name"] != "bob" {
		t.Fatalf("newest follower %+v", first)
	}
	cur := nextCursor(body)
	if cur == "" {
		t.Fatal("missing next_cursor")
	}

	resp, body = f.listFollows(t, "", alice, "followers", "limit=2&cursor="+cur)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("followers page2 %d %+v", resp.StatusCode, body)
	}
	items = listItems(t, body)
	if len(items) != 1 {
		t.Fatalf("page2 len %d %+v", len(items), body["items"])
	}
	gone := items[0]["follower"].(map[string]any)
	if gone["id"] != strconv.Itoa(u2UserGone) || gone["name"] != nil {
		t.Fatalf("unresolvable follower must be the placeholder ref: %+v", gone)
	}
	if nextCursor(body) != "" {
		t.Fatalf("last page still has cursor %v", body["next_cursor"])
	}

	resp, body = f.listFollows(t, "", alice, "following", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("following %d %+v", resp.StatusCode, body)
	}
	items = listItems(t, body)
	if len(items) != 1 {
		t.Fatalf("following items %+v", body["items"])
	}
	followee := items[0]["followee"].(map[string]any)
	if items[0]["object"] != "user_followee" || followee["id"] != strconv.Itoa(w3UserBob) {
		t.Fatalf("following %+v", items[0])
	}
}

func TestV1GetUserFollowCounts(t *testing.T) {
	f := newFollowFix(t)
	f.cm.seedFollow(int64(w3UserBob), int64(w3UserAlice), "2026-09-25T00:00:00Z")
	f.cm.seedFollow(int64(w3UserAlice), int64(w3UserOther), "2026-09-25T00:00:00Z")
	f.cm.seedFollow(int64(w3UserAlice), int64(w3UserBob), "2026-09-25T01:00:00Z")

	resp, body := f.callJSON(t, http.MethodGet, "/api/v1/users/"+strconv.Itoa(w3UserAlice), "/users/{user_id}", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get user %d %+v", resp.StatusCode, body)
	}
	counts := userCounts(t, body)
	if asInt(counts["follower_count"]) != 1 || asInt(counts["following_count"]) != 2 {
		t.Fatalf("counts %+v", counts)
	}

	f.cm.down.Store(true)
	resp, body = f.callJSON(t, http.MethodGet, "/api/v1/users/"+strconv.Itoa(w3UserAlice), "/users/{user_id}", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get user after community down %d %+v", resp.StatusCode, body)
	}
	counts = userCounts(t, body)
	if counts["follower_count"] != nil || counts["following_count"] != nil {
		t.Fatalf("failed followStates must null the counts: %+v", counts)
	}
}

func TestV1SetUserFollowNotify(t *testing.T) {
	f := newFollowFix(t)
	bob := strconv.Itoa(w3UserBob)

	resp, body := f.patchFollowNotify(t, "sess-alice", bob, map[string]any{"notify": "feed"})
	mustCode(t, resp, body, http.StatusNotFound, "NOT_FOUND")

	resp, body = f.followOp(t, http.MethodPut, "sess-alice", bob)
	wantFollowState(t, resp, body, bob, true, false, "all")

	resp, body = f.patchFollowNotify(t, "sess-alice", bob, map[string]any{"notify": "feed"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch %d %+v", resp.StatusCode, body)
	}
	wantFollowState(t, resp, body, bob, true, false, "feed")
	if f.cm.lastFollowPatch != `{"notify":"feed"}` {
		t.Fatalf("community body %q", f.cm.lastFollowPatch)
	}

	resp, body = f.followOp(t, http.MethodGet, "sess-alice", bob)
	wantFollowState(t, resp, body, bob, true, false, "feed")

	resp, body = f.followOp(t, http.MethodPut, "sess-alice", bob)
	wantFollowState(t, resp, body, bob, true, false, "feed")

	resp, body = f.patchFollowNotify(t, "sess-alice", bob, map[string]any{})
	mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")

	resp, body = f.patchFollowNotify(t, "", bob, map[string]any{"notify": "feed"})
	mustCode(t, resp, body, http.StatusUnauthorized, "MISSING_CREDENTIAL")

	f.cm.down.Store(true)
	resp, body = f.patchFollowNotify(t, "sess-alice", bob, map[string]any{"notify": "all"})
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
}

func TestV1GetUserFollowStateMapsViewerNotify(t *testing.T) {
	f := newFollowFix(t)
	bob := strconv.Itoa(w3UserBob)

	resp, body := f.followOp(t, http.MethodGet, "sess-alice", bob)
	wantFollowState(t, resp, body, bob, false, false, nil)

	f.cm.seedFollow(int64(w3UserAlice), int64(w3UserBob), "2026-09-25T00:00:00Z")
	resp, body = f.followOp(t, http.MethodGet, "sess-alice", bob)
	wantFollowState(t, resp, body, bob, true, false, "all")

	f.cm.mu.Lock()
	edge := f.cm.userFollows[int64(w3UserAlice)][int64(w3UserBob)]
	edge.notify = "feed"
	f.cm.userFollows[int64(w3UserAlice)][int64(w3UserBob)] = edge
	f.cm.mu.Unlock()
	resp, body = f.followOp(t, http.MethodGet, "sess-alice", bob)
	wantFollowState(t, resp, body, bob, true, false, "feed")
}

func TestV1CreateTopicWritesNoFolloweeTopic(t *testing.T) {
	f := newFollowFix(t)
	f.cm.seedFollow(int64(w3UserBob), int64(w3UserAlice), "2026-09-25T00:00:00Z")
	f.cm.seedFollow(int64(w3UserOther), int64(w3UserAlice), "2026-09-25T01:00:00Z")

	body := map[string]any{
		"title": "hello", "content_markdown": "x", "category": "galgame",
		"sections": []string{"g-news"}, "is_nsfw": false, "access_scope": "public",
	}
	resp, raw := f.doJSON(t, http.MethodPost, "/api/v1/topics", "sess-alice", "/topics", keyUUID(81), nil, body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %s", resp.StatusCode, raw)
	}
	if n := f.cm.followLists.Load(); n != 0 {
		t.Fatalf("listed followers: %d", n)
	}
	var count int64
	f.db.Raw(`SELECT COUNT(*) FROM message WHERE type = ?`, "followee-topic").Scan(&count)
	if count != 0 {
		t.Fatalf("public topic wrote %d followee-topic rows", count)
	}
}

func TestV1CreateTopicNonPublicNotifiesNobody(t *testing.T) {
	f := newFollowFix(t)
	f.cm.seedFollow(int64(w3UserBob), int64(w3UserAlice), "2026-09-25T00:00:00Z")
	body := map[string]any{
		"title": "private", "content_markdown": "x", "category": "galgame",
		"sections": []string{"g-news"}, "is_nsfw": false, "access_scope": "login",
	}
	resp, raw := f.doJSON(t, http.MethodPost, "/api/v1/topics", "sess-alice", "/topics", keyUUID(82), nil, body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %s", resp.StatusCode, raw)
	}
	if n := f.cm.followLists.Load(); n != 0 {
		t.Fatalf("non-public listed followers: %d", n)
	}
	var count int64
	f.db.Raw(`SELECT COUNT(*) FROM message WHERE type = ?`, "followee-topic").Scan(&count)
	if count != 0 {
		t.Fatalf("non-public wrote %d followee-topic rows", count)
	}
}

func TestV1CreateTopicCommunityFailureStillCreates(t *testing.T) {
	f := newFollowFix(t)
	f.cm.seedFollow(int64(w3UserBob), int64(w3UserAlice), "2026-09-25T00:00:00Z")
	f.cm.down.Store(true)
	body := map[string]any{
		"title": "still", "content_markdown": "x", "category": "galgame",
		"sections": []string{"g-news"}, "is_nsfw": false, "access_scope": "public",
	}
	resp, raw := f.doJSON(t, http.MethodPost, "/api/v1/topics", "sess-alice", "/topics", keyUUID(83), nil, body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %s", resp.StatusCode, raw)
	}
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	var count int64
	f.db.Raw(`SELECT COUNT(*) FROM message WHERE type = ?`, "followee-topic").Scan(&count)
	if count != 0 {
		t.Fatalf("failed community wrote %d followee-topic rows", count)
	}
}

func wantFollowState(t *testing.T, resp *http.Response, body map[string]any, userID string, following, followedBy bool, notify any) {
	t.Helper()
	if resp.StatusCode != http.StatusOK || body["object"] != "user_follow_state" ||
		body["id"] != userID || body["user_id"] != userID ||
		body["is_following"] != following || body["is_followed_by"] != followedBy ||
		body["notify_level"] != notify {
		t.Fatalf("state %d %+v, want user %s following=%v followed_by=%v notify_level=%v",
			resp.StatusCode, body, userID, following, followedBy, notify)
	}
}

func listItems(t *testing.T, body map[string]any) []map[string]any {
	t.Helper()
	raw, _ := body["items"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("item %v", item)
		}
		out = append(out, m)
	}
	return out
}
