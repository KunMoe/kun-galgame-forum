package repository

import (
	"fmt"
	"strings"
	"time"
)

// CountFeedSince counts rows that occurred after the second named by seen:
// occurred_at goes out at second precision, so a client marking the newest
// item it showed passes that item's truncated time.
func (r *ActivityRepository) CountFeedSince(q FeedQuery, seen time.Time, limit int) (int, error) {
	conds, args := feedConds(q)
	conds = append(conds, "fa.created >= ?")
	args = append(args, seen.Truncate(time.Second).Add(time.Second))
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
