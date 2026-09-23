package repository

import (
	"time"

	topicRepo "kun-galgame-api/internal/topic/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type V1Query struct {
	Keywords      []string
	Offset        int
	Limit         int
	Authenticated bool
	IncludeNSFW   bool
}

type PostRow struct {
	ID          int
	TopicID     int
	TopicTitle  string
	TopicUserID int
	Content     string
	Floor       int
	UserID      int
	Created     time.Time
}

func orderBy(sql string, args []any) clause.OrderBy {
	return clause.OrderBy{Expression: clause.Expr{SQL: sql, Vars: args, WithoutParentheses: true}}
}

func topicVisible(alias string, q V1Query) string {
	where := alias + ".status != 1 AND " + topicRepo.SharedListPredicate(alias, q.Authenticated)
	if !q.IncludeNSFW {
		where += " AND " + alias + ".is_nsfw = false"
	}
	return where
}

func (r *SearchRepository) SearchTopicRowsV1(q V1Query) ([]topicRepo.TopicKeysetRow, int64, error) {
	score, scoreArgs := topicRelevance(q.Keywords)
	query := r.db.Table("topic t").Where(topicVisible("t", q))
	for _, kw := range q.Keywords {
		like := "%" + kw + "%"
		query = query.Where("(t.title ILIKE ? OR t.content ILIKE ? OR t.category ILIKE ?)", like, like, like)
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []topicRepo.TopicKeysetRow
	err := query.
		Select(`t.id, t.title, t.view, t.status, t.is_nsfw, t.like_count, t.reply_count,
			t.comment_count, t.best_answer_id, t.status_update_time, t.created, t.upvote_time,
			t.cover_images, t.user_id, t.category, t.favorite_count, t.upvote_count`).
		Order(orderBy(score+" DESC, t.status_update_time DESC, t.id DESC", scoreArgs)).
		Offset(q.Offset).Limit(q.Limit).
		Find(&rows).Error
	return rows, total, err
}

func (r *SearchRepository) searchPostRowsV1(table, alias string, withFloor bool, q V1Query) ([]PostRow, int64, error) {
	col := alias + ".content"
	score, scoreArgs := contentRelevance(col, q.Keywords)
	query := r.db.Table(table + " " + alias).
		Joins("JOIN topic t ON t.id = " + alias + ".topic_id").
		Where(alias + ".status = 0").
		Where(topicVisible("t", q))
	for _, kw := range q.Keywords {
		query = query.Where(col+" ILIKE ?", "%"+kw+"%")
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	floor := "0 AS floor"
	if withFloor {
		floor = alias + ".floor"
	}
	var rows []PostRow
	err := query.
		Select(alias + ".id, " + alias + ".topic_id, t.title AS topic_title, t.user_id AS topic_user_id, " +
			col + " AS content, " + floor + ", " + alias + ".user_id, " + alias + ".created").
		Order(orderBy(score+" DESC, "+alias+".created DESC, "+alias+".id DESC", scoreArgs)).
		Offset(q.Offset).Limit(q.Limit).
		Find(&rows).Error
	return rows, total, err
}

func (r *SearchRepository) SearchReplyRowsV1(q V1Query) ([]PostRow, int64, error) {
	return r.searchPostRowsV1("topic_reply", "r", true, q)
}

func (r *SearchRepository) SearchCommentRowsV1(q V1Query) ([]PostRow, int64, error) {
	return r.searchPostRowsV1("topic_comment", "c", false, q)
}
