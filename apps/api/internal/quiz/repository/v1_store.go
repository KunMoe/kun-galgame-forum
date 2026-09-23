package repository

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/internal/galgame/model"
	msgModel "kun-galgame-api/internal/message/model"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var (
	ErrNotFound      = errors.New("quiz: row not found")
	ErrAlreadyExists = errors.New("quiz: already exists")
)

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Ready() bool {
	return s != nil && s.db != nil
}

func (s *Store) DB() *gorm.DB { return s.db }

func (s *Store) InTx(fn func(tx *gorm.DB) error) error {
	return s.db.Transaction(fn)
}

func UniqueViolation(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}

func notFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

type ListFilter struct {
	Type        string
	Category    string
	Spoiler     string
	Difficulty  int
	WorkID      int
	AuthorID    int
	IncludeNSFW bool
	AuthorIDs   []int
}

type SortSpec struct {
	Expr string
}

func (f ListFilter) apply(q *gorm.DB) *gorm.DB {
	if f.Type != "" {
		q = q.Where("q.type = ?", f.Type)
	}
	if f.Category != "" {
		q = q.Where("q.category = ?", f.Category)
	}
	if f.Spoiler != "" {
		q = q.Where("q.spoiler_level = ?", f.Spoiler)
	}
	if f.Difficulty > 0 {
		q = q.Where("q.difficulty = ?", f.Difficulty)
	}
	if f.WorkID > 0 {
		q = q.Where(`EXISTS (SELECT 1 FROM galgame_quiz_galgame gg WHERE gg.quiz_id = q.id AND gg.work_id = ?)`, f.WorkID)
	}
	if f.AuthorID > 0 {
		q = q.Where("q.user_id = ?", f.AuthorID)
	}
	if !f.IncludeNSFW {
		q = q.Where(`NOT EXISTS (
			SELECT 1 FROM galgame_quiz_galgame gg
			JOIN galgame g ON g.id = gg.work_id
			WHERE gg.quiz_id = q.id AND g.content_limit = 'nsfw')`)
	}
	if f.AuthorIDs != nil {
		q = q.Where("q.user_id IN ?", f.AuthorIDs)
	}
	return q
}

func (s *Store) DistinctAuthors(f ListFilter) ([]int, error) {
	f.AuthorIDs = nil
	var ids []int
	err := f.apply(s.db.Table("galgame_quiz q")).Distinct("q.user_id").Pluck("q.user_id", &ids).Error
	return ids, err
}

func (s *Store) Count(f ListFilter) (int, error) {
	if f.AuthorIDs != nil && len(f.AuthorIDs) == 0 {
		return 0, nil
	}
	var n int64
	err := f.apply(s.db.Table("galgame_quiz q")).Count(&n).Error
	return int(n), err
}

