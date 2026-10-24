package repository

import (
	"time"

	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
)

type PollTotals struct {
	PollID         int `gorm:"column:poll_id"`
	TotalVoteCount int `gorm:"column:total_votes"`
	VoterCount     int `gorm:"column:voter_count"`
}

type PollVoterSample struct {
	PollID int `gorm:"column:poll_id"`
	UserID int `gorm:"column:user_id"`
}

type PollChoice struct {
	PollID   int `gorm:"column:poll_id"`
	OptionID int `gorm:"column:option_id"`
}

type PollVoteRow struct {
	ID       int       `gorm:"column:id"`
	UserID   int       `gorm:"column:user_id"`
	OptionID int       `gorm:"column:option_id"`
	Created  time.Time `gorm:"column:created"`
}

func (r *PollRepository) ListPollsOfTopic(topicID int) ([]model.TopicPoll, error) {
	var rows []model.TopicPoll
	err := r.db.Where("topic_id = ?", topicID).
		Order("created DESC, id DESC").
		Find(&rows).Error
	return rows, err
}

func (r *PollRepository) ListPollOptions(pollIDs []int) ([]model.TopicPollOption, error) {
	if len(pollIDs) == 0 {
		return nil, nil
	}
	var rows []model.TopicPollOption
	err := r.db.Where("poll_id IN ?", pollIDs).
		Order("poll_id ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *PollRepository) PollTotals(pollIDs []int) ([]PollTotals, error) {
	if len(pollIDs) == 0 {
		return nil, nil
	}
	var rows []PollTotals
	err := r.db.Table("topic_poll_vote").
		Select("poll_id, COUNT(*) AS total_votes, COUNT(DISTINCT user_id) AS voter_count").
		Where("poll_id IN ?", pollIDs).
		Group("poll_id").
		Scan(&rows).Error
	return rows, err
}

// The sample is the earliest distinct voters, and a voter's position is their
// first vote: ORDER BY created, id with no id tie-breaker used to let Postgres
// pick which five of a same-second batch to show.
func (r *PollRepository) SamplePollVoters(pollIDs []int, perPoll int) ([]PollVoterSample, error) {
	if len(pollIDs) == 0 || perPoll <= 0 {
		return nil, nil
	}
	var rows []PollVoterSample
	err := r.db.Raw(`
		SELECT poll_id, user_id FROM (
			SELECT poll_id, user_id,
			       ROW_NUMBER() OVER (PARTITION BY poll_id ORDER BY created, id) AS rn
			FROM (
				SELECT DISTINCT ON (poll_id, user_id) poll_id, user_id, created, id
				FROM topic_poll_vote
				WHERE poll_id IN ?
				ORDER BY poll_id, user_id, created, id
			) first_votes
		) ranked
		WHERE rn <= ?
		ORDER BY poll_id, rn`, pollIDs, perPoll).Scan(&rows).Error
	return rows, err
}

func (r *PollRepository) ViewerChoices(pollIDs []int, userID int) ([]PollChoice, error) {
	if len(pollIDs) == 0 || userID <= 0 {
		return nil, nil
	}
	var rows []PollChoice
	err := r.db.Table("topic_poll_vote").
		Select("poll_id, option_id").
		Where("poll_id IN ? AND user_id = ?", pollIDs, userID).
		Scan(&rows).Error
	return rows, err
}

func (r *PollRepository) ListPollVotesKeyset(pollID, limit int, pos *EngageHistoryPos) ([]PollVoteRow, error) {
	q := r.db.Table("topic_poll_vote").
		Select("id, user_id, option_id, created").
		Where("poll_id = ?", pollID)
	if pos != nil {
		q = q.Where("(created, id) < (?, ?)", pos.Created, pos.ID)
	}
	var rows []PollVoteRow
	err := q.Order("created DESC, id DESC").Limit(limit + 1).Scan(&rows).Error
	return rows, err
}

// topic_poll.updated, topic_poll_option.updated and topic_poll_vote.updated
// are all NOT NULL with no default: leaving one out of a raw INSERT is what
// made every v1 favorite and upvote answer 500 with SQLSTATE 23502 in W4.
func (r *PollRepository) InsertPoll(tx *gorm.DB, p *model.TopicPoll) (int, time.Time, error) {
	var (
		id      int
		created time.Time
	)
	row := tx.Raw(`
		INSERT INTO topic_poll
			(title, description, type, min_choice, max_choice, deadline,
			 result_visibility, is_anonymous, can_change_vote, topic_id, user_id, created, updated)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now(), now())
		RETURNING id, created`,
		p.Title, p.Description, p.Type, p.MinChoice, p.MaxChoice, p.Deadline,
		p.ResultVisibility, p.IsAnonymous, p.CanChangeVote, p.TopicID, p.UserID).Row()
	if err := row.Err(); err != nil {
		return 0, time.Time{}, err
	}
	if err := row.Scan(&id, &created); err != nil {
		return 0, time.Time{}, err
	}
	return id, created, nil
}

func (r *PollRepository) InsertPollOptions(tx *gorm.DB, pollID int, texts []string) error {
	for _, text := range texts {
		if err := tx.Exec(`
			INSERT INTO topic_poll_option (text, poll_id, vote_count, created, updated)
			VALUES (?, ?, 0, now(), now())`, text, pollID).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *PollRepository) UpdatePollRow(tx *gorm.DB, pollID int, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	fields["updated"] = time.Now()
	return tx.Table("topic_poll").Where("id = ?", pollID).Updates(fields).Error
}

func (r *PollRepository) UpdatePollOptionText(tx *gorm.DB, optionID int, text string) error {
	return tx.Exec(`UPDATE topic_poll_option SET text = ?, updated = now() WHERE id = ?`, text, optionID).Error
}

func (r *PollRepository) DeletePollOptions(tx *gorm.DB, optionIDs []int) error {
	if len(optionIDs) == 0 {
		return nil
	}
	return tx.Exec(`DELETE FROM topic_poll_option WHERE id IN ?`, optionIDs).Error
}

func (r *PollRepository) LockUserVotes(tx *gorm.DB, pollID, userID int) ([]int, error) {
	var ids []int
	err := tx.Raw(`
		SELECT option_id FROM topic_poll_vote
		WHERE poll_id = ? AND user_id = ?
		ORDER BY option_id
		FOR UPDATE`, pollID, userID).Scan(&ids).Error
	return ids, err
}

// The counter moves only for the rows that really moved: an unconditional
// vote_count + 1 is how the reaction counters drifted before W4, and here a
// repeated PUT of the same choice would double every option it names.
func (r *PollRepository) DeleteUserVotesExcept(tx *gorm.DB, pollID, userID int, keep []int) ([]int, error) {
	q := `DELETE FROM topic_poll_vote WHERE poll_id = ? AND user_id = ?`
	args := []any{pollID, userID}
	if len(keep) > 0 {
		q += ` AND option_id NOT IN ?`
		args = append(args, keep)
	}
	q += ` RETURNING option_id`
	var ids []int
	err := tx.Raw(q, args...).Scan(&ids).Error
	return ids, err
}

func (r *PollRepository) InsertUserVote(tx *gorm.DB, pollID, optionID, userID int) (int, bool, error) {
	return returningID(tx, `
		INSERT INTO topic_poll_vote (poll_id, option_id, user_id, created, updated)
		VALUES (?, ?, ?, now(), now())
		ON CONFLICT (poll_id, option_id, user_id) DO NOTHING
		RETURNING id`, pollID, optionID, userID)
}

func (r *PollRepository) AddOptionVoteCounts(tx *gorm.DB, optionIDs []int, delta int) error {
	if len(optionIDs) == 0 {
		return nil
	}
	return tx.Exec(`
		UPDATE topic_poll_option SET vote_count = vote_count + ?, updated = now()
		WHERE id IN ?`, delta, optionIDs).Error
}
