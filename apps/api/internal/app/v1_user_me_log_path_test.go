package app

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm"
)

const (
	mpTopic         = 981000201
	mpReply         = 981000301
	mpReplyMissing  = 981000399
	mpUpvote        = 981000501
	mpUpvoteMissing = 981000599
	mpOldWork       = 981000801
	mpNewWork       = 981000901
	mpOldPR         = 981000802
	mpNewPR         = 981000902
	mpUnmapped      = 981000888

	mpBefore = "2026-09-22T00:00:00Z"
	mpAfter  = "2026-09-24T00:00:00Z"
	mpSite   = "test-client"
	mpOther  = "other-app"
)

var mpRenumberAt = time.Date(2026, 9, 23, 9, 49, 44, 0, time.UTC)

func (f *meFix) cleanupMpPaths(t *testing.T) {
	t.Helper()
	run := func(q string, args ...any) {
		t.Helper()
		if err := f.db.Exec(q, args...).Error; err != nil {
			t.Errorf("mp path cleanup: %v\n%s", err, q)
		}
	}
	run(`DELETE FROM feed_activity WHERE source_id BETWEEN ? AND ? OR link LIKE ?`,
		981000200, 981000599, "/topic/981000%")
	run(`DELETE FROM topic_upvote WHERE id BETWEEN ? AND ?`, 981000500, 981000599)
	run(`UPDATE topic SET pinned_reply_id = NULL, best_answer_id = NULL WHERE id = ?`, mpTopic)
	run(`DELETE FROM topic_reply WHERE id BETWEEN ? AND ?`, 981000300, 981000399)
	run(`DELETE FROM topic WHERE id = ?`, mpTopic)
	run(`DELETE FROM galgame_renumber_2026 WHERE old_id BETWEEN ? AND ?`, 981000800, 981000899)
}

func (f *meFix) seedMpTopic(t *testing.T) {
	t.Helper()
	created := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	f.execSQL(t, `INSERT INTO topic (
			id, title, content, view, status, category, status_update_time, created, updated,
			user_id, is_nsfw, access_scope, cover_images, like_count, dislike_count, reply_count, comment_count,
			favorite_count, upvote_count, view_7d, view_30d, hidden_by, last_reply_floor
		) VALUES (?, 'mp-path', 'body', 0, 0, 'galgame', ?, ?, ?, ?, false, 'public', '', 0, 0, 0, 0, 0, 0, 0, 0, '', 0)`,
		mpTopic, created, created, created, w3UserAlice)
}

