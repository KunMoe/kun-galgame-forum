package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kun-galgame-api/internal/community/anchor"
	msgRepo "kun-galgame-api/internal/message/repository"
	"kun-galgame-api/internal/testdb"
	"kun-galgame-api/pkg/communityclient"

	"gorm.io/gorm"
)

func TestWritePageKind8Followed(t *testing.T) {
	db := testdb.Open(t)
	ensureMirrorSchema(t, db)
	repo := msgRepo.NewMessageRepository(db)
	const nid int64 = 9_100_000_801
	db.Exec("DELETE FROM message WHERE community_notification_id = ?", nid)
	t.Cleanup(func() { db.Exec("DELETE FROM message WHERE community_notification_id = ?", nid) })

	resolves := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/posts/resolve" {
			resolves++
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"posts": []any{}}})
	}))
	t.Cleanup(srv.Close)
	cli := communityclient.New(communityclient.Config{BaseURL: srv.URL, ClientID: "c", ClientSecret: "s"})
	p := New(cli, repo, anchor.New(nil, nil), nil)

	actor := int64(9)
	note := communityclient.NotificationView{
		ID: nid, UserID: 3, Kind: communityclient.InboxFollowed,
		ActorID: &actor, ActorCount: 2, ItemCount: 2,
		UpdatedAt: "2026-09-25T00:00:00Z", Seq: 10,
	}
	if err := p.writePage(context.Background(), []communityclient.NotificationView{note}); err != nil {
		t.Fatalf("writePage: %v", err)
	}
	if resolves != 0 {
		t.Fatalf("kind 8 resolved posts: %d", resolves)
	}
	var typ, link, status string
	var sender, actorCount, itemCount int
	var thread *int64
	if err := db.Raw(`SELECT type, link, status, sender_id, actor_count, item_count, community_thread_id
		FROM message WHERE community_notification_id = ?`, nid).Row().
		Scan(&typ, &link, &status, &sender, &actorCount, &itemCount, &thread); err != nil {
		t.Fatalf("row: %v", err)
	}
	if typ != "user-followed" || link != "/user/9" || status != "unread" || sender != 9 || actorCount != 2 || itemCount != 2 || thread != nil {
		t.Fatalf("row type=%s link=%s status=%s sender=%d actor=%d item=%d thread=%v",
			typ, link, status, sender, actorCount, itemCount, thread)
	}

	note.Seq = 20
	note.ActorCount = 5
	note.ItemCount = 5
	note.UpdatedAt = "2026-09-25T01:00:00Z"
	if err := p.writePage(context.Background(), []communityclient.NotificationView{note}); err != nil {
		t.Fatalf("fold: %v", err)
	}
	if err := db.Raw(`SELECT actor_count, item_count FROM message WHERE community_notification_id = ?`, nid).Row().
		Scan(&actorCount, &itemCount); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if actorCount != 5 || itemCount != 5 {
		t.Fatalf("folded actor=%d item=%d, want 5 5", actorCount, itemCount)
	}
}

func TestWritePageKind9WritesNothing(t *testing.T) {
	db := testdb.Open(t)
	ensureMirrorSchema(t, db)
	repo := msgRepo.NewMessageRepository(db)
	const nid int64 = 9_100_000_901
	db.Exec("DELETE FROM message WHERE community_notification_id = ?", nid)
	t.Cleanup(func() { db.Exec("DELETE FROM message WHERE community_notification_id = ?", nid) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"posts": []any{}}})
	}))
	t.Cleanup(srv.Close)
	cli := communityclient.New(communityclient.Config{BaseURL: srv.URL, ClientID: "c", ClientSecret: "s"})
	p := New(cli, repo, anchor.New(nil, nil), nil)

	actor := int64(9)
	note := communityclient.NotificationView{
		ID: nid, UserID: 3, Kind: communityclient.InboxFolloweeTopic,
		ActorID: &actor, ActorCount: 1, ItemCount: 1,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339), Seq: 1,
	}
	if err := p.writePage(context.Background(), []communityclient.NotificationView{note}); err != nil {
		t.Fatalf("writePage: %v", err)
	}
	var n int64
	db.Raw(`SELECT COUNT(*) FROM message WHERE community_notification_id = ?`, nid).Scan(&n)
	if n != 0 {
		t.Fatalf("kind 9 wrote %d rows", n)
	}
}

