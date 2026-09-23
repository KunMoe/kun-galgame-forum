package repository

import (
	topicRepo "kun-galgame-api/internal/topic/repository"
	"time"

	"gorm.io/gorm"

	"kun-galgame-api/pkg/miniapp"
)

type HomeRepository struct {
	db *gorm.DB
}

func NewHomeRepository(db *gorm.DB) *HomeRepository {
	return &HomeRepository{db: db}
}

type GalgameLocalRow struct {
	ID                 int       `gorm:"column:id"`
	View               int       `gorm:"column:view"`
	LikeCount          int       `gorm:"column:like_count"`
	ResourceUpdateTime time.Time `gorm:"column:resource_update_time"`
	CreatorUserID      *int      `gorm:"column:creator_user_id"`
}

type ResourcePLRow struct {
	WorkID   int    `gorm:"column:work_id"`
	Platform string `gorm:"column:platform"`
	Language string `gorm:"column:language"`
}

type TopicRow struct {
	ID               int        `gorm:"column:id"`
	Title            string     `gorm:"column:title"`
	View             int        `gorm:"column:view"`
	IsNSFW           bool       `gorm:"column:is_nsfw"`
	Status           int        `gorm:"column:status"`
	LikeCount        int        `gorm:"column:like_count"`
	ReplyCount       int        `gorm:"column:reply_count"`
	CommentCount     int        `gorm:"column:comment_count"`
	BestAnswerID     *int       `gorm:"column:best_answer_id"`
	UpvoteTime       *time.Time `gorm:"column:upvote_time"`
	StatusUpdateTime time.Time  `gorm:"column:status_update_time"`
	UserID           int        `gorm:"column:user_id"`
}

type SectionRelationRow struct {
	TopicID     int    `gorm:"column:topic_id"`
	SectionName string `gorm:"column:name"`
}

func (r *HomeRepository) FindRecentGalgames(limit int) ([]GalgameLocalRow, error) {
	var rows []GalgameLocalRow
	err := r.db.Table("galgame").
		Select("id, view, like_count, resource_update_time, creator_user_id").
		Where("published").
		Order("resource_update_time DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (r *HomeRepository) FindResourcePlatformLanguage(workIDs []int) []ResourcePLRow {
	if len(workIDs) == 0 {
		return nil
	}
	var resources []ResourcePLRow
	r.db.Table("galgame_resource").
		Select("work_id, platform, language").
		Where("work_id IN ?", workIDs).
		Find(&resources)
	return resources
}

func (r *HomeRepository) FindHomeTopics(limit int, isSFW bool) ([]TopicRow, error) {
	threeMonthsAgo := time.Now().AddDate(0, -3, 0)
	excludedSections := []string{"g-seeking", "g-other", "t-help"}

	query := r.db.Table("topic").
		Select(`topic.id, topic.title, topic.view, topic.is_nsfw, topic.status,
			topic.like_count, topic.reply_count, topic.comment_count,
			topic.best_answer_id, topic.upvote_time, topic.status_update_time,
			topic.user_id`).
		Where("topic.status != 1").
		Where(topicRepo.SharedListPredicate("topic", false)).
		Where(`topic.id NOT IN (
			SELECT tsr.topic_id FROM topic_section_relation tsr
			JOIN topic_section ts ON ts.id = tsr.topic_section_id
			WHERE ts.name IN ?
		)`, excludedSections).
		Where(`(topic.edited >= ? OR (topic.edited IS NULL AND topic.created >= ?))`, threeMonthsAgo, threeMonthsAgo).
		Order("topic.status_update_time DESC").
		Limit(limit)

	if isSFW {
		query = query.Where("topic.is_nsfw = false")
	}

	var rows []TopicRow
	err := query.Find(&rows).Error
	return rows, err
}

func (r *HomeRepository) FindTopicSections(topicIDs []int) []SectionRelationRow {
	if len(topicIDs) == 0 {
		return nil
	}
	var sections []SectionRelationRow
	r.db.Table("topic_section_relation tsr").
		Select("tsr.topic_id, ts.name").
		Joins("JOIN topic_section ts ON ts.id = tsr.topic_section_id").
		Where("tsr.topic_id IN ?", topicIDs).
		Find(&sections)
	return sections
}

func (r *HomeRepository) FindTopicMiniApps(topicIDs []int) map[int][]string {
	return miniapp.ByTopic(r.db, topicIDs)
}