func (s *Store) List(f ListFilter, sort SortSpec, offset, limit int) ([]model.GalgameQuiz, error) {
	if f.AuthorIDs != nil && len(f.AuthorIDs) == 0 {
		return nil, nil
	}
	var rows []model.GalgameQuiz
	err := f.apply(s.db.Table("galgame_quiz q")).
		Select("q.*").
		Order(sort.Expr).
		Offset(offset).Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (s *Store) DistinctAnsweredAuthors(answererID int) ([]int, error) {
	var ids []int
	err := s.db.Table("galgame_quiz q").
		Joins("JOIN galgame_quiz_answer a ON a.quiz_id = q.id").
		Where("a.user_id = ? AND a.role = ?", answererID, "answerer").
		Distinct("q.user_id").Pluck("q.user_id", &ids).Error
	return ids, err
}

func (s *Store) CountAnswered(answererID int, authorIDs []int) (int, error) {
	if authorIDs != nil && len(authorIDs) == 0 {
		return 0, nil
	}
	q := s.db.Table("galgame_quiz q").
		Joins("JOIN galgame_quiz_answer a ON a.quiz_id = q.id").
		Where("a.user_id = ? AND a.role = ?", answererID, "answerer")
	if authorIDs != nil {
		q = q.Where("q.user_id IN ?", authorIDs)
	}
	var n int64
	err := q.Count(&n).Error
	return int(n), err
}

func (s *Store) ListAnswered(answererID int, authorIDs []int, offset, limit int) ([]model.GalgameQuiz, error) {
	if authorIDs != nil && len(authorIDs) == 0 {
		return nil, nil
	}
	q := s.db.Table("galgame_quiz q").
		Joins("JOIN galgame_quiz_answer a ON a.quiz_id = q.id").
		Where("a.user_id = ? AND a.role = ?", answererID, "answerer")
	if authorIDs != nil {
		q = q.Where("q.user_id IN ?", authorIDs)
	}
	var rows []model.GalgameQuiz
	err := q.Select("q.*").
		Order("a.created DESC, a.id DESC").
		Offset(offset).Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (s *Store) Find(id int) (*model.GalgameQuiz, error) {
	var row model.GalgameQuiz
	err := s.db.First(&row, id).Error
	if err != nil {
		return nil, notFound(err)
	}
	return &row, nil
}

func (s *Store) FindMany(ids []int) ([]model.GalgameQuiz, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []model.GalgameQuiz
	err := s.db.Where("id IN ?", ids).Find(&rows).Error
	return rows, err
}

func (s *Store) IncrementView(id int) error {
	res := s.db.Exec(`UPDATE galgame_quiz SET view = view + 1 WHERE id = ?`, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Create(tx *gorm.DB, q *model.GalgameQuiz) error {
	return tx.Create(q).Error
}

func (s *Store) CreateAuthorRow(tx *gorm.DB, quizID, userID int) error {
	return tx.Create(&model.GalgameQuizAnswer{
		QuizID: quizID, UserID: userID, Role: "author",
	}).Error
}

func (s *Store) Patch(tx *gorm.DB, id int, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	res := tx.Model(&model.GalgameQuiz{}).Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Delete(tx *gorm.DB, id int) error {
	res := tx.Where("id = ?", id).Delete(&model.GalgameQuiz{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) WorkIDs(quizID int) ([]int, error) {
	var ids []int
	err := s.db.Table("galgame_quiz_galgame").
		Where("quiz_id = ?", quizID).
		Order("work_id").
		Pluck("work_id", &ids).Error
	if ids == nil {
		ids = []int{}
	}
	return ids, err
}

func (s *Store) WorkIDsMany(quizIDs []int) (map[int][]int, error) {
	out := map[int][]int{}
	if len(quizIDs) == 0 {
		return out, nil
	}
	var rows []model.GalgameQuizGalgame
	if err := s.db.Where("quiz_id IN ?", quizIDs).Order("work_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.QuizID] = append(out[r.QuizID], r.WorkID)
	}
	return out, nil
}

func (s *Store) SetWorks(tx *gorm.DB, quizID int, workIDs []int) error {
	if err := tx.Where("quiz_id = ?", quizID).Delete(&model.GalgameQuizGalgame{}).Error; err != nil {
		return err
	}
	if len(workIDs) == 0 {
		return nil
	}
	rows := make([]model.GalgameQuizGalgame, 0, len(workIDs))
	seen := map[int]struct{}{}
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

type ViewerAnswer struct {
	QuizID    int
	Role      string
	IsCorrect *bool
	Submitted json.RawMessage
	Quality   *int
	CreatedAt time.Time
	ID        int
}

func (s *Store) FindAnswer(quizID, userID int) (*model.GalgameQuizAnswer, error) {
	var a model.GalgameQuizAnswer
	err := s.db.Where("quiz_id = ? AND user_id = ?", quizID, userID).First(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Store) FindViewerAnswers(quizIDs []int, userID int) (map[int]ViewerAnswer, error) {
	out := map[int]ViewerAnswer{}
	if len(quizIDs) == 0 || userID == 0 {
		return out, nil
	}
	var rows []model.GalgameQuizAnswer
	if err := s.db.Where("user_id = ? AND quiz_id IN ?", userID, quizIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.QuizID] = ViewerAnswer{
			QuizID: r.QuizID, Role: r.Role, IsCorrect: r.IsCorrect,
			Submitted: r.Submitted, Quality: r.QualityRating, CreatedAt: r.CreatedAt, ID: r.ID,
		}
	}
	return out, nil
}

func (s *Store) InsertAnswer(tx *gorm.DB, a *model.GalgameQuizAnswer) error {
	err := tx.Create(a).Error
	if UniqueViolation(err) {
		return ErrAlreadyExists
	}
	return err
}

func (s *Store) BumpAnswerStats(tx *gorm.DB, quizID int, correct bool) error {
	cutoff := model.QuizBumpCutoff(time.Now())
	sql := `UPDATE galgame_quiz SET
		answer_count = answer_count + 1,
		correct_count = correct_count + ?,
		status_update_time = CASE WHEN created > ? THEN now() ELSE status_update_time END
		WHERE id = ?`
	delta := 0
	if correct {
		delta = 1
	}
	return tx.Exec(sql, delta, cutoff, quizID).Error
}

func AnswerNotice(qtype string, choices []string, indexes []int, statementTrue *bool, correct bool) string {
	var picked string
	if qtype == "judge" {
		picked = "错误"
		if statementTrue != nil && *statementTrue {
			picked = "正确"
		}
	} else {
		labels := make([]string, 0, len(indexes))
		for _, i := range indexes {
			labels = append(labels, string(rune('A'+i))+". "+choices[i])
		}
		picked = strings.Join(labels, "、")
	}
	if correct {
		return "选择「" + picked + "」，回答正确"
	}
	return "选择「" + picked + "」，回答错误"
}

func (s *Store) NotifyAnswered(tx *gorm.DB, senderID, receiverID, quizID int, content string) error {
	link := "/galgame-quiz/" + strconv.Itoa(quizID)
	var count int64
	if err := tx.Model(&msgModel.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND type = ? AND link = ?", senderID, receiverID, "quiz-answered", link).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return tx.Create(&msgModel.Message{
		SenderID: senderID, ReceiverID: receiverID,
		Type: "quiz-answered", Content: content, Link: link, Status: "unread",
	}).Error
}

func (s *Store) AnswerersForRegrade(tx *gorm.DB, quizID int) ([]model.GalgameQuizAnswer, error) {
	var rows []model.GalgameQuizAnswer
	err := tx.Where("quiz_id = ? AND role = ?", quizID, "answerer").Find(&rows).Error
	return rows, err
}

func (s *Store) SetAnswerCorrect(tx *gorm.DB, answerID int, correct bool) error {
	return tx.Exec(`UPDATE galgame_quiz_answer SET is_correct = ? WHERE id = ?`, correct, answerID).Error
}

func (s *Store) RecountCorrect(tx *gorm.DB, quizID int) error {
	return tx.Exec(`UPDATE galgame_quiz SET correct_count = (
		SELECT COUNT(*) FROM galgame_quiz_answer
		WHERE quiz_id = ? AND role = 'answerer' AND is_correct
	) WHERE id = ?`, quizID, quizID).Error
}

func (s *Store) SetAnswerQuality(tx *gorm.DB, answerID, rating int) error {
	return tx.Exec(`UPDATE galgame_quiz_answer SET quality_rating = ? WHERE id = ?`, rating, answerID).Error
}

func (s *Store) AdjustQuality(tx *gorm.DB, quizID, sumDelta, countDelta int) error {
	return tx.Exec(`UPDATE galgame_quiz SET
		quality_sum = quality_sum + ?,
		quality_count = quality_count + ?
		WHERE id = ?`, sumDelta, countDelta, quizID).Error
}

func (s *Store) ReadQuality(tx *gorm.DB, quizID int) (sum, count int, err error) {
	row := tx.Raw(`SELECT quality_sum, quality_count FROM galgame_quiz WHERE id = ?`, quizID).Row()
	err = row.Scan(&sum, &count)
	return sum, count, err
}

func returningInt64(tx *gorm.DB, q string, args ...any) (int64, bool, error) {
	rows, err := tx.Raw(q, args...).Rows()
	if err != nil {
		return 0, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return 0, false, rows.Err()
	}
	var id int64
	if err := rows.Scan(&id); err != nil {
		return 0, false, err
	}
	return id, true, rows.Err()
}

func (s *Store) InsertFavorite(tx *gorm.DB, quizID, userID int) (int64, bool, error) {
	return returningInt64(tx, `
		INSERT INTO galgame_quiz_favorite (quiz_id, user_id)
		VALUES (?, ?)
		ON CONFLICT (quiz_id, user_id) DO NOTHING
		RETURNING id`, quizID, userID)
}

func (s *Store) DeleteFavorite(tx *gorm.DB, quizID, userID int) (int64, bool, error) {
	return returningInt64(tx, `
		DELETE FROM galgame_quiz_favorite
		WHERE quiz_id = ? AND user_id = ?
		RETURNING id`, quizID, userID)
}

func (s *Store) AdjustFavoriteCount(tx *gorm.DB, quizID, delta int) error {
	if delta >= 0 {
		return tx.Exec(`UPDATE galgame_quiz SET favorite_count = favorite_count + ? WHERE id = ?`, delta, quizID).Error
	}
	return tx.Exec(`UPDATE galgame_quiz SET favorite_count = GREATEST(favorite_count + ?, 0) WHERE id = ?`, delta, quizID).Error
}

func (s *Store) HasFavorited(quizID, userID int) (bool, error) {
	if userID == 0 {
		return false, nil
	}
	var n int64
	err := s.db.Table("galgame_quiz_favorite").
		Where("quiz_id = ? AND user_id = ?", quizID, userID).
		Count(&n).Error
	return n > 0, err
}

func (s *Store) FavoritedSet(userID int, quizIDs []int) (map[int]bool, error) {
	out := map[int]bool{}
	if userID == 0 || len(quizIDs) == 0 {
		return out, nil
	}
	var ids []int
	err := s.db.Table("galgame_quiz_favorite").
		Where("user_id = ? AND quiz_id IN ?", userID, quizIDs).
		Pluck("quiz_id", &ids).Error
	for _, id := range ids {
		out[id] = true
	}
	return out, err
}

func (s *Store) FavoriteCount(quizID int) (int, error) {
	var n int
	err := s.db.Raw(`SELECT favorite_count FROM galgame_quiz WHERE id = ?`, quizID).Scan(&n).Error
	return n, err
}

type AnswerCursor struct {
	Created time.Time
	ID      int
}

func (s *Store) DistinctAnswererIDs(quizID int) ([]int, error) {
	var ids []int
	err := s.db.Table("galgame_quiz_answer").
		Where("quiz_id = ? AND role = ?", quizID, "answerer").
		Distinct("user_id").Pluck("user_id", &ids).Error
	return ids, err
}

func (s *Store) CountAnswers(quizID int, userIDs []int) (int, error) {
	if userIDs != nil && len(userIDs) == 0 {
		return 0, nil
	}
	q := s.db.Table("galgame_quiz_answer").
		Where("quiz_id = ? AND role = ?", quizID, "answerer")
	if userIDs != nil {
		q = q.Where("user_id IN ?", userIDs)
	}
	var n int64
	err := q.Count(&n).Error
	return int(n), err
}

func (s *Store) ListAnswers(quizID int, userIDs []int, after *AnswerCursor, limit int) ([]model.GalgameQuizAnswer, error) {
	if userIDs != nil && len(userIDs) == 0 {
		return nil, nil
	}
	q := s.db.Where("quiz_id = ? AND role = ?", quizID, "answerer")
	if userIDs != nil {
		q = q.Where("user_id IN ?", userIDs)
	}
	if after != nil {
		q = q.Where("(created < ?) OR (created = ? AND id < ?)", after.Created, after.Created, after.ID)
	}
	var rows []model.GalgameQuizAnswer
	err := q.Order("created DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}
