package model

import "time"

// restricted ("visible to named users") went with the cutover to catalog
// folders, which know only private and public. Production held 0 restricted
// collections and 0 rows in galgame_collection_viewer on the day it was
// removed, so nothing had to be migrated — the option had been offered in the
// edit modal since the feature shipped and nobody ever used it.
const (
	CollectionPublic  = "public"
	CollectionPrivate = "private"
)

type GalgameCollection struct {
	ID          int    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      int    `gorm:"column:user_id;not null" json:"user_id"`
	Name        string `gorm:"column:name;type:varchar(60);not null" json:"name"`
	Description string `gorm:"column:description;type:varchar(500);not null;default:''" json:"description"`
	Visibility  string `gorm:"column:visibility;type:varchar(16);not null;default:'public'" json:"visibility"`
	IsDefault   bool   `gorm:"column:is_default;not null;default:false" json:"is_default"`
	ItemCount   int    `gorm:"column:item_count;not null;default:0" json:"item_count"`

	// The catalog folder this row is an alias for. Everything above it is
	// frozen content, kept as the rollback material for the guarded retirement
	// and read by nothing since migration 091.
	CatalogFolderID *int64 `gorm:"column:catalog_folder_id" json:"-"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameCollection) TableName() string { return "galgame_collection" }

type GalgameCollectionItem struct {
	ID           int `gorm:"primaryKey;autoIncrement" json:"id"`
	CollectionID int `gorm:"column:collection_id;not null;uniqueIndex:idx_gci_unique" json:"collection_id"`
	GalgameID    int `gorm:"column:galgame_id;not null;uniqueIndex:idx_gci_unique" json:"galgame_id"`
	UserID       int `gorm:"column:user_id;not null" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameCollectionItem) TableName() string { return "galgame_collection_item" }

