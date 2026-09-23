package model

import "time"

type Message struct {
	ID      int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Content string `gorm:"type:varchar(233);default:''" json:"content"`
	Link    string `gorm:"type:varchar(100);default:''" json:"link"`
	Status  string `gorm:"default:'unread'" json:"status"`
	Type    string `gorm:"not null" json:"type"`

	SenderID   int `gorm:"column:sender_id;not null" json:"sender_id"`
	ReceiverID int `gorm:"column:receiver_id;not null" json:"receiver_id"`

	CommunityNotificationID *int64 `gorm:"column:community_notification_id" json:"community_notification_id"`
	CommunitySeq            *int64 `gorm:"column:community_seq" json:"community_seq"`
	CommunityThreadID       *int64 `gorm:"column:community_thread_id" json:"community_thread_id"`
	CommunityPostNumber     *int   `gorm:"column:community_post_number" json:"community_post_number"`
	ItemCount               int    `gorm:"column:item_count;default:1;not null" json:"item_count"`
	ActorCount              int    `gorm:"column:actor_count;default:1;not null" json:"actor_count"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (Message) TableName() string { return "message" }
