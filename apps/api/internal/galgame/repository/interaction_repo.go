package repository

import (
	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GalgameInteractionRepository struct {
	db *gorm.DB
}

func NewGalgameInteractionRepository(db *gorm.DB) *GalgameInteractionRepository {
	return &GalgameInteractionRepository{db: db}
}

// Only likes are still stored here. Whether a person has favourited a work is
// a question about their catalog folders now, and the answer comes from
// /v2/me/folders?contains_work_id= with their own token.
func (r *GalgameInteractionRepository) UserLiked(userID, galgameID int) bool {
	if userID <= 0 {
		return false
	}
	var lc int64
	r.db.Model(&model.GalgameLike{}).
		Where("user_id = ? AND galgame_id = ?", userID, galgameID).Count(&lc)
	return lc > 0
}

func (r *GalgameInteractionRepository) UserLikedGalgames(userID int) []int {
	liked := []int{}
	if userID <= 0 {
		return liked
	}
	r.db.Model(&model.GalgameLike{}).
		Where("user_id = ?", userID).Pluck("galgame_id", &liked)
	return liked
}

func (r *GalgameInteractionRepository) ToggleLike(tx *gorm.DB, userID, galgameID int) (bool, error) {
	var existing model.GalgameLike
	result := tx.Where("user_id = ? AND galgame_id = ?", userID, galgameID).First(&existing)

	if result.Error == gorm.ErrRecordNotFound {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&model.GalgameLocal{ID: galgameID}).Error; err != nil {
			return false, err
		}
		if err := tx.Create(&model.GalgameLike{UserID: userID, GalgameID: galgameID}).Error; err != nil {
			return false, err
		}
		return true, tx.Model(&model.GalgameLocal{}).Where("id = ?", galgameID).
			Update("like_count", gorm.Expr("like_count + 1")).Error
	}
	if result.Error != nil {
		return false, result.Error
	}

	if err := tx.Delete(&existing).Error; err != nil {
		return false, err
	}
	return false, tx.Model(&model.GalgameLocal{}).Where("id = ?", galgameID).
		Update("like_count", gorm.Expr("like_count - 1")).Error
}
