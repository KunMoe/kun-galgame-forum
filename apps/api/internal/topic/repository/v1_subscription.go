package repository

import (
	"errors"
	"time"

	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
)

// FindSubscription returns nil when the reader has no row, which is the
// normal level.
func FindSubscription(tx *gorm.DB, userID, topicID int) (*model.TopicSubscription, error) {
	var row model.TopicSubscription
	err := tx.Where("user_id = ? AND topic_id = ?", userID, topicID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func LastVisibleFloor(tx *gorm.DB, topicID int) (int, error) {
	var floor int
	err := tx.Raw(`SELECT COALESCE(MAX(floor), 0) FROM topic_reply WHERE topic_id = ? AND status = 0`, topicID).
		Row().Scan(&floor)
	return floor, err
}

func SetSubscriptionLevel(tx *gorm.DB, userID, topicID int, level string) (*model.TopicSubscription, error) {
	switch level {
	case model.SubscriptionNormal:
		err := tx.Exec(`DELETE FROM topic_subscription WHERE user_id = ? AND topic_id = ?`, userID, topicID).Error
		return nil, err
	case model.SubscriptionWatching:
		last, err := LastVisibleFloor(tx, topicID)
		if err != nil {
			return nil, err
		}
		if err := tx.Exec(`INSERT INTO topic_subscription (user_id, topic_id, notification_level, last_read_floor, activity_at)
			VALUES (?, ?, 'watching', ?, now())
			ON CONFLICT (user_id, topic_id) DO UPDATE SET
				notification_level = 'watching',
				last_read_floor = EXCLUDED.last_read_floor,
				activity_at = EXCLUDED.activity_at,
				notice_message_id = NULL,
				notice_from_floor = NULL,
				updated = now()
			WHERE topic_subscription.notification_level <> 'watching'`, userID, topicID, last).Error; err != nil {
			return nil, err
		}
	case model.SubscriptionMuted:
		if err := tx.Exec(`INSERT INTO topic_subscription (user_id, topic_id, notification_level)
			VALUES (?, ?, 'muted')
			ON CONFLICT (user_id, topic_id) DO UPDATE SET notification_level = 'muted', updated = now()
			WHERE topic_subscription.notification_level <> 'muted'`, userID, topicID).Error; err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unknown subscription level " + level)
	}
	return FindSubscription(tx, userID, topicID)
}

func WatchOwnTopic(tx *gorm.DB, userID, topicID int) error {
	return tx.Exec(`INSERT INTO topic_subscription (user_id, topic_id, notification_level)
		VALUES (?, ?, 'watching') ON CONFLICT (user_id, topic_id) DO NOTHING`, userID, topicID).Error
}

// MarkTopicRead moves a watching reader's position forward, never back, and
// closes their open folded notice once they have seen the last visible floor.
func MarkTopicRead(tx *gorm.DB, userID, topicID, floor int) (*model.TopicSubscription, error) {
	last, err := LastVisibleFloor(tx, topicID)
	if err != nil {
		return nil, err
	}
	floor = min(floor, last)
	if err := tx.Exec(`UPDATE topic_subscription SET last_read_floor = ?, updated = now()
		WHERE user_id = ? AND topic_id = ? AND notification_level = 'watching' AND last_read_floor < ?`,
		floor, userID, topicID, floor).Error; err != nil {
		return nil, err
	}
	row, err := FindSubscription(tx, userID, topicID)
	if err != nil || row == nil || row.NotificationLevel != model.SubscriptionWatching {
		return row, err
	}
	if row.NoticeMessageID != nil && row.LastReadFloor >= last {
		if err := tx.Exec(`UPDATE message SET status = 'read', updated = now()
			WHERE id = ? AND receiver_id = ? AND status = 'unread'`, *row.NoticeMessageID, userID).Error; err != nil {
			return nil, err
		}
	}
	return row, nil
}

func SubscriptionLevels(tx *gorm.DB, topicID int, userIDs []int) (map[int]string, error) {
	out := make(map[int]string, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		UserID            int
		NotificationLevel string
	}
	if err := tx.Raw(`SELECT user_id, notification_level FROM topic_subscription
		WHERE topic_id = ? AND user_id IN ?`, topicID, userIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.UserID] = r.NotificationLevel
	}
	return out, nil
}

type ReplyFanOut struct {
	TopicID    int
	Floor      int
	ReplierID  int
	Preview    string
	Link       string
	NoticeType string
	// Exclude holds the readers this reply already notifies another way: the
	// author's replied message and the mentioned users' mentioned message.
	Exclude []int
	// Notify is false for a topic the fan-out cannot judge visibility of.
	Notify bool
}

func FanOutReply(tx *gorm.DB, in ReplyFanOut) error {
	if err := tx.Exec(`UPDATE topic_subscription SET last_read_floor = GREATEST(last_read_floor, ?), updated = now()
		WHERE user_id = ? AND topic_id = ? AND notification_level = 'watching'`,
		in.Floor, in.ReplierID, in.TopicID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`UPDATE topic_subscription SET activity_at = now()
		WHERE topic_id = ? AND notification_level = 'watching'`, in.TopicID).Error; err != nil {
		return err
	}
	if !in.Notify {
		return nil
	}
	exclude := append([]int{in.ReplierID}, in.Exclude...)
	var recipients []int
	if err := tx.Raw(`SELECT user_id FROM topic_subscription
		WHERE topic_id = ? AND notification_level = 'watching' AND user_id NOT IN ?`,
		in.TopicID, exclude).Scan(&recipients).Error; err != nil {
		return err
	}
	if len(recipients) == 0 {
		return nil
	}

	continued, err := returnedIDs(tx.Raw(`UPDATE message m SET
			sender_id = ?, content = ?, item_count = m.item_count + 1,
			actor_count = (SELECT COUNT(DISTINCT r.user_id) FROM topic_reply r
				WHERE r.topic_id = s.topic_id AND r.status = 0
				  AND r.floor >= s.notice_from_floor AND r.floor <= ? AND r.user_id <> m.receiver_id),
			created = now(), updated = now()
		FROM topic_subscription s
		WHERE s.topic_id = ? AND s.user_id IN ? AND s.notice_message_id = m.id
		  AND m.receiver_id = s.user_id AND m.type = ? AND m.status = 'unread'
		RETURNING m.receiver_id`,
		in.ReplierID, in.Preview, in.Floor, in.TopicID, recipients, in.NoticeType))
	if err != nil {
		return err
	}
	open := make(map[int]bool, len(continued))
	for _, id := range continued {
		open[id] = true
	}
	var fresh []int
	for _, id := range recipients {
		if !open[id] {
			fresh = append(fresh, id)
		}
	}
	if len(fresh) == 0 {
		return nil
	}
	return tx.Exec(`WITH opened AS (
			INSERT INTO message (sender_id, receiver_id, type, content, link, status, item_count, actor_count, created, updated)
			SELECT ?, s.user_id, ?, ?, ?, 'unread', 1, 1, now(), now()
			FROM topic_subscription s WHERE s.topic_id = ? AND s.user_id IN ?
			RETURNING id, receiver_id
		)
		UPDATE topic_subscription s SET notice_message_id = opened.id, notice_from_floor = ?
		FROM opened WHERE s.topic_id = ? AND s.user_id = opened.receiver_id`,
		in.ReplierID, in.NoticeType, in.Preview, in.Link, in.TopicID, fresh,
		in.Floor, in.TopicID).Error
}

// Raw(...RETURNING...).Scan reported an error after the UPDATE had committed
// (the notification mark-read face), so RETURNING rows are read by hand.
func returnedIDs(q *gorm.DB) ([]int, error) {
	rows, err := q.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

type SubscribedTopicRow struct {
	TopicKeysetRow   `gorm:"embedded"`
	AccessScope      string
	HiddenBy         string
	LastReadFloor    int
	ActivityAt       time.Time
	UnreadCount      int
	FirstUnreadFloor *int
}

type SubscribedPos struct {
	ActivityAt time.Time
	TopicID    int
}

type SubscribedQuery struct {
	UserID      int
	HasUnread   bool
	IncludeNSFW bool
	Limit       int
	Pos         *SubscribedPos
}

const unreadReplies = `FROM topic_reply r WHERE r.topic_id = s.topic_id AND r.status = 0
	AND r.floor > s.last_read_floor AND r.user_id <> s.user_id`

func (r *TopicRepository) ListSubscribed(q SubscribedQuery) ([]SubscribedTopicRow, error) {
	query := r.db.Table("topic_subscription s").
		Joins("JOIN topic ON topic.id = s.topic_id").
		Select(`topic.id, topic.title, topic.view, topic.status,
			topic.is_nsfw, topic.like_count, topic.reply_count,
			topic.comment_count, topic.best_answer_id,
			topic.status_update_time, topic.created, topic.upvote_time, topic.edited,
			topic.cover_images, topic.user_id, topic.category,
			topic.favorite_count, topic.upvote_count, topic.access_scope, topic.hidden_by,
			s.last_read_floor, s.activity_at,
			(SELECT COUNT(*) `+unreadReplies+`) AS unread_count,
			(SELECT MIN(r.floor) `+unreadReplies+`) AS first_unread_floor`).
		Where("s.user_id = ? AND s.notification_level = 'watching'", q.UserID)
	if !q.IncludeNSFW {
		query = query.Where("topic.is_nsfw = false")
	}
	if q.HasUnread {
		query = query.Where("EXISTS (SELECT 1 " + unreadReplies + ")")
	}
	if q.Pos != nil {
		query = query.Where("(s.activity_at, s.topic_id) < (?, ?)", q.Pos.ActivityAt, q.Pos.TopicID)
	}
	var rows []SubscribedTopicRow
	err := query.Order("s.activity_at DESC, s.topic_id DESC").Limit(q.Limit + 1).Scan(&rows).Error
	return rows, err
}
