package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "", nil
	}
	b, err := json.Marshal([]string(s))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (s *StringSlice) Scan(src any) error {
	var str string
	switch v := src.(type) {
	case nil:
		*s = nil
		return nil
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("StringSlice: unsupported Scan type %T", src)
	}
	if strings.TrimSpace(str) == "" {
		*s = nil
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(str), &out); err != nil {
		return err
	}
	*s = out
	return nil
}

type TopicDraft struct {
	ID          int         `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      int         `gorm:"column:user_id;not null" json:"user_id"`
	Title       string      `gorm:"type:varchar(233);not null;default:''" json:"title"`
	Content     string      `gorm:"type:text;not null;default:''" json:"content"`
	Category    string      `gorm:"not null;default:''" json:"category"`
	Sections    StringSlice `gorm:"column:sections;type:text;not null;default:''" json:"sections"`
	CoverImages ImageTokens `gorm:"column:cover_images;type:text;not null;default:''" json:"cover_images"`
	IsNSFW      bool        `gorm:"column:is_nsfw;default:false" json:"is_nsfw"`

	Polls            json.RawMessage `gorm:"column:polls;serializer:json;type:jsonb;not null;default:'[]'" json:"polls"`
	Lotteries        json.RawMessage `gorm:"column:lotteries;serializer:json;type:jsonb;not null;default:'[]'" json:"lotteries"`
	PrizeImageHashes []string        `gorm:"column:prize_image_hashes;serializer:json;type:jsonb;not null;default:'[]'" json:"prize_image_hashes"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicDraft) TableName() string { return "topic_draft" }
