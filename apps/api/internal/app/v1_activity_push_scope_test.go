package app

import (
	"context"
	"testing"
	"time"

	"kun-galgame-api/pkg/communityclient"
)

const apTopicKey = "topic_creation:940100101"

func (f *pushFix) setScope(t *testing.T, id int, scope string) {
	t.Helper()
	if err := f.db.Exec(`UPDATE topic SET access_scope = ? WHERE id = ?`, scope, id).Error; err != nil {
		t.Fatal(err)
	}
}

func (f *pushFix) drain(t *testing.T) {
	t.Helper()
	if _, err := f.pusher.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func (f *pushFix) sentKeys() map[string][]bool {
	out := map[string][]bool{}
	for _, w := range f.writes() {
		for _, it := range w.Items {
			out[it.Key] = append(out[it.Key], it.Removed)
		}
	}
	return out
}

func TestActivityPushNeverSendsLoginTopic(t *testing.T) {
	f := newPushFix(t)
	f.topic(t, apTopicA, w3UserOther, false, time.Now().UTC().Add(-time.Hour), "Members only")
	f.setScope(t, apTopicA, "login")
	f.drain(t)
	if sent := f.sentKeys()[apTopicKey]; len(sent) != 0 {
		t.Fatalf("login topic reached community: %v", sent)
	}
	if _, ok := f.queueRow(t, "TOPIC_CREATION", apTopicA); ok {
		t.Fatal("queue row remained")
	}
	if _, _, _, ok := f.sentRow(t, "TOPIC_CREATION", apTopicA); ok {
		t.Fatal("sent row written for a login topic")
	}
}

func TestActivityPushWithdrawsTopicTurnedLogin(t *testing.T) {
	f := newPushFix(t)
	f.topic(t, apTopicA, w3UserOther, false, time.Now().UTC().Add(-time.Hour), "Public first")
	f.drain(t)
	f.setScope(t, apTopicA, "login")
	f.drain(t)
	f.setScope(t, apTopicA, "public")
	f.drain(t)
	sent := f.sentKeys()[apTopicKey]
	if len(sent) != 3 || sent[0] || !sent[1] || sent[2] {
		t.Fatalf("removed flags per write = %v, want [false true false]", sent)
	}
	if _, _, removed, ok := f.sentRow(t, "TOPIC_CREATION", apTopicA); !ok || removed {
		t.Fatalf("sent row found=%v removed=%v", ok, removed)
	}
}

func TestActivityPushReconcileTombstonesLoginTopic(t *testing.T) {
	f := newPushFix(t)
	f.topic(t, apTopicA, w3UserOther, false, time.Date(2026, 9, 20, 8, 0, 0, 0, time.UTC), "Stored live")
	f.drain(t)
	f.cm.mu.Lock()
	view := f.cm.stored[apTopicKey]
	f.cm.mu.Unlock()
	f.setScope(t, apTopicA, "login")
	f.resetQueue(t)

	f.cm.mu.Lock()
	f.cm.list = []communityclient.SiteActivityView{view}
	f.cm.mu.Unlock()
	f.pusher.Reconcile(context.Background())
	sent := f.sentKeys()[apTopicKey]
	if len(sent) != 1 || !sent[0] {
		t.Fatalf("reconcile writes = %v, want one tombstone", sent)
	}
}
