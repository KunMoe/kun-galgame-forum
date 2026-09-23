package repository

import (
	"context"
	"strconv"
	"time"

	"kun-galgame-api/internal/admin/model"

	"gorm.io/gorm"
)

type UserPermissionRepository struct {
	db *gorm.DB
}

func NewUserPermissionRepository(db *gorm.DB) *UserPermissionRepository {
	return &UserPermissionRepository{db: db}
}

func (r *UserPermissionRepository) ListAll(ctx context.Context) ([]model.UserPermissionOverride, error) {
	var rows []model.UserPermissionOverride
	err := r.db.WithContext(ctx).Order("user_id ASC, permission ASC").Find(&rows).Error
	return rows, err
}

func (r *UserPermissionRepository) ListForUser(ctx context.Context, userID int) ([]model.UserPermissionOverride, error) {
	var rows []model.UserPermissionOverride
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("permission ASC").Find(&rows).Error
	return rows, err
}

func (r *UserPermissionRepository) Replace(ctx context.Context, userID, operatorUID int, plan func([]model.UserPermissionOverride) ([]model.UserPermissionOverride, error)) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext('user_permission_override'), ?)", userID).Error; err != nil {
			return err
		}
		var before []model.UserPermissionOverride
		if err := tx.Where("user_id = ?", userID).Order("permission ASC").Find(&before).Error; err != nil {
			return err
		}
		rows, err := plan(before)
		if err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&model.UserPermissionOverride{}).Error; err != nil {
			return err
		}
		now := time.Now()
		for i := range rows {
			rows[i].UserID = userID
			rows[i].UpdatedBy = operatorUID
			rows[i].UpdatedAt = now
		}
		if len(rows) > 0 {
			if err := tx.Create(&rows).Error; err != nil {
				return err
			}
		}
		return writeAudit(tx, operatorUID, "user", strconv.Itoa(userID),
			userRowsToDeltas(before), userRowsToDeltas(rows))
	})
}

func userRowsToDeltas(rows []model.UserPermissionOverride) []model.AuditDelta {
	out := make([]model.AuditDelta, 0, len(rows))
	for _, r := range rows {
		out = append(out, model.AuditDelta{Permission: r.Permission, Effect: r.Effect})
	}
	return out
}
