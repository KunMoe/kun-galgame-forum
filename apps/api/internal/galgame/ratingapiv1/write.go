package ratingapiv1

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"unicode/utf8"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
	"gorm.io/gorm"
)

type accessTokenKey struct{}

// withAccessToken hands the caller's OAuth token to the play-status sync, which
// writes the work state to catalog as the user.
func withAccessToken(ctx huma.Context, next func(huma.Context)) {
	next(huma.WithValue(ctx, accessTokenKey{}, middleware.GetAccessToken(humafiber.Unwrap(ctx))))
}

func accessToken(ctx context.Context) string {
	s, _ := ctx.Value(accessTokenKey{}).(string)
	return s
}

func ratingReward(summary string) int {
	n := len(summary)
	switch {
	case n >= constants.RatingLenThresholdHigh:
		return constants.RatingRewardHigh
	case n >= constants.RatingLenThresholdMedium:
		return constants.RatingRewardMedium
	default:
		return constants.RatingRewardLow
	}
}

func scoreValue(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func gameTypesJSON(types []GameType) json.RawMessage {
	keys := make([]string, len(types))
	for i, t := range types {
		keys[i] = string(t)
	}
	raw, _ := json.Marshal(keys)
	return raw
}

func (s *Service) checkText(ctx context.Context, text string, authorID int) *problem.Problem {
	if text == "" {
		return nil
	}
	id := int64(authorID)
	decision, matched := s.Check.Decision(ctx, text, &id)
	if decision == gate.DecisionDeny {
		return problem.New(problem.CodeContentRejected, "The trust-and-safety check refused the submitted text. Nothing was written.")
	}
	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindGalgameRating, "author_id", authorID, "matched", matched)
	}
	return nil
}

func unknownWork() *problem.Problem {
	return problem.New(problem.CodeValidationFailed, "No work with that id is visible.",
		problem.AtPointer("/work_id", problem.ReasonUnknownReference, "work_id names no work catalog shows", nil))
}

type createRatingInput struct {
	Body RatingCreate
}

type createRatingOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new rating, /api/v1/ratings/{rating_id}."`
	Body     Rating
}

func (s *Service) createRating(ctx context.Context, in *createRatingInput) (*createRatingOutput, error) {
	if !s.ready() {
		return nil, problem.Internal(errUnconfigured)
	}
	viewer := v1.User(ctx)
	if p := s.requireActive(ctx, viewer); p != nil {
		return nil, p
	}
	b := in.Body
	workID, ok := repr.ParseID(b.WorkID)
	if !ok {
		return nil, unknownWork()
	}
	summary := ""
	if b.ShortSummary != nil {
		summary = *b.ShortSummary
	}
	if !utf8.ValidString(summary) {
		return nil, problem.New(problem.CodeValidationFailed, "short_summary is not valid UTF-8.",
			problem.AtPointer("/short_summary", problem.ReasonInvalidFormat, "short_summary must be valid UTF-8", nil))
	}
	works, p := s.visibleWorks(ctx, []int{workID})
	if p != nil {
		return nil, p
	}
	if _, ok := works[workID]; !ok {
		return nil, unknownWork()
	}
	if p := s.checkText(ctx, summary, viewer.ID); p != nil {
		return nil, p
	}
	scores := b.AspectScores
	if scores == nil {
		scores = &AspectScores{}
	}
	row := &model.GalgameRating{
		WorkID: workID, UserID: viewer.ID,
		Recommend: string(b.Recommend), Overall: b.Overall, GalgameType: gameTypesJSON(b.GameTypes),
		PlayStatus: string(b.PlayStatus), ShortSummary: summary, SpoilerLevel: string(b.SpoilerLevel),
		Art: scoreValue(scores.Art), Story: scoreValue(scores.Story), Music: scoreValue(scores.Music),
		Character: scoreValue(scores.Character), Route: scoreValue(scores.Route), System: scoreValue(scores.System),
		Voice: scoreValue(scores.Voice), ReplayValue: scoreValue(scores.ReplayValue),
	}
	err := s.Store.Create(row)
	if errors.Is(err, repository.ErrRatingExists) {
		return nil, problem.New(problem.CodeAlreadyExists, "The caller has already rated this work; change that rating instead.")
	}
	if err != nil {
		return nil, problem.Internal(err)
	}
	if s.Award != nil {
		ref := moemoepoint.Ref("galgame_rating", row.ID)
		s.Award(viewer.ID, ratingReward(summary), moemoepoint.ReasonContentApproved, ref, moemoepoint.KeyNonce(moemoepoint.ReasonContentApproved, ref))
	}
	s.Scan.ScanBg(gate.SubjectKindGalgameRating, strconv.Itoa(row.ID), summary, int64(viewer.ID))
	if s.Sync != nil {
		s.Sync(ctx, workID, accessToken(ctx), row.PlayStatus)
	}
	rating, p := s.detail(ctx, row.ID)
	if p != nil {
		return nil, p
	}
	return &createRatingOutput{Location: "/api/v1/ratings/" + strconv.Itoa(row.ID), Body: *rating}, nil
}

