package push

import (
	"time"

	"gorm.io/gorm"
)

type presentationKey struct {
	Kind int16
	ID   string
}

type presentationClaim struct {
	AnchorKind int16     `gorm:"column:anchor_kind"`
	AnchorID   string    `gorm:"column:anchor_id"`
	Enqueued   time.Time `gorm:"column:enqueued"`
}

type presentationSentRow struct {
	AnchorKind int16  `gorm:"column:anchor_kind"`
	AnchorID   string `gorm:"column:anchor_id"`
	Revision   int64  `gorm:"column:revision"`
	Removed    bool   `gorm:"column:removed"`
}

func (c presentationClaim) key() presentationKey {
	return presentationKey{Kind: c.AnchorKind, ID: c.AnchorID}
}

func (p *Presenter) claimPresentations(limit int) ([]presentationClaim, error) {
	var rows []presentationClaim
	err := p.db.Raw(`
		SELECT anchor_kind, anchor_id, enqueued
		FROM anchor_presentation_queue
		ORDER BY enqueued
		LIMIT ?`, limit).Scan(&rows).Error
	return rows, err
}

func (p *Presenter) presentationSentByKeys(keys []presentationKey) (map[presentationKey]presentationSentRow, error) {
	out := map[presentationKey]presentationSentRow{}
	if len(keys) == 0 {
		return out, nil
	}
	pairs := make([][]any, len(keys))
	for i, k := range keys {
		pairs[i] = []any{k.Kind, k.ID}
	}
	var rows []presentationSentRow
	err := p.db.Raw(`
		SELECT anchor_kind, anchor_id, revision, removed
		FROM anchor_presentation_sent
		WHERE (anchor_kind, anchor_id) IN ?`, pairs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[presentationKey{Kind: r.AnchorKind, ID: r.AnchorID}] = r
	}
	return out, nil
}

func (p *Presenter) upsertPresentationSent(k presentationKey, rev int64, removed bool) error {
	return p.db.Exec(`
		INSERT INTO anchor_presentation_sent (anchor_kind, anchor_id, revision, removed, sent_at)
		VALUES (?, ?, ?, ?, now())
		ON CONFLICT (anchor_kind, anchor_id) DO UPDATE SET
			revision = EXCLUDED.revision,
			removed = EXCLUDED.removed,
			sent_at = EXCLUDED.sent_at`,
		k.Kind, k.ID, rev, removed).Error
}

func (p *Presenter) ackPresentation(c presentationClaim) error {
	return p.db.Exec(`
		DELETE FROM anchor_presentation_queue
		WHERE anchor_kind = ? AND anchor_id = ? AND enqueued = ?`,
		c.AnchorKind, c.AnchorID, c.Enqueued).Error
}

func (p *Presenter) enqueuePresentation(k presentationKey) error {
	return p.db.Exec(`
		INSERT INTO anchor_presentation_queue (anchor_kind, anchor_id)
		VALUES (?, ?)
		ON CONFLICT DO NOTHING`, k.Kind, k.ID).Error
}

func (p *Presenter) ackPresentations(claims []presentationClaim) error {
	for _, c := range claims {
		if err := p.ackPresentation(c); err != nil {
			return err
		}
	}
	return nil
}

const presentationLocalPage = 500

func (p *Presenter) pageLocalPresentations(after *presentationKey, limit int) ([]presentationKey, error) {
	q := `
		SELECT anchor_kind, anchor_id FROM (
			SELECT 1::smallint AS anchor_kind, id::text AS anchor_id FROM galgame
			UNION
			SELECT 1::smallint, work_id::text FROM feed_activity
				WHERE type = 'GALGAME_COMMENT_CREATION' AND work_id > 0
			UNION
			SELECT 2::smallint, 'resource:' || id::text FROM galgame_resource
			UNION
			SELECT 2::smallint, 'rating:' || id::text FROM galgame_rating
			UNION
			SELECT 2::smallint, 'quiz:' || id::text FROM galgame_quiz
			UNION
			SELECT 2::smallint, 'toolset:' || id::text FROM galgame_toolset
			UNION
			SELECT 2::smallint, 'website:' || id::text FROM galgame_website
		) p
	`
	args := []any{}
	if after != nil {
		q += ` WHERE (anchor_kind > ? OR (anchor_kind = ? AND anchor_id > ?))`
		args = append(args, after.Kind, after.Kind, after.ID)
	}
	q += ` ORDER BY anchor_kind, anchor_id LIMIT ?`
	args = append(args, limit)
	var rows []presentationClaim
	err := p.db.Raw(q, args...).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]presentationKey, len(rows))
	for i, r := range rows {
		out[i] = r.key()
	}
	return out, nil
}

func scanIntMap[T any](db *gorm.DB, q string, ids []T, dest any) error {
	if len(ids) == 0 {
		return nil
	}
	return db.Raw(q, ids).Scan(dest).Error
}
