package repository

import (
	"database/sql"
	"strings"
	"time"

	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/infrastructure/viewstats"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GalgameRepository struct {
	db *gorm.DB
}

func NewGalgameRepository(db *gorm.DB) *GalgameRepository {
	return &GalgameRepository{db: db}
}

func (r *GalgameRepository) DB() *gorm.DB {
	return r.db
}

type GalgameLocalRow struct {
	ID                 int       `gorm:"column:id"`
	LikeCount          int       `gorm:"column:like_count"`
	FavoriteCount      int       `gorm:"column:favorite_count"`
	View               int       `gorm:"column:view"`
	ResourceUpdateTime time.Time `gorm:"column:resource_update_time"`
	CreatorUserID      *int      `gorm:"column:creator_user_id"`
	Published          bool      `gorm:"column:published"`
}

func (r *GalgameRepository) FindLocal(id int) model.GalgameLocal {
	var row model.GalgameLocal
	r.db.Where("id = ?", id).First(&row)
	return row
}

func (r *GalgameRepository) FindLocalBatch(ids []int) map[int]GalgameLocalRow {
	if len(ids) == 0 {
		return map[int]GalgameLocalRow{}
	}
	var rows []GalgameLocalRow
	r.db.Table("galgame").Select("id, like_count, favorite_count, view, resource_update_time, creator_user_id, published").
		Where("id IN ?", ids).Scan(&rows)
	out := make(map[int]GalgameLocalRow, len(rows))
	for _, row := range rows {
		out[row.ID] = row
	}
	return out
}

// Owner, not actor. Catalog's by-uid claim face answers "every work this user
// TOUCHED" — approving someone else's submission puts it under the reviewer —
// so the profile used to list 13 entries a moderator had merely reviewed, and
// count them toward creator eligibility. The owner lives here.
func (r *GalgameRepository) PublishedIDsByCreator(userID, page, limit int) ([]int, int64, error) {
	base := r.db.Table("galgame").Where("published AND catalog_rendered AND creator_user_id = ?", userID)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var ids []int
	err := base.Order("created DESC").
		Offset((page-1)*limit).Limit(limit).
		Pluck("id", &ids).Error
	return ids, total, err
}

func (r *GalgameRepository) CountPublishedByCreatorSince(userID int, since time.Time) int {
	var n int64
	r.db.Table("galgame").
		Where("published AND creator_user_id = ? AND created >= ?", userID, since).
		Count(&n)
	return int(n)
}

func (r *GalgameRepository) IncrementView(id int) {
	r.db.Table("galgame").Where("id = ?", id).
		Update("view", gorm.Expr("view + 1"))
	_ = viewstats.BumpDaily(r.db, viewstats.GalgameDaily, id)
}

func (r *GalgameRepository) PublishLocal(tx *gorm.DB, workID int) error {
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.Assignments(map[string]any{"published": true}),
	}).Create(&model.GalgameLocal{ID: workID, Published: true}).Error
}

// The verify lane's next window: rows never checked first, then the ones
// checked longest ago. Every row catalog is asked about gets a checked_at, so a
// row catalog cannot answer moves to the back instead of heading every window.
func (r *GalgameRepository) MirrorVerifyIDs(limit int) []int {
	var ids []int
	r.db.Table("galgame").
		Order("catalog_checked_at ASC NULLS FIRST, id").
		Limit(limit).Pluck("id", &ids)
	return ids
}

func (r *GalgameRepository) LocalAmong(ids []int) ([]int, error) {
	var out []int
	if len(ids) == 0 {
		return out, nil
	}
	err := r.db.Table("galgame").
		Where("id = ANY(?::int[])", intArrayLit(ids)).
		Order("id").Pluck("id", &out).Error
	return out, err
}

func (r *GalgameRepository) RenderedAmong(ids []int) ([]int, error) {
	var out []int
	if len(ids) == 0 {
		return out, nil
	}
	err := r.db.Table("galgame").
		Where("id = ANY(?::int[]) AND catalog_rendered", intArrayLit(ids)).
		Order("id").Pluck("id", &out).Error
	return out, err
}