func (s *Service) owned(ctx context.Context, rawID string) (repository.RatingRecord, *problem.Problem) {
	id, ok := repr.ParseID(repr.DecimalID(rawID))
	if !ok {
		return repository.RatingRecord{}, notFound()
	}
	r, err := s.Store.Get(id)
	if errors.Is(err, repository.ErrRatingNotFound) {
		return repository.RatingRecord{}, notFound()
	}
	if err != nil {
		return repository.RatingRecord{}, problem.Internal(err)
	}
	return r, nil
}

func (s *Service) readable(ctx context.Context, r repository.RatingRecord) *problem.Problem {
	users, err := s.Users.Users(ctx, []int{r.UserID})
	if err != nil {
		return problem.Unavailable(err)
	}
	if _, ok := s.authorRef(users, r.UserID); !ok {
		return notFound()
	}
	works, p := s.visibleWorks(ctx, []int{r.WorkID})
	if p != nil {
		return p
	}
	if _, ok := works[r.WorkID]; !ok {
		return notFound()
	}
	return nil
}

func permissionRequired(detail string) *problem.Problem {
	return problem.New(problem.CodePermissionRequired, detail)
}

type updateRatingInput struct {
	RatingID string `path:"rating_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Rating id."`
	Body     RatingPatch
}

func (s *Service) updateRating(ctx context.Context, in *updateRatingInput) (*ratingOutput, error) {
	if !s.ready() {
		return nil, problem.Internal(errUnconfigured)
	}
	viewer := v1.User(ctx)
	if p := s.requireActive(ctx, viewer); p != nil {
		return nil, p
	}
	r, p := s.owned(ctx, in.RatingID)
	if p != nil {
		return nil, p
	}
	if p := s.readable(ctx, r); p != nil {
		return nil, p
	}
	if r.UserID != viewer.ID {
		return nil, permissionRequired("Only the author may change a rating.")
	}
	b := in.Body
	fields := map[string]any{}
	if b.Recommend != nil {
		fields["recommend"] = string(*b.Recommend)
	}
	if b.Overall != nil {
		fields["overall"] = *b.Overall
	}
	if b.GameTypes != nil {
		fields["galgame_type"] = gorm.Expr("?::jsonb", string(gameTypesJSON(b.GameTypes)))
	}
	if b.PlayStatus != nil {
		fields["play_status"] = string(*b.PlayStatus)
	}
	if b.SpoilerLevel != nil {
		fields["spoiler_level"] = string(*b.SpoilerLevel)
	}
	summaryChanged := b.ShortSummary != nil && *b.ShortSummary != r.ShortSummary
	if summaryChanged {
		if p := s.checkText(ctx, *b.ShortSummary, viewer.ID); p != nil {
			return nil, p
		}
		fields["short_summary"] = *b.ShortSummary
	}
	if sc := b.AspectScores; sc != nil {
		fields["art"], fields["story"], fields["music"] = scoreValue(sc.Art), scoreValue(sc.Story), scoreValue(sc.Music)
		fields["character"], fields["route"], fields["system"] = scoreValue(sc.Character), scoreValue(sc.Route), scoreValue(sc.System)
		fields["voice"], fields["replay_value"] = scoreValue(sc.Voice), scoreValue(sc.ReplayValue)
	}
	if len(fields) > 0 {
		if err := s.Store.Update(r.ID, fields); err != nil {
			return nil, problem.Internal(err)
		}
	}
	if summaryChanged {
		if diff := ratingReward(*b.ShortSummary) - ratingReward(r.ShortSummary); diff != 0 && s.Award != nil {
			ref := moemoepoint.Ref("galgame_rating", r.ID)
			s.Award(r.UserID, diff, moemoepoint.ReasonContentApproved, ref, moemoepoint.KeyNonce(moemoepoint.ReasonContentApproved, ref))
		}
		s.Scan.ScanBg(gate.SubjectKindGalgameRating, strconv.Itoa(r.ID), *b.ShortSummary, int64(r.UserID))
	}
	if b.PlayStatus != nil && s.Sync != nil {
		s.Sync(ctx, r.WorkID, accessToken(ctx), string(*b.PlayStatus))
	}
	rating, p := s.detail(ctx, r.ID)
	if p != nil {
		return nil, p
	}
	return &ratingOutput{Body: *rating}, nil
}