func TestWritePageKind10FolloweeActivity(t *testing.T) {
	db := testdb.Open(t)
	ensureMirrorSchema(t, db)
	repo := msgRepo.NewMessageRepository(db)
	const nid int64 = 9_100_000_101
	db.Exec("DELETE FROM message WHERE community_notification_id = ?", nid)
	t.Cleanup(func() { db.Exec("DELETE FROM message WHERE community_notification_id = ?", nid) })

	resolves := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/posts/resolve" {
			resolves++
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"posts": []any{}}})
	}))
	t.Cleanup(srv.Close)
	cli := communityclient.New(communityclient.Config{BaseURL: srv.URL, ClientID: "c", ClientSecret: "s"})
	p := New(cli, repo, anchor.New(nil, nil), nil)

	actor := int64(9)
	note := communityclient.NotificationView{
		ID: nid, UserID: 3, Kind: communityclient.InboxFolloweeActivity,
		ActorID: &actor, ActorCount: 1, ItemCount: 2,
		UpdatedAt: "2026-09-26T00:00:00Z", Seq: 10,
		Activity: &communityclient.NotificationActivityView{
			Title: "Hello topic", URL: "https://www.kungal.com/topic/12",
		},
	}
	if err := p.writePage(context.Background(), []communityclient.NotificationView{note}); err != nil {
		t.Fatalf("writePage: %v", err)
	}
	if resolves != 0 {
		t.Fatalf("kind 10 resolved posts: %d", resolves)
	}
	var typ, link, status, content string
	var sender, actorCount, itemCount int
	var thread *int64
	if err := db.Raw(`SELECT type, link, status, content, sender_id, actor_count, item_count, community_thread_id
		FROM message WHERE community_notification_id = ?`, nid).Row().
		Scan(&typ, &link, &status, &content, &sender, &actorCount, &itemCount, &thread); err != nil {
		t.Fatalf("row: %v", err)
	}
	if typ != "followee-activity" || link != "/topic/12" || status != "unread" || content != "Hello topic" ||
		sender != 9 || actorCount != 1 || itemCount != 2 || thread != nil {
		t.Fatalf("row type=%s link=%s status=%s content=%s sender=%d actor=%d item=%d thread=%v",
			typ, link, status, content, sender, actorCount, itemCount, thread)
	}

	note.ItemCount = 0
	note.Seq = 20
	note.Activity = nil
	if err := p.writePage(context.Background(), []communityclient.NotificationView{note}); err != nil {
		t.Fatalf("retract: %v", err)
	}
	var n int64
	db.Raw(`SELECT COUNT(*) FROM message WHERE community_notification_id = ?`, nid).Scan(&n)
	if n != 0 {
		t.Fatalf("item_count 0 left %d rows", n)
	}

	note.ItemCount = 1
	note.Seq = 30
	note.Activity = &communityclient.NotificationActivityView{
		Title: "Again", URL: "https://www.kungal.com/topic/12",
	}
	if err := p.writePage(context.Background(), []communityclient.NotificationView{note}); err != nil {
		t.Fatalf("reinsert: %v", err)
	}
	db.Raw(`SELECT COUNT(*) FROM message WHERE community_notification_id = ?`, nid).Scan(&n)
	if n != 1 {
		t.Fatalf("higher seq did not re-insert, count=%d", n)
	}
}

func TestWritePageKind10OlderRetractDoesNotDelete(t *testing.T) {
	db := testdb.Open(t)
	ensureMirrorSchema(t, db)
	repo := msgRepo.NewMessageRepository(db)
	const nid int64 = 9_100_000_102
	db.Exec("DELETE FROM message WHERE community_notification_id = ?", nid)
	t.Cleanup(func() { db.Exec("DELETE FROM message WHERE community_notification_id = ?", nid) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"posts": []any{}}})
	}))
	t.Cleanup(srv.Close)
	cli := communityclient.New(communityclient.Config{BaseURL: srv.URL, ClientID: "c", ClientSecret: "s"})
	p := New(cli, repo, anchor.New(nil, nil), nil)

	actor := int64(9)
	note := communityclient.NotificationView{
		ID: nid, UserID: 3, Kind: communityclient.InboxFolloweeActivity,
		ActorID: &actor, ActorCount: 1, ItemCount: 2,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339), Seq: 20,
		Activity: &communityclient.NotificationActivityView{
			Title: "Live", URL: "https://www.kungal.com/topic/1",
		},
	}
	if err := p.writePage(context.Background(), []communityclient.NotificationView{note}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	older := note
	older.Seq = 10
	older.ItemCount = 0
	older.Activity = nil
	if err := p.writePage(context.Background(), []communityclient.NotificationView{older}); err != nil {
		t.Fatalf("older retract: %v", err)
	}
	var n int64
	db.Raw(`SELECT COUNT(*) FROM message WHERE community_notification_id = ?`, nid).Scan(&n)
	if n != 1 {
		t.Fatalf("older seq deleted the newer row, count=%d", n)
	}
}

func ensureMirrorSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
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
}
