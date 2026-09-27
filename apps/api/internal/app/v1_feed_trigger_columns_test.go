package app

import (
	"testing"
	"time"
)

const (
	ftWork   = 940100360
	ftRating = 940100361
	ftReply  = 940100362
	ftMsg    = 940100363
)

func TestFeedTriggersIgnoreColumnsNothingReads(t *testing.T) {
	f := newPushFix(t)
	at := time.Now().UTC().Add(-48 * time.Hour).Truncate(time.Second)
	t.Cleanup(func() {
		_ = f.db.Exec(`DELETE FROM message WHERE id = ?`, ftMsg).Error
		_ = f.db.Exec(`DELETE FROM topic_reply WHERE id = ?`, ftReply).Error
		_ = f.db.Exec(`DELETE FROM galgame_rating WHERE id = ?`, ftRating).Error
		_ = f.db.Exec(`DELETE FROM galgame WHERE id = ?`, ftWork).Error
	})
	f.topic(t, apTopicA, w3UserOther, false, at, "")
	seed := []string{
		`INSERT INTO galgame (id, published, content_limit, creator_user_id, catalog_rendered, created, updated)
			VALUES (940100360, true, 'sfw', 930000005, true, now(), now())`,
		`INSERT INTO galgame_rating (id, work_id, user_id, overall, recommend, short_summary, spoiler_level, created, updated)
			VALUES (940100361, 940100360, 930000005, 8, 'yes', 'short', 'none', now(), now())`,
		`INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, like_count, created, updated)
			VALUES (940100362, 'reply', 1, 930000005, 940100101, 0, 0, now(), now())`,
		`INSERT INTO message (id, sender_id, receiver_id, type, content, link, status, created, updated)
			VALUES (940100363, 930000005, 930000001, 'upvoted', 'm', '/topic/940100101', 'unread', now(), now())`,
	}
	for _, q := range seed {
		if err := f.db.Exec(q).Error; err != nil {
			t.Fatal(err)
		}
	}

	cases := []struct {
		name   string
		typ    string
		source int
		noise  string
		change string
	}{
		{"topic content", "TOPIC_CREATION", apTopicA,
			`UPDATE topic SET view = view + 1, like_count = 1, reply_count = 1, comment_count = 1, favorite_count = 1,
				upvote_count = 1, view_7d = 1, view_30d = 1, status_update_time = now(), upvote_time = now(),
				best_answer_id = 940100362, pinned_reply_id = 940100362, last_reply_floor = 1, updated = now() WHERE id = 940100101`,
			`UPDATE topic SET content = 'edited body' WHERE id = 940100101`},
		{"topic cover", "TOPIC_CREATION", apTopicA, "",
			`UPDATE topic SET cover_images = 'x' WHERE id = 940100101`},
		{"galgame creator", "GALGAME_CREATION", ftWork,
			`UPDATE galgame SET view = view + 1, like_count = 1, favorite_count = 1, resource_count = 1, comment_count = 1,
				contributor_count = 1, rating_count = 1, view_7d = 1, view_30d = 1, resource_update_time = now(),
				content_limit = 'nsfw', catalog_checked_at = now(), catalog_rendered = false, release_date_synced_at = now(),
				updated = now() WHERE id = 940100360`,
			`UPDATE galgame SET creator_user_id = 930000001 WHERE id = 940100360`},
		{"rating summary", "GALGAME_RATING_CREATION", ftRating,
			`UPDATE galgame_rating SET view = view + 1, like_count = 1, comment_count = 1, overall = 9, updated = now() WHERE id = 940100361`,
			`UPDATE galgame_rating SET short_summary = 'longer' WHERE id = 940100361`},
		{"reply content", "TOPIC_REPLY_CREATION", ftReply,
			`UPDATE topic_reply SET like_count = 1, dislike_count = 1, updated = now() WHERE id = 940100362`,
			`UPDATE topic_reply SET content = 'edited reply' WHERE id = 940100362`},
		{"message link", "MESSAGE_UPVOTE", ftMsg,
			`UPDATE message SET status = 'read', updated = now() WHERE id = 940100363`,
			`UPDATE message SET link = '/topic/940100101?reply=1' WHERE id = 940100363`},
	}
	for _, c := range cases {
		if !f.hasFeedRow(t, c.typ, c.source) {
			t.Fatalf("%s: no feed row to watch", c.name)
		}
		if c.noise != "" {
			f.resetQueue(t)
			res := f.db.Exec(c.noise)
			if res.Error != nil || res.RowsAffected != 1 {
				t.Fatalf("%s: noise err=%v rows=%d", c.name, res.Error, res.RowsAffected)
			}
			if _, queued := f.queueRow(t, c.typ, c.source); queued {
				t.Errorf("%s: a write to columns nothing reads queued the row", c.name)
			}
		}
		f.resetQueue(t)
		res := f.db.Exec(c.change)
		if res.Error != nil || res.RowsAffected != 1 {
			t.Fatalf("%s: change err=%v rows=%d", c.name, res.Error, res.RowsAffected)
		}
		if _, queued := f.queueRow(t, c.typ, c.source); !queued {
			t.Errorf("%s: a write the pusher sends did not queue the row", c.name)
		}
	}
}

func TestFeedTriggersAllNameTheirUpdateColumns(t *testing.T) {
	f := newPushFix(t)
	unscoped := func() []string {
		var names []string
		if err := f.db.Raw(`
			SELECT tgname FROM pg_trigger
			WHERE NOT tgisinternal AND tgname LIKE 'trg_feed_%'
				AND tgtype & 16 <> 0 AND cardinality(tgattr::int2[]) = 0
			ORDER BY tgname`).Scan(&names).Error; err != nil {
			t.Fatal(err)
		}
		return names
	}
	if got := unscoped(); len(got) != 0 {
		t.Fatalf("feed triggers that fire on every UPDATE: %v", got)
	}
	t.Cleanup(func() { _ = f.db.Exec(`DROP TRIGGER IF EXISTS trg_feed_probe_unscoped ON todo`).Error })
	if err := f.db.Exec(`CREATE TRIGGER trg_feed_probe_unscoped AFTER INSERT OR UPDATE ON todo
		FOR EACH ROW EXECUTE FUNCTION feed_sync_todo()`).Error; err != nil {
		t.Fatal(err)
	}
	if got := unscoped(); len(got) != 1 || got[0] != "trg_feed_probe_unscoped" {
		t.Fatalf("probe not found: %v", got)
	}
}

func (f *pushFix) hasFeedRow(t *testing.T, typ string, source int) bool {
	t.Helper()
	var n int
	if err := f.db.Raw(`SELECT count(*) FROM feed_activity WHERE type = ? AND source_id = ?`, typ, source).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n == 1
}
