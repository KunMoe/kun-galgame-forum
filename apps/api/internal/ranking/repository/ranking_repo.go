package repository

import (
	topicRepo "kun-galgame-api/internal/topic/repository"
	"strconv"

	"gorm.io/gorm"
)

const rankingBayesianPriorC = 10.0

var galgameSortColumn = map[string]string{
	"view":     "view",
	"like":     "like_count",
	"favorite": "favorite_count",
	"resource": "resource_count",
}

var topicSortColumn = map[string]string{
	"view":     "view",
	"upvote":   "upvote_count",
	"like":     "like_count",
	"reply":    "reply_count",
	"comment":  "comment_count",
	"favorite": "favorite_count",
}

var userCountSource = map[string]struct {
	table string
	where string
}{
	"topic":            {table: "topic", where: "status != 1 AND " + topicRepo.SharedListPredicate("", false)},
	"reply_created":    {table: "topic_reply"},
	"comment_created":  {table: "topic_comment"},
	"galgame_resource": {table: "galgame_resource"},
}

type RankingRepository struct {
	db *gorm.DB
}

func NewRankingRepository(db *gorm.DB) *RankingRepository {
	return &RankingRepository{db: db}
}

type GalgameLocalRow struct {
	ID            int     `gorm:"column:id"`
	Value         float64 `gorm:"column:value"`
	CreatorUserID *int    `gorm:"column:creator_user_id"`
}

type TopicRankingRow struct {
	ID     int    `gorm:"column:id"`
	Title  string `gorm:"column:title"`
	UserID int    `gorm:"column:user_id"`
	Value  int    `gorm:"column:value"`
}

type UserRankingRow struct {
	UserID int `gorm:"column:user_id"`
	Value  int `gorm:"column:value"`
}

func (r *RankingRepository) FindGalgameLocal(sortField, sortOrder string, page, limit int, showNoResource bool) []GalgameLocalRow {
	var rows []GalgameLocalRow

	if sortField == "rating" {
		var m float64
		r.db.Table("galgame_rating").Select("COALESCE(AVG(overall), 0)").Scan(&m)
		c := strconv.FormatFloat(rankingBayesianPriorC, 'f', -1, 64)
		ms := strconv.FormatFloat(m, 'f', 6, 64)
		bayes := "(" + c + " * " + ms + " + rt.rsum) / (" + c + " + rt.rcnt)"
		q := r.db.Table("galgame g").
			Joins("JOIN (SELECT work_id, SUM(overall) AS rsum, COUNT(*) AS rcnt " +
				"FROM galgame_rating GROUP BY work_id) rt ON rt.work_id = g.id").
			Select("g.id, g.creator_user_id, ROUND((" + bayes + ")::numeric, 2) AS value").
			Where("g.published")
		if !showNoResource {
			q = q.Where("EXISTS (SELECT 1 FROM galgame_resource gr WHERE gr.work_id = g.id)")
		}
		q.Order(bayes + " " + sortOrder).
			Offset((page - 1) * limit).
			Limit(limit).
			Scan(&rows)
		return rows
	}

	col := galgameSortColumn[sortField]
	if col == "" {
		col = "view"
	}
	q := r.db.Table("galgame").
		Select("id, creator_user_id, " + col + " AS value").
		Where("published")
	if !showNoResource {
		q = q.Where("EXISTS (SELECT 1 FROM galgame_resource gr WHERE gr.work_id = galgame.id)")
	}
	q.Order(col + " " + sortOrder).
		Offset((page - 1) * limit).
		Limit(limit).
		Scan(&rows)
	return rows
}

func (r *RankingRepository) FindTopicRanking(sortField, sortOrder string, page, limit int, isSFW bool) []TopicRankingRow {
	var rows []TopicRankingRow
	col := topicSortColumn[sortField]
	if col == "" {
		col = "view"
	}
	q := r.db.Table("topic t").
		Select(`t.id, t.title, t.user_id, t.` + col + ` AS value`).
		Where("t.status != 1").
		Where(topicRepo.SharedListPredicate("t", false))
	if isSFW {
		q = q.Where("t.is_nsfw = false")
	}
	q.Order("t." + col + " " + sortOrder).
		Offset((page - 1) * limit).Limit(limit).
		Find(&rows)
	return rows
}

func (r *RankingRepository) FindUserRanking(sortField, sortOrder string, page, limit int) []UserRankingRow {
	var rows []UserRankingRow

	if sortField == "moemoepoint" {
		r.db.Table("kungal_user_state").
			Select("user_id, moemoepoint AS value").
			Order("moemoepoint " + sortOrder).
			Offset((page - 1) * limit).Limit(limit).
			Find(&rows)
		return rows
	}

	src, ok := userCountSource[sortField]
	if !ok {
		return rows
	}
	q := r.db.Table(src.table).Select("user_id, COUNT(*) AS value")
	if src.where != "" {
		q = q.Where(src.where)
	}
	q.Group("user_id").
		Order("value " + sortOrder).
		Offset((page - 1) * limit).Limit(limit).
		Find(&rows)
	return rows
}
