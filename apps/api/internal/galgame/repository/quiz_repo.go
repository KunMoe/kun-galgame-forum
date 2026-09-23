package repository

import (
	"time"

	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/infrastructure/viewstats"

	"gorm.io/gorm"
)

type QuizRepository struct {
	db *gorm.DB
}

func NewQuizRepository(db *gorm.DB) *QuizRepository {
	return &QuizRepository{db: db}
}

func (r *QuizRepository) DB() *gorm.DB { return r.db }

const quizCardColumns = `q.id, q.user_id, q.category, q.spoiler_level,
	q.type, q.difficulty, q.question, q.view, q.answer_count, q.correct_count,
	q.favorite_count, q.quality_sum, q.quality_count, q.comment_count,
	q.status_update_time,
	q.created, q.updated`

func (r *QuizRepository) FindByID(id int) (*model.GalgameQuiz, bool) {
	var q model.GalgameQuiz
	if err := r.db.First(&q, id).Error; err != nil {
		return nil, false
	}
	return &q, true
}

func (r *QuizRepository) FindAnswer(quizID, userID int) (*model.GalgameQuizAnswer, bool) {
	var a model.GalgameQuizAnswer
	err := r.db.Where("quiz_id = ? AND user_id = ?", quizID, userID).First(&a).Error
	if err != nil {
		return nil, false
	}
	return &a, true
}

func (r *QuizRepository) ListPaginated(f model.QuizFilter) ([]model.GalgameQuizRow, int64) {
	query := r.db.Table("galgame_quiz q")
	if f.Category != "" && f.Category != "all" {
		query = query.Where("q.category = ?", f.Category)
	}
	if f.Type != "" && f.Type != "all" {
		query = query.Where("q.type = ?", f.Type)
	}
	if f.Difficulty > 0 {
		query = query.Where("q.difficulty = ?", f.Difficulty)
	}
	if f.WorkID > 0 {
		query = query.Where(
			"EXISTS (SELECT 1 FROM galgame_quiz_galgame gg WHERE gg.quiz_id = q.id AND gg.work_id = ?)",
			f.WorkID,
		)
	}
	if f.UserID > 0 {
		query = query.Where("q.user_id = ?", f.UserID)
	}

	var total int64
	query.Count(&total)

	orderCol := "q.created"
	switch f.SortField {
	case "view":
		orderCol = "q.view"
	case "view_1d":
		orderCol = "COALESCE((SELECT SUM(d.count) FROM galgame_quiz_view_daily d " +
			"WHERE d.entity_id = q.id AND d.day = CURRENT_DATE), 0)"
	case "view_7d":
		orderCol = "q.view_7d"
	case "view_30d":
		orderCol = "q.view_30d"
	case "difficulty":
		orderCol = "q.difficulty"
	case "answer_count":
		orderCol = "q.answer_count"
	case "update_time":
		orderCol = "q.status_update_time"
	}

	var rows []model.GalgameQuizRow
	query.Select(quizCardColumns).
		Order(orderCol + " " + f.SortOrder).
		Offset((f.Page - 1) * f.Limit).Limit(f.Limit).
		Scan(&rows)
	return rows, total
}

func (r *QuizRepository) ListAnsweredByUser(userID, page, limit int) ([]model.GalgameQuizRow, int64) {
	base := r.db.Table("galgame_quiz q").
		Joins("JOIN galgame_quiz_answer a ON a.quiz_id = q.id").
		Where("a.user_id = ? AND a.role = ?", userID, "answerer")

	var total int64
	base.Count(&total)

	var rows []model.GalgameQuizRow
	base.Select(quizCardColumns).
		Order("a.created DESC").
		Offset((page - 1) * limit).Limit(limit).
		Scan(&rows)
	return rows, total
}

func (r *QuizRepository) IncrementView(id int) {
	go func() {
		r.db.Table("galgame_quiz").Where("id = ?", id).
			Update("view", gorm.Expr("view + 1"))
		_ = viewstats.BumpDaily(r.db, viewstats.QuizDaily, id)
	}()
}

func (r *QuizRepository) Create(tx *gorm.DB, q *model.GalgameQuiz) error {
	return tx.Create(q).Error
}

func (r *QuizRepository) CreateAnswer(tx *gorm.DB, a *model.GalgameQuizAnswer) error {
	return tx.Create(a).Error
}

func (r *QuizRepository) BumpAnswerStats(tx *gorm.DB, quizID int, correct bool) error {
	fields := map[string]any{
		"answer_count": gorm.Expr("answer_count + 1"),
		"status_update_time": gorm.Expr(
			"CASE WHEN created > ? THEN now() ELSE status_update_time END",
			model.QuizBumpCutoff(time.Now())),
	}
	if correct {
		fields["correct_count"] = gorm.Expr("correct_count + 1")
	}
	return tx.Table("galgame_quiz").Where("id = ?", quizID).Updates(fields).Error
}

func (r *QuizRepository) AdjustCorrectCount(tx *gorm.DB, quizID, delta int) error {
	return tx.Table("galgame_quiz").Where("id = ?", quizID).
		UpdateColumn("correct_count", gorm.Expr("correct_count + ?", delta)).Error
}

