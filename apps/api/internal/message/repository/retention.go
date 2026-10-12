package repository

import "time"

// trg_feed_message deletes the best_answer_set activity of a deleted solution
// row, so expiring one would erase that activity from the public feed.
const retentionKeepsTypes = "'solution'"

func (r *MessageRepository) DeleteReadBefore(cutoff time.Time, batch int) (int64, error) {
	var total int64
	for {
		res := r.db.Exec(`DELETE FROM message WHERE id IN (
			SELECT id FROM message
			WHERE status = 'read' AND created < ? AND type NOT IN (`+retentionKeepsTypes+`)
			LIMIT ?)`, cutoff, batch)
		if res.Error != nil {
			return total, res.Error
		}
		total += res.RowsAffected
		if res.RowsAffected < int64(batch) {
			return total, nil
		}
	}
}
