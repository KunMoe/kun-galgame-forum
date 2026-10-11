package repository

import (
	"context"
	"encoding/json"
	"errors"

	"kun-galgame-api/internal/user/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type StateRepository struct {
	db *gorm.DB
}

func NewStateRepository(db *gorm.DB) *StateRepository {
	return &StateRepository{db: db}
}

func (r *StateRepository) Ensure(userID int) error {
	if userID <= 0 {
		return errors.New("invalid userID")
	}
	row := model.KungalUserState{UserID: userID, Moemoepoint: 7}
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
}

func (r *StateRepository) FindByID(userID int) (*model.KungalUserState, error) {
	var s model.KungalUserState
	err := r.db.First(&s, "user_id = ?", userID).Error
	return &s, err
}

func (r *StateRepository) LockForUpdate(tx *gorm.DB, userID int) (*model.KungalUserState, error) {
	var s model.KungalUserState
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).
		First(&s).Error
	return &s, err
}

func (r *StateRepository) CheckIn(userID int) (bool, error) {
	res := r.db.Model(&model.KungalUserState{}).
		Where("user_id = ? AND daily_check_in = 0", userID).
		Update("daily_check_in", 1)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *StateRepository) ResetDailyCheckIn(userID int) error {
	return r.db.Model(&model.KungalUserState{}).
		Where("user_id = ?", userID).
		Update("daily_check_in", 0).Error
}

func (r *StateRepository) SetMoemoepoint(userID, balance int) error {
	return r.db.Exec(
		`UPDATE kungal_user_state SET moemoepoint = ? WHERE user_id = ?`,
		balance, userID,
	).Error
}

func (r *StateRepository) SetMutedType(userID int, key string, muted bool) ([]string, error) {
	set := `CASE WHEN jsonb_exists(muted_notification_types, ?) THEN muted_notification_types
		ELSE muted_notification_types || to_jsonb(?::text) END`
	args := []any{key, key}
	if !muted {
		set = `muted_notification_types - ?::text`
		args = []any{key}
	}
	var raw string
	err := r.db.Raw(
		`UPDATE kungal_user_state SET muted_notification_types = `+set+`
		WHERE user_id = ? RETURNING muted_notification_types::text`,
		append(args, userID)...,
	).Row().Scan(&raw)
	if err != nil {
		return nil, err
	}
	keys := []string{}
	return keys, json.Unmarshal([]byte(raw), &keys)
}

func (r *StateRepository) UpdateMutedTypes(userID int, keys []string) error {
	if keys == nil {
		keys = []string{}
	}
	data, err := json.Marshal(keys)
	if err != nil {
		return err
	}
	return r.db.Exec(
		`UPDATE kungal_user_state SET muted_notification_types = ?::jsonb WHERE user_id = ?`,
		string(data), userID,
	).Error
}

func (r *StateRepository) ProfileSignInRequired(ctx context.Context, userID int) (bool, error) {
	var required bool
	err := r.db.WithContext(ctx).Raw(
		`SELECT is_profile_sign_in_required FROM kungal_user_state WHERE user_id = ?`, userID,
	).Scan(&required).Error
	return required, err
}

func (r *StateRepository) SetProfileSignInRequired(userID int, required bool) error {
	return r.db.Exec(
		`UPDATE kungal_user_state SET is_profile_sign_in_required = ? WHERE user_id = ?`,
		required, userID,
	).Error
}
