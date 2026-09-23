package repository

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type HiddenTopicRow struct {
	ID               int       `gorm:"column:id"`
	Title            string    `gorm:"column:title"`
	Status           int       `gorm:"column:status"`
	HiddenBy         string    `gorm:"column:hidden_by"`
	ReplyCount       int       `gorm:"column:reply_count"`
	StatusUpdateTime time.Time `gorm:"column:status_update_time"`
	Created          time.Time `gorm:"column:created"`
	UserID           int       `gorm:"column:user_id"`
}

func (r *TopicRepository) hiddenTopicsQuery(hiddenBy, keywords string) *gorm.DB {
	q := r.db.Table("topic").Where("status = ?", 1)
	if hiddenBy != "" {
		q = q.Where("hidden_by = ?", hiddenBy)
	}
	if keywords != "" {
		esc := strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(keywords)
		q = q.Where("title ILIKE ?", "%"+esc+"%")
	}
	return q
}

func (r *TopicRepository) ListHiddenTopics(hiddenBy, keywords string, offset, limit int) ([]HiddenTopicRow, int, error) {
	var total int64
	if err := r.hiddenTopicsQuery(hiddenBy, keywords).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []HiddenTopicRow
	err := r.hiddenTopicsQuery(hiddenBy, keywords).
		Select("id, title, status, hidden_by, reply_count, status_update_time, created, user_id").
		Order("status_update_time DESC, id DESC").
		Offset(offset).Limit(limit).
		Scan(&rows).Error
	return rows, int(total), err
}

func (r *TopicRepository) FindHiddenTopicRow(tx *gorm.DB, topicID int, lock bool) (*HiddenTopicRow, error) {
	q := tx.Table("topic").
		Select("id, title, status, hidden_by, reply_count, status_update_time, created, user_id").
		Where("id = ?", topicID)
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var row HiddenTopicRow
	if err := q.Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

type TopicPurgeCounts struct {
	Replies        int64
	Comments       int64
	Polls          int64
	Lotteries      int64
	DrawnLotteries int64
	Favorites      int64
}

func (r *TopicRepository) CountTopicDependents(tx *gorm.DB, topicID int) (TopicPurgeCounts, error) {
	var c TopicPurgeCounts
	for _, x := range []struct {
		table string
		out   *int64
	}{
		{"topic_reply", &c.Replies},
		{"topic_comment", &c.Comments},
		{"topic_poll", &c.Polls},
		{"topic_lottery", &c.Lotteries},
		{"topic_favorite", &c.Favorites},
	} {
		if err := tx.Table(x.table).Where("topic_id = ?", topicID).Count(x.out).Error; err != nil {
			return c, err
		}
	}
	err := tx.Table("topic_lottery").Where("topic_id = ? AND status = ?", topicID, "drawn").Count(&c.DrawnLotteries).Error
	return c, err
}

type LotteryEscrowRow struct {
	ID          int    `gorm:"column:id"`
	UserID      int    `gorm:"column:user_id"`
	Status      string `gorm:"column:status"`
	PointEscrow int    `gorm:"column:point_escrow"`
}

// LockTopicLotteries takes the topic's lotteries FOR UPDATE, so the minute
// sweep, which claims with SKIP LOCKED, leaves them alone until the purge ends.
func (r *TopicRepository) LockTopicLotteries(tx *gorm.DB, topicID int) ([]LotteryEscrowRow, error) {
	var rows []LotteryEscrowRow
	err := tx.Raw(`SELECT id, user_id, status, point_escrow FROM topic_lottery WHERE topic_id = ? ORDER BY id FOR UPDATE`,
		topicID).Scan(&rows).Error
	return rows, err
}

func (r *TopicRepository) SumOpenEscrow(tx *gorm.DB, topicID int) (int, error) {
	var n int
	err := tx.Raw(`SELECT COALESCE(SUM(point_escrow), 0) FROM topic_lottery WHERE topic_id = ? AND status = 'open'`,
		topicID).Scan(&n).Error
	return n, err
}

func (r *TopicRepository) PurgeTopic(tx *gorm.DB, topicID int) error {
	// topic_view_daily has no FK to topic (migration 050 created it
	// without one), so the cascade on topic cannot reach it. Its key
	// column is entity_id, not topic_id — the viewstats bucket tables
	// share one shape across domains.
	if err := tx.Exec("DELETE FROM topic_view_daily WHERE entity_id = ?", topicID).Error; err != nil {
		return err
	}
	// BuildTopicLink produces only '/topic/<id>' and '/topic/<id>?...';
	// LIKE '/topic/<id>%' would also match topic 1234 when deleting 123.
	link := fmt.Sprintf("/topic/%d", topicID)
	if err := tx.Exec("DELETE FROM message WHERE link = ? OR link LIKE ?", link, link+"?%").Error; err != nil {
		return err
	}
	return tx.Exec("DELETE FROM topic WHERE id = ?", topicID).Error
}
