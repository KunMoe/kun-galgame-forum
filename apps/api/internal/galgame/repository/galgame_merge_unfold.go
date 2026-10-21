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
	return foldChain(r.db, workID)
}

func foldChain(db *gorm.DB, workID int) ([]int, error) {
	chain := []int{workID}
	for len(chain) < 16 {
		next, err := activeFold(db, chain[len(chain)-1])
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
		chain, err := foldChain(tx, fold.NewWorkID)
		if err != nil {
			return err
		}
		u := &unfolder{tx: tx, old: fold.OldWorkID, survivor: fold.NewWorkID, chain: chain,
			columns: map[string][]string{},
			report:  &UnfoldReport{FoldID: fold.ID, OldWorkID: fold.OldWorkID, NewWorkID: fold.NewWorkID}}

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
			if err := u.applyIsolated(row); err != nil {
				return fmt.Errorf("unfold %s %s: %w", row.Table, row.Action, err)
			}
		}
		if err := u.restoreSurvivor(fold.OldRow, fold.SurvivorBefore, fold.SurvivorAfter); err != nil {
			return err
		}
		for _, id := range append([]int{fold.OldWorkID}, chain...) {
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
	old      int
	survivor int
	chain    []int
	columns  map[string][]string
	report   *UnfoldReport
}

func (u *unfolder) problem(table string, key int64, reason string, workID *int) {
	u.report.Problems = append(u.report.Problems, UnfoldProblem{Table: table, Key: key, Reason: reason, WorkID: workID})
}

// applyIsolated undoes one ledger row inside a savepoint. A row the database
// refuses (its quiz deleted since, a key taken) is recorded and skipped: a
// refusal that repeats on every run would otherwise hold the redirect feed
// cursor for good.
func (u *unfolder) applyIsolated(row foldRow) error {
	if err := u.tx.SavePoint("unfold_row").Error; err != nil {
		return err
	}
	err := u.apply(row)
	if err == nil {
		return nil
	}
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || (!strings.HasPrefix(pg.Code, "23") && !strings.HasPrefix(pg.Code, "22")) {
		return err
	}
	if rb := u.tx.RollbackTo("unfold_row").Error; rb != nil {
		return rb
	}
	reason := UnfoldConflict
	if pg.Code == "23503" {
		reason = UnfoldGone
	}
	u.problem(row.Table, firstKey(row.RowKey), reason, nil)
	return nil
}

func firstKey(raw []byte) int64 {
	var key map[string]json.Number
	if json.Unmarshal(raw, &key) != nil {
		return 0
	}
	for _, v := range key {
		n, _ := v.Int64()
		return n
	}
	return 0
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
		return u.reinsert(row)
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
	// A key that is also the peer names a row only together with its work: a
	// quiz linked to a later survivor in its own right carries the same key.
	from := u.chain
	if t.key == t.peer {
		from = []int{u.survivor}
	}
	guard := ""
	if t.peer != "" {
		guard = fmt.Sprintf("AND NOT EXISTS (SELECT 1 FROM %[1]s x WHERE x.work_id = @old AND x.%[2]s = %[1]s.%[2]s)", t.table, t.peer)
	}
	res := u.tx.Exec(fmt.Sprintf(`UPDATE %[1]s SET work_id = @old WHERE %[2]s = @key AND work_id IN @from %[3]s`,
		t.table, t.key, guard), map[string]any{"old": u.old, "key": key, "from": from})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		u.report.Restored += res.RowsAffected
		return nil
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

func (u *unfolder) reinsert(row foldRow) error {
	t, ok := mergeTableOf(row.Table)
	if !ok {
		return fmt.Errorf("table %s is not folded", row.Table)
	}
	key, err := ledgerKey(row.RowKey, t.key)
	if err != nil {
		return err
	}
	var before struct {
		UserID *int `json:"user_id"`
	}
	if err := json.Unmarshal(row.Before, &before); err != nil {
		return err
	}
	if before.UserID != nil {
		var purged bool
		if err := u.tx.Raw(`SELECT EXISTS (SELECT 1 FROM user_purge_archive WHERE target_user_id = ? AND restored_at IS NULL)`,
			*before.UserID).Scan(&purged).Error; err != nil {
			return err
		}
		if purged {
			u.problem(t.table, key, UnfoldPurgedUser, nil)
			return nil
		}
	}
	n, err := u.insertJSON(t.table, row.Before)
	if err != nil {
		return err
	}
	if n == 0 {
		u.problem(t.table, key, UnfoldConflict, &u.old)
		return nil
	}
	u.report.Restored += n
	return nil
}

func (u *unfolder) unsumContributor(row foldRow) error {
	key, err := ledgerKey(row.RowKey, "id")
	if err != nil {
		return err
	}
	var parts struct {
		Survivor json.RawMessage `json:"survivor"`
		Absorbed json.RawMessage `json:"absorbed"`
	}
	if err := json.Unmarshal(row.Before, &parts); err != nil {
		return err
	}
	res := u.tx.Exec(`
		UPDATE galgame_contributor t SET
			revision_count = GREATEST(t.revision_count - a.revision_count, 0),
			first_at = CASE WHEN t.first_at = LEAST(b.first_at, a.first_at) THEN b.first_at ELSE t.first_at END,
			last_at  = CASE WHEN t.last_at = GREATEST(b.last_at, a.last_at) THEN b.last_at ELSE t.last_at END
		FROM jsonb_populate_record(NULL::galgame_contributor, ?::jsonb) b,
		     jsonb_populate_record(NULL::galgame_contributor, ?::jsonb) a
		WHERE t.id = ?`, string(parts.Survivor), string(parts.Absorbed), key)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		u.problem("galgame_contributor", key, UnfoldGone, nil)
	}
	return nil
}

func (u *unfolder) unsumViews(row foldRow) error {
	var bucket struct {
		Day   string `json:"day"`
		Count int    `json:"count"`
	}
	if err := json.Unmarshal(row.Before, &bucket); err != nil {
		return err
	}
	for _, workID := range u.chain {
		res := u.tx.Exec(`UPDATE galgame_view_daily SET count = GREATEST(count - ?, 0) WHERE entity_id = ? AND day = ?`,
			bucket.Count, workID, bucket.Day)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected > 0 {
			if err := u.tx.Exec(`DELETE FROM galgame_view_daily WHERE entity_id = ? AND day = ? AND count = 0`,
				workID, bucket.Day).Error; err != nil {
				return err
			}
			break
		}
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
	res := u.tx.Exec(`UPDATE feed_activity SET work_id = ?, link = ? WHERE id = ? AND work_id IN ?`,
		before.WorkID, before.Link, key, u.chain)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		u.problem("feed_activity", key, UnfoldGone, nil)
		return nil
	}
	u.report.Restored += res.RowsAffected
	return nil
}

// restoreOldRow puts the old work's galgame row back. A row can already be
// there: a like or a wall comment on the revived work creates a stub before
// the next sync; the snapshot then folds into it the way FoldTx folds rows.
func (u *unfolder) restoreOldRow(oldRow []byte) error {
	n, err := u.insertJSON("galgame", oldRow)
	if err != nil || n > 0 {
		return err
	}
	return u.tx.Exec(`
		UPDATE galgame g SET
			view = g.view + COALESCE(o.view, 0),
			published = g.published OR COALESCE(o.published, false),
			resource_publish_banned = g.resource_publish_banned OR COALESCE(o.resource_publish_banned, false),
			creator_user_id = COALESCE(g.creator_user_id, o.creator_user_id),
			created = LEAST(g.created, o.created),
			resource_update_time = GREATEST(g.resource_update_time, o.resource_update_time)
		FROM jsonb_populate_record(NULL::galgame, ?::jsonb) o WHERE g.id = ?`, string(oldRow), u.old).Error
}

// restoreSurvivor takes back what the fold added to the survivor's own row:
// the old work's views, and every flag the fold changed that nothing has
// changed since. A survivor row the fold created goes once nothing else
// references it.
func (u *unfolder) restoreSurvivor(oldRow, before, after []byte) error {
	var exists bool
	if err := u.tx.Raw(`SELECT EXISTS (SELECT 1 FROM galgame WHERE id = ?)`, u.survivor).Scan(&exists).Error; err != nil {
		return err
	}
	if !exists {
		if oldRow != nil {
			u.problem("galgame", int64(u.survivor), UnfoldSurvivorFolded, &u.chain[len(u.chain)-1])
		}
		return nil
	}
	created := before == nil && oldRow != nil
	if created {
		referenced, err := (&GalgameMergeRepository{db: u.tx}).referencedBeyondRow(u.survivor)
		if err != nil {
			return err
		}
		if !referenced {
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
