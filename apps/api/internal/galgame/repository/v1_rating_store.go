package repository

import (
	"encoding/json"
	"errors"
	"time"

	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
)

var (
	ErrRatingNotFound = errors.New("rating not found")
	ErrRatingExists   = errors.New("rating already exists for this work")
)

type RatingStore struct {
	db *gorm.DB
}

func NewRatingStore(db *gorm.DB) *RatingStore {
	return &RatingStore{db: db}
}

func (s *RatingStore) DB() *gorm.DB { return s.db }

type RatingRecord struct {
	ID           int             `gorm:"column:id"`
	WorkID       int             `gorm:"column:work_id"`
	UserID       int             `gorm:"column:user_id"`
	Recommend    string          `gorm:"column:recommend"`
	Overall      int             `gorm:"column:overall"`
	View         int             `gorm:"column:view"`
	GalgameType  json.RawMessage `gorm:"column:galgame_type"`
	PlayStatus   string          `gorm:"column:play_status"`
	ShortSummary string          `gorm:"column:short_summary"`
	SpoilerLevel string          `gorm:"column:spoiler_level"`
	Art          int             `gorm:"column:art"`
	Story        int             `gorm:"column:story"`
	Music        int             `gorm:"column:music"`
	Character    int             `gorm:"column:character"`
	Route        int             `gorm:"column:route"`
	System       int             `gorm:"column:system"`
	Voice        int             `gorm:"column:voice"`
	ReplayValue  int             `gorm:"column:replay_value"`
	LikeCount    int             `gorm:"column:like_count"`
	CommentCount int             `gorm:"column:comment_count"`
	Created      time.Time       `gorm:"column:created"`
	Updated      time.Time       `gorm:"column:updated"`
}

type RatingQuery struct {
	WorkID       int
	AuthorID     int
	SpoilerLevel string
	PlayStatus   string
	GameType     string
	SFWOnly      bool
	SortColumn   string
	Descending   bool
	Offset       int
	Limit        int
}

var ratingSortColumns = map[string]string{
	"created": "r.created",
	"view":    "r.view",
	"overall": "r.overall",
}

func (s *RatingStore) filtered(q RatingQuery) *gorm.DB {
	tx := s.db.Table("galgame_rating r").Joins("JOIN galgame g ON g.id = r.work_id").Where("g.catalog_rendered")
	if q.WorkID > 0 {
		tx = tx.Where("r.work_id = ?", q.WorkID)
	}
	if q.AuthorID > 0 {
		tx = tx.Where("r.user_id = ?", q.AuthorID)
	}
	if q.SpoilerLevel != "" {
		tx = tx.Where("r.spoiler_level = ?", q.SpoilerLevel)
	}
	if q.PlayStatus != "" {
		tx = tx.Where("r.play_status = ?", q.PlayStatus)
	}
	if q.GameType != "" {
		raw, _ := json.Marshal([]string{q.GameType})
		tx = tx.Where("r.galgame_type @> ?::jsonb", string(raw))
	}
	if q.SFWOnly {
		tx = tx.Where("g.content_limit = 'sfw'")
	}
	return tx
}

func (s *RatingStore) List(q RatingQuery) ([]RatingRecord, int, error) {
	var total int64
	if err := s.filtered(q).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	col, ok := ratingSortColumns[q.SortColumn]
	if !ok {
		col = ratingSortColumns["created"]
	}
	dir := " ASC"
	if q.Descending {
		dir = " DESC"
	}
	var rows []RatingRecord
	err := s.filtered(q).Select("r.*").
		Order(col + dir).Order("r.id" + dir).
		Offset(q.Offset).Limit(q.Limit).
		Scan(&rows).Error
	return rows, int(total), err
}

func (s *RatingStore) Get(id int) (RatingRecord, error) {
	var rows []RatingRecord
	if err := s.db.Table("galgame_rating").Where("id = ?", id).Limit(1).Scan(&rows).Error; err != nil {
		return RatingRecord{}, err
	}
	if len(rows) == 0 {
		return RatingRecord{}, ErrRatingNotFound
	}
	return rows[0], nil
}

func (s *RatingStore) IncrementView(id int) error {
	return s.db.Table("galgame_rating").Where("id = ?", id).
		UpdateColumn("view", gorm.Expr("view + 1")).Error
}