type deleteRatingOutput struct{}

func (s *Service) deleteRating(ctx context.Context, in *ratingPathInput) (*deleteRatingOutput, error) {
	if !s.ready() {
		return nil, problem.Internal(errUnconfigured)
	}
	viewer := v1.User(ctx)
	if p := s.requireActive(ctx, viewer); p != nil {
		return nil, p
	}
	r, p := s.owned(ctx, in.RatingID)
	if p != nil {
		return nil, p
	}
	creators, err := s.Store.WorkCreators([]int{r.WorkID})
	if err != nil {
		return nil, problem.Internal(err)
	}
	if !canDelete(viewer, r, creators[r.WorkID]) {
		return nil, permissionRequired("Only the author, the work page's creator or staff holding rating.delete_any may delete a rating.")
	}
	if err := s.Store.Delete(r.ID); err != nil {
		return nil, problem.Internal(err)
	}
	if s.Award != nil {
		ref := moemoepoint.Ref("galgame_rating", r.ID)
		s.Award(r.UserID, -ratingReward(r.ShortSummary), moemoepoint.ReasonContentRemoved, ref, moemoepoint.KeyNonce(moemoepoint.ReasonContentRemoved, ref))
	}
	return &deleteRatingOutput{}, nil
}

type engagementOutput struct {
	Body RatingEngagement
}

func (s *Service) likeRating(ctx context.Context, in *ratingPathInput) (*engagementOutput, error) {
	return s.setLike(ctx, in.RatingID, true)
}

func (s *Service) unlikeRating(ctx context.Context, in *ratingPathInput) (*engagementOutput, error) {
	return s.setLike(ctx, in.RatingID, false)
}

func (s *Service) setLike(ctx context.Context, rawID string, want bool) (*engagementOutput, error) {
	if !s.ready() {
		return nil, problem.Internal(errUnconfigured)
	}
	viewer := v1.User(ctx)
	if p := s.requireActive(ctx, viewer); p != nil {
		return nil, p
	}
	r, p := s.owned(ctx, rawID)
	if p != nil {
		return nil, p
	}
	if p := s.readable(ctx, r); p != nil {
		return nil, p
	}
	if r.UserID == viewer.ID {
		return nil, problem.New(problem.CodeSelfLikeForbidden, "Users cannot like their own ratings.")
	}
	likeID := 0
	err := s.Store.DB().Transaction(func(tx *gorm.DB) error {
		var err error
		likeID, err = s.Store.SetLike(tx, r.ID, viewer.ID, want)
		if err != nil || likeID == 0 || !want || s.Notify == nil {
			return err
		}
		preview := r.ShortSummary
		if utf8.RuneCountInString(preview) > constants.TextPreviewLength {
			preview = string([]rune(preview)[:constants.TextPreviewLength])
		}
		return s.Notify(tx, viewer.ID, r.UserID, preview, r.WorkID)
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	if likeID > 0 && s.Award != nil {
		delta, event := 1, "liked"
		if !want {
			delta, event = -1, "unliked"
		}
		s.Award(r.UserID, delta, moemoepoint.ReasonLiked, moemoepoint.Ref("galgame_rating", r.ID),
			moemoepoint.Key(event, "galgame_rating_like_"+strconv.Itoa(likeID)))
	}
	fresh, err := s.Store.Get(r.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &engagementOutput{Body: RatingEngagement{
		Object:    "rating_engagement",
		RatingID:  repr.ID(r.ID),
		LikeCount: max(fresh.LikeCount, 0),
		Viewer:    RatingEngagementViewer{HasLiked: want},
	}}, nil
}
