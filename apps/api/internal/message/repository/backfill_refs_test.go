package repository

import (
	"fmt"
	"testing"
	"time"

	"kun-galgame-api/internal/testdb"

	"gorm.io/gorm"
)

const (
	bfUserA = 971000001
	bfUserB = 971000002
	bfUserC = 971000003
	bfUserD = 971000004
	bfUserE = 971000005

	bfTopic       = 971000101
	bfTopicOne    = 971000102
	bfTopicMany   = 971000103
	bfTopicTwice  = 971000104
	bfReplyFloor3 = 971000201
	bfReplyFloor5 = 971000202
	bfLotteryOne  = 971000301
	bfLotteryA    = 971000302
	bfLotteryB    = 971000303
	bfLotteryC    = 971000304
	bfLotteryD    = 971000305
	bfPollOne     = 971000401
	bfPollA       = 971000402
	bfPollB       = 971000403
	bfWork        = 971000501
	bfWorkMoved   = 971000502
	bfResource    = 971000601
	bfResMoved    = 971000602
	bfRating      = 971000701
	bfQuiz        = 971000801
	bfWebsite     = 971001001
	bfCategory    = 971001101

	bfMsgMin = 971100000
	bfMsgMax = 971199999
)

func bfClean(db *gorm.DB) {
	for _, q := range []string{
		`DELETE FROM feed_activity WHERE source_id BETWEEN 971100000 AND 971199999 AND type LIKE 'MESSAGE_%'`,
		`DELETE FROM message WHERE id BETWEEN 971100000 AND 971199999`,
		`DELETE FROM topic_poll_vote WHERE poll_id BETWEEN 971000401 AND 971000499`,
		`DELETE FROM topic_poll_option WHERE poll_id BETWEEN 971000401 AND 971000499`,
		`DELETE FROM topic_poll WHERE id BETWEEN 971000401 AND 971000499`,
		`DELETE FROM topic_lottery_entry WHERE lottery_id BETWEEN 971000301 AND 971000399`,
		`DELETE FROM topic_lottery WHERE id BETWEEN 971000301 AND 971000399`,
		`DELETE FROM topic_reply WHERE topic_id BETWEEN 971000101 AND 971000199`,
		`DELETE FROM topic WHERE id BETWEEN 971000101 AND 971000199`,
		`DELETE FROM galgame_quiz_answer WHERE quiz_id = 971000801`,
		`DELETE FROM galgame_quiz WHERE id = 971000801`,
		`DELETE FROM galgame_rating_like WHERE galgame_rating_id = 971000701`,
		`DELETE FROM galgame_rating WHERE id = 971000701`,
		`DELETE FROM galgame_resource_like WHERE galgame_resource_id BETWEEN 971000601 AND 971000699`,
		`DELETE FROM galgame_resource WHERE id BETWEEN 971000601 AND 971000699`,
		`DELETE FROM galgame_resource_relocation WHERE resource_id BETWEEN 971000601 AND 971000699`,
		`DELETE FROM galgame_like WHERE work_id BETWEEN 971000501 AND 971000599`,
		`DELETE FROM galgame WHERE id BETWEEN 971000501 AND 971000599`,
		`DELETE FROM galgame_website WHERE id = 971001001`,
		`DELETE FROM galgame_website_category WHERE id = 971001101`,
	} {
		_ = db.Exec(q).Error
	}
}

