package repository

import (
	"math"
	"testing"
	"time"

	"kun-galgame-api/internal/message/model"
	"kun-galgame-api/internal/testdb"
)

func TestUpsertCommunityMirrorSeqGuard(t *testing.T) {
	db := testdb.Open(t)
	for _, stmt := range []string{
		`ALTER TABLE message ADD COLUMN IF NOT EXISTS community_notification_id bigint NULL`,
		`ALTER TABLE message ADD COLUMN IF NOT EXISTS community_seq bigint NULL`,
		`ALTER TABLE message ADD COLUMN IF NOT EXISTS community_thread_id bigint NULL`,
		`ALTER TABLE message ADD COLUMN IF NOT EXISTS community_post_number integer NULL`,
		`ALTER TABLE message ADD COLUMN IF NOT EXISTS item_count integer NOT NULL DEFAULT 1`,
		`ALTER TABLE message ADD COLUMN IF NOT EXISTS actor_count integer NOT NULL DEFAULT 1`,
		`CREATE UNIQUE INDEX IF NOT EXISTS message_community_notification_id_key ON message (community_notification_id) WHERE community_notification_id IS NOT NULL`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("ensure schema: %v", err)
		}
	}
	repo := NewMessageRepository(db)

	const nid int64 = 9_000_000_101
	db.Exec("DELETE FROM message WHERE community_notification_id = ?", nid)
	t.Cleanup(func() {
		db.Exec("DELETE FROM message WHERE community_notification_id = ?", nid)
	})

	id := nid
	seq10 := int64(10)
	seq5 := int64(5)
	seq10again := int64(10)
	thread := int64(7)
	postN := 3
	created := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)

	first := &model.Message{
		Content: "hello", Link: "/galgame/1", Status: "unread", Type: "followed",
		SenderID: 9, ReceiverID: 3, CreatedAt: created,
		CommunityNotificationID: &id, CommunitySeq: &seq10, CommunityThreadID: &thread,
		CommunityPostNumber: &postN, ItemCount: 2, ActorCount: 1,
	}
	if err := repo.UpsertCommunityMirror(first); err != nil {
		t.Fatalf("insert: %v", err)
	}

	older := *first
	older.Status = "read"
	older.Content = "stale"
	older.CommunitySeq = &seq5
	if err := repo.UpsertCommunityMirror(&older); err != nil {
		t.Fatalf("older seq: %v", err)
	}

	var row model.Message
	if err := db.Where("community_notification_id = ?", nid).First(&row).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if row.Status != "unread" || row.Content != "hello" || row.CommunitySeq == nil || *row.CommunitySeq != 10 {
		t.Fatalf("older seq overwrote the row: %+v", row)
	}

	replay := *first
	replay.Status = "unread"
	replay.CommunitySeq = &seq10again
	if err := repo.UpsertCommunityMirror(&replay); err != nil {
		t.Fatalf("replay: %v", err)
	}
	if err := db.Where("community_notification_id = ?", nid).First(&row).Error; err != nil {
		t.Fatalf("reload after replay: %v", err)
	}
	if row.Status != "unread" || row.CommunitySeq == nil || *row.CommunitySeq != 10 {
		t.Fatalf("replay mutated the row: %+v", row)
	}

	read := *first
	read.Status = "read"
	seq20 := int64(20)
	read.CommunitySeq = &seq20
	if err := repo.UpsertCommunityMirror(&read); err != nil {
		t.Fatalf("newer seq: %v", err)
	}
	if err := db.Where("community_notification_id = ?", nid).First(&row).Error; err != nil {
		t.Fatalf("reload after newer: %v", err)
	}
	if row.Status != "read" {
		t.Fatalf("newer seq did not map read: %+v", row)
	}
}

