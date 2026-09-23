package apiv1

import (
	"context"
	"errors"
	"net/url"
	"strconv"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/infrastructure/viewstats"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/quiz/repository"
	"kun-galgame-api/internal/trust/gate"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

var errUnconfigured = errors.New("apiv1 quiz: service is not configured")

type Catalog interface {
	CatalogRowsByWorkIDs(ctx context.Context, ids []int, include, contentLimit string) (map[int]client.CatalogWorkListItem, *legacyErrors.AppError)
	CatalogWorksSearch(ctx context.Context, q url.Values) (*client.CatalogWorksPage, *legacyErrors.AppError)
}

type AwardFunc func(userID, delta int, reason, ref, idempotencyKey string)

type pendingAward struct {
	userID int
	delta  int
	reason string
	ref    string
	key    string
}

type Service struct {
	store      *repository.Store
	users      *userclient.Client
	convert    *content.Converter
	check      *gate.CheckService
	scan       *gate.ScanService
	award      AwardFunc
	getCatalog func() Catalog
	cdn        string
}

func New(
	store *repository.Store,
	users *userclient.Client,
	convert *content.Converter,
	check *gate.CheckService,
	scan *gate.ScanService,
	award AwardFunc,
	getCatalog func() Catalog,
	cdn string,
) *Service {
	if check == nil {
		check = gate.NewCheckService(nil)
	}
	if scan == nil {
		scan = gate.NewScanService(nil)
	}
	if award == nil {
		award = moemoepoint.Award
	}
	return &Service{
		store: store, users: users, convert: convert,
		check: check, scan: scan, award: award,
		getCatalog: getCatalog, cdn: cdn,
	}
}

func (s *Service) ready() *problem.Problem {
	if s == nil || s.store == nil || !s.store.Ready() || s.users == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

func (s *Service) catalog() Catalog {
	if s == nil || s.getCatalog == nil {
		return nil
	}
	return s.getCatalog()
}

func (s *Service) requireActive(ctx context.Context) (*middleware.UserInfo, *problem.Problem) {
	user := v1.User(ctx)
	if user == nil {
		return nil, problem.New(problem.CodeMissingCredential, "The request has no credentials.")
	}
	users, err := s.users.Users(ctx, []int{user.ID})
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	if u, ok := users[user.ID]; ok && !userclient.IsRenderable(u) {
		return nil, problem.New(problem.CodeAccountBanned, "The signed-in user's account is banned.")
	}
	return user, nil
}

func (s *Service) lookupUsers(ctx context.Context, ids []int) (map[int]userclient.User, *problem.Problem) {
	if s.users == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	users, err := s.users.Users(ctx, ids)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	if users == nil {
		users = map[int]userclient.User{}
	}
	return users, nil
}

func renderable(users map[int]userclient.User, id int) bool {
	u, ok := users[id]
	return ok && userclient.IsRenderable(u)
}

func (s *Service) convertBody(ctx context.Context, source string) (content.ContentDocument, *problem.Problem) {
	if s.convert == nil {
		return content.ContentDocument{}, problem.Internal(errUnconfigured)
	}
	docs, err := s.convert.Convert(ctx, []string{source})
	if err != nil {
		return content.ContentDocument{}, problem.Unavailable(err)
	}
	return docs[0], nil
}

func (s *Service) rejectContent(ctx context.Context, text string, authorID int) (decision string, matched []string, p *problem.Problem) {
	if trimSpace(text) == "" {
		return gate.DecisionAllow, nil, nil
	}
	aid := int64(authorID)
	decision, matched = s.check.Decision(ctx, text, &aid)
	if decision == gate.DecisionDeny {
		return decision, matched, contentRejected()
	}
	return decision, matched, nil
}

func (s *Service) flushAwards(jobs []pendingAward) {
	for _, j := range jobs {
		s.award(j.userID, j.delta, j.reason, j.ref, j.key)
	}
}

func (s *Service) visibleQuiz(ctx context.Context, rawID string) (*model.GalgameQuiz, map[int]userclient.User, *problem.Problem) {
	id, ok := parseID(rawID)
	if !ok {
		return nil, nil, notFound()
	}
	row, err := s.store.Find(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, notFound()
		}
		return nil, nil, problem.Internal(err)
	}
	users, p := s.lookupUsers(ctx, []int{row.UserID})
	if p != nil {
		return nil, nil, p
	}
	if !renderable(users, row.UserID) {
		return nil, nil, notFound()
	}
	return row, users, nil
}

func (s *Service) keepAuthors(ctx context.Context, ids []int) ([]int, *problem.Problem) {
	users, p := s.lookupUsers(ctx, ids)
	if p != nil {
		return nil, p
	}
	keep := make([]int, 0, len(ids))
	for _, id := range ids {
		if renderable(users, id) {
			keep = append(keep, id)
		}
	}
	return keep, nil
}

func (s *Service) summaries(ctx context.Context, rows []model.GalgameQuiz, viewer *middleware.UserInfo) ([]QuizSummary, *problem.Problem) {
	if len(rows) == 0 {
		return []QuizSummary{}, nil
	}
	authorIDs := make([]int, 0, len(rows))
	quizIDs := make([]int, 0, len(rows))
	for _, r := range rows {
		authorIDs = append(authorIDs, r.UserID)
		quizIDs = append(quizIDs, r.ID)
	}
	users, p := s.lookupUsers(ctx, authorIDs)
	if p != nil {
		return nil, p
	}
	var answers map[int]repository.ViewerAnswer
	if viewer != nil {
		var err error
		answers, err = s.store.FindViewerAnswers(quizIDs, viewer.ID)
		if err != nil {
			return nil, problem.Internal(err)
		}
	}
	out := make([]QuizSummary, 0, len(rows))
	for _, r := range rows {
		item := fromRow(r)
		author, ok := users[r.UserID]
		if !ok {
			author = userclient.User{ID: r.UserID}
		}
		item.Author = repr.NewUserRef(s.cdn, author)
		if viewer != nil {
			va, ok := answers[r.ID]
			sv := &QuizSummaryViewer{}
			if ok && va.Role == "answerer" {
				sv.HasAnswered = true
				sv.IsCorrect = va.IsCorrect
			}
			item.Viewer = sv
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) worksOf(ctx context.Context, workIDs []int, hide bool, seeKey bool) ([]repr.WorkRef, *problem.Problem) {
	if hide && !seeKey {
		return []repr.WorkRef{}, nil
	}
	if len(workIDs) == 0 {
		return []repr.WorkRef{}, nil
	}
	cat := s.catalog()
	if cat == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	rows, appErr := cat.CatalogRowsByWorkIDs(ctx, workIDs, "names,covers", "all")
	if appErr != nil {
		return nil, problem.Unavailable(appErr)
	}
	out := make([]repr.WorkRef, 0, len(workIDs))
	for _, id := range workIDs {
		row, ok := rows[id]
		if !ok {
			continue
		}
		if !client.CatalogItemRenderable(&row) {
			continue
		}
		cp := row
		out = append(out, galgameapiv1.WorkRefOf(ctx, &cp, s.cdn))
	}
	return out, nil
}

func (s *Service) lookupWorks(ctx context.Context, workIDs []int) (map[int]client.CatalogWorkListItem, *problem.Problem) {
	if len(workIDs) == 0 {
		return map[int]client.CatalogWorkListItem{}, nil
	}
	cat := s.catalog()
	if cat == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	rows, appErr := cat.CatalogRowsByWorkIDs(ctx, workIDs, "names,covers", "all")
	if appErr != nil {
		return nil, problem.Unavailable(appErr)
	}
	return rows, nil
}

func seeKey(row *model.GalgameQuiz, viewer *middleware.UserInfo, answer *model.GalgameQuizAnswer) bool {
	if viewer == nil {
		return false
	}
	if viewer.ID == row.UserID {
		return true
	}
	if canEditQuiz(row.UserID, viewer) {
		return true
	}
	return answer != nil && answer.Role == "answerer"
}

func (s *Service) detail(ctx context.Context, row *model.GalgameQuiz, users map[int]userclient.User, viewer *middleware.UserInfo) (*Quiz, *problem.Problem) {
	sums, p := s.summaries(ctx, []model.GalgameQuiz{*row}, viewer)
	if p != nil {
		return nil, p
	}
	doc, p := s.convertBody(ctx, row.Description)
	if p != nil {
		return nil, p
	}
	choices, err := parseChoices(row.Type, row.Content)
	if err != nil {
		return nil, problem.Internal(err)
	}
	var answer *model.GalgameQuizAnswer
	if viewer != nil {
		answer, err = s.store.FindAnswer(row.ID, viewer.ID)
		if err != nil {
			return nil, problem.Internal(err)
		}
	}
	okKey := seeKey(row, viewer, answer)
	workIDs, err := s.store.WorkIDs(row.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	works, p := s.worksOf(ctx, workIDs, row.HideGalgame, okKey)
	if p != nil {
		return nil, p
	}
	item := sums[0]
	out := Quiz{
		Object: item.Object, ID: item.ID, Prompt: item.Prompt, QuizType: item.QuizType,
		QuizCategory: item.QuizCategory, Difficulty: item.Difficulty, SpoilerLevel: item.SpoilerLevel,
		Author: item.Author, ViewCount: item.ViewCount, AnswerCount: item.AnswerCount,
		CorrectCount: item.CorrectCount, FavoriteCount: item.FavoriteCount,
		QualityAverage: item.QualityAverage, QualityCount: item.QualityCount, CommentCount: item.CommentCount,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, BumpedAt: item.BumpedAt,
		Content: doc, Choices: typedChoices(choices), IsWorkHidden: row.HideGalgame, Works: works,
	}
	if okKey {
		sol, p := s.solutionOf(ctx, row)
		if p != nil {
			return nil, p
		}
		out.Solution = sol
	}
	if viewer != nil {
		fav, err := s.store.HasFavorited(row.ID, viewer.ID)
		if err != nil {
			return nil, problem.Internal(err)
		}
		qv := &QuizViewer{
			CanEdit: canEditQuiz(row.UserID, viewer), CanDelete: canDeleteQuiz(row.UserID, viewer),
			HasFavorited: fav,
		}
		if answer != nil && answer.Role == "answerer" {
			qv.HasAnswered = true
			sub, err := decodeSubmission(row.Type, answer.Submitted)
			if err != nil {
				return nil, problem.Internal(err)
			}
			correct := answer.IsCorrect != nil && *answer.IsCorrect
			qv.Answer = &QuizViewerAnswer{
				Submission: sub, IsCorrect: correct, AnsweredAt: repr.Timestamp(answer.CreatedAt),
			}
			qv.QualityRating = answer.QualityRating
		}
		out.Viewer = qv
	}
	return &out, nil
}

func (s *Service) solutionOf(ctx context.Context, row *model.GalgameQuiz) (*QuizSolution, *problem.Problem) {
	indexes, judge, err := parseKey(row.Type, row.Content)
	if err != nil {
		return nil, problem.Internal(err)
	}
	expl, p := s.convertBody(ctx, row.Explanation)
	if p != nil {
		return nil, p
	}
	return &QuizSolution{
		Object: "quiz_solution", CorrectChoiceIndexes: copyInts(indexes),
		JudgeAnswer: judge, Explanation: expl,
	}, nil
}

func (s *Service) sourceOf(row *model.GalgameQuiz, workIDs []int) (QuizSource, *problem.Problem) {
	choices, err := parseChoices(row.Type, row.Content)
	if err != nil {
		return QuizSource{}, problem.Internal(err)
	}
	indexes, judge, err := parseKey(row.Type, row.Content)
	if err != nil {
		return QuizSource{}, problem.Internal(err)
	}
	return QuizSource{
		Object: "quiz_source", QuizID: repr.ID(row.ID), QuizType: row.Type, QuizCategory: row.Category,
		Difficulty: row.Difficulty, SpoilerLevel: row.SpoilerLevel, PromptText: row.Question,
		DescriptionMarkdown: row.Description, ExplanationMarkdown: row.Explanation,
		Choices: typedChoices(choices), CorrectChoiceIndexes: copyInts(indexes),
		JudgeAnswer: judge, WorkIDs: workIDStrings(workIDs), IsWorkHidden: row.HideGalgame,
	}, nil
}

func (s *Service) bumpView(id int) *problem.Problem {
	if err := s.store.IncrementView(id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return notFound()
		}
		return problem.Internal(err)
	}
	if err := viewstats.BumpDaily(s.store.DB(), viewstats.QuizDaily, id); err != nil {
		return problem.Internal(err)
	}
	return nil
}

func (s *Service) afterCommit(err error, jobs []pendingAward) error {
	if err != nil {
		return err
	}
	s.flushAwards(jobs)
	return nil
}

func txProblem(err error) *problem.Problem {
	if err == nil {
		return nil
	}
	var p *problem.Problem
	if errors.As(err, &p) {
		return p
	}
	return nil
}

func (s *Service) checkWorksExistOrUnavail(ctx context.Context, ids []int) ([]problem.FieldError, *problem.Problem) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, p := s.lookupWorks(ctx, ids)
	if p != nil {
		return nil, p
	}
	var errs []problem.FieldError
	for i, id := range ids {
		row, ok := rows[id]
		if !ok || !client.CatalogItemRenderable(&row) {
			errs = append(errs, unknownRef("/work_ids/"+strconv.Itoa(i)))
		}
	}
	return errs, nil
}
