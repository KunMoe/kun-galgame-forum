package app

import (
	"strconv"
	"testing"
	"time"

	msgRepo "kun-galgame-api/internal/message/repository"
	"kun-galgame-api/internal/testdb"
)

func TestExpireReadNotificationsFollowsRetentionDays(t *testing.T) {
	db := testdb.Open(t)
	const receiver = 971300001
	clean := func() { db.Exec(`DELETE FROM message WHERE receiver_id = ?`, receiver) }
	clean()
	t.Cleanup(clean)
	repo := msgRepo.NewMessageRepository(db)

	days := func(n int) func() int { return func() int { return n } }
	now := time.Now()
	for _, age := range []int{40, 20} {
		at := now.AddDate(0, 0, -age)
		if err := db.Exec(`INSERT INTO message (content, link, status, type, sender_id, receiver_id, created, updated)
			VALUES (?, '/topic/1', 'read', 'liked', 971300002, ?, ?, ?)`, strconv.Itoa(age), receiver, at, at).Error; err != nil {
			t.Fatal(err)
		}
	}
	count := func() int64 {
		var n int64
		db.Raw(`SELECT COUNT(*) FROM message WHERE receiver_id = ?`, receiver).Scan(&n)
		return n
	}
	expireReadNotifications(repo, days(0))()
	if got := count(); got != 2 {
		t.Fatalf("a retention of 0 deleted rows: %d left, want 2", got)
	}
	expireReadNotifications(repo, days(30))()
	var left []string
	if err := db.Raw(`SELECT content FROM message WHERE receiver_id = ? ORDER BY id`, receiver).Scan(&left).Error; err != nil {
		t.Fatal(err)
	}
	if len(left) != 1 || left[0] != "20" {
		t.Errorf("left %v, want only the 20-day-old row", left)
	}
}
