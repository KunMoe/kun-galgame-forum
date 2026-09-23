package repository

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type V1ConversationRow struct {
	RoomID         int        `gorm:"column:room_id"`
	PeerID         int        `gorm:"column:peer_id"`
	LastMessageAt  time.Time  `gorm:"column:last_message_at"`
	LastID         int        `gorm:"column:last_id"`
	LastSenderID   int        `gorm:"column:last_sender_id"`
	LastContent    string     `gorm:"column:last_content"`
	LastIsRecall   bool       `gorm:"column:last_is_recall"`
	LastRecallTime *time.Time `gorm:"column:last_recall_time"`
	LastCreated    time.Time  `gorm:"column:last_created"`
	MessageCount   int        `gorm:"column:message_count"`
	UnreadCount    int        `gorm:"column:unread_count"`
}

type V1ConversationPos struct {
	LastMessageAt time.Time
	RoomID        int
}

type V1ChatMessageRow struct {
	ID           int        `gorm:"column:id"`
	ChatRoomID   int        `gorm:"column:chat_room_id"`
	ChatroomName string     `gorm:"column:chatroom_name"`
	SenderID     int        `gorm:"column:sender_id"`
	ReceiverID   *int       `gorm:"column:receiver_id"`
	Content      string     `gorm:"column:content"`
	IsRecall     bool       `gorm:"column:is_recall"`
	RecallTime   *time.Time `gorm:"column:recall_time"`
	Created      time.Time  `gorm:"column:created"`
}

type V1MessagePos struct {
	ID int
}

func PrivateRoomName(a, b int) string {
	if a < b {
		return fmt.Sprintf("%d-%d", a, b)
	}
	return fmt.Sprintf("%d-%d", b, a)
}

func (r *ChatRepository) FindRoomByName(name string) (int, error) {
	return findRoomByName(r.db, name)
}

func findRoomByName(db *gorm.DB, name string) (int, error) {
	var id int
	err := db.Raw(`SELECT id FROM chat_room WHERE name = ?`, name).Scan(&id).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	return id, nil
}

func conversationSelectSQL() string {
	return `
SELECT
	cr.id AS room_id,
	peer.user_id AS peer_id,
	last.created AS last_message_at,
	last.id AS last_id,
	last.sender_id AS last_sender_id,
	last.content AS last_content,
	last.is_recall AS last_is_recall,
	last.recall_time AS last_recall_time,
	last.created AS last_created,
	(SELECT COUNT(*)::int FROM chat_message cm WHERE cm.chat_room_id = cr.id) AS message_count,
	(SELECT COUNT(*)::int FROM chat_message cm
		WHERE cm.chat_room_id = cr.id
		  AND cm.sender_id <> ?
		  AND NOT EXISTS (
			SELECT 1 FROM chat_message_read_by rb
			WHERE rb.chat_message_id = cm.id AND rb.user_id = ?
		  )) AS unread_count
FROM chat_room_participant me
JOIN chat_room cr ON cr.id = me.chat_room_id
JOIN LATERAL (
	SELECT p.user_id
	FROM chat_room_participant p
	WHERE p.chat_room_id = cr.id AND p.user_id <> me.user_id
	ORDER BY p.user_id
	LIMIT 1
) peer ON TRUE
JOIN LATERAL (
	SELECT cm.id, cm.sender_id, cm.content, cm.is_recall, cm.recall_time, cm.created
	FROM chat_message cm
	WHERE cm.chat_room_id = cr.id
	ORDER BY cm.id DESC
	LIMIT 1
) last ON TRUE
WHERE me.user_id = ?`
}

func (r *ChatRepository) FindConversationsKeyset(userID, limit int, pos *V1ConversationPos) ([]V1ConversationRow, error) {
	q := conversationSelectSQL()
	args := []any{userID, userID, userID}
	if pos != nil {
		q += ` AND (last.created, cr.id) < (?, ?)`
		args = append(args, pos.LastMessageAt, pos.RoomID)
	}
	q += ` ORDER BY last.created DESC, cr.id DESC LIMIT ?`
	args = append(args, limit)
	var rows []V1ConversationRow
	err := r.db.Raw(q, args...).Scan(&rows).Error
	if rows == nil {
		rows = []V1ConversationRow{}
	}
	return rows, err
}

