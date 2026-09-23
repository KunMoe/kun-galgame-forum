package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/url"
	"strconv"
	"time"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

type QuizService struct {
	quizRepo      *repository.QuizRepository
	galgameClient *client.GalgameClient
	userClient    *userclient.Client
	check         *gate.CheckService
	scan          *gate.ScanService
	helpers       InteractionHelpers
}

func NewQuizService(
	quizRepo *repository.QuizRepository,
	galgameClient *client.GalgameClient,
	userClient *userclient.Client,
	check *gate.CheckService,
	scan *gate.ScanService,
) *QuizService {
	return &QuizService{
		quizRepo:      quizRepo,
		galgameClient: galgameClient,
		userClient:    userClient,
		check:         check,
		scan:          scan,
	}
}

func quizAuthoringModerationText(question, description, explanation, qtype string, content json.RawMessage) string {
	return gate.ComposeText(question, description, explanation, quizContentModerationText(qtype, content))
}

func quizCreateReward(_ int) int {
	return constants.QuizCreateReward
}

func quizCorrectReward(_ int) int {
	return constants.QuizCorrectReward
}

func (s *QuizService) GetAllQuizzes(
	ctx context.Context,
	req *dto.QuizListRequest,
	isSFW bool,
	viewerID int,
) (*dto.QuizListPage, *errors.AppError) {
	sortOrder := req.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}
	rows, total := s.quizRepo.ListPaginated(model.QuizFilter{
		Category:   req.Category,
		Type:       req.Type,
		SortField:  req.SortField,
		SortOrder:  sortOrder,
		Difficulty: req.Difficulty,
		WorkID:     req.WorkID,
		UserID:     req.UserID,
		Page:       req.Page,
		Limit:      req.Limit,
	})
	return s.hydrateCards(ctx, rows, total, viewerID), nil
}

func (s *QuizService) GetMyAnswered(
	ctx context.Context,
	userID, page, limit int,
) (*dto.QuizListPage, *errors.AppError) {
	rows, total := s.quizRepo.ListAnsweredByUser(userID, page, limit)
	return s.hydrateCards(ctx, rows, total, userID), nil
}

func (s *QuizService) hydrateCards(
	ctx context.Context,
	rows []model.GalgameQuizRow,
	total int64,
	viewerID int,
) *dto.QuizListPage {
	userIDs := make([]int, 0, len(rows))
	quizIDs := make([]int, 0, len(rows))
	for _, r := range rows {
		userIDs = append(userIDs, r.UserID)
		quizIDs = append(quizIDs, r.ID)
	}
	userMap := s.userClient.Hydrate(ctx, userIDs)
	viewerAnswers := s.quizRepo.FindViewerAnswers(quizIDs, viewerID)

	cards := make([]dto.QuizCard, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		card := quizRowToCard(r, u)
		va, answered := viewerAnswers[r.ID]
		card.MyStatus = quizViewerStatus(va, answered)
		cards = append(cards, card)
	}
	return &dto.QuizListPage{QuizData: cards, Total: total}
}

func quizViewerStatus(va repository.QuizViewerAnswer, answered bool) string {
	switch {
	case !answered:
		return "unanswered"
	case va.Role == "author":
		return "author"
	case va.IsCorrect == nil:
		return "answered"
	case *va.IsCorrect:
		return "correct"
	default:
		return "incorrect"
	}
}

