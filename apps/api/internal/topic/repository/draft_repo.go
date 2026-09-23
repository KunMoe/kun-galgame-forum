package repository

import (
	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
)

type TopicDraftRepository struct {
	db *gorm.DB
}

func NewTopicDraftRepository(db *gorm.DB) *TopicDraftRepository {
	return &TopicDraftRepository{db: db}
}

func (r *TopicDraftRepository) CountByUser(userID int) (int64, error) {
	var n int64
	err := r.db.Model(&model.TopicDraft{}).Where("user_id = ?", userID).Count(&n).Error
	return n, err
}

func (r *TopicDraftRepository) Create(d *model.TopicDraft) error {
	return r.db.Create(d).Error
}

func (r *TopicDraftRepository) GetByIDForUser(id, userID int) (*model.TopicDraft, error) {
	var d model.TopicDraft
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *TopicDraftRepository) DeleteForUser(id, userID int) (int64, error) {
	res := r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&model.TopicDraft{})
	return res.RowsAffected, res.Error
}