func (r *GalgameRepository) MarkCatalogChecked(ids []int, rendered bool) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Exec(
		"UPDATE galgame SET catalog_checked_at = now(), catalog_rendered = ? WHERE id = ANY(?::int[])",
		rendered, intArrayLit(ids),
	).Error
}

// SetReleaseDates writes catalog's dates onto the local rows, as date strings so
// no timezone ever touches a day. An empty string is catalog answering "this
// work has no date": the row is still marked confirmed, or the fill lane comes
// back for every TBA work on every tick.
func (r *GalgameRepository) SetReleaseDates(dates map[int]string) (int64, error) {
	if len(dates) == 0 {
		return 0, nil
	}
	rows := make([]string, 0, len(dates))
	args := make([]any, 0, len(dates)*2)
	for workID, date := range dates {
		rows = append(rows, "(?::int, ?::date)")
		args = append(args, workID, sql.NullString{String: date, Valid: date != ""})
	}
	res := r.db.Exec(`UPDATE galgame g SET release_date = v.d, release_date_synced_at = now()
		FROM (VALUES `+strings.Join(rows, ",")+`) AS v(id, d)
		WHERE g.id = v.id
		  AND (g.release_date IS DISTINCT FROM v.d OR g.release_date_synced_at IS NULL)`, args...)
	return res.RowsAffected, res.Error
}

func (r *GalgameRepository) SetContentLimits(idsByLimit map[string][]int) (int64, error) {
	var affected int64
	for limit, ids := range idsByLimit {
		if len(ids) == 0 {
			continue
		}
		res := r.db.Exec(
			"UPDATE galgame SET content_limit = ? WHERE id = ANY(?::int[]) AND content_limit IS DISTINCT FROM ?",
			limit, intArrayLit(ids), limit,
		)
		if res.Error != nil {
			return affected, res.Error
		}
		affected += res.RowsAffected
	}
	return affected, nil
}

func (r *GalgameRepository) UnpublishLocal(workID int) error {
	return r.db.Model(&model.GalgameLocal{}).Where("id = ?", workID).
		UpdateColumn("published", false).Error
}

// DeleteLocalDraft is the only cleanup a deleted draft will ever get — catalog's
// delete writes no claim event, so no cron comes along behind it. It refuses
// rather than cascades: galgame_resource is ON DELETE CASCADE, and a draft claim
// carrying a published resource is reachable, because publishing a resource sets
// `published` without moving the claim state.
func (r *GalgameRepository) DeleteLocalDraft(workID int) error {
	return r.db.Exec(`DELETE FROM galgame WHERE id = ?
		AND NOT EXISTS (SELECT 1 FROM galgame_resource r WHERE r.work_id = galgame.id)`,
		workID).Error
}

func (r *GalgameRepository) EnsureLocalStub(tx *gorm.DB, workID int) error {
	return tx.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&model.GalgameLocal{ID: workID}).Error
}

// Two callers only: SubmitLocal, and the claim feed's approval branch. The
// column is the 066 migration's frozen wiki-era submitter, and every other
// caller has been wrong — the resource lane called it on a first download link,
// so 2,073 pages ended up naming an author the retired wiki does not.
func (r *GalgameRepository) SetCreatorIfUnset(tx *gorm.DB, workID, userID int) error {
	return tx.Model(&model.GalgameLocal{}).
		Where("id = ? AND creator_user_id IS NULL", workID).
		UpdateColumn("creator_user_id", userID).Error
}

func (r *GalgameRepository) Touch(tx *gorm.DB, workID int) error {
	if err := r.EnsureLocalStub(tx, workID); err != nil {
		return err
	}
	return tx.Model(&model.GalgameLocal{}).Where("id = ?", workID).
		UpdateColumn("resource_update_time", time.Now()).Error
}

func (r *GalgameRepository) SubmitLocal(tx *gorm.DB, workID, userID int) error {
	if err := r.EnsureLocalStub(tx, workID); err != nil {
		return err
	}
	return r.SetCreatorIfUnset(tx, workID, userID)
}
