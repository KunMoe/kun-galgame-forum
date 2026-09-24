package repository

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"kun-galgame-api/internal/testdb"

	"gorm.io/gorm"
)

const (
	pwTarget = 930400001
	pwOther  = 930400002
	pwThird  = 930400003
	pwAdmin  = 930400009

	pwGalgame  = 930400101
	pwTopicT   = 930400201
	pwTopicO   = 930400202
	pwReplyTO  = 930400301
	pwReplyOT  = 930400302
	pwReplyTT  = 930400303
	pwReplyOO  = 930400304
	pwCommT    = 930400401
	pwCommO    = 930400402
	pwWebsiteT = 930400501
	pwWebsiteO = 930400502
	pwToolsetT = 930400601
	pwToolsetO = 930400602
	pwTodoT    = 930400701
	pwTodoO    = 930400702
	pwLottT    = 930400801
	pwLottO    = 930400802
)

type purgeWorld struct {
	db   *gorm.DB
	seed *seeder
	repo *PurgeRepository
}

func newPurgeWorld(t *testing.T) *purgeWorld {
	t.Helper()
	db := testdb.Open(t)
	clean := func() {
		_ = db.Exec(`DELETE FROM user_purge_archive WHERE target_user_id BETWEEN 930400000 AND 930400999`).Error
		_ = db.Exec(`DELETE FROM feed_activity WHERE user_id BETWEEN 930400000 AND 930400999`).Error
	}
	clean()
	t.Cleanup(clean)
	w := &purgeWorld{db: db, seed: newSeeder(t, db), repo: NewPurgeRepository(db)}
	w.populate(t)
	return w
}

