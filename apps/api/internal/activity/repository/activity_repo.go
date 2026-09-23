package repository

import (
	"fmt"
	topicRepo "kun-galgame-api/internal/topic/repository"
	"strings"
	"time"

	topicModel "kun-galgame-api/internal/topic/model"
	"kun-galgame-api/pkg/miniapp"

	"gorm.io/gorm"
)

type ActivityRepository struct {
	db *gorm.DB
}

func NewActivityRepository(db *gorm.DB) *ActivityRepository {
	return &ActivityRepository{db: db}
}

var knownActivityTypes = map[string]struct{}{
	"TOPIC_CREATION": {}, "TOPIC_REPLY_CREATION": {}, "TOPIC_COMMENT_CREATION": {},
	"TOPIC_UPVOTE": {}, "MESSAGE_UPVOTE": {}, "MESSAGE_SOLUTION": {},
	"GALGAME_CREATION": {}, "GALGAME_COMMENT_CREATION": {}, "GALGAME_RESOURCE_CREATION": {},
	"GALGAME_EDIT": {}, "GALGAME_PR_CREATION": {}, "GALGAME_RATING_CREATION": {},
	"GALGAME_RATING_COMMENT_CREATION": {}, "GALGAME_WEBSITE_CREATION": {},
	"GALGAME_WEBSITE_COMMENT_CREATION": {}, "TOOLSET_CREATION": {},
	"TOOLSET_RESOURCE_CREATION": {}, "TOOLSET_COMMENT_CREATION": {},
	"TODO_CREATION": {}, "UPDATE_LOG_CREATION": {},
	"GALGAME_QUIZ_CREATION":             {},
	"GALGAME_RESOURCE_COMMENT_CREATION": {}, "GALGAME_QUIZ_COMMENT_CREATION": {},
}

type ActivityRow struct {
	TypeStr string    `gorm:"column:type_str"`
	ID      int       `gorm:"column:id"`
	Content string    `gorm:"column:content"`
	Link    string    `gorm:"column:link"`
	Created time.Time `gorm:"column:created"`
	UserID  int       `gorm:"column:user_id"`
	WorkID  int       `gorm:"column:work_id"`
}

func (r *ActivityRepository) IsKnownType(typeStr string) bool {
	_, ok := knownActivityTypes[typeStr]
	return ok
}

type TopicCardData struct {
	ID            int                    `gorm:"column:id"`
	Title         string                 `gorm:"column:title"`
	Excerpt       string                 `gorm:"column:excerpt"`
	CoverImages   topicModel.ImageTokens `gorm:"column:cover_images"`
	View          int                    `gorm:"column:view"`
	LikeCount     int                    `gorm:"column:like_count"`
	FavoriteCount int                    `gorm:"column:favorite_count"`
	ReplyCount    int                    `gorm:"column:reply_count"`
	CommentCount  int                    `gorm:"column:comment_count"`
	IsNSFW        bool                   `gorm:"column:is_nsfw"`
	UpvoteTime    *time.Time             `gorm:"column:upvote_time"`
	BestAnswerID  *int                   `gorm:"column:best_answer_id"`
	AuthorID      int                    `gorm:"column:user_id"`
	Edited        *time.Time             `gorm:"column:edited"`
}

func (r *ActivityRepository) FetchTopicActivityData(ids []int) (map[int]TopicCardData, error) {
	out := make(map[int]TopicCardData, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []TopicCardData
	if err := r.db.Table("topic").
		Select("id, title, user_id, SUBSTRING(content, 1, 300) AS excerpt, cover_images, view, like_count, favorite_count, reply_count, comment_count, is_nsfw, upvote_time, best_answer_id, edited").
		Where("id IN ?", ids).
		Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

func (r *ActivityRepository) FetchUpvoteTopics(upvoteIDs []int) (map[int]int, error) {
	out := map[int]int{}
	if len(upvoteIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		ID      int `gorm:"column:id"`
		TopicID int `gorm:"column:topic_id"`
	}
	if err := r.db.Table("topic_upvote").Select("id, topic_id").
		Where("id IN ?", upvoteIDs).Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.ID] = row.TopicID
	}
	return out, nil
}

