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

	AuthorState      *string `gorm:"column:author_state"`
	AuthorCompletion *string `gorm:"column:author_completion"`
	WrittenHere      bool    `gorm:"column:written_here"`
}

type RatingQuery struct {
	WorkID       int
	AuthorID     int
	SpoilerLevel string
	// AuthorState and AuthorCompletion filter on the author's current play
	// state; an empty AuthorState filters nothing, a nil completion means none.
	AuthorState      string
	AuthorCompletion *string
	GameType         string
	SFWOnly          bool
	SortColumn       string
	Descending       bool
	Offset           int
	Limit            int
}

var ratingSortColumns = map[string]string{
	"created": "r.created",
	"view":    "r.view",
	"overall": "r.overall",
}

// filtered reads every mirrored rating, including those on works the forum has
// no local row for yet: such a work counts as rendered, and as adult for the
// SFW gate, the way a missing content_limit does everywhere else.
func (s *RatingStore) filtered(q RatingQuery) *gorm.DB {
	tx := s.db.Table("galgame_rating r").Joins("LEFT JOIN galgame g ON g.id = r.work_id").
		Where("COALESCE(g.catalog_rendered, true)")
	if q.WorkID > 0 {
		tx = tx.Where("r.work_id = ?", q.WorkID)
	}
	if q.AuthorID > 0 {
		tx = tx.Where("r.user_id = ?", q.AuthorID)
	}
	if q.SpoilerLevel != "" {
		tx = tx.Where("r.spoiler_level = ?", q.SpoilerLevel)
	}
	if q.AuthorState != "" {
		tx = tx.Where("r.author_state = ? AND r.author_completion IS NOT DISTINCT FROM ?", q.AuthorState, q.AuthorCompletion)
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

// MirrorRating writes NextMoe's copy of one rating over the local row with the
// same id, keeping the forum's own columns. NextMoe keys ratings on (user,
// work) too, so a local row for the same person and work under another id is a
// rating NextMoe has replaced; it goes first, or the unique key would refuse
// the new one.
func (s *RatingStore) MirrorRating(r model.GalgameRating) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM galgame_rating WHERE user_id = ? AND work_id = ? AND id <> ?`,
			r.UserID, r.WorkID, r.ID).Error; err != nil {
			return err
		}
		return tx.Exec(`INSERT INTO galgame_rating
			(id, work_id, user_id, overall, art, story, music, character, route, system, voice, replay_value,
			 recommend, spoiler_level, short_summary, author_state, author_completion, client_id, site, written_here,
			 created, updated)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (id) DO UPDATE SET
			  work_id = EXCLUDED.work_id, user_id = EXCLUDED.user_id, overall = EXCLUDED.overall,
			  art = EXCLUDED.art, story = EXCLUDED.story, music = EXCLUDED.music, character = EXCLUDED.character,
			  route = EXCLUDED.route, system = EXCLUDED.system, voice = EXCLUDED.voice,
			  replay_value = EXCLUDED.replay_value, recommend = EXCLUDED.recommend,
			  spoiler_level = EXCLUDED.spoiler_level, short_summary = EXCLUDED.short_summary,
			  author_state = EXCLUDED.author_state, author_completion = EXCLUDED.author_completion,
			  client_id = EXCLUDED.client_id, site = EXCLUDED.site, written_here = EXCLUDED.written_here,
			  created = EXCLUDED.created, updated = EXCLUDED.updated
			WHERE (galgame_rating.work_id, galgame_rating.user_id, galgame_rating.overall, galgame_rating.art,
			       galgame_rating.story, galgame_rating.music, galgame_rating.character, galgame_rating.route,
			       galgame_rating.system, galgame_rating.voice, galgame_rating.replay_value, galgame_rating.recommend,
			       galgame_rating.spoiler_level, galgame_rating.short_summary, galgame_rating.author_state,
			       galgame_rating.author_completion, galgame_rating.client_id, galgame_rating.site,
			       galgame_rating.written_here, galgame_rating.created, galgame_rating.updated)
			  IS DISTINCT FROM
			      (EXCLUDED.work_id, EXCLUDED.user_id, EXCLUDED.overall, EXCLUDED.art, EXCLUDED.story, EXCLUDED.music,
			       EXCLUDED.character, EXCLUDED.route, EXCLUDED.system, EXCLUDED.voice, EXCLUDED.replay_value,
			       EXCLUDED.recommend, EXCLUDED.spoiler_level, EXCLUDED.short_summary, EXCLUDED.author_state,
			       EXCLUDED.author_completion, EXCLUDED.client_id, EXCLUDED.site, EXCLUDED.written_here,
			       EXCLUDED.created, EXCLUDED.updated)`,
			r.ID, r.WorkID, r.UserID, r.Overall, r.Art, r.Story, r.Music, r.Character, r.Route, r.System,
			r.Voice, r.ReplayValue, r.Recommend, r.SpoilerLevel, r.ShortSummary, r.AuthorState,
			r.AuthorCompletion, r.ClientID, r.Site, r.WrittenHere, r.CreatedAt, r.UpdatedAt).Error
	})
}

// SetGameTypes stores the game types the author filed the work under, which
// only the forum keeps.
func (s *RatingStore) SetGameTypes(id int, types json.RawMessage) error {
	return s.db.Exec(`UPDATE galgame_rating SET galgame_type = ?::jsonb WHERE id = ?`, string(types), id).Error
}

// HideWork keeps a deleted work's ratings and hides them with the work: every
// rating read leaves out a work whose local row is not rendered, and the
// catalog mirror renders it again on restore.
func (s *RatingStore) HideWork(workID int) error {
	return s.db.Exec(`INSERT INTO galgame (id, updated, catalog_rendered) VALUES (?, now(), false)
		ON CONFLICT (id) DO UPDATE SET catalog_rendered = false`, workID).Error
}

// DropRating removes a rating NextMoe deleted; its likes cascade with it.
func (s *RatingStore) DropRating(id int) error {
	return s.db.Exec(`DELETE FROM galgame_rating WHERE id = ?`, id).Error
}

// DropUserRating removes the person's rating of a work after they cleared it.
func (s *RatingStore) DropUserRating(userID, workID int) error {
	return s.db.Exec(`DELETE FROM galgame_rating WHERE user_id = ? AND work_id = ?`, userID, workID).Error
}

// UserRatingIDs lists every rating the person has in the mirror.
func (s *RatingStore) UserRatingIDs(userID int) ([]int, error) {
	var ids []int
	err := s.db.Table("galgame_rating").Where("user_id = ?", userID).Order("id").Pluck("id", &ids).Error
	return ids, err
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
