package repository

import (
	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
)

type ReplyRepository struct {
	db *gorm.DB
}

func NewReplyRepository(db *gorm.DB) *ReplyRepository {
	return &ReplyRepository{db: db}
}

func (r *ReplyRepository) DB() *gorm.DB {
	return r.db
}

func (r *ReplyRepository) FindByID(id int) (*model.TopicReply, error) {
	var reply model.TopicReply
	err := r.db.First(&reply, id).Error
	return &reply, err
}

type ReplyRow struct {
	model.TopicReply
	UserName        string
	UserAvatar      string
	UserMoemoepoint int
}

func (r *ReplyRepository) FindRepliesByIDs(ids []int) ([]ReplyRow, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []ReplyRow
	err := r.db.Table("topic_reply").
		Select(`topic_reply.*`).
		Where("topic_reply.id IN ?", ids).
		Where("topic_reply.status = ?", 0).
		Find(&rows).Error
	return rows, err
}

func findInteractionStatus(db *gorm.DB, table, fkCol string, userID int, ids []int) (map[int]bool, error) {
	if len(ids) == 0 || userID == 0 {
		return make(map[int]bool), nil
	}
	var foundIDs []int
	err := db.Table(table).
		Where("user_id = ? AND "+fkCol+" IN ?", userID, ids).
		Pluck(fkCol, &foundIDs).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int]bool, len(foundIDs))
	for _, id := range foundIDs {
		result[id] = true
	}
	return result, nil
}

func (r *ReplyRepository) DeleteRepliesByIDs(tx *gorm.DB, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	if err := tx.Exec(
		"DELETE FROM topic_comment_like WHERE topic_comment_id IN (SELECT id FROM topic_comment WHERE topic_reply_id IN ?)",
		ids,
	).Error; err != nil {
		return err
	}
	for _, m := range []any{
		&model.TopicComment{}, &model.TopicReplyLike{}, &model.TopicReplyDislike{},
	} {
		if err := tx.Where("topic_reply_id IN ?", ids).Delete(m).Error; err != nil {
			return err
		}
	}

	return tx.Where("id IN ?", ids).Delete(&model.TopicReply{}).Error
}

func (r *ReplyRepository) SetStatus(id, status int) error {
	return r.db.Model(&model.TopicReply{}).Where("id = ?", id).Update("status", status).Error
}

func (r *ReplyRepository) CreateReply(tx *gorm.DB, reply *model.TopicReply) error {
	return tx.Create(reply).Error
}

func (r *ReplyRepository) UpdateReplyContent(tx *gorm.DB, replyID int, fields map[string]any) error {
	return tx.Model(&model.TopicReply{}).Where("id = ?", replyID).Updates(fields).Error
}
