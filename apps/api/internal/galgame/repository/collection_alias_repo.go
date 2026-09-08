package repository

import (
	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
)

// The catalog owns a collection's content from the cutover on; galgame_collection
// survives as the ALIAS that maps a forum collection id — the id in every
// /galgame/collection/<id> link ever shared — to the catalog folder behind it.
// Its content columns are frozen at cutover and read by nothing; see migration
// 091.

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

// AliasesForFolders maps catalog folder ids back to forum ids, minting a row
// for any folder that has none. A folder can arrive without one: the moyu
// backfill created 3,634 default folders that never passed through this site,
// and the patch site creates more every day.
func (r *GalgameCollectionRepository) AliasesForFolders(ownerID int, folderIDs []int64) (map[int64]int, error) {
	out := map[int64]int{}
	if len(folderIDs) == 0 {
		return out, nil
	}
	var rows []model.GalgameCollection
	if err := r.db.Select("id, catalog_folder_id").
		Where("catalog_folder_id IN ?", folderIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.CatalogFolderID != nil {
			out[*row.CatalogFolderID] = row.ID
		}
	}
	for _, fid := range folderIDs {
		if _, ok := out[fid]; ok {
			continue
		}
		id, err := r.MintAlias(r.db, ownerID, fid)
		if err != nil {
			return nil, err
		}
		out[fid] = id
	}
	return out, nil
}

func (r *GalgameCollectionRepository) MintAlias(tx *gorm.DB, ownerID int, folderID int64) (int, error) {
	row := model.GalgameCollection{UserID: ownerID, CatalogFolderID: &folderID}
	if err := tx.Create(&row).Error; err != nil {
		// A concurrent request may have minted the same alias; the partial
		// unique index makes that a conflict rather than a duplicate row.
		var existing model.GalgameCollection
		if e := tx.Select("id").Where("catalog_folder_id = ?", folderID).First(&existing).Error; e == nil {
			return existing.ID, nil
		}
		return 0, err
	}
	return row.ID, nil
}

func (r *GalgameCollectionRepository) DeleteAlias(id int) error {
	return r.db.Where("id = ?", id).Delete(&model.GalgameCollection{}).Error
}

func (r *GalgameCollectionRepository) FolderIDsOwnedBy(userID int, ids []int) (map[int]int64, error) {
	out := map[int]int64{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []model.GalgameCollection
	if err := r.db.Select("id, catalog_folder_id").
		Where("user_id = ? AND id IN ? AND catalog_folder_id IS NOT NULL", userID, ids).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = *row.CatalogFolderID
	}
	return out, nil
}
