package repository

import (
	"errors"
	"fmt"
	"slices"
	"time"

	topicModel "kun-galgame-api/internal/topic/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PurgeRepository struct {
	db *gorm.DB
}

func NewPurgeRepository(db *gorm.DB) *PurgeRepository {
	return &PurgeRepository{db: db}
}

var ErrLotteryDrawing = errors.New("purge: one of the user's lotteries is being drawn")

const ArchiveRetention = 30 * 24 * time.Hour

type UserContentCounts struct {
	Topics           int64
	Replies          int64
	TopicComments    int64
	Ratings          int64
	Resources        int64
	Websites         int64
	Toolsets         int64
	ToolsetResources int64
	Polls            int64
	Lotteries        int64
	Drafts           int64
	Quizzes          int64
	Collections      int64
	Todos            int64
	ChatMessages     int64
	Messages         int64
	Interactions     int64
	Total            int64
}

type PurgeReceipt struct {
	PurgeID  uuid.UUID
	Archived map[string]int64
}

func (r *PurgeRepository) CountUserContent(userID int) (UserContentCounts, error) {
	var s UserContentCounts
	var firstErr error
	countBy := func(table, col string) int64 {
		var n int64
		if err := r.db.Table(table).Where(col+" = ?", userID).Count(&n).Error; err != nil && firstErr == nil {
			firstErr = fmt.Errorf("count %s.%s: %w", table, col, err)
		}
		return n
	}

	s.Topics = countBy("topic", "user_id")
	s.Replies = countBy("topic_reply", "user_id")
	s.TopicComments = countBy("topic_comment", "user_id")
	s.Ratings = countBy("galgame_rating", "user_id")
	s.Resources = countBy("galgame_resource", "user_id")
	s.Websites = countBy("galgame_website", "user_id")
	s.Toolsets = countBy("galgame_toolset", "user_id")
	s.ToolsetResources = countBy("galgame_toolset_resource", "user_id")
	s.Polls = countBy("topic_poll", "user_id")
	s.Lotteries = countBy("topic_lottery", "user_id")
	s.Drafts = countBy("topic_draft", "user_id")
	s.Quizzes = countBy("galgame_quiz", "user_id")
	s.Collections = countBy("galgame_collection", "user_id")
	s.Todos = countBy("todo", "user_id")
	s.ChatMessages = countBy("chat_message", "sender_id")
	s.Messages = countBy("message", "sender_id") + countBy("message", "receiver_id")

	for _, t := range interactionTables {
		s.Interactions += countBy(t, "user_id")
	}

	s.Total = s.Topics + s.Replies + s.TopicComments +
		s.Ratings + s.Resources + s.Websites +
		s.Toolsets + s.ToolsetResources +
		s.Polls + s.Lotteries + s.Drafts +
		s.Quizzes + s.Collections + s.Todos +
		s.ChatMessages + s.Messages + s.Interactions
	return s, firstErr
}

var interactionTables = []string{
	"topic_like", "topic_dislike", "topic_favorite", "topic_upvote",
	"topic_reaction", "topic_reply_reaction",
	"topic_reply_like", "topic_reply_dislike", "topic_comment_like",
	"topic_poll_vote", "topic_lottery_entry",
	"galgame_like", "galgame_favorite",
	"galgame_collection_item", "galgame_collection_viewer",
	"galgame_post_like",
	"galgame_quiz_answer", "galgame_quiz_favorite",
	"galgame_rating_like", "galgame_resource_like",
	"galgame_website_like", "galgame_website_favorite",
	"galgame_toolset_practicality", "galgame_toolset_contributor",
}