func (r *ChatRepository) FindConversation(userID, peerID int) (*V1ConversationRow, error) {
	name := PrivateRoomName(userID, peerID)
	roomID, err := r.FindRoomByName(name)
	if err != nil {
		return nil, err
	}
	if roomID == 0 {
		return nil, nil
	}
	q := conversationSelectSQL() + ` AND cr.id = ?`
	var row V1ConversationRow
	err = r.db.Raw(q, userID, userID, userID, roomID).Scan(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if row.RoomID == 0 {
		return nil, nil
	}
	return &row, nil
}

func (r *ChatRepository) FindDirectMessagesKeyset(roomID, limit int, pos *V1MessagePos) ([]V1ChatMessageRow, error) {
	q := r.db.Table("chat_message").
		Select("id, chat_room_id, chatroom_name, sender_id, receiver_id, content, is_recall, recall_time, created").
		Where("chat_room_id = ?", roomID)
	if pos != nil {
		q = q.Where("id < ?", pos.ID)
	}
	var rows []V1ChatMessageRow
	err := q.Order("id DESC").Limit(limit).Scan(&rows).Error
	if rows == nil {
		rows = []V1ChatMessageRow{}
	}
	return rows, err
}

func (r *ChatRepository) FindDirectMessage(id, roomID int) (*V1ChatMessageRow, error) {
	var row V1ChatMessageRow
	err := r.db.Table("chat_message").
		Select("id, chat_room_id, chatroom_name, sender_id, receiver_id, content, is_recall, recall_time, created").
		Where("id = ? AND chat_room_id = ?", id, roomID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if row.ID == 0 {
		return nil, nil
	}
	return &row, nil
}

// Private rooms are found by the unique name <smaller id>-<larger id>, which
// matches the participant pair in production and uses the name index.
func ensurePrivateRoom(tx *gorm.DB, uid1, uid2 int) (id int, name string, err error) {
	name = PrivateRoomName(uid1, uid2)
	now := time.Now().UTC()
	if err = tx.Exec(
		`INSERT INTO chat_room (name, type, created, updated) VALUES (?, 'private', ?, ?) ON CONFLICT (name) DO NOTHING`,
		name, now, now,
	).Error; err != nil {
		return 0, "", err
	}
	id, err = findRoomByName(tx, name)
	if err != nil {
		return 0, "", err
	}
	if id == 0 {
		return 0, "", fmt.Errorf("chat_room %s missing after insert", name)
	}
	if err = tx.Exec(
		`INSERT INTO chat_room_participant (chat_room_id, user_id, created, updated)
VALUES (?, ?, ?, ?), (?, ?, ?, ?)
ON CONFLICT (chat_room_id, user_id) DO NOTHING`,
		id, uid1, now, now, id, uid2, now, now,
	).Error; err != nil {
		return 0, "", err
	}
	return id, name, nil
}

func (r *ChatRepository) SendDirectMessage(senderID, peerID int, senderName, content string) (*V1ChatMessageRow, error) {
	var out V1ChatMessageRow
	err := r.db.Transaction(func(tx *gorm.DB) error {
		roomID, roomName, err := ensurePrivateRoom(tx, senderID, peerID)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if err := tx.Raw(
			`INSERT INTO chat_message (chat_room_id, chatroom_name, sender_id, receiver_id, content, created, updated)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING id, chat_room_id, chatroom_name, sender_id, receiver_id, content, is_recall, recall_time, created`,
			roomID, roomName, senderID, peerID, content, now, now,
		).Scan(&out).Error; err != nil {
			return err
		}
		return tx.Exec(
			`UPDATE chat_room SET last_message_content = ?, last_message_time = ?,
last_message_sender_id = ?, last_message_sender_name = ?, updated = ? WHERE id = ?`,
			content, now, senderID, senderName, now, roomID,
		).Error
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *ChatRepository) RecallDirectMessage(id, roomID int, now time.Time) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			`UPDATE chat_message SET is_recall = TRUE, recall_time = ?, updated = ?, content = '' WHERE id = ? AND chat_room_id = ?`,
			now, now, id, roomID,
		).Error; err != nil {
			return err
		}
		var latest int
		if err := tx.Raw(`SELECT COALESCE(MAX(id), 0) FROM chat_message WHERE chat_room_id = ?`, roomID).Scan(&latest).Error; err != nil {
			return err
		}
		if latest != id {
			return nil
		}
		return tx.Exec(`UPDATE chat_room SET last_message_content = '', updated = ? WHERE id = ?`, now, roomID).Error
	})
}

func (r *ChatRepository) MarkDirectReadUpTo(userID, roomID, upToID int) (marked int, unread int, err error) {
	err = r.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Exec(
			`INSERT INTO chat_message_read_by (chat_message_id, user_id, created, updated, read_time)
SELECT id, ?, now(), now(), now()
FROM chat_message
WHERE chat_room_id = ? AND sender_id <> ? AND id <= ?
ON CONFLICT DO NOTHING`,
			userID, roomID, userID, upToID,
		)
		if res.Error != nil {
			return res.Error
		}
		marked = int(res.RowsAffected)
		var remaining int64
		if err := tx.Raw(
			`SELECT COUNT(*) FROM chat_message cm
WHERE cm.chat_room_id = ? AND cm.sender_id <> ?
  AND NOT EXISTS (
	SELECT 1 FROM chat_message_read_by rb
	WHERE rb.chat_message_id = cm.id AND rb.user_id = ?
  )`,
			roomID, userID, userID,
		).Scan(&remaining).Error; err != nil {
			return err
		}
		unread = int(remaining)
		return nil
	})
	return marked, unread, err
}
