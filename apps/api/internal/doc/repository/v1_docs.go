package repository

import (
	"errors"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrSlugTaken       = errors.New("doc: slug already in use")
	ErrUnknownCategory = errors.New("doc: no doc_category row with that slug")
)

type DocRow struct {
	ID              int        `gorm:"column:id"`
	Slug            string     `gorm:"column:slug"`
	Title           string     `gorm:"column:title"`
	Description     string     `gorm:"column:description"`
	BannerImageHash string     `gorm:"column:banner_image_hash"`
	IsPin           bool       `gorm:"column:is_pin"`
	View            int        `gorm:"column:view"`
	SortOrder       int        `gorm:"column:sort_order"`
	PublishedTime   time.Time  `gorm:"column:published_time"`
	EditedTime      *time.Time `gorm:"column:edited_time"`
	AuthorID        int        `gorm:"column:author_id"`
	CategorySlug    string     `gorm:"column:category_slug"`
	ContentMarkdown string     `gorm:"column:content_markdown"`
}

const docSummaryColumns = "a.id, a.slug, a.title, a.description, a.banner_image_hash, a.is_pin, a.view, " +
	"a.sort_order, a.published_time, a.edited_time, a.author_id, c.slug AS category_slug"

type DocSort int

const (
	DocSortPosition DocSort = iota
	DocSortPublished
	DocSortViews
)

type DocPos struct {
	Int  int64
	Time time.Time
	ID   int
}

type DocKeysetQuery struct {
	Sort     DocSort
	Category string
	Pinned   *bool
	After    *DocPos
	Limit    int
}

type DocRepository struct {
	db *gorm.DB
}

func NewDocRepository(db *gorm.DB) *DocRepository {
	return &DocRepository{db: db}
}

func (r *DocRepository) base(db *gorm.DB) *gorm.DB {
	return db.Table("doc_article a").Joins("JOIN doc_category c ON c.id = a.category_id")
}

// FindKeyset reads one row past Limit so the caller can tell whether a next
// page exists.
func (r *DocRepository) FindKeyset(q DocKeysetQuery) ([]DocRow, error) {
	tx := r.base(r.db).Select(docSummaryColumns)
	if q.Category != "" {
		tx = tx.Where("c.slug = ?", q.Category)
	}
	if q.Pinned != nil {
		tx = tx.Where("a.is_pin = ?", *q.Pinned)
	}
	switch q.Sort {
	case DocSortPublished:
		if q.After != nil {
			tx = tx.Where("(a.published_time, a.id) < (?, ?)", q.After.Time, q.After.ID)
		}
		tx = tx.Order("a.published_time DESC, a.id DESC")
	case DocSortViews:
		if q.After != nil {
			tx = tx.Where("(a.view, a.id) < (?, ?)", q.After.Int, q.After.ID)
		}
		tx = tx.Order("a.view DESC, a.id DESC")
	default:
		if q.After != nil {
			tx = tx.Where("(a.sort_order, a.id) > (?, ?)", q.After.Int, q.After.ID)
		}
		tx = tx.Order("a.sort_order ASC, a.id ASC")
	}
	var rows []DocRow
	err := tx.Limit(q.Limit + 1).Scan(&rows).Error
	return rows, err
}

func (r *DocRepository) find(db *gorm.DB, where string, arg any) (*DocRow, error) {
	var row DocRow
	err := r.base(db).
		Select(docSummaryColumns+", a.content_markdown").
		Where(where, arg).
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *DocRepository) FindBySlug(slug string) (*DocRow, error) {
	return r.find(r.db, "a.slug = ?", slug)
}

func (r *DocRepository) FindByID(id int) (*DocRow, error) {
	return r.find(r.db, "a.id = ?", id)
}

// CountView bumps the counter without touching `updated`: the legacy
// IncrementView went through GORM's Update, which also stamps updated_at, so
// that column became "last viewed" on every doc.
func (r *DocRepository) CountView(id int) (int, error) {
	var view int
	err := r.db.Raw(`UPDATE doc_article SET view = view + 1 WHERE id = ? RETURNING view`, id).Scan(&view).Error
	return view, err
}

func (r *DocRepository) categoryID(tx *gorm.DB, slug string) (int, error) {
	var ids []int
	if err := tx.Table("doc_category").Where("slug = ?", slug).Pluck("id", &ids).Error; err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, ErrUnknownCategory
	}
	return ids[0], nil
}

