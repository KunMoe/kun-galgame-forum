package repository

import (
	"time"

	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
)

type LotteryRepository struct {
	db *gorm.DB
}

func NewLotteryRepository(db *gorm.DB) *LotteryRepository {
	return &LotteryRepository{db: db}
}

func (r *LotteryRepository) DB() *gorm.DB { return r.db }

func (r *LotteryRepository) FindByID(id int) (*model.TopicLottery, error) {
	var lottery model.TopicLottery
	err := r.db.First(&lottery, id).Error
	return &lottery, err
}

func (r *LotteryRepository) CountByTopicID(topicID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.TopicLottery{}).Where("topic_id = ?", topicID).Count(&count).Error
	return count, err
}

func (r *LotteryRepository) Create(tx *gorm.DB, lottery *model.TopicLottery) error {
	return tx.Create(lottery).Error
}

func (r *LotteryRepository) UpdateFields(tx *gorm.DB, lotteryID int, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	return tx.Model(&model.TopicLottery{}).Where("id = ?", lotteryID).Updates(fields).Error
}

func (r *LotteryRepository) FindPrizes(lotteryID int) ([]model.TopicLotteryPrize, error) {
	var rows []model.TopicLotteryPrize
	err := r.db.Where("lottery_id = ?", lotteryID).Order("sort_order ASC, id ASC").Find(&rows).Error
	return rows, err
}

func (r *LotteryRepository) FindPrizesForLotteries(ids []int) ([]model.TopicLotteryPrize, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []model.TopicLotteryPrize
	err := r.db.Where("lottery_id IN ?", ids).Order("lottery_id ASC, sort_order ASC, id ASC").Find(&rows).Error
	return rows, err
}

func (r *LotteryRepository) CreatePrize(tx *gorm.DB, prize *model.TopicLotteryPrize) error {
	return tx.Create(prize).Error
}

func (r *LotteryRepository) DeletePrizes(tx *gorm.DB, lotteryID int) error {
	if err := tx.Where("lottery_id = ?", lotteryID).Delete(&model.TopicLotteryCode{}).Error; err != nil {
		return err
	}
	return tx.Where("lottery_id = ?", lotteryID).Delete(&model.TopicLotteryPrize{}).Error
}

func (r *LotteryRepository) CreateCode(tx *gorm.DB, code *model.TopicLotteryCode) error {
	return tx.Create(code).Error
}

// Keyed by prize id across every lottery asked for: the topic page renders up
// to ten lotteries at once and one query per lottery is one query too many on a
// path every topic view hits.
func (r *LotteryRepository) CountCodesForLotteries(lotteryIDs []int) (map[int]int, error) {
	if len(lotteryIDs) == 0 {
		return map[int]int{}, nil
	}
	var rows []struct {
		PrizeID int
		Total   int
	}
	err := r.db.Model(&model.TopicLotteryCode{}).
		Select("prize_id, COUNT(*) AS total").
		Where("lottery_id IN ?", lotteryIDs).
		Group("prize_id").Find(&rows).Error
	out := make(map[int]int, len(rows))
	for _, row := range rows {
		out[row.PrizeID] = row.Total
	}
	return out, err
}

// TakeCode hands one unclaimed code to a winner. The UPDATE is the lock: two
// concurrent claims cannot both match claimed_by = 0 on the same row, so the
// loser gets RowsAffected 0 and retries against the next code instead of two
// people being handed the same key.
func (r *LotteryRepository) TakeCode(tx *gorm.DB, prizeID, userID int) (*model.TopicLotteryCode, error) {
	var code model.TopicLotteryCode
	err := tx.Where("prize_id = ? AND claimed_by = 0", prizeID).Order("id ASC").First(&code).Error
	if err != nil {
		return nil, err
	}
	now := time.Now()
	res := tx.Model(&model.TopicLotteryCode{}).
		Where("id = ? AND claimed_by = 0", code.ID).
		Updates(map[string]any{"claimed_by": userID, "claimed_at": now})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	code.ClaimedBy = userID
	code.ClaimedAt = &now
	return &code, nil
}

func (r *LotteryRepository) FindCodeByID(id int) (*model.TopicLotteryCode, error) {
	var code model.TopicLotteryCode
	err := r.db.First(&code, id).Error
	return &code, err
}

func (r *LotteryRepository) FindEntry(lotteryID, userID int) (*model.TopicLotteryEntry, error) {
	var entry model.TopicLotteryEntry
	err := r.db.Where("lottery_id = ? AND user_id = ?", lotteryID, userID).First(&entry).Error
	return &entry, err
}

func (r *LotteryRepository) FindEntriesForUser(lotteryIDs []int, userID int) (map[int]model.TopicLotteryEntry, error) {
	if len(lotteryIDs) == 0 || userID == 0 {
		return map[int]model.TopicLotteryEntry{}, nil
	}
	var rows []model.TopicLotteryEntry
	err := r.db.Where("lottery_id IN ? AND user_id = ?", lotteryIDs, userID).Find(&rows).Error
	out := make(map[int]model.TopicLotteryEntry, len(rows))
	for _, row := range rows {
		out[row.LotteryID] = row
	}
	return out, err
}

func (r *LotteryRepository) FindEntries(lotteryID int) ([]model.TopicLotteryEntry, error) {
	var rows []model.TopicLotteryEntry
	err := r.db.Where("lottery_id = ?", lotteryID).Order("id ASC").Find(&rows).Error
	return rows, err
}

