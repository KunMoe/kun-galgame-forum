package repository

import (
	"time"

	"gorm.io/gorm"
)

type UserStatsRepository struct {
	db *gorm.DB
}

func NewUserStatsRepository(db *gorm.DB) *UserStatsRepository {
	return &UserStatsRepository{db: db}
}

type LocalProfileCounts struct {
	TopicCount           int64 `gorm:"column:topic_count"`
	PollCount            int64 `gorm:"column:poll_count"`
	LotteryCount         int64 `gorm:"column:lottery_count"`
	ReplyCount           int64 `gorm:"column:reply_count"`
	TopicCommentCount    int64 `gorm:"column:topic_comment_count"`
	GalgameRatingCount   int64 `gorm:"column:galgame_rating_count"`
	GalgameResourceCount int64 `gorm:"column:galgame_resource_count"`
	ToolsetCount         int64 `gorm:"column:toolset_count"`
	ToolsetResourceCount int64 `gorm:"column:toolset_resource_count"`
	ReceivedUpvoteCount  int64 `gorm:"column:received_upvote_count"`
	ReceivedLikeCount    int64 `gorm:"column:received_like_count"`
	ReceivedDislikeCount int64 `gorm:"column:received_dislike_count"`
	TopicTodayCount      int64 `gorm:"column:topic_today_count"`
}

func (r *UserStatsRepository) ProfileCounts(userID int, todayStart time.Time) (LocalProfileCounts, error) {
	var stats LocalProfileCounts
	// The production profile counted galgame_website, the website directory.
	// The DB session zone is UTC, so CURRENT_DATE started "today" at 08:00 Beijing.
	err := r.db.Raw(`
		SELECT
			(SELECT COUNT(*) FROM topic WHERE user_id = @userID AND status != 1) AS topic_count,
			(SELECT COUNT(*) FROM topic_poll WHERE user_id = @userID) AS poll_count,
			(SELECT COUNT(*) FROM topic_lottery WHERE user_id = @userID) AS lottery_count,
			(SELECT COUNT(*) FROM topic_reply WHERE user_id = @userID AND status = 0) AS reply_count,
			(SELECT COUNT(*) FROM topic_comment WHERE user_id = @userID AND status = 0) AS topic_comment_count,
			(SELECT COUNT(*) FROM galgame_rating WHERE user_id = @userID) AS galgame_rating_count,
			(SELECT COUNT(*) FROM galgame_resource WHERE user_id = @userID) AS galgame_resource_count,
			(SELECT COUNT(*) FROM galgame_toolset WHERE user_id = @userID) AS toolset_count,
			(SELECT COUNT(*) FROM galgame_toolset_resource WHERE user_id = @userID) AS toolset_resource_count,
			(SELECT COUNT(*) FROM topic_upvote WHERE topic_id IN (SELECT id FROM topic WHERE user_id = @userID)) AS received_upvote_count,
			(SELECT COUNT(*) FROM topic_reaction WHERE reaction = 'like' AND topic_id IN (SELECT id FROM topic WHERE user_id = @userID)) AS received_like_count,
			(SELECT COUNT(*) FROM topic_reaction WHERE reaction = 'dislike' AND topic_id IN (SELECT id FROM topic WHERE user_id = @userID)) AS received_dislike_count,
			(SELECT COUNT(*) FROM topic WHERE user_id = @userID AND created >= @todayStart) AS topic_today_count
	`, map[string]any{"userID": userID, "todayStart": todayStart}).Scan(&stats).Error
	return stats, err
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