func (r *QuizRepository) UpdateAnswerFields(tx *gorm.DB, answerID int, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	return tx.Table("galgame_quiz_answer").Where("id = ?", answerID).Updates(fields).Error
}

func (r *QuizRepository) AdjustQuality(tx *gorm.DB, quizID, sumDelta, countDelta int) error {
	return tx.Table("galgame_quiz").Where("id = ?", quizID).Updates(map[string]any{
		"quality_sum":   gorm.Expr("quality_sum + ?", sumDelta),
		"quality_count": gorm.Expr("quality_count + ?", countDelta),
	}).Error
}

func (r *QuizRepository) SetAnswerQuality(tx *gorm.DB, answerID, rating int) error {
	return tx.Table("galgame_quiz_answer").Where("id = ?", answerID).
		Update("quality_rating", rating).Error
}

func (r *QuizRepository) DeleteByID(tx *gorm.DB, id int) error {
	return tx.Where("id = ?", id).Delete(&model.GalgameQuiz{}).Error
}

func (r *QuizRepository) UpdateQuizFields(tx *gorm.DB, id int, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	return tx.Table("galgame_quiz").Where("id = ?", id).Updates(fields).Error
}

func (r *QuizRepository) FindQuizFavorite(quizID, userID int) bool {
	if userID == 0 {
		return false
	}
	var count int64
	r.db.Table("galgame_quiz_favorite").
		Where("quiz_id = ? AND user_id = ?", quizID, userID).
		Count(&count)
	return count > 0
}

func (r *QuizRepository) FindFavoritedQuizIDs(userID int) []int {
	ids := []int{}
	if userID == 0 {
		return ids
	}
	r.db.Table("galgame_quiz_favorite").
		Where("user_id = ?", userID).
		Pluck("quiz_id", &ids)
	return ids
}

func (r *QuizRepository) CreateQuizFavorite(tx *gorm.DB, quizID, userID int) error {
	return tx.Create(&model.GalgameQuizFavorite{QuizID: quizID, UserID: userID}).Error
}

func (r *QuizRepository) DeleteQuizFavorite(tx *gorm.DB, quizID, userID int) error {
	return tx.Where("quiz_id = ? AND user_id = ?", quizID, userID).
		Delete(&model.GalgameQuizFavorite{}).Error
}

func (r *QuizRepository) AdjustQuizFavoriteCount(tx *gorm.DB, quizID, delta int) error {
	return tx.Model(&model.GalgameQuiz{}).Where("id = ?", quizID).
		UpdateColumn("favorite_count", gorm.Expr("favorite_count + ?", delta)).Error
}

func (r *QuizRepository) FindQuizWorkIDs(quizID int) []int {
	var ids []int
	r.db.Table("galgame_quiz_galgame").
		Where("quiz_id = ?", quizID).
		Order("work_id").
		Pluck("work_id", &ids)
	return ids
}

func (r *QuizRepository) SetQuizGalgames(tx *gorm.DB, quizID int, workIDs []int) error {
	if err := tx.Where("quiz_id = ?", quizID).
		Delete(&model.GalgameQuizGalgame{}).Error; err != nil {
		return err
	}
	seen := make(map[int]struct{}, len(workIDs))
	rows := make([]model.GalgameQuizGalgame, 0, len(workIDs))
	for _, id := range workIDs {
		if id <= 0 {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		rows = append(rows, model.GalgameQuizGalgame{QuizID: quizID, WorkID: id})
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

type QuizViewerAnswer struct {
	QuizID    int    `gorm:"column:quiz_id"`
	Role      string `gorm:"column:role"`
	IsCorrect *bool  `gorm:"column:is_correct"`
}

func (r *QuizRepository) FindViewerAnswers(quizIDs []int, userID int) map[int]QuizViewerAnswer {
	if len(quizIDs) == 0 || userID == 0 {
		return map[int]QuizViewerAnswer{}
	}
	var rows []QuizViewerAnswer
	r.db.Table("galgame_quiz_answer").
		Select("quiz_id, role, is_correct").
		Where("user_id = ? AND quiz_id IN ?", userID, quizIDs).
		Scan(&rows)
	m := make(map[int]QuizViewerAnswer, len(rows))
	for _, row := range rows {
		m[row.QuizID] = row
	}
	return m
}

func (r *QuizRepository) FindQuizAnswerers(quizID, limit int) []model.GalgameQuizAnswererRow {
	var rows []model.GalgameQuizAnswererRow
	r.db.Table("galgame_quiz_answer").
		Select("user_id, submitted, is_correct, created").
		Where("quiz_id = ? AND role = ?", quizID, "answerer").
		Order("created DESC").
		Limit(limit).
		Scan(&rows)
	return rows
}

func (r *QuizRepository) FindAnswerersForRegrade(quizID int) []model.GalgameQuizAnswer {
	var rows []model.GalgameQuizAnswer
	r.db.Where("quiz_id = ? AND role = ?", quizID, "answerer").Find(&rows)
	return rows
}
