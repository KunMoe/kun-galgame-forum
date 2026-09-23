package apiv1

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/quiz/repository"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

const answersSort = "created_id_desc"

func (s *Service) listQuizAnswers(ctx context.Context, in *listAnswersInput) (*listAnswersOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	row, _, p := s.visibleQuiz(ctx, in.QuizID)
	if p != nil {
		return nil, p
	}
	fp := collect.Fingerprint(strconv.Itoa(row.ID))
	keys, curErr := collect.DecodeCursor(in.Cursor, answersSort, fp)
	if curErr != nil {
		return nil, curErr
	}
	var after *repository.AnswerCursor
	if keys != nil {
		if len(keys) != 2 {
			return nil, invalidCursor()
		}
		t, err := time.Parse(time.RFC3339Nano, keys[0])
		id, ok := parseID(keys[1])
		if err != nil || !ok {
			return nil, invalidCursor()
		}
		after = &repository.AnswerCursor{Created: t, ID: id}
	}
	ids, err := s.store.DistinctAnswererIDs(row.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	keep, p := s.keepAuthors(ctx, ids)
	if p != nil {
		return nil, p
	}
	limit := in.Limit
	if limit < 1 {
		limit = collect.DefaultLimit
	}
	rows, err := s.store.ListAnswers(row.ID, keep, after, limit+1)
	if err != nil {
		return nil, problem.Internal(err)
	}
	more := false
	if len(rows) > limit {
		rows, more = rows[:limit], true
	}
	answererIDs := make([]int, 0, len(rows))
	for _, r := range rows {
		answererIDs = append(answererIDs, r.UserID)
	}
	users, p := s.lookupUsers(ctx, answererIDs)
	if p != nil {
		return nil, p
	}
	viewer := v1.User(ctx)
	var own *model.GalgameQuizAnswer
	if viewer != nil {
		own, err = s.store.FindAnswer(row.ID, viewer.ID)
		if err != nil {
			return nil, problem.Internal(err)
		}
	}
	show := seeKey(row, viewer, own)
	items := make([]QuizAnswer, 0, len(rows))
	for _, r := range rows {
		item, p := s.answerItem(row.Type, r, users, show)
		if p != nil {
			return nil, p
		}
		items = append(items, item)
	}
	var next *string
	if more && len(rows) > 0 {
		last := rows[len(rows)-1]
		cur := collect.EncodeCursor(answersSort, fp, last.CreatedAt.UTC().Format(time.RFC3339Nano), strconv.Itoa(last.ID))
		next = &cur
	}
	var total *int
	if in.IncludeTotal {
		n, err := s.store.CountAnswers(row.ID, keep)
		if err != nil {
			return nil, problem.Internal(err)
		}
		total = &n
	}
	return &listAnswersOutput{Body: repr.NewCountedList(items, next, total)}, nil
}

func invalidCursor() *problem.Problem {
	return problem.New(
		problem.CodeInvalidCursor,
		"The cursor cannot be parsed or is no longer valid.",
		problem.AtParameter("cursor", problem.ReasonInvalidFormat, "pass the next_cursor from a previous page of this collection", nil),
	)
}

func (s *Service) answerItem(qtype string, r model.GalgameQuizAnswer, users map[int]userclient.User, show bool) (QuizAnswer, *problem.Problem) {
	u, ok := users[r.UserID]
	if !ok {
		u = userclient.User{ID: r.UserID}
	}
	item := QuizAnswer{
		Object: "quiz_answer", ID: repr.ID(r.ID), QuizID: repr.ID(r.QuizID),
		Answerer: repr.NewUserRef(s.cdn, u), AnsweredAt: repr.Timestamp(r.CreatedAt),
	}
	if show {
		sub, err := decodeSubmission(qtype, r.Submitted)
		if err != nil {
			return QuizAnswer{}, problem.Internal(err)
		}
		item.Submission = &sub
		item.IsCorrect = r.IsCorrect
	}
	return item, nil
}

func (s *Service) createQuizAnswer(ctx context.Context, in *createAnswerInput) (*createAnswerOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	row, _, p := s.visibleQuiz(ctx, in.QuizID)
	if p != nil {
		return nil, p
	}
	if user.ID == row.UserID {
		return nil, selfAnswerForbidden()
	}
	existing, err := s.store.FindAnswer(row.ID, user.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if existing != nil {
		return nil, alreadyExists()
	}
	choices, err := parseChoices(row.Type, row.Content)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if errs := validateSubmission(row.Type, len(choices), in.Body); len(errs) > 0 {
		return nil, validationFailed(errs...)
	}
	submitted, err := encodeSubmission(row.Type, in.Body)
	if err != nil {
		return nil, problem.Internal(err)
	}
	correct, err := grade(row.Type, row.Content, submitted)
	if err != nil {
		return nil, problem.Internal(err)
	}
	ans := &model.GalgameQuizAnswer{
		QuizID: row.ID, UserID: user.ID, Role: "answerer",
		Submitted: submitted, IsCorrect: boolPtr(correct),
	}
	err = s.store.InTx(func(tx *gorm.DB) error {
		if err := s.store.InsertAnswer(tx, ans); err != nil {
			return err
		}
		if err := s.store.BumpAnswerStats(tx, row.ID, correct); err != nil {
			return err
		}
		return s.store.NotifyAnswered(tx, user.ID, row.UserID, row.ID, answerNotice(row.Type, choices, in.Body, correct))
	})
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
			return nil, alreadyExists()
		}
		return nil, problem.Internal(err)
	}
	more, p := s.lookupUsers(ctx, []int{user.ID})
	if p != nil {
		return nil, p
	}
	item, p := s.answerItem(row.Type, *ans, more, true)
	if p != nil {
		return nil, p
	}
	sol, p := s.solutionOf(ctx, row)
	if p != nil {
		return nil, p
	}
	return &createAnswerOutput{
		Location: "/api/v1/quizzes/" + strconv.Itoa(row.ID),
		Body:     QuizAnswerResult{Object: "quiz_answer_result", Answer: item, Solution: *sol},
	}, nil
}

func boolPtr(b bool) *bool { return &b }

func answerNotice(qtype string, choices []string, sub QuizSubmission, correct bool) string {
	var picked string
	if qtype == quizTypeJudge {
		picked = "错误"
		if sub.JudgeChoice != nil && *sub.JudgeChoice {
			picked = "正确"
		}
	} else {
		labels := make([]string, 0, len(sub.ChoiceIndexes))
		for _, i := range sub.ChoiceIndexes {
			labels = append(labels, string(rune('A'+i))+". "+choices[i])
		}
		picked = strings.Join(labels, "、")
	}
	if correct {
		return "选择「" + picked + "」，回答正确"
	}
	return "选择「" + picked + "」，回答错误"
}
