package repository

import (
	"errors"
	"slices"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Categories is the display order of the friend-link shelves.
var Categories = []string{"official", "galgame", "others"}

const categoryRank = "array_position(ARRAY['official','galgame','others']::text[], category::text)"

type LinkRow struct {
	ID              int    `gorm:"column:id"`
	Category        string `gorm:"column:category"`
	Rank            int    `gorm:"column:rank"`
	Name            string `gorm:"column:name"`
	Link            string `gorm:"column:link"`
	Description     string `gorm:"column:description"`
	BannerImageHash string `gorm:"column:banner_image_hash"`
	Status          string `gorm:"column:status"`
	SortOrder       int    `gorm:"column:sort_order"`
}

const linkColumns = "id, category, " + categoryRank + " AS rank, name, link, description, banner_image_hash, status, sort_order"

type LinkPos struct {
	Rank      int
	SortOrder int
	ID        int
}

func (r *FriendLinkRepository) ListKeyset(category string, after *LinkPos, limit int) ([]LinkRow, error) {
	q := r.db.Table("friend_link").Select(linkColumns)
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if after != nil {
		q = q.Where("("+categoryRank+", sort_order, id) > (?, ?, ?)", after.Rank, after.SortOrder, after.ID)
	}
	var rows []LinkRow
	err := q.Order(categoryRank + " ASC, sort_order ASC, id ASC").Limit(limit + 1).Scan(&rows).Error
	return rows, err
}

func (r *FriendLinkRepository) FindRow(id int) (*LinkRow, error) {
	var row LinkRow
	if err := r.db.Table("friend_link").Select(linkColumns).Where("id = ?", id).Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

type LinkWrite struct {
	Category        string
	Name            string
	Link            string
	Description     string
	BannerImageHash string
	Status          string
}

func nextPosition(tx *gorm.DB, category string) (int, error) {
	var next int
	err := tx.Raw(`SELECT COALESCE(MAX(sort_order), -1) + 1 FROM friend_link WHERE category = ?`, category).Scan(&next).Error
	return next, err
}

func (r *FriendLinkRepository) CreateLink(w LinkWrite) (int, error) {
	var id int
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`LOCK TABLE friend_link IN SHARE ROW EXCLUSIVE MODE`).Error; err != nil {
			return err
		}
		pos, err := nextPosition(tx, w.Category)
		if err != nil {
			return err
		}
		return tx.Raw(`INSERT INTO friend_link (category, name, link, description, banner, banner_image_hash, status, sort_order, created, updated)
			VALUES (?, ?, ?, ?, '', ?, ?, ?, now(), now()) RETURNING id`,
			w.Category, w.Name, w.Link, w.Description, w.BannerImageHash, w.Status, pos).Scan(&id).Error
	})
	return id, err
}

type LinkChanges struct {
	Category        *string
	Name            *string
	Link            *string
	Description     *string
	BannerImageHash *string
	Status          *string
}

// UpdateLink moves a link that changes shelf to the end of its new shelf. A
// missing link is left for the caller's reread to report.
func (r *FriendLinkRepository) UpdateLink(id int, ch LinkChanges) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`LOCK TABLE friend_link IN SHARE ROW EXCLUSIVE MODE`).Error; err != nil {
			return err
		}
		var cur LinkRow
		err := tx.Table("friend_link").Select(linkColumns).Where("id = ?", id).Take(&cur).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		updates := map[string]any{}
		set := func(col string, next *string, stored string) {
			if next != nil && *next != stored {
				updates[col] = *next
			}
		}
		set("name", ch.Name, cur.Name)
		set("link", ch.Link, cur.Link)
		set("description", ch.Description, cur.Description)
		set("banner_image_hash", ch.BannerImageHash, cur.BannerImageHash)
		set("status", ch.Status, cur.Status)
		if ch.Category != nil && *ch.Category != cur.Category {
			pos, err := nextPosition(tx, *ch.Category)
			if err != nil {
				return err
			}
			updates["category"] = *ch.Category
			updates["sort_order"] = pos
		}
		if len(updates) == 0 {
			return nil
		}
		updates["updated"] = gorm.Expr("now()")
		return tx.Table("friend_link").Where("id = ?", id).UpdateColumns(updates).Error
	})
}

func (r *FriendLinkRepository) DeleteLink(id int) (bool, error) {
	res := r.db.Exec(`DELETE FROM friend_link WHERE id = ?`, id)
	return res.RowsAffected > 0, res.Error
}

type ReorderResult struct {
	UnknownAt int
	Count     int
}

// ReorderShelf writes positions 0…n-1 on one shelf. The list must name every
// link on that shelf exactly once; nothing is written otherwise.
func (r *FriendLinkRepository) ReorderShelf(category string, ids []int) (ReorderResult, bool, error) {
	res := ReorderResult{UnknownAt: -1}
	applied := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var existing []int
		if err := tx.Table("friend_link").Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("category = ?", category).Order("id").Pluck("id", &existing).Error; err != nil {
			return err
		}
		res.Count = len(existing)
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
			if err := tx.Exec(`UPDATE friend_link SET sort_order = ? WHERE id = ?`, pos, id).Error; err != nil {
				return err
			}
		}
		applied = true
		return nil
	})
	return res, applied, err
}
