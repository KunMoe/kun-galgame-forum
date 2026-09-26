package app

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"kun-galgame-api/internal/apiv1/repr"
)

const (
	faTopicMin = 930001001
	faReplyMin = 930001501
	faPath     = "/me/following-activities"
)

type followingFix struct {
	*followFix
}

func newFollowingFix(t *testing.T) *followingFix {
	t.Helper()
	return &followingFix{newFollowFix(t)}
}

func (f *followingFix) topic(t *testing.T, id, user int, nsfw bool, created time.Time) {
	t.Helper()
	if err := f.db.Exec(`INSERT INTO topic (
		id, title, content, view, status, category, status_update_time, created, updated,
		user_id, is_nsfw, access_scope, cover_images, like_count, dislike_count, reply_count, comment_count,
		favorite_count, upvote_count, view_7d, view_30d, hidden_by, last_reply_floor
	) VALUES (?, ?, 'body', 0, 0, 'galgame', ?, ?, ?, ?, ?, 'public', '', 0, 0, 0, 0, 0, 0, 0, 0, '', 0)`,
		id, fmt.Sprintf("t%d", id), created, created, created, user, nsfw).Error; err != nil {
		t.Fatalf("topic %d: %v", id, err)
	}
}

func (f *followingFix) reply(t *testing.T, id, topicID, user int, created time.Time) {
	t.Helper()
	if err := f.db.Exec(`INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, like_count, created, updated)
		VALUES (?, 'a reply', 1, ?, ?, 0, 0, ?, ?)`, id, user, topicID, created, created).Error; err != nil {
		t.Fatalf("reply %d: %v", id, err)
	}
}

func (f *followingFix) follows(viewer int, followees ...int) {
	for _, id := range followees {
		f.cm.seedFollow(int64(viewer), int64(id), "2026-09-25T00:00:00Z")
	}
}

func (f *followingFix) list(t *testing.T, session string, q url.Values) (*http.Response, map[string]any) {
	t.Helper()
	return f.callJSON(t, http.MethodGet, "/api/v1"+faPath+"?"+q.Encode(), faPath, session, "", nil)
}

func (f *followingFix) summary(t *testing.T, session string, q url.Values) (*http.Response, map[string]any) {
	t.Helper()
	return f.callJSON(t, http.MethodGet, "/api/v1"+faPath+"/summary?"+q.Encode(), faPath+"/summary", session, "", nil)
}

func (f *followingFix) mark(t *testing.T, session, seenAt string) (*http.Response, map[string]any) {
	t.Helper()
	return f.callJSON(t, http.MethodPut, "/api/v1"+faPath+"/read-marker", faPath+"/read-marker", session, "",
		map[string]any{"seen_at": seenAt})
}

func (f *followingFix) walk(t *testing.T, session string, q url.Values, limit int) []map[string]any {
	t.Helper()
	q.Set("limit", strconv.Itoa(limit))
	var all []map[string]any
	for range 200 {
		resp, body := f.list(t, session, q)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("list %d %+v", resp.StatusCode, body)
		}
		all = append(all, listItems(t, body)...)
		next, _ := body["next_cursor"].(string)
		if next == "" {
			return all
		}
		q.Set("cursor", next)
	}
	t.Fatal("walk did not end")
	return nil
}

func unseenCount(t *testing.T, resp *http.Response, body map[string]any) int {
	t.Helper()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("summary %d %+v", resp.StatusCode, body)
	}
	n, ok := body["unseen_count"].(float64)
	if !ok {
		t.Fatalf("unseen_count %v", body["unseen_count"])
	}
	return int(n)
}

func followedActivity(t *testing.T, item map[string]any) map[string]any {
	t.Helper()
	if item["object"] != "following_activity" || item["site"] != "kungal" {
		t.Fatalf("entry %+v", item)
	}
	a, ok := item["activity"].(map[string]any)
	if !ok {
		t.Fatalf("activity %+v", item["activity"])
	}
	return a
}

func performerID(a map[string]any) string {
	p, _ := a["performer"].(map[string]any)
	return fmt.Sprint(p["id"])
}