type TopicReactionCountRow struct {
	TopicID  int    `gorm:"column:topic_id"`
	Reaction string `gorm:"column:reaction"`
	Count    int    `gorm:"column:cnt"`
	UserID   int    `gorm:"column:user_id"`
}

const feedReactionAvatarCap = 3

func (r *ActivityRepository) FetchTopicsReactions(ids []int) ([]TopicReactionCountRow, error) {
	out := []TopicReactionCountRow{}
	if len(ids) == 0 {
		return out, nil
	}
	err := r.db.Raw(`
		SELECT topic_id, reaction, user_id, cnt FROM (
			SELECT topic_id, reaction, user_id,
				COUNT(*) OVER (PARTITION BY topic_id, reaction) AS cnt,
				ROW_NUMBER() OVER (PARTITION BY topic_id, reaction ORDER BY id) AS rn
			FROM topic_reaction WHERE topic_id IN ?
		) t WHERE rn <= ? ORDER BY topic_id, reaction, rn`, ids, feedReactionAvatarCap).
		Scan(&out).Error
	return out, err
}

func (r *ActivityRepository) FetchTodoStatuses(ids []int) (map[int]int, error) {
	out := map[int]int{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		ID     int `gorm:"column:id"`
		Status int `gorm:"column:status"`
	}
	if err := r.db.Table("todo").Select("id, status").
		Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.ID] = row.Status
	}
	return out, nil
}

func (r *ActivityRepository) FetchUpdateLogVersions(ids []int) (map[int]string, error) {
	out := map[int]string{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		ID      int    `gorm:"column:id"`
		Version string `gorm:"column:version"`
	}
	if err := r.db.Table("update_log").Select("id, version").
		Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.ID] = row.Version
	}
	return out, nil
}

type TopicCommentContext struct {
	CommentID    int    `gorm:"column:comment_id"`
	TopicTitle   string `gorm:"column:topic_title"`
	ReplyFloor   int    `gorm:"column:reply_floor"`
	ReplyContent string `gorm:"column:reply_content"`
}

func (r *ActivityRepository) FetchTopicCommentContext(ids []int) (map[int]TopicCommentContext, error) {
	out := map[int]TopicCommentContext{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []TopicCommentContext
	if err := r.db.Table("topic_comment c").
		Select(`c.id AS comment_id, tp.title AS topic_title,
			r.floor AS reply_floor, r.content AS reply_content`).
		Joins("JOIN topic_reply r ON r.id = c.topic_reply_id").
		Joins("JOIN topic tp ON tp.id = r.topic_id").
		Where("c.id IN ?", ids).Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.CommentID] = row
	}
	return out, nil
}