// populate gives the target a footprint in every table the purge deletes
// from, and gives other users rows that only the purge's cascades, SET NULL
// keys, recounts, hand-over and release reach.
func (w *purgeWorld) populate(t *testing.T) {
	t.Helper()
	s := w.seed
	at := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	for _, uid := range []int{pwTarget, pwOther, pwThird} {
		s.insert("kungal_user_state", map[string]any{"user_id": uid, "moemoepoint": 50, "created": at, "updated": at})
	}

	s.insert("galgame", map[string]any{"id": pwGalgame, "created": at, "updated": at, "published": true})

	topic := func(id, user int) {
		s.insert("topic", map[string]any{
			"id": id, "title": fmt.Sprintf("pw topic %d", id), "content": "body", "status": 0,
			"access_scope": "public", "user_id": user, "created": at, "updated": at, "status_update_time": at,
			"category": "galgame", "hidden_by": "", "cover_images": "",
		})
	}
	topic(pwTopicT, pwTarget)
	topic(pwTopicO, pwOther)
	s.insert("topic_section", map[string]any{"id": 930400901, "name": "pw-section", "created": at, "updated": at})
	s.insert("topic_section_relation", map[string]any{"topic_id": pwTopicT, "topic_section_id": 930400901, "created": at, "updated": at})
	s.insert("topic_access_grant", map[string]any{"topic_id": pwTopicT, "subject_type": "user", "subject_value": fmt.Sprint(pwOther)})

	reply := func(id, user, topicID, floor int) {
		s.insert("topic_reply", map[string]any{
			"id": id, "content": "reply", "floor": floor, "user_id": user, "topic_id": topicID, "status": 0,
			"created": at, "updated": at,
		})
	}
	reply(pwReplyTO, pwTarget, pwTopicO, 1)
	reply(pwReplyOT, pwOther, pwTopicT, 1)
	reply(pwReplyTT, pwTarget, pwTopicT, 2)
	reply(pwReplyOO, pwOther, pwTopicO, 2)
	s.run(`UPDATE topic SET reply_count = 2, best_answer_id = ?, pinned_reply_id = ? WHERE id = ?`, pwReplyTO, pwReplyTO, pwTopicO)
	s.run(`UPDATE topic SET reply_count = 2 WHERE id = ?`, pwTopicT)

	comment := func(id, user, replyID, topicID int, parent any) {
		s.insert("topic_comment", map[string]any{
			"id": id, "content": "comment", "user_id": user, "topic_reply_id": replyID, "topic_id": topicID,
			"target_user_id": pwOther, "parent_comment_id": parent, "status": 0, "created": at, "updated": at,
		})
	}
	comment(pwCommT, pwTarget, pwReplyOO, pwTopicO, nil)
	comment(pwCommO, pwOther, pwReplyOO, pwTopicO, pwCommT)
	s.run(`UPDATE topic SET comment_count = 2 WHERE id = ?`, pwTopicO)
	s.insert("topic_comment_like", map[string]any{"topic_comment_id": pwCommT, "user_id": pwOther, "created": at, "updated": at})
	s.insert("topic_comment_like", map[string]any{"topic_comment_id": pwCommO, "user_id": pwTarget, "created": at, "updated": at})

	s.insert("topic_reaction", map[string]any{"topic_id": pwTopicO, "user_id": pwTarget, "reaction": "like", "created": at})
	s.insert("topic_reaction", map[string]any{"topic_id": pwTopicO, "user_id": pwThird, "reaction": "like", "created": at})
	s.insert("topic_reaction", map[string]any{"topic_id": pwTopicT, "user_id": pwOther, "reaction": "like", "created": at})
	s.run(`UPDATE topic SET like_count = 2 WHERE id = ?`, pwTopicO)
	s.insert("topic_reply_reaction", map[string]any{"topic_reply_id": pwReplyTO, "user_id": pwOther, "reaction": "like", "created": at})
	s.insert("topic_reply_reaction", map[string]any{"topic_reply_id": pwReplyOO, "user_id": pwTarget, "reaction": "like", "created": at})
	s.run(`UPDATE topic_reply SET like_count = 1 WHERE id IN (?, ?)`, pwReplyTO, pwReplyOO)
	s.insert("topic_favorite", map[string]any{"topic_id": pwTopicO, "user_id": pwTarget, "created": at, "updated": at})
	s.insert("topic_favorite", map[string]any{"topic_id": pwTopicO, "user_id": pwThird, "created": at, "updated": at})
	s.run(`UPDATE topic SET favorite_count = 2 WHERE id = ?`, pwTopicO)
	s.insert("topic_upvote", map[string]any{"topic_id": pwTopicO, "user_id": pwTarget, "created": at, "updated": at})
	s.run(`UPDATE topic SET upvote_count = 1 WHERE id = ?`, pwTopicO)
	s.insert("topic_like", map[string]any{"topic_id": pwTopicO, "user_id": pwTarget, "created": at, "updated": at})
	s.insert("topic_dislike", map[string]any{"topic_id": pwTopicO, "user_id": pwTarget, "created": at, "updated": at})
	s.insert("topic_reply_like", map[string]any{"topic_reply_id": pwReplyOO, "user_id": pwTarget, "created": at, "updated": at})
	s.insert("topic_reply_dislike", map[string]any{"topic_reply_id": pwReplyOO, "user_id": pwTarget, "created": at, "updated": at})

	pollT := s.insert("topic_poll", map[string]any{"topic_id": pwTopicT, "user_id": pwTarget, "title": "pw poll", "created": at, "updated": at})
	optT := s.insert("topic_poll_option", map[string]any{"poll_id": pollT["id"], "text": "a", "vote_count": 1, "created": at, "updated": at})
	s.insert("topic_poll_vote", map[string]any{"poll_id": pollT["id"], "option_id": optT["id"], "user_id": pwOther, "created": at, "updated": at})
	pollO := s.insert("topic_poll", map[string]any{"topic_id": pwTopicO, "user_id": pwOther, "title": "pw poll o", "created": at, "updated": at})
	optO := s.insert("topic_poll_option", map[string]any{"poll_id": pollO["id"], "text": "b", "vote_count": 2, "created": at, "updated": at})
	s.insert("topic_poll_vote", map[string]any{"poll_id": pollO["id"], "option_id": optO["id"], "user_id": pwTarget, "created": at, "updated": at})
	s.insert("topic_poll_vote", map[string]any{"poll_id": pollO["id"], "option_id": optO["id"], "user_id": pwThird, "created": at, "updated": at})

	s.insert("topic_lottery", map[string]any{"id": pwLottT, "topic_id": pwTopicT, "user_id": pwTarget, "status": "open",
		"point_escrow": 30, "entry_count": 1, "created": at, "updated": at})
	prize := s.insert("topic_lottery_prize", map[string]any{"lottery_id": pwLottT, "created": at, "updated": at})
	s.insert("topic_lottery_code", map[string]any{"lottery_id": pwLottT, "prize_id": prize["id"], "secret": "k", "created": at})
	s.insert("topic_lottery_entry", map[string]any{"lottery_id": pwLottT, "user_id": pwOther, "created": at, "updated": at})
	s.insert("topic_lottery", map[string]any{"id": pwLottO, "topic_id": pwTopicO, "user_id": pwOther, "status": "open",
		"entry_count": 2, "created": at, "updated": at})
	s.insert("topic_lottery_entry", map[string]any{"lottery_id": pwLottO, "user_id": pwTarget, "created": at, "updated": at})
	s.insert("topic_lottery_entry", map[string]any{"lottery_id": pwLottO, "user_id": pwThird, "created": at, "updated": at})
	s.insert("topic_draft", map[string]any{"user_id": pwTarget, "created": at, "updated": at})

	s.insert("galgame_like", map[string]any{"work_id": pwGalgame, "user_id": pwTarget, "created": at, "updated": at})
	s.insert("galgame_like", map[string]any{"work_id": pwGalgame, "user_id": pwThird, "created": at, "updated": at})
	s.insert("galgame_favorite", map[string]any{"work_id": pwGalgame, "user_id": pwTarget, "created": at, "updated": at})
	ratingT := s.insert("galgame_rating", map[string]any{"work_id": pwGalgame, "user_id": pwTarget, "like_count": 1, "created": at, "updated": at})
	s.insert("galgame_rating_like", map[string]any{"galgame_rating_id": ratingT["id"], "user_id": pwOther, "created": at, "updated": at})
	ratingO := s.insert("galgame_rating", map[string]any{"work_id": pwGalgame, "user_id": pwOther, "like_count": 1, "created": at, "updated": at})
	s.insert("galgame_rating_like", map[string]any{"galgame_rating_id": ratingO["id"], "user_id": pwTarget, "created": at, "updated": at})
	resT := s.insert("galgame_resource", map[string]any{"work_id": pwGalgame, "user_id": pwTarget, "like_count": 1, "created": at, "updated": at})
	s.insert("galgame_resource_link", map[string]any{"galgame_resource_id": resT["id"], "url": "https://pw.example/t", "created": at, "updated": at})
	s.insert("galgame_resource_provider", map[string]any{"resource_id": resT["id"], "name": "pw", "created": at, "updated": at})
	s.insert("galgame_resource_like", map[string]any{"galgame_resource_id": resT["id"], "user_id": pwOther, "created": at, "updated": at})
	resO := s.insert("galgame_resource", map[string]any{"work_id": pwGalgame, "user_id": pwOther, "like_count": 1, "created": at, "updated": at})
	s.insert("galgame_resource_like", map[string]any{"galgame_resource_id": resO["id"], "user_id": pwTarget, "created": at, "updated": at})
	s.run(`UPDATE galgame SET like_count = 2, rating_count = 2, resource_count = 2, favorite_count = 1 WHERE id = ?`, pwGalgame)

	cat := s.insert("galgame_website_category", map[string]any{"name": "pw-category", "created": at, "updated": at})
	website := func(id, user int, likes int) {
		s.insert("galgame_website", map[string]any{"id": id, "name": fmt.Sprintf("pw site %d", id),
			"url": fmt.Sprintf("pw-%d.example", id), "category_id": cat["id"], "user_id": user, "like_count": likes,
			"created": at, "updated": at})
	}
	website(pwWebsiteT, pwTarget, 1)
	website(pwWebsiteO, pwOther, 0)
	s.insert("galgame_website_like", map[string]any{"website_id": pwWebsiteT, "user_id": pwOther, "created": at, "updated": at})
	s.insert("galgame_website_favorite", map[string]any{"website_id": pwWebsiteO, "user_id": pwTarget, "created": at, "updated": at})
	s.run(`UPDATE galgame_website SET favorite_count = 1 WHERE id = ?`, pwWebsiteO)

	toolset := func(id, user int) {
		s.insert("galgame_toolset", map[string]any{"id": id, "name": fmt.Sprintf("pw tool %d", id), "user_id": user, "created": at, "updated": at})
	}
	toolset(pwToolsetT, pwTarget)
	toolset(pwToolsetO, pwOther)
	s.insert("galgame_toolset_alias", map[string]any{"toolset_id": pwToolsetT, "name": "pw-alias", "created": at, "updated": at})
	s.insert("galgame_toolset_resource", map[string]any{"toolset_id": pwToolsetO, "user_id": pwTarget, "content": "", "created": at, "updated": at})
	s.insert("galgame_toolset_contributor", map[string]any{"toolset_id": pwToolsetO, "user_id": pwTarget, "created": at, "updated": at})
	s.insert("galgame_toolset_practicality", map[string]any{"toolset_id": pwToolsetO, "user_id": pwTarget, "rate": 3, "created": at, "updated": at})
	s.insert("toolset_upload", map[string]any{"artifact_uuid": "pw-upload-1", "toolset_id": pwToolsetO, "user_id": pwTarget, "created": at})

	colT := s.insert("galgame_collection", map[string]any{"user_id": pwTarget, "name": "pw col", "visibility": "public", "item_count": 1, "created": at, "updated": at})
	s.insert("galgame_collection_item", map[string]any{"collection_id": colT["id"], "work_id": pwGalgame, "user_id": pwTarget, "created": at})
	s.insert("galgame_collection_viewer", map[string]any{"collection_id": colT["id"], "user_id": pwOther, "created": at})
	colO := s.insert("galgame_collection", map[string]any{"user_id": pwOther, "name": "pw col o", "visibility": "public", "item_count": 1, "created": at, "updated": at})
	s.insert("galgame_collection_viewer", map[string]any{"collection_id": colO["id"], "user_id": pwTarget, "created": at})

	quiz := func(user int) map[string]any {
		return s.insert("galgame_quiz", map[string]any{"user_id": user, "type": "single", "category": "plot",
			"spoiler_level": "none", "difficulty": 3, "created": at, "updated": at})
	}
	quizT := quiz(pwTarget)
	s.insert("galgame_quiz_galgame", map[string]any{"quiz_id": quizT["id"], "work_id": pwGalgame})
	s.insert("galgame_quiz_answer", map[string]any{"quiz_id": quizT["id"], "user_id": pwOther, "role": "answerer", "quality_rating": 5, "created": at, "updated": at})
	quizO := quiz(pwOther)
	s.insert("galgame_quiz_answer", map[string]any{"quiz_id": quizO["id"], "user_id": pwTarget, "role": "answerer", "is_correct": true, "quality_rating": 7, "created": at, "updated": at})
	s.insert("galgame_quiz_favorite", map[string]any{"quiz_id": quizO["id"], "user_id": pwTarget, "created": at})
	s.run(`UPDATE galgame_quiz SET answer_count = 1, correct_count = 1, quality_sum = 7, quality_count = 1, favorite_count = 1 WHERE id = ?`, quizO["id"])
	s.run(`UPDATE galgame_quiz SET answer_count = 1, quality_sum = 5, quality_count = 1 WHERE id = ?`, quizT["id"])

	s.insert("galgame_post_like", map[string]any{"post_id": 930400991, "user_id": pwTarget, "created": at})
	s.insert("galgame_activity", map[string]any{"work_id": pwGalgame, "user_id": pwTarget, "created": at})
	s.insert("galgame_contributor", map[string]any{"work_id": pwGalgame, "user_id": pwTarget})

	s.insert("todo", map[string]any{"id": pwTodoT, "user_id": pwTarget, "status": 0, "created": at, "updated": at})
	s.insert("todo", map[string]any{"id": pwTodoO, "user_id": pwOther, "status": 1, "claimed_user_id": pwTarget, "created": at, "updated": at})

	room := s.insert("chat_room", map[string]any{"name": fmt.Sprintf("pw-%d-%d", pwTarget, pwOther), "type": "private",
		"last_message_sender_id": pwTarget, "last_message_content": "hi", "created": at, "updated": at})
	s.insert("chat_room_participant", map[string]any{"chat_room_id": room["id"], "user_id": pwTarget, "created": at})
	s.insert("chat_room_participant", map[string]any{"chat_room_id": room["id"], "user_id": pwOther, "created": at})
	fromOther := s.insert("chat_message", map[string]any{"chat_room_id": room["id"], "sender_id": pwOther, "receiver_id": pwTarget, "content": "yo", "created": at, "updated": at})
	fromTarget := s.insert("chat_message", map[string]any{"chat_room_id": room["id"], "sender_id": pwTarget, "receiver_id": pwOther, "content": "hi", "created": at.Add(time.Minute), "updated": at})
	s.insert("chat_message_read_by", map[string]any{"chat_message_id": fromOther["id"], "user_id": pwTarget})
	s.insert("chat_message_reaction", map[string]any{"chat_message_id": fromTarget["id"], "user_id": pwOther, "reaction": "like", "created": at})

	s.insert("message", map[string]any{"sender_id": pwTarget, "receiver_id": pwOther, "type": "liked", "content": "m", "created": at, "updated": at})
	s.insert("message", map[string]any{"sender_id": pwOther, "receiver_id": pwTarget, "type": "liked", "content": "m", "created": at, "updated": at})
	s.insert("system_message_read_state", map[string]any{"user_id": pwTarget, "last_read_message_id": 0, "updated_at": at})
	s.insert("user_follow", map[string]any{"follower_id": pwTarget, "followed_id": pwOther, "created": at, "updated": at})
	s.insert("user_follow", map[string]any{"follower_id": pwOther, "followed_id": pwTarget, "created": at, "updated": at})
	s.insert("user_friend", map[string]any{"user_id": pwTarget, "friend_id": pwOther, "created": at, "updated": at})
	s.insert("user_permission_override", map[string]any{"user_id": pwTarget, "permission": "topic.view_hidden", "effect": "grant", "updated_by": pwAdmin, "updated_at": at})
}

