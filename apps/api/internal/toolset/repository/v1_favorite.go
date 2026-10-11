package repository

import (
	"kun-galgame-api/internal/toolset/model"

	"gorm.io/gorm"
)

func (s *Store) SetFavorite(toolsetID, userID int, on bool) (rowID int64, changed bool, err error) {
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var ids []int64
		var q *gorm.DB
		if on {
			q = tx.Raw(`INSERT INTO galgame_toolset_favorite (toolset_id, user_id)
				VALUES (?, ?)
				ON CONFLICT (toolset_id, user_id) DO NOTHING
				RETURNING id`, toolsetID, userID)
		} else {
			q = tx.Raw(`DELETE FROM galgame_toolset_favorite
				WHERE toolset_id = ? AND user_id = ?
				RETURNING id`, toolsetID, userID)
		}
		if err := q.Scan(&ids).Error; err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		rowID, changed = ids[0], true
		delta := 1
		if !on {
			delta = -1
		}
		return tx.Exec(`UPDATE galgame_toolset SET favorite_count = GREATEST(favorite_count + ?, 0) WHERE id = ?`,
			delta, toolsetID).Error
	})
	return rowID, changed, err
}

func (s *Store) Engagement(toolsetID, userID int) (count int, favorited bool, err error) {
	row := s.db.Raw(`SELECT favorite_count,
		EXISTS (SELECT 1 FROM galgame_toolset_favorite WHERE toolset_id = t.id AND user_id = ?)
		FROM galgame_toolset t WHERE t.id = ?`, userID, toolsetID).Row()
	err = row.Scan(&count, &favorited)
	return count, favorited, err
}

func (s *Store) HasFavorited(toolsetID, userID int) (bool, error) {
	var n int64
	err := s.db.Table("galgame_toolset_favorite").
		Where("toolset_id = ? AND user_id = ?", toolsetID, userID).
		Count(&n).Error
	return n > 0, err
}

func (s *Store) ListFavorited(f ListFilter, offset, limit int) ([]model.GalgameToolset, error) {
	if f.AuthorIDs != nil && len(f.AuthorIDs) == 0 {
		return nil, nil
	}
	q := s.db.Table("galgame_toolset AS t").Select("t.*").
		Joins("JOIN galgame_toolset_favorite fav ON fav.toolset_id = t.id").
		Where("fav.user_id = ?", f.FavoritedBy)
	if f.AuthorIDs != nil {
		q = q.Where("t.user_id IN ?", f.AuthorIDs)
	}
	var rows []model.GalgameToolset
	err := q.Order("fav.created DESC, fav.id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, err
}
