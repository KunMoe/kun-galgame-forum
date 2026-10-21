package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type MergeFold struct {
	ID        int64
	OldWorkID int
	NewWorkID int
}

const (
	UnfoldGone           = "gone"
	UnfoldElsewhere      = "elsewhere"
	UnfoldConflict       = "conflict"
	UnfoldPurgedUser     = "purged_user"
	UnfoldSurvivorFolded = "survivor_folded"
)

type UnfoldProblem struct {
	Table  string `json:"table"`
	Key    int64  `json:"key"`
	Reason string `json:"reason"`
	WorkID *int   `json:"work_id,omitempty"`
}

type UnfoldReport struct {
	FoldID    int64
	OldWorkID int
	NewWorkID int
	Restored  int64
	Problems  []UnfoldProblem
}

type foldRow struct {
	Table  string `gorm:"column:table_name"`
	Action string
	RowKey []byte `gorm:"column:row_key"`
	Before []byte
}

func (r *GalgameMergeRepository) ActiveFold(oldWorkID int) (*MergeFold, error) {
	return activeFold(r.db, oldWorkID)
}

func activeFold(db *gorm.DB, oldWorkID int) (*MergeFold, error) {
	var f MergeFold
	res := db.Raw(`SELECT id, old_work_id, new_work_id FROM galgame_merge_fold
		WHERE old_work_id = ? AND undone_at IS NULL ORDER BY id DESC LIMIT 1`, oldWorkID).Scan(&f)
	if res.Error != nil || res.RowsAffected == 0 {
		return nil, res.Error
	}
	return &f, nil
}

// FoldChain is workID followed by every work it has since been folded into:
// rows a fold moved onto workID travel on with each later fold.
func (r *GalgameMergeRepository) FoldChain(workID int) ([]int, error) {
	chain := []int{workID}
	for len(chain) < 16 {
		next, err := activeFold(r.db, chain[len(chain)-1])
		if err != nil {
			return nil, err
		}
		if next == nil || slices.Contains(chain, next.NewWorkID) {
			break
		}
		chain = append(chain, next.NewWorkID)
	}
	return chain, nil
}