func (s *QuizService) GetQuizPlay(
	ctx context.Context,
	quizID, currentUserID int,
) (*dto.QuizPlay, *errors.AppError) {
	quiz, ok := s.quizRepo.FindByID(quizID)
	if !ok {
		return nil, errors.ErrNotFound("题目不存在")
	}
	author, _, _ := s.userClient.User(ctx, quiz.UserID)
	if !userclient.IsRenderable(author) {
		return nil, errors.ErrNotFound("题目不存在")
	}

	s.quizRepo.IncrementView(quizID)
	quiz.View++

	isAuthor := currentUserID != 0 && currentUserID == quiz.UserID

	var myAnswer *dto.QuizAnswerResult
	if currentUserID != 0 {
		if row, has := s.quizRepo.FindAnswer(quizID, currentUserID); has {
			myAnswer = &dto.QuizAnswerResult{
				Submitted:     row.Submitted,
				IsCorrect:     row.IsCorrect,
				Answer:        quiz.Content,
				Explanation:   quiz.Explanation,
				QualityRating: row.QualityRating,
			}
		}
	}

	galgames := []dto.QuizGalgameDetail{}
	if !quiz.HideGalgame || myAnswer != nil {
		galgames = s.galgamesDetailFor(ctx, s.quizRepo.FindQuizWorkIDs(quizID))
	}

	play := &dto.QuizPlay{
		ID:              quiz.ID,
		User:            userBriefToDTO(author),
		Category:        quiz.Category,
		SpoilerLevel:    quiz.SpoilerLevel,
		Type:            quiz.Type,
		Difficulty:      quiz.Difficulty,
		Question:        quiz.Question,
		QuestionHtml:    markdown.RenderQuestionPlain(quiz.Question),
		DescriptionHtml: markdown.Render(quiz.Description),
		Content:         stripQuizContent(quiz.Type, quiz.Content),
		QuizStats:       quizStats(quiz.View, quiz.AnswerCount, quiz.CorrectCount, quiz.FavoriteCount, quiz.QualitySum, quiz.QualityCount, quiz.CommentCount),
		Created:         quiz.CreatedAt.Format(time.RFC3339),
		Updated:         quiz.UpdatedAt.Format(time.RFC3339),
		HideGalgame:     quiz.HideGalgame,
		Galgames:        galgames,
		IsAuthor:        isAuthor,
		IsFavorited:     s.quizRepo.FindQuizFavorite(quizID, currentUserID),
		MyAnswer:        myAnswer,
	}
	return play, nil
}

