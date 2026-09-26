package repository

import (
	"kun-galgame-api/internal/message/model"
)

func (r *MessageRepository) UpsertCommunityMirror(m *model.Message) error {
	// The seq guard makes a replayed page (cursor lost, two replicas) a no-op,
	// so it cannot flip a read row back to unread.
	return r.db.Exec(`
		INSERT INTO message (
			content, link, status, type, sender_id, receiver_id, created, updated,
			community_notification_id, community_seq, community_thread_id, community_post_number,
			item_count, actor_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, now(), ?, ?, ?, ?, ?, ?)
		ON CONFLICT (community_notification_id) WHERE community_notification_id IS NOT NULL
		DO UPDATE SET
			content = EXCLUDED.content,
			link = EXCLUDED.link,
			status = EXCLUDED.status,
			type = EXCLUDED.type,
			sender_id = EXCLUDED.sender_id,
			receiver_id = EXCLUDED.receiver_id,
			created = EXCLUDED.created,
			community_seq = EXCLUDED.community_seq,
			community_thread_id = EXCLUDED.community_thread_id,
			community_post_number = EXCLUDED.community_post_number,
			item_count = EXCLUDED.item_count,
			actor_count = EXCLUDED.actor_count,
			updated = now()
		WHERE message.community_seq < EXCLUDED.community_seq
	`,
		m.Content, m.Link, m.Status, m.Type, m.SenderID, m.ReceiverID, m.CreatedAt,
		m.CommunityNotificationID, m.CommunitySeq, m.CommunityThreadID, m.CommunityPostNumber,
		m.ItemCount, m.ActorCount,
	).Error
}

func (r *MessageRepository) DeleteCommunityMirror(id, seq int64) error {
	return r.db.Exec(
		`DELETE FROM message WHERE community_notification_id = ? AND community_seq < ?`,
		id, seq,
	).Error
}

func (r *MessageRepository) MaxCommunitySeq() (int64, error) {
	var max int64
	err := r.db.Raw(`SELECT COALESCE(MAX(community_seq), 0) FROM message`).Scan(&max).Error
	return max, err
}
