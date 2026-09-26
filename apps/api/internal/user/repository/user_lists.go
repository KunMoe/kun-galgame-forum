package repository

import (
	"time"

	topicRepo "kun-galgame-api/internal/topic/repository"

	"gorm.io/gorm"
)

type UserListQuery struct {
	OwnerID       int
	Relation      string
	IncludeNSFW   bool
	Authenticated bool
	Offset        int
	Limit         int
}

type UserTopicRow struct {
	ID      int       `gorm:"column:id"`
	Title   string    `gorm:"column:title"`
	Created time.Time `gorm:"column:created"`
}

type UserReplyRow struct {
	ID          int       `gorm:"column:id"`
	UserID      int       `gorm:"column:user_id"`
	TopicID     int       `gorm:"column:topic_id"`
	TopicTitle  string    `gorm:"column:topic_title"`
	TopicUserID int       `gorm:"column:topic_user_id"`
	Floor       int       `gorm:"column:floor"`
	Content     string    `gorm:"column:content"`
	Created     time.Time `gorm:"column:created"`
}

type UserCommentRow struct {
	ID          int       `gorm:"column:id"`
	UserID      int       `gorm:"column:user_id"`
	TopicID     int       `gorm:"column:topic_id"`
	TopicTitle  string    `gorm:"column:topic_title"`
	TopicUserID int       `gorm:"column:topic_user_id"`
	Content     string    `gorm:"column:content"`
	Created     time.Time `gorm:"column:created"`
}

func (r *UserContentRepository) ListUserTopics(q UserListQuery) ([]UserTopicRow, int, error) {
	return pageScan[UserTopicRow](r.topicsQuery(q), "topic.id, topic.title, topic.created",
		"topic.created DESC, topic.id DESC", q.Offset, q.Limit)
}

func (r *UserContentRepository) ListUserReplies(q UserListQuery) ([]UserReplyRow, int, error) {
	return pageScan[UserReplyRow](r.repliesQuery(q),
		"topic_reply.id, topic_reply.user_id, topic_reply.topic_id, topic.title AS topic_title, topic.user_id AS topic_user_id, "+
			"topic_reply.floor, COALESCE(topic_reply.content, '') AS content, topic_reply.created",
		"topic_reply.created DESC, topic_reply.id DESC", q.Offset, q.Limit)
}

func (r *UserContentRepository) ListUserComments(q UserListQuery) ([]UserCommentRow, int, error) {
	return pageScan[UserCommentRow](r.commentsQuery(q),
		"topic_comment.id, topic_comment.user_id, topic_comment.topic_id, topic.title AS topic_title, topic.user_id AS topic_user_id, "+
			"topic_comment.content, topic_comment.created",
		"topic_comment.created DESC, topic_comment.id DESC", q.Offset, q.Limit)
}

func (r *UserContentRepository) topicsQuery(q UserListQuery) *gorm.DB {
	db := r.db.Table("topic")
	switch q.Relation {
	case "authored":
		db = db.Where("topic.user_id = ?", q.OwnerID)
		db = sharedVisibleTopic(db, q.Authenticated)
	case "liked":
		db = db.
			Joins("JOIN topic_reaction ON topic_reaction.topic_id = topic.id AND topic_reaction.reaction = 'like'").
			Where("topic_reaction.user_id = ?", q.OwnerID)
		db = sharedVisibleTopic(db, q.Authenticated)
	case "upvoted":
		db = db.
			Joins("JOIN topic_upvote ON topic_upvote.topic_id = topic.id").
			Where("topic_upvote.user_id = ?", q.OwnerID)
		db = sharedVisibleTopic(db, q.Authenticated)
	case "favorited":
		db = db.
			Joins("JOIN topic_favorite ON topic_favorite.topic_id = topic.id").
			Where("topic_favorite.user_id = ?", q.OwnerID)
		db = sharedVisibleTopic(db, q.Authenticated)
	case "hidden":
		db = db.Where("topic.user_id = ? AND topic.status = 1", q.OwnerID)
	default:
		db = db.Where("1 = 0")
	}
	if !q.IncludeNSFW {
		db = db.Where("topic.is_nsfw = false")
	}
	return db
}

func (r *UserContentRepository) repliesQuery(q UserListQuery) *gorm.DB {
	db := r.db.Table("topic_reply").
		Joins("JOIN topic ON topic.id = topic_reply.topic_id").
		Where("topic_reply.status = 0")
	db = sharedVisibleTopic(db, q.Authenticated)
	switch q.Relation {
	case "authored":
		db = db.Where("topic_reply.user_id = ?", q.OwnerID)
	case "received":
		db = db.Where("topic.user_id = ?", q.OwnerID)
	case "liked":
		db = db.
			Joins("JOIN topic_reply_reaction ON topic_reply_reaction.topic_reply_id = topic_reply.id AND topic_reply_reaction.reaction = 'like'").
			Where("topic_reply_reaction.user_id = ?", q.OwnerID)
	default:
		db = db.Where("1 = 0")
	}
	if !q.IncludeNSFW {
		db = db.Where("topic.is_nsfw = false")
	}
	return db
}

func (r *UserContentRepository) commentsQuery(q UserListQuery) *gorm.DB {
	db := r.db.Table("topic_comment").
		Joins("JOIN topic ON topic.id = topic_comment.topic_id").
		Where("topic_comment.status = 0")
	db = sharedVisibleTopic(db, q.Authenticated)
	switch q.Relation {
	case "authored":
		db = db.Where("topic_comment.user_id = ?", q.OwnerID)
	case "received":
		db = db.Where("topic_comment.target_user_id = ?", q.OwnerID)
	case "liked":
		db = db.
			Joins("JOIN topic_comment_like ON topic_comment_like.topic_comment_id = topic_comment.id").
			Where("topic_comment_like.user_id = ?", q.OwnerID)
	default:
		db = db.Where("1 = 0")
	}
	if !q.IncludeNSFW {
		db = db.Where("topic.is_nsfw = false")
	}
	return db
}

func sharedVisibleTopic(q *gorm.DB, authenticated bool) *gorm.DB {
	// Hidden titles used to leak through every non-hidden profile list.
	q = q.Where("topic.status != 1")
	// view_restricted must not widen a shared list.
	return q.Where(topicRepo.SharedListPredicate("topic", authenticated))
}

func pageScan[T any](q *gorm.DB, columns, order string, offset, limit int) ([]T, int, error) {
	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []T
	if err := q.Select(columns).Order(order).Offset(offset).Limit(limit).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	if rows == nil {
		rows = []T{}
	}
	return rows, int(total), nil
}
