package repository

import (
	"errors"
	"time"

	"kun-galgame-api/internal/website/model"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("website: row not found")

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Ready() bool {
	return s != nil && s.db != nil
}

func notFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

type WebsiteRow struct {
	model.GalgameWebsite
	Score int `gorm:"column:score"`
}

const scoreColumn = `COALESCE((SELECT SUM(t.level) FROM galgame_website_tag_relation r
	JOIN galgame_website_tag t ON t.id = r.galgame_website_tag_id
	WHERE r.galgame_website_id = galgame_website.id), 0) AS score`

type WebsiteFilter struct {
	IncludeNSFW bool
	CategoryID  *int
	TagID       *int
}

type TimePos struct {
	Created time.Time
	ID      int
}

func (f WebsiteFilter) apply(q *gorm.DB) *gorm.DB {
	if !f.IncludeNSFW {
		q = q.Where("galgame_website.age_limit = ?", "all")
	}
	if f.CategoryID != nil {
		q = q.Where("galgame_website.category_id = ?", *f.CategoryID)
	}
	if f.TagID != nil {
		q = q.Where(`EXISTS (SELECT 1 FROM galgame_website_tag_relation r
			WHERE r.galgame_website_id = galgame_website.id AND r.galgame_website_tag_id = ?)`, *f.TagID)
	}
	return q
}

func (s *Store) websites() *gorm.DB {
	return s.db.Model(&model.GalgameWebsite{}).Select("galgame_website.*, " + scoreColumn)
}

func (s *Store) ListWebsites(f WebsiteFilter, pos *TimePos, limit int) ([]WebsiteRow, error) {
	q := f.apply(s.websites())
	if pos != nil {
		q = q.Where("(galgame_website.created < ? OR (galgame_website.created = ? AND galgame_website.id < ?))",
			pos.Created, pos.Created, pos.ID)
	}
	var rows []WebsiteRow
	err := q.Order("galgame_website.created DESC, galgame_website.id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (s *Store) CountWebsites(f WebsiteFilter) (int, error) {
	var n int64
	err := f.apply(s.db.Model(&model.GalgameWebsite{})).Count(&n).Error
	return int(n), err
}

func (s *Store) FindWebsiteByHost(host string) (*WebsiteRow, error) {
	var row WebsiteRow
	err := s.websites().Where("galgame_website.url = ?", host).Take(&row).Error
	if err != nil {
		return nil, notFound(err)
	}
	return &row, nil
}

func (s *Store) FindWebsite(id int) (*WebsiteRow, error) {
	var row WebsiteRow
	err := s.websites().Where("galgame_website.id = ?", id).Take(&row).Error
	if err != nil {
		return nil, notFound(err)
	}
	return &row, nil
}

// A counter bump must not touch updated: through GORM's Update it did, so the
// page's 更新于 and JSON-LD dateModified were the last time anyone viewed it.
func (s *Store) IncrementView(id int) error {
	return s.db.Exec(`UPDATE galgame_website SET view = view + 1 WHERE id = ?`, id).Error
}

func (s *Store) WebsiteTagIDs(websiteID int) ([]int, error) {
	var ids []int
	err := s.db.Table("galgame_website_tag_relation").Where("galgame_website_id = ?", websiteID).
		Order("galgame_website_tag_id").Pluck("galgame_website_tag_id", &ids).Error
	return ids, err
}

func (s *Store) WebsiteTags(websiteID int) ([]model.GalgameWebsiteTag, error) {
	var tags []model.GalgameWebsiteTag
	err := s.db.Table("galgame_website_tag AS t").Select("t.*").
		Joins("JOIN galgame_website_tag_relation r ON r.galgame_website_tag_id = t.id").
		Joins("LEFT JOIN galgame_website_tag_group g ON g.id = t.group_id").
		Where("r.galgame_website_id = ?", websiteID).
		Order("g.sort_order ASC NULLS LAST, g.id ASC NULLS LAST, t.level DESC, t.id ASC").
		Find(&tags).Error
	return tags, err
}

type Engagement struct {
	LikeCount     int
	FavoriteCount int
	HasLiked      bool
	HasFavorited  bool
}

func (s *Store) Engagement(websiteID, userID int) (*Engagement, error) {
	var e Engagement
	err := s.db.Raw(`SELECT like_count, favorite_count,
		EXISTS (SELECT 1 FROM galgame_website_like WHERE website_id = w.id AND user_id = ?) AS has_liked,
		EXISTS (SELECT 1 FROM galgame_website_favorite WHERE website_id = w.id AND user_id = ?) AS has_favorited
		FROM galgame_website w WHERE w.id = ?`, userID, userID, websiteID).Scan(&e).Error
	return &e, err
}

type Slot string

const (
	SlotLike     Slot = "like"
	SlotFavorite Slot = "favorite"
)

func (sl Slot) tables() (rows, counter string) {
	if sl == SlotLike {
		return "galgame_website_like", "like_count"
	}
	return "galgame_website_favorite", "favorite_count"
}

// SetSlot is idempotent: the counter moves only when a row was really inserted
// or deleted, so a replayed PUT or a DELETE of nothing leaves it alone.
func (s *Store) SetSlot(slot Slot, websiteID, userID int, on bool) error {
	rows, counter := slot.tables()
	return s.db.Transaction(func(tx *gorm.DB) error {
		var res *gorm.DB
		delta := 1
		if on {
			res = tx.Exec(`INSERT INTO `+rows+` (user_id, website_id, created, updated) VALUES (?, ?, now(), now())
				ON CONFLICT (user_id, website_id) DO NOTHING`, userID, websiteID)
		} else {
			delta = -1
			res = tx.Exec(`DELETE FROM `+rows+` WHERE user_id = ? AND website_id = ?`, userID, websiteID)
		}
		if res.Error != nil || res.RowsAffected == 0 {
			return res.Error
		}
		return tx.Exec(`UPDATE galgame_website SET `+counter+` = `+counter+` + ? WHERE id = ?`, delta, websiteID).Error
	})
}

type Conflict struct {
	Host  bool
	Title bool
}

func (s *Store) WebsiteConflict(host, title string, excludeID int) (Conflict, error) {
	var rows []struct {
		URL  string
		Name string
	}
	q := s.db.Table("galgame_website").Select("url, name").Where("url = ? OR name = ?", host, title)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Scan(&rows).Error; err != nil {
		return Conflict{}, err
	}
	var c Conflict
	for _, r := range rows {
		c.Host = c.Host || r.URL == host
		c.Title = c.Title || r.Name == title
	}
	return c, nil
}

func replaceTags(tx *gorm.DB, websiteID int, tagIDs []int) error {
	if err := tx.Exec(`DELETE FROM galgame_website_tag_relation WHERE galgame_website_id = ?`, websiteID).Error; err != nil {
		return err
	}
	for _, id := range tagIDs {
		if err := tx.Exec(`INSERT INTO galgame_website_tag_relation (galgame_website_id, galgame_website_tag_id, created, updated)
			VALUES (?, ?, now(), now())`, websiteID, id).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) CreateWebsite(row *model.GalgameWebsite, tagIDs []int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(row).Error; err != nil {
			return err
		}
		return replaceTags(tx, row.ID, tagIDs)
	})
}

// PatchWebsite writes fields and, when tagIDs is non-nil, replaces the tag set.
func (s *Store) PatchWebsite(id int, fields map[string]any, tagIDs []int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		fields["updated"] = time.Now()
		res := tx.Model(&model.GalgameWebsite{}).Where("id = ?", id).UpdateColumns(fields)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		if tagIDs == nil {
			return nil
		}
		return replaceTags(tx, id, tagIDs)
	})
}

