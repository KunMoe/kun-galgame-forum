package repository

import (
	"fmt"
	"strings"
	"time"

	topicRepo "kun-galgame-api/internal/topic/repository"
)

type FeedRow struct {
	RowID    int64     `gorm:"column:row_id"`
	TypeStr  string    `gorm:"column:type_str"`
	SourceID int       `gorm:"column:source_id"`
	UserID   int       `gorm:"column:user_id"`
	WorkID   int       `gorm:"column:work_id"`
	Content  string    `gorm:"column:content"`
	Link     string    `gorm:"column:link"`
	Created  time.Time `gorm:"column:created"`
	Bumped   time.Time `gorm:"column:bumped"`
}

type FeedQuery struct {
	Types              []string
	TopicSections      string
	IncludeNSFW        bool
	IncludeUnresourced bool
	Limit              int
	After              *FeedPos
}

type FeedPos struct {
	Created  time.Time
	TypeStr  string
	SourceID int
}

type TopicFeedQuery struct {
	TopicSections string
	IncludeNSFW   bool
	Limit         int
	After         *TopicFeedPos
}

type TopicFeedPos struct {
	Bumped time.Time
	ID     int
}

const helpSections = "EXISTS (SELECT 1 FROM topic_section_relation tsr " +
	"JOIN topic_section ts ON ts.id = tsr.topic_section_id " +
	"WHERE tsr.topic_id = %s AND ts.name IN ('g-seeking','g-other','t-help'))"

func sectionCond(mode, topicCol string, onlyTopicCreation bool) string {
	in := fmt.Sprintf(helpSections, topicCol)
	var cond string
	switch mode {
	case "help":
		cond = in
	case "normal":
		cond = "NOT " + in
	default:
		return ""
	}
	if onlyTopicCreation {
		return "(fa.type <> 'TOPIC_CREATION' OR " + cond + ")"
	}
	return cond
}

func (r *ActivityRepository) FeedPage(q FeedQuery) ([]FeedRow, error) {
	conds := []string{"fa.type IN ?"}
	args := []any{q.Types}
	if !q.IncludeNSFW {
		conds = append(conds, "NOT fa.is_nsfw")
	}
	if !q.IncludeUnresourced {
		conds = append(conds, "(fa.type <> 'GALGAME_CREATION' OR EXISTS (SELECT 1 FROM galgame_resource gr WHERE gr.work_id = fa.work_id))")
	}
	if c := sectionCond(q.TopicSections, "fa.source_id", true); c != "" {
		conds = append(conds, c)
	}
	if q.After != nil {
		conds = append(conds, "(fa.created, fa.type, fa.source_id) < (?, ?, ?)")
		args = append(args, q.After.Created, q.After.TypeStr, q.After.SourceID)
	}
	sql := "SELECT fa.id AS row_id, fa.type AS type_str, fa.source_id, fa.user_id, fa.work_id, fa.content, fa.link, fa.created " +
		"FROM feed_activity fa WHERE " + strings.Join(conds, " AND ") +
		fmt.Sprintf(" ORDER BY fa.created DESC, fa.type DESC, fa.source_id DESC LIMIT %d", q.Limit)
	var rows []FeedRow
	err := r.db.Raw(sql, args...).Scan(&rows).Error
	return rows, err
}

func (r *ActivityRepository) TopicFeedPage(q TopicFeedQuery) ([]FeedRow, error) {
	conds := []string{"t.status = 0", topicRepo.SharedListPredicate("t", false)}
	args := []any{}
	if !q.IncludeNSFW {
		conds = append(conds, "NOT t.is_nsfw")
	}
	if c := sectionCond(q.TopicSections, "t.id", false); c != "" {
		conds = append(conds, c)
	}
	if q.After != nil {
		conds = append(conds, "(t.status_update_time, t.id) < (?, ?)")
		args = append(args, q.After.Bumped, q.After.ID)
	}
	sql := "SELECT fa.id AS row_id, fa.type AS type_str, fa.source_id, fa.user_id, fa.work_id, fa.content, fa.link, fa.created, " +
		"t.status_update_time AS bumped FROM topic t " +
		"JOIN feed_activity fa ON fa.type = 'TOPIC_CREATION' AND fa.source_id = t.id WHERE " + strings.Join(conds, " AND ") +
		fmt.Sprintf(" ORDER BY t.status_update_time DESC, t.id DESC LIMIT %d", q.Limit)
	var rows []FeedRow
	err := r.db.Raw(sql, args...).Scan(&rows).Error
	return rows, err
}

type TopicCardRow struct {
	topicRepo.TopicKeysetRow
	Excerpt string     `gorm:"column:excerpt"`
	Edited  *time.Time `gorm:"column:edited"`
}

