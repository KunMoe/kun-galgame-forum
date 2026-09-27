package app

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestActivityPushWithdrawsWallComments(t *testing.T) {
	f := newPushFix(t)
	const live, neverSent, gone, fresh = 940100370, 940100371, 940100372, 940100373
	at := time.Now().UTC().Add(-time.Hour)
	for _, id := range []int{live, neverSent, gone} {
		if err := f.db.Exec(`INSERT INTO feed_activity (type, source_id, user_id, content, link, created)
			VALUES ('GALGAME_COMMENT_CREATION', ?, ?, 'wall', '/galgame/1', ?)`, id, w3UserOther, at).Error; err != nil {
			t.Fatal(err)
		}
	}
	f.resetQueue(t)
	if err := f.db.Exec(`INSERT INTO activity_push_sent (type, source_id, actor_id, revision, removed) VALUES
		('GALGAME_COMMENT_CREATION', ?, ?, 1, false), ('GALGAME_COMMENT_CREATION', ?, ?, 1, true)`,
		live, w3UserOther, gone, w3UserOther).Error; err != nil {
		t.Fatal(err)
	}
	up, err := os.ReadFile("../../migrations/201_withdraw_wall_comment_activities.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.db.Exec(string(up)).Error; err != nil {
		t.Fatal(err)
	}
	if _, ok := f.queueRow(t, "GALGAME_COMMENT_CREATION", live); !ok || f.queueCount(t) != 1 {
		t.Fatalf("migration queued %d rows, want only the live accepted key", f.queueCount(t))
	}
	if err := f.db.Exec(`INSERT INTO activity_push_queue (type, source_id) VALUES
		('GALGAME_COMMENT_CREATION', ?), ('GALGAME_COMMENT_CREATION', ?)`, neverSent, gone).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := f.pusher.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	w := f.writes()
	if len(w) != 1 || len(w[0].Items) != 1 {
		t.Fatalf("writes = %+v, want one tombstone", w)
	}
	it := w[0].Items[0]
	if it.Key != "galgame_comment_creation:940100370" || !it.Removed || it.ActorID != int64(w3UserOther) {
		t.Fatalf("tombstone = %+v", it)
	}
	if _, _, removed, ok := f.sentRow(t, "GALGAME_COMMENT_CREATION", live); !ok || !removed {
		t.Fatal("withdrawn key not recorded as removed")
	}
	if f.queueCount(t) != 0 {
		t.Fatalf("queue = %d after the drain", f.queueCount(t))
	}

	if err := f.db.Exec(`INSERT INTO feed_activity (type, source_id, user_id, content, link, created)
		VALUES ('GALGAME_COMMENT_CREATION', ?, ?, 'new wall comment', '/galgame/1', now())`, fresh, w3UserOther).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := f.pusher.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(f.writes()) != 1 || f.queueCount(t) != 0 {
		t.Fatalf("a new wall comment was pushed: writes=%d queue=%d", len(f.writes()), f.queueCount(t))
	}
}