func TestV1FollowingActivitiesOnlyFollowees(t *testing.T) {
	f := newFollowingFix(t)
	base := time.Now().Add(-time.Hour).Truncate(time.Second)
	f.topic(t, faTopicMin, w3UserOther, false, base)
	f.topic(t, faTopicMin+1, w3UserGrant, false, base.Add(time.Minute))
	f.topic(t, faTopicMin+2, w3UserBanned, false, base.Add(2*time.Minute))
	f.follows(w3UserBob, w3UserOther, w3UserGrant, w3UserBanned)

	items := f.walk(t, "sess-bob", url.Values{}, 20)
	want := []string{strconv.Itoa(w3UserGrant), strconv.Itoa(w3UserOther)}
	if len(items) != len(want) {
		t.Fatalf("items %d, want %d: %+v", len(items), len(want), items)
	}
	for i, it := range items {
		a := followedActivity(t, it)
		if a["activity_type"] != "topic_creation" || performerID(a) != want[i] {
			t.Fatalf("item %d: %v by %s, want topic_creation by %s", i, a["activity_type"], performerID(a), want[i])
		}
	}

	resp, body := f.summary(t, "sess-bob", url.Values{})
	if n := unseenCount(t, resp, body); n != 3 {
		t.Fatalf("unseen %d, want 3 (the banned author's row is counted, not listed)", n)
	}
	if body["last_seen_at"] != nil {
		t.Fatalf("last_seen_at %v, want null", body["last_seen_at"])
	}
}

func TestV1FollowingActivitiesFollowsNobody(t *testing.T) {
	f := newFollowingFix(t)
	f.topic(t, faTopicMin, w3UserOther, false, time.Now().Add(-time.Hour))

	resp, body := f.list(t, "sess-bob", url.Values{})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d %+v", resp.StatusCode, body)
	}
	if items := listItems(t, body); len(items) != 0 {
		t.Fatalf("items %+v, want none", items)
	}
	if _, ok := body["next_cursor"]; ok {
		t.Fatalf("next_cursor on an empty list")
	}
	resp, body = f.summary(t, "sess-bob", url.Values{})
	if n := unseenCount(t, resp, body); n != 0 {
		t.Fatalf("unseen %d, want 0", n)
	}
}

func TestV1FollowingActivitiesNSFW(t *testing.T) {
	f := newFollowingFix(t)
	base := time.Now().Add(-time.Hour).Truncate(time.Second)
	f.topic(t, faTopicMin, w3UserOther, false, base)
	f.topic(t, faTopicMin+1, w3UserOther, true, base.Add(time.Minute))
	f.follows(w3UserBob, w3UserOther)

	if items := f.walk(t, "sess-bob", url.Values{}, 20); len(items) != 1 {
		t.Fatalf("sfw items %d, want 1", len(items))
	}
	if items := f.walk(t, "sess-bob", url.Values{"include_nsfw": {"true"}}, 20); len(items) != 2 {
		t.Fatalf("nsfw items %d, want 2", len(items))
	}
	resp, body := f.summary(t, "sess-bob", url.Values{})
	if n := unseenCount(t, resp, body); n != 1 {
		t.Fatalf("sfw unseen %d, want 1", n)
	}
}

func TestV1FollowingActivitiesWalkWithTies(t *testing.T) {
	f := newFollowingFix(t)
	tie := time.Now().Add(-3 * time.Hour).Truncate(time.Second)
	for i := range 5 {
		f.topic(t, faTopicMin+i, []int{w3UserOther, w3UserGrant}[i%2], false, tie)
	}
	f.topic(t, faTopicMin+5, w3UserOther, false, tie.Add(time.Minute))
	f.reply(t, faReplyMin, faTopicMin+5, w3UserGrant, tie)
	f.follows(w3UserBob, w3UserOther, w3UserGrant)

	var want []string
	if err := f.db.Raw(`SELECT id::text FROM feed_activity WHERE user_id IN (?, ?) AND NOT is_nsfw
		ORDER BY created DESC, type DESC, source_id DESC`, w3UserOther, w3UserGrant).Scan(&want).Error; err != nil {
		t.Fatal(err)
	}
	if len(want) != 7 {
		t.Fatalf("seeded rows %d, want 7", len(want))
	}
	for _, limit := range []int{1, 2, 3, 100} {
		items := f.walk(t, "sess-bob", url.Values{}, limit)
		var got []string
		for _, it := range items {
			got = append(got, fmt.Sprint(followedActivity(t, it)["id"]))
		}
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("limit %d: %v, want %v", limit, got, want)
		}
	}
}

