package model

import "time"

type RolePermissionOverride struct {
	Role       string    `gorm:"column:role;primaryKey" json:"role"`
	Permission string    `gorm:"column:permission;primaryKey" json:"permission"`
	Effect     string    `gorm:"column:effect;not null" json:"effect"`
	UpdatedBy  int       `gorm:"column:updated_by;not null;default:0" json:"updated_by"`
	UpdatedAt  time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (RolePermissionOverride) TableName() string { return "role_permission_override" }

type RoleReplacement struct {
	Role string
	Rows []RolePermissionOverride
}