// Unfold reverses the newest active fold of oldWorkID and returns nil when
// there is none. A work can have been folded more than once, so callers repeat
// it until it returns nil. Rows the survivor gained after the fold stay there.
//
// Each row is followed through the folds of the survivor that came after this
// one, by their ledgers: an undo that runs before theirs takes its rows back
// from wherever those folds put them and corrects their ledgers, so their own
// undo does not move the same rows a second time.
func (r *GalgameMergeRepository) Unfold(oldWorkID int) (*UnfoldReport, error) {
	var report *UnfoldReport
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var fold struct {
			ID             int64
			OldWorkID      int
			NewWorkID      int
			OldRow         []byte
			SurvivorBefore []byte
			SurvivorAfter  []byte
		}
		res := tx.Raw(`SELECT id, old_work_id, new_work_id, old_row, survivor_before, survivor_after
			FROM galgame_merge_fold WHERE old_work_id = ? AND undone_at IS NULL
			ORDER BY id DESC LIMIT 1 FOR UPDATE`, oldWorkID).Scan(&fold)
		if res.Error != nil || res.RowsAffected == 0 {
			return res.Error
		}
		u := &unfolder{tx: tx, foldID: fold.ID, old: fold.OldWorkID, survivor: fold.NewWorkID,
			columns: map[string][]string{}, touched: map[int]bool{fold.OldWorkID: true, fold.NewWorkID: true},
			report: &UnfoldReport{FoldID: fold.ID, OldWorkID: fold.OldWorkID, NewWorkID: fold.NewWorkID}}

		if fold.OldRow != nil {
			if err := u.restoreOldRow(fold.OldRow); err != nil {
				return err
			}
		}

		var rows []foldRow
		if err := tx.Raw(`SELECT table_name, action, row_key, before FROM galgame_merge_fold_row
			WHERE fold_id = ? ORDER BY id`, fold.ID).Scan(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			if err := u.apply(row); err != nil {
				return fmt.Errorf("unfold %s %s: %w", row.Table, row.Action, err)
			}
		}
		if err := u.restoreSurvivor(fold.OldRow, fold.SurvivorBefore, fold.SurvivorAfter); err != nil {
			return err
		}
		for id := range u.touched {
			if err := recountAfterFold(tx, id); err != nil {
				return err
			}
		}
		problems, err := json.Marshal(u.report.Problems)
		if err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE galgame_merge_fold SET undone_at = ?, undo_problems = ?::jsonb WHERE id = ?`,
			time.Now(), string(problems), fold.ID).Error; err != nil {
			return err
		}
		report = u.report
		return nil
	})
	return report, err
}

type unfolder struct {
	tx       *gorm.DB
	foldID   int64
	old      int
	survivor int
	columns  map[string][]string
	touched  map[int]bool
	report   *UnfoldReport
}

func (u *unfolder) problem(table string, key int64, reason string, workID *int) {
	u.report.Problems = append(u.report.Problems, UnfoldProblem{Table: table, Key: key, Reason: reason, WorkID: workID})
}

// laterEntry is what a later fold's ledger says about the same row.
type laterEntry struct {
	ID      int64
	FoldID  int64
	NewWork int
	Action  string
	Before  []byte
}

func (u *unfolder) later(table string, rowKey []byte, at int, after int64) (*laterEntry, error) {
	var e laterEntry
	res := u.tx.Raw(`SELECT r.id, g.id AS fold_id, g.new_work_id AS new_work, r.action, r.before
		FROM galgame_merge_fold_row r JOIN galgame_merge_fold g ON g.id = r.fold_id
		WHERE g.undone_at IS NULL AND g.old_work_id = ? AND g.id > ? AND r.table_name = ? AND r.row_key = ?::jsonb
		ORDER BY g.id LIMIT 1`, at, after, table, string(rowKey)).Scan(&e)
	if res.Error != nil || res.RowsAffected == 0 {
		return nil, res.Error
	}
	return &e, nil
}

// trace follows a row this fold left on the survivor through the later folds
// that moved or repointed it. It ends where the row is now, or at the later
// ledger entry that dropped it.
func (u *unfolder) trace(table string, rowKey []byte) (at int, carried []*laterEntry, dropped *laterEntry, err error) {
	at, after := u.survivor, u.foldID
	for range 16 {
		e, err := u.later(table, rowKey, at, after)
		if err != nil || e == nil {
			return at, carried, nil, err
		}
		switch e.Action {
		case "moved", "repointed":
			carried = append(carried, e)
			at, after = e.NewWork, e.FoldID
			u.touched[at] = true
		case "dropped":
			return at, carried, e, nil
		default:
			return at, carried, nil, nil
		}
	}
	return at, carried, nil, nil
}

func (u *unfolder) chainFrom(workID int) ([]int, error) {
	return (&GalgameMergeRepository{db: u.tx}).FoldChain(workID)
}

func (u *unfolder) void(entries ...*laterEntry) error {
	ids := make([]int64, 0, len(entries))
	for _, e := range entries {
		if e != nil {
			ids = append(ids, e.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return u.tx.Exec(`DELETE FROM galgame_merge_fold_row WHERE id IN ?`, ids).Error
}

func mergeTableOf(name string) (mergeTable, bool) {
	for _, t := range slices.Concat(mergeMovableTables, mergeUniqueTables) {
		if t.table == name {
			return t, true
		}
	}
	return mergeTable{}, false
}

func ledgerKey(raw []byte, column string) (int64, error) {
	var key map[string]json.Number
	if err := json.Unmarshal(raw, &key); err != nil {
		return 0, err
	}
	return key[column].Int64()
}

func (u *unfolder) apply(row foldRow) error {
	switch {
	case row.Action == "moved":
		return u.moveBack(row)
	case row.Action == "dropped":
		t, ok := mergeTableOf(row.Table)
		if !ok {
			return fmt.Errorf("table %s is not folded", row.Table)
		}
		key, err := ledgerKey(row.RowKey, t.key)
		if err != nil {
			return err
		}
		return u.restoreSnapshot(t, key, row.Before, u.survivor)
	case row.Table == "galgame_contributor" && row.Action == "summed":
		return u.unsumContributor(row)
	case row.Table == "galgame_view_daily" && row.Action == "summed":
		return u.unsumViews(row)
	case row.Table == "feed_activity" && row.Action == "repointed":
		return u.repointFeed(row)
	}
	return fmt.Errorf("unknown ledger row")
}

func (u *unfolder) moveBack(row foldRow) error {
	t, ok := mergeTableOf(row.Table)
	if !ok {
		return fmt.Errorf("table %s is not folded", row.Table)
	}
	key, err := ledgerKey(row.RowKey, t.key)
	if err != nil {
		return err
	}
	at, carried, dropped, err := u.trace(t.table, row.RowKey)
	if err != nil {
		return err
	}
	if dropped != nil {
		if err := u.restoreSnapshot(t, key, dropped.Before, dropped.NewWork); err != nil {
			return err
		}
		if t.table == "galgame_contributor" {
			if err := u.unsumAbsorbedBy(dropped.FoldID, key); err != nil {
				return err
			}
		}
		return u.void(append(carried, dropped)...)
	}
	guard := ""
	if t.peer != "" {
		guard = fmt.Sprintf("AND NOT EXISTS (SELECT 1 FROM %[1]s x WHERE x.work_id = @old AND x.%[2]s = %[1]s.%[2]s)", t.table, t.peer)
	}
	moved, refused, err := u.isolated(func() (int64, error) {
		res := u.tx.Exec(fmt.Sprintf(`UPDATE %[1]s SET work_id = @old WHERE %[2]s = @key AND work_id = @at %[3]s`,
			t.table, t.key, guard), map[string]any{"old": u.old, "key": key, "at": at})
		return res.RowsAffected, res.Error
	})
	if err != nil {
		return err
	}
	if refused != "" {
		u.problem(t.table, key, refused, nil)
		return nil
	}
	if moved > 0 {
		u.report.Restored += moved
		return u.void(carried...)
	}
	var now []int
	if err := u.tx.Raw(fmt.Sprintf(`SELECT work_id FROM %s WHERE %s = ?`, t.table, t.key), key).Scan(&now).Error; err != nil {
		return err
	}
	switch {
	case len(now) == 0:
		u.problem(t.table, key, UnfoldGone, nil)
	case slices.Contains(now, u.old):
		u.problem(t.table, key, UnfoldConflict, &u.old)
	default:
		u.problem(t.table, key, UnfoldElsewhere, &now[0])
	}
	return nil
}

// restoreSnapshot puts a row a fold deleted back on the old work. It was
// deleted because into, the fold's survivor, held the same user's or quiz's
// row; catalog's own undo splits a folded row back out only while the row it
// was folded into still exists, so this one does the same: someone who undid
// the survivor's copy since keeps it undone. A purged author's row stays out.
func (u *unfolder) restoreSnapshot(t mergeTable, key int64, before []byte, into int) error {
	table := t.table
	var row map[string]any
	if err := json.Unmarshal(before, &row); err != nil {
		return err
	}
	if uid, ok := row["user_id"].(float64); ok {
		var purged bool
		if err := u.tx.Raw(`SELECT EXISTS (SELECT 1 FROM user_purge_archive WHERE target_user_id = ? AND restored_at IS NULL)`,
			int(uid)).Scan(&purged).Error; err != nil {
			return err
		}
		if purged {
			u.problem(table, key, UnfoldPurgedUser, nil)
			return nil
		}
	}
	if peer, ok := row[t.peer]; ok && t.peer != "" {
		chain, err := u.chainFrom(into)
		if err != nil {
			return err
		}
		var alive bool
		if err := u.tx.Raw(fmt.Sprintf(`SELECT EXISTS (SELECT 1 FROM %s WHERE work_id IN ? AND %s = ?)`, table, t.peer),
			chain, peer).Scan(&alive).Error; err != nil {
			return err
		}
		if !alive {
			u.problem(table, key, UnfoldGone, nil)
			return nil
		}
	}
	row["work_id"] = u.old
	raw, err := json.Marshal(row)
	if err != nil {
		return err
	}
	n, refused, err := u.isolated(func() (int64, error) { return u.insertJSON(table, raw) })
	if err != nil {
		return err
	}
	switch {
	case refused != "":
		u.problem(table, key, refused, nil)
	case n == 0:
		u.problem(table, key, UnfoldConflict, &u.old)
	default:
		u.report.Restored += n
	}
	return nil
}

// isolated runs a write that the database may refuse (its quiz deleted since,
// a key taken) inside a savepoint and reports the refusal instead of failing:
// a refusal that repeats on every run would hold the redirect feed cursor for
// good.
func (u *unfolder) isolated(write func() (int64, error)) (int64, string, error) {
	if err := u.tx.SavePoint("unfold_row").Error; err != nil {
		return 0, "", err
	}
	n, err := write()
	if err == nil {
		return n, "", nil
	}
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || (!strings.HasPrefix(pg.Code, "23") && !strings.HasPrefix(pg.Code, "22")) {
		return 0, "", err
	}
	if rb := u.tx.RollbackTo("unfold_row").Error; rb != nil {
		return 0, "", rb
	}
	if pg.Code == "23503" {
		return 0, UnfoldGone, nil
	}
	return 0, UnfoldConflict, nil
}

type contributorParts struct {
	Survivor json.RawMessage `json:"survivor"`
	Absorbed json.RawMessage `json:"absorbed"`
}

const unsumContributorSQL = `
	UPDATE galgame_contributor t SET
		revision_count = GREATEST(t.revision_count - a.revision_count, 0),
		first_at = CASE WHEN t.first_at = LEAST(b.first_at, a.first_at) THEN b.first_at ELSE t.first_at END,
		last_at  = CASE WHEN t.last_at = GREATEST(b.last_at, a.last_at) THEN b.last_at ELSE t.last_at END
	FROM jsonb_populate_record(NULL::galgame_contributor, ?::jsonb) b,
	     jsonb_populate_record(NULL::galgame_contributor, ?::jsonb) a
	WHERE t.id = ?`

// unsumContributor takes this fold's absorbed revisions back off the survivor's
// contributor row. When a later fold dropped that row into another one, the
// count comes off that one instead, and the later fold's snapshots stop
// carrying it.
func (u *unfolder) unsumContributor(row foldRow) error {
	key, err := ledgerKey(row.RowKey, "id")
	if err != nil {
		return err
	}
	var parts contributorParts
	if err := json.Unmarshal(row.Before, &parts); err != nil {
		return err
	}
	_, _, dropped, err := u.trace("galgame_contributor", row.RowKey)
	if err != nil {
		return err
	}
	target := key
	if dropped != nil {
		var into struct {
			ID     int64
			RowKey []byte
		}
		res := u.tx.Raw(`SELECT id, row_key FROM galgame_merge_fold_row WHERE fold_id = ? AND action = 'summed'
			AND table_name = 'galgame_contributor' AND before->'absorbed'->>'id' = ?`, dropped.FoldID, strconv.FormatInt(key, 10)).Scan(&into)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			u.problem("galgame_contributor", key, UnfoldGone, nil)
			return nil
		}
		if target, err = ledgerKey(into.RowKey, "id"); err != nil {
			return err
		}
		var absorbed struct {
			RevisionCount int `json:"revision_count"`
		}
		if err := json.Unmarshal(parts.Absorbed, &absorbed); err != nil {
			return err
		}
		if err := u.tx.Exec(`UPDATE galgame_merge_fold_row SET before = jsonb_set(before, '{revision_count}',
			to_jsonb(GREATEST((before->>'revision_count')::int - ?, 0))) WHERE id = ?`, absorbed.RevisionCount, dropped.ID).Error; err != nil {
			return err
		}
		if err := u.tx.Exec(`UPDATE galgame_merge_fold_row SET before = jsonb_set(before, '{absorbed,revision_count}',
			to_jsonb(GREATEST((before->'absorbed'->>'revision_count')::int - ?, 0))) WHERE id = ?`, absorbed.RevisionCount, into.ID).Error; err != nil {
			return err
		}
	}
	res := u.tx.Exec(unsumContributorSQL, string(parts.Survivor), string(parts.Absorbed), target)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		u.problem("galgame_contributor", key, UnfoldGone, nil)
	}
	return nil
}

// unsumAbsorbedBy reverses what a later fold added into one of its survivor's
// contributor rows from a row of this fold's, and drops that entry: the row
// itself is back on the old work.
func (u *unfolder) unsumAbsorbedBy(foldID, key int64) error {
	var entry struct {
		ID     int64
		RowKey []byte
		Before []byte
	}
	res := u.tx.Raw(`SELECT id, row_key, before FROM galgame_merge_fold_row WHERE fold_id = ? AND action = 'summed'
		AND table_name = 'galgame_contributor' AND before->'absorbed'->>'id' = ?`, foldID, strconv.FormatInt(key, 10)).Scan(&entry)
	if res.Error != nil || res.RowsAffected == 0 {
		return res.Error
	}
	target, err := ledgerKey(entry.RowKey, "id")
	if err != nil {
		return err
	}
	var parts contributorParts
	if err := json.Unmarshal(entry.Before, &parts); err != nil {
		return err
	}
	if err := u.tx.Exec(unsumContributorSQL, string(parts.Survivor), string(parts.Absorbed), target).Error; err != nil {
		return err
	}
	return u.tx.Exec(`DELETE FROM galgame_merge_fold_row WHERE id = ?`, entry.ID).Error
}

// unsumViews takes the old work's daily views back off wherever later folds
// summed them, and lowers those folds' snapshots by the same amount so their
// own undo moves only what was theirs.
func (u *unfolder) unsumViews(row foldRow) error {
	var bucket struct {
		Day   string `json:"day"`
		Count int    `json:"count"`
	}
	if err := json.Unmarshal(row.Before, &bucket); err != nil {
		return err
	}
	at, after := u.survivor, u.foldID
	for range 16 {
		e, err := u.later("galgame_view_daily", row.RowKey, at, after)
		if err != nil {
			return err
		}
		if e == nil || e.Action != "summed" {
			break
		}
		if err := u.tx.Exec(`UPDATE galgame_merge_fold_row SET before = jsonb_set(before, '{count}',
			to_jsonb(GREATEST((before->>'count')::int - ?, 0))) WHERE id = ?`, bucket.Count, e.ID).Error; err != nil {
			return err
		}
		at, after = e.NewWork, e.FoldID
		u.touched[at] = true
	}
	if err := u.tx.Exec(`UPDATE galgame_view_daily SET count = GREATEST(count - ?, 0) WHERE entity_id = ? AND day = ?`,
		bucket.Count, at, bucket.Day).Error; err != nil {
		return err
	}
	if err := u.tx.Exec(`DELETE FROM galgame_view_daily WHERE entity_id = ? AND day = ? AND count = 0`, at, bucket.Day).Error; err != nil {
		return err
	}
	return u.tx.Exec(`
		INSERT INTO galgame_view_daily (entity_id, day, count) VALUES (?, ?, ?)
		ON CONFLICT (entity_id, day) DO UPDATE SET count = galgame_view_daily.count + EXCLUDED.count`,
		u.old, bucket.Day, bucket.Count).Error
}

func (u *unfolder) repointFeed(row foldRow) error {
	key, err := ledgerKey(row.RowKey, "id")
	if err != nil {
		return err
	}
	var before struct {
		WorkID int    `json:"work_id"`
		Link   string `json:"link"`
	}
	if err := json.Unmarshal(row.Before, &before); err != nil {
		return err
	}
	at, carried, _, err := u.trace("feed_activity", row.RowKey)
	if err != nil {
		return err
	}
	res := u.tx.Exec(`UPDATE feed_activity SET work_id = ?, link = ? WHERE id = ? AND work_id = ?`,
		before.WorkID, before.Link, key, at)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		u.problem("feed_activity", key, UnfoldGone, nil)
		return nil
	}
	u.report.Restored += res.RowsAffected
	return u.void(carried...)
}

// restoreOldRow puts the old work's galgame row back. A row can already be
// there: a like or a wall comment on the revived work creates a stub before
// the next sync; the snapshot then folds into it the way FoldTx folds rows,
// and the counters nothing recounts are added.
func (u *unfolder) restoreOldRow(oldRow []byte) error {
	n, err := u.insertJSON("galgame", oldRow)
	if err != nil || n > 0 {
		return err
	}
	return u.tx.Exec(`
		UPDATE galgame g SET
			view = g.view + COALESCE(o.view, 0),
			favorite_count = g.favorite_count + COALESCE(o.favorite_count, 0),
			comment_count = g.comment_count + COALESCE(o.comment_count, 0),
			published = g.published OR COALESCE(o.published, false),
			resource_publish_banned = g.resource_publish_banned OR COALESCE(o.resource_publish_banned, false),
			creator_user_id = COALESCE(g.creator_user_id, o.creator_user_id),
			created = LEAST(g.created, o.created),
			resource_update_time = GREATEST(g.resource_update_time, o.resource_update_time),
			release_date = COALESCE(g.release_date, o.release_date),
			release_date_synced_at = COALESCE(g.release_date_synced_at, o.release_date_synced_at),
			content_limit = COALESCE(g.content_limit, o.content_limit),
			original_language = COALESCE(g.original_language, o.original_language),
			catalog_checked_at = COALESCE(g.catalog_checked_at, o.catalog_checked_at)
		FROM jsonb_populate_record(NULL::galgame, ?::jsonb) o WHERE g.id = ?`, string(oldRow), u.old).Error
}

// The columns of the survivor's row a fold writes, compared to tell whether
// anyone has touched the row since.
const survivorColumns = `published, resource_publish_banned, creator_user_id, view, comment_count, favorite_count, created, resource_update_time`

// restoreSurvivor takes back what the fold added to the survivor's own row:
// the old work's views, and every flag the fold changed that nothing has
// changed since. A survivor row the fold created goes, but only while nothing
// references it and nobody has written to it.
func (u *unfolder) restoreSurvivor(oldRow, before, after []byte) error {
	var exists bool
	if err := u.tx.Raw(`SELECT EXISTS (SELECT 1 FROM galgame WHERE id = ?)`, u.survivor).Scan(&exists).Error; err != nil {
		return err
	}
	if !exists {
		return u.restoreFoldedSurvivor(oldRow, before, after)
	}
	if before == nil && oldRow != nil {
		var untouched bool
		if err := u.tx.Raw(`SELECT (`+prefixed("g", survivorColumns)+`) IS NOT DISTINCT FROM (`+prefixed("a", survivorColumns)+`)
			FROM galgame g, jsonb_populate_record(NULL::galgame, ?::jsonb) a WHERE g.id = ?`,
			sqlJSON(after), u.survivor).Scan(&untouched).Error; err != nil {
			return err
		}
		referenced, err := (&GalgameMergeRepository{db: u.tx}).referencedBeyondRow(u.survivor)
		if err != nil {
			return err
		}
		if untouched && !referenced {
			delete(u.touched, u.survivor)
			return u.tx.Exec(`DELETE FROM galgame WHERE id = ?`, u.survivor).Error
		}
		before = []byte(`{"published": false, "resource_publish_banned": false, "creator_user_id": null}`)
	}
	return u.tx.Exec(`
		UPDATE galgame g SET
			view = GREATEST(g.view - COALESCE(o.view, 0), 0),
			published = CASE WHEN g.published IS NOT DISTINCT FROM a.published
				THEN COALESCE(b.published, g.published) OR EXISTS (SELECT 1 FROM galgame_resource r WHERE r.work_id = g.id)
				ELSE g.published END,
			resource_publish_banned = CASE WHEN g.resource_publish_banned IS NOT DISTINCT FROM a.resource_publish_banned
				THEN COALESCE(b.resource_publish_banned, g.resource_publish_banned) ELSE g.resource_publish_banned END,
			creator_user_id = CASE WHEN g.creator_user_id IS NOT DISTINCT FROM a.creator_user_id
				THEN b.creator_user_id ELSE g.creator_user_id END,
			created = CASE WHEN g.created IS NOT DISTINCT FROM a.created
				THEN COALESCE(b.created, g.created) ELSE g.created END,
			resource_update_time = CASE WHEN g.resource_update_time IS NOT DISTINCT FROM a.resource_update_time
				THEN COALESCE(b.resource_update_time, g.resource_update_time) ELSE g.resource_update_time END
		FROM jsonb_populate_record(NULL::galgame, ?::jsonb) a,
		     jsonb_populate_record(NULL::galgame, ?::jsonb) b,
		     jsonb_populate_record(NULL::galgame, ?::jsonb) o
		WHERE g.id = ?`, sqlJSON(after), sqlJSON(before), sqlJSON(oldRow), u.survivor).Error
}

// restoreFoldedSurvivor handles a survivor a later fold has merged away: the
// old work's views come off the row they ended up in, and the later fold's
// snapshot of the survivor goes back to what it was before this fold where
// nothing changed it in between. Flags the later fold ORed into its own
// survivor cannot be taken apart, so a flag this fold had changed is reported.
func (u *unfolder) restoreFoldedSurvivor(oldRow, before, after []byte) error {
	if oldRow == nil {
		return nil
	}
	at, after64 := u.survivor, u.foldID
	var first *struct {
		ID     int64
		OldRow []byte
	}
	for range 16 {
		var next struct {
			ID        int64
			NewWorkID int
			OldRow    []byte
		}
		res := u.tx.Raw(`SELECT id, new_work_id, old_row FROM galgame_merge_fold
			WHERE old_work_id = ? AND id > ? AND undone_at IS NULL ORDER BY id LIMIT 1`, at, after64).Scan(&next)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			break
		}
		if first == nil {
			first = &struct {
				ID     int64
				OldRow []byte
			}{next.ID, next.OldRow}
		}
		if err := u.tx.Exec(`UPDATE galgame_merge_fold SET old_row = jsonb_set(old_row, '{view}',
			to_jsonb(GREATEST((old_row->>'view')::int - COALESCE((?::jsonb->>'view')::int, 0), 0)))
			WHERE id = ? AND old_row IS NOT NULL`, string(oldRow), next.ID).Error; err != nil {
			return err
		}
		at, after64 = next.NewWorkID, next.ID
	}
	u.touched[at] = true
	if err := u.tx.Exec(`UPDATE galgame SET view = GREATEST(view - COALESCE((?::jsonb->>'view')::int, 0), 0) WHERE id = ?`,
		string(oldRow), at).Error; err != nil {
		return err
	}
	if first == nil || after == nil {
		return nil
	}
	flags := []string{"published", "resource_publish_banned", "creator_user_id"}
	var b, a map[string]any
	_ = json.Unmarshal(after, &a)
	if before != nil {
		_ = json.Unmarshal(before, &b)
	}
	changed := false
	for _, f := range flags {
		was := any(nil)
		if b != nil {
			was = b[f]
		} else if f != "creator_user_id" {
			was = false
		}
		if fmt.Sprint(was) != fmt.Sprint(a[f]) {
			changed = true
			if err := u.tx.Exec(`UPDATE galgame_merge_fold SET old_row = jsonb_set(old_row, ?::text[], ?::jsonb)
				WHERE id = ? AND old_row->? = ?::jsonb`, "{"+f+"}", jsonValue(was), first.ID, f, jsonValue(a[f])).Error; err != nil {
				return err
			}
		}
	}
	if changed {
		u.problem("galgame", int64(u.survivor), UnfoldSurvivorFolded, &at)
	}
	return nil
}

func jsonValue(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

func prefixed(alias, columns string) string {
	parts := strings.Split(columns, ", ")
	for i, c := range parts {
		parts[i] = alias + "." + c
	}
	return strings.Join(parts, ", ")
}

// sqlJSON passes a missing snapshot as SQL NULL: jsonb_populate_record refuses
// the JSON scalar null ("cannot call populate_composite on a scalar").
func sqlJSON(raw []byte) any {
	if raw == nil {
		return nil
	}
	return string(raw)
}

// insertJSON inserts a row archived with to_jsonb, naming only the columns the
// archive holds: a column added since then takes its default instead of NULL.
func (u *unfolder) insertJSON(table string, raw []byte) (int64, error) {
	var row map[string]json.RawMessage
	if err := json.Unmarshal(raw, &row); err != nil {
		return 0, err
	}
	columns, ok := u.columns[table]
	if !ok {
		if err := u.tx.Raw(`SELECT column_name FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = ? AND is_generated = 'NEVER'`, table).Scan(&columns).Error; err != nil {
			return 0, err
		}
		u.columns[table] = columns
	}
	keep := make([]string, 0, len(columns))
	for _, c := range columns {
		if _, ok := row[c]; ok {
			keep = append(keep, strconv.Quote(c))
		}
	}
	if len(keep) == 0 {
		return 0, errors.New("archived row names no column of " + table)
	}
	list := strings.Join(keep, ", ")
	res := u.tx.Exec(fmt.Sprintf(`INSERT INTO %[1]s (%[2]s) SELECT %[2]s FROM jsonb_populate_record(NULL::%[1]s, ?::jsonb)
		ON CONFLICT DO NOTHING`, table, list), string(raw))
	return res.RowsAffected, res.Error
}

type FoldSummary struct {
	ID        int64
	OldWorkID int
	NewWorkID int
	FoldedAt  time.Time
	UndoneAt  *time.Time
	Moved     int
	Dropped   int
	Summed    int
	Repointed int
	Problems  []UnfoldProblem
}

// ListFolds pages the fold ledger newest first.
func (r *GalgameMergeRepository) ListFolds(offset, limit int) ([]FoldSummary, int, error) {
	var total int64
	if err := r.db.Raw(`SELECT count(*) FROM galgame_merge_fold`).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []struct {
		ID        int64
		OldWorkID int
		NewWorkID int
		FoldedAt  time.Time
		UndoneAt  *time.Time
		Moved     int
		Dropped   int
		Summed    int
		Repointed int
		Problems  string
	}
	if err := r.db.Raw(`
		SELECT f.id, f.old_work_id, f.new_work_id, f.folded_at, f.undone_at,
			COALESCE(f.undo_problems, '[]'::jsonb)::text AS problems,
			count(r.id) FILTER (WHERE r.action = 'moved') AS moved,
			count(r.id) FILTER (WHERE r.action = 'dropped') AS dropped,
			count(r.id) FILTER (WHERE r.action = 'summed') AS summed,
			count(r.id) FILTER (WHERE r.action = 'repointed') AS repointed
		FROM galgame_merge_fold f LEFT JOIN galgame_merge_fold_row r ON r.fold_id = f.id
		GROUP BY f.id ORDER BY f.id DESC LIMIT ? OFFSET ?`, limit, offset).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]FoldSummary, 0, len(rows))
	for _, row := range rows {
		s := FoldSummary{ID: row.ID, OldWorkID: row.OldWorkID, NewWorkID: row.NewWorkID, FoldedAt: row.FoldedAt,
			UndoneAt: row.UndoneAt, Moved: row.Moved, Dropped: row.Dropped, Summed: row.Summed, Repointed: row.Repointed}
		if err := json.Unmarshal([]byte(row.Problems), &s.Problems); err != nil {
			return nil, 0, err
		}
		out = append(out, s)
	}
	return out, int(total), nil
}
