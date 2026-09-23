package repository

import (
	"kun-galgame-api/internal/user/model"

	"gorm.io/gorm"
)

type UserStatsRepository struct {
	db *gorm.DB
}

func NewUserStatsRepository(db *gorm.DB) *UserStatsRepository {
	return &UserStatsRepository{db: db}
}

type UserStats = model.UserStats

func (r *UserStatsRepository) GetUserStats(userID int) (*model.UserStats, error) {
	var stats model.UserStats
	err := r.db.Raw(`
		SELECT
			(SELECT COUNT(*) FROM topic WHERE user_id = @userID) AS topic,
			(SELECT COUNT(*) FROM topic_poll WHERE user_id = @userID) AS topic_poll,
			(SELECT COUNT(*) FROM topic_lottery WHERE user_id = @userID) AS topic_lottery,
			(SELECT COUNT(*) FROM topic_reply WHERE user_id = @userID AND status = 0) AS reply_created,
			(SELECT COUNT(*) FROM topic_comment WHERE user_id = @userID AND status = 0) AS comment_created,
			-- galgame_comment (now "all community comment areas") is overlaid by the
			-- service from the community primitive's visible_posts (charter step 06a),
			-- not counted from the frozen galgame_comment table here.
			(SELECT COUNT(*) FROM galgame_rating WHERE user_id = @userID) AS galgame_rating,
			(SELECT COUNT(*) FROM galgame_resource WHERE user_id = @userID) AS galgame_resource,
			(SELECT COUNT(*) FROM galgame_website WHERE user_id = @userID) AS galgame_toolset,
			(SELECT COUNT(*) FROM galgame_toolset_resource WHERE user_id = @userID) AS galgame_toolset_resource,
			(SELECT COUNT(*) FROM topic_upvote WHERE topic_id IN (SELECT id FROM topic WHERE user_id = @userID)) AS upvote,
			(SELECT COUNT(*) FROM topic_reaction WHERE reaction = 'like' AND topic_id IN (SELECT id FROM topic WHERE user_id = @userID)) AS "like",
			(SELECT COUNT(*) FROM topic_reaction WHERE reaction = 'dislike' AND topic_id IN (SELECT id FROM topic WHERE user_id = @userID)) AS dislike,
			(SELECT COUNT(*) FROM topic WHERE user_id = @userID AND created >= CURRENT_DATE) AS daily_topic_count
	`, map[string]any{"userID": userID}).Scan(&stats).Error
	return &stats, err
}

func (r *UserStatsRepository) CountUnreadMessages(userID int, mutedLocal []string) (int64, error) {
	var count int64
	q := r.db.Table("message").
		Where("receiver_id = ? AND status = 'unread'", userID)
	if len(mutedLocal) > 0 {
		q = q.Not("type IN ?", mutedLocal)
	}
	err := q.Count(&count).Error
	return count, err
}

func (r *UserStatsRepository) CountUnreadChatMessages(userID int) (int64, error) {
	var count int64
	err := r.db.Table("chat_message").
		Where("sender_id != ?", userID).
		Where("chat_room_id IN (SELECT chat_room_id FROM chat_room_participant WHERE user_id = ?)", userID).
		Where("id NOT IN (SELECT chat_message_id FROM chat_message_read_by WHERE user_id = ?)", userID).
		Count(&count).Error
	return count, err
}

type FloatingStatsRow struct {
	TopicCount        int64 `gorm:"column:topic_count"`
	TopicReplyCount   int64 `gorm:"column:topic_reply_count"`
	TopicCommentCount int64 `gorm:"column:topic_comment_count"`
	ResourceCount     int64 `gorm:"column:resource_count"`
}

func (r *UserStatsRepository) FindFloatingStats(userID int) FloatingStatsRow {
	var stats FloatingStatsRow
	r.db.Raw(`
		SELECT
			(SELECT COUNT(*) FROM topic WHERE user_id = @userID) AS topic_count,
			(SELECT COUNT(*) FROM topic_reply WHERE user_id = @userID AND status = 0) AS topic_reply_count,
			-- Only the local topic_comment count here; the community comment areas
			-- (galgame + website + …) are added by the service from the primitive's
			-- visible_posts (charter step 06a), replacing the old galgame_comment +
			-- galgame_website_comment frozen-table terms.
			(SELECT COUNT(*) FROM topic_comment WHERE user_id = @userID AND status = 0) AS topic_comment_count,
			(SELECT COUNT(*) FROM galgame_resource WHERE user_id = @userID) AS resource_count
	`, map[string]any{"userID": userID}).Scan(&stats)
	return stats
}
