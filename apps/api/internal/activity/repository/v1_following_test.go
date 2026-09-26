package repository

import (
	"testing"
	"time"
)

func TestEmptyActorIDsDoNotQuery(t *testing.T) {
	r := NewActivityRepository(nil)
	q := FeedQuery{ActorIDs: []int{}, Types: []string{"TOPIC_CREATION"}, Limit: 20}
	rows, err := r.FeedPage(q)
	if err != nil || len(rows) != 0 {
		t.Fatalf("FeedPage rows=%v err=%v; a query would panic on a nil DB", rows, err)
	}
	n, err := r.CountFeedSince(q, time.Time{}, 100)
	if err != nil || n != 0 {
		t.Fatalf("CountFeedSince n=%d err=%v; a query would panic on a nil DB", n, err)
	}
}
