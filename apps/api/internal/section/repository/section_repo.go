package repository

import (
	"errors"
	"time"

	"kun-galgame-api/internal/topic/model"
	topicRepo "kun-galgame-api/internal/topic/repository"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("section: not found")

type SectionRepository struct {
	db *gorm.DB
}

func NewSectionRepository(db *gorm.DB) *SectionRepository {
	return &SectionRepository{db: db}
}

type SectionRow struct {
	model.TopicSection
	GroupKey       string `gorm:"column:group_key"`
	GroupLabel     string `gorm:"column:group_label"`
	GroupPosition  int    `gorm:"column:group_position"`
	LegacyCategory string `gorm:"column:legacy_category"`
	TopicCount     int64  `gorm:"column:topic_count"`
	ViewCount      int64  `gorm:"column:view_count"`
}

type LatestTopicRow struct {
	ID      int       `gorm:"column:id"`
	Title   string    `gorm:"column:title"`
	Created time.Time `gorm:"column:created"`
	UserID  int       `gorm:"column:user_id"`
}

// List returns every section in display order, counting the published topics
// an anonymous reader can open. legacyCategory, when set, keeps only the
// groups that map to it.
func (r *SectionRepository) List(legacyCategory string) ([]SectionRow, error) {
	q := r.db.Table("topic_section ts").
		Select(`ts.*, g.key AS group_key, g.label AS group_label, g.position AS group_position,
			g.legacy_category, COUNT(t.id) AS topic_count, COALESCE(SUM(t.view), 0) AS view_count`).
		Joins("JOIN topic_section_group g ON g.id = ts.group_id").
		Joins("LEFT JOIN topic_section_relation tsr ON tsr.topic_section_id = ts.id").
		Joins("LEFT JOIN topic t ON t.id = tsr.topic_id AND t.status != 1 AND " + topicRepo.SharedListPredicate("t", false)).
		Group("ts.id, g.id").
		Order("g.position, g.id, ts.position, ts.id")
	if legacyCategory != "" {
		q = q.Where("g.legacy_category = ?", legacyCategory)
	}
	var rows []SectionRow
	err := q.Scan(&rows).Error
	return rows, err
}

func (r *SectionRepository) LatestTopics(sectionID, limit int) ([]LatestTopicRow, error) {
	q := `
		SELECT t.id, t.title, t.created, t.user_id
		FROM topic t
		JOIN topic_section_relation tsr ON tsr.topic_id = t.id
		WHERE tsr.topic_section_id = ? AND t.status != 1
			AND ` + topicRepo.SharedListPredicate("t", false) + `
		ORDER BY t.created DESC, t.id DESC
		LIMIT ?`
	var rows []LatestTopicRow
	err := r.db.Raw(q, sectionID, limit).Scan(&rows).Error
	return rows, err
}

type SectionChange struct {
	Label       *string
	Description *string
	Icon        *string
	Color       *string
	IsArchived  *bool
}

func (r *SectionRepository) UpdateSection(name string, c SectionChange) error {
	set := map[string]any{}
	if c.Label != nil {
		set["label"] = *c.Label
	}
	if c.Description != nil {
		set["description"] = *c.Description
	}
	if c.Icon != nil {
		set["icon"] = *c.Icon
	}
	if c.Color != nil {
		set["color"] = *c.Color
	}
	if c.IsArchived != nil {
		set["is_archived"] = *c.IsArchived
	}
	return r.update(&model.TopicSection{}, "name = ?", name, set)
}

func (r *SectionRepository) UpdateGroupLabel(key, label string) error {
	return r.update(&model.TopicSectionGroup{}, "key = ?", key, map[string]any{"label": label})
}

func (r *SectionRepository) update(m any, where, key string, set map[string]any) error {
	var n int64
	if err := r.db.Model(m).Where(where, key).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	if len(set) == 0 {
		return nil
	}
	set["updated"] = time.Now()
	return r.db.Model(m).Where(where, key).Updates(set).Error
}

type GroupOrder struct {
	Key      string
	Sections []string
}

// Reorder writes the order of the groups and of the sections inside each
// group. It does not move a section between groups.
func (r *SectionRepository) Reorder(order []GroupOrder) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		for gi, g := range order {
			if err := tx.Model(&model.TopicSectionGroup{}).Where("key = ?", g.Key).
				Updates(map[string]any{"position": gi, "updated": now}).Error; err != nil {
				return err
			}
			for si, name := range g.Sections {
				if err := tx.Model(&model.TopicSection{}).Where("name = ?", name).
					Updates(map[string]any{"position": si, "updated": now}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