func (r *LotteryRepository) FindWinnersForLotteries(ids []int) ([]model.TopicLotteryEntry, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []model.TopicLotteryEntry
	err := r.db.Where("lottery_id IN ? AND prize_id > 0", ids).
		Order("lottery_id ASC, prize_id ASC, id ASC").Find(&rows).Error
	return rows, err
}

func (r *LotteryRepository) CreateEntry(tx *gorm.DB, entry *model.TopicLotteryEntry) error {
	return tx.Create(entry).Error
}

func (r *LotteryRepository) SyncEntryCount(tx *gorm.DB, lotteryID int) error {
	return tx.Exec(`
		UPDATE topic_lottery SET entry_count =
			(SELECT COUNT(*) FROM topic_lottery_entry WHERE lottery_id = ?)
		WHERE id = ?`, lotteryID, lotteryID).Error
}

func (r *LotteryRepository) UpdateEntryFields(tx *gorm.DB, entryID int, fields map[string]any) error {
	return tx.Model(&model.TopicLotteryEntry{}).Where("id = ?", entryID).Updates(fields).Error
}

func (r *LotteryRepository) HasRepliedTo(topicID, userID int) (bool, error) {
	var count int64
	err := r.db.Table("topic_reply").
		Where("topic_id = ? AND user_id = ? AND status = 0", topicID, userID).
		Count(&count).Error
	return count > 0, err
}

type FloorReply struct {
	Floor  int
	UserID int
}

func (r *LotteryRepository) FindRepliesByFloors(topicID int, floors []int) ([]FloorReply, error) {
	if len(floors) == 0 {
		return nil, nil
	}
	var rows []FloorReply
	err := r.db.Table("topic_reply").
		Select("floor, user_id").
		Where("topic_id = ? AND status = 0 AND floor IN ?", topicID, floors).
		Order("floor ASC").Find(&rows).Error
	return rows, err
}

// A lottery left in 'drawing' is picked up again after this long. Recovery is
// safe rather than a double-draw risk: the draw writes its winners in one
// transaction, so a process that died mid-draw committed nothing, and one that
// finished left status = 'drawn', which this does not match.
const drawStaleAfter = 15 * time.Minute

// ClaimDue takes at most limit open lotteries whose draw condition has fired and
// flips them to 'drawing' in the same statement. The status flip IS the lock:
// the sweep runs every minute and a slow draw must not be started twice, which
// would hand out a second set of winners for the same prizes.
func (r *LotteryRepository) ClaimDue(now time.Time, limit int) ([]model.TopicLottery, error) {
	var rows []model.TopicLottery
	err := r.db.Raw(`
		UPDATE topic_lottery SET status = 'drawing', updated = ?
		WHERE id IN (
			SELECT id FROM topic_lottery
			WHERE (
				status = 'open'
				AND (
				  (draw_mode = 'deadline' AND deadline IS NOT NULL AND deadline <= ?)
				  OR (draw_mode = 'threshold' AND draw_threshold > 0 AND entry_count >= draw_threshold)
				)
			  ) OR (status = 'drawing' AND updated <= ?)
			ORDER BY id ASC
			LIMIT ?
			FOR UPDATE SKIP LOCKED
		)
		RETURNING *`, now, now, now.Add(-drawStaleAfter), limit).Scan(&rows).Error
	return rows, err
}

// ReleaseDrawing puts a lottery whose draw transaction failed back in the queue.
func (r *LotteryRepository) ReleaseDrawing(lotteryID int) error {
	return r.db.Exec(`UPDATE topic_lottery SET status = 'open' WHERE id = ? AND status = 'drawing'`,
		lotteryID).Error
}

// ClaimPollsPastDeadline is the poll half of the deadline sweep. topic_poll has
// carried a notification_sent column since it was created with no reader and no
// writer: the "tell the voters it closed" feature was declared and never built,
// so a poll simply went quiet. This is that writer, and the flag is what keeps
// the every-minute sweep from re-notifying.
func (r *LotteryRepository) ClaimPollsPastDeadline(now time.Time, limit int) ([]model.TopicPoll, error) {
	var rows []model.TopicPoll
	err := r.db.Raw(`
		UPDATE topic_poll SET notification_sent = true, updated = ?
		WHERE id IN (
			SELECT id FROM topic_poll
			WHERE notification_sent = false
			  AND deadline IS NOT NULL AND deadline <= ?
			ORDER BY id ASC
			LIMIT ?
			FOR UPDATE SKIP LOCKED
		)
		RETURNING *`, now, now, limit).Scan(&rows).Error
	return rows, err
}

// ClaimExpiredCodeWins flips past-deadline pending code wins to 'forfeited' in
// the same statement that selects them, so the every-minute sweep cannot notify
// the same winner twice.
func (r *LotteryRepository) ClaimExpiredCodeWins(now time.Time, limit int) ([]model.TopicLotteryEntry, error) {
	var rows []model.TopicLotteryEntry
	err := r.db.Raw(`
		UPDATE topic_lottery_entry SET fulfillment = 'forfeited', updated = ?
		WHERE id IN (
			SELECT id FROM topic_lottery_entry
			WHERE fulfillment = 'pending'
			  AND code_id <> 0
			  AND claim_deadline IS NOT NULL AND claim_deadline <= ?
			ORDER BY id ASC
			LIMIT ?
			FOR UPDATE SKIP LOCKED
		)
		RETURNING *`, now, now, limit).Scan(&rows).Error
	return rows, err
}

func (r *LotteryRepository) FindPollVoterIDs(pollID int) ([]int, error) {
	var ids []int
	err := r.db.Table("topic_poll_vote").Distinct("user_id").
		Where("poll_id = ?", pollID).Pluck("user_id", &ids).Error
	return ids, err
}
