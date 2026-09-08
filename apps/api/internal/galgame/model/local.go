package model

import "time"

type GalgameLocal struct {
	ID                    int        `gorm:"primaryKey" json:"id"`
	View                  int        `gorm:"default:0" json:"view"`
	LikeCount             int        `gorm:"column:like_count;default:0" json:"like_count"`
	FavoriteCount         int        `gorm:"column:favorite_count;default:0" json:"favorite_count"`
	ResourceCount         int        `gorm:"column:resource_count;default:0" json:"resource_count"`
	CommentCount          int        `gorm:"column:comment_count;default:0" json:"comment_count"`
	ContributorCount      int        `gorm:"column:contributor_count;default:0" json:"contributor_count"`
	RatingCount           int        `gorm:"column:rating_count;default:0" json:"rating_count"`
	ReleaseDate           *time.Time `gorm:"column:release_date" json:"release_date"`
	ReleaseDateSyncedAt   *time.Time `gorm:"column:release_date_synced_at" json:"-"`
	CreatedAt             time.Time  `gorm:"column:created" json:"created"`
	UpdatedAt             time.Time  `gorm:"column:updated" json:"updated"`
	ResourceUpdateTime    time.Time  `gorm:"column:resource_update_time;autoCreateTime" json:"resource_update_time"`
	ResourcePublishBanned bool       `gorm:"column:resource_publish_banned;default:false" json:"resource_publish_banned"`
	Published             bool       `gorm:"column:published;not null" json:"published"`
	ContentLimit          *string    `gorm:"column:content_limit" json:"content_limit"`
	CreatorUserID         *int       `gorm:"column:creator_user_id" json:"creator_user_id"`
}

func (GalgameLocal) TableName() string { return "galgame" }

type GalgameLike struct {
	ID        int `gorm:"primaryKey;autoIncrement" json:"id"`
	GalgameID int `gorm:"column:galgame_id;not null;uniqueIndex:idx_galgame_like" json:"galgame_id"`
	UserID    int `gorm:"column:user_id;not null;uniqueIndex:idx_galgame_like" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameLike) TableName() string { return "galgame_like" }

type GalgameFavorite struct {
	ID        int `gorm:"primaryKey;autoIncrement" json:"id"`
	GalgameID int `gorm:"column:galgame_id;not null;uniqueIndex:idx_galgame_favorite" json:"galgame_id"`
	UserID    int `gorm:"column:user_id;not null;uniqueIndex:idx_galgame_favorite" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameFavorite) TableName() string { return "galgame_favorite" }

type GalgameRedirect struct {
	OldGID  int       `gorm:"column:old_gid;primaryKey" json:"old_gid"`
	NewGID  int       `gorm:"column:new_gid;not null" json:"new_gid"`
	Created time.Time `gorm:"column:created" json:"created"`
}

func (GalgameRedirect) TableName() string { return "galgame_redirect" }
