package repository

import (
	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TopicTaxonomyRepository struct {
	db *gorm.DB
}

func NewTopicTaxonomyRepository(db *gorm.DB) *TopicTaxonomyRepository {
	return &TopicTaxonomyRepository{db: db}
}

func (r *TopicTaxonomyRepository) ReplaceSectionRelations(tx *gorm.DB, topicID int, sectionIDs []int) error {
	if err := tx.Where("topic_id = ?", topicID).Delete(&model.TopicSectionRelation{}).Error; err != nil {
		return err
	}
	for i, sID := range sectionIDs {
		rel := model.TopicSectionRelation{TopicID: topicID, TopicSectionID: sID, Position: i}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rel).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *TopicTaxonomyRepository) FindSectionNamesByTopicIDs(topicIDs []int) (map[int][]string, error) {
	type row struct {
		TopicID int
		Name    string
	}
	var rows []row
	err := r.db.Table("topic_section_relation").
		Select("topic_section_relation.topic_id, topic_section.name").
		Joins("JOIN topic_section ON topic_section.id = topic_section_relation.topic_section_id").
		Where("topic_section_relation.topic_id IN ?", topicIDs).
		Order("topic_section_relation.topic_id, topic_section_relation.position, topic_section_relation.topic_section_id").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int][]string)
	for _, r := range rows {
		result[r.TopicID] = append(result[r.TopicID], r.Name)
	}
	return result, nil
}

func (r *TopicTaxonomyRepository) FindSectionNamesByTopicID(topicID int) ([]string, error) {
	var names []string
	err := r.db.Table("topic_section_relation").
		Select("topic_section.name").
		Joins("JOIN topic_section ON topic_section.id = topic_section_relation.topic_section_id").
		Where("topic_section_relation.topic_id = ?", topicID).
		Order("topic_section_relation.position, topic_section_relation.topic_section_id").
		Pluck("topic_section.name", &names).Error
	return names, err
}

func (r *TopicTaxonomyRepository) FindSectionsByNamesTx(tx *gorm.DB, names []string) ([]model.TopicSection, error) {
	var sections []model.TopicSection
	err := tx.Where("name IN ?", names).Find(&sections).Error
	return sections, err
}

type SectionRule struct {
	ID               int    `gorm:"column:id"`
	Name             string `gorm:"column:name"`
	IsArchived       bool   `gorm:"column:is_archived"`
	CostsMoemoepoint bool   `gorm:"column:costs_moemoepoint"`
	LegacyCategory   string `gorm:"column:legacy_category"`
}

func (r *TopicTaxonomyRepository) FindSectionRules(db *gorm.DB, names []string) (map[string]SectionRule, error) {
	if db == nil {
		db = r.db
	}
	var rows []SectionRule
	err := db.Table("topic_section ts").
		Select("ts.id, ts.name, ts.is_archived, ts.costs_moemoepoint, g.legacy_category").
		Joins("JOIN topic_section_group g ON g.id = ts.group_id").
		Where("ts.name IN ?", names).
		Scan(&rows).Error
	out := make(map[string]SectionRule, len(rows))
	for _, row := range rows {
		out[row.Name] = row
	}
	return out, err
}
