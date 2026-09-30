package apiv1

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
)

const submissionSort = "newest_desc"

type accessTokenCtxKey struct{}
type idempotencyKeyCtxKey struct{}

func withAccessToken(ctx huma.Context, next func(huma.Context)) {
	fc := humafiber.Unwrap(ctx)
	ctx = huma.WithValue(ctx, accessTokenCtxKey{}, middleware.GetAccessToken(fc))
	next(huma.WithValue(ctx, idempotencyKeyCtxKey{}, fc.Get("Idempotency-Key")))
}

func accessToken(ctx context.Context) string {
	s, _ := ctx.Value(accessTokenCtxKey{}).(string)
	return s
}

func idempotencyKey(ctx context.Context) string {
	s, _ := ctx.Value(idempotencyKeyCtxKey{}).(string)
	return s
}

func requireToken(ctx context.Context) (string, *problem.Problem) {
	tok := accessToken(ctx)
	if tok == "" {
		return "", problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	}
	return tok, nil
}

func (s *Service) requireActive(ctx context.Context) (*middleware.UserInfo, *problem.Problem) {
	user := v1.User(ctx)
	if user == nil {
		return nil, problem.New(problem.CodeMissingCredential, "The request has no credentials.")
	}
	if s.users == nil {
		return nil, problem.Internal(errUnconfigured)
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

func (s *Service) submissionsReady() *problem.Problem {
	if s == nil || s.catalog == nil {
		return problem.Unavailable(errUnconfigured)
	}
	return nil
}

func (s *Service) submissionCaller(ctx context.Context) (*middleware.UserInfo, string, *problem.Problem) {
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, "", p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, "", p
	}
	if p := s.submissionsReady(); p != nil {
		return nil, "", p
	}
	return user, token, nil
}

func (s *Service) checkSubmissionText(ctx context.Context, text string, authorID int) *problem.Problem {
	if text == "" || s.check == nil {
		return nil
	}
	id := int64(authorID)
	decision, matched := s.check.Decision(ctx, text, &id)
	if decision == gate.DecisionDeny {
		return problem.New(problem.CodeContentRejected, "The trust-and-safety check refused the submitted text. Nothing was written.")
	}
	if decision == gate.DecisionHold {
		// Trust has no subject kind for a news submission, so a hold is logged and nothing is scanned in the background.
		slog.Info("trust check hold", "author_id", authorID, "matched", matched)
	}
	return nil
}

func toNewsSubmission(in catalogclient.NewsSubmission) NewsSubmission {
	return NewsSubmission{
		Object:          "news_submission",
		ID:              repr.DecimalID(strconv.FormatInt(in.ID, 10)),
		State:           in.Status,
		NewsSource:      in.SourceKey,
		Lane:            in.Lane,
		Title:           in.Title,
		Preview:         in.Summary,
		ContentMarkdown: in.Body,
		SourceURL:       in.SourceURL,
		PublishedAt:     repr.Timestamp(in.PublishedAt),
	}
}

func (s *Service) listMyNewsSubmissions(ctx context.Context, in *listMyNewsSubmissionsInput) (*listMyNewsSubmissionsOutput, error) {
	_, token, p := s.submissionCaller(ctx)
	if p != nil {
		return nil, p
	}
	fp := collect.Fingerprint(submissionSort, strconv.Itoa(in.Limit))
	keys, curErr := collect.DecodeCursor(in.Cursor, submissionSort, fp)
	if curErr != nil {
		return nil, curErr
	}
	upstreamCursor := ""
	if keys != nil {
		if len(keys) != 1 || keys[0] == "" {
			return nil, invalidCursor()
		}
		upstreamCursor = keys[0]
	}
	items, next, err := s.catalog.ListMyNews(ctx, token, upstreamCursor, in.Limit)
	if err != nil {
		return nil, mapSubmissionUpstream(err)
	}
	out := make([]NewsSubmission, 0, len(items))
	for _, it := range items {
		out = append(out, toNewsSubmission(it))
	}
	var nextCur *string
	if next != "" {
		cur := collect.EncodeCursor(submissionSort, fp, next)
		nextCur = &cur
	}
	return &listMyNewsSubmissionsOutput{Body: repr.NewList(out, nextCur)}, nil
}

func (s *Service) getMyNewsSubmission(ctx context.Context, in *getMyNewsSubmissionInput) (*getMyNewsSubmissionOutput, error) {
	_, token, p := s.submissionCaller(ctx)
	if p != nil {
		return nil, p
	}
	id, ok := parseNewsItemID(in.NewsSubmissionID)
	if !ok {
		return nil, notFound()
	}
	sub, err := s.catalog.GetMyNews(ctx, token, id)
	if err != nil {
		return nil, mapSubmissionUpstream(err)
	}
	return &getMyNewsSubmissionOutput{Body: toNewsSubmission(*sub)}, nil
}

func (s *Service) createNewsSubmission(ctx context.Context, in *createNewsSubmissionInput) (*createNewsSubmissionOutput, error) {
	user, token, p := s.submissionCaller(ctx)
	if p != nil {
		return nil, p
	}
	title := strings.TrimSpace(in.Body.Title)
	preview := strings.TrimSpace(in.Body.Preview)
	sourceURL := strings.TrimSpace(in.Body.SourceURL)
	var fields []problem.FieldError
	if title == "" {
		fields = append(fields, problem.AtPointer("/title", problem.ReasonRequired, "title is blank", nil))
	}
	if preview == "" {
		fields = append(fields, problem.AtPointer("/preview", problem.ReasonRequired, "preview is blank", nil))
	}
	if strings.TrimSpace(in.Body.ContentMarkdown) == "" && sourceURL == "" {
		fields = append(fields, problem.AtPointer("/content_markdown", problem.ReasonRequired, "send content_markdown or source_url", nil))
	}
	if len(fields) > 0 {
		return nil, validationFailed(fields...)
	}
	if p := s.checkSubmissionText(ctx, gate.ComposeText(title, preview, in.Body.ContentMarkdown), user.ID); p != nil {
		return nil, p
	}
	sub, err := s.catalog.CreateMyNews(ctx, token, catalogclient.NewsSubmissionWrite{
		Title:     title,
		Summary:   preview,
		Body:      in.Body.ContentMarkdown,
		SourceURL: sourceURL,
	}, idempotencyKey(ctx))
	if err != nil {
		return nil, mapSubmissionUpstream(err)
	}
	out := toNewsSubmission(*sub)
	return &createNewsSubmissionOutput{
		Location: "/api/v1/me/news-submissions/" + string(out.ID),
		Body:     out,
	}, nil
}

func (s *Service) updateNewsSubmission(ctx context.Context, in *updateNewsSubmissionInput) (*getMyNewsSubmissionOutput, error) {
	user, token, p := s.submissionCaller(ctx)
	if p != nil {
		return nil, p
	}
	id, ok := parseNewsItemID(in.NewsSubmissionID)
	if !ok {
		return nil, notFound()
	}
	b := in.Body
	editing := b.Title != nil || b.Preview != nil || b.ContentMarkdown != nil || b.SourceURL != nil
	if b.State == nil && !editing {
		return nil, validationFailed(problem.AtPointer("", problem.ReasonRequired, "send state=withdrawn or at least one field to edit", nil))
	}
	if b.State != nil && editing {
		return nil, validationFailed(problem.AtPointer("/state", problem.ReasonNotAllowedValue, "withdraw on its own", nil))
	}
	if b.State != nil {
		sub, err := s.catalog.PatchMyNews(ctx, token, id, catalogclient.NewsSubmissionPatch{Withdraw: true})
		if err != nil {
			return nil, mapSubmissionUpstream(err)
		}
		return &getMyNewsSubmissionOutput{Body: toNewsSubmission(*sub)}, nil
	}
	var fields []problem.FieldError
	if b.Title != nil && strings.TrimSpace(*b.Title) == "" {
		fields = append(fields, problem.AtPointer("/title", problem.ReasonRequired, "title is blank", nil))
	}
	if b.Preview != nil && strings.TrimSpace(*b.Preview) == "" {
		fields = append(fields, problem.AtPointer("/preview", problem.ReasonRequired, "preview is blank", nil))
	}
	if len(fields) > 0 {
		return nil, validationFailed(fields...)
	}
	if b.ContentMarkdown != nil || b.SourceURL != nil {
		cur, err := s.catalog.GetMyNews(ctx, token, id)
		if err != nil {
			return nil, mapSubmissionUpstream(err)
		}
		content := cur.Body
		if b.ContentMarkdown != nil {
			content = *b.ContentMarkdown
		}
		source := cur.SourceURL
		if b.SourceURL != nil {
			source = *b.SourceURL
		}
		if strings.TrimSpace(content) == "" && strings.TrimSpace(source) == "" {
			return nil, validationFailed(problem.AtPointer("/content_markdown", problem.ReasonRequired, "send content_markdown or source_url", nil))
		}
	}
	var parts []string
	if b.Title != nil {
		parts = append(parts, strings.TrimSpace(*b.Title))
	}
	if b.Preview != nil {
		parts = append(parts, strings.TrimSpace(*b.Preview))
	}
	if b.ContentMarkdown != nil {
		parts = append(parts, *b.ContentMarkdown)
	}
	if p := s.checkSubmissionText(ctx, gate.ComposeText(parts...), user.ID); p != nil {
		return nil, p
	}
	patch := catalogclient.NewsSubmissionPatch{}
	if b.Title != nil {
		v := strings.TrimSpace(*b.Title)
		patch.Title = &v
	}
	if b.Preview != nil {
		v := strings.TrimSpace(*b.Preview)
		patch.Summary = &v
	}
	if b.ContentMarkdown != nil {
		patch.Body = b.ContentMarkdown
	}
	if b.SourceURL != nil {
		v := strings.TrimSpace(*b.SourceURL)
		patch.SourceURL = &v
	}
	sub, err := s.catalog.PatchMyNews(ctx, token, id, patch)
	if err != nil {
		return nil, mapSubmissionUpstream(err)
	}
	return &getMyNewsSubmissionOutput{Body: toNewsSubmission(*sub)}, nil
}
