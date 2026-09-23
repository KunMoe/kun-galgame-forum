package repository

import (
	"time"

	"kun-galgame-api/internal/website/model"

	"gorm.io/gorm"
)

type OrderPos struct {
	SortOrder int
	ID        int
}

func afterOrder(q *gorm.DB, pos *OrderPos) *gorm.DB {
	if pos == nil {
		return q
	}
	return q.Where("(sort_order > ? OR (sort_order = ? AND id > ?))", pos.SortOrder, pos.SortOrder, pos.ID)
}

type CategoryRow struct {
	model.GalgameWebsiteCategory
	WebsiteCount int `gorm:"column:website_count"`
}

const websiteCountColumn = `(SELECT COUNT(*) FROM galgame_website w WHERE w.category_id = galgame_website_category.id) AS website_count`

func (s *Store) categories() *gorm.DB {
	return s.db.Model(&model.GalgameWebsiteCategory{}).Select("galgame_website_category.*, " + websiteCountColumn)
}

func (s *Store) ListCategories(pos *OrderPos, limit int) ([]CategoryRow, error) {
	var rows []CategoryRow
	err := afterOrder(s.categories(), pos).Order("sort_order ASC, id ASC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (s *Store) FindCategoryBySlug(slug string) (*CategoryRow, error) {
	var row CategoryRow
	if err := s.categories().Where("name = ?", slug).Take(&row).Error; err != nil {
		return nil, notFound(err)
	}
	return &row, nil
}

func (s *Store) FindCategory(id int) (*CategoryRow, error) {
	var row CategoryRow
	if err := s.categories().Where("id = ?", id).Take(&row).Error; err != nil {
		return nil, notFound(err)
	}
	return &row, nil
}

func (s *Store) ListTagGroups(pos *OrderPos, limit int) ([]model.GalgameWebsiteTagGroup, error) {
	var rows []model.GalgameWebsiteTagGroup
	err := afterOrder(s.db.Model(&model.GalgameWebsiteTagGroup{}), pos).Order("sort_order ASC, id ASC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (s *Store) FindTagGroup(id int) (*model.GalgameWebsiteTagGroup, error) {
	var row model.GalgameWebsiteTagGroup
	if err := s.db.Where("id = ?", id).Take(&row).Error; err != nil {
		return nil, notFound(err)
	}
	return &row, nil
}

func (s *Store) ListTags(afterID int, limit int) ([]model.GalgameWebsiteTag, error) {
	var rows []model.GalgameWebsiteTag
	err := s.db.Where("id > ?", afterID).Order("id ASC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (s *Store) FindTagBySlug(slug string) (*model.GalgameWebsiteTag, error) {
	var row model.GalgameWebsiteTag
	if err := s.db.Where("name = ?", slug).Take(&row).Error; err != nil {
		return nil, notFound(err)
	}
	return &row, nil
}

func (s *Store) FindTag(id int) (*model.GalgameWebsiteTag, error) {
	var row model.GalgameWebsiteTag
	if err := s.db.Where("id = ?", id).Take(&row).Error; err != nil {
		return nil, notFound(err)
	}
	return &row, nil
}

// SlugTaken reports whether another row of table already uses slug; the
// vocabularies keep their URL key in the name column.
func (s *Store) SlugTaken(table, slug string, excludeID int) (bool, error) {
	q := s.db.Table(table).Where("name = ?", slug)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var n int64
	err := q.Count(&n).Error
	return n > 0, err
}

func (s *Store) Create(row any) error {
	return s.db.Create(row).Error
}

func (s *Store) Patch(table string, id int, fields map[string]any) error {
	fields["updated"] = time.Now()
	res := s.db.Table(table).Where("id = ?", id).UpdateColumns(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Delete(table string, id int) error {
	res := s.db.Exec(`DELETE FROM `+table+` WHERE id = ?`, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CountCategoryWebsites(id int) (int, error) {
	var n int64
	err := s.db.Model(&model.GalgameWebsite{}).Where("category_id = ?", id).Count(&n).Error
	return int(n), err
}
