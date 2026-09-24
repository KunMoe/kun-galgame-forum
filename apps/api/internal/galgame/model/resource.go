package model

import (
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