type DocWrite struct {
	Slug            string
	Title           string
	Description     string
	Category        string
	BannerImageHash string
	IsPin           bool
	ContentMarkdown string
	AuthorID        int
}

func (r *DocRepository) Create(w DocWrite) (int, error) {
	var id int
	err := r.db.Transaction(func(tx *gorm.DB) error {
		catID, err := r.categoryID(tx, w.Category)
		if err != nil {
			return err
		}
		return tx.Raw(`INSERT INTO doc_article (
			title, slug, path, description, banner_image_hash, is_pin, content_markdown,
			category_id, author_id, published_time, created, updated, sort_order
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, now(), now(), now(),
			(SELECT COALESCE(MAX(sort_order), -1) + 1 FROM doc_article))
		RETURNING id`,
			w.Title, w.Slug, "/doc/"+w.Slug, w.Description, w.BannerImageHash, w.IsPin, w.ContentMarkdown,
			catID, w.AuthorID).Scan(&id).Error
	})
	return id, slugTaken(err)
}

type DocChanges struct {
	Slug            *string
	Title           *string
	Description     *string
	Category        *string
	BannerImageHash *string
	IsPin           *bool
	ContentMarkdown *string
}

// Update applies the changes; a missing doc is left for the caller's reread to
// report. edited_time moves only when something other than the pin flag takes
// a new value.
func (r *DocRepository) Update(id int, ch DocChanges) error {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		cur, err := r.find(tx.Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "a"}}), "a.id = ?", id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		updates := map[string]any{}
		edited := false
		setText := func(col string, next *string, stored string) {
			if next != nil && *next != stored {
				updates[col] = *next
				edited = true
			}
		}
		setText("title", ch.Title, cur.Title)
		setText("description", ch.Description, cur.Description)
		setText("banner_image_hash", ch.BannerImageHash, cur.BannerImageHash)
		setText("content_markdown", ch.ContentMarkdown, cur.ContentMarkdown)
		if ch.Slug != nil && *ch.Slug != cur.Slug {
			updates["slug"] = *ch.Slug
			updates["path"] = "/doc/" + *ch.Slug
			edited = true
		}
		if ch.Category != nil && *ch.Category != cur.CategorySlug {
			catID, err := r.categoryID(tx, *ch.Category)
			if err != nil {
				return err
			}
			updates["category_id"] = catID
			edited = true
		}
		if ch.IsPin != nil && *ch.IsPin != cur.IsPin {
			updates["is_pin"] = *ch.IsPin
		}
		if len(updates) == 0 {
			return nil
		}
		if edited {
			updates["edited_time"] = gorm.Expr("now()")
		}
		updates["updated"] = gorm.Expr("now()")
		return tx.Table("doc_article").Where("id = ?", id).UpdateColumns(updates).Error
	})
	return slugTaken(err)
}

func (r *DocRepository) Delete(id int) (bool, error) {
	res := r.db.Exec(`DELETE FROM doc_article WHERE id = ?`, id)
	return res.RowsAffected > 0, res.Error
}

type ReorderResult struct {
	UnknownAt  int
	TotalCount int
}

// Reorder writes positions 0…n-1 in the given order. The list must name every
// doc exactly once: UnknownAt is the index of the first id that is not a doc
// (-1 when all are), and TotalCount is how many docs exist, so the caller can
// tell a stale list from an applied one. Nothing is written unless both hold.
func (r *DocRepository) Reorder(ids []int) (ReorderResult, bool, error) {
	res := ReorderResult{UnknownAt: -1}
	applied := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var existing []int
		if err := tx.Table("doc_article").Clauses(clause.Locking{Strength: "UPDATE"}).
			Order("id").Pluck("id", &existing).Error; err != nil {
			return err
		}
		res.TotalCount = len(existing)
		for i, id := range ids {
			if _, ok := slices.BinarySearch(existing, id); !ok {
				res.UnknownAt = i
				return nil
			}
		}
		if len(ids) != len(existing) {
			return nil
		}
		for pos, id := range ids {
			if err := tx.Exec(`UPDATE doc_article SET sort_order = ? WHERE id = ?`, pos, id).Error; err != nil {
				return err
			}
		}
		applied = true
		return nil
	})
	return res, applied, err
}

func slugTaken(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return ErrSlugTaken
	}
	return err
}
