package relocation

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

const awayPredicate = `g.original_language IS NOT NULL
	AND lower(g.original_language) <> 'ja' AND lower(g.original_language) NOT LIKE 'zh%'`

// update_time is written once, by the column DEFAULT at insert; an edit only
// moves edited. LetMoe orders by update_time, so the later of the two is sent.
const payloadExpr = `jsonb_build_object(
	'forum_id', r.id, 'work_id', r.work_id, 'user_id', r.user_id,
	'type', r.type, 'title', r.title, 'version_label', r.version_label,
	'note', r.note, 'size', r.size,
	'languages', r.languages, 'platforms', r.platforms, 'runtimes', r.runtimes,
	'code', r.code, 'password', r.password, 'status', r.status,
	'download', r.download, 'view', r.view,
	'created', r.created, 'update_time', GREATEST(r.update_time, r.edited), 'edited', r.edited,
	'links', COALESCE((SELECT jsonb_agg(jsonb_build_object('url', l.url, 'created', l.created) ORDER BY l.id)
		FROM galgame_resource_link l WHERE l.galgame_resource_id = r.id), '[]'::jsonb),
	'likes', COALESCE((SELECT jsonb_agg(jsonb_build_object('user_id', k.user_id, 'created', k.created) ORDER BY k.id)
		FROM galgame_resource_like k WHERE k.galgame_resource_id = r.id), '[]'::jsonb))`

// Counters and likes move on their own between the snapshot and the retire, so
// they are left out of the "did the uploader change it" comparison.
const stableKeys = ` - 'download' - 'view' - 'likes'`

var ErrChanged = errors.New("relocation: the resource changed or went away after it was exported")

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

type Census struct {
	Works     int64
	Resources int64
	Uploaders int64
	Snapshots int64
	Pushed    int64
	Retired   int64
}

func (s *Store) Census(excludeWorks []int) (Census, error) {
	var c Census
	err := s.db.Raw(`SELECT COUNT(DISTINCT r.work_id) AS works, COUNT(*) AS resources, COUNT(DISTINCT r.user_id) AS uploaders
		FROM galgame_resource r JOIN galgame g ON g.id = r.work_id
		WHERE `+awayPredicate+` AND r.work_id <> ALL(?::int[])`, intArray(excludeWorks)).Scan(&c).Error
	if err != nil {
		return c, err
	}
	var ledger struct{ Snapshots, Pushed, Retired int64 }
	err = s.db.Raw(`SELECT COUNT(*) AS snapshots, COUNT(pushed_at) AS pushed, COUNT(retired_at) AS retired
		FROM galgame_resource_relocation`).Scan(&ledger).Error
	c.Snapshots, c.Pushed, c.Retired = ledger.Snapshots, ledger.Pushed, ledger.Retired
	return c, err
}

func (s *Store) UnsyncedWorks() ([]int, error) {
	var ids []int
	err := s.db.Raw(`SELECT DISTINCT r.work_id FROM galgame_resource r JOIN galgame g ON g.id = r.work_id
		WHERE g.original_language IS NULL ORDER BY 1`).Scan(&ids).Error
	return ids, err
}

type WorkLanguage struct {
	WorkID   int    `gorm:"column:work_id"`
	Language string `gorm:"column:original_language"`
}

func (s *Store) CandidateWorks(excludeWorks []int) ([]WorkLanguage, error) {
	var rows []WorkLanguage
	err := s.db.Raw(`SELECT DISTINCT r.work_id, g.original_language
		FROM galgame_resource r JOIN galgame g ON g.id = r.work_id
		WHERE `+awayPredicate+` AND r.work_id <> ALL(?::int[])
		ORDER BY r.work_id`, intArray(excludeWorks)).Scan(&rows).Error
	return rows, err
}

// Snapshot records every candidate resource. A resource already pushed keeps
// the payload LetMoe was sent; one that is not is refreshed to its current state.
func (s *Store) Snapshot(excludeWorks []int) (int64, error) {
	res := s.db.Exec(`INSERT INTO galgame_resource_relocation (resource_id, work_id, uploader_id, payload)
		SELECT r.id, r.work_id, r.user_id, `+payloadExpr+`
		FROM galgame_resource r JOIN galgame g ON g.id = r.work_id
		WHERE `+awayPredicate+` AND r.work_id <> ALL(?::int[])
		ON CONFLICT (resource_id) DO UPDATE
			SET payload = EXCLUDED.payload, work_id = EXCLUDED.work_id,
			    uploader_id = EXCLUDED.uploader_id, exported_at = now()
			WHERE galgame_resource_relocation.pushed_at IS NULL`, intArray(excludeWorks))
	return res.RowsAffected, res.Error
}

func (s *Store) UnpushedLikers() ([]int, error) {
	var ids []int
	err := s.db.Raw(`SELECT DISTINCT (k->>'user_id')::int
		FROM galgame_resource_relocation m, jsonb_array_elements(m.payload->'likes') k
		WHERE m.pushed_at IS NULL ORDER BY 1`).Scan(&ids).Error
	return ids, err
}