func TestDeleteCommunityMirrorSeqGuard(t *testing.T) {
	db := testdb.Open(t)
	for _, stmt := range []string{
		`ALTER TABLE message ADD COLUMN IF NOT EXISTS community_notification_id bigint NULL`,
		`ALTER TABLE message ADD COLUMN IF NOT EXISTS community_seq bigint NULL`,
		`ALTER TABLE message ADD COLUMN IF NOT EXISTS community_thread_id bigint NULL`,
		`ALTER TABLE message ADD COLUMN IF NOT EXISTS community_post_number integer NULL`,
		`ALTER TABLE message ADD COLUMN IF NOT EXISTS item_count integer NOT NULL DEFAULT 1`,
		`ALTER TABLE message ADD COLUMN IF NOT EXISTS actor_count integer NOT NULL DEFAULT 1`,
		`CREATE UNIQUE INDEX IF NOT EXISTS message_community_notification_id_key ON message (community_notification_id) WHERE community_notification_id IS NOT NULL`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("ensure schema: %v", err)
		}
	}
	repo := NewMessageRepository(db)

	const nid int64 = 9_000_000_201
	db.Exec("DELETE FROM message WHERE community_notification_id = ?", nid)
	t.Cleanup(func() {
		db.Exec("DELETE FROM message WHERE community_notification_id = ?", nid)
	})

	id, seq20 := nid, int64(20)
	if err := repo.UpsertCommunityMirror(&model.Message{
		Link: "/topic/1", Status: "unread", Type: "followee-activity", SenderID: 9, ReceiverID: 3,
		CreatedAt: time.Now(), CommunityNotificationID: &id, CommunitySeq: &seq20, ItemCount: 2, ActorCount: 1,
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	if err := repo.DeleteCommunityMirror(nid, 10); err != nil {
		t.Fatalf("older seq: %v", err)
	}
	var n int64
	db.Raw(`SELECT COUNT(*) FROM message WHERE community_notification_id = ?`, nid).Scan(&n)
	if n != 1 {
		t.Fatalf("older seq deleted the row, count=%d", n)
	}

	if err := repo.DeleteCommunityMirror(nid, 30); err != nil {
		t.Fatalf("newer seq: %v", err)
	}
	db.Raw(`SELECT COUNT(*) FROM message WHERE community_notification_id = ?`, nid).Scan(&n)
	if n != 0 {
		t.Fatalf("newer seq left %d rows", n)
	}

	seq40 := int64(40)
	if err := repo.UpsertCommunityMirror(&model.Message{
		Link: "/topic/1", Status: "unread", Type: "followee-activity", SenderID: 9, ReceiverID: 3,
		CreatedAt: time.Now(), CommunityNotificationID: &id, CommunitySeq: &seq40, ItemCount: 1, ActorCount: 1,
	}); err != nil {
		t.Fatalf("reinsert: %v", err)
	}
	db.Raw(`SELECT COUNT(*) FROM message WHERE community_notification_id = ?`, nid).Scan(&n)
	if n != 1 {
		t.Fatalf("reinsert count=%d", n)
	}
}

func TestMarkReadUpToReturnsMirroredIDs(t *testing.T) {
	db := testdb.Open(t)
	repo := NewMessageRepository(db)

	const receiver = 900_000_101
	const nid int64 = 9_000_000_102
	db.Exec("DELETE FROM message WHERE receiver_id = ?", receiver)
	t.Cleanup(func() {
		db.Exec("DELETE FROM message WHERE receiver_id = ?", receiver)
	})

	id, seq := nid, int64(1)
	if err := repo.UpsertCommunityMirror(&model.Message{
		Link: "/galgame/1", Status: "unread", Type: "followed", SenderID: 9, ReceiverID: receiver,
		CreatedAt: time.Now(), CommunityNotificationID: &id, CommunitySeq: &seq, ItemCount: 1, ActorCount: 1,
	}); err != nil {
		t.Fatalf("insert mirrored: %v", err)
	}
	if err := db.Create(&model.Message{
		Link: "/topic/1", Status: "unread", Type: "replied", SenderID: 9, ReceiverID: receiver,
	}).Error; err != nil {
		t.Fatalf("insert local: %v", err)
	}

	marked, ids, unreadLeft, err := repo.MarkReadUpTo(receiver, math.MaxInt32, false, nil)
	if err != nil {
		t.Fatalf("MarkReadUpTo: %v", err)
	}
	if marked != 2 || unreadLeft != 0 {
		t.Fatalf("marked = %d, unread = %d, want 2 and 0", marked, unreadLeft)
	}
	if len(ids) != 1 || ids[0] != nid {
		t.Fatalf("ids = %v, want [%d]", ids, nid)
	}
	var unread int64
	db.Model(&model.Message{}).Where("receiver_id = ? AND status = 'unread'", receiver).Count(&unread)
	if unread != 0 {
		t.Fatalf("unread = %d, want 0", unread)
	}
}
