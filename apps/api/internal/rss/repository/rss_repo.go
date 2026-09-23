package repository

import (
	"gorm.io/gorm"
)

type RSSRepository struct {
	db *gorm.DB
}

func NewRSSRepository(db *gorm.DB) *RSSRepository {
	return &RSSRepository{db: db}
}

type RecentGalgameRow struct {
	ID            int    `gorm:"column:id"`
	Created       string `gorm:"column:created"`
	CreatorUserID *int   `gorm:"column:creator_user_id"`
}

func (r *RSSRepository) FindRecentWorkIDs(limit int) []RecentGalgameRow {
	var rows []RecentGalgameRow
	r.db.Table("galgame").
		Select("id, created, creator_user_id").
		Where("published").
		Order("created DESC").
		Limit(limit).
		Scan(&rows)
	return rows
}
