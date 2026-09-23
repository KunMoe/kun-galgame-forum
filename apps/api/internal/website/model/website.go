package model

import (
	"encoding/json"
	"time"
)

type GalgameWebsite struct {
	ID            int             `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string          `gorm:"uniqueIndex;not null" json:"name"`
	URL           string          `gorm:"column:url;uniqueIndex;not null" json:"url"`
	CreateTime    string          `gorm:"column:create_time;not null" json:"create_time"`
	Description   string          `gorm:"default:''" json:"description"`
	Icon          string          `gorm:"default:''" json:"icon"`
	IconImageHash string          `gorm:"column:icon_image_hash;default:''" json:"icon_image_hash"`
	View          int             `gorm:"default:0" json:"view"`
	Language      string          `gorm:"default:'JA'" json:"language"`
	AgeLimit      string          `gorm:"column:age_limit;default:'all'" json:"age_limit"`
	Status        string          `gorm:"default:'normal'" json:"status"`
	Domain        json.RawMessage `gorm:"type:jsonb;default:'[]'" json:"domain"`

	CategoryID int `gorm:"column:category_id;not null;constraint:OnDelete:RESTRICT" json:"category_id"`
	UserID     int `gorm:"column:user_id;not null;default:2" json:"user_id"`

	LikeCount     int `gorm:"column:like_count;default:0" json:"like_count"`
	FavoriteCount int `gorm:"column:favorite_count;default:0" json:"favorite_count"`
	CommentCount  int `gorm:"column:comment_count;default:0" json:"comment_count"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameWebsite) TableName() string { return "galgame_website" }

type GalgameWebsiteCategory struct {
	ID          int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string `gorm:"uniqueIndex;not null" json:"name"`
	Label       string `gorm:"default:''" json:"label"`
	Description string `gorm:"default:''" json:"description"`
	SortOrder   int    `gorm:"column:sort_order;default:0" json:"sort_order"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameWebsiteCategory) TableName() string { return "galgame_website_category" }

type GalgameWebsiteTagGroup struct {
	ID          int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string `gorm:"uniqueIndex;not null" json:"name"`
	Label       string `gorm:"default:''" json:"label"`
	Description string `gorm:"default:''" json:"description"`
	SortOrder   int    `gorm:"column:sort_order;default:0" json:"sort_order"`
	MultiSelect bool   `gorm:"column:multi_select;default:false" json:"multi_select"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameWebsiteTagGroup) TableName() string { return "galgame_website_tag_group" }

type GalgameWebsiteTag struct {
	ID          int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Level       int    `gorm:"not null" json:"level"`
	Name        string `gorm:"uniqueIndex;not null" json:"name"`
	Label       string `gorm:"default:''" json:"label"`
	Description string `gorm:"default:''" json:"description"`
	GroupID     *int   `gorm:"column:group_id" json:"group_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameWebsiteTag) TableName() string { return "galgame_website_tag" }
