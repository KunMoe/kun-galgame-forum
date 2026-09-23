package repository

import (
	"time"

	topicRepo "kun-galgame-api/internal/topic/repository"

	"gorm.io/gorm"
)

type SectionRepository struct {
	db *gorm.DB
}

func NewSectionRepository(db *gorm.DB) *SectionRepository {
	return &SectionRepository{db: db}
}

type SectionStatRow struct {
	SectionID   int    `gorm:"column:section_id"`
	SectionName string `gorm:"column:section_name"`
	TopicCount  int64  `gorm:"column:topic_count"`
	ViewCount   int64  `gorm:"column:view_count"`
}

type LatestTopicRow struct {
	ID      int       `gorm:"column:id"`
	Title   string    `gorm:"column:title"`
	Created time.Time `gorm:"column:created"`
	UserID  int       `gorm:"column:user_id"`
}

// sectionCategory maps a section to the topic category its name prefix files
// it under; a topic counts toward a section only when its own category agrees.
const sectionCategory = `CASE left(ts.name, 2) WHEN 'g-' THEN 'galgame' WHEN 't-' THEN 'technique' WHEN 'o-' THEN 'others' END`

// Stats counts every section, empty ones included, over published topics an
// anonymous reader can open.
func (r *SectionRepository) Stats(prefix string) ([]SectionStatRow, error) {
	q := `
		SELECT ts.id AS section_id, ts.name AS section_name,
			COUNT(t.id) AS topic_count, COALESCE(SUM(t.view), 0) AS view_count
		FROM topic_section ts
		LEFT JOIN topic_section_relation tsr ON tsr.topic_section_id = ts.id
		LEFT JOIN topic t ON t.id = tsr.topic_id AND t.status != 1
			AND ` + topicRepo.SharedListPredicate("t", false) + `
			AND t.category = ` + sectionCategory + `
		WHERE ts.name LIKE ?
		GROUP BY ts.id, ts.name
		ORDER BY ts.id`
	var rows []SectionStatRow
	err := r.db.Raw(q, prefix+"%").Scan(&rows).Error
	return rows, err
}

func (r *SectionRepository) LatestTopics(sectionID, limit int) ([]LatestTopicRow, error) {
	q := `
		SELECT t.id, t.title, t.created, t.user_id
		FROM topic t
		JOIN topic_section_relation tsr ON tsr.topic_id = t.id
		JOIN topic_section ts ON ts.id = tsr.topic_section_id
		WHERE ts.id = ? AND t.status != 1
			AND ` + topicRepo.SharedListPredicate("t", false) + `
			AND t.category = ` + sectionCategory + `
		ORDER BY t.created DESC, t.id DESC
		LIMIT ?`
	var rows []LatestTopicRow
	err := r.db.Raw(q, sectionID, limit).Scan(&rows).Error
	return rows, err
}
