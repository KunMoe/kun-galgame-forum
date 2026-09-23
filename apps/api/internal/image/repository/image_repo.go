package repository

import (
	"context"

	userModel "kun-galgame-api/internal/user/model"

	"gorm.io/gorm"
)

type ImageRepository struct {
	db *gorm.DB
}

func NewImageRepository(db *gorm.DB) *ImageRepository {
	return &ImageRepository{db: db}
}

func (r *ImageRepository) ReserveDaily(ctx context.Context, userID, limit int) (bool, error) {
	res := r.db.WithContext(ctx).Model(&userModel.KungalUserState{}).
		Where("user_id = ? AND daily_image_count < ?", userID, limit).
		Update("daily_image_count", gorm.Expr("daily_image_count + 1"))
	return res.RowsAffected == 1, res.Error
}

func (r *ImageRepository) RefundDaily(ctx context.Context, userID int) error {
	return r.db.WithContext(ctx).Model(&userModel.KungalUserState{}).
		Where("user_id = ? AND daily_image_count > 0", userID).
		Update("daily_image_count", gorm.Expr("daily_image_count - 1")).Error
}
