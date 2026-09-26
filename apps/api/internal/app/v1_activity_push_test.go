package app

import (
	"context"
	"testing"
	"time"

	"kun-galgame-api/pkg/communityclient"
)

func TestActivityPushTrigger(t *testing.T) {
	f := newPushFix(t)
	now := time.Now().UTC().Truncate(time.Second)
	if err := f.db.Exec(`INSERT INTO feed_activity (type, source_id, user_id, content, link, is_nsfw, created)
		VALUES ('TOPIC_CREATION', ?, ?, 't', '/topic/1', false, ?)`, apTopicA, w3UserOther, now).Error; err != nil {
		t.Fatal(err)
	}
	if f.queueCount(t) != 1 {
		t.Fatalf("after insert queue = %d, want 1", f.queueCount(t))
	}
	if err := f.db.Exec(`UPDATE activity_push_queue SET backfill = true WHERE type = 'TOPIC_CREATION' AND source_id = ?`, apTopicA).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Exec(`UPDATE feed_activity SET content = 'changed' WHERE type = 'TOPIC_CREATION' AND source_id = ?`, apTopicA).Error; err != nil {
		t.Fatal(err)
	}
	if f.queueCount(t) != 1 {
		t.Fatalf("after update queue = %d, want 1", f.queueCount(t))
	}
	backfill, ok := f.queueRow(t, "TOPIC_CREATION", apTopicA)
	if !ok || !backfill {
		t.Fatal("update cleared backfill")
	}
	if err := f.db.Exec(`DELETE FROM feed_activity WHERE type = 'TOPIC_CREATION' AND source_id = ?`, apTopicA).Error; err != nil {
		t.Fatal(err)
	}
	if f.queueCount(t) != 1 {
		t.Fatalf("after delete queue = %d, want 1", f.queueCount(t))
	}
}

func TestActivityPushLiveTopic(t *testing.T) {
	f := newPushFix(t)
	created := time.Date(2026, 9, 25, 12, 0, 0, 123000000, time.UTC)
	f.topic(t, apTopicA, w3UserOther, false, created, "Hello topic")
	before := time.Now()
	if _, err := f.pusher.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	after := time.Now()
	items := f.lastItems(t)
	if len(items) != 1 {
		t.Fatalf("items = %d", len(items))
	}
	it := items[0]
	if it.Key != "topic_creation:940100101" || it.Verb != "publish" || it.ObjectKind != "topic" || it.ObjectLabel != "话题" {
		t.Fatalf("identity %+v", it)
	}
	if it.ActorID != int64(w3UserOther) || it.Title != "Hello topic" || !it.Notify {
		t.Fatalf("fields %+v", it)
	}
	if it.URL != "https://www.kungal.com/topic/940100101" || it.ContentLimit != "sfw" {
		t.Fatalf("url/limit %+v", it)
	}
	if it.Revision < before.UnixMicro() || it.Revision > after.UnixMicro() {
		t.Fatalf("revision %d outside [%d, %d]", it.Revision, before.UnixMicro(), after.UnixMicro())
	}
	if _, ok := f.queueRow(t, "TOPIC_CREATION", apTopicA); ok {
		t.Fatal("queue row remained")
	}
	actor, rev, removed, ok := f.sentRow(t, "TOPIC_CREATION", apTopicA)
	if !ok || actor != w3UserOther || removed || rev != it.Revision {
		t.Fatalf("sent actor=%d rev=%d removed=%v found=%v", actor, rev, removed, ok)
	}
}

func TestActivityPushBackfillNotifyFalse(t *testing.T) {
	f := newPushFix(t)
	f.topic(t, apTopicA, w3UserOther, false, time.Now().UTC().Add(-time.Hour), "Backfill")
	f.resetQueue(t)
	if err := f.db.Exec(`INSERT INTO activity_push_queue (type, source_id, backfill) VALUES ('TOPIC_CREATION', ?, true)`, apTopicA).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := f.pusher.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	it := f.lastItems(t)[0]
	if it.Notify {
		t.Fatalf("backfill notified: %+v", it)
	}
}

