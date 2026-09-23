package apiv1

import (
	"context"
	"errors"
	"strconv"
	"time"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/quiz/repository"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

func (s *Service) listQuizzes(ctx context.Context, in *listQuizzesInput) (*listQuizzesOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	pg := pageOf(in.Page, in.Limit)
	if p := pg.CheckDepth(); p != nil {
		return nil, p
	}
	sort := sortSpecs[in.Sort]
	filter := repository.ListFilter{
		Type: in.QuizType, Category: in.QuizCategory, Spoiler: in.SpoilerLevel,
		IncludeNSFW: in.IncludeNSFW,
	}
	filter.Difficulty = in.Difficulty
	if in.WorkID != "" {
		id, ok := parseID(in.WorkID)
		if !ok {
			return nil, problem.New(problem.CodeInvalidParameter, "work_id must be a positive integer.",
				problem.AtParameter("work_id", problem.ReasonInvalidFormat, "work_id must be a positive integer", nil))
		}
		filter.WorkID = id
	}
	if in.AuthorID != "" {
		id, ok := parseID(in.AuthorID)
		if !ok {
			return nil, problem.New(problem.CodeInvalidParameter, "author_id must be a positive integer.",
				problem.AtParameter("author_id", problem.ReasonInvalidFormat, "author_id must be a positive integer", nil))
		}
		filter.AuthorID = id
	}
	authors, err := s.store.DistinctAuthors(filter)
	if err != nil {
		return nil, problem.Internal(err)
	}
	keep, p := s.keepAuthors(ctx, authors)
	if p != nil {
		return nil, p
	}
	filter.AuthorIDs = keep
	total, err := s.store.Count(filter)
	if err != nil {
		return nil, problem.Internal(err)
	}
	n, rel := collect.ClampTotal(total)
	rows, err := s.store.List(filter, sort, pg.Offset(), pg.Limit)
	if err != nil {
		return nil, problem.Internal(err)
	}
	items, p := s.summaries(ctx, rows, v1.User(ctx))
	if p != nil {
		return nil, p
	}
	return &listQuizzesOutput{Body: repr.NewPageList(items, n, rel)}, nil
}

func (s *Service) getQuiz(ctx context.Context, in *quizIDInput) (*quizOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	row, users, p := s.visibleQuiz(ctx, in.QuizID)
	if p != nil {
		return nil, p
	}
	if p := s.bumpView(row.ID); p != nil {
		return nil, p
	}
	row.View++
	out, p := s.detail(ctx, row, users, v1.User(ctx))
	if p != nil {
		return nil, p
	}
	return &quizOutput{Body: *out}, nil
}

func (s *Service) getQuizSource(ctx context.Context, in *quizIDInput) (*sourceOutput, error) {
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
	if !canEditQuiz(row.UserID, user) {
		return nil, permissionRequired()
	}
	workIDs, err := s.store.WorkIDs(row.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	src, p := s.sourceOf(row, workIDs)
	if p != nil {
		return nil, p
	}
	return &sourceOutput{Body: src}, nil
}

func (s *Service) createQuiz(ctx context.Context, in *createQuizInput) (*createQuizOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	body := in.Body
	var errs []problem.FieldError
	if len(body.PromptText) > maxPrompt {
		errs = append(errs, tooLong("/prompt_text", maxPrompt))
	}
	prompt := trimSpace(body.PromptText)
	if prompt == "" {
		errs = append(errs, tooShort("/prompt_text", 1))
	}
	desc := markdown.NormalizeStoredContent(body.DescriptionMarkdown)
	if len(body.DescriptionMarkdown) > maxDescription {
		errs = append(errs, tooLong("/description_markdown", maxDescription))
	}
	expl := markdown.NormalizeStoredContent(body.ExplanationMarkdown)
	if len(body.ExplanationMarkdown) > maxExplanation {
		errs = append(errs, tooLong("/explanation_markdown", maxExplanation))
	}
	spoiler := body.SpoilerLevel
	if spoiler == "" {
		spoiler = "none"
	}
	choices, cerrs := validateChoices(body.QuizType, body.Choices, true)
	errs = append(errs, cerrs...)
	errs = append(errs, validateIndexes(body.QuizType, toInts(body.CorrectChoiceIndexes), len(choices), body.CorrectChoiceIndexes != nil || body.QuizType == quizTypeJudge)...)
	errs = append(errs, validateJudge(body.QuizType, body.JudgeAnswer, true)...)
	workIDs, werrs := parseWorkIDs(body.WorkIDs, "/work_ids")
	errs = append(errs, werrs...)
	if workIDs == nil {
		workIDs = []int{}
	}
	if len(errs) > 0 {
		return nil, validationFailed(errs...)
	}
	if werrs, p := s.checkWorksExistOrUnavail(ctx, workIDs); p != nil {
		return nil, p
	} else if len(werrs) > 0 {
		return nil, validationFailed(werrs...)
	}
	raw, err := encodeContent(body.QuizType, choices, toInts(body.CorrectChoiceIndexes), body.JudgeAnswer)
	if err != nil {
		return nil, problem.Internal(err)
	}
	moderation := gate.ComposeText(append([]string{prompt, desc, expl}, choices...)...)
	if _, _, p := s.rejectContent(ctx, moderation, user.ID); p != nil {
		return nil, p
	}
	row := model.GalgameQuiz{
		UserID: user.ID, Category: body.QuizCategory, SpoilerLevel: spoiler,
		Type: body.QuizType, Difficulty: body.Difficulty, Question: prompt,
		Description: desc, Content: raw, Explanation: expl, HideGalgame: body.IsWorkHidden,
		StatusUpdateTime: time.Now(),
	}
	err = s.store.InTx(func(tx *gorm.DB) error {
		if err := s.store.Create(tx, &row); err != nil {
			return err
		}
		if err := s.store.SetWorks(tx, row.ID, workIDs); err != nil {
			return err
		}
		return s.store.CreateAuthorRow(tx, row.ID, user.ID)
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	s.award(user.ID, 2, moemoepoint.ReasonContentApproved,
		moemoepoint.Ref("galgame_quiz", row.ID),
		moemoepoint.Key("quiz_create", strconv.Itoa(row.ID)))
	s.scan.ScanBg(gate.SubjectKindGalgameQuiz, strconv.Itoa(row.ID), moderation, int64(user.ID))
	created, err := s.store.Find(row.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	users, p := s.lookupUsers(ctx, []int{created.UserID})
	if p != nil {
		return nil, p
	}
	out, p := s.detail(ctx, created, users, user)
	if p != nil {
		return nil, p
	}
	return &createQuizOutput{Location: "/api/v1/quizzes/" + strconv.Itoa(created.ID), Body: *out}, nil
}

func (s *Service) updateQuiz(ctx context.Context, in *patchQuizInput) (*quizOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	row, users, p := s.visibleQuiz(ctx, in.QuizID)
	if p != nil {
		return nil, p
	}
	if !canEditQuiz(row.UserID, user) {
		return nil, permissionRequired()
	}
	patch := in.Body
	if patch.PromptText == nil && patch.DescriptionMarkdown == nil && patch.ExplanationMarkdown == nil &&
		patch.QuizType == nil && patch.QuizCategory == nil && patch.Difficulty == nil &&
		patch.SpoilerLevel == nil && patch.Choices == nil && patch.CorrectChoiceIndexes == nil &&
		patch.JudgeAnswer == nil && patch.WorkIDs == nil && patch.IsWorkHidden == nil {
		out, p := s.detail(ctx, row, users, user)
		if p != nil {
			return nil, p
		}
		return &quizOutput{Body: *out}, nil
	}

	var errs []problem.FieldError
	fields := map[string]any{}
	var modParts []string
	qtype := row.Type
	if patch.QuizType != nil {
		if *patch.QuizType != row.Type {
			return nil, validationFailed(immutable("/quiz_type"))
		}
	}

	prompt := row.Question
	if patch.PromptText != nil {
		if len(*patch.PromptText) > maxPrompt {
			errs = append(errs, tooLong("/prompt_text", maxPrompt))
		} else {
			prompt = trimSpace(*patch.PromptText)
			if prompt == "" {
				errs = append(errs, tooShort("/prompt_text", 1))
			} else {
				fields["question"] = prompt
				modParts = append(modParts, prompt)
			}
		}
	}
	desc := row.Description
	if patch.DescriptionMarkdown != nil {
		if len(*patch.DescriptionMarkdown) > maxDescription {
			errs = append(errs, tooLong("/description_markdown", maxDescription))
		} else {
			desc = markdown.NormalizeStoredContent(*patch.DescriptionMarkdown)
			fields["description"] = desc
			modParts = append(modParts, desc)
		}
	}
	expl := row.Explanation
	if patch.ExplanationMarkdown != nil {
		if len(*patch.ExplanationMarkdown) > maxExplanation {
			errs = append(errs, tooLong("/explanation_markdown", maxExplanation))
		} else {
			expl = markdown.NormalizeStoredContent(*patch.ExplanationMarkdown)
			fields["explanation"] = expl
			modParts = append(modParts, expl)
		}
	}
	if patch.QuizCategory != nil {
		fields["category"] = *patch.QuizCategory
	}
	if patch.Difficulty != nil {
		fields["difficulty"] = *patch.Difficulty
	}
	if patch.SpoilerLevel != nil {
		fields["spoiler_level"] = *patch.SpoilerLevel
	}
	if patch.IsWorkHidden != nil {
		fields["hide_galgame"] = *patch.IsWorkHidden
	}

	oldChoices, err := parseChoices(qtype, row.Content)
	if err != nil {
		return nil, problem.Internal(err)
	}
	oldIndexes, oldJudge, err := parseKey(qtype, row.Content)
	if err != nil {
		return nil, problem.Internal(err)
	}
	choices := oldChoices
	if patch.Choices != nil {
		cleaned, cerrs := validateChoices(qtype, patch.Choices, true)
		errs = append(errs, cerrs...)
		if len(cerrs) == 0 {
			choices = cleaned
			modParts = append(modParts, cleaned...)
		}
	}
	indexes := oldIndexes
	indexesPresent := patch.CorrectChoiceIndexes != nil
	if indexesPresent {
		indexes = toInts(patch.CorrectChoiceIndexes)
	}
	judge := oldJudge
	if patch.JudgeAnswer != nil {
		judge = patch.JudgeAnswer
	}
	if patch.Choices != nil || indexesPresent || patch.JudgeAnswer != nil {
		idxPresent := indexesPresent || qtype == quizTypeJudge || patch.Choices == nil
		if patch.Choices != nil && !indexesPresent && qtype != quizTypeJudge {
			idxPresent = true
		}
		errs = append(errs, validateIndexes(qtype, indexes, len(choices), idxPresent)...)
		errs = append(errs, validateJudge(qtype, judge, qtype == quizTypeJudge && patch.JudgeAnswer != nil)...)
	}

	var workIDs []int
	replaceWorks := patch.WorkIDs != nil
	if replaceWorks {
		ids, werrs := parseWorkIDs(patch.WorkIDs, "/work_ids")
		errs = append(errs, werrs...)
		workIDs = ids
		if workIDs == nil {
			workIDs = []int{}
		}
	}
	if len(errs) > 0 {
		return nil, validationFailed(errs...)
	}
	if replaceWorks {
		if werrs, p := s.checkWorksExistOrUnavail(ctx, workIDs); p != nil {
			return nil, p
		} else if len(werrs) > 0 {
			return nil, validationFailed(werrs...)
		}
	}

	newRaw := row.Content
	if patch.Choices != nil || indexesPresent || patch.JudgeAnswer != nil {
		encoded, err := encodeContent(qtype, choices, indexes, judge)
		if err != nil {
			return nil, problem.Internal(err)
		}
		newRaw = encoded
		fields["content"] = newRaw
	}

	if len(modParts) > 0 {
		moderation := gate.ComposeText(modParts...)
		if _, _, p := s.rejectContent(ctx, moderation, row.UserID); p != nil {
			return nil, p
		}
	}

	keyChanged := !bytesEqual(row.Content, newRaw)
	now := time.Now()
	fields["updated"] = now
	fields["status_update_time"] = now

	err = s.store.InTx(func(tx *gorm.DB) error {
		if err := s.store.Patch(tx, row.ID, fields); err != nil {
			return err
		}
		if replaceWorks {
			if err := s.store.SetWorks(tx, row.ID, workIDs); err != nil {
				return err
			}
		}
		if keyChanged {
			answers, err := s.store.AnswerersForRegrade(tx, row.ID)
			if err != nil {
				return err
			}
			for _, a := range answers {
				ok, gerr := grade(qtype, newRaw, a.Submitted)
				if gerr != nil {
					return gerr
				}
				if err := s.store.SetAnswerCorrect(tx, a.ID, ok); err != nil {
					return err
				}
			}
			if err := s.store.RecountCorrect(tx, row.ID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, notFound()
		}
		if p := txProblem(err); p != nil {
			return nil, p
		}
		return nil, problem.Internal(err)
	}
	if len(modParts) > 0 {
		s.scan.ScanBg(gate.SubjectKindGalgameQuiz, strconv.Itoa(row.ID), gate.ComposeText(modParts...), int64(row.UserID))
	}
	fresh, err := s.store.Find(row.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	out, p := s.detail(ctx, fresh, users, user)
	if p != nil {
		return nil, p
	}
	return &quizOutput{Body: *out}, nil
}

func (s *Service) deleteQuiz(ctx context.Context, in *quizIDInput) (*noContentOutput, error) {
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
	if !canDeleteQuiz(row.UserID, user) {
		return nil, permissionRequired()
	}
	if err := s.store.InTx(func(tx *gorm.DB) error {
		return s.store.Delete(tx, row.ID)
	}); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, notFound()
		}
		return nil, problem.Internal(err)
	}
	s.award(row.UserID, -2, moemoepoint.ReasonContentRemoved,
		moemoepoint.Ref("galgame_quiz", row.ID),
		moemoepoint.Key("quiz_delete", strconv.Itoa(row.ID)))
	return &noContentOutput{}, nil
}
