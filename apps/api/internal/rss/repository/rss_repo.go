package repository

import (
	"kun-galgame-api/internal/rss/dto"
	topicRepo "kun-galgame-api/internal/topic/repository"

	"gorm.io/gorm"
)

type RSSRepository struct {
	db *gorm.DB
}

func NewRSSRepository(db *gorm.DB) *RSSRepository {
	return &RSSRepository{db: db}
}

func (r *RSSRepository) FindRecentSFWTopics() []dto.TopicRSSItem {
	var topics []dto.TopicRSSItem
	r.db.Table("topic t").
		Select(`t.id, t.title, SUBSTRING(t.content, 1, 233) AS description,
			t.user_id, t.created`).
		Where("t.status != 1 AND t.is_nsfw = false").
		Where(topicRepo.SharedListPredicate("t", false)).
		Order("t.created DESC").
		Limit(10).
		Find(&topics)
	return topics
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