func (r *ActivityRepository) TopicCards(ids []int) (map[int]TopicCardRow, error) {
	out := map[int]TopicCardRow{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []TopicCardRow
	if err := r.db.Raw(`
		SELECT id, title, view, status, is_nsfw, like_count, reply_count, comment_count, best_answer_id,
			status_update_time, created, upvote_time, cover_images, user_id, category, favorite_count, upvote_count,
			SUBSTRING(content, 1, 300) AS excerpt, edited
		FROM topic WHERE id IN ?`, ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

type ReplyExcerptRow struct {
	TopicID   int       `gorm:"column:topic_id"`
	ID        int       `gorm:"column:id"`
	Floor     int       `gorm:"column:floor"`
	Content   string    `gorm:"column:content"`
	LikeCount int       `gorm:"column:like_count"`
	UserID    int       `gorm:"column:user_id"`
	Created   time.Time `gorm:"column:created"`
}

func byTopic(rows []ReplyExcerptRow) map[int]ReplyExcerptRow {
	out := make(map[int]ReplyExcerptRow, len(rows))
	for _, row := range rows {
		out[row.TopicID] = row
	}
	return out
}

func (r *ActivityRepository) TopReplies(ids []int) (map[int]ReplyExcerptRow, error) {
	if len(ids) == 0 {
		return map[int]ReplyExcerptRow{}, nil
	}
	var rows []ReplyExcerptRow
	err := r.db.Raw(`
		SELECT DISTINCT ON (topic_id) topic_id, id, floor, SUBSTRING(content, 1, 200) AS content, like_count, user_id, created
		FROM topic_reply
		WHERE topic_id IN ? AND status = 0 AND like_count > 0
		ORDER BY topic_id, like_count DESC, id DESC`, ids).Scan(&rows).Error
	return byTopic(rows), err
}

func (r *ActivityRepository) BestAnswers(ids []int) (map[int]ReplyExcerptRow, error) {
	if len(ids) == 0 {
		return map[int]ReplyExcerptRow{}, nil
	}
	var rows []ReplyExcerptRow
	err := r.db.Raw(`
		SELECT t.id AS topic_id, rp.id, rp.floor, SUBSTRING(rp.content, 1, 200) AS content, rp.like_count, rp.user_id, rp.created
		FROM topic t JOIN topic_reply rp ON rp.id = t.best_answer_id
		WHERE t.id IN ? AND rp.status = 0`, ids).Scan(&rows).Error
	return byTopic(rows), err
}

type LatestRow struct {
	TopicID int       `gorm:"column:topic_id"`
	Kind    string    `gorm:"column:kind"`
	ID      int       `gorm:"column:id"`
	Floor   int       `gorm:"column:floor"`
	Content string    `gorm:"column:content"`
	LikeCnt int       `gorm:"column:like_count"`
	UserID  int       `gorm:"column:user_id"`
	Created time.Time `gorm:"column:created"`
}

func (r *ActivityRepository) LatestReplyOrComment(ids []int) (map[int]LatestRow, error) {
	out := map[int]LatestRow{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []LatestRow
	if err := r.db.Raw(`
		SELECT DISTINCT ON (topic_id) topic_id, kind, id, floor, content, like_count, user_id, created FROM (
			SELECT topic_id, 'reply' AS kind, id, floor, SUBSTRING(content, 1, 200) AS content, like_count, user_id, created
				FROM topic_reply WHERE topic_id IN ? AND status = 0
			UNION ALL
			SELECT topic_id, 'comment' AS kind, id, 0 AS floor, SUBSTRING(content, 1, 200) AS content, 0 AS like_count, user_id, created
				FROM topic_comment WHERE topic_id IN ? AND status = 0
		) x
		ORDER BY topic_id, created DESC, id DESC`, ids, ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.TopicID] = row
	}
	return out, nil
}

type UpvoteRow struct {
	TopicID     int       `gorm:"column:topic_id"`
	UserID      int       `gorm:"column:user_id"`
	Description string    `gorm:"column:description"`
	Created     time.Time `gorm:"column:created"`
}

func (r *ActivityRepository) LatestUpvotes(ids []int) (map[int]UpvoteRow, error) {
	out := map[int]UpvoteRow{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []UpvoteRow
	if err := r.db.Raw(`
		SELECT DISTINCT ON (topic_id) topic_id, user_id, COALESCE(description, '') AS description, created
		FROM topic_upvote WHERE topic_id IN ?
		ORDER BY topic_id, created DESC, id DESC`, ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.TopicID] = row
	}
	return out, nil
}

type ReactionRow struct {
	TopicID  int    `gorm:"column:topic_id"`
	Reaction string `gorm:"column:reaction"`
	UserID   int    `gorm:"column:user_id"`
	Count    int    `gorm:"column:cnt"`
}

func (r *ActivityRepository) ReactionSamples(ids []int) ([]ReactionRow, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []ReactionRow
	err := r.db.Raw(`
		SELECT topic_id, reaction, user_id, cnt FROM (
			SELECT topic_id, reaction, user_id, id,
				COUNT(*) OVER (PARTITION BY topic_id, reaction) AS cnt,
				MIN(id) OVER (PARTITION BY topic_id, reaction) AS first_id,
				ROW_NUMBER() OVER (PARTITION BY topic_id, reaction ORDER BY id) AS rn
			FROM topic_reaction WHERE topic_id IN ?
		) t WHERE rn <= 8 ORDER BY topic_id, first_id, rn`, ids).Scan(&rows).Error
	return rows, err
}

type ReplyContext struct {
	ID         int    `gorm:"column:id"`
	TopicID    int    `gorm:"column:topic_id"`
	TopicTitle string `gorm:"column:topic_title"`
	Floor      int    `gorm:"column:floor"`
}

func (r *ActivityRepository) ReplyContexts(ids []int) (map[int]ReplyContext, error) {
	out := map[int]ReplyContext{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []ReplyContext
	if err := r.db.Raw(`
		SELECT rp.id, rp.topic_id, t.title AS topic_title, rp.floor
		FROM topic_reply rp JOIN topic t ON t.id = rp.topic_id
		WHERE rp.id IN ?`, ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

type QuotedRow struct {
	ID      int    `gorm:"column:id"`
	Floor   int    `gorm:"column:floor"`
	Content string `gorm:"column:content"`
	UserID  int    `gorm:"column:user_id"`
}

func (r *ActivityRepository) VisibleReplies(ids []int) (map[int]QuotedRow, error) {
	out := map[int]QuotedRow{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []QuotedRow
	if err := r.db.Raw(`
		SELECT id, floor, SUBSTRING(content, 1, 200) AS content, user_id
		FROM topic_reply WHERE id IN ? AND status = 0`, ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

type CommentContext struct {
	ID         int    `gorm:"column:id"`
	TopicID    int    `gorm:"column:topic_id"`
	TopicTitle string `gorm:"column:topic_title"`
	ReplyID    int    `gorm:"column:reply_id"`
}

func (r *ActivityRepository) CommentContexts(ids []int) (map[int]CommentContext, error) {
	out := map[int]CommentContext{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []CommentContext
	if err := r.db.Raw(`
		SELECT c.id, c.topic_id, t.title AS topic_title, c.topic_reply_id AS reply_id
		FROM topic_comment c JOIN topic t ON t.id = c.topic_id
		WHERE c.id IN ?`, ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

type BestAnswerContext struct {
	TopicID    int    `gorm:"column:topic_id"`
	TopicTitle string `gorm:"column:topic_title"`
	ReplyID    *int   `gorm:"column:reply_id"`
	Floor      *int   `gorm:"column:floor"`
}

func (r *ActivityRepository) BestAnswerContexts(topicIDs []int) (map[int]BestAnswerContext, error) {
	out := map[int]BestAnswerContext{}
	if len(topicIDs) == 0 {
		return out, nil
	}
	var rows []BestAnswerContext
	if err := r.db.Raw(`
		SELECT t.id AS topic_id, t.title AS topic_title, rp.id AS reply_id, rp.floor
		FROM topic t LEFT JOIN topic_reply rp ON rp.id = t.best_answer_id AND rp.status = 0
		WHERE t.id IN ?`, topicIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.TopicID] = row
	}
	return out, nil
}

type WorkRow struct {
	ID            int  `gorm:"column:id"`
	ResourceCount int  `gorm:"column:resource_count"`
	LikeCount     int  `gorm:"column:like_count"`
	FavoriteCount int  `gorm:"column:favorite_count"`
	CreatorUserID *int `gorm:"column:creator_user_id"`
}

func (r *ActivityRepository) PublishedWorks(workIDs []int) (map[int]WorkRow, error) {
	out := map[int]WorkRow{}
	if len(workIDs) == 0 {
		return out, nil
	}
	var rows []WorkRow
	if err := r.db.Raw(`
		SELECT id, resource_count, like_count, favorite_count, creator_user_id
		FROM galgame WHERE id IN ? AND published`, workIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

type ToolsetParent struct {
	ID        int    `gorm:"column:id"`
	ToolsetID int    `gorm:"column:toolset_id"`
	Name      string `gorm:"column:name"`
}

func (r *ActivityRepository) ToolsetParents(resourceIDs []int) (map[int]ToolsetParent, error) {
	out := map[int]ToolsetParent{}
	if len(resourceIDs) == 0 {
		return out, nil
	}
	var rows []ToolsetParent
	if err := r.db.Raw(`
		SELECT c.id, p.id AS toolset_id, p.name
		FROM galgame_toolset_resource c JOIN galgame_toolset p ON p.id = c.toolset_id
		WHERE c.id IN ?`, resourceIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}
