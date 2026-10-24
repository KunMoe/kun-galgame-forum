package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrRequestNotFound = errors.New("topic section request: not found")

type SectionChoiceRow struct {
	Name        string `gorm:"column:name"`
	Label       string `gorm:"column:label"`
	Description string `gorm:"column:description"`
	IsArchived  bool   `gorm:"column:is_archived"`
}

type TopicClassification struct {
	TopicID int       `gorm:"column:topic_id;primaryKey"`
	Section string    `gorm:"column:section"`
	Source  string    `gorm:"column:source"`
	Model   string    `gorm:"column:model"`
	Reason  string    `gorm:"column:reason"`
	Created time.Time `gorm:"column:created"`
	Updated time.Time `gorm:"column:updated"`
}

func (TopicClassification) TableName() string { return "topic_classification" }

type TopicSectionRequest struct {
	ID          int        `gorm:"column:id;primaryKey;autoIncrement"`
	TopicID     int        `gorm:"column:topic_id"`
	RequestedBy *int       `gorm:"column:requested_by"`
	FromSection string     `gorm:"column:from_section"`
	ToSection   string     `gorm:"column:to_section"`
	Reason      string     `gorm:"column:reason"`
	State       string     `gorm:"column:state"`
	DecidedBy   *int       `gorm:"column:decided_by"`
	DecidedAt   *time.Time `gorm:"column:decided_at"`
	Created     time.Time  `gorm:"column:created"`
}

func (TopicSectionRequest) TableName() string { return "topic_section_request" }

type AIRepository struct {
	db *gorm.DB
}

func NewAIRepository(db *gorm.DB) *AIRepository {
	return &AIRepository{db: db}
}

func (r *AIRepository) DB() *gorm.DB { return r.db }

func (r *AIRepository) SectionChoices() ([]SectionChoiceRow, error) {
	var rows []SectionChoiceRow
	err := r.db.Raw(`SELECT ts.name, ts.label, ts.description, ts.is_archived
		FROM topic_section ts JOIN topic_section_group g ON g.id = ts.group_id
		ORDER BY g.position, g.id, ts.position, ts.id`).Scan(&rows).Error
	return rows, err
}

func (r *AIRepository) SaveClassification(tx *gorm.DB, c TopicClassification) error {
	now := time.Now()
	c.Created, c.Updated = now, now
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "topic_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"section", "source", "model", "reason", "updated"}),
	}).Create(&c).Error
}

func (r *AIRepository) FindClassification(topicID int) (*TopicClassification, error) {
	var c TopicClassification
	err := r.db.Where("topic_id = ?", topicID).Take(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &c, err
}

type PendingTopic struct {
	ID      int    `gorm:"column:id"`
	UserID  int    `gorm:"column:user_id"`
	Title   string `gorm:"column:title"`
	Content string `gorm:"column:content"`
	Section string `gorm:"column:section"`
}

const pendingTopicSelect = `SELECT t.id, t.user_id, t.title, t.content,
	COALESCE((SELECT ts.name FROM topic_section_relation r JOIN topic_section ts ON ts.id = r.topic_section_id WHERE r.topic_id = t.id), '') AS section
	FROM topic t`

// UnverifiedTopics are topics published while the model was unreachable.
func (r *AIRepository) UnverifiedTopics(olderThan time.Time, limit int) ([]PendingTopic, error) {
	var rows []PendingTopic
	err := r.db.Raw(pendingTopicSelect+`
		JOIN topic_classification c ON c.topic_id = t.id
		WHERE c.source = 'fallback' AND c.created < ?
		ORDER BY c.created LIMIT ?`, olderThan, limit).Scan(&rows).Error
	return rows, err
}

// TopicsNeedingSummary are recent topics with a body of at least minBytes and
// no summary, or whose edits drifted past the threshold. The rune count is
// checked by the caller.
func (r *AIRepository) TopicsNeedingSummary(since time.Time, minBytes, drift, limit int) ([]PendingTopic, error) {
	var rows []PendingTopic
	err := r.db.Raw(pendingTopicSelect+`
		WHERE t.status = 0 AND octet_length(t.content) >= ?
			AND ((t.ai_summary IS NULL AND t.ai_summary_at IS NULL AND t.created >= ?) OR t.ai_summary_drift >= ?)
		ORDER BY t.id DESC LIMIT ?`, minBytes, since, drift, limit).Scan(&rows).Error
	return rows, err
}

// SetSummary writes the summary, or records an attempt with a nil one so the
// sweep does not retry a body the model would not summarise.
func (r *AIRepository) SetSummary(topicID int, summary *string, model string) error {
	return r.db.Table("topic").Where("id = ?", topicID).Updates(map[string]any{
		"ai_summary":       summary,
		"ai_summary_model": model,
		"ai_summary_at":    time.Now(),
		"ai_summary_drift": 0,
	}).Error
}

func (r *AIRepository) AddSummaryDrift(tx *gorm.DB, topicID, runes int) error {
	return tx.Table("topic").Where("id = ?", topicID).
		UpdateColumn("ai_summary_drift", gorm.Expr("ai_summary_drift + ?", runes)).Error
}

func (r *AIRepository) HasPendingRequest(tx *gorm.DB, topicID int) (bool, error) {
	var n int64
	err := tx.Model(&TopicSectionRequest{}).Where("topic_id = ? AND state = 'pending'", topicID).Count(&n).Error
	return n > 0, err
}

func (r *AIRepository) CreateRequest(tx *gorm.DB, req *TopicSectionRequest) error {
	req.State = "pending"
	req.Created = time.Now()
	return tx.Create(req).Error
}

func (r *AIRepository) TopicRequests(topicID, limit int) ([]TopicSectionRequest, error) {
	var rows []TopicSectionRequest
	err := r.db.Where("topic_id = ?", topicID).Order("id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *AIRepository) Requests(state string, beforeID, limit int) ([]TopicSectionRequest, error) {
	q := r.db.Where("state = ?", state)
	if beforeID > 0 {
		q = q.Where("id < ?", beforeID)
	}
	var rows []TopicSectionRequest
	err := q.Order("id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *AIRepository) FindRequest(id int) (*TopicSectionRequest, error) {
	var req TopicSectionRequest
	err := r.db.Where("id = ?", id).Take(&req).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRequestNotFound
	}
	return &req, err
}

func (r *AIRepository) LockRequest(tx *gorm.DB, id int) (*TopicSectionRequest, error) {
	var req TopicSectionRequest
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).Take(&req).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRequestNotFound
	}
	return &req, err
}

func (r *AIRepository) DecideRequest(tx *gorm.DB, id int, state string, by int) error {
	now := time.Now()
	return tx.Model(&TopicSectionRequest{}).Where("id = ?", id).Updates(map[string]any{
		"state": state, "decided_by": by, "decided_at": now,
	}).Error
}
