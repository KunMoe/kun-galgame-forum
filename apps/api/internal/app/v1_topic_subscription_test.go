package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

type subNotice struct {
	ID         int
	Sender     int
	Link       string
	Status     string
	ItemCount  int
	ActorCount int
	Created    time.Time
}

func (f *writeFix) subscription(t *testing.T, session string, topicID int) map[string]any {
	t.Helper()
	resp, raw := f.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/topics/%d/subscription", topicID), session,
		"/topics/{topic_id}/subscription", "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get subscription %d %s", resp.StatusCode, raw)
	}
	return problemMap(t, raw)
}

func (f *writeFix) setSubscription(t *testing.T, session string, topicID int, level string) (int, map[string]any) {
	t.Helper()
	resp, raw := f.doJSON(t, http.MethodPut, fmt.Sprintf("/api/v1/topics/%d/subscription", topicID), session,
		"/topics/{topic_id}/subscription", "", nil, map[string]any{"notification_level": level})
	return resp.StatusCode, problemMap(t, raw)
}

func (f *writeFix) markRead(t *testing.T, session string, topicID, floor int) map[string]any {
	t.Helper()
	resp, raw := f.doJSON(t, http.MethodPut, fmt.Sprintf("/api/v1/topics/%d/subscription/read-marker", topicID), session,
		"/topics/{topic_id}/subscription/read-marker", "", nil, map[string]any{"floor": floor})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("read-marker %d %s", resp.StatusCode, raw)
	}
	return problemMap(t, raw)
}

func (f *writeFix) mustReply(t *testing.T, topicID int, session string, key int, content string) int {
	t.Helper()
	resp, out := f.createReply(t, topicID, session, keyUUID(key), content)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("reply %d %v", resp.StatusCode, out)
	}
	return asInt(out["floor"])
}

func (f *writeFix) notices(t *testing.T, receiver int, typ string) []subNotice {
	t.Helper()
	var out []subNotice
	if err := f.db.Raw(`SELECT id, sender_id AS sender, link, status, item_count, actor_count, created
		FROM message WHERE receiver_id = ? AND type = ? ORDER BY id`, receiver, typ).Scan(&out).Error; err != nil {
		t.Fatal(err)
	}
	return out
}

func (f *writeFix) readFloor(t *testing.T, userID, topicID int) (int, bool) {
	t.Helper()
	var floors []int
	if err := f.db.Raw(`SELECT last_read_floor FROM topic_subscription WHERE user_id = ? AND topic_id = ?`,
		userID, topicID).Scan(&floors).Error; err != nil {
		t.Fatal(err)
	}
	if len(floors) == 0 {
		return 0, false
	}
	return floors[0], true
}

func mention(id int) string {
	return fmt.Sprintf("[@m](kungal-user:%d)", id)
}

func TestV1TopicSubscriptionLevels(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)

	got := f.subscription(t, "sess-alice", w3TopicFloors)
	if got["notification_level"] != "watching" || asInt(got["last_read_floor"]) != 3 {
		t.Fatalf("author default %v", got)
	}
	got = f.subscription(t, "sess-bob", w3TopicFloors)
	if got["notification_level"] != "normal" || asInt(got["last_read_floor"]) != 0 || got["topic_id"] != fmt.Sprint(w3TopicFloors) {
		t.Fatalf("reader default %v", got)
	}

	status, got := f.setSubscription(t, "sess-bob", w3TopicFloors, "watching")
	if status != 200 || got["notification_level"] != "watching" || asInt(got["last_read_floor"]) != 3 {
		t.Fatalf("watch starts at the last floor: %d %v", status, got)
	}
	if err := f.db.Exec(`UPDATE topic_subscription SET last_read_floor = 1 WHERE user_id = ? AND topic_id = ?`,
		w3UserBob, w3TopicFloors).Error; err != nil {
		t.Fatal(err)
	}
	if _, got = f.setSubscription(t, "sess-bob", w3TopicFloors, "watching"); asInt(got["last_read_floor"]) != 1 {
		t.Fatalf("watching again reset the position: %v", got)
	}

	_, got = f.setSubscription(t, "sess-bob", w3TopicFloors, "muted")
	if got["notification_level"] != "muted" || asInt(got["last_read_floor"]) != 0 {
		t.Fatalf("muted %v", got)
	}
	_, got = f.setSubscription(t, "sess-bob", w3TopicFloors, "normal")
	if got["notification_level"] != "normal" {
		t.Fatalf("normal %v", got)
	}
	if _, ok := f.readFloor(t, w3UserBob, w3TopicFloors); ok {
		t.Fatal("normal left a row behind")
	}

	status, got = f.setSubscription(t, "sess-bob", w3TopicFloors, "everything")
	if status != http.StatusUnprocessableEntity && status != http.StatusBadRequest {
		t.Fatalf("unknown level %d %v", status, got)
	}
	if status, got = f.setSubscription(t, "sess-bob", w3TopicHidden, "watching"); status != 404 {
		t.Fatalf("hidden topic %d %v", status, got)
	}
	resp, raw := f.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/topics/%d/subscription", w3TopicFloors), "",
		"/topics/{topic_id}/subscription", "", nil, nil)
	if resp.StatusCode != 401 {
		t.Fatalf("anonymous %d %s", resp.StatusCode, raw)
	}
}

