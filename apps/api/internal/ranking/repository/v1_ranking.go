package repository

import (
	"strconv"

	topicRepo "kun-galgame-api/internal/topic/repository"
)

type RankedRow struct {
	ID    int     `gorm:"column:id"`
	Title string  `gorm:"column:title"`
	Value float64 `gorm:"column:value"`
	Owner int     `gorm:"column:owner_id"`
}

var listedTopic = "t.status != 1 AND " + topicRepo.SharedListPredicate("t", false)

var topicRankingColumns = map[string]string{
	"views":     "view",
	"replies":   "reply_count",
	"comments":  "comment_count",
	"likes":     "like_count",
	"upvotes":   "upvote_count",
	"favorites": "favorite_count",
}

func (r *RankingRepository) TopTopics(key string, includeNSFW bool, limit int) ([]RankedRow, error) {
	col := topicRankingColumns[key]
	q := r.db.Table("topic t").
		Select("t.id, t.title, t.user_id AS owner_id, t." + col + " AS value").
		Where(listedTopic)
	if !includeNSFW {
		q = q.Where("t.is_nsfw = false")
	}
	var rows []RankedRow
	err := q.Order("t." + col + " DESC, t.id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

var userRankingQueries = map[string]string{
	"moemoepoint": `SELECT user_id AS id, moemoepoint AS value FROM kungal_user_state`,
	"topics":      `SELECT t.user_id AS id, COUNT(*) AS value FROM topic t WHERE ` + listedTopic + ` GROUP BY t.user_id`,
	"replies": `SELECT r.user_id AS id, COUNT(*) AS value FROM topic_reply r JOIN topic t ON t.id = r.topic_id
		WHERE r.status = 0 AND ` + listedTopic + ` GROUP BY r.user_id`,
	"comments": `SELECT c.user_id AS id, COUNT(*) AS value FROM topic_comment c JOIN topic t ON t.id = c.topic_id
		WHERE c.status = 0 AND ` + listedTopic + ` GROUP BY c.user_id`,
	"resources": `SELECT user_id AS id, COUNT(*) AS value FROM galgame_resource WHERE status = 0 GROUP BY user_id`,
}

func (r *RankingRepository) TopUsers(key string, limit int) ([]RankedRow, error) {
	var rows []RankedRow
	err := r.db.Raw(`SELECT id, value FROM (`+userRankingQueries[key]+`) ranked
		ORDER BY value DESC, id DESC LIMIT ?`, limit).Scan(&rows).Error
	return rows, err
}

var workRankingColumns = map[string]string{
	"views":     "view",
	"likes":     "like_count",
	"favorites": "favorite_count",
	"resources": "resource_count",
}

func (r *RankingRepository) TopWorks(key string, includeNSFW, includeResourceless bool, limit int) ([]RankedRow, error) {
	q := r.db.Table("galgame g").Where("g.published")
	if !includeNSFW {
		q = q.Where("g.content_limit IS NULL OR g.content_limit = 'sfw'")
	}
	if !includeResourceless {
		q = q.Where("EXISTS (SELECT 1 FROM galgame_resource gr WHERE gr.work_id = g.id)")
	}
	var rows []RankedRow
	if key == "rating" {
		var mean float64
		if err := r.db.Table("galgame_rating").Select("COALESCE(AVG(overall), 0)").Scan(&mean).Error; err != nil {
			return nil, err
		}
		c := strconv.FormatFloat(rankingBayesianPriorC, 'f', -1, 64)
		m := strconv.FormatFloat(mean, 'f', 6, 64)
		bayes := "(" + c + " * " + m + " + rt.rsum) / (" + c + " + rt.rcnt)"
		err := q.Joins("JOIN (SELECT work_id, SUM(overall) AS rsum, COUNT(*) AS rcnt FROM galgame_rating GROUP BY work_id) rt ON rt.work_id = g.id").
			Select("g.id, COALESCE(g.creator_user_id, 0) AS owner_id, ROUND((" + bayes + ")::numeric, 2) AS value").
			Order(bayes + " DESC, g.id DESC").Limit(limit).Scan(&rows).Error
		return rows, err
	}
	col := workRankingColumns[key]
	err := q.Select("g.id, COALESCE(g.creator_user_id, 0) AS owner_id, g." + col + " AS value").
		Order("g." + col + " DESC, g.id DESC").Limit(limit).Scan(&rows).Error
	return rows, err
}