func TestMessageBackfillRefs(t *testing.T) {
	db := testdb.Open(t)
	bfClean(db)
	t.Cleanup(func() { bfClean(db) })

	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	run := func(q string, args ...any) {
		t.Helper()
		if err := db.Exec(q, args...).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, q)
		}
	}
	for _, id := range []int{bfTopic, bfTopicOne, bfTopicMany, bfTopicTwice} {
		run(`INSERT INTO topic (id, title, content, category, user_id, created, updated) VALUES (?, 't', 'b', 'galgame', ?, ?, ?)`,
			id, bfUserA, base, base)
	}
	run(`INSERT INTO topic_reply (id, topic_id, floor, user_id, content, created, updated) VALUES (?, ?, 3, ?, 'r3', ?, ?), (?, ?, 5, ?, 'r5', ?, ?)`,
		bfReplyFloor3, bfTopic, bfUserB, base, base,
		bfReplyFloor5, bfTopic, bfUserC, base.Add(time.Hour), base.Add(time.Hour))
	lottery := func(id, topic int) {
		run(`INSERT INTO topic_lottery (id, topic_id, user_id, title) VALUES (?, ?, ?, 'l')`, id, topic, bfUserA)
	}
	lottery(bfLotteryOne, bfTopicOne)
	lottery(bfLotteryA, bfTopicMany)
	lottery(bfLotteryB, bfTopicMany)
	lottery(bfLotteryC, bfTopicTwice)
	lottery(bfLotteryD, bfTopicTwice)
	for _, l := range []int{bfLotteryB, bfLotteryC, bfLotteryD} {
		run(`INSERT INTO topic_lottery_entry (lottery_id, user_id) VALUES (?, ?)`, l, bfUserB)
	}
	poll := func(id, topic int, voter int) {
		run(`INSERT INTO topic_poll (id, title, topic_id, user_id, updated) VALUES (?, 'p', ?, ?, ?)`, id, topic, bfUserA, base)
		run(`INSERT INTO topic_poll_option (id, text, poll_id, updated) VALUES (?, 'o', ?, ?)`, id, id, base)
		if voter > 0 {
			run(`INSERT INTO topic_poll_vote (poll_id, option_id, user_id, updated) VALUES (?, ?, ?, ?)`, id, id, voter, base)
		}
	}
	poll(bfPollOne, bfTopicOne, 0)
	poll(bfPollA, bfTopicMany, bfUserB)
	poll(bfPollB, bfTopicMany, 0)

	run(`INSERT INTO galgame (id, updated, creator_user_id) VALUES (?, ?, ?), (?, ?, NULL)`, bfWork, base, bfUserA, bfWorkMoved, base)
	run(`INSERT INTO galgame_resource (id, work_id, user_id, status, updated) VALUES (?, ?, ?, 1, ?)`, bfResource, bfWork, bfUserA, base)
	run(`INSERT INTO galgame_resource_relocation (resource_id, work_id, uploader_id, payload) VALUES (?, ?, ?, '{}')`, bfResMoved, bfWorkMoved, bfUserA)
	run(`INSERT INTO galgame_rating (id, recommend, overall, user_id, work_id, updated) VALUES (?, 'yes', 8, ?, ?, ?)`, bfRating, bfUserA, bfWork, base)
	run(`INSERT INTO galgame_resource_like (galgame_resource_id, user_id, updated) VALUES (?, ?, ?)`, bfResource, bfUserB, base)
	run(`INSERT INTO galgame_like (work_id, user_id, updated) VALUES (?, ?, ?)`, bfWork, bfUserC, base)
	run(`INSERT INTO galgame_rating_like (galgame_rating_id, user_id, updated) VALUES (?, ?, ?), (?, ?, ?)`,
		bfRating, bfUserC, base, bfRating, bfUserD, base)
	run(`INSERT INTO galgame_quiz (id, user_id, type, question) VALUES (?, ?, 'single', 'q')`, bfQuiz, bfUserA)
	var answerID int
	if err := db.Raw(`INSERT INTO galgame_quiz_answer (quiz_id, user_id, role) VALUES (?, ?, 'answerer') RETURNING id`, bfQuiz, bfUserB).
		Row().Scan(&answerID); err != nil {
		t.Fatal(err)
	}
	run(`INSERT INTO galgame_website_category (id, name, updated) VALUES (?, 'bf', ?)`, bfCategory, base)
	run(`INSERT INTO galgame_website (id, name, url, create_time, category_id, updated) VALUES (?, 'bf', 'bf.example.org', '2020', ?, ?)`,
		bfWebsite, bfCategory, base)

	type row struct {
		typ, link       string
		sender, created int
		mirror          bool
		want            string
		receiver        int
	}
	rows := []row{
		{"upvoted", fmt.Sprintf("/topic/%d", bfTopic), bfUserB, 0, false, fmt.Sprintf("topic:%d topic:%d", bfTopic, bfTopic), 0},
		{"liked", fmt.Sprintf("/topic/%d?reply=3", bfTopic), bfUserB, 0, false, fmt.Sprintf("reply:%d topic:%d", bfReplyFloor3, bfTopic), 0},
		{"replied", fmt.Sprintf("/topic/%d?reply=99", bfTopic), bfUserB, 0, false, fmt.Sprintf("topic:%d topic:%d", bfTopic, bfTopic), 0},
		{"replied", fmt.Sprintf("/topic/%d", bfTopic), bfUserB, 0, false, fmt.Sprintf("topic:%d topic:%d", bfTopic, bfTopic), 0},
		{"commented", fmt.Sprintf("/topic/%d?comment=777", bfTopic), bfUserB, 0, false, fmt.Sprintf("topic_comment:777 topic:%d", bfTopic), 0},
		{"subscribed-topic", fmt.Sprintf("/topic/%d?reply=3", bfTopic), bfUserC, 61, false, fmt.Sprintf("reply:%d topic:%d", bfReplyFloor5, bfTopic), 0},
		{"lottery-won", fmt.Sprintf("/topic/%d", bfTopicOne), bfUserA, 0, false, fmt.Sprintf("lottery:%d topic:%d", bfLotteryOne, bfTopicOne), 0},
		{"lottery-closed", fmt.Sprintf("/topic/%d", bfTopicMany), bfUserA, 0, false, fmt.Sprintf("lottery:%d topic:%d", bfLotteryB, bfTopicMany), bfUserB},
		{"lottery-expired", fmt.Sprintf("/topic/%d", bfTopicTwice), bfUserA, 0, false, fmt.Sprintf("topic:%d topic:%d", bfTopicTwice, bfTopicTwice), bfUserB},
		{"poll-closed", fmt.Sprintf("/topic/%d", bfTopicOne), bfUserA, 0, false, fmt.Sprintf("poll:%d topic:%d", bfPollOne, bfTopicOne), 0},
		{"poll-closed", fmt.Sprintf("/topic/%d", bfTopicMany), bfUserA, 0, false, fmt.Sprintf("poll:%d topic:%d", bfPollA, bfTopicMany), bfUserB},
		{"liked", fmt.Sprintf("/galgame/%d", bfWork), bfUserB, 0, false, fmt.Sprintf("resource:%d work:%d", bfResource, bfWork), 0},
		{"liked", fmt.Sprintf("/galgame/%d", bfWork), bfUserC, 0, false, fmt.Sprintf("work:%d work:%d", bfWork, bfWork), 0},
		{"liked", fmt.Sprintf("/galgame/%d", bfWork), bfUserD, 0, false, fmt.Sprintf("rating:%d work:%d", bfRating, bfWork), 0},
		{"liked", fmt.Sprintf("/galgame/%d", bfWork), bfUserE, 0, false, fmt.Sprintf("work:%d work:%d", bfWork, bfWork), 0},
		{"expired", fmt.Sprintf("/galgame/%d", bfWork), bfUserB, 0, false, fmt.Sprintf("resource:%d work:%d", bfResource, bfWork), 0},
		{"requested", fmt.Sprintf("/galgame/%d", bfWork), bfUserB, 0, false, fmt.Sprintf("work:%d work:%d", bfWork, bfWork), 0},
		{"followed", fmt.Sprintf("/galgame/%d?comment=55", bfWork), bfUserB, 0, true, fmt.Sprintf("community_post:55 work:%d", bfWork), 0},
		{"commented", fmt.Sprintf("/galgame/resource/%d", bfResource), bfUserB, 0, false, fmt.Sprintf("resource:%d resource:%d", bfResource, bfResource), 0},
		{"resource-relocated", fmt.Sprintf("/galgame/resource/%d", bfResMoved), bfUserB, 0, false, fmt.Sprintf("resource:%d work:%d", bfResMoved, bfWorkMoved), 0},
		{"commented", fmt.Sprintf("/galgame-rating/%d", bfRating), bfUserB, 0, true, fmt.Sprintf("rating:%d rating:%d", bfRating, bfRating), 0},
		{"quiz-answered", fmt.Sprintf("/galgame-quiz/%d", bfQuiz), bfUserB, 0, false, fmt.Sprintf("quiz_answer:%d quiz:%d", answerID, bfQuiz), 0},
		{"quiz-answered", fmt.Sprintf("/galgame-quiz/%d", bfQuiz), bfUserC, 0, false, fmt.Sprintf("quiz:%d quiz:%d", bfQuiz, bfQuiz), 0},
		{"replied", "/toolset/971000901", bfUserB, 0, false, "toolset:971000901 toolset:971000901", 0},
		{"commented", "/website/bf.example.org", bfUserB, 0, false, fmt.Sprintf("website:%d website:%d", bfWebsite, bfWebsite), 0},
		{"commented", "/website/gone.example.org", bfUserB, 0, false, "-:- -:-", 0},
		{"user-followed", fmt.Sprintf("/user/%d", bfUserB), bfUserB, 0, true, fmt.Sprintf("user:%d user:%d", bfUserB, bfUserB), 0},
		{"notice", "/shop", 0, 0, true, "-:- -:-", 0},
		{"admin", "/topic/12abc", bfUserB, 0, false, "-:- -:-", 0},
	}
	for i, r := range rows {
		var community any
		if r.mirror {
			community = int64(bfMsgMin + i)
		}
		receiver := r.receiver
		if receiver == 0 {
			receiver = bfUserA
		}
		created := base.Add(time.Duration(r.created) * time.Minute)
		run(`INSERT INTO message (id, content, link, status, type, sender_id, receiver_id, created, updated, community_notification_id)
			VALUES (?, 'c', ?, 'read', ?, ?, ?, ?, ?, ?)`,
			bfMsgMin+i, r.link, r.typ, r.sender, receiver, created, created, community)
	}
	placedID := bfMsgMin + len(rows)
	run(`INSERT INTO message (id, content, link, status, type, sender_id, receiver_id, created, updated, subject_kind, subject_id, root_kind, root_id)
		VALUES (?, 'c', ?, 'read', 'liked', ?, ?, ?, ?, 'topic_comment', 1, 'topic', 2)`,
		placedID, fmt.Sprintf("/topic/%d", bfTopic), bfUserB, bfUserA, base, base)
	run(`UPDATE feed_activity SET content = 'sentinel' WHERE type = 'MESSAGE_UPVOTE' AND source_id = ?`, bfMsgMin)

	var placed int64
	if err := db.Raw(`SELECT message_backfill_refs()`).Row().Scan(&placed); err != nil {
		t.Fatal(err)
	}

	got := map[int]string{}
	var out []struct {
		ID  int
		Ref string
	}
	if err := db.Raw(`SELECT id, COALESCE(subject_kind, '-') || ':' || COALESCE(subject_id::text, '-')
		|| ' ' || COALESCE(root_kind, '-') || ':' || COALESCE(root_id::text, '-') AS ref
		FROM message WHERE id BETWEEN ? AND ?`, bfMsgMin, bfMsgMax).Scan(&out).Error; err != nil {
		t.Fatal(err)
	}
	for _, o := range out {
		got[o.ID] = o.Ref
	}
	wantPlaced := int64(0)
	for i, r := range rows {
		if got[bfMsgMin+i] != r.want {
			t.Errorf("%s %s from %d: got %q, want %q", r.typ, r.link, r.sender, got[bfMsgMin+i], r.want)
		}
		if r.want != "-:- -:-" {
			wantPlaced++
		}
	}
	if got[placedID] != "topic_comment:1 topic:2" {
		t.Errorf("an already placed row was rewritten: %q", got[placedID])
	}
	if placed < wantPlaced {
		t.Errorf("the function reported %d placed rows, want at least %d", placed, wantPlaced)
	}

	var content string
	if err := db.Raw(`SELECT content FROM feed_activity WHERE type = 'MESSAGE_UPVOTE' AND source_id = ?`, bfMsgMin).
		Row().Scan(&content); err != nil {
		t.Fatal(err)
	}
	if content != "sentinel" {
		t.Errorf("the backfill fired trg_feed_message: feed content %q", content)
	}

	var again int64
	if err := db.Raw(`SELECT message_backfill_refs()`).Row().Scan(&again); err != nil {
		t.Fatal(err)
	}
	var stillNull int64
	if err := db.Raw(`SELECT count(*) FROM message WHERE id BETWEEN ? AND ? AND subject_kind IS NULL`, bfMsgMin, bfMsgMax).
		Row().Scan(&stillNull); err != nil {
		t.Fatal(err)
	}
	if stillNull != 3 {
		t.Errorf("%d rows left unplaced, want the 3 nothing parses", stillNull)
	}
}
