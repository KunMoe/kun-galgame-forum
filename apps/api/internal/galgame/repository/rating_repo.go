package repository

import (
	"fmt"

	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
)

type RatingRepository struct {
	db *gorm.DB
}

func NewRatingRepository(db *gorm.DB) *RatingRepository {
	return &RatingRepository{db: db}
}

func (r *RatingRepository) DB() *gorm.DB { return r.db }

func (r *RatingRepository) CountReviewsWithMinLength(userID, minLen int) (int64, error) {
	var n int64
	err := r.db.Table("galgame_rating").
		Where("user_id = ? AND char_length(short_summary) >= ?", userID, minLen).
		Count(&n).Error
	return n, err
}

func (r *RatingRepository) FindByID(id int) (model.GalgameRatingRow, bool) {
	var row model.GalgameRatingRow
	if err := r.db.Table("galgame_rating").Where("id = ?", id).Scan(&row).Error; err != nil || row.ID == 0 {
		return row, false
	}
	return row, true
}

func (r *RatingRepository) FindLikerIDs(ratingID int) []int {
	type row struct {
		UserID int `gorm:"column:user_id"`
	}
	var rows []row
	r.db.Table("galgame_rating_like").Select("user_id").
		Where("galgame_rating_id = ?", ratingID).Scan(&rows)
	out := make([]int, len(rows))
	for i, r := range rows {
		out[i] = r.UserID
	}
	return out
}

func (r *RatingRepository) GalgameRatingStats(workID int) (sum, count int64) {
	r.db.Table("galgame_rating").Select("COALESCE(SUM(overall), 0)").
		Where("work_id = ?", workID).Scan(&sum)
	r.db.Table("galgame_rating").
		Where("work_id = ?", workID).Count(&count)
	return
}

func (r *RatingRepository) IncrementView(ratingID int) {
	go r.db.Table("galgame_rating").Where("id = ?", ratingID).
		Update("view", gorm.Expr("view + 1"))
}

func (r *RatingRepository) ListPaginated(f model.RatingFilter) ([]model.GalgameRatingRow, int64) {
	query := r.db.Table("galgame_rating r")
	if f.SpoilerLevel != "" && f.SpoilerLevel != "all" {
		query = query.Where("r.spoiler_level = ?", f.SpoilerLevel)
	}
	if f.PlayStatus != "" && f.PlayStatus != "all" {
		query = query.Where("r.play_status = ?", f.PlayStatus)
	}
	if f.GalgameType != "" && f.GalgameType != "all" {
		query = query.Where("r.galgame_type @> ?", fmt.Sprintf(`["%s"]`, f.GalgameType))
	}

	var total int64
	query.Count(&total)

	orderCol := "r.created"
	switch f.SortField {
	case "view":
		orderCol = "r.view"
	case "overall":
		orderCol = "r.overall"
	}

	var rows []model.GalgameRatingRow
	query.Select("r.*").
		Order(orderCol + " " + f.SortOrder).
		Offset((f.Page - 1) * f.Limit).Limit(f.Limit).
		Scan(&rows)
	return rows, total
}

func (r *RatingRepository) ExistsByUserGalgame(workID, userID int) bool {
	var cnt int64
	r.db.Table("galgame_rating").
		Where("work_id = ? AND user_id = ?", workID, userID).
		Count(&cnt)
	return cnt > 0
}

func (r *RatingRepository) FindRatingForWrite(id int) (*model.GalgameRating, error) {
	var rating model.GalgameRating
	err := r.db.First(&rating, id).Error
	if err != nil {
		return nil, err
	}
	return &rating, nil
}

func (r *RatingRepository) Create(tx *gorm.DB, rating *model.GalgameRating) error {
	return tx.Create(rating).Error
}

func (r *RatingRepository) Update(tx *gorm.DB, ratingID int, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	return tx.Table("galgame_rating").Where("id = ?", ratingID).Updates(fields).Error
}

func (r *RatingRepository) DeleteByID(tx *gorm.DB, ratingID int) error {
	return tx.Where("id = ?", ratingID).Delete(&model.GalgameRating{}).Error
}

// The column is creator_user_id. `Select("user_id")` named a column the table
// has never had, and Scan's error is dropped, so this quietly returned 0 —
// every "someone rated your entry" notification went to nobody.
func (r *RatingRepository) FindGalgameOwner(workID int) int {
	var userID int
	r.db.Table("galgame").Select("creator_user_id").Where("id = ?", workID).Scan(&userID)
	return userID
}

func (r *RatingRepository) FindLike(tx *gorm.DB, ratingID, userID int) (model.GalgameRatingLike, bool) {
	var like model.GalgameRatingLike
	err := tx.Where("galgame_rating_id = ? AND user_id = ?", ratingID, userID).
		First(&like).Error
	if err != nil {
		return like, false
	}
	return like, true
}

func (r *RatingRepository) CreateLike(tx *gorm.DB, ratingID, userID int) error {
	return tx.Create(&model.GalgameRatingLike{
		GalgameRatingID: ratingID, UserID: userID,
	}).Error
}

func (r *RatingRepository) DeleteLike(tx *gorm.DB, like model.GalgameRatingLike) error {
	return tx.Delete(&like).Error
}

func (r *RatingRepository) AdjustLikeCount(tx *gorm.DB, ratingID, delta int) error {
	return tx.Table("galgame_rating").Where("id = ?", ratingID).
		Update("like_count", gorm.Expr("like_count + ?", delta)).Error
}
