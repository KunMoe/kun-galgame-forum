package repository

import (
	"fmt"
	"time"

	"kun-galgame-api/internal/topic/model"
)

const view1dExpr = "COALESCE((SELECT SUM(d.count) FROM topic_view_daily d " +
	"WHERE d.entity_id = topic.id AND d.day = CURRENT_DATE), 0)"

type TopicKeysetRow struct {
	ID               int
	Title            string
	View             int
	Status           int
	IsNSFW           bool
	LikeCount        int
	ReplyCount       int
	CommentCount     int
	BestAnswerID     *int
	StatusUpdateTime time.Time
	Created          time.Time
	UpvoteTime       *time.Time
	CoverImages      model.ImageTokens
	UserID           int
	Category         string
	FavoriteCount    int
	UpvoteCount      int
	SortInt          int64     `gorm:"column:sort_int"`
	SortTime         time.Time `gorm:"column:sort_time"`
}

type KeysetPos struct {
	ID       int
	SortInt  int64
	SortTime time.Time
	TimeSort bool
}

type KeysetQuery struct {
	SortKey       string
	Direction     string
	Category      string
	Section       string
	IncludeNSFW   bool
	Authenticated bool
	Limit         int
	Pos           *KeysetPos
}

func keysetExpr(sortKey string) (string, bool, error) {
	switch sortKey {
	case "view_1d":
		return view1dExpr, false, nil
	case "status_update_time", "created":
		return "topic." + sortKey, true, nil
	case "view", "view_7d", "view_30d", "like_count", "favorite_count", "upvote_count":
		return "topic." + sortKey, false, nil
	default:
		return "", false, fmt.Errorf("unknown topic sort key %q", sortKey)
	}
}

func (r *TopicListRepository) FindKeyset(q KeysetQuery) ([]TopicKeysetRow, error) {
	dir := q.Direction
	if dir != "asc" && dir != "desc" {
		return nil, fmt.Errorf("unsupported sort direction %q", dir)
	}
	expr, timeSort, err := keysetExpr(q.SortKey)
	if err != nil {
		return nil, err
	}

	alias := expr + " AS sort_int"
	if timeSort {
		alias = expr + " AS sort_time"
	}
	selectList := `topic.id, topic.title, topic.view, topic.status,
			topic.is_nsfw, topic.like_count, topic.reply_count,
			topic.comment_count, topic.best_answer_id,
			topic.status_update_time, topic.created, topic.upvote_time,
			topic.cover_images, topic.user_id, topic.category,
			topic.favorite_count, topic.upvote_count, ` + alias

	query := r.db.Table("topic").
		Select(selectList).
		Where("topic.status != 1").
		Where(SharedListPredicate("topic", q.Authenticated))

	if !q.IncludeNSFW {
		query = query.Where("topic.is_nsfw = false")
	}
	if q.Category != "" {
		query = query.Where("topic.category = ?", q.Category)
	}
	if q.Section != "" {
		query = query.Where(`topic.id IN (SELECT tsr.topic_id FROM topic_section_relation tsr
			JOIN topic_section ts ON ts.id = tsr.topic_section_id WHERE ts.name = ?)`, q.Section)
	}

	if q.Pos != nil {
		cmp := "<"
		if dir == "asc" {
			cmp = ">"
		}
		if timeSort {
			query = query.Where("("+expr+", topic.id) "+cmp+" (?, ?)", q.Pos.SortTime, q.Pos.ID)
		} else {
			query = query.Where("(("+expr+")::bigint, topic.id) "+cmp+" (?, ?)", q.Pos.SortInt, q.Pos.ID)
		}
	}

	var rows []TopicKeysetRow
	err = query.
		Order(expr + " " + dir + ", topic.id " + dir).
		Limit(q.Limit + 1).
		Find(&rows).Error
	return rows, err
}
