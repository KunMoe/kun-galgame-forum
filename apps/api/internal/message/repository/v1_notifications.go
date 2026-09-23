package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

type V1NotificationRow struct {
	ID                      int
	SenderID                int `gorm:"column:sender_id"`
	Link                    string
	Content                 string
	Status                  string
	Type                    string
	Created                 time.Time `gorm:"column:created"`
	ItemCount               int       `gorm:"column:item_count"`
	ActorCount              int       `gorm:"column:actor_count"`
	CommunityNotificationID *int64    `gorm:"column:community_notification_id"`
}

type V1NotificationPos struct {
	Created time.Time
	ID      int
}

func applyTypePartition(q *gorm.DB, muted bool, mutedTypes []string) *gorm.DB {
	if muted {
		if len(mutedTypes) == 0 {
			return q.Where("1 = 0")
		}
		return q.Where("type IN ?", mutedTypes)
	}
	if len(mutedTypes) == 0 {
		return q
	}
	return q.Where("type NOT IN ?", mutedTypes)
}

func partitionSQL(muted bool, mutedTypes []string) (string, []any) {
	if muted {
		if len(mutedTypes) == 0 {
			return " AND 1 = 0", nil
		}
		return " AND type IN ?", []any{mutedTypes}
	}
	if len(mutedTypes) == 0 {
		return "", nil
	}
	return " AND type NOT IN ?", []any{mutedTypes}
}

func (r *MessageRepository) FindMutedTypes(userID int) ([]string, error) {
	var raw string
	err := r.db.Raw(
		`SELECT COALESCE(muted_notification_types::text, '[]') FROM kungal_user_state WHERE user_id = ?`,
		userID,
	).Scan(&raw).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if raw == "" {
		return nil, nil
	}
	var keys []string
	if err := json.Unmarshal([]byte(raw), &keys); err != nil {
		return nil, err
	}
	return keys, nil
}

func (r *MessageRepository) FindNotificationsKeyset(
	receiverID int,
	muted bool,
	mutedTypes []string,
	onlyType string,
	pos *V1NotificationPos,
	limit int,
) ([]V1NotificationRow, error) {
	q := r.db.Table("message").
		Select("id, sender_id, link, content, status, type, created, item_count, actor_count, community_notification_id").
		Where("receiver_id = ?", receiverID)
	q = applyTypePartition(q, muted, mutedTypes)
	if onlyType != "" {
		q = q.Where("type = ?", onlyType)
	}
	if pos != nil {
		q = q.Where("(created, id) < (?, ?)", pos.Created, pos.ID)
	}
	var rows []V1NotificationRow
	err := q.Order("created DESC, id DESC").Limit(limit).Scan(&rows).Error
	if rows == nil {
		rows = []V1NotificationRow{}
	}
	return rows, err
}

func (r *MessageRepository) CountUnreadPartition(receiverID int, muted bool, mutedTypes []string) (int, error) {
	q := r.db.Table("message").Where("receiver_id = ? AND status = 'unread'", receiverID)
	q = applyTypePartition(q, muted, mutedTypes)
	var n int64
	err := q.Count(&n).Error
	return int(n), err
}

// Rows, not Scan: gorm's Scan into []*int64 fails after the UPDATE has already
// committed, so the request reported 标记已读失败 and the ids were never
// forwarded, leaving the upstream fold open.
func (r *MessageRepository) MarkReadUpTo(
	receiverID, upToID int,
	muted bool,
	mutedTypes []string,
) (marked int, mirrorIDs []int64, unread int, err error) {
	mirrorIDs = []int64{}
	err = r.db.Transaction(func(tx *gorm.DB) error {
		q := `UPDATE message
SET status = 'read', updated = now()
WHERE receiver_id = ? AND status = 'unread' AND id <= ?`
		args := []any{receiverID, upToID}
		frag, extra := partitionSQL(muted, mutedTypes)
		q += frag
		args = append(args, extra...)
		q += ` RETURNING community_notification_id`

		rows, qerr := tx.Raw(q, args...).Rows()
		if qerr != nil {
			return qerr
		}
		defer rows.Close()

		n := 0
		ids := []int64{}
		for rows.Next() {
			var id sql.NullInt64
			if serr := rows.Scan(&id); serr != nil {
				return serr
			}
			n++
			if id.Valid && id.Int64 > 0 {
				ids = append(ids, id.Int64)
			}
		}
		if rerr := rows.Err(); rerr != nil {
			return rerr
		}
		marked = n
		mirrorIDs = ids

		cq := tx.Table("message").Where("receiver_id = ? AND status = 'unread'", receiverID)
		cq = applyTypePartition(cq, muted, mutedTypes)
		var remaining int64
		if cerr := cq.Count(&remaining).Error; cerr != nil {
			return cerr
		}
		unread = int(remaining)
		return nil
	})
	return marked, mirrorIDs, unread, err
}

func (r *MessageRepository) FindNotification(id, receiverID int) (*V1NotificationRow, error) {
	var row V1NotificationRow
	err := r.db.Table("message").
		Select("id, sender_id, link, content, status, type, created, item_count, actor_count, community_notification_id").
		Where("id = ? AND receiver_id = ?", id, receiverID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if row.ID == 0 {
		return nil, nil
	}
	return &row, nil
}

func (r *MessageRepository) DeleteNotification(id, receiverID int) (status string, communityID *int64, found bool, err error) {
	rows, err := r.db.Raw(
		`DELETE FROM message WHERE id = ? AND receiver_id = ? RETURNING status, community_notification_id`,
		id, receiverID,
	).Rows()
	if err != nil {
		return "", nil, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return "", nil, false, rows.Err()
	}
	var comm sql.NullInt64
	if err := rows.Scan(&status, &comm); err != nil {
		return "", nil, false, err
	}
	if comm.Valid && comm.Int64 > 0 {
		v := comm.Int64
		communityID = &v
	}
	return status, communityID, true, rows.Err()
}
