package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"kun-galgame-api/internal/admin/repository"
	"kun-galgame-api/internal/testdb"
	"kun-galgame-api/pkg/communityclient"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	prTarget   = 930470001
	prOperator = 930470009
	prTopic    = 930470101
)

type fakeCommunityRestore struct {
	calls   atomic.Int32
	mu      sync.Mutex
	authors []string
	status  atomic.Int32
}

func (f *fakeCommunityRestore) serve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/purge/restore") {
		http.NotFound(w, r)
		return
	}
	f.calls.Add(1)
	f.mu.Lock()
	f.authors = append(f.authors, strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/authors/"), "/purge/restore"))
	f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch status := int(f.status.Load()); status {
	case 0:
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"posts_restored": 2, "reactions_restored": 1}})
	default:
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 4, "message": "no purge of this author in the last 30 days is left to restore"})
	}
}

func newRestoreFix(t *testing.T) (*gorm.DB, *PurgeService, *fakeCommunityRestore, uuid.UUID) {
	t.Helper()
	db := testdb.Open(t)
	clean := func() {
		for _, q := range []string{
			`DELETE FROM user_purge_archive WHERE target_user_id = 930470001`,
			`DELETE FROM topic WHERE id = 930470101`,
			`DELETE FROM feed_activity WHERE user_id = 930470001`,
			`DELETE FROM kungal_user_state WHERE user_id = 930470001`,
		} {
			if err := db.Exec(q).Error; err != nil {
				t.Errorf("cleanup: %v", err)
			}
		}
	}
	clean()
	t.Cleanup(clean)
	at := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO kungal_user_state (user_id, moemoepoint, created, updated) VALUES (?, 5, ?, ?)`, []any{prTarget, at, at}},
		{`INSERT INTO topic (
			id, title, content, view, status, category, status_update_time, created, updated,
			user_id, is_nsfw, access_scope, cover_images, like_count, dislike_count, reply_count, comment_count,
			favorite_count, upvote_count, view_7d, view_30d, hidden_by, last_reply_floor
		) VALUES (?, 'pr spam', 'body', 0, 0, 'galgame', ?, ?, ?, ?, false, 'public', '', 0, 0, 0, 0, 0, 0, 0, 0, '', 0)`,
			[]any{prTopic, at, at, at, prTarget}},
	} {
		if err := db.Exec(q.sql, q.args...).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	repo := repository.NewPurgeRepository(db)
	receipt, err := repo.PurgeUserContent(prTarget, prOperator)
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakeCommunityRestore{}
	srv := httptest.NewServer(http.HandlerFunc(fake.serve))
	t.Cleanup(srv.Close)
	community := communityclient.New(communityclient.Config{BaseURL: srv.URL, ClientID: "c", ClientSecret: "s"})
	return db, NewPurgeService(repo, nil, community, nil), fake, receipt.PurgeID
}

func topicCount(t *testing.T, db *gorm.DB) int {
	t.Helper()
	var n int
	if err := db.Raw(`SELECT count(*) FROM topic WHERE id = ?`, prTopic).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func TestPurgeRestoreDryRun(t *testing.T) {
	db, svc, fake, purgeID := newRestoreFix(t)
	report, err := svc.Restore(context.Background(), purgeID, false)
	if err != nil {
		t.Fatal(err)
	}
	if report.Local.TargetUserID != prTarget || len(report.Local.Rows) == 0 {
		t.Errorf("dry-run report %+v", report.Local)
	}
	if topicCount(t, db) != 0 {
		t.Error("a dry run brought the topic back")
	}
	var unrestored int
	if err := db.Raw(`SELECT count(*) FROM user_purge_archive WHERE purge_id = ? AND restored_at IS NULL`, purgeID).Scan(&unrestored).Error; err != nil {
		t.Fatal(err)
	}
	if unrestored == 0 {
		t.Error("a dry run marked the archive restored")
	}
	if n := fake.calls.Load(); n != 0 {
		t.Errorf("a dry run called community %d times", n)
	}
}

func TestPurgeRestoreCommit(t *testing.T) {
	db, svc, fake, purgeID := newRestoreFix(t)
	report, err := svc.Restore(context.Background(), purgeID, true)
	if err != nil {
		t.Fatal(err)
	}
	if topicCount(t, db) != 1 {
		t.Error("the committed restore did not bring the topic back")
	}
	if report.Community == nil || report.Community.PostsRestored != 2 {
		t.Errorf("community report %+v", report.Community)
	}
	fake.mu.Lock()
	authors := append([]string(nil), fake.authors...)
	fake.mu.Unlock()
	if len(authors) != 1 || authors[0] != "930470001" {
		t.Errorf("community was asked to restore authors %v, want the purge target", authors)
	}

	fake.status.Store(http.StatusNotFound)
	again, err := svc.Restore(context.Background(), purgeID, true)
	if err != nil {
		t.Fatalf("a second restore after community reports nothing left: %v", err)
	}
	if !again.Local.AlreadyRestored || !again.CommunityNothing || fake.calls.Load() != 2 {
		t.Errorf("second restore %+v, community calls %d", again, fake.calls.Load())
	}
}

func TestPurgeRestoreCommunityFailure(t *testing.T) {
	db, svc, fake, purgeID := newRestoreFix(t)
	fake.status.Store(http.StatusBadGateway)
	if _, err := svc.Restore(context.Background(), purgeID, true); err == nil {
		t.Fatal("a community 502 was reported as success")
	}
	if topicCount(t, db) != 1 {
		t.Error("the forum half was not committed before community failed")
	}
	fake.status.Store(0)
	report, err := svc.Restore(context.Background(), purgeID, true)
	if err != nil || !report.Local.AlreadyRestored || report.Community == nil {
		t.Errorf("rerun after the community failure: %+v %v", report, err)
	}
}

func TestPurgeRestoreUnknownPurge(t *testing.T) {
	_, svc, fake, _ := newRestoreFix(t)
	if _, err := svc.Restore(context.Background(), uuid.New(), true); !errors.Is(err, repository.ErrPurgeNotArchived) {
		t.Errorf("unknown purge id: %v", err)
	}
	if fake.calls.Load() != 0 {
		t.Error("community was called for a purge the archive does not know")
	}
}