func TestV1TopicCreateWatchesOwnTopic(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	resp, raw := f.doJSON(t, http.MethodPost, "/api/v1/topics", "sess-alice", "/topics", keyUUID(90), nil, map[string]any{
		"title": "watch me", "content_markdown": "x", "category": "galgame",
		"sections": []string{"g-news"}, "is_nsfw": false, "access_scope": "public",
	})
	if resp.StatusCode != 201 {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	id := asInt(problemMap(t, raw)["id"])
	if got := f.subscription(t, "sess-alice", id); got["notification_level"] != "watching" {
		t.Fatalf("new topic %v", got)
	}
}

func TestV1SubscriptionFanOutFolds(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	f.setSubscription(t, "sess-bob", w3TopicFloors, "watching")
	f.setSubscription(t, "sess-other", w3TopicFloors, "watching")

	f.mustReply(t, w3TopicFloors, "sess-other", 1, "first")
	bob := f.notices(t, w3UserBob, "subscribed-topic")
	if len(bob) != 1 || bob[0].Sender != w3UserOther || bob[0].ItemCount != 1 || bob[0].ActorCount != 1 ||
		bob[0].Link != fmt.Sprintf("/topic/%d?reply=4", w3TopicFloors) {
		t.Fatalf("first fold %+v", bob)
	}
	if n := f.notices(t, w3UserOther, "subscribed-topic"); len(n) != 0 {
		t.Fatalf("the replier notified themself: %+v", n)
	}
	if n := f.notices(t, w3UserAlice, "subscribed-topic"); len(n) != 0 {
		t.Fatalf("the author got a fold on top of replied: %+v", n)
	}
	if n := f.notices(t, w3UserAlice, "replied"); len(n) != 1 {
		t.Fatalf("the watching author's replied: %+v", n)
	}
	first := bob[0]

	f.mustReply(t, w3TopicFloors, "sess-alice", 2, "second")
	f.mustReply(t, w3TopicFloors, "sess-other", 3, "third")
	bob = f.notices(t, w3UserBob, "subscribed-topic")
	if len(bob) != 1 || bob[0].ID != first.ID {
		t.Fatalf("fold did not continue: %+v", bob)
	}
	if bob[0].Sender != w3UserOther || bob[0].ItemCount != 3 || bob[0].ActorCount != 2 ||
		bob[0].Link != fmt.Sprintf("/topic/%d?reply=4", w3TopicFloors) {
		t.Fatalf("continued fold %+v", bob[0])
	}
	if !bob[0].Created.After(first.Created) {
		t.Fatalf("fold created_at stayed at %v", bob[0].Created)
	}
	other := f.notices(t, w3UserOther, "subscribed-topic")
	if len(other) != 1 || other[0].Sender != w3UserAlice || other[0].ItemCount != 1 {
		t.Fatalf("other's fold opened on alice's reply: %+v", other)
	}

	resp, raw := f.doJSON(t, http.MethodGet, "/api/v1/me/notifications?limit=5", "sess-bob",
		"/me/notifications", "", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, raw)
	}
	var list struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) == 0 || list.Items[0]["notification_type"] != "subscribed_topic_replied" ||
		asInt(list.Items[0]["item_count"]) != 3 || asInt(list.Items[0]["actor_count"]) != 2 {
		t.Fatalf("notification face %v", list.Items)
	}
}

