package repository

import (
	"fmt"
	"strconv"
	"strings"

	"kun-galgame-api/internal/galgame/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GalgameMergeRepository struct {
	db *gorm.DB
}

func NewGalgameMergeRepository(db *gorm.DB) *GalgameMergeRepository {
	return &GalgameMergeRepository{db: db}
}

type mergeTable struct {
	table, key, peer string
}

var mergeMovableTables = []mergeTable{
	{table: "galgame_resource", key: "id"},
	{table: "galgame_activity", key: "id"},
	{table: "galgame_comment_community_map", key: "old_comment_id"},
	{table: "galgame_resource_relocation", key: "resource_id"},
}

var mergeUniqueTables = []mergeTable{
	{table: "galgame_like", key: "id", peer: "user_id"},
	{table: "galgame_favorite", key: "id", peer: "user_id"},
	{table: "galgame_contributor", key: "id", peer: "user_id"},
	{table: "galgame_quiz_galgame", key: "quiz_id", peer: "quiz_id"},
}

type MergeCounts struct {
	FoldID   int64
	Moved    int64
	Dropped  int64
	Comments int
}

// ReferencedIDsIn narrows a page of catalog redirect ids to the ones the fold
// has something to move for. Asking only the galgame table left three quiz
// links and two feed rows on works merged on 2026-09-05 until a census found
// them on 2026-09-28: a work nobody opened on the forum has no local row.
func (r *GalgameMergeRepository) ReferencedIDsIn(ids []int) ([]int, error) {
	return r.referencedIn(ids, true)
}

func (r *GalgameMergeRepository) referencedBeyondRow(workID int) (bool, error) {
	ids, err := r.referencedIn([]int{workID}, false)
	return len(ids) > 0, err
}

func (r *GalgameMergeRepository) referencedIn(ids []int, withRow bool) ([]int, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	feed := "SELECT work_id FROM feed_activity WHERE work_id IN @ids"
	var sources []string
	if withRow {
		sources = append(sources, "SELECT id FROM galgame WHERE id IN @ids")
	} else {
		feed += " AND type <> 'GALGAME_CREATION'"
	}
	sources = append(sources,
		"SELECT entity_id FROM galgame_view_daily WHERE entity_id IN @ids",
		feed,
	)
	for _, t := range mergeMovableTables {
		sources = append(sources, fmt.Sprintf("SELECT work_id FROM %s WHERE work_id IN @ids", t.table))
	}
	for _, t := range mergeUniqueTables {
		sources = append(sources, fmt.Sprintf("SELECT work_id FROM %s WHERE work_id IN @ids", t.table))
	}
	var out []int
	err := r.db.Raw("SELECT id FROM ("+strings.Join(sources, " UNION ")+") x(id) ORDER BY id",
		map[string]any{"ids": ids}).Scan(&out).Error
	return out, err
}

func (r *GalgameMergeRepository) Fold(oldWorkID, newWorkID int) (MergeCounts, error) {
	var counts MergeCounts
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var err error
		counts, err = r.FoldTx(tx, oldWorkID, newWorkID)
		return err
	})
	if err != nil {
		return MergeCounts{}, err
	}
	return counts, nil
}

