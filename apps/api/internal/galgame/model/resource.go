package model

import (
	"encoding/json"
	"time"

	"kun-galgame-api/internal/galgame/resourcevocab"
)

type GalgameResource struct {
	ID           int                `gorm:"primaryKey;autoIncrement" json:"id"`
	Type         string             `gorm:"type:varchar" json:"type"`
	Language     string             `gorm:"type:varchar" json:"language"`
	Platform     string             `gorm:"type:varchar" json:"platform"`
	Title        string             `gorm:"column:title" json:"title"`
	VersionLabel string             `gorm:"column:version_label" json:"version_label"`
	Languages    resourcevocab.Keys `gorm:"column:languages;type:jsonb" json:"languages"`
	Platforms    resourcevocab.Keys `gorm:"column:platforms;type:jsonb" json:"platforms"`
	Runtimes     resourcevocab.Keys `gorm:"column:runtimes;type:jsonb" json:"runtimes"`
	Size         string             `gorm:"type:varchar(107)" json:"size"`
	Code         string             `gorm:"type:varchar(1007)" json:"code"`
	Password     string             `gorm:"type:varchar(1007)" json:"password"`
	Note         string             `gorm:"type:varchar(10000)" json:"note"`
	View         int                `gorm:"default:0" json:"view"`
	Status       int                `gorm:"default:0" json:"status"`
	Download     int                `gorm:"default:0" json:"download"`
	WorkID       int                `gorm:"column:work_id;not null" json:"galgame_id"`
	UserID       int                `gorm:"column:user_id;not null" json:"user_id"`
	LikeCount    int                `gorm:"column:like_count;default:0" json:"like_count"`
	CommentCount int                `gorm:"column:comment_count;default:0" json:"comment_count"`

	Edited    *time.Time `gorm:"column:edited" json:"edited"`
	CreatedAt time.Time  `gorm:"column:created" json:"created"`
	UpdatedAt time.Time  `gorm:"column:updated" json:"updated"`
}

func (GalgameResource) TableName() string { return "galgame_resource" }

type GalgameResourceLink struct {
	ID                int    `gorm:"primaryKey;autoIncrement" json:"id"`
	URL               string `gorm:"column:url" json:"url"`
	GalgameResourceID int    `gorm:"column:galgame_resource_id;not null" json:"galgame_resource_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameResourceLink) TableName() string { return "galgame_resource_link" }

type GalgameResourceLike struct {
	ID                int `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID            int `gorm:"column:user_id;not null;uniqueIndex:idx_resource_like" json:"user_id"`
	GalgameResourceID int `gorm:"column:galgame_resource_id;not null;uniqueIndex:idx_resource_like" json:"galgame_resource_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameResourceLike) TableName() string { return "galgame_resource_like" }

type GalgameResourceRow struct {
	ID           int                `gorm:"column:id"`
	View         int                `gorm:"column:view"`
	WorkID       int                `gorm:"column:work_id"`
	UserID       int                `gorm:"column:user_id"`
	Type         string             `gorm:"column:type"`
	Language     string             `gorm:"column:language"`
	Platform     string             `gorm:"column:platform"`
	Title        string             `gorm:"column:title"`
	VersionLabel string             `gorm:"column:version_label"`
	Languages    resourcevocab.Keys `gorm:"column:languages"`
	Platforms    resourcevocab.Keys `gorm:"column:platforms"`
	Runtimes     resourcevocab.Keys `gorm:"column:runtimes"`
	Size         string             `gorm:"column:size"`
	Status       int                `gorm:"column:status"`
	Download     int                `gorm:"column:download"`
	LikeCount    int                `gorm:"column:like_count"`
	CommentCount int                `gorm:"column:comment_count"`
	Code         string             `gorm:"column:code"`
	Password     string             `gorm:"column:password"`
	Note         string             `gorm:"column:note"`
	ProviderName json.RawMessage    `gorm:"column:provider_name"`
	Created      string             `gorm:"column:created"`
	Edited       *string            `gorm:"column:edited"`
}

type ResourceAggregate struct {
	Platform string `gorm:"column:platform"`
	Language string `gorm:"column:language"`
	Type     string `gorm:"column:type"`
}
