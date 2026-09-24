package repository

import (
	"time"

	"kun-galgame-api/internal/infrastructure/viewstats"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/pkg/miniapp"

	"gorm.io/gorm"
)

type TopicRepository struct {
	db *gorm.DB
}

func NewTopicRepository(db *gorm.DB) *TopicRepository {
	return &TopicRepository{db: db}
}

func (r *TopicRepository) DB() *gorm.DB {
	return r.db
}

func (r *TopicRepository) FindByID(id int) (*model.Topic, error) {
	var topic model.Topic
	err := r.db.First(&topic, id).Error
	return &topic, err
}

func (r *TopicRepository) UpdateFields(id int, fields map[string]any) error {
	return r.db.Model(&model.Topic{}).Where("id = ?", id).Updates(fields).Error
}

func (r *TopicRepository) IncrementView(id int) error {
	if err := r.db.Model(&model.Topic{}).Where("id = ?", id).
		Update("view", gorm.Expr("view + 1")).Error; err != nil {
		return err
	}
	return viewstats.BumpDaily(r.db, viewstats.TopicDaily, id)
}

func (r *TopicRepository) HasUserFavorited(userID, topicID int) (bool, error) {
	var count int64
	err := r.db.Model(&model.TopicFavorite{}).Where("user_id = ? AND topic_id = ?", userID, topicID).Count(&count).Error
	return count > 0, err
}

func (r *TopicRepository) HasUserUpvoted(userID, topicID int) (bool, error) {
	var count int64
	err := r.db.Model(&model.TopicUpvote{}).Where("user_id = ? AND topic_id = ?", userID, topicID).Count(&count).Error
	return count > 0, err
}

func (r *TopicRepository) CountTodayTopicsByUser(tx *gorm.DB, userID int) (int64, error) {
	var count int64
	oneDayAgo := time.Now().Add(-24 * time.Hour)
	err := tx.Model(&model.Topic{}).
		Where("user_id = ? AND created >= ?", userID, oneDayAgo).
		Count(&count).Error
	return count, err
}

func (r *TopicRepository) LookupMiniApps(topicIDs []int) (map[int][]string, error) {
	return miniapp.Lookup(r.db, topicIDs)
}

func (r *TopicRepository) CreateTopic(tx *gorm.DB, topic *model.Topic) error {
	return tx.Create(topic).Error
}

func (r *TopicRepository) UpdateTopicFields(tx *gorm.DB, topicID int, fields map[string]any) error {
	return tx.Model(&model.Topic{}).Where("id = ?", topicID).Updates(fields).Error
}

func (r *TopicRepository) TouchStatusUpdateTime(tx *gorm.DB, topicID int, t time.Time) error {
	return tx.Model(&model.Topic{}).
		Where("id = ? AND created > ?", topicID, model.BumpCutoff(t)).
		Updates(map[string]any{"status_update_time": t}).Error
}

func (r *TopicRepository) ApplyUpvoteCountAndTime(tx *gorm.DB, topicID int, t time.Time) error {
	return tx.Model(&model.Topic{}).Where("id = ?", topicID).Updates(map[string]any{
		"upvote_count": gorm.Expr("upvote_count + 1"),
		"upvote_time":  &t,
		"status_update_time": gorm.Expr(
			"CASE WHEN created > ? THEN ? ELSE status_update_time END", model.BumpCutoff(t), t),
	}).Error
}
