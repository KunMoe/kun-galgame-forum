package repository

import (
	"kun-galgame-api/pkg/miniapp"

	"gorm.io/gorm"
)

type ActivityRepository struct {
	db *gorm.DB
}

func NewActivityRepository(db *gorm.DB) *ActivityRepository {
	return &ActivityRepository{db: db}
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