func TestV1FollowingActivitiesCursorIsItsOwn(t *testing.T) {
	f := newFollowingFix(t)
	base := time.Now().Add(-time.Hour).Truncate(time.Second)
	f.topic(t, faTopicMin, w3UserOther, false, base)
	f.topic(t, faTopicMin+1, w3UserOther, false, base.Add(time.Minute))
	f.follows(w3UserBob, w3UserOther)

	resp, body := f.callJSON(t, http.MethodGet, "/api/v1/activities?limit=1", "/activities", "", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("activities %d %+v", resp.StatusCode, body)
	}
	foreign, _ := body["next_cursor"].(string)
	if foreign == "" {
		t.Fatal("no cursor from listActivities")
	}
	resp, body = f.list(t, "sess-bob", url.Values{"cursor": {foreign}})
	mustCode(t, resp, body, http.StatusBadRequest, "INVALID_CURSOR")

	resp, body = f.list(t, "sess-bob", url.Values{"limit": {"1"}})
	own, _ := body["next_cursor"].(string)
	if resp.StatusCode != http.StatusOK || own == "" {
		t.Fatalf("first page %d %+v", resp.StatusCode, body)
	}
	resp, body = f.list(t, "sess-bob", url.Values{"limit": {"1"}, "cursor": {own}, "include_nsfw": {"true"}})
	mustCode(t, resp, body, http.StatusBadRequest, "INVALID_CURSOR")
}

func TestV1FollowingActivitiesSeeAFollowAtOnce(t *testing.T) {
	f := newFollowingFix(t)
	f.topic(t, faTopicMin, w3UserOther, false, time.Now().Add(-time.Hour))

	resp, body := f.list(t, "sess-bob", url.Values{})
	if resp.StatusCode != http.StatusOK || len(listItems(t, body)) != 0 {
		t.Fatalf("before %d %+v", resp.StatusCode, body)
	}
	resp, body = f.followOp(t, http.MethodPut, "sess-bob", strconv.Itoa(w3UserOther))
	wantFollowState(t, resp, body, strconv.Itoa(w3UserOther), true, false)
	if items := f.walk(t, "sess-bob", url.Values{}, 20); len(items) != 1 {
		t.Fatalf("after follow %d items, want 1", len(items))
	}
	resp, body = f.followOp(t, http.MethodDelete, "sess-bob", strconv.Itoa(w3UserOther))
	wantFollowState(t, resp, body, strconv.Itoa(w3UserOther), false, false)
	if items := f.walk(t, "sess-bob", url.Values{}, 20); len(items) != 0 {
		t.Fatalf("after unfollow %d items, want 0", len(items))
	}
}

func TestV1FollowingActivitiesCommunityDown(t *testing.T) {
	f := newFollowingFix(t)
	f.follows(w3UserBob, w3UserOther)
	f.cm.down.Store(true)

	resp, body := f.list(t, "sess-bob", url.Values{})
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
	resp, body = f.summary(t, "sess-bob", url.Values{})
	mustCode(t, resp, body, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE")
	resp, body = f.mark(t, "sess-bob", faTime(time.Now().Add(-time.Minute)))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("mark %d %+v", resp.StatusCode, body)
	}

	resp, body = f.list(t, "", url.Values{})
	mustCode(t, resp, body, http.StatusUnauthorized, "MISSING_CREDENTIAL")
}

func TestV1FollowingActivitySummaryWindow(t *testing.T) {
	f := newFollowingFix(t)
	now := time.Now()
	f.topic(t, faTopicMin, w3UserOther, false, now.Add(-8*24*time.Hour))
	f.topic(t, faTopicMin+1, w3UserOther, false, now.Add(-2*24*time.Hour))
	f.follows(w3UserBob, w3UserOther)

	resp, body := f.summary(t, "sess-bob", url.Values{})
	if n := unseenCount(t, resp, body); n != 1 {
		t.Fatalf("unmarked unseen %d, want 1 (only the last seven days)", n)
	}
}

