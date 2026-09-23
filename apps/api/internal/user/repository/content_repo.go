package repository

import (
	"gorm.io/gorm"
)

type UserContentRepository struct {
	db *gorm.DB
}

func NewUserContentRepository(db *gorm.DB) *UserContentRepository {
	return &UserContentRepository{db: db}
}

type LikedPostRow struct {
	ID     int64 `gorm:"column:id"`
	PostID int64 `gorm:"column:post_id"`
}
