// Package envconfig is the write side of the config store that pkg/config
// reads at startup: sealed rows, their audit, and how a value is shown.
package envconfig

import (
	"context"
	"errors"
	"time"

	"kun-galgame-api/pkg/config"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrStoreOff = errors.New(config.KEKEnv + " is not set, so the config store is off")

const (
	ActionSet    = "set"
	ActionDelete = "delete"
	ActionImport = "import"
)

type secretRow struct {
	Key       string    `gorm:"column:key;primaryKey"`
	Cipher    []byte    `gorm:"column:cipher"`
	UpdatedBy *int      `gorm:"column:updated_by"`
	Updated   time.Time `gorm:"column:updated"`
}

func (secretRow) TableName() string { return "config_secret" }

type AuditRow struct {
	ID       int64     `gorm:"column:id;primaryKey"`
	Key      string    `gorm:"column:key"`
	Action   string    `gorm:"column:action"`
	OldShown string    `gorm:"column:old_shown"`
	NewShown string    `gorm:"column:new_shown"`
	Note     string    `gorm:"column:note"`
	UserID   *int      `gorm:"column:user_id"`
	Created  time.Time `gorm:"column:created"`
}

func (AuditRow) TableName() string { return "config_secret_audit" }

type Stored struct {
	Key     string
	Value   string
	Updated time.Time
}

type Store struct {
	db  *gorm.DB
	key []byte
}

// New seals with key; a nil key leaves the store readable as empty and
// refuses every write.
func New(db *gorm.DB, key []byte) *Store {
	return &Store{db: db, key: key}
}

func (s *Store) Enabled() bool { return s != nil && s.db != nil && s.key != nil }

// Shown is how a value appears outside the store: a secret by its
// fingerprint and last characters, anything else as itself.
func (s *Store) Shown(key, value string) string {
	if value == "" || !config.IsSecret(key) {
		return value
	}
	return config.FingerprintWith(s.key, value) + " " + config.Hint(value)
}

func (s *Store) All(ctx context.Context) ([]Stored, error) {
	if !s.Enabled() {
		return nil, nil
	}
	var rows []secretRow
	if err := s.db.WithContext(ctx).Order("key").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]Stored, 0, len(rows))
	for _, r := range rows {
		plain, err := config.Open(s.key, r.Cipher)
		if err != nil {
			return nil, err
		}
		out = append(out, Stored{Key: r.Key, Value: plain, Updated: r.Updated})
	}
	return out, nil
}

// Set writes value and its audit row together. It reports whether anything
// changed: storing the value already held writes nothing.
func (s *Store) Set(ctx context.Context, key, value, note, action string, userID *int) (bool, error) {
	if !s.Enabled() {
		return false, ErrStoreOff
	}
	sealed, err := config.Seal(s.key, value)
	if err != nil {
		return false, err
	}
	changed := false
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		old, err := s.current(tx, key)
		if err != nil {
			return err
		}
		if old != nil && *old == value {
			return nil
		}
		now := time.Now()
		row := secretRow{Key: key, Cipher: sealed, UpdatedBy: userID, Updated: now}
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error; err != nil {
			return err
		}
		changed = true
		return tx.Create(&AuditRow{
			Key: key, Action: action, OldShown: s.Shown(key, deref(old)), NewShown: s.Shown(key, value),
			Note: note, UserID: userID, Created: now,
		}).Error
	})
	return changed, err
}

// Delete reports whether there was a row to delete.
func (s *Store) Delete(ctx context.Context, key, note string, userID *int) (bool, error) {
	if !s.Enabled() {
		return false, ErrStoreOff
	}
	deleted := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		old, err := s.current(tx, key)
		if err != nil || old == nil {
			return err
		}
		if err := tx.Where("key = ?", key).Delete(&secretRow{}).Error; err != nil {
			return err
		}
		deleted = true
		return tx.Create(&AuditRow{
			Key: key, Action: ActionDelete, OldShown: s.Shown(key, *old), Note: note, UserID: userID,
			Created: time.Now(),
		}).Error
	})
	return deleted, err
}

func (s *Store) Changes(ctx context.Context, offset, limit int) ([]AuditRow, int64, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&AuditRow{}).Count(&count).Error; err != nil {
		return nil, 0, err
	}
	var rows []AuditRow
	err := s.db.WithContext(ctx).Order("created DESC, id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, count, err
}

func (s *Store) current(tx *gorm.DB, key string) (*string, error) {
	var row secretRow
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("key = ?", key).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	plain, err := config.Open(s.key, row.Cipher)
	if err != nil {
		return nil, err
	}
	return &plain, nil
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
