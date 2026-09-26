package repository

import (
	"time"

	"gorm.io/gorm"
)

type MoemoepointPathRepository struct {
	db *gorm.DB
}

func NewMoemoepointPathRepository(db *gorm.DB) *MoemoepointPathRepository {
	return &MoemoepointPathRepository{db: db}
}

type TopicReplyRef struct {
	TopicID int64
	Floor   int
}

type GalgameRenumber struct {
	NewID   int64
	Created time.Time
}

func (r *MoemoepointPathRepository) TopicReplies(ids []int64) (map[int64]TopicReplyRef, error) {
	out := map[int64]TopicReplyRef{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		ID      int64 `gorm:"column:id"`
		TopicID int64 `gorm:"column:topic_id"`
		Floor   int   `gorm:"column:floor"`
	}
	if err := r.db.Table("topic_reply").Select("id, topic_id, floor").
		Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.ID] = TopicReplyRef{TopicID: row.TopicID, Floor: row.Floor}
	}
	return out, nil
}

func (r *MoemoepointPathRepository) TopicUpvoteTopics(ids []int64) (map[int64]int64, error) {
	out := map[int64]int64{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		ID      int64 `gorm:"column:id"`
		TopicID int64 `gorm:"column:topic_id"`
	}
	if err := r.db.Table("topic_upvote").Select("id, topic_id").
		Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.ID] = row.TopicID
	}
	return out, nil
}

func (r *MoemoepointPathRepository) GalgameRenumbers(oldIDs []int64) (map[int64]GalgameRenumber, error) {
	out := map[int64]GalgameRenumber{}
	if len(oldIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		OldID   int64     `gorm:"column:old_id"`
		NewID   int64     `gorm:"column:new_id"`
		Created time.Time `gorm:"column:created"`
	}
	if err := r.db.Table("galgame_renumber_2026").Select("old_id, new_id, created").
		Where("old_id IN ?", oldIDs).Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.OldID] = GalgameRenumber{NewID: row.NewID, Created: row.Created}
	}
	return out, nil
}