// PurgeUserContent runs with the kungal.purge_* settings, so migration 120's
// trigger archives every row the transaction deletes, inserts or changes,
// cascades included.
func (r *PurgeRepository) PurgeUserContent(userID, operatorID int) (PurgeReceipt, error) {
	receipt := PurgeReceipt{PurgeID: uuid.New(), Archived: map[string]int64{}}

	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`SELECT set_config('kungal.purge_id', ?, true),
				set_config('kungal.purge_target_user_id', ?, true),
				set_config('kungal.purge_operator_id', ?, true)`,
			receipt.PurgeID.String(), fmt.Sprint(userID), fmt.Sprint(operatorID)).Error; err != nil {
			return err
		}

		var lotteryStates []string
		if err := tx.Raw(`SELECT status FROM topic_lottery
			WHERE user_id = ? OR topic_id IN (SELECT id FROM topic WHERE user_id = ?)
			FOR UPDATE`, userID, userID).Scan(&lotteryStates).Error; err != nil {
			return err
		}
		if slices.Contains(lotteryStates, topicModel.LotteryStatusDrawing) {
			return ErrLotteryDrawing
		}

		var affTopics, affGalgames, affChatRooms []int
		captures := []struct {
			dst *[]int
			sql string
		}{
			{&affTopics, `SELECT DISTINCT topic_id FROM topic_reply WHERE user_id = ?
				UNION SELECT DISTINCT topic_id FROM topic_comment WHERE user_id = ?`},
			{&affGalgames, `SELECT DISTINCT work_id FROM galgame_rating WHERE user_id = ?
				UNION SELECT DISTINCT work_id FROM galgame_resource WHERE user_id = ?`},
			{&affChatRooms, `SELECT DISTINCT chat_room_id FROM chat_message WHERE sender_id = ?
				UNION SELECT chat_room_id FROM chat_room_participant WHERE user_id = ?`},
		}
		for _, c := range captures {
			args := make([]any, countPlaceholders(c.sql))
			for i := range args {
				args[i] = userID
			}
			if err := tx.Raw(c.sql, args...).Scan(c.dst).Error; err != nil {
				return err
			}
		}

		// Interactions go before authored content: a collection item and a quiz
		// answer are the only handle on the parent whose cached counter has to
		// be recomputed, and deleting the parent first takes that handle away.
		for _, it := range interactionDeletes {
			captured := map[string][]int{}
			for _, rc := range it.recounts {
				if _, done := captured[rc.parentCol]; done {
					continue
				}
				var ids []int
				if err := tx.Raw(
					"SELECT DISTINCT "+rc.parentCol+" FROM "+it.table+" WHERE user_id = ?", userID,
				).Scan(&ids).Error; err != nil {
					return err
				}
				captured[rc.parentCol] = ids
			}
			if err := del(tx, "DELETE FROM "+it.table+" WHERE user_id = ?", userID); err != nil {
				return err
			}
			for _, rc := range it.recounts {
				if err := recount(tx, rc, it.table, captured[rc.parentCol]); err != nil {
					return err
				}
			}
		}

		// A directory entry is shared: deleting it took every other user's likes,
		// favorites and tag links with it. Nothing shows who listed a site, so the
		// row goes back to the column's default owner, as the imported rows have.
		if err := del(tx, "UPDATE galgame_website SET user_id = DEFAULT WHERE user_id = ?", userID); err != nil {
			return err
		}

		for _, q := range []string{
			"DELETE FROM topic WHERE user_id = ?",
			"DELETE FROM galgame_toolset WHERE user_id = ?",
			"DELETE FROM topic_poll WHERE user_id = ?",
			"DELETE FROM topic_lottery WHERE user_id = ?",
			"DELETE FROM topic_draft WHERE user_id = ?",
			"DELETE FROM galgame_collection WHERE user_id = ?",
			"DELETE FROM galgame_quiz WHERE user_id = ?",
			"DELETE FROM todo WHERE user_id = ?",
		} {
			if err := del(tx, q, userID); err != nil {
				return err
			}
		}

		for _, q := range []string{
			"DELETE FROM topic_reply WHERE user_id = ?",
			"DELETE FROM topic_comment WHERE user_id = ?",
			"DELETE FROM galgame_rating WHERE user_id = ?",
			"DELETE FROM galgame_resource WHERE user_id = ?",
			"DELETE FROM galgame_toolset_resource WHERE user_id = ?",
			"DELETE FROM toolset_upload WHERE user_id = ?",
			"DELETE FROM galgame_activity WHERE user_id = ?",
			"DELETE FROM galgame_contributor WHERE user_id = ?",
			"DELETE FROM user_permission_override WHERE user_id = ?",
		} {
			if err := del(tx, q, userID); err != nil {
				return err
			}
		}

		if err := del(tx, `UPDATE todo SET status = 0, claimed_user_id = NULL
			WHERE claimed_user_id = ? AND status = 1`, userID); err != nil {
			return err
		}

		for _, q := range []string{
			"DELETE FROM chat_message WHERE sender_id = ?",
			"DELETE FROM chat_message_reaction WHERE user_id = ?",
			"DELETE FROM chat_message_read_by WHERE user_id = ?",
			"DELETE FROM chat_room_admin WHERE user_id = ?",
			"DELETE FROM chat_room_participant WHERE user_id = ?",
		} {
			if err := del(tx, q, userID); err != nil {
				return err
			}
		}
		if len(affChatRooms) > 0 {
			// A private room is a two-party object, so purging one party ends it.
			// Deleting only the participant row left the counterparty looking at a
			// thread whose peer resolved to nothing (rooms 1356 and 1737 survived
			// user 7152 that way), and re-opening the DM would have collided with
			// the unique index on chat_room.name.
			if err := del(tx, "DELETE FROM chat_room WHERE id IN ? AND type = 'private'",
				affChatRooms); err != nil {
				return err
			}
			if err := del(tx, `UPDATE chat_room cr SET
					last_message_content = lm.content,
					last_message_time = lm.created,
					last_message_sender_id = lm.sender_id,
					last_message_sender_name = ''
				FROM (
					SELECT DISTINCT ON (chat_room_id) chat_room_id, content, created, sender_id
					FROM chat_message WHERE chat_room_id IN ?
					ORDER BY chat_room_id, created DESC
				) lm
				WHERE cr.id = lm.chat_room_id`, affChatRooms); err != nil {
				return err
			}
			if err := del(tx, `DELETE FROM chat_room WHERE id IN ?
				AND NOT EXISTS (SELECT 1 FROM chat_message m WHERE m.chat_room_id = chat_room.id)`,
				affChatRooms); err != nil {
				return err
			}
		}

		if err := del(tx, "DELETE FROM message WHERE sender_id = ? OR receiver_id = ?", userID, userID); err != nil {
			return err
		}
		for _, q := range []string{
			"DELETE FROM system_message WHERE user_id = ?",
			"DELETE FROM system_message_read_state WHERE user_id = ?",
		} {
			if err := del(tx, q, userID); err != nil {
				return err
			}
		}

		for _, q := range []string{
			"DELETE FROM user_follow WHERE follower_id = ? OR followed_id = ?",
			"DELETE FROM user_friend WHERE user_id = ? OR friend_id = ?",
		} {
			if err := del(tx, q, userID, userID); err != nil {
				return err
			}
		}

		for _, q := range []string{
			"DELETE FROM feed_activity WHERE user_id = ?",
			"DELETE FROM kungal_user_state WHERE user_id = ?",
		} {
			if err := del(tx, q, userID); err != nil {
				return err
			}
		}

		recounts := []struct {
			spec       recountSpec
			childTable string
			ids        []int
		}{
			{recountSpec{"topic", "reply_count", "topic_id", ""}, "topic_reply", affTopics},
			{recountSpec{"topic", "comment_count", "topic_id", ""}, "topic_comment", affTopics},
			{recountSpec{"galgame", "rating_count", "work_id", ""}, "galgame_rating", affGalgames},
			{recountSpec{"galgame", "resource_count", "work_id", ""}, "galgame_resource", affGalgames},
		}
		for _, rc := range recounts {
			if err := recount(tx, rc.spec, rc.childTable, rc.ids); err != nil {
				return err
			}
		}

		var archived []struct {
			TableName string
			Rows      int64
		}
		if err := tx.Raw(`SELECT table_name, count(*) AS rows FROM user_purge_archive
			WHERE purge_id = ? GROUP BY table_name`, receipt.PurgeID).Scan(&archived).Error; err != nil {
			return err
		}
		for _, a := range archived {
			receipt.Archived[a.TableName] = a.Rows
		}
		return nil
	})
	if err != nil {
		return PurgeReceipt{}, err
	}
	return receipt, nil
}