func (s *Store) DeleteWebsite(id int) error {
	res := s.db.Where("id = ?", id).Delete(&model.GalgameWebsite{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ExistingCategory(id int) (bool, error) {
	var n int64
	err := s.db.Model(&model.GalgameWebsiteCategory{}).Where("id = ?", id).Count(&n).Error
	return n > 0, err
}

func (s *Store) TagsByIDs(ids []int) ([]model.GalgameWebsiteTag, error) {
	var tags []model.GalgameWebsiteTag
	if len(ids) == 0 {
		return tags, nil
	}
	err := s.db.Where("id IN ?", ids).Find(&tags).Error
	return tags, err
}

func (s *Store) GroupsByIDs(ids []int) ([]model.GalgameWebsiteTagGroup, error) {
	var groups []model.GalgameWebsiteTagGroup
	if len(ids) == 0 {
		return groups, nil
	}
	err := s.db.Where("id IN ?", ids).Find(&groups).Error
	return groups, err
}

func (s *Store) CategoriesByIDs(ids []int) (map[int]model.GalgameWebsiteCategory, error) {
	out := map[int]model.GalgameWebsiteCategory{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []model.GalgameWebsiteCategory
	if err := s.db.Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, c := range rows {
		out[c.ID] = c
	}
	return out, nil
}
