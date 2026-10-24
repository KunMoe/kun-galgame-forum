// Package setting holds the values admins change at runtime. A key without a
// row reads as its default.
package setting

import (
	"encoding/json"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	TopicAIClassifyEnabled          = "topic_ai.classify_enabled"
	TopicAISummaryEnabled           = "topic_ai.summary_enabled"
	TopicAIModel                    = "topic_ai.model"
	TopicAITimeoutMs                = "topic_ai.timeout_ms"
	TopicSummaryMinLength           = "topic_summary.min_length"
	TopicSummaryRegenerateThreshold = "topic_summary.regenerate_threshold"
)

var defaults = map[string]any{
	TopicAIClassifyEnabled:          false,
	TopicAISummaryEnabled:           false,
	TopicAIModel:                    "@cf/deepseek-ai/deepseek-v4.1-flash-1017",
	TopicAITimeoutMs:                10000,
	TopicSummaryMinLength:           500,
	TopicSummaryRegenerateThreshold: 300,
}

const ttl = 30 * time.Second

type row struct {
	Key       string          `gorm:"column:key;primaryKey"`
	Value     json.RawMessage `gorm:"column:value;type:jsonb"`
	UpdatedBy *int            `gorm:"column:updated_by"`
	Updated   time.Time       `gorm:"column:updated"`
}

func (row) TableName() string { return "site_setting" }

type Store struct {
	db *gorm.DB

	mu       sync.Mutex
	values   map[string]json.RawMessage
	loadedAt time.Time
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) load() map[string]json.RawMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.values != nil && time.Since(s.loadedAt) < ttl {
		return s.values
	}
	var rows []row
	values := map[string]json.RawMessage{}
	if s.db != nil && s.db.Find(&rows).Error == nil {
		for _, r := range rows {
			values[r.Key] = r.Value
		}
	}
	s.values, s.loadedAt = values, time.Now()
	return values
}

func get[T any](s *Store, key string) T {
	def, _ := defaults[key].(T)
	if s == nil {
		return def
	}
	raw, ok := s.load()[key]
	if !ok {
		return def
	}
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		return def
	}
	return v
}

func (s *Store) Bool(key string) bool     { return get[bool](s, key) }
func (s *Store) Int(key string) int       { return get[int](s, key) }
func (s *Store) String(key string) string { return get[string](s, key) }

// Set writes the given keys in one transaction. Values are not validated
// here; the API checks them against its schema.
func (s *Store) Set(values map[string]any, userID int) error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		for key, v := range values {
			raw, err := json.Marshal(v)
			if err != nil {
				return err
			}
			uid := userID
			r := row{Key: key, Value: raw, UpdatedBy: &uid, Updated: time.Now()}
			if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&r).Error; err != nil {
				return err
			}
		}
		return nil
	})
	s.mu.Lock()
	s.values = nil
	s.mu.Unlock()
	return err
}