// Rows of one purge share created_at, the purging transaction's start, so a
// purge expires whole.
func (r *PurgeRepository) ExpireArchive(cutoff time.Time, batch int) (int64, error) {
	var total int64
	for {
		res := r.db.Exec(`DELETE FROM user_purge_archive WHERE id IN (
			SELECT id FROM user_purge_archive WHERE created_at < ? ORDER BY id LIMIT ?)`, cutoff, batch)
		if res.Error != nil {
			return total, res.Error
		}
		total += res.RowsAffected
		if res.RowsAffected < int64(batch) {
			return total, nil
		}
	}
}

// recountSpec rebuilds one cached counter on parentTable from the rows the
// child table still holds. agg defaults to COUNT(*).
type recountSpec struct {
	parentTable string
	countCol    string
	parentCol   string
	agg         string
}

type interactionDelete struct {
	table    string
	recounts []recountSpec
}

var interactionDeletes = []interactionDelete{
	// Purge used to recount like_count from topic_like and then delete
	// topic_reaction without recounting, so remaining reaction rows left the
	// cached count high.
	{table: "topic_like"},
	{table: "topic_dislike"},
	{table: "topic_favorite", recounts: []recountSpec{{"topic", "favorite_count", "topic_id", ""}}},
	{table: "topic_upvote", recounts: []recountSpec{{"topic", "upvote_count", "topic_id", ""}}},
	{table: "topic_reaction", recounts: []recountSpec{
		{"topic", "like_count", "topic_id", "COUNT(*) FILTER (WHERE reaction = 'like')"},
		{"topic", "dislike_count", "topic_id", "COUNT(*) FILTER (WHERE reaction = 'dislike')"},
	}},
	{table: "topic_reply_reaction", recounts: []recountSpec{
		{"topic_reply", "like_count", "topic_reply_id", "COUNT(*) FILTER (WHERE reaction = 'like')"},
		{"topic_reply", "dislike_count", "topic_reply_id", "COUNT(*) FILTER (WHERE reaction = 'dislike')"},
	}},
	{table: "topic_reply_like"},
	{table: "topic_reply_dislike"},
	{table: "topic_comment_like"},
	{table: "topic_poll_vote", recounts: []recountSpec{{"topic_poll_option", "vote_count", "option_id", ""}}},
	{table: "topic_lottery_entry", recounts: []recountSpec{{"topic_lottery", "entry_count", "lottery_id", ""}}},
	{table: "galgame_like", recounts: []recountSpec{{"galgame", "like_count", "work_id", ""}}},
	{table: "galgame_favorite"},
	{table: "galgame_collection_item", recounts: []recountSpec{
		{"galgame_collection", "item_count", "collection_id", ""},
		{"galgame", "favorite_count", "work_id", "COUNT(DISTINCT user_id)"},
	}},
	{table: "galgame_collection_viewer"},
	{table: "galgame_post_like"},
	{table: "galgame_quiz_answer", recounts: []recountSpec{
		{"galgame_quiz", "answer_count", "quiz_id", "COUNT(*) FILTER (WHERE role = 'answerer')"},
		{"galgame_quiz", "correct_count", "quiz_id", "COUNT(*) FILTER (WHERE is_correct)"},
		{"galgame_quiz", "quality_sum", "quiz_id", "COALESCE(SUM(quality_rating), 0)"},
		{"galgame_quiz", "quality_count", "quiz_id", "COUNT(*) FILTER (WHERE quality_rating > 0)"},
	}},
	{table: "galgame_quiz_favorite", recounts: []recountSpec{{"galgame_quiz", "favorite_count", "quiz_id", ""}}},
	{table: "galgame_rating_like", recounts: []recountSpec{{"galgame_rating", "like_count", "galgame_rating_id", ""}}},
	{table: "galgame_resource_like", recounts: []recountSpec{{"galgame_resource", "like_count", "galgame_resource_id", ""}}},
	{table: "galgame_website_like", recounts: []recountSpec{{"galgame_website", "like_count", "website_id", ""}}},
	{table: "galgame_website_favorite", recounts: []recountSpec{{"galgame_website", "favorite_count", "website_id", ""}}},
	{table: "galgame_toolset_practicality"},
	{table: "galgame_toolset_contributor"},
}

func del(tx *gorm.DB, query string, args ...any) error {
	return tx.Exec(query, args...).Error
}

func recount(tx *gorm.DB, spec recountSpec, childTable string, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	agg := spec.agg
	if agg == "" {
		agg = "COUNT(*)"
	}
	sql := fmt.Sprintf(
		"UPDATE %s SET %s = (SELECT %s FROM %s WHERE %s.%s = %s.id) WHERE %s.id IN ?",
		spec.parentTable, spec.countCol, agg, childTable, childTable, spec.parentCol,
		spec.parentTable, spec.parentTable,
	)
	return tx.Exec(sql, ids).Error
}

func countPlaceholders(sql string) int {
	n := 0
	for _, c := range sql {
		if c == '?' {
			n++
		}
	}
	return n
}