func (r *GalgameMergeRepository) FoldTx(tx *gorm.DB, oldWorkID, newWorkID int) (MergeCounts, error) {
	var counts MergeCounts
	if oldWorkID == newWorkID || oldWorkID <= 0 || newWorkID <= 0 {
		return counts, fmt.Errorf("拒绝合并 galgame %d -> %d", oldWorkID, newWorkID)
	}
	var dead model.GalgameLocal
	found := tx.Where("id = ?", oldWorkID).Limit(1).Find(&dead)
	if found.Error != nil {
		return counts, found.Error
	}
	hasRow := found.RowsAffected > 0

	if err := tx.Raw(`
		INSERT INTO galgame_merge_fold (old_work_id, new_work_id, old_row, survivor_before)
		VALUES (?, ?, (SELECT to_jsonb(g) FROM galgame g WHERE g.id = ?), (SELECT to_jsonb(g) FROM galgame g WHERE g.id = ?))
		RETURNING id`, oldWorkID, newWorkID, oldWorkID, newWorkID).Scan(&counts.FoldID).Error; err != nil {
		return counts, err
	}
	ledger := foldLedger{tx: tx, foldID: counts.FoldID, old: oldWorkID, new: newWorkID}

	if hasRow {
		counts.Comments = dead.CommentCount

		// Seeded from the dead row, not from GORM's defaults. ResourceUpdateTime
		// is autoCreateTime, so a survivor created here would be stamped now();
		// the GREATEST below then keeps now() and a 2021 resource sorts to the
		// top of 最新资源更新 as if it had just been posted. 11 of the first 30
		// merges land on a work id with no local row, so this is the common path.
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.GalgameLocal{
			ID:                 newWorkID,
			CreatedAt:          dead.CreatedAt,
			ResourceUpdateTime: dead.ResourceUpdateTime,
		}).Error; err != nil {
			return counts, err
		}
	}

	for _, t := range mergeMovableTables {
		n, err := ledger.move(t, "")
		if err != nil {
			return counts, err
		}
		counts.Moved += n
	}

	// Two contributors of the same game are one contributor after the fold,
	// and their edit counts add up: dropping the dead row instead would
	// silently reduce someone's revision_count on the survivor.
	if err := ledger.record(`
			SELECT 'galgame_contributor', 'summed', jsonb_build_object('id', t.id),
				jsonb_build_object('survivor', to_jsonb(t), 'absorbed', to_jsonb(s))
			FROM galgame_contributor t JOIN galgame_contributor s ON s.user_id = t.user_id
			WHERE s.work_id = @old AND t.work_id = @new`); err != nil {
		return counts, err
	}
	if err := tx.Exec(`
			UPDATE galgame_contributor t SET
				revision_count = t.revision_count + s.revision_count,
				first_at = LEAST(t.first_at, s.first_at),
				last_at  = GREATEST(t.last_at, s.last_at)
			FROM galgame_contributor s
			WHERE s.work_id = ? AND t.work_id = ? AND t.user_id = s.user_id`,
		oldWorkID, newWorkID).Error; err != nil {
		return counts, err
	}

	for _, t := range mergeUniqueTables {
		moved, err := ledger.move(t, fmt.Sprintf(
			"AND NOT EXISTS (SELECT 1 FROM %[1]s x WHERE x.work_id = @new AND x.%[2]s = %[1]s.%[2]s)", t.table, t.peer))
		if err != nil {
			return counts, err
		}
		counts.Moved += moved

		if err := ledger.record(fmt.Sprintf(`
				SELECT '%[1]s', 'dropped', jsonb_build_object('%[2]s', t.%[2]s), to_jsonb(t)
				FROM %[1]s t WHERE t.work_id = @old`, t.table, t.key)); err != nil {
			return counts, err
		}
		if err := archiveByGalgame(tx, t.table, oldWorkID, newWorkID); err != nil {
			return counts, err
		}
		dropped := tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE work_id = ?", t.table), oldWorkID)
		if dropped.Error != nil {
			return counts, dropped.Error
		}
		counts.Dropped += dropped.RowsAffected
	}

	// entity_id, not work_id, which is why a column sweep does not find
	// this one. Same-day buckets add rather than collide.
	if err := ledger.record(`
			SELECT 'galgame_view_daily', 'summed', jsonb_build_object('day', v.day), to_jsonb(v)
			FROM galgame_view_daily v WHERE v.entity_id = @old`); err != nil {
		return counts, err
	}
	if err := tx.Exec(`
			INSERT INTO galgame_view_daily (entity_id, day, count)
			SELECT ?, day, count FROM galgame_view_daily WHERE entity_id = ?
			ON CONFLICT (entity_id, day) DO UPDATE SET count = galgame_view_daily.count + EXCLUDED.count`,
		newWorkID, oldWorkID).Error; err != nil {
		return counts, err
	}
	if err := tx.Exec("DELETE FROM galgame_view_daily WHERE entity_id = ?", oldWorkID).Error; err != nil {
		return counts, err
	}

	if hasRow {
		// published is sticky since 078 and a ban must not be shed by merging
		// into an unbanned duplicate, so both fold as OR.
		if err := tx.Exec(`
				UPDATE galgame t SET
					view = t.view + s.view,
					published = t.published OR s.published,
					resource_publish_banned = t.resource_publish_banned OR s.resource_publish_banned,
					creator_user_id = COALESCE(t.creator_user_id, s.creator_user_id),
					created = LEAST(t.created, s.created),
					resource_update_time = GREATEST(t.resource_update_time, s.resource_update_time)
				FROM galgame s WHERE t.id = ? AND s.id = ?`, newWorkID, oldWorkID).Error; err != nil {
			return counts, err
		}

		if err := tx.Exec("DELETE FROM galgame WHERE id = ?", oldWorkID).Error; err != nil {
			return counts, err
		}
	}

	// The galgame comment has no source table in this database: the comment
	// itself lives in infra's community service and the forum writes this
	// feed row by hand (feedParityUpsert), so no trigger re-points it. The
	// feed read drops a row whose work id no longer resolves to a game, so on
	// 2026-09-05 two comments left the home feed and their authors'
	// timelines with nothing in the logs. Everything trigger-backed is
	// already on the survivor by now; this only catches what was missed.
	if err := ledger.record(`
			SELECT 'feed_activity', 'repointed', jsonb_build_object('id', f.id),
				jsonb_build_object('work_id', f.work_id, 'link', f.link)
			FROM feed_activity f WHERE f.work_id = @old`); err != nil {
		return counts, err
	}
	if err := tx.Exec(`
			UPDATE feed_activity SET work_id = ?,
				link = CASE WHEN link = ? THEN ? ELSE link END
			WHERE work_id = ?`,
		newWorkID, "/galgame/"+strconv.Itoa(oldWorkID), "/galgame/"+strconv.Itoa(newWorkID), oldWorkID).Error; err != nil {
		return counts, err
	}

	if err := recountAfterFold(tx, newWorkID); err != nil {
		return counts, err
	}
	if err := tx.Exec(`UPDATE galgame_merge_fold SET survivor_after = (SELECT to_jsonb(g) FROM galgame g WHERE g.id = ?)
		WHERE id = ?`, newWorkID, counts.FoldID).Error; err != nil {
		return counts, err
	}
	return counts, nil
}