func (w *purgeWorld) scalar(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	if err := w.db.Raw(q, args...).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func (w *purgeWorld) restore(id any) ([]restoreReport, error) {
	var out []restoreReport
	err := w.db.Raw(`SELECT * FROM user_purge_restore(?)`, id).Scan(&out).Error
	return out, err
}

type restoreReport struct {
	TableName string
	Operation string
	Archived  int64
	Restored  int64
}

func TestPurgeArchiveRoundTrip(t *testing.T) {
	w := newPurgeWorld(t)
	// A feed row the purge's own recount recreates: feed_sync_topic upserts the
	// topic's row on every UPDATE, so an INSERT must be archived and undone too.
	w.seed.run(`DELETE FROM feed_activity WHERE type = 'TOPIC_CREATION' AND source_id = ?`, pwTopicO)
	before := snapshot(t, w.db)

	receipt, err := w.repo.PurgeUserContent(pwTarget, pwAdmin)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range UserColumns {
		if c.Handling != HandlingDelete && c.Handling != HandlingTransfer {
			continue
		}
		if n := w.scalar(t, fmt.Sprintf(`SELECT count(*) FROM %q WHERE %q = ?`, c.Table, c.Column), pwTarget); n != 0 {
			t.Errorf("after the purge %s.%s still has %d rows of the target", c.Table, c.Column, n)
		}
	}
	for _, q := range []string{
		`SELECT count(*) FROM topic WHERE id = 930400202 AND best_answer_id IS NULL AND pinned_reply_id IS NULL`,
		`SELECT count(*) FROM topic_comment WHERE id = 930400402 AND parent_comment_id IS NULL`,
		`SELECT count(*) FROM todo WHERE id = 930400702 AND status = 0 AND claimed_user_id IS NULL`,
		`SELECT count(*) FROM galgame_website WHERE id = 930400501 AND user_id <> 930400001`,
		`SELECT count(*) FROM feed_activity WHERE type = 'TOPIC_CREATION' AND source_id = 930400202`,
	} {
		if w.scalar(t, q) != 1 {
			t.Errorf("the purge did not rewrite what the seed expects: %s", q)
		}
	}

	after := snapshot(t, w.db)
	gone, added := diffSnapshots(before, after)
	var archived []struct {
		TableName string
		RowData   string
		RowPK     string
	}
	if err := w.db.Raw(`SELECT table_name, row_data::text AS row_data, row_pk::text AS row_pk
		FROM user_purge_archive WHERE purge_id = ?`, receipt.PurgeID).Scan(&archived).Error; err != nil {
		t.Fatal(err)
	}
	images, keys := map[string]bool{}, map[string]bool{}
	for _, a := range archived {
		images[a.TableName+" "+a.RowData] = true
		keys[a.TableName+" "+a.RowPK] = true
	}
	for tbl, rows := range gone {
		for _, r := range rows {
			if !images[tbl+" "+r] {
				t.Errorf("a %s row the purge deleted or changed is not in the archive: %s", tbl, r)
			}
		}
	}
	for tbl, rows := range added {
		for _, r := range rows {
			if !keys[tbl+" "+pkOf(t, w.db, tbl, r)] {
				t.Errorf("a %s row the purge wrote is not in the archive: %s", tbl, r)
			}
		}
	}
	for _, tbl := range []string{"topic_reply", "topic_comment", "topic_reply_reaction", "topic_poll_vote", "topic_lottery_entry", "chat_message", "feed_activity"} {
		if receipt.Archived[tbl] == 0 {
			t.Errorf("the receipt archived nothing from %s", tbl)
		}
	}

	report, err := w.restore(receipt.PurgeID)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range report {
		if r.Archived != r.Restored {
			t.Errorf("restore skipped %d of %d %s %s rows", r.Archived-r.Restored, r.Archived, r.TableName, r.Operation)
		}
	}
	restored := snapshot(t, w.db)
	gone, added = diffSnapshots(before, restored)
	for tbl, rows := range gone {
		t.Errorf("after restore %s is missing %d rows, first %s", tbl, len(rows), rows[0])
	}
	for tbl, rows := range added {
		t.Errorf("after restore %s has %d extra rows, first %s", tbl, len(rows), rows[0])
	}
	if n := w.scalar(t, `SELECT count(*) FROM user_purge_archive WHERE purge_id = ? AND restored_at IS NULL`, receipt.PurgeID); n != 0 {
		t.Errorf("%d archive rows are not marked restored", n)
	}
}

func pkOf(t *testing.T, db *gorm.DB, table, row string) string {
	t.Helper()
	cols := primaryKey(t, db, table)
	var out string
	if err := db.Raw(`SELECT jsonb_object_agg(k, ?::jsonb -> k)::text FROM unnest(?::text[]) k`,
		row, "{"+strings.Join(cols, ",")+"}").Scan(&out).Error; err != nil {
		t.Fatal(err)
	}
	return out
}

func TestPurgeRestoreKeepsLaterCounterChanges(t *testing.T) {
	w := newPurgeWorld(t)
	receipt, err := w.repo.PurgeUserContent(pwTarget, pwAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if n := w.scalar(t, `SELECT favorite_count FROM topic WHERE id = ?`, pwTopicO); n != 1 {
		t.Fatalf("the purge left favorite_count %d, want 1", n)
	}
	w.seed.insert("topic_favorite", map[string]any{"topic_id": pwTopicO, "user_id": pwAdmin, "created": time.Now(), "updated": time.Now()})
	w.seed.run(`UPDATE topic SET favorite_count = favorite_count + 1 WHERE id = ?`, pwTopicO)

	if _, err := w.restore(receipt.PurgeID); err != nil {
		t.Fatal(err)
	}
	if n := w.scalar(t, `SELECT favorite_count FROM topic WHERE id = ?`, pwTopicO); n != 3 {
		t.Errorf("favorite_count after restore %d, want 3: the two before the purge and the one added since", n)
	}
	if n := w.scalar(t, `SELECT count(*) FROM topic_favorite WHERE topic_id = ?`, pwTopicO); n != 3 {
		t.Errorf("topic_favorite rows after restore %d, want 3", n)
	}
}

func TestPurgeRestoreAbortsOnAMissingParent(t *testing.T) {
	w := newPurgeWorld(t)
	receipt, err := w.repo.PurgeUserContent(pwTarget, pwAdmin)
	if err != nil {
		t.Fatal(err)
	}
	w.seed.run(`DELETE FROM topic WHERE id = ?`, pwTopicO)

	if _, err := w.restore(receipt.PurgeID); err == nil || !strings.Contains(err.Error(), "point at a missing") {
		t.Fatalf("restore over a deleted parent: %v, want it to abort", err)
	}
	if n := w.scalar(t, `SELECT count(*) FROM topic_reply WHERE user_id = ?`, pwTarget); n != 0 {
		t.Errorf("an aborted restore left %d of the target's replies", n)
	}
	if n := w.scalar(t, `SELECT count(*) FROM topic WHERE id = ?`, pwTopicT); n != 0 {
		t.Error("an aborted restore brought back the target's topic")
	}
	if n := w.scalar(t, `SELECT count(*) FROM user_purge_archive WHERE purge_id = ? AND restored_at IS NOT NULL`, receipt.PurgeID); n != 0 {
		t.Errorf("an aborted restore marked %d archive rows restored", n)
	}
}

func TestPurgeRestoreRunsOnce(t *testing.T) {
	w := newPurgeWorld(t)
	receipt, err := w.repo.PurgeUserContent(pwTarget, pwAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.restore(receipt.PurgeID); err != nil {
		t.Fatal(err)
	}
	likes := w.scalar(t, `SELECT like_count FROM topic WHERE id = ?`, pwTopicO)
	if _, err := w.restore(receipt.PurgeID); err == nil {
		t.Error("a second restore of the same purge succeeded")
	}
	if n := w.scalar(t, `SELECT like_count FROM topic WHERE id = ?`, pwTopicO); n != likes {
		t.Errorf("a second restore moved like_count %d -> %d", likes, n)
	}
}

func TestPurgeFailureLeavesNothing(t *testing.T) {
	w := newPurgeWorld(t)
	w.seed.run(`CREATE OR REPLACE FUNCTION pw_refuse_message_delete() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'pw: injected failure'; END $$`)
	w.seed.run(`CREATE TRIGGER pw_refuse_message_delete BEFORE DELETE ON message
		FOR EACH ROW WHEN (OLD.sender_id = 930400001 OR OLD.receiver_id = 930400001)
		EXECUTE FUNCTION pw_refuse_message_delete()`)
	t.Cleanup(func() {
		_ = w.db.Exec(`DROP TRIGGER IF EXISTS pw_refuse_message_delete ON message`).Error
		_ = w.db.Exec(`DROP FUNCTION IF EXISTS pw_refuse_message_delete()`).Error
	})

	if _, err := w.repo.PurgeUserContent(pwTarget, pwAdmin); err == nil || !strings.Contains(err.Error(), "injected failure") {
		t.Fatalf("purge with a failing step: %v", err)
	}
	if n := w.scalar(t, `SELECT count(*) FROM topic WHERE id = ?`, pwTopicT); n != 1 {
		t.Error("a failed purge deleted the target's topic")
	}
	if n := w.scalar(t, `SELECT count(*) FROM topic_reply WHERE user_id = ?`, pwTarget); n != 2 {
		t.Errorf("a failed purge left %d of the target's 2 replies", n)
	}
	if n := w.scalar(t, `SELECT count(*) FROM user_purge_archive WHERE target_user_id = ?`, pwTarget); n != 0 {
		t.Errorf("a failed purge left %d archive rows", n)
	}
}

func TestPurgeSettingsEndWithTheTransaction(t *testing.T) {
	w := newPurgeWorld(t)
	stray := w.seed.insert("user_follow", map[string]any{"follower_id": pwThird, "followed_id": pwOther,
		"created": time.Now(), "updated": time.Now()})
	err := w.db.Connection(func(conn *gorm.DB) error {
		if _, err := NewPurgeRepository(conn).PurgeUserContent(pwTarget, pwAdmin); err != nil {
			return err
		}
		return conn.Exec(`DELETE FROM user_follow WHERE id = ?`, stray["id"]).Error
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := w.scalar(t, `SELECT count(*) FROM user_purge_archive
		WHERE table_name = 'user_follow' AND (row_pk ->> 'id')::int = ?`, stray["id"]); n != 0 {
		t.Error("a delete after the purge, on the purge's connection, was archived")
	}
}

func TestPurgeRefusesADrawingLottery(t *testing.T) {
	w := newPurgeWorld(t)
	w.seed.run(`UPDATE topic_lottery SET status = 'drawing' WHERE id = ?`, pwLottT)

	if _, err := w.repo.PurgeUserContent(pwTarget, pwAdmin); !errors.Is(err, ErrLotteryDrawing) {
		t.Fatalf("purge during a draw: %v", err)
	}
	if n := w.scalar(t, `SELECT count(*) FROM topic WHERE user_id = ?`, pwTarget); n != 1 {
		t.Error("the purge deleted the topic whose lottery is being drawn")
	}
	if n := w.scalar(t, `SELECT count(*) FROM user_purge_archive WHERE target_user_id = ?`, pwTarget); n != 0 {
		t.Errorf("a refused purge left %d archive rows", n)
	}
}

func TestPurgeArchiveExpiry(t *testing.T) {
	db := testdb.Open(t)
	clean := func() {
		_ = db.Exec(`DELETE FROM user_purge_archive WHERE target_user_id BETWEEN 930400000 AND 930400999`).Error
	}
	clean()
	t.Cleanup(clean)
	now := time.Now()
	add := func(purge string, age time.Duration, restored bool, n int) {
		t.Helper()
		var restoredAt any
		if restored {
			restoredAt = now.Add(-time.Hour)
		}
		for range n {
			if err := db.Exec(`INSERT INTO user_purge_archive
				(purge_id, target_user_id, operator_id, table_name, operation, row_pk, row_data, created_at, restored_at)
				VALUES (?, 930400001, 930400009, 'topic', 'delete', '{"id": 1}', '{"id": 1}', ?, ?)`,
				purge, now.Add(-age), restoredAt).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	day := 24 * time.Hour
	add("00000000-0000-4000-8000-930400000001", 31*day, false, 5)
	add("00000000-0000-4000-8000-930400000002", 29*day, false, 3)
	add("00000000-0000-4000-8000-930400000003", 10*day, true, 2)

	n, err := NewPurgeRepository(db).ExpireArchive(now.Add(-ArchiveRetention), 2)
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Errorf("expired %d rows, want the 5 of the 31-day-old purge", n)
	}
	var left []string
	if err := db.Raw(`SELECT DISTINCT purge_id::text FROM user_purge_archive WHERE target_user_id = 930400001 ORDER BY 1`).Scan(&left).Error; err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(left, []string{"00000000-0000-4000-8000-930400000002", "00000000-0000-4000-8000-930400000003"}) {
		t.Errorf("purges left %v, want the 29-day-old one and the restored 10-day-old one", left)
	}
}

var userColumn = regexp.MustCompile(`(^|_)(user|sender|receiver|follower|followed|friend|author|operator|owner|actor|voter|winner|uploader|creator)_id$|^(claimed|updated|created|deleted|banned)_by$`)

func TestPurgeCoverage(t *testing.T) {
	db := testdb.Open(t)

	var untriggered []string
	if err := db.Raw(`SELECT c.relname FROM pg_class c
		WHERE c.relnamespace = 'public'::regnamespace AND c.relkind = 'r' AND NOT c.relispartition
		  AND c.relname NOT IN ('user_purge_archive', '_migrations')
		  AND NOT EXISTS (SELECT 1 FROM pg_trigger g WHERE g.tgrelid = c.oid AND g.tgname = 'trg_user_purge_archive')
		ORDER BY 1`).Scan(&untriggered).Error; err != nil {
		t.Fatal(err)
	}
	if len(untriggered) > 0 {
		t.Errorf("tables without the purge-archive trigger: %v; end the migration that creates a table with SELECT user_purge_archive_attach();", untriggered)
	}

	var keyless []string
	if err := db.Raw(`SELECT c.relname FROM pg_class c
		WHERE c.relnamespace = 'public'::regnamespace AND c.relkind = 'r'
		  AND NOT EXISTS (SELECT 1 FROM pg_index i WHERE i.indrelid = c.oid AND i.indisprimary)
		ORDER BY 1`).Scan(&keyless).Error; err != nil {
		t.Fatal(err)
	}
	if len(keyless) > 0 {
		t.Errorf("tables without a primary key, which the archive needs to address a row: %v", keyless)
	}

	var cols []struct {
		TableName  string
		ColumnName string
	}
	if err := db.Raw(`SELECT c.table_name, c.column_name FROM information_schema.columns c
		JOIN information_schema.tables t USING (table_schema, table_name)
		WHERE c.table_schema = 'public' AND t.table_type = 'BASE TABLE'
		  AND c.data_type IN ('integer', 'bigint')
		ORDER BY 1, 2`).Scan(&cols).Error; err != nil {
		t.Fatal(err)
	}
	known := map[string]UserColumn{}
	for _, c := range UserColumns {
		if c.Handling == HandlingKeep && c.Why == "" {
			t.Errorf("%s.%s is kept without a reason", c.Table, c.Column)
		}
		known[c.Table+"."+c.Column] = c
	}
	inSchema := map[string]bool{}
	for _, c := range cols {
		key := c.TableName + "." + c.ColumnName
		inSchema[key] = true
		if userColumn.MatchString(c.ColumnName) {
			if _, ok := known[key]; !ok {
				t.Errorf("%s names a user and is not in UserColumns: decide what a purge does to it", key)
			}
		}
	}
	for key := range known {
		if !inSchema[key] {
			t.Errorf("UserColumns names %s, which the schema does not have", key)
		}
	}
}
