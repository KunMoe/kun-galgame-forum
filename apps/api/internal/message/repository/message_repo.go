package repository

import "gorm.io/gorm"

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) DB() *gorm.DB {
	return r.db
}

func (r *MessageRepository) MarkCommunityThreadRead(receiverID int, threadID int64, lastRead int32) error {
	return r.db.Exec(`
		UPDATE message
		SET status = 'read', updated = now()
		WHERE receiver_id = ?
		  AND community_thread_id = ?
		  AND status = 'unread'
		  AND type IN ('replied', 'commented', 'mentioned', 'followed')
		  AND community_post_number <= ?
	`, receiverID, threadID, lastRead).Error
}
