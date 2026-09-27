package model

import "time"

const (
	SubscriptionWatching = "watching"
	SubscriptionNormal   = "normal"
	SubscriptionMuted    = "muted"
)

type TopicSubscription struct {
	UserID            int       `gorm:"column:user_id;primaryKey"`
	TopicID           int       `gorm:"column:topic_id;primaryKey"`
	NotificationLevel string    `gorm:"column:notification_level;not null"`
	LastReadFloor     int       `gorm:"column:last_read_floor;not null;default:0"`
	ActivityAt        time.Time `gorm:"column:activity_at;not null"`
	NoticeMessageID   *int      `gorm:"column:notice_message_id"`
	NoticeFromFloor   *int      `gorm:"column:notice_from_floor"`
	CreatedAt         time.Time `gorm:"column:created"`
	UpdatedAt         time.Time `gorm:"column:updated"`
}

func (TopicSubscription) TableName() string { return "topic_subscription" }