func (s *QuizService) CreateQuiz(
	ctx context.Context,
	userID int,
	req *dto.CreateQuizRequest,
) (*dto.CreatedQuiz, *errors.AppError) {
	req.Description = markdown.NormalizeStoredContent(req.Description)
	if appErr := validateQuizContent(req.Type, req.Content); appErr != nil {
		return nil, appErr
	}

	moderationText := quizAuthoringModerationText(req.Question, req.Description, req.Explanation, req.Type, req.Content)
	authorID := int64(userID)
	decision, matched := s.check.Decision(ctx, moderationText, &authorID)
	if decision == gate.DecisionDeny {
		return nil, gate.ErrContentBlocked()
	}

	spoiler := req.SpoilerLevel
	if spoiler == "" {
		spoiler = "none"
	}
	quiz := &model.GalgameQuiz{
		UserID:           userID,
		Category:         req.Category,
		SpoilerLevel:     spoiler,
		Type:             req.Type,
		Difficulty:       req.Difficulty,
		Question:         req.Question,
		Description:      req.Description,
		Content:          req.Content,
		Explanation:      req.Explanation,
		HideGalgame:      req.HideGalgame,
		StatusUpdateTime: time.Now(),
	}
	reward := quizCreateReward(req.Difficulty)

	txErr := s.quizRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.quizRepo.Create(tx, quiz); err != nil {
			return err
		}
		if err := s.quizRepo.SetQuizGalgames(tx, quiz.ID, req.WorkIDs); err != nil {
			return err
		}
		if err := s.quizRepo.CreateAnswer(tx, &model.GalgameQuizAnswer{
			QuizID: quiz.ID, UserID: userID, Role: "author",
		}); err != nil {
			return err
		}
		s.helpers.AdjustMoemoepoint(tx, userID, reward,
			moemoepoint.ReasonContentApproved, moemoepoint.Ref("galgame_quiz", quiz.ID))
		return nil
	})
	if txErr != nil {
		return nil, errors.ErrInternal("创建题目失败")
	}

	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindGalgameQuiz, "subject_id", quiz.ID, "author_id", userID, "matched", matched)
	}
	s.scan.ScanBg(gate.SubjectKindGalgameQuiz, strconv.Itoa(quiz.ID), moderationText, int64(userID))

	author, _, _ := s.userClient.User(ctx, userID)
	return &dto.CreatedQuiz{
		ID:           quiz.ID,
		User:         userBriefToDTO(author),
		Category:     quiz.Category,
		SpoilerLevel: quiz.SpoilerLevel,
		Type:         quiz.Type,
		Difficulty:   quiz.Difficulty,
		QuestionHtml: markdown.RenderQuestionPlain(quiz.Question),
		QuizStats:    quizStats(0, 0, 0, 0, 0, 0, 0),
		Created:      quiz.CreatedAt.Format(time.RFC3339),
		Updated:      quiz.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *QuizService) AnswerQuiz(
	ctx context.Context,
	userID int,
	req *dto.AnswerQuizRequest,
) (*dto.QuizAnswerResult, *errors.AppError) {
	quiz, ok := s.quizRepo.FindByID(req.QuizID)
	if !ok {
		return nil, errors.ErrNotFound("题目不存在")
	}
	if existing, has := s.quizRepo.FindAnswer(req.QuizID, userID); has {
		if existing.Role == "author" {
			return nil, errors.ErrBadRequest("不能回答自己出的题目")
		}
		return nil, errors.ErrBadRequest("您已经回答过该题目了")
	}

	grade, appErr := gradeQuiz(quiz.Type, quiz.Content, req.Submitted)
	if appErr != nil {
		return nil, appErr
	}

	moderationText := quizAnswerModerationText(quiz.Type, req.Submitted)
	decision := gate.DecisionAllow
	var matched []string
	if moderationText != "" {
		authorID := int64(userID)
		decision, matched = s.check.Decision(ctx, moderationText, &authorID)
		if decision == gate.DecisionDeny {
			return nil, gate.ErrContentBlocked()
		}
	}

	correct := grade != nil && *grade
	reward := 0
	if correct {
		reward = quizCorrectReward(quiz.Difficulty)
	}

	notifyContent := quizAnswerSummary(quiz.Type, quiz.Content, req.Submitted)
	if grade != nil {
		if *grade {
			notifyContent += "，回答正确"
		} else {
			notifyContent += "，回答错误"
		}
	}

	row := &model.GalgameQuizAnswer{
		QuizID:    req.QuizID,
		UserID:    userID,
		Role:      "answerer",
		Submitted: req.Submitted,
		IsCorrect: grade,
		Rewarded:  reward > 0,
	}
	txErr := s.quizRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.quizRepo.CreateAnswer(tx, row); err != nil {
			return err
		}
		if err := s.quizRepo.BumpAnswerStats(tx, req.QuizID, correct); err != nil {
			return err
		}
		if err := s.helpers.CreateQuizAnswerMessage(tx, userID, quiz.UserID, notifyContent, req.QuizID); err != nil {
			return err
		}
		if reward > 0 {
			s.helpers.AdjustMoemoepoint(tx, userID, reward,
				moemoepoint.ReasonContentApproved, moemoepoint.Ref("galgame_quiz_answer", row.ID))
		}
		return nil
	})
	if txErr != nil {
		return nil, errors.ErrInternal("提交答案失败")
	}

	if moderationText != "" {
		if decision == gate.DecisionHold {
			slog.Info("trust check hold", "subject_kind", gate.SubjectKindGalgameQuizAnswer, "subject_id", row.ID, "author_id", userID, "matched", matched)
		}
		s.scan.ScanBg(gate.SubjectKindGalgameQuizAnswer, strconv.Itoa(row.ID), moderationText, int64(userID))
	}

	return &dto.QuizAnswerResult{
		Submitted:   req.Submitted,
		IsCorrect:   grade,
		Answer:      quiz.Content,
		Explanation: quiz.Explanation,
		RewardDelta: reward,
	}, nil
}

func (s *QuizService) RateQuizQuality(
	userID int,
	req *dto.RateQuizQualityRequest,
) (*dto.QuizQualityResult, *errors.AppError) {
	quiz, ok := s.quizRepo.FindByID(req.QuizID)
	if !ok {
		return nil, errors.ErrNotFound("题目不存在")
	}
	row, has := s.quizRepo.FindAnswer(req.QuizID, userID)
	if !has {
		return nil, errors.ErrForbidden("请先回答题目再评分")
	}
	if row.Role == "author" {
		return nil, errors.ErrForbidden("不能给自己出的题目评分")
	}

	sumDelta, countDelta := req.QualityRating, 1
	if row.QualityRating != nil {
		sumDelta, countDelta = req.QualityRating-*row.QualityRating, 0
	}

	txErr := s.quizRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.quizRepo.SetAnswerQuality(tx, row.ID, req.QualityRating); err != nil {
			return err
		}
		return s.quizRepo.AdjustQuality(tx, req.QuizID, sumDelta, countDelta)
	})
	if txErr != nil {
		return nil, errors.ErrInternal("评分失败")
	}

	newSum := quiz.QualitySum + sumDelta
	newCount := quiz.QualityCount + countDelta
	return &dto.QuizQualityResult{
		QualityAverage: quizQualityAverage(newSum, newCount),
		QualityCount:   newCount,
		QualityRating:  req.QualityRating,
	}, nil
}

