package repository

import (
	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
)

type CommunityPostRepository struct {
	db *gorm.DB
}

func NewCommunityPostRepository(db *gorm.DB) *CommunityPostRepository {
	return &CommunityPostRepository{db: db}
}

func (r *CommunityPostRepository) FindMapByLegacyID(legacyID int) (*model.GalgameCommentCommunityMap, error) {
	var row model.GalgameCommentCommunityMap
	err := r.db.Where("old_comment_id = ?", legacyID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}