func TestV1SubscriptionMentionAndMute(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	muted := w3MentionMin
	f.putSession(t, "sess-muted", muted)
	f.setSubscription(t, "sess-bob", w3TopicFloors, "watching")
	f.setSubscription(t, "sess-muted", w3TopicFloors, "muted")

	f.mustReply(t, w3TopicFloors, "sess-other", 1, "hey "+mention(w3UserBob)+" and "+mention(muted))
	if n := f.notices(t, w3UserBob, "mentioned"); len(n) != 1 {
		t.Fatalf("mention %+v", n)
	}
	if n := f.notices(t, w3UserBob, "subscribed-topic"); len(n) != 0 {
		t.Fatalf("a mentioned watcher also got the fold: %+v", n)
	}
	if n := f.notices(t, muted, "mentioned"); len(n) != 0 {
		t.Fatalf("a muted reader was mentioned: %+v", n)
	}

	var reply2 int
	if err := f.db.Raw(`SELECT id FROM topic_reply WHERE topic_id = ? AND floor = 2`, w3TopicFloors).Row().Scan(&reply2); err != nil {
		t.Fatal(err)
	}
	f.setSubscription(t, "sess-alice", w3TopicFloors, "muted")
	resp, raw := f.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/replies/%d/comments", reply2), "sess-bob",
		"/replies/{reply_id}/comments", keyUUID(7), nil, map[string]any{"text": "a comment"})
	if resp.StatusCode != 201 {
		t.Fatalf("comment %d %s", resp.StatusCode, raw)
	}
	if n := f.notices(t, w3UserAlice, "commented"); len(n) != 0 {
		t.Fatalf("a muted reply author got commented: %+v", n)
	}
	f.mustReply(t, w3TopicFloors, "sess-bob", 2, "to a muted author")
	for _, n := range f.notices(t, w3UserAlice, "replied") {
		if n.Sender == w3UserBob {
			t.Fatalf("a muted author got replied: %+v", n)
		}
	}
}

func TestV1SubscriptionAuthorAtNormal(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	f.setSubscription(t, "sess-alice", w3TopicFloors, "normal")
	f.mustReply(t, w3TopicFloors, "sess-bob", 1, "hello")
	if n := f.notices(t, w3UserAlice, "replied"); len(n) != 0 {
		t.Fatalf("author at normal got replied: %+v", n)
	}
	f.mustReply(t, w3TopicFloors, "sess-bob", 2, "hi "+mention(w3UserAlice))
	if n := f.notices(t, w3UserAlice, "mentioned"); len(n) != 1 {
		t.Fatalf("author at normal lost the mention: %+v", n)
	}
}

func TestV1SubscriptionRestrictedTopicSendsNoFold(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	if err := f.db.Exec(`INSERT INTO topic_subscription (user_id, topic_id, notification_level) VALUES (?, ?, 'watching')`,
		w3UserBob, w3TopicRole).Error; err != nil {
		t.Fatal(err)
	}
	f.mustReply(t, w3TopicRole, "sess-alice", 1, "role only")
	if n := f.notices(t, w3UserBob, "subscribed-topic"); len(n) != 0 {
		t.Fatalf("role topic fanned out: %+v", n)
	}
}

func TestV1TopicReadMarker(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	f.setSubscription(t, "sess-bob", w3TopicFloors, "watching")
	f.mustReply(t, w3TopicFloors, "sess-other", 1, "four")
	f.mustReply(t, w3TopicFloors, "sess-other", 2, "five")

	if got := f.markRead(t, "sess-bob", w3TopicFloors, 4); asInt(got["last_read_floor"]) != 4 {
		t.Fatalf("mark 4 %v", got)
	}
	if n := f.notices(t, w3UserBob, "subscribed-topic"); len(n) != 1 || n[0].Status != "unread" {
		t.Fatalf("partial read closed the fold: %+v", n)
	}
	if got := f.markRead(t, "sess-bob", w3TopicFloors, 2); asInt(got["last_read_floor"]) != 4 {
		t.Fatalf("marker moved back: %v", got)
	}
	if got := f.markRead(t, "sess-bob", w3TopicFloors, 999); asInt(got["last_read_floor"]) != 5 {
		t.Fatalf("marker not capped at the last floor: %v", got)
	}
	n := f.notices(t, w3UserBob, "subscribed-topic")
	if len(n) != 1 || n[0].Status != "read" {
		t.Fatalf("reading to the end left the fold unread: %+v", n)
	}

	f.mustReply(t, w3TopicFloors, "sess-other", 3, "six")
	n = f.notices(t, w3UserBob, "subscribed-topic")
	if len(n) != 2 || n[1].Status != "unread" || n[1].ItemCount != 1 ||
		n[1].Link != fmt.Sprintf("/topic/%d?reply=6", w3TopicFloors) {
		t.Fatalf("a read fold did not start over: %+v", n)
	}

	floor := f.mustReply(t, w3TopicFloors, "sess-bob", 4, "mine")
	if got, _ := f.readFloor(t, w3UserBob, w3TopicFloors); got != floor {
		t.Fatalf("replying left the marker at %d, want %d", got, floor)
	}

	resp, raw := f.doJSON(t, http.MethodPut, fmt.Sprintf("/api/v1/topics/%d/subscription/read-marker", w3TopicFloors), "sess-bob",
		"/topics/{topic_id}/subscription/read-marker", "", nil, map[string]any{"floor": -1})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("negative floor %d %s", resp.StatusCode, raw)
	}

	if got := f.markRead(t, "sess-other", w3TopicPub, 1); got["notification_level"] != "normal" {
		t.Fatalf("normal reader %v", got)
	}
	if _, ok := f.readFloor(t, w3UserOther, w3TopicPub); ok {
		t.Fatal("reading an unsubscribed topic created a row")
	}
}