type foldLedger struct {
	tx       *gorm.DB
	foldID   int64
	old, new int
}

func (l foldLedger) args() map[string]any {
	return map[string]any{"fold": l.foldID, "old": l.old, "new": l.new}
}

func (l foldLedger) record(query string) error {
	return l.tx.Exec(`
		INSERT INTO galgame_merge_fold_row (fold_id, table_name, action, row_key, before)
		SELECT @fold, x.* FROM (`+query+`) x(table_name, action, row_key, before)`, l.args()).Error
}

func (l foldLedger) move(t mergeTable, extra string) (int64, error) {
	res := l.tx.Exec(fmt.Sprintf(`
		WITH moved AS (
			UPDATE %[1]s SET work_id = @new WHERE work_id = @old %[3]s RETURNING %[2]s AS k
		)
		INSERT INTO galgame_merge_fold_row (fold_id, table_name, action, row_key)
		SELECT @fold, '%[1]s', 'moved', jsonb_build_object('%[2]s', k) FROM moved`, t.table, t.key, extra), l.args())
	return res.RowsAffected, res.Error
}

// archiveByGalgame copies every row a drop is about to delete into
// galgame_merge_discarded, so the fold never destroys something a user wrote.
// The rows to archive are always the dead game's, so oldWorkID is both the label
// and the filter — an earlier revision took them as separate parameters and a
// swapped call wrote every archive row with old and new the wrong way round,
// which reads fine and finds nothing when someone tries to recover from it.
func archiveByGalgame(tx *gorm.DB, table string, oldWorkID, newWorkID int) error {
	return tx.Exec(fmt.Sprintf(`
		INSERT INTO galgame_merge_discarded (old_work_id, new_work_id, table_name, row)
		SELECT ?, ?, ?, to_jsonb(t) FROM %s t WHERE t.work_id = ?`, table),
		oldWorkID, newWorkID, table, oldWorkID).Error
}

// favorite_count is absent for a related reason since the folder cutover:
// galgame_collection_item is frozen rollback material, so recomputing from it
// would replace the live counter with a snapshot, and the memberships that
// actually moved were rehung upstream by the catalog's own merge. The counter
// is maintained by this site's writes and is not recomputed here.
//
// comment_count is absent on purpose: it mirrors the community thread anchored
// at site_game:<work id>, which lives in infra and does not move when the forum
// folds two local rows. The survivor keeps its own count and the merge sync logs
// the abandoned thread.
func recountAfterFold(tx *gorm.DB, workID int) error {
	return tx.Exec(`
		UPDATE galgame SET
			like_count        = (SELECT COUNT(*) FROM galgame_like WHERE work_id = galgame.id),
			resource_count    = (SELECT COUNT(*) FROM galgame_resource WHERE work_id = galgame.id),
			rating_count      = (SELECT COUNT(*) FROM galgame_rating WHERE work_id = galgame.id),
			contributor_count = (SELECT COUNT(*) FROM galgame_contributor WHERE work_id = galgame.id),
			view_7d           = (SELECT COALESCE(SUM(count), 0) FROM galgame_view_daily
			                      WHERE entity_id = galgame.id AND day >= CURRENT_DATE - INTERVAL '6 days'),
			view_30d          = (SELECT COALESCE(SUM(count), 0) FROM galgame_view_daily
			                      WHERE entity_id = galgame.id AND day >= CURRENT_DATE - INTERVAL '29 days')
		WHERE id = ?`, workID).Error
}