func TestActivityPushTombstoneAndNeverPushedDelete(t *testing.T) {
	f := newPushFix(t)
	f.topic(t, apTopicA, w3UserOther, false, time.Now().UTC().Add(-time.Hour), "Gone")
	if _, err := f.pusher.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	actor, _, _, ok := f.sentRow(t, "TOPIC_CREATION", apTopicA)
	if !ok {
		t.Fatal("missing sent")
	}
	if err := f.db.Exec(`DELETE FROM feed_activity WHERE type = 'TOPIC_CREATION' AND source_id = ?`, apTopicA).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := f.pusher.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	it := f.lastItems(t)[0]
	if !it.Removed || it.ActorID != int64(actor) || it.Key != "topic_creation:940100101" {
		t.Fatalf("tombstone %+v", it)
	}

	n0 := len(f.writes())
	if err := f.db.Exec(`INSERT INTO feed_activity (type, source_id, user_id, content, link, created)
		VALUES ('TODO_CREATION', ?, ?, 'todo', '/update/todo', now())`, apTodo, w3UserOther).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := f.pusher.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := f.db.Exec(`DELETE FROM feed_activity WHERE type = 'TODO_CREATION' AND source_id = ?`, apTodo).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := f.pusher.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(f.writes()) != n0 {
		t.Fatalf("never-pushed delete sent %d extra writes", len(f.writes())-n0)
	}
}

func TestActivityPushAssembleFailureKeepsQueue(t *testing.T) {
	f := newPushFix(t)
	f.topic(t, apTopicA, w3UserOther, false, time.Now().UTC().Add(-time.Hour), "Fail")
	f.failOA.Store(true)
	if _, err := f.pusher.RunOnce(context.Background()); err == nil {
		t.Fatal("want assemble error")
	}
	if f.queueCount(t) == 0 {
		t.Fatal("queue drained after assemble failure")
	}
	if len(f.writes()) != 0 {
		t.Fatalf("posted %d times", len(f.writes()))
	}
}

func TestActivityPushNeverSendsMessageUpvoteOrTodo(t *testing.T) {
	f := newPushFix(t)
	if err := f.db.Exec(`INSERT INTO feed_activity (type, source_id, user_id, content, link, created) VALUES
		('MESSAGE_UPVOTE', ?, ?, 'up', '/topic/1', now()),
		('TODO_CREATION', ?, ?, 'todo', '/update/todo', now())`,
		apMsgUp, w3UserOther, apTodo, w3UserOther).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := f.pusher.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(f.writes()) != 0 {
		t.Fatalf("sent %+v", f.writes())
	}
	if _, ok := f.queueRow(t, "MESSAGE_UPVOTE", apMsgUp); ok {
		t.Fatal("MESSAGE_UPVOTE queue remained")
	}
	if _, ok := f.queueRow(t, "TODO_CREATION", apTodo); ok {
		t.Fatal("TODO_CREATION queue remained")
	}
}

func TestActivityPushReenqueueKeepsRow(t *testing.T) {
	f := newPushFix(t)
	f.topic(t, apTopicA, w3UserOther, false, time.Now().UTC().Add(-time.Hour), "Race")
	claims, err := f.pusher.Claim(context.Background())
	if err != nil || len(claims) == 0 {
		t.Fatalf("claim %d %v", len(claims), err)
	}
	rev := time.Now().UnixMicro()
	handled, err := f.pusher.PushClaimed(context.Background(), claims, rev)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.db.Exec(`UPDATE activity_push_queue SET enqueued = clock_timestamp()
		WHERE type = 'TOPIC_CREATION' AND source_id = ?`, apTopicA).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.pusher.Ack(context.Background(), handled); err != nil {
		t.Fatal(err)
	}
	if _, ok := f.queueRow(t, "TOPIC_CREATION", apTopicA); !ok {
		t.Fatal("re-enqueued row was deleted")
	}
}

