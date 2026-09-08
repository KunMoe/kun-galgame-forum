package repository

import (
	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// What is left here after the catalog cutover is the local bookkeeping a
// collection write still owns: the galgame_local row and its favourite
// counter. Everything that used to read or write galgame_collection /
// galgame_collection_item moved to pkg/catalogclient; the alias half lives in
// collection_alias_repo.go.
type GalgameCollectionRepository struct {
	db *gorm.DB
}

func NewGalgameCollectionRepository(db *gorm.DB) *GalgameCollectionRepository {
	return &GalgameCollectionRepository{db: db}
}

func (r *GalgameCollectionRepository) DB() *gorm.DB { return r.db }

func (r *GalgameCollectionRepository) EnsureGalgameLocal(tx *gorm.DB, galgameID int) error {
	return tx.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&model.GalgameLocal{ID: galgameID}).Error
}

// galgame_local.favorite_count backs the "most favourited" ranking and nothing
// else — the number a reader sees on a game page comes from the catalog's
// nextmoe/favorites popularity row, which counts distinct people across every
// site. This counter moves only on writes made HERE, so a favourite added from
// the patch site does not reach it. That is a known, bounded divergence in a
// ranking's sort key; the cure is a catalog-side sort=popularity, not a second
// copy of the memberships.
func (r *GalgameCollectionRepository) AdjustGalgameFavoriteCount(tx *gorm.DB, galgameID, delta int) error {
	return tx.Model(&model.GalgameLocal{}).Where("id = ?", galgameID).
		Update("favorite_count", gorm.Expr("favorite_count + ?", delta)).Error
}

func (r *GalgameCollectionRepository) DecrementFavoriteCounts(tx *gorm.DB, galgameIDs []int) error {
	if len(galgameIDs) == 0 {
		return nil
	}
	return tx.Model(&model.GalgameLocal{}).Where("id IN ?", galgameIDs).
		Update("favorite_count", gorm.Expr("GREATEST(favorite_count - 1, 0)")).Error
}