func TestV1MoemoepointEntryPaths(t *testing.T) {
	f := newMeFix(t)
	t.Cleanup(func() { f.cleanupMpPaths(t) })
	f.cleanupMpPaths(t)
	f.seedMpTopic(t)
	f.execSQL(t, `INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, like_count, created, updated)
		VALUES (?, 'mp-reply', 7, ?, ?, 0, 0, ?, ?)`, mpReply, w3UserAlice, mpTopic, mpRenumberAt, mpRenumberAt)
	f.execSQL(t, `INSERT INTO topic_upvote (id, topic_id, user_id, description, created, updated)
		VALUES (?, ?, ?, '', ?, ?)`, mpUpvote, mpTopic, w3UserAlice, mpRenumberAt, mpRenumberAt)
	f.execSQL(t, `INSERT INTO galgame_renumber_2026 (old_id, new_id, how, created) VALUES (?, ?, 'curated', ?), (?, ?, 'curated', ?)`,
		mpOldWork, mpNewWork, mpRenumberAt, mpOldPR, mpNewPR, mpRenumberAt)

	f.setMoeLog([]map[string]any{
		moeLogItem(1, fmt.Sprintf("topic:%d", mpTopic), mpSite, mpBefore),
		moeLogItem(2, fmt.Sprintf("topic_reply:%d", mpReply), mpSite, mpBefore),
		moeLogItem(3, fmt.Sprintf("topic_reply:%d", mpReplyMissing), mpSite, mpBefore),
		moeLogItem(4, fmt.Sprintf("topic_upvote:%d", mpUpvote), mpSite, mpBefore),
		moeLogItem(5, fmt.Sprintf("topic_upvote:%d", mpUpvoteMissing), mpSite, mpBefore),
		moeLogItem(6, fmt.Sprintf("galgame:%d", mpOldWork), mpSite, mpBefore),
		moeLogItem(7, fmt.Sprintf("galgame:%d", mpOldWork), mpSite, mpAfter),
		moeLogItem(8, fmt.Sprintf("galgame:%d", mpUnmapped), mpSite, mpBefore),
		moeLogItem(9, fmt.Sprintf("galgame_pr:%d", mpOldPR), mpSite, mpBefore),
		moeLogItem(10, "galgame_quiz:42", mpSite, mpBefore),
		moeLogItem(11, "toolset:99", mpSite, mpBefore),
		moeLogItem(12, "galgame_resource:1", mpSite, mpBefore),
		moeLogItem(13, "topic:abc", mpSite, mpBefore),
		moeLogItem(14, "topic:0", mpSite, mpBefore),
		moeLogItem(15, "", mpSite, mpBefore),
		moeLogItem(16, fmt.Sprintf("topic:%d", mpTopic), mpOther, mpBefore),
		moeLogItem(17, "topic:01", mpSite, mpBefore),
	})

	resp, body := f.call(t, http.MethodGet, mePath+"/moemoepoint-entries?limit=50", "/me/moemoepoint-entries", "sess-alice", "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	items, _ := body["items"].([]any)
	got := map[string]any{}
	for _, it := range items {
		m, _ := it.(map[string]any)
		got[strID(m["id"])] = m["ref_path"]
	}
	want := map[string]any{
		"1":  "/topic/" + strconv.Itoa(mpTopic),
		"2":  "/topic/" + strconv.Itoa(mpTopic) + "?reply=7",
		"3":  nil,
		"4":  "/topic/" + strconv.Itoa(mpTopic),
		"5":  nil,
		"6":  "/galgame/" + strconv.Itoa(mpNewWork),
		"7":  "/galgame/" + strconv.Itoa(mpOldWork),
		"8":  "/galgame/" + strconv.Itoa(mpUnmapped),
		"9":  "/galgame/" + strconv.Itoa(mpNewPR),
		"10": "/galgame-quiz/42",
		"11": "/toolset/99",
		"12": nil,
		"13": nil,
		"14": nil,
		"15": nil,
		"16": nil,
		"17": nil,
	}
	for id, path := range want {
		if got[id] != path {
			t.Errorf("entry %s ref_path %v, want %v", id, got[id], path)
		}
	}
}

func TestV1MoemoepointEntryPathLookupsAreBatched(t *testing.T) {
	f := newMeFix(t)
	t.Cleanup(func() { f.cleanupMpPaths(t) })
	f.cleanupMpPaths(t)
	f.seedMpTopic(t)

	const n = 16
	created := mpRenumberAt
	items := make([]map[string]any, 0, n*3)
	id := 1
	for i := 0; i < n; i++ {
		replyID := 981000310 + i
		f.execSQL(t, `INSERT INTO topic_reply (id, content, floor, user_id, topic_id, status, like_count, created, updated)
			VALUES (?, 'batch', ?, ?, ?, 0, 0, ?, ?)`, replyID, 10+i, w3UserAlice, mpTopic, created, created)
		items = append(items, moeLogItem(id, fmt.Sprintf("topic_reply:%d", replyID), mpSite, mpBefore))
		id++
	}
	for i := 0; i < n; i++ {
		upvoteID := 981000510 + i
		f.execSQL(t, `INSERT INTO topic_upvote (id, topic_id, user_id, description, created, updated)
			VALUES (?, ?, ?, '', ?, ?)`, upvoteID, mpTopic, w3UserAlice, created, created)
		items = append(items, moeLogItem(id, fmt.Sprintf("topic_upvote:%d", upvoteID), mpSite, mpBefore))
		id++
	}
	for i := 0; i < n; i++ {
		oldID := 981000810 + i
		f.execSQL(t, `INSERT INTO galgame_renumber_2026 (old_id, new_id, how, created) VALUES (?, ?, 'curated', ?)`,
			oldID, 981000910+i, mpRenumberAt)
		items = append(items, moeLogItem(id, fmt.Sprintf("galgame:%d", oldID), mpSite, mpBefore))
		id++
	}
	f.setMoeLog(items)

	var nSQL atomic.Int32
	name := "count-mp-path-sql-" + t.Name()
	count := func(tx *gorm.DB) {
		sql := tx.Statement.SQL.String()
		if strings.Contains(sql, "topic_reply") || strings.Contains(sql, "topic_upvote") || strings.Contains(sql, "galgame_renumber_2026") {
			nSQL.Add(1)
		}
	}
	if err := f.db.Callback().Query().After("gorm:query").Register(name, count); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.db.Callback().Query().Remove(name) })
	if err := f.db.Callback().Row().After("gorm:row").Register(name, count); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.db.Callback().Row().Remove(name) })

	resp, body := f.call(t, http.MethodGet, mePath+"/moemoepoint-entries?limit=50", "/me/moemoepoint-entries", "sess-alice", "", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d %+v", resp.StatusCode, body)
	}
	got := itemIDs(t, body)
	if len(got) != n*3 {
		t.Fatalf("got %d items, want %d", len(got), n*3)
	}
	if nSQL.Load() != 3 {
		t.Fatalf("lookups issued %d SQL statements, want 3 (one per table, independent of %d entries)", nSQL.Load(), n*3)
	}
}