func TestActivityPushBisect422(t *testing.T) {
	f := newPushFix(t)
	created := time.Now().UTC().Add(-time.Hour)
	f.topic(t, apTopicA, w3UserOther, false, created, "Good")
	f.topic(t, apTopicB, w3UserAlice, false, created.Add(time.Second), "Bad")
	f.cm.mu.Lock()
	f.cm.bisect = true
	f.cm.badKey = "topic_creation:940100102"
	f.cm.mu.Unlock()
	if _, err := f.pusher.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	writes := f.writes()
	if len(writes) < 2 {
		t.Fatalf("writes = %d, want bisect", len(writes))
	}
	if _, ok := f.queueRow(t, "TOPIC_CREATION", apTopicB); ok {
		t.Fatal("bad item queue remained")
	}
	if _, _, _, ok := f.sentRow(t, "TOPIC_CREATION", apTopicA); !ok {
		t.Fatal("good item was not committed")
	}
	if _, _, _, ok := f.sentRow(t, "TOPIC_CREATION", apTopicB); ok {
		t.Fatal("bad item was recorded")
	}
}

func TestActivityPushStaleAndInvalid(t *testing.T) {
	f := newPushFix(t)
	f.topic(t, apTopicA, w3UserOther, false, time.Now().UTC().Add(-time.Hour), "Stale")
	if _, err := f.pusher.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, rev1, _, ok := f.sentRow(t, "TOPIC_CREATION", apTopicA)
	if !ok {
		t.Fatal("missing sent")
	}
	f.cm.mu.Lock()
	f.cm.outcome["topic_creation:940100101"] = "stale"
	f.cm.mu.Unlock()
	if err := f.db.Exec(`INSERT INTO activity_push_queue (type, source_id) VALUES ('TOPIC_CREATION', ?)
		ON CONFLICT (type, source_id) DO UPDATE SET enqueued = clock_timestamp()`, apTopicA).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := f.pusher.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, rev2, _, ok := f.sentRow(t, "TOPIC_CREATION", apTopicA)
	if !ok || rev2 != rev1 {
		t.Fatalf("stale moved sent %d → %d", rev1, rev2)
	}

	f.topic(t, apTopicB, w3UserAlice, false, time.Now().UTC().Add(-time.Hour), "Invalid")
	f.cm.mu.Lock()
	f.cm.outcome["topic_creation:940100102"] = "invalid"
	f.cm.mu.Unlock()
	if _, err := f.pusher.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, _, _, ok := f.sentRow(t, "TOPIC_CREATION", apTopicB); ok {
		t.Fatal("invalid was recorded")
	}
	if _, ok := f.queueRow(t, "TOPIC_CREATION", apTopicB); ok {
		t.Fatal("invalid queue remained")
	}
}

func TestActivityPushReconcile(t *testing.T) {
	f := newPushFix(t)
	created := time.Date(2026, 9, 20, 8, 0, 0, 0, time.UTC)
	f.topic(t, apTopicA, w3UserOther, false, created, "Same")
	if _, err := f.pusher.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.resetQueue(t)
	f.cm.mu.Lock()
	view := f.cm.stored["topic_creation:940100101"]
	view.Notify = !view.Notify
	f.cm.list = []communityclient.SiteActivityView{view}
	f.cm.mu.Unlock()
	f.pusher.Reconcile(context.Background())
	if _, ok := f.queueRow(t, "TOPIC_CREATION", apTopicA); ok {
		t.Fatal("notify-only difference enqueued")
	}

	f.cm.mu.Lock()
	view = f.cm.stored["topic_creation:940100101"]
	f.cm.list = []communityclient.SiteActivityView{view}
	f.cm.mu.Unlock()
	f.pusher.Reconcile(context.Background())
	if _, ok := f.queueRow(t, "TOPIC_CREATION", apTopicA); ok {
		t.Fatal("identical item enqueued")
	}

	f.cm.mu.Lock()
	view = f.cm.stored["topic_creation:940100101"]
	view.ContentLimit = "nsfw"
	f.cm.list = []communityclient.SiteActivityView{view}
	f.cm.mu.Unlock()
	f.pusher.Reconcile(context.Background())
	if _, ok := f.queueRow(t, "TOPIC_CREATION", apTopicA); !ok {
		t.Fatal("content_limit drift was not enqueued")
	}

	n0 := len(f.writes())
	f.cm.mu.Lock()
	f.cm.list = []communityclient.SiteActivityView{{
		ID: 99, Key: "topic_creation:940199999", ActorID: 42, Verb: "publish",
		ObjectKind: "topic", Removed: false,
	}}
	f.cm.mu.Unlock()
	f.resetQueue(t)
	f.pusher.Reconcile(context.Background())
	writes := f.writes()
	if len(writes) <= n0 {
		t.Fatal("orphan was not tombstoned")
	}
	last := writes[len(writes)-1].Items
	found := false
	for _, it := range last {
		if it.Key == "topic_creation:940199999" && it.Removed && it.ActorID == 42 {
			found = true
		}
	}
	if !found {
		t.Fatalf("tombstone missing: %+v", last)
	}
}

