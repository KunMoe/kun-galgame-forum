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

	if expireReadNotifications(repo, 0) != nil || expireReadNotifications(repo, -1) != nil {
		t.Fatal("a retention of 0 or less must leave the job unregistered")
	}

	now := time.Now()
	for _, age := range []int{40, 20} {
		at := now.AddDate(0, 0, -age)
		if err := db.Exec(`INSERT INTO message (content, link, status, type, sender_id, receiver_id, created, updated)
			VALUES (?, '/topic/1', 'read', 'liked', 971300002, ?, ?, ?)`, strconv.Itoa(age), receiver, at, at).Error; err != nil {
			t.Fatal(err)
		}
	}
	job := expireReadNotifications(repo, 30)
	if job == nil {
		t.Fatal("a 30-day retention registered no job")
	}
	job()
	var left []string
	if err := db.Raw(`SELECT content FROM message WHERE receiver_id = ? ORDER BY id`, receiver).Scan(&left).Error; err != nil {
		t.Fatal(err)
	}
	if len(left) != 1 || left[0] != "20" {
		t.Errorf("left %v, want only the 20-day-old row", left)
	}
}
