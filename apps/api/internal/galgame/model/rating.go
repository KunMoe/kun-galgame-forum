package model

import (
	"encoding/json"
	"time"
)

type GalgameRating struct {
	ID           int             `gorm:"primaryKey;autoIncrement" json:"id"`
	Recommend    string          `gorm:"type:varchar" json:"recommend"`
	Overall      int             `json:"overall"`
	View         int             `gorm:"default:0" json:"view"`
	GalgameType  json.RawMessage `gorm:"column:galgame_type;type:jsonb" json:"galgame_type"`
	PlayStatus   string          `gorm:"column:play_status;type:varchar" json:"play_status"`
	ShortSummary string          `gorm:"column:short_summary;type:varchar(1314)" json:"short_summary"`
	SpoilerLevel string          `gorm:"column:spoiler_level;type:varchar" json:"spoiler_level"`
	Art          int             `gorm:"default:0" json:"art"`
	Story        int             `gorm:"default:0" json:"story"`
	Music        int             `gorm:"default:0" json:"music"`
	Character    int             `gorm:"default:0" json:"character"`
	Route        int             `gorm:"default:0" json:"route"`
	System       int             `gorm:"default:0" json:"system"`
	Voice        int             `gorm:"default:0" json:"voice"`
	ReplayValue  int             `gorm:"column:replay_value;default:0" json:"replay_value"`

	WorkID       int `gorm:"column:work_id;not null;constraint:OnDelete:RESTRICT" json:"galgame_id"`
	UserID       int `gorm:"column:user_id;not null" json:"user_id"`
	LikeCount    int `gorm:"column:like_count;default:0" json:"like_count"`
	CommentCount int `gorm:"column:comment_count;default:0" json:"comment_count"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}
