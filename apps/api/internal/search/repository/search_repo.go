package repository

import (
	topicRepo "kun-galgame-api/internal/topic/repository"

	"gorm.io/gorm"
)

type SearchRepository struct {
	db *gorm.DB
}

func NewSearchRepository(db *gorm.DB) *SearchRepository {
	return &SearchRepository{db: db}
}

// CountUserPosts is what a user result shows besides a name and a bio: how much
// of the forum this account actually wrote. OAuth's /users/search cannot answer
// it — the counts are local.
func (r *SearchRepository) CountUserPosts(userIDs []int, authenticated bool) (topics, replies map[int]int) {
	topics, replies = map[int]int{}, map[int]int{}
	if len(userIDs) == 0 {
		return
	}
	type countRow struct {
		UserID int   `gorm:"column:user_id"`
		Total  int64 `gorm:"column:total"`
	}
	for _, lane := range []struct {
		table string
		where string
		out   map[int]int
	}{
		{"topic", "status != 1 AND " + topicRepo.SharedListPredicate("", authenticated), topics},
		{"topic_reply", "status = 0", replies},
	} {
		var rows []countRow
		r.db.Table(lane.table).
			Select("user_id, count(*) AS total").
			Where("user_id IN ?", userIDs).
			Where(lane.where).
			Group("user_id").
			Scan(&rows)
		for _, row := range rows {
			lane.out[row.UserID] = int(row.Total)
		}
	}
	return
}
