package repository

import (
	"context"
	"time"

	"kun-galgame-api/internal/admin/model"

	"gorm.io/gorm"
)

type RolePermissionRepository struct {
	db *gorm.DB
}

func NewRolePermissionRepository(db *gorm.DB) *RolePermissionRepository {
	return &RolePermissionRepository{db: db}
}

func (r *RolePermissionRepository) ListAll(ctx context.Context) ([]model.RolePermissionOverride, error) {
	var rows []model.RolePermissionOverride
	err := r.db.WithContext(ctx).Order("role ASC, permission ASC").Find(&rows).Error
	return rows, err
}

// Validation reads rows of more than one role (moderator ⊆ admin), so the read,
// the judgement and the write share one lock or two writers can break it.
func (r *RolePermissionRepository) Replace(ctx context.Context, operatorUID int, plan func([]model.RolePermissionOverride) ([]model.RoleReplacement, error)) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext('role_permission_override'))").Error; err != nil {
			return err
		}
		var current []model.RolePermissionOverride
		if err := tx.Order("role ASC, permission ASC").Find(&current).Error; err != nil {
			return err
		}
		next, err := plan(current)
		if err != nil {
			return err
		}
		now := time.Now()
		for _, rep := range next {
			var before []model.RolePermissionOverride
			for _, row := range current {
				if row.Role == rep.Role {
					before = append(before, row)
				}
			}
			if err := tx.Where("role = ?", rep.Role).Delete(&model.RolePermissionOverride{}).Error; err != nil {
				return err
			}
			for i := range rep.Rows {
				rep.Rows[i].Role = rep.Role
				rep.Rows[i].UpdatedBy = operatorUID
				rep.Rows[i].UpdatedAt = now
			}
			if len(rep.Rows) > 0 {
				if err := tx.Create(&rep.Rows).Error; err != nil {
					return err
				}
			}
			if err := writeAudit(tx, operatorUID, "role", rep.Role, roleRowsToDeltas(before), roleRowsToDeltas(rep.Rows)); err != nil {
				return err
			}
		}
		return nil
	})
}

func roleRowsToDeltas(rows []model.RolePermissionOverride) []model.AuditDelta {
	out := make([]model.AuditDelta, 0, len(rows))
	for _, r := range rows {
		out = append(out, model.AuditDelta{Permission: r.Permission, Effect: r.Effect})
	}
	return out
}
