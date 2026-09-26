package repository

import (
	"fmt"
	"strings"
	"time"
)

func (r *ActivityRepository) CountFeedSince(q FeedQuery, from time.Time, limit int) (int, error) {
	if q.ActorIDs != nil && len(q.ActorIDs) == 0 {
		return 0, nil
	}
	created, createdArgs := []string{"fa.created >= ?"}, []any{from}
	if q.ActorIDs != nil {
		sql, args := feedActorUnion(q, "SELECT 1 ", created, createdArgs, false, limit)
		var n int
		err := r.db.Raw(sql, args...).Scan(&n).Error
		return n, err
	}
	conds, args := feedConds(q)
	conds = append(conds, created...)
	args = append(args, createdArgs...)
	sql := "SELECT count(*) FROM (SELECT 1 FROM feed_activity fa WHERE " + strings.Join(conds, " AND ") +
		fmt.Sprintf(" LIMIT %d) capped", limit)
	var n int
	err := r.db.Raw(sql, args...).Scan(&n).Error
	return n, err
}

func (r *ActivityRepository) FollowingSeenAt(userID int) (*time.Time, error) {
	var rows []struct {
		SeenAt *time.Time `gorm:"column:following_activity_seen_at"`
	}
	err := r.db.Raw(`SELECT following_activity_seen_at FROM kungal_user_state WHERE user_id = ?`, userID).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[0].SeenAt, nil
}

func (r *ActivityRepository) MarkFollowingSeen(userID int, at time.Time) (time.Time, error) {
	var seen time.Time
	err := r.db.Raw(`
		INSERT INTO kungal_user_state (user_id, following_activity_seen_at)
		VALUES (?, date_trunc('second', LEAST(?::timestamptz, now())))
		ON CONFLICT (user_id) DO UPDATE SET following_activity_seen_at = GREATEST(
			kungal_user_state.following_activity_seen_at, EXCLUDED.following_activity_seen_at)
		RETURNING following_activity_seen_at`, userID, at).Row().Scan(&seen)
	return seen, err
}