func (s *RatingStore) RecentLikers(ratingID, limit int) ([]int, error) {
	var ids []int
	err := s.db.Table("galgame_rating_like").Where("galgame_rating_id = ?", ratingID).
		Order("created DESC").Order("id DESC").Limit(limit).Pluck("user_id", &ids).Error
	return ids, err
}

func (s *RatingStore) LikedSet(userID int, ratingIDs []int) (map[int]bool, error) {
	out := make(map[int]bool, len(ratingIDs))
	if userID <= 0 || len(ratingIDs) == 0 {
		return out, nil
	}
	var ids []int
	err := s.db.Table("galgame_rating_like").
		Where("user_id = ? AND galgame_rating_id IN ?", userID, ratingIDs).
		Pluck("galgame_rating_id", &ids).Error
	for _, id := range ids {
		out[id] = true
	}
	return out, err
}

// Create writes the rating and the local work row it hangs off; a second rating
// of the same work by the same user loses to the unique (user_id, work_id) key.
func (s *RatingStore) Create(r *model.GalgameRating) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`INSERT INTO galgame (id, updated) VALUES (?, now()) ON CONFLICT (id) DO NOTHING`, r.WorkID).Error; err != nil {
			return err
		}
		var ids []int
		err := tx.Raw(`INSERT INTO galgame_rating
			(work_id, user_id, recommend, overall, galgame_type, play_status, short_summary, spoiler_level,
			 art, story, music, character, route, system, voice, replay_value, created, updated)
			VALUES (?, ?, ?, ?, ?::jsonb, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now(), now())
			ON CONFLICT (user_id, work_id) DO NOTHING RETURNING id`,
			r.WorkID, r.UserID, r.Recommend, r.Overall, string(r.GalgameType), r.PlayStatus, r.ShortSummary, r.SpoilerLevel,
			r.Art, r.Story, r.Music, r.Character, r.Route, r.System, r.Voice, r.ReplayValue).Scan(&ids).Error
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			return ErrRatingExists
		}
		r.ID = ids[0]
		return nil
	})
}

func (s *RatingStore) Update(id int, fields map[string]any) error {
	fields["updated"] = gorm.Expr("now()")
	return s.db.Table("galgame_rating").Where("id = ?", id).Updates(fields).Error
}

func (s *RatingStore) Delete(id int) error {
	return s.db.Exec(`DELETE FROM galgame_rating WHERE id = ?`, id).Error
}

// SetLike returns the id of the like row it inserted or deleted, or 0 when the
// row was already in the wanted state; only a change moves the counter.
func (s *RatingStore) SetLike(tx *gorm.DB, ratingID, userID int, want bool) (int, error) {
	var ids []int
	var err error
	if want {
		err = tx.Raw(`INSERT INTO galgame_rating_like (galgame_rating_id, user_id, created, updated)
			VALUES (?, ?, now(), now()) ON CONFLICT (galgame_rating_id, user_id) DO NOTHING RETURNING id`, ratingID, userID).Scan(&ids).Error
	} else {
		err = tx.Raw(`DELETE FROM galgame_rating_like WHERE galgame_rating_id = ? AND user_id = ? RETURNING id`, ratingID, userID).Scan(&ids).Error
	}
	if err != nil || len(ids) == 0 {
		return 0, err
	}
	delta := 1
	if !want {
		delta = -1
	}
	if err := tx.Exec(`UPDATE galgame_rating SET like_count = GREATEST(like_count + ?, 0) WHERE id = ?`, delta, ratingID).Error; err != nil {
		return 0, err
	}
	return ids[0], nil
}

func (s *RatingStore) CountReviewsWithMinLength(userID, minLen int) (int64, error) {
	var n int64
	err := s.db.Table("galgame_rating").
		Where("user_id = ? AND char_length(short_summary) >= ?", userID, minLen).
		Count(&n).Error
	return n, err
}

func (s *RatingStore) WorkCreators(workIDs []int) (map[int]int, error) {
	out := make(map[int]int, len(workIDs))
	if len(workIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		ID            int `gorm:"column:id"`
		CreatorUserID int `gorm:"column:creator_user_id"`
	}
	err := s.db.Table("galgame").Select("id, creator_user_id").
		Where("id IN ? AND creator_user_id IS NOT NULL", workIDs).Scan(&rows).Error
	for _, r := range rows {
		out[r.ID] = r.CreatorUserID
	}
	return out, err
}
