package repository

type UserWorkQuery struct {
	OwnerID     int
	Relation    string
	IncludeNSFW bool
	Offset      int
	Limit       int
}

type workIDRow struct {
	ID int `gorm:"column:id"`
}

func (r *UserContentRepository) ListUserWorkIDs(q UserWorkQuery) ([]int, int, error) {
	db := r.db.Table("galgame")
	switch q.Relation {
	case "published":
		db = db.Where("galgame.creator_user_id = ? AND galgame.published", q.OwnerID)
	case "liked":
		db = db.
			Joins("JOIN galgame_like ON galgame_like.work_id = galgame.id").
			Where("galgame_like.user_id = ? AND galgame.published", q.OwnerID)
	default:
		db = db.Where("1 = 0")
	}
	db = db.Where("galgame.catalog_rendered")
	if !q.IncludeNSFW {
		db = db.Where("galgame.content_limit IS NULL OR galgame.content_limit <> ?", "nsfw")
	}
	rows, total, err := pageScan[workIDRow](db, "galgame.id", "galgame.created DESC, galgame.id DESC", q.Offset, q.Limit)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]int, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	return ids, total, nil
}

func (r *UserContentRepository) ListUserLikedPosts(userID int, afterID int64, limit int) ([]LikedPostRow, error) {
	q := r.db.Table("galgame_post_like").
		Select("id, post_id").
		Where("user_id = ?", userID)
	if afterID > 0 {
		q = q.Where("id < ?", afterID)
	}
	var rows []LikedPostRow
	err := q.Order("id DESC").Limit(limit).Scan(&rows).Error
	if rows == nil {
		rows = []LikedPostRow{}
	}
	return rows, err
}
