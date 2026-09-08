package repository

import (
	"fmt"
	"strconv"

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

// A child table whose galgame_id may simply be rewritten: nothing in it is
// unique per game, so both games' rows can coexist under the survivor.
var mergeMovableTables = []string{
	"galgame_resource",
	"galgame_activity",
	"galgame_quiz",
	"galgame_comment_community_map",
}

// A child table with a unique key over (galgame_id, peer). When the same peer
// already has a row under the survivor the two rows cannot both survive, so the
// dead game's row is dropped rather than moved — a user who liked both copies
// keeps one like, not two.
var mergeUniqueTables = []struct{ table, peer string }{
	{"galgame_like", "user_id"},
	{"galgame_favorite", "user_id"},
	{"galgame_rating", "user_id"},
	{"galgame_contributor", "user_id"},
	{"galgame_quiz_galgame", "quiz_id"},
}

type MergeCounts struct {
	Moved    int64
	Dropped  int64
	Comments int
}

// LocalIDsIn narrows a page of catalog redirect ids to the ones that are also a
// row in the local galgame table.
func (r *GalgameMergeRepository) LocalIDsIn(ids []int) []int {
	if len(ids) == 0 {
		return nil
	}
	var out []int
	r.db.Table("galgame").Where("id IN ?", ids).Order("id").Pluck("id", &out)
	return out
}

// Fold moves every row the dead gid owns onto the survivor, deletes the dead
// galgame row, and records where it went.
//
// Order is load-bearing. galgame_favorite / galgame_like / galgame_resource are
// ON DELETE CASCADE, so anything still pointing at the dead row when it is
// deleted is deleted with it; galgame_rating is ON DELETE RESTRICT, so a rating
// left behind aborts the whole transaction instead. Both are why the children
// move first and the parent goes last.
//
// Every source table here carries an AFTER UPDATE trigger that re-upserts its
// feed_activity row with the new galgame_id and link, and deleting the dead
// galgame row deletes its own GALGAME_CREATION entry — so the feed mostly takes
// care of itself. The sweep at the end is for what has no source table in this
// database at all.
func (r *GalgameMergeRepository) Fold(oldGID, newGID int) (MergeCounts, error) {
	var counts MergeCounts
	if oldGID == newGID || oldGID <= 0 || newGID <= 0 {
		return counts, fmt.Errorf("拒绝合并 galgame %d -> %d", oldGID, newGID)
	}

	err := r.db.Transaction(func(tx *gorm.DB) error {
		var dead model.GalgameLocal
		if err := tx.Where("id = ?", oldGID).First(&dead).Error; err != nil {
			return err
		}
		counts.Comments = dead.CommentCount

		// Seeded from the dead row, not from GORM's defaults. ResourceUpdateTime
		// is autoCreateTime, so a survivor created here would be stamped now();
		// the GREATEST below then keeps now() and a 2021 resource sorts to the
		// top of 最新资源更新 as if it had just been posted. 11 of the first 30
		// merges land on a gid with no local row, so this is the common path.
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.GalgameLocal{
			ID:                 newGID,
			CreatedAt:          dead.CreatedAt,
			ResourceUpdateTime: dead.ResourceUpdateTime,
		}).Error; err != nil {
			return err
		}

		for _, table := range mergeMovableTables {
			res := tx.Exec(fmt.Sprintf("UPDATE %s SET galgame_id = ? WHERE galgame_id = ?", table), newGID, oldGID)
			if res.Error != nil {
				return res.Error
			}
			counts.Moved += res.RowsAffected
		}

		// Two contributors of the same game are one contributor after the fold,
		// and their edit counts add up: dropping the dead row instead would
		// silently reduce someone's revision_count on the survivor.
		if err := tx.Exec(`
			UPDATE galgame_contributor t SET
				revision_count = t.revision_count + s.revision_count,
				first_at = LEAST(t.first_at, s.first_at),
				last_at  = GREATEST(t.last_at, s.last_at)
			FROM galgame_contributor s
			WHERE s.galgame_id = ? AND t.galgame_id = ? AND t.user_id = s.user_id`,
			oldGID, newGID).Error; err != nil {
			return err
		}

		if err := preferEngagedRating(tx, oldGID, newGID); err != nil {
			return err
		}

		for _, t := range mergeUniqueTables {
			moved := tx.Exec(fmt.Sprintf(`
				UPDATE %[1]s SET galgame_id = ? WHERE galgame_id = ?
				  AND NOT EXISTS (
					SELECT 1 FROM %[1]s x WHERE x.galgame_id = ? AND x.%[2]s = %[1]s.%[2]s)`,
				t.table, t.peer), newGID, oldGID, newGID)
			if moved.Error != nil {
				return moved.Error
			}
			counts.Moved += moved.RowsAffected

			if err := archiveByGalgame(tx, t.table, oldGID, newGID); err != nil {
				return err
			}
			dropped := tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE galgame_id = ?", t.table), oldGID)
			if dropped.Error != nil {
				return dropped.Error
			}
			counts.Dropped += dropped.RowsAffected
		}

		// entity_id, not galgame_id, which is why a column sweep does not find
		// this one. Same-day buckets add rather than collide.
		if err := tx.Exec(`
			INSERT INTO galgame_view_daily (entity_id, day, count)
			SELECT ?, day, count FROM galgame_view_daily WHERE entity_id = ?
			ON CONFLICT (entity_id, day) DO UPDATE SET count = galgame_view_daily.count + EXCLUDED.count`,
			newGID, oldGID).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM galgame_view_daily WHERE entity_id = ?", oldGID).Error; err != nil {
			return err
		}

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
			FROM galgame s WHERE t.id = ? AND s.id = ?`, newGID, oldGID).Error; err != nil {
			return err
		}

		if err := tx.Exec("DELETE FROM galgame WHERE id = ?", oldGID).Error; err != nil {
			return err
		}

		// The galgame comment has no source table in this database: the comment
		// itself lives in infra's community service and the forum writes this
		// feed row by hand (feedParityUpsert), so no trigger re-points it. The
		// feed read drops a row whose gid no longer resolves to a game, so on
		// 2026-09-05 two comments left the home feed and their authors'
		// timelines with nothing in the logs. Everything trigger-backed is
		// already on the survivor by now; this only catches what was missed.
		if err := tx.Exec(`
			UPDATE feed_activity SET galgame_id = ?,
				link = CASE WHEN link = ? THEN ? ELSE link END
			WHERE galgame_id = ?`,
			newGID, "/galgame/"+strconv.Itoa(oldGID), "/galgame/"+strconv.Itoa(newGID), oldGID).Error; err != nil {
			return err
		}

		if err := recountAfterFold(tx, newGID); err != nil {
			return err
		}

		if err := tx.Exec(`
			INSERT INTO galgame_redirect (old_gid, new_gid) VALUES (?, ?)
			ON CONFLICT (old_gid) DO UPDATE SET new_gid = EXCLUDED.new_gid`,
			oldGID, newGID).Error; err != nil {
			return err
		}
		// Catalog merges chain: a survivor can itself be merged later. Chase the
		// ledger forward so an old link still lands on the game that exists,
		// rather than on a gid that was deleted one merge ago.
		return tx.Exec("UPDATE galgame_redirect SET new_gid = ? WHERE new_gid = ? AND old_gid <> ?",
			newGID, oldGID, newGID).Error
	})
	if err != nil {
		return MergeCounts{}, err
	}
	return counts, nil
}

// archiveByGalgame copies every row a drop is about to delete into
// galgame_merge_discarded, so the fold never destroys something a user wrote.
// galgame_rating carries its likes inline because galgame_rating_like cascades
// off the rating and would vanish with it.
// The rows to archive are always the dead game's, so oldGID is both the label
// and the filter — an earlier revision took them as separate parameters and a
// swapped call wrote every archive row with old and new the wrong way round,
// which reads fine and finds nothing when someone tries to recover from it.
func archiveByGalgame(tx *gorm.DB, table string, oldGID, newGID int) error {
	if table == "galgame_rating" {
		return archiveRatings(tx, "galgame_id = ?", oldGID, oldGID, newGID)
	}
	return tx.Exec(fmt.Sprintf(`
		INSERT INTO galgame_merge_discarded (old_gid, new_gid, table_name, row)
		SELECT ?, ?, ?, to_jsonb(t) FROM %s t WHERE t.galgame_id = ?`, table),
		oldGID, newGID, table, oldGID).Error
}

func archiveRatings(tx *gorm.DB, where string, arg any, oldGID, newGID int) error {
	return tx.Exec(fmt.Sprintf(`
		INSERT INTO galgame_merge_discarded (old_gid, new_gid, table_name, row)
		SELECT ?, ?, 'galgame_rating',
		       to_jsonb(r) || jsonb_build_object('likes', COALESCE((
		         SELECT jsonb_agg(to_jsonb(l)) FROM galgame_rating_like l
		         WHERE l.galgame_rating_id = r.id), '[]'::jsonb))
		FROM galgame_rating r WHERE %s`, where), oldGID, newGID, arg).Error
}

// A user who reviewed both copies keeps one review of the one game that is
// left. Which one survives is not arbitrary: the drop below always favours the
// survivor's row, so hand the slot to the dead game's review first when more
// people liked and replied to it — otherwise merging silently demotes the
// review readers actually engaged with.
func preferEngagedRating(tx *gorm.DB, oldGID, newGID int) error {
	var loser []int
	if err := tx.Raw(`
		SELECT s.id FROM galgame_rating s JOIN galgame_rating d ON d.user_id = s.user_id
		WHERE s.galgame_id = ? AND d.galgame_id = ?
		  AND (d.like_count + d.comment_count) > (s.like_count + s.comment_count)`,
		newGID, oldGID).Scan(&loser).Error; err != nil {
		return err
	}
	if len(loser) == 0 {
		return nil
	}
	if err := archiveRatings(tx, "r.id IN ?", loser, oldGID, newGID); err != nil {
		return err
	}
	return tx.Exec("DELETE FROM galgame_rating WHERE id IN ?", loser).Error
}

// favorite_count is absent for a related reason since the folder cutover:
// galgame_collection_item is frozen rollback material, so recomputing from it
// would replace the live counter with a snapshot, and the memberships that
// actually moved were rehung upstream by the catalog's own merge. The counter
// is maintained by this site's writes and is not recomputed here.
//
// comment_count is absent on purpose: it mirrors the community thread anchored
// at site_game:<gid>, which lives in infra and does not move when the forum
// folds two local rows. The survivor keeps its own count and the merge sync logs
// the abandoned thread.
func recountAfterFold(tx *gorm.DB, gid int) error {
	return tx.Exec(`
		UPDATE galgame SET
			like_count        = (SELECT COUNT(*) FROM galgame_like WHERE galgame_id = galgame.id),
			resource_count    = (SELECT COUNT(*) FROM galgame_resource WHERE galgame_id = galgame.id),
			rating_count      = (SELECT COUNT(*) FROM galgame_rating WHERE galgame_id = galgame.id),
			contributor_count = (SELECT COUNT(*) FROM galgame_contributor WHERE galgame_id = galgame.id),
			view_7d           = (SELECT COALESCE(SUM(count), 0) FROM galgame_view_daily
			                      WHERE entity_id = galgame.id AND day >= CURRENT_DATE - INTERVAL '6 days'),
			view_30d          = (SELECT COALESCE(SUM(count), 0) FROM galgame_view_daily
			                      WHERE entity_id = galgame.id AND day >= CURRENT_DATE - INTERVAL '29 days')
		WHERE id = ?`, gid).Error
}

// RedirectTarget follows the ledger to the gid that still exists. A chain is
// already collapsed on write, so one hop is the answer.
func (r *GalgameMergeRepository) RedirectTarget(oldGID int) (int, bool) {
	var newGID []int
	r.db.Table("galgame_redirect").Where("old_gid = ?", oldGID).Pluck("new_gid", &newGID)
	if len(newGID) == 0 || newGID[0] <= 0 {
		return 0, false
	}
	return newGID[0], true
}
