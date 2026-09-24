package repository

import (
	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
)

// The catalog owns a collection's content from the cutover on; galgame_collection
// survives as the ALIAS that maps a forum collection id — the id in every
// /galgame/collection/<id> link ever shared — to the catalog folder behind it.
// Its content columns are frozen at cutover and read by nothing; see migration
// 091. v1 never mints a row; GET /collection-aliases/{alias_id} is read-only.

type GalgameCollectionRepository struct {
	db *gorm.DB
}

func NewGalgameCollectionRepository(db *gorm.DB) *GalgameCollectionRepository {
	return &GalgameCollectionRepository{db: db}
}

type CollectionAlias struct {
	ID              int
	UserID          int
	CatalogFolderID int64
}

func (r *GalgameCollectionRepository) AliasByID(id int) (*CollectionAlias, error) {
	var row model.GalgameCollection
	if err := r.db.Select("id, user_id, catalog_folder_id").
		Where("id = ? AND catalog_folder_id IS NOT NULL", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &CollectionAlias{ID: row.ID, UserID: row.UserID, CatalogFolderID: *row.CatalogFolderID}, nil
}