func (r *ActivityRepository) fetchParentNames(childTable, parentTable, fk string, ids []int) (map[int]string, error) {
	out := map[int]string{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		ID   int    `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	if err := r.db.Table(childTable+" c").
		Select("c.id, p.name").
		Joins("JOIN "+parentTable+" p ON p.id = c."+fk).
		Where("c.id IN ?", ids).Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.ID] = row.Name
	}
	return out, nil
}

func (r *ActivityRepository) FetchToolsetResourceParents(ids []int) (map[int]string, error) {
	return r.fetchParentNames("galgame_toolset_resource", "galgame_toolset", "toolset_id", ids)
}

type GalgameResourceRow struct {
	ID        int    `gorm:"column:id"`
	Type      string `gorm:"column:type"`
	Language  string `gorm:"column:language"`
	Platform  string `gorm:"column:platform"`
	Size      string `gorm:"column:size"`
	Note      string `gorm:"column:note"`
	LikeCount int    `gorm:"column:like_count"`
}

func (r *ActivityRepository) FetchGalgameResourceDetails(ids []int) (map[int]GalgameResourceRow, error) {
	out := map[int]GalgameResourceRow{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []GalgameResourceRow
	if err := r.db.Table("galgame_resource").
		Select("id, type, language, platform, size, note, like_count").
		Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

type idNameRow struct {
	TopicID int    `gorm:"column:topic_id"`
	Name    string `gorm:"column:name"`
}

func collectIDNames(rows []idNameRow) map[int][]string {
	out := map[int][]string{}
	for _, row := range rows {
		out[row.TopicID] = append(out[row.TopicID], row.Name)
	}
	return out
}

func (r *ActivityRepository) FetchTopicSections(ids []int) (map[int][]string, error) {
	if len(ids) == 0 {
		return map[int][]string{}, nil
	}
	var rows []idNameRow
	if err := r.db.Table("topic_section_relation tsr").
		Select("tsr.topic_id, ts.name").
		Joins("JOIN topic_section ts ON ts.id = tsr.topic_section_id").
		Where("tsr.topic_id IN ?", ids).
		Scan(&rows).Error; err != nil {
		return map[int][]string{}, err
	}
	return collectIDNames(rows), nil
}

func (r *ActivityRepository) FetchTopicMiniApps(ids []int) map[int][]string {
	return miniapp.ByTopic(r.db, ids)
}

func (r *ActivityRepository) FetchReplyTopicTitles(replyIDs []int) (map[int]string, error) {
	out := map[int]string{}
	if len(replyIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		ID    int    `gorm:"column:id"`
		Title string `gorm:"column:title"`
	}
	if err := r.db.Raw(`
		SELECT r.id, t.title
		FROM topic_reply r JOIN topic t ON t.id = r.topic_id
		WHERE r.id IN ?`, replyIDs).Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.ID] = row.Title
	}
	return out, nil
}

func (r *ActivityRepository) FetchReplyFloors(replyIDs []int) (map[int]int, error) {
	out := map[int]int{}
	if len(replyIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		ID    int `gorm:"column:id"`
		Floor int `gorm:"column:floor"`
	}
	if err := r.db.Raw(`SELECT id, floor FROM topic_reply WHERE id IN ?`, replyIDs).
		Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.ID] = row.Floor
	}
	return out, nil
}

func (r *ActivityRepository) FetchTopicTitles(topicIDs []int) (map[int]string, error) {
	out := map[int]string{}
	if len(topicIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		ID    int    `gorm:"column:id"`
		Title string `gorm:"column:title"`
	}
	if err := r.db.Raw(`SELECT id, title FROM topic WHERE id IN ?`, topicIDs).
		Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.ID] = row.Title
	}
	return out, nil
}

type ReplyContent struct {
	Floor   int
	Content string
}

func (r *ActivityRepository) FetchReplyContents(replyIDs []int) (map[int]ReplyContent, error) {
	out := map[int]ReplyContent{}
	if len(replyIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		ID      int    `gorm:"column:id"`
		Floor   int    `gorm:"column:floor"`
		Content string `gorm:"column:content"`
	}
	if err := r.db.Raw(`
		SELECT id, floor, content
		FROM topic_reply
		WHERE id IN ?`, replyIDs).Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.ID] = ReplyContent{Floor: row.Floor, Content: row.Content}
	}
	return out, nil
}

type GalgameCounts struct {
	ResourceCount int
	LikeCount     int
	FavoriteCount int
	CreatorUserID *int
}

func (r *ActivityRepository) FetchGalgameCounts(workIDs []int) (map[int]GalgameCounts, error) {
	out := map[int]GalgameCounts{}
	if len(workIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		ID            int  `gorm:"column:id"`
		ResourceCount int  `gorm:"column:resource_count"`
		LikeCount     int  `gorm:"column:like_count"`
		FavoriteCount int  `gorm:"column:favorite_count"`
		CreatorUserID *int `gorm:"column:creator_user_id"`
	}
	if err := r.db.Raw(`
		SELECT id, resource_count, like_count, favorite_count, creator_user_id
		FROM galgame
		WHERE id IN ? AND published`, workIDs).Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.ID] = GalgameCounts{
			ResourceCount: row.ResourceCount,
			LikeCount:     row.LikeCount,
			FavoriteCount: row.FavoriteCount,
			CreatorUserID: row.CreatorUserID,
		}
	}
	return out, nil
}

type EditRevision struct {
	RevisionID     int
	RevisionNumber int
}

func (r *ActivityRepository) FetchEditRevisions(activityIDs []int) (map[int]EditRevision, error) {
	out := map[int]EditRevision{}
	if len(activityIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		ID         int  `gorm:"column:id"`
		RevisionID int  `gorm:"column:wiki_revision_id"`
		RevisionNo *int `gorm:"column:wiki_revision_number"`
	}
	if err := r.db.Raw(`
		SELECT id, COALESCE(wiki_revision_id, 0) AS wiki_revision_id, wiki_revision_number
		FROM galgame_activity
		WHERE id IN ?`, activityIDs).Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		num := 0
		if row.RevisionNo != nil {
			num = *row.RevisionNo
		}
		out[row.ID] = EditRevision{RevisionID: row.RevisionID, RevisionNumber: num}
	}
	return out, nil
}

type RatingActivity struct {
	Overall      int
	PlayStatus   string
	Recommend    string
	ShortSummary string
	SpoilerLevel string
	LikeCount    int
	AuthorID     int
}

func (r *ActivityRepository) FetchRatingActivityData(ratingIDs []int) (map[int]RatingActivity, error) {
	out := map[int]RatingActivity{}
	if len(ratingIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		ID           int    `gorm:"column:id"`
		Overall      int    `gorm:"column:overall"`
		PlayStatus   string `gorm:"column:play_status"`
		Recommend    string `gorm:"column:recommend"`
		ShortSummary string `gorm:"column:short_summary"`
		SpoilerLevel string `gorm:"column:spoiler_level"`
		LikeCount    int    `gorm:"column:like_count"`
		UserID       int    `gorm:"column:user_id"`
	}
	if err := r.db.Raw(`
		SELECT id, overall, play_status, recommend, short_summary, spoiler_level, like_count, user_id
		FROM galgame_rating
		WHERE id IN ?`, ratingIDs).Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		summary := row.ShortSummary
		if row.SpoilerLevel != "none" {
			summary = ""
		}
		out[row.ID] = RatingActivity{
			Overall:      row.Overall,
			PlayStatus:   row.PlayStatus,
			Recommend:    row.Recommend,
			ShortSummary: summary,
			SpoilerLevel: row.SpoilerLevel,
			LikeCount:    row.LikeCount,
			AuthorID:     row.UserID,
		}
	}
	return out, nil
}

type QuizActivity struct {
	Category      string
	Type          string
	Difficulty    int
	AnswerCount   int
	CorrectCount  int
	FavoriteCount int
	Description   string
}

func (r *ActivityRepository) FetchQuizActivityData(quizIDs []int) (map[int]QuizActivity, error) {
	out := map[int]QuizActivity{}
	if len(quizIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		ID            int    `gorm:"column:id"`
		Category      string `gorm:"column:category"`
		Type          string `gorm:"column:type"`
		Difficulty    int    `gorm:"column:difficulty"`
		AnswerCount   int    `gorm:"column:answer_count"`
		CorrectCount  int    `gorm:"column:correct_count"`
		FavoriteCount int    `gorm:"column:favorite_count"`
		Description   string `gorm:"column:description"`
	}
	if err := r.db.Raw(`
		SELECT id, category, type, difficulty, answer_count, correct_count, favorite_count,
			CASE WHEN CHAR_LENGTH(description) > 200
				THEN LEFT(description, 200) || '…'
				ELSE description END AS description
		FROM galgame_quiz
		WHERE id IN ?`, quizIDs).Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.ID] = QuizActivity{
			Category:      row.Category,
			Type:          row.Type,
			Difficulty:    row.Difficulty,
			AnswerCount:   row.AnswerCount,
			CorrectCount:  row.CorrectCount,
			FavoriteCount: row.FavoriteCount,
			Description:   row.Description,
		}
	}
	return out, nil
}

type TopReplyRow struct {
	TopicID   int    `gorm:"column:topic_id"`
	ID        int    `gorm:"column:id"`
	Floor     int    `gorm:"column:floor"`
	Content   string `gorm:"column:content"`
	LikeCount int    `gorm:"column:like_count"`
	UserID    int    `gorm:"column:user_id"`
}

func (r *ActivityRepository) FetchTopicTopReply(ids []int) (map[int]TopReplyRow, error) {
	out := map[int]TopReplyRow{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []TopReplyRow
	if err := r.db.Raw(`
		SELECT DISTINCT ON (topic_id) topic_id, id, floor,
			SUBSTRING(content, 1, 200) AS content, like_count, user_id
		FROM topic_reply
		WHERE topic_id IN ? AND like_count > 0
		ORDER BY topic_id, like_count DESC, id DESC`, ids).
		Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.TopicID] = row
	}
	return out, nil
}

type BestAnswerRow struct {
	TopicID   int    `gorm:"column:topic_id"`
	ReplyID   int    `gorm:"column:reply_id"`
	Floor     int    `gorm:"column:floor"`
	Content   string `gorm:"column:content"`
	LikeCount int    `gorm:"column:like_count"`
	UserID    int    `gorm:"column:user_id"`
}

func (r *ActivityRepository) FetchTopicBestAnswers(ids []int) (map[int]BestAnswerRow, error) {
	out := map[int]BestAnswerRow{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []BestAnswerRow
	if err := r.db.Raw(`
		SELECT t.id AS topic_id, r.id AS reply_id, r.floor,
			SUBSTRING(r.content, 1, 200) AS content, r.like_count, r.user_id
		FROM topic t
		JOIN topic_reply r ON r.id = t.best_answer_id
		WHERE t.id IN ? AND t.best_answer_id IS NOT NULL`, ids).
		Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.TopicID] = row
	}
	return out, nil
}

type TopicUpvoteRow struct {
	TopicID     int       `gorm:"column:topic_id"`
	ID          int       `gorm:"column:id"`
	UserID      int       `gorm:"column:user_id"`
	Description string    `gorm:"column:description"`
	Created     time.Time `gorm:"column:created"`
}

func (r *ActivityRepository) FetchTopicUpvotesBatch(ids []int) (map[int][]TopicUpvoteRow, error) {
	out := map[int][]TopicUpvoteRow{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []TopicUpvoteRow
	if err := r.db.Table("topic_upvote").
		Select("topic_id, id, user_id, description, created").
		Where("topic_id IN ?", ids).
		Order("topic_id, created DESC, id DESC").
		Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.TopicID] = append(out[row.TopicID], row)
	}
	return out, nil
}

type Cursor struct {
	Created time.Time
	TypeStr string
	ID      int
}

func (r *ActivityRepository) FetchFeed(types []string, limit int, cur *Cursor, isSFW, showNoResource bool, sectionMode string) ([]ActivityRow, error) {
	conds := make([]string, 0, 6)
	args := make([]any, 0, 6)
	if len(types) > 0 {
		conds = append(conds, "fa.type IN ?")
		args = append(args, types)
	}
	if isSFW {
		conds = append(conds, "NOT fa.is_nsfw")
	}
	if !showNoResource {
		conds = append(conds, "(fa.type <> 'GALGAME_CREATION' OR EXISTS (SELECT 1 FROM galgame_resource r WHERE r.work_id = fa.work_id))")
	}
	const inHelp = "EXISTS (SELECT 1 FROM topic_section_relation tsr " +
		"JOIN topic_section ts ON ts.id = tsr.topic_section_id " +
		"WHERE tsr.topic_id = fa.source_id AND ts.name IN ('g-seeking','g-other','t-help'))"
	switch sectionMode {
	case "help":
		conds = append(conds, "(fa.type <> 'TOPIC_CREATION' OR "+inHelp+")")
	case "normal":
		conds = append(conds, "(fa.type <> 'TOPIC_CREATION' OR NOT "+inHelp+")")
	}
	if cur != nil {
		conds = append(conds, "(fa.created, fa.type, fa.source_id) < (?, ?, ?)")
		args = append(args, cur.Created, cur.TypeStr, cur.ID)
	}

	sql := "SELECT fa.type AS type_str, fa.source_id AS id, fa.content, fa.link, fa.created, fa.user_id, fa.work_id FROM feed_activity fa"
	if len(conds) > 0 {
		sql += " WHERE " + strings.Join(conds, " AND ")
	}
	sql += fmt.Sprintf(" ORDER BY fa.created DESC, fa.type DESC, fa.source_id DESC LIMIT %d", limit)

	var rows []ActivityRow
	if err := r.db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *ActivityRepository) FetchTopicFeed(limit int, cur *Cursor, isSFW bool, sectionMode string) ([]ActivityRow, error) {
	conds := []string{"t.status != 1", topicRepo.SharedListPredicate("t", false)}
	args := []any{}
	if isSFW {
		conds = append(conds, "NOT t.is_nsfw")
	}
	const inHelp = "EXISTS (SELECT 1 FROM topic_section_relation tsr " +
		"JOIN topic_section ts ON ts.id = tsr.topic_section_id " +
		"WHERE tsr.topic_id = t.id AND ts.name IN ('g-seeking','g-other','t-help'))"
	switch sectionMode {
	case "help":
		conds = append(conds, inHelp)
	case "normal":
		conds = append(conds, "NOT "+inHelp)
	}
	if cur != nil {
		conds = append(conds, "(t.status_update_time, t.id) < (?, ?)")
		args = append(args, cur.Created, cur.ID)
	}
	sql := "SELECT 'TOPIC_CREATION' AS type_str, t.id, t.title AS content, " +
		"'/topic/' || t.id::text AS link, t.status_update_time AS created, " +
		"t.user_id, 0 AS work_id FROM topic t WHERE " + strings.Join(conds, " AND ") +
		fmt.Sprintf(" ORDER BY t.status_update_time DESC, t.id DESC LIMIT %d", limit)
	var rows []ActivityRow
	if err := r.db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

type LatestActivityRow struct {
	TopicID int       `gorm:"column:topic_id"`
	Kind    string    `gorm:"column:kind"`
	ID      int       `gorm:"column:id"`
	Floor   int       `gorm:"column:floor"`
	Content string    `gorm:"column:content"`
	UserID  int       `gorm:"column:user_id"`
	Created time.Time `gorm:"column:created"`
}

func (r *ActivityRepository) FetchTopicLatestActivity(ids []int) (map[int]LatestActivityRow, error) {
	out := map[int]LatestActivityRow{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []LatestActivityRow
	if err := r.db.Raw(`
		SELECT DISTINCT ON (topic_id) topic_id, kind, id, floor, content, user_id, created FROM (
			SELECT topic_id, 'reply' AS kind, id, floor, SUBSTRING(content, 1, 200) AS content, user_id, created
				FROM topic_reply WHERE topic_id IN ? AND status = 0
			UNION ALL
			SELECT topic_id, 'comment' AS kind, id, 0 AS floor, SUBSTRING(content, 1, 200) AS content, user_id, created
				FROM topic_comment WHERE topic_id IN ? AND status = 0
		) x
		ORDER BY topic_id, created DESC, id DESC`, ids, ids).
		Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out[row.TopicID] = row
	}
	return out, nil
}
