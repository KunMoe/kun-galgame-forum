package repository

import (
	"time"

	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *LotteryRepository) FindByTopicNewestFirst(topicID int) ([]model.TopicLottery, error) {
	var rows []model.TopicLottery
	err := r.db.Where("topic_id = ?", topicID).Order("created DESC, id DESC").Find(&rows).Error
	return rows, err
}

func (r *LotteryRepository) LockByID(tx *gorm.DB, id int) (*model.TopicLottery, error) {
	var row model.TopicLottery
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).Take(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

type LotteryEntryKey struct {
	Created time.Time
	ID      int
}

func (r *LotteryRepository) FindEntriesAfter(lotteryID, limit int, after *LotteryEntryKey) ([]model.TopicLotteryEntry, error) {
	q := r.db.Where("lottery_id = ?", lotteryID)
	if after != nil {
		q = q.Where("(created, id) > (?, ?)", after.Created, after.ID)
	}
	var rows []model.TopicLotteryEntry
	err := q.Order("created ASC, id ASC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *LotteryRepository) InsertEntryIfAbsent(tx *gorm.DB, lotteryID, userID int) (bool, error) {
	var ids []int
	err := tx.Raw(`
		INSERT INTO topic_lottery_entry (lottery_id, user_id, created, updated)
		VALUES (?, ?, NOW(), NOW())
		ON CONFLICT (lottery_id, user_id) DO NOTHING
		RETURNING id`, lotteryID, userID).Scan(&ids).Error
	return len(ids) > 0, err
}

func (r *LotteryRepository) DeleteUnwonEntry(tx *gorm.DB, lotteryID, userID int) (bool, error) {
	res := tx.Where("lottery_id = ? AND user_id = ? AND prize_id = 0", lotteryID, userID).
		Delete(&model.TopicLotteryEntry{})
	return res.RowsAffected > 0, res.Error
}

// CancelIfOpen flips an open lottery to cancelled and hands back the escrow it
// held, zeroing it in the same statement: the guard on status is what keeps a
// cancel from racing the sweep's open -> drawing flip, and the zeroing is what
// keeps a later delete from refunding the author a second time.
func (r *LotteryRepository) CancelIfOpen(tx *gorm.DB, lotteryID int, now time.Time) (escrow int, ok bool, err error) {
	var rows []struct{ Old int }
	err = tx.Raw(`
		UPDATE topic_lottery l SET status = 'cancelled', point_escrow = 0, updated = ?
		FROM (SELECT id, point_escrow AS old FROM topic_lottery WHERE id = ? AND status = 'open' FOR UPDATE) o
		WHERE l.id = o.id
		RETURNING o.old`, now, lotteryID).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return 0, false, err
	}
	return rows[0].Old, true, nil
}

// DeleteDeletable removes the lottery unless it is being drawn, and unless it
// is drawn when allowDrawn is false. It returns the escrow the row held.
func (r *LotteryRepository) DeleteDeletable(tx *gorm.DB, lotteryID int, allowDrawn bool) (escrow int, ok bool, err error) {
	excluded := []string{model.LotteryStatusDrawing}
	if !allowDrawn {
		excluded = append(excluded, model.LotteryStatusDrawn)
	}
	var rows []struct{ PointEscrow int }
	err = tx.Raw(`DELETE FROM topic_lottery WHERE id = ? AND status NOT IN ? RETURNING point_escrow`,
		lotteryID, excluded).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return 0, false, err
	}
	return rows[0].PointEscrow, true, nil
}

func (r *LotteryRepository) AdjustCachedMoemoepoint(tx *gorm.DB, userID, delta int) error {
	return tx.Exec(`UPDATE kungal_user_state SET moemoepoint = moemoepoint + ? WHERE user_id = ?`,
		delta, userID).Error
}

func (r *LotteryRepository) FindWinner(lotteryID, entryID int) (*model.TopicLotteryEntry, error) {
	var row model.TopicLotteryEntry
	err := r.db.Where("id = ? AND lottery_id = ? AND prize_id > 0", entryID, lotteryID).Take(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *LotteryRepository) FindPrizeByID(id int) (*model.TopicLotteryPrize, error) {
	var row model.TopicLotteryPrize
	err := r.db.Where("id = ?", id).Take(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// MoveFulfillment only moves the entry if it is still in one of from, so two
// writers racing on the same winner cannot both succeed.
func (r *LotteryRepository) MoveFulfillment(tx *gorm.DB, entryID int, from []string, to string, now time.Time) (bool, error) {
	res := tx.Model(&model.TopicLotteryEntry{}).
		Where("id = ? AND fulfillment IN ?", entryID, from).
		Updates(map[string]any{"fulfillment": to, "updated": now})
	return res.RowsAffected > 0, res.Error
}

func (r *LotteryRepository) TouchTopic(tx *gorm.DB, topicID int, now time.Time) error {
	return tx.Model(&model.Topic{}).
		Where("id = ? AND created > ?", topicID, model.BumpCutoff(now)).
		Updates(map[string]any{"status_update_time": now}).Error
}