func (s *QuizService) DeleteQuiz(userID int, canModerate bool, quizID int) *errors.AppError {
	quiz, ok := s.quizRepo.FindByID(quizID)
	if !ok {
		return errors.ErrNotFound("题目不存在")
	}
	if quiz.UserID != userID && !canModerate {
		return errors.ErrForbidden("没有删除该题目的权限")
	}

	refund := -quizCreateReward(quiz.Difficulty)
	txErr := s.quizRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.quizRepo.DeleteByID(tx, quizID); err != nil {
			return err
		}
		s.helpers.AdjustMoemoepoint(tx, quiz.UserID, refund,
			moemoepoint.ReasonContentRemoved, moemoepoint.Ref("galgame_quiz", quizID))
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("删除题目失败")
	}
	return nil
}

func (s *QuizService) ToggleQuizFavorite(userID, quizID int) *errors.AppError {
	quiz, ok := s.quizRepo.FindByID(quizID)
	if !ok {
		return errors.ErrNotFound("题目不存在")
	}
	delta := 1
	if s.quizRepo.FindQuizFavorite(quizID, userID) {
		delta = -1
	}
	txErr := s.quizRepo.DB().Transaction(func(tx *gorm.DB) error {
		if delta < 0 {
			if err := s.quizRepo.DeleteQuizFavorite(tx, quizID, userID); err != nil {
				return err
			}
		} else if err := s.quizRepo.CreateQuizFavorite(tx, quizID, userID); err != nil {
			return err
		}
		if err := s.quizRepo.AdjustQuizFavoriteCount(tx, quizID, delta); err != nil {
			return err
		}
		if userID != quiz.UserID {
			s.helpers.AdjustMoemoepoint(tx, quiz.UserID, delta,
				moemoepoint.ReasonLiked, moemoepoint.Ref("galgame_quiz", quizID))
		}
		return nil
	})
	if txErr != nil {
		return errors.ErrInternal("操作失败")
	}
	return nil
}

func (s *QuizService) UpdateQuiz(
	ctx context.Context,
	userID int, canModerate bool, req *dto.UpdateQuizRequest,
) (int, *errors.AppError) {
	req.Description = markdown.NormalizeStoredContent(req.Description)
	quiz, ok := s.quizRepo.FindByID(req.QuizID)
	if !ok {
		return 0, errors.ErrNotFound("题目不存在")
	}
	if quiz.UserID != userID && !canModerate {
		return 0, errors.ErrForbidden("没有编辑该题目的权限")
	}
	if appErr := validateQuizContent(req.Type, req.Content); appErr != nil {
		return 0, appErr
	}

	moderationText := quizAuthoringModerationText(req.Question, req.Description, req.Explanation, req.Type, req.Content)
	authorID := int64(quiz.UserID)
	decision, matched := s.check.Decision(ctx, moderationText, &authorID)
	if decision == gate.DecisionDeny {
		return 0, gate.ErrContentBlocked()
	}

	spoiler := req.SpoilerLevel
	if spoiler == "" {
		spoiler = "none"
	}
	regradable := req.Type == quiz.Type
	fields := map[string]any{
		"category":           req.Category,
		"spoiler_level":      spoiler,
		"type":               req.Type,
		"difficulty":         req.Difficulty,
		"question":           req.Question,
		"description":        req.Description,
		"content":            req.Content,
		"explanation":        req.Explanation,
		"hide_galgame":       req.HideGalgame,
		"status_update_time": gorm.Expr("now()"),
	}
	quiz.Type = req.Type
	quiz.Content = req.Content
	quiz.Difficulty = req.Difficulty

	regraded := 0
	txErr := s.quizRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.quizRepo.UpdateQuizFields(tx, req.QuizID, fields); err != nil {
			return err
		}
		if err := s.quizRepo.SetQuizGalgames(tx, req.QuizID, req.WorkIDs); err != nil {
			return err
		}
		if regradable {
			n, err := s.regradeAnswers(tx, quiz)
			if err != nil {
				return err
			}
			regraded = n
		}
		return nil
	})
	if txErr != nil {
		return 0, errors.ErrInternal("更新题目失败")
	}

	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindGalgameQuiz, "subject_id", req.QuizID, "author_id", quiz.UserID, "matched", matched)
	}
	s.scan.ScanBg(gate.SubjectKindGalgameQuiz, strconv.Itoa(req.QuizID), moderationText, int64(quiz.UserID))
	return regraded, nil
}