func TestActivityPushOneBadRowDoesNotHoldQueue(t *testing.T) {
	f := newPushFix(t)
	f.topic(t, apTopicA, w3UserOther, false, time.Now().UTC().Add(-time.Hour), "Good")
	now := time.Now().UTC()
	if err := f.db.Exec(`INSERT INTO galgame (id, created, updated, published) VALUES (?, ?, ?, true) ON CONFLICT (id) DO NOTHING`,
		940100888, now, now).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Exec(`INSERT INTO galgame_resource (id, work_id, user_id, type, created, updated) VALUES (?, ?, ?, 'no-such-type', ?, ?)`,
		apBadRes, 940100888, w3UserOther, now, now).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Exec(`INSERT INTO feed_activity (type, source_id, user_id, work_id, content, link, is_nsfw, created)
		VALUES ('GALGAME_RESOURCE_CREATION', ?, ?, 940100888, '', '/galgame/940100888', false, ?)
		ON CONFLICT (type, source_id) DO NOTHING`, apBadRes, w3UserOther, now).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := f.pusher.RunOnce(context.Background()); err != nil {
		t.Fatalf("one bad row failed the batch: %v", err)
	}
	items := f.lastItems(t)
	if len(items) != 1 || items[0].Key != "topic_creation:940100101" {
		t.Fatalf("items = %+v", items)
	}
	if _, ok := f.queueRow(t, "TOPIC_CREATION", apTopicA); ok {
		t.Fatal("the good row stayed queued")
	}
	if _, ok := f.queueRow(t, "GALGAME_RESOURCE_CREATION", apBadRes); !ok {
		t.Fatal("the bad row left the queue without being pushed")
	}
}

func TestActivityPushShortResultsKeepQueue(t *testing.T) {
	f := newPushFix(t)
	f.topic(t, apTopicA, w3UserOther, false, time.Now().UTC().Add(-time.Hour), "A")
	f.topic(t, apTopicB, w3UserOther, false, time.Now().UTC().Add(-time.Hour), "B")
	f.cm.mu.Lock()
	f.cm.short = true
	f.cm.mu.Unlock()
	if _, err := f.pusher.RunOnce(context.Background()); err == nil {
		t.Fatal("a short results list was accepted")
	}
	if f.queueCount(t) != 2 {
		t.Fatalf("queue = %d, want both rows kept", f.queueCount(t))
	}
}

func TestActivityPushReconcileTombstonesHiddenRow(t *testing.T) {
	f := newPushFix(t)
	f.topic(t, apTopicA, w3UserBanned, false, time.Date(2026, 9, 20, 8, 0, 0, 0, time.UTC), "By a banned author")
	f.resetQueue(t)
	f.cm.mu.Lock()
	f.cm.list = []communityclient.SiteActivityView{{
		ID: 7, Key: "topic_creation:940100101", ActorID: int64(w3UserBanned), Verb: "publish",
		ObjectKind: "topic", Removed: false,
	}}
	f.cm.mu.Unlock()
	f.pusher.Reconcile(context.Background())
	var tomb *communityclient.ActivityWriteItem
	for _, w := range f.writes() {
		for i, it := range w.Items {
			if it.Key == "topic_creation:940100101" {
				tomb = &w.Items[i]
			}
		}
	}
	if tomb == nil || !tomb.Removed || tomb.ActorID != int64(w3UserBanned) {
		t.Fatalf("hidden row not tombstoned: %+v", tomb)
	}
	if _, ok := f.queueRow(t, "TOPIC_CREATION", apTopicA); ok {
		t.Fatal("hidden row was enqueued instead of tombstoned")
	}
}
