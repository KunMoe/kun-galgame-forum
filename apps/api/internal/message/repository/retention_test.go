package repository

import (
	"testing"
	"time"

	"kun-galgame-api/internal/testdb"
)

func TestDeleteReadBefore(t *testing.T) {
	db := testdb.Open(t)
	const receiver = 971200001
	clean := func() { db.Exec(`DELETE FROM message WHERE receiver_id = ?`, receiver) }
	clean()
	t.Cleanup(clean)

	now := time.Now()
	day := 24 * time.Hour
	insert := func(typ, status string, age time.Duration) {
		t.Helper()
		at := now.Add(-age)
		if err := db.Exec(`INSERT INTO message (content, link, status, type, sender_id, receiver_id, created, updated)
			VALUES ('c', '/topic/1', ?, ?, 971200002, ?, ?, ?)`, status, typ, receiver, at, at).Error; err != nil {
			t.Fatal(err)
		}
	}
	for range 5 {
		insert("liked", "read", 400*day)
	}
	insert("replied", "unread", 400*day)
	insert("solution", "read", 400*day)
	insert("liked", "read", 300*day)

	n, err := NewMessageRepository(db).DeleteReadBefore(now.Add(-365*day), 2)
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Errorf("deleted %d rows, want the 5 read ones past the cutoff", n)
	}
	var left []string
	if err := db.Raw(`SELECT type || ' ' || status FROM message WHERE receiver_id = ? ORDER BY id`, receiver).Scan(&left).Error; err != nil {
		t.Fatal(err)
	}
	want := []string{"replied unread", "solution read", "liked read"}
	if len(left) != len(want) {
		t.Fatalf("left %v, want %v", left, want)
	}
	for i := range want {
		if left[i] != want[i] {
			t.Errorf("left %v, want %v", left, want)
			break
		}
	}
}