func (s *QuizService) regradeAnswers(tx *gorm.DB, quiz *model.GalgameQuiz) (int, error) {
	if quiz.Type == "essay" {
		return 0, nil
	}
	reward := quizCorrectReward(quiz.Difficulty)
	flipped := 0
	for _, a := range s.quizRepo.FindAnswerersForRegrade(quiz.ID) {
		grade, appErr := gradeQuiz(quiz.Type, quiz.Content, a.Submitted)
		if appErr != nil {
			continue
		}
		newCorrect := grade != nil && *grade
		wasCorrect := a.IsCorrect != nil && *a.IsCorrect
		if !newCorrect || wasCorrect {
			continue
		}
		fields := map[string]any{"is_correct": true}
		if reward > 0 && !a.Rewarded {
			fields["rewarded"] = true
			s.helpers.AdjustMoemoepoint(tx, a.UserID, reward,
				moemoepoint.ReasonContentApproved,
				moemoepoint.Ref("galgame_quiz_answer", a.ID))
		}
		if err := s.quizRepo.UpdateAnswerFields(tx, a.ID, fields); err != nil {
			return 0, err
		}
		flipped++
	}
	if flipped > 0 {
		if err := s.quizRepo.AdjustCorrectCount(tx, quiz.ID, flipped); err != nil {
			return 0, err
		}
	}
	return flipped, nil
}

func (s *QuizService) GetQuizForEdit(
	ctx context.Context, userID int, canModerate bool, quizID int,
) (*dto.QuizEditData, *errors.AppError) {
	quiz, ok := s.quizRepo.FindByID(quizID)
	if !ok {
		return nil, errors.ErrNotFound("题目不存在")
	}
	if quiz.UserID != userID && !canModerate {
		return nil, errors.ErrForbidden("没有编辑该题目的权限")
	}
	workIDs := s.quizRepo.FindQuizWorkIDs(quizID)
	return &dto.QuizEditData{
		ID:           quiz.ID,
		WorkIDs:      workIDs,
		HideGalgame:  quiz.HideGalgame,
		Category:     quiz.Category,
		Type:         quiz.Type,
		Difficulty:   quiz.Difficulty,
		SpoilerLevel: quiz.SpoilerLevel,
		Question:     quiz.Question,
		Description:  quiz.Description,
		Content:      quiz.Content,
		Explanation:  quiz.Explanation,
		Galgames:     s.galgameBriefsFor(ctx, workIDs),
	}, nil
}

func (s *QuizService) GetMyFavorites(userID int) []int {
	return s.quizRepo.FindFavoritedQuizIDs(userID)
}

const quizAnswerersLimit = 100

func (s *QuizService) GetQuizAnswers(
	ctx context.Context, quizID, viewerID int,
) []dto.QuizAnswererRecord {
	rows := s.quizRepo.FindQuizAnswerers(quizID, quizAnswerersLimit)
	if len(rows) == 0 {
		return []dto.QuizAnswererRecord{}
	}
	_, viewerEngaged := s.quizRepo.FindAnswer(quizID, viewerID)

	userIDs := make([]int, 0, len(rows))
	for _, r := range rows {
		userIDs = append(userIDs, r.UserID)
	}
	userMap := s.userClient.Hydrate(ctx, userIDs)
	records := make([]dto.QuizAnswererRecord, 0, len(rows))
	for _, r := range rows {
		u := userMap[r.UserID]
		if !userclient.IsRenderable(u) {
			continue
		}
		rec := dto.QuizAnswererRecord{
			User:      userBriefToDTO(u),
			IsCorrect: r.IsCorrect,
			Created:   r.Created,
		}
		if viewerEngaged {
			rec.Submitted = r.Submitted
		}
		records = append(records, rec)
	}
	return records
}