func TestV1FollowingActivitySummaryAfterMark(t *testing.T) {
	f := newFollowingFix(t)
	mark := time.Now().Add(-time.Hour).Truncate(time.Second)
	f.topic(t, faTopicMin, w3UserOther, false, mark.Add(-time.Minute))
	f.topic(t, faTopicMin+1, w3UserOther, false, mark)
	f.topic(t, faTopicMin+2, w3UserOther, false, mark.Add(500*time.Millisecond))
	f.topic(t, faTopicMin+3, w3UserOther, false, mark.Add(time.Second))
	f.reply(t, faReplyMin, faTopicMin, w3UserOther, mark.Add(2*time.Second))
	f.follows(w3UserBob, w3UserOther)

	resp, body := f.mark(t, "sess-bob", faTime(mark))
	if resp.StatusCode != http.StatusOK || body["seen_at"] != faTime(mark) {
		t.Fatalf("mark %d %+v", resp.StatusCode, body)
	}
	resp, body = f.summary(t, "sess-bob", url.Values{})
	if n := unseenCount(t, resp, body); n != 2 {
		t.Fatalf("unseen %d, want 2: the topic a second after the mark and the reply", n)
	}
	if body["last_seen_at"] != faTime(mark) {
		t.Fatalf("last_seen_at %v", body["last_seen_at"])
	}
	resp, body = f.summary(t, "sess-bob", url.Values{"activity_types": {"topic_creation"}})
	if n := unseenCount(t, resp, body); n != 1 {
		t.Fatalf("topic-only unseen %d, want 1", n)
	}
	resp, body = f.summary(t, "sess-bob", url.Values{"activity_types": {"topic_reply_creation"}})
	if n := unseenCount(t, resp, body); n != 1 {
		t.Fatalf("reply-only unseen %d, want 1", n)
	}
}

func TestV1FollowingActivitySummaryCap(t *testing.T) {
	f := newFollowingFix(t)
	base := time.Now().Add(-2 * time.Hour)
	if err := f.db.Exec(`INSERT INTO topic (
		id, title, content, view, status, category, status_update_time, created, updated,
		user_id, is_nsfw, access_scope, cover_images, like_count, dislike_count, reply_count, comment_count,
		favorite_count, upvote_count, view_7d, view_30d, hidden_by, last_reply_floor
	) SELECT ? + g, 'bulk', 'body', 0, 0, 'galgame', ?, ?, ?, ?, false, 'public', '', 0, 0, 0, 0, 0, 0, 0, 0, '', 0
	FROM generate_series(0, 149) g`, faTopicMin, base, base, base, w3UserOther).Error; err != nil {
		t.Fatal(err)
	}
	f.follows(w3UserBob, w3UserOther)
	resp, body := f.summary(t, "sess-bob", url.Values{})
	if n := unseenCount(t, resp, body); n != 100 {
		t.Fatalf("unseen %d, want the cap of 100", n)
	}
}

func TestV1FollowingActivityReadMarker(t *testing.T) {
	f := newFollowingFix(t)
	later := time.Now().Add(-time.Hour).Truncate(time.Second)
	earlier := later.Add(-24 * time.Hour)

	resp, body := f.mark(t, "sess-bob", faTime(later))
	if resp.StatusCode != http.StatusOK || body["object"] != "following_activity_read_marker" || body["seen_at"] != faTime(later) {
		t.Fatalf("mark %d %+v", resp.StatusCode, body)
	}
	resp, body = f.mark(t, "sess-bob", faTime(earlier))
	if resp.StatusCode != http.StatusOK || body["seen_at"] != faTime(later) {
		t.Fatalf("an earlier mark moved it back: %d %+v", resp.StatusCode, body)
	}

	before := time.Now().Truncate(time.Second)
	resp, body = f.mark(t, "sess-bob", faTime(time.Now().Add(24*time.Hour)))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("future mark %d %+v", resp.StatusCode, body)
	}
	stored, err := time.Parse(time.RFC3339, fmt.Sprint(body["seen_at"]))
	if err != nil || stored.Before(before) || stored.After(time.Now()) {
		t.Fatalf("future mark stored %v, want the current time", body["seen_at"])
	}

	resp, body = f.mark(t, "sess-other", "2026-02-30T00:00:00Z")
	mustCode(t, resp, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	resp, body = f.mark(t, "", faTime(later))
	mustCode(t, resp, body, http.StatusUnauthorized, "MISSING_CREDENTIAL")
}

func faTime(t time.Time) string {
	return string(repr.Timestamp(t))
}