func (f *writeFix) listSubscriptions(t *testing.T, session, query string) ([]map[string]any, string) {
	t.Helper()
	resp, raw := f.doJSON(t, http.MethodGet, "/api/v1/me/topic-subscriptions?"+query, session,
		"/me/topic-subscriptions", "", nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("list %d %s", resp.StatusCode, raw)
	}
	var out struct {
		Items      []map[string]any `json:"items"`
		NextCursor string           `json:"next_cursor"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out.Items, out.NextCursor
}

func TestV1ListTopicSubscriptions(t *testing.T) {
	f := newWriteFix(t, nil)
	f.alice(t)
	tie := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for _, row := range []struct {
		topic int
		read  int
		at    time.Time
	}{
		{w3TopicFloors, 1, tie},
		{w3TopicPub, 0, tie},
		{w3TopicOld, 0, tie},
		{w3TopicPaid, 0, tie.Add(-time.Hour)},
		{w3TopicNSFW, 0, tie.Add(time.Hour)},
		{w3TopicHidden, 0, tie.Add(2 * time.Hour)},
		{w3TopicRole, 0, tie.Add(3 * time.Hour)},
	} {
		if err := f.db.Exec(`INSERT INTO topic_subscription (user_id, topic_id, notification_level, last_read_floor, activity_at)
			VALUES (?, ?, 'watching', ?, ?)`, w3UserBob, row.topic, row.read, row.at).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := f.db.Exec(`INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, created, updated)
		VALUES (?, 'own', 1, ?, ?, 0, now(), now())`, w3ReplyMin+50, w3UserBob, w3TopicPub).Error; err != nil {
		t.Fatal(err)
	}

	var ids []string
	cursor := ""
	for range 10 {
		q := "limit=1"
		if cursor != "" {
			q += "&cursor=" + cursor
		}
		items, next := f.listSubscriptions(t, "sess-bob", q)
		for _, it := range items {
			ids = append(ids, strID(it["id"]))
		}
		if next == "" {
			break
		}
		cursor = next
	}
	want := []string{
		fmt.Sprint(w3TopicFloors), fmt.Sprint(w3TopicOld), fmt.Sprint(w3TopicPub), fmt.Sprint(w3TopicPaid),
	}
	if fmt.Sprint(ids) != fmt.Sprint(want) {
		t.Fatalf("walk %v, want %v (ties broken by topic id desc; hidden, role and nsfw left out)", ids, want)
	}

	items, _ := f.listSubscriptions(t, "sess-bob", "include_nsfw=true&limit=10")
	if len(items) != 5 || strID(items[0]["id"]) != fmt.Sprint(w3TopicNSFW) {
		t.Fatalf("include_nsfw %v", items)
	}

	items, _ = f.listSubscriptions(t, "sess-bob", "has_unread=true&limit=10")
	if len(items) != 1 || strID(items[0]["id"]) != fmt.Sprint(w3TopicFloors) {
		t.Fatalf("has_unread %v", items)
	}
	it := items[0]
	if asInt(it["unread_reply_count"]) != 1 || asInt(it["first_unread_floor"]) != 2 || asInt(it["last_read_floor"]) != 1 {
		t.Fatalf("unread fields %v", it)
	}
	topic, _ := it["topic"].(map[string]any)
	if topic == nil || topic["object"] != "topic" || topic["title"] != "floors" {
		t.Fatalf("topic summary %v", it["topic"])
	}

	resp, raw := f.doJSON(t, http.MethodGet, "/api/v1/me/topic-subscriptions?has_unread=true&cursor="+cursor, "sess-bob",
		"/me/topic-subscriptions", "", nil, nil)
	if resp.StatusCode != 400 || problemMap(t, raw)["code"] != "INVALID_CURSOR" {
		t.Fatalf("cursor reused across filters %d %s", resp.StatusCode, raw)
	}
	resp, raw = f.doJSON(t, http.MethodGet, "/api/v1/me/topic-subscriptions?limit=101", "sess-bob",
		"/me/topic-subscriptions", "", nil, nil)
	if resp.StatusCode != 400 || problemMap(t, raw)["code"] != "LIMIT_TOO_LARGE" {
		t.Fatalf("limit 101 %d %s", resp.StatusCode, raw)
	}
}