// DropLikes takes these users' likes out of every snapshot not pushed yet.
// LetMoe turns a like into a favourite, and its clean-up of deleted accounts
// has already passed: a favourite imported for one would never be removed.
func (s *Store) DropLikes(userIDs []int) (int64, error) {
	if len(userIDs) == 0 {
		return 0, nil
	}
	gone := intArray(userIDs)
	res := s.db.Exec(`UPDATE galgame_resource_relocation m
		SET payload = jsonb_set(m.payload, '{likes}', COALESCE((
			SELECT jsonb_agg(t.k ORDER BY t.ord)
			FROM jsonb_array_elements(m.payload->'likes') WITH ORDINALITY AS t(k, ord)
			WHERE (t.k->>'user_id')::int <> ALL(?::int[])), '[]'::jsonb))
		WHERE m.pushed_at IS NULL AND EXISTS (
			SELECT 1 FROM jsonb_array_elements(m.payload->'likes') k
			WHERE (k->>'user_id')::int = ANY(?::int[]))`, gone, gone)
	return res.RowsAffected, res.Error
}

type Snapshot struct {
	ResourceID int             `gorm:"column:resource_id"`
	Payload    json.RawMessage `gorm:"column:payload"`
}

func (s *Store) Unpushed(limit int) ([]Snapshot, error) {
	var rows []Snapshot
	err := s.db.Raw(`SELECT resource_id, payload FROM galgame_resource_relocation
		WHERE pushed_at IS NULL ORDER BY resource_id LIMIT ?`, limit).Scan(&rows).Error
	return rows, err
}

func (s *Store) All() ([]Snapshot, error) {
	var rows []Snapshot
	err := s.db.Raw(`SELECT resource_id, payload FROM galgame_resource_relocation ORDER BY resource_id`).Scan(&rows).Error
	return rows, err
}

func (s *Store) SaveReceipt(resourceID int, destinationID int64, public bool) error {
	return s.db.Exec(`UPDATE galgame_resource_relocation
		SET destination_id = ?, destination_public = ?, pushed_at = COALESCE(pushed_at, now())
		WHERE resource_id = ?`, destinationID, public, resourceID).Error
}

func (s *Store) Receipts() ([]Receipt, error) {
	var rows []Receipt
	err := s.db.Raw(`SELECT resource_id AS forum_id, destination_id AS resource_id, destination_public AS public
		FROM galgame_resource_relocation WHERE destination_id IS NOT NULL ORDER BY resource_id`).Scan(&rows).Error
	return rows, err
}

func (s *Store) Retirable() ([]int, error) {
	var ids []int
	err := s.db.Raw(`SELECT m.resource_id FROM galgame_resource_relocation m
		JOIN galgame_resource r ON r.id = m.resource_id
		WHERE m.destination_id IS NOT NULL AND m.retired_at IS NULL
		ORDER BY m.resource_id`).Scan(&ids).Error
	return ids, err
}

// Retire deletes the forum's row for a resource LetMoe has confirmed. It goes
// around resourceapiv1's delete on purpose: that path debits the publish reward
// as content_removed, which the uploader would read as "资源被移除".
func (s *Store) Retire(resourceID int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var workID int
		err := tx.Raw(`SELECT r.work_id FROM galgame_resource r
			JOIN galgame_resource_relocation m ON m.resource_id = r.id
			WHERE r.id = ? AND m.destination_id IS NOT NULL AND m.retired_at IS NULL
			  AND (`+payloadExpr+`)`+stableKeys+` = m.payload`+stableKeys+`
			FOR UPDATE OF r`, resourceID).Scan(&workID).Error
		if err != nil {
			return err
		}
		if workID == 0 {
			return ErrChanged
		}
		if err := tx.Exec(`DELETE FROM galgame_resource WHERE id = ?`, resourceID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE galgame SET resource_count = GREATEST(resource_count - 1, 0) WHERE id = ?`, workID).Error; err != nil {
			return err
		}
		return tx.Exec(`UPDATE galgame_resource_relocation SET retired_at = now() WHERE resource_id = ?`, resourceID).Error
	})
}

// Forget empties the snapshot of every retired resource. The row stays: it is
// what the old resource page redirects by.
func (s *Store) Forget(dryRun bool) (int64, error) {
	const kept = `retired_at IS NOT NULL AND payload <> '{}'::jsonb`
	if dryRun {
		var n int64
		err := s.db.Raw(`SELECT COUNT(*) FROM galgame_resource_relocation WHERE ` + kept).Scan(&n).Error
		return n, err
	}
	res := s.db.Exec(`UPDATE galgame_resource_relocation SET payload = '{}'::jsonb WHERE ` + kept)
	return res.RowsAffected, res.Error
}

func intArray(ids []int) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return "{" + strings.Join(parts, ",") + "}"
}
