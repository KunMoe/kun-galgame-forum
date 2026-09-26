package push

import (
	"time"

	"kun-galgame-api/internal/activity/repository"
)

type Key struct {
	Type     string
	SourceID int
}

type Claim struct {
	Type     string
	SourceID int
	Backfill bool
	Enqueued time.Time
}

type feedRecord struct {
	repository.FeedRow
	IsNSFW bool `gorm:"column:is_nsfw"`
}

type sentRow struct {
	Type     string `gorm:"column:type"`
	SourceID int    `gorm:"column:source_id"`
	ActorID  int    `gorm:"column:actor_id"`
	Revision int64  `gorm:"column:revision"`
	Removed  bool   `gorm:"column:removed"`
}

func (p *Pusher) claim(limit int) ([]Claim, error) {
	var rows []Claim
	err := p.db.Raw(`
		SELECT type, source_id, backfill, enqueued
		FROM activity_push_queue
		ORDER BY enqueued
		LIMIT ?`, limit).Scan(&rows).Error
	return rows, err
}

func (p *Pusher) feedByKeys(keys []Key) (map[Key]feedRecord, error) {
	out := map[Key]feedRecord{}
	if len(keys) == 0 {
		return out, nil
	}
	pairs := make([][]any, len(keys))
	for i, k := range keys {
		pairs[i] = []any{k.Type, k.SourceID}
	}
	var rows []feedRecord
	err := p.db.Raw(`
		SELECT fa.id AS row_id, fa.type AS type_str, fa.source_id, fa.user_id, fa.work_id,
			fa.content, fa.link, fa.created, fa.is_nsfw
		FROM feed_activity fa
		WHERE (fa.type, fa.source_id) IN ?`, pairs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[Key{Type: r.TypeStr, SourceID: r.SourceID}] = r
	}
	return out, nil
}

func (p *Pusher) sentByKeys(keys []Key) (map[Key]sentRow, error) {
	out := map[Key]sentRow{}
	if len(keys) == 0 {
		return out, nil
	}
	pairs := make([][]any, len(keys))
	for i, k := range keys {
		pairs[i] = []any{k.Type, k.SourceID}
	}
	var rows []sentRow
	err := p.db.Raw(`
		SELECT type, source_id, actor_id, revision, removed
		FROM activity_push_sent
		WHERE (type, source_id) IN ?`, pairs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[Key{Type: r.Type, SourceID: r.SourceID}] = r
	}
	return out, nil
}

func (p *Pusher) upsertSent(k Key, actorID int, rev int64, removed bool) error {
	return p.db.Exec(`
		INSERT INTO activity_push_sent (type, source_id, actor_id, revision, removed, sent_at)
		VALUES (?, ?, ?, ?, ?, now())
		ON CONFLICT (type, source_id) DO UPDATE SET
			actor_id = EXCLUDED.actor_id,
			revision = EXCLUDED.revision,
			removed = EXCLUDED.removed,
			sent_at = EXCLUDED.sent_at`,
		k.Type, k.SourceID, actorID, rev, removed).Error
}

func (p *Pusher) ack(c Claim) error {
	return p.db.Exec(`
		DELETE FROM activity_push_queue
		WHERE type = ? AND source_id = ? AND enqueued = ?`,
		c.Type, c.SourceID, c.Enqueued).Error
}

func (p *Pusher) requeue(k Key) error {
	return p.db.Exec(`
		UPDATE activity_push_queue SET enqueued = clock_timestamp()
		WHERE type = ? AND source_id = ?`, k.Type, k.SourceID).Error
}

func (p *Pusher) enqueue(k Key) error {
	return p.db.Exec(`
		INSERT INTO activity_push_queue (type, source_id, backfill)
		VALUES (?, ?, false)
		ON CONFLICT DO NOTHING`, k.Type, k.SourceID).Error
}

func (p *Pusher) pageLocal(after *Key, limit int) ([]feedRecord, error) {
	types := make([]string, 0, len(pushed))
	for t := range pushed {
		types = append(types, t)
	}
	q := `
		SELECT fa.id AS row_id, fa.type AS type_str, fa.source_id, fa.user_id, fa.work_id,
			fa.content, fa.link, fa.created, fa.is_nsfw
		FROM feed_activity fa
		WHERE fa.type IN ?
	`
	args := []any{types}
	if after != nil {
		q += ` AND (fa.type > ? OR (fa.type = ? AND fa.source_id > ?))`
		args = append(args, after.Type, after.Type, after.SourceID)
	}
	q += ` ORDER BY fa.type, fa.source_id LIMIT ?`
	args = append(args, limit)
	var rows []feedRecord
	err := p.db.Raw(q, args...).Scan(&rows).Error
	return rows, err
}

func (p *Pusher) namesByToolset(ids []int) (map[int]string, error) {
	out := map[int]string{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		ID   int    `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	err := p.db.Raw(`SELECT id, name FROM galgame_toolset WHERE id IN ?`, ids).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ID] = r.Name
	}
	return out, nil
}

func (p *Pusher) namesByWebsiteURL(urls []string) (map[string]string, error) {
	out := map[string]string{}
	if len(urls) == 0 {
		return out, nil
	}
	var rows []struct {
		URL  string `gorm:"column:url"`
		Name string `gorm:"column:name"`
	}
	err := p.db.Raw(`SELECT url, name FROM galgame_website WHERE url IN ?`, urls).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.URL] = r.Name
	}
	return out, nil
}
