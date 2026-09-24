package model

import (
	"encoding/json"
	"time"
)

type GalgameToolset struct {
	ID                 int             `gorm:"primaryKey;autoIncrement" json:"id"`
	Name               string          `gorm:"type:varchar(500);default:''" json:"name"`
	Description        string          `gorm:"type:varchar(2000);default:''" json:"description"`
	Status             int             `gorm:"default:0" json:"status"`
	View               int             `gorm:"default:0" json:"view"`
	Type               string          `gorm:"default:''" json:"type"`
	Language           string          `gorm:"default:''" json:"language"`
	Platform           string          `gorm:"default:''" json:"platform"`
	Homepage           json.RawMessage `gorm:"type:jsonb;default:'[]'" json:"homepage"`
	ResourceUpdateTime time.Time       `gorm:"column:resource_update_time;autoCreateTime" json:"resource_update_time"`
	Edited             *time.Time      `gorm:"" json:"edited"`
	Version            string          `gorm:"type:varchar(233);default:''" json:"version"`
	CommentCount       int             `gorm:"column:comment_count;not null;default:0" json:"comment_count"`

	UserID int `gorm:"column:user_id;not null" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameToolset) TableName() string { return "galgame_toolset" }

type GalgameToolsetContributor struct {
	ID        int `gorm:"primaryKey;autoIncrement" json:"id"`
	ToolsetID int `gorm:"column:toolset_id;not null;uniqueIndex:idx_toolset_contributor" json:"toolset_id"`
	UserID    int `gorm:"column:user_id;not null;uniqueIndex:idx_toolset_contributor" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameToolsetContributor) TableName() string { return "galgame_toolset_contributor" }

type GalgameToolsetPracticality struct {
	ID        int `gorm:"primaryKey;autoIncrement" json:"id"`
	Rate      int `gorm:"default:1" json:"rate"`
	UserID    int `gorm:"column:user_id;not null" json:"user_id"`
	ToolsetID int `gorm:"column:toolset_id;not null" json:"toolset_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameToolsetPracticality) TableName() string { return "galgame_toolset_practicality" }

type GalgameToolsetAlias struct {
	ID        int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string `gorm:"default:''" json:"name"`
	ToolsetID int    `gorm:"column:toolset_id;not null;uniqueIndex:idx_toolset_alias" json:"toolset_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameToolsetAlias) TableName() string { return "galgame_toolset_alias" }

type GalgameToolsetResource struct {
	ID           int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Content      string     `gorm:"type:varchar(1007);default:''" json:"content"`
	Type         string     `gorm:"default:''" json:"type"`
	ArtifactUUID string     `gorm:"column:artifact_uuid;type:varchar(36);not null;default:''" json:"artifact_uuid"`
	Code         string     `gorm:"type:varchar(1007);default:''" json:"code"`
	Password     string     `gorm:"type:varchar(1007);default:''" json:"password"`
	Size         string     `gorm:"type:varchar(107);default:''" json:"size"`
	Note         string     `gorm:"type:varchar(1007);default:''" json:"note"`
	Download     int        `gorm:"default:0" json:"download"`
	Status       int        `gorm:"default:0" json:"status"`
	Edited       *time.Time `gorm:"" json:"edited"`

	ToolsetID int `gorm:"column:toolset_id;not null;uniqueIndex:idx_toolset_resource" json:"toolset_id"`
	UserID    int `gorm:"column:user_id;not null" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameToolsetResource) TableName() string { return "galgame_toolset_resource" }

type GalgameToolsetCategoryRelation struct {
	ToolsetID  int `gorm:"column:toolset_id;primaryKey" json:"toolset_id"`
	CategoryID int `gorm:"column:category_id;primaryKey" json:"category_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameToolsetCategoryRelation) TableName() string { return "galgame_toolset_category_relation" }

type ToolsetUpload struct {
	ArtifactUUID string     `gorm:"column:artifact_uuid;primaryKey;type:varchar(36)" json:"artifact_uuid"`
	ToolsetID    int        `gorm:"column:toolset_id;not null" json:"toolset_id"`
	UserID       int        `gorm:"column:user_id;not null" json:"user_id"`
	Filename     string     `gorm:"type:varchar(1007);not null" json:"filename"`
	FileSize     int64      `gorm:"column:file_size;not null" json:"file_size"`
	CreatedAt    time.Time  `gorm:"column:created" json:"created"`
	CompletedAt  *time.Time `gorm:"column:completed_at" json:"completed_at"`
}

func (ToolsetUpload) TableName() string { return "toolset_upload" }