func (s *QuizService) galgameBriefsFor(ctx context.Context, ids []int) []dto.QuizGalgameBrief {
	out := []dto.QuizGalgameBrief{}
	if len(ids) == 0 {
		return out
	}
	m := s.fetchBriefs(ctx, ids)
	for _, id := range ids {
		b, ok := m[id]
		if !ok {
			continue
		}
		out = append(out, dto.QuizGalgameBrief{
			ID:           b.ID,
			ContentLimit: b.ContentLimit,
			Name:         b.Name,
		})
	}
	return out
}

func (s *QuizService) galgamesDetailFor(ctx context.Context, ids []int) []dto.QuizGalgameDetail {
	out := []dto.QuizGalgameDetail{}
	if len(ids) == 0 {
		return out
	}
	m := s.fetchDetailBriefs(ctx, ids, false)
	for _, id := range ids {
		b, ok := m[id]
		if !ok {
			continue
		}
		officials := b.Officials
		if officials == nil {
			officials = []string{}
		}
		out = append(out, dto.QuizGalgameDetail{
			ID:               b.ID,
			Name:             b.Name,
			ContentLimit:     b.ContentLimit,
			AgeLimit:         b.AgeLimit,
			OriginalLanguage: b.OriginalLanguage,
			Banner:           b.EffectiveBannerURL,
			BannerThumbhash:  b.EffectiveBannerThumbhash,
			Officials:        officials,
		})
	}
	return out
}

func (s *QuizService) fetchBriefs(ctx context.Context, workIDs []int) map[int]client.GalgameBrief {
	if len(workIDs) == 0 {
		return map[int]client.GalgameBrief{}
	}
	m, _ := s.galgameClient.GetBatch(ctx, workIDs)
	if m == nil {
		return map[int]client.GalgameBrief{}
	}
	return m
}

const quizGalgameSearchLimit = 12

func (s *QuizService) SearchGalgameOptions(
	ctx context.Context, keywords string, isSFW bool,
) []dto.QuizGalgameOption {
	empty := []dto.QuizGalgameOption{}
	q := url.Values{
		"q":     {keywords},
		"limit": {strconv.Itoa(quizGalgameSearchLimit)},
		"sort":  {"relevance"},
	}
	client.ApplyWorksGate(q, isSFW)
	res, appErr := s.galgameClient.CatalogWorksSearch(ctx, q)
	if appErr != nil {
		return empty
	}
	ids := make([]int, 0, quizGalgameSearchLimit)
	for i := range res.Items {
		if !client.CatalogItemRenderable(&res.Items[i]) {
			continue
		}
		if id := int(res.Items[i].ID); id > 0 {
			ids = append(ids, id)
		}
		if len(ids) >= quizGalgameSearchLimit {
			break
		}
	}
	if len(ids) == 0 {
		return empty
	}

	briefs := s.fetchDetailBriefs(ctx, ids, isSFW)
	options := make([]dto.QuizGalgameOption, 0, len(ids))
	for _, id := range ids {
		b, ok := briefs[id]
		if !ok {
			continue
		}
		officials := b.Officials
		if officials == nil {
			officials = []string{}
		}
		options = append(options, dto.QuizGalgameOption{
			ID:              b.ID,
			Name:            b.Name,
			Banner:          b.EffectiveBannerURL,
			BannerThumbhash: b.EffectiveBannerThumbhash,
			Officials:       officials,
		})
	}
	return options
}

func (s *QuizService) fetchDetailBriefs(ctx context.Context, ids []int, isSFW bool) map[int]client.GalgameDetailBrief {
	if len(ids) == 0 {
		return map[int]client.GalgameDetailBrief{}
	}
	m, _ := s.galgameClient.GetBatchDetailPublic(ctx, ids, isSFW)
	if m == nil {
		return map[int]client.GalgameDetailBrief{}
	}
	return m
}
