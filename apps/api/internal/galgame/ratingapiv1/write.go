package ratingapiv1

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"unicode/utf8"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
)

type accessTokenKey struct{}

// withAccessToken hands the caller's OAuth token to the rating writes, which
// reach NextMoe as the user.
func withAccessToken(ctx huma.Context, next func(huma.Context)) {
	next(huma.WithValue(ctx, accessTokenKey{}, middleware.GetAccessToken(humafiber.Unwrap(ctx))))
}

func accessToken(ctx context.Context) string {
	s, _ := ctx.Value(accessTokenKey{}).(string)
	return s
}

func requireToken(ctx context.Context) (string, *problem.Problem) {
	tok := accessToken(ctx)
	if tok == "" {
		return "", problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	}
	return tok, nil
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

func invalidSummary() *problem.Problem {
	return problem.New(problem.CodeValidationFailed, "short_summary is not valid UTF-8.",
		problem.AtPointer("/short_summary", problem.ReasonInvalidFormat, "short_summary must be valid UTF-8", nil))
}

// upstream maps a NextMoe answer to the rating plane. A 403 other than the
// moderation one means the forum's own credentials are wrong, not the caller.
func upstream(err error) *problem.Problem {
	var api *catalogclient.UserAPIError
	switch {
	case errors.Is(err, catalogclient.ErrUnauthorized):
		return problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	case errors.Is(err, catalogclient.ErrNotFound):
		return notFound()
	case errors.As(err, &api) && api.Status == http.StatusUnprocessableEntity:
		return problem.New(problem.CodeValidationFailed, "NextMoe refused the rating: "+api.Message)
	case errors.As(err, &api) && api.Status == http.StatusForbidden:
		return problem.Internal(err)
	default:
		return problem.Unavailable(err)
	}
}

func aspectsInput(sc *AspectScores) *catalogclient.RatingAspects {
	if sc == nil {
		return nil
	}
	return &catalogclient.RatingAspects{
		Art: sc.Art, Story: sc.Story, Music: sc.Music, Character: sc.Character,
		Route: sc.Route, System: sc.System, Voice: sc.Voice, ReplayValue: sc.ReplayValue,
	}
}

func inputOf(r *catalogclient.Rating) catalogclient.RatingInput {
	a := r.Aspects
	return catalogclient.RatingInput{
		Overall: r.Overall, AspectScores: &a, Recommend: r.Recommend,
		SpoilerLevel: r.SpoilerLevel, ShortSummary: r.ShortSummary,
	}
}

func sameInput(a, b catalogclient.RatingInput) bool {
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	return string(ja) == string(jb)
}

// mirror writes NextMoe's answer into the local copy at once, so the caller
// reads their own write; the mirror job would bring the same row within
// minutes. A failure here leaves that to the job: the write upstream stands.
func (s *Service) mirror(r *catalogclient.Rating, types []GameType) {
	if err := s.Store.MirrorRating(s.Mirror(r)); err != nil {
		slog.Error("rating write-through failed; the mirror job will bring it", "rating_id", r.ID, "error", err)
		return
	}
	if types != nil {
		if err := s.Store.SetGameTypes(int(r.ID), gameTypesJSON(types)); err != nil {
			slog.Error("rating game types not stored", "rating_id", r.ID, "error", err)
		}
	}
}

// reward moves the author's moemoepoints from what the rating earned before to
// what it earns now. The forum pays only for ratings first written here: a
// rating written on another site was never paid for, so neither its edit nor
// its deletion here moves anything.
func (s *Service) reward(userID int, before, after *catalogclient.Rating) {
	if s.Award == nil {
		return
	}
	earned := func(r *catalogclient.Rating) int {
		if r == nil {
			return 0
		}
		return ratingReward(r.ShortSummary)
	}
	ref := before
	if ref == nil {
		ref = after
	}
	if ref == nil || !s.Mirror(ref).WrittenHere {
		return
	}
	delta := earned(after) - earned(before)
	if delta == 0 {
		return
	}
	reason := moemoepoint.ReasonContentApproved
	if after == nil {
		reason = moemoepoint.ReasonContentRemoved
	}
	key := moemoepoint.Ref("galgame_rating", int(ref.ID))
	s.Award(userID, delta, reason, key, moemoepoint.KeyNonce(reason, key))
}

type createRatingInput struct {
	Body RatingCreate
}

type createRatingOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new rating, /api/v1/ratings/{rating_id}."`
	Body     Rating
}

func (s *Service) createRating(ctx context.Context, in *createRatingInput) (*createRatingOutput, error) {
	if !s.writable() {
		return nil, problem.Internal(errUnconfigured)
	}
	viewer := v1.User(ctx)
	if p := s.requireActive(ctx, viewer); p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
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
		return nil, invalidSummary()
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
	existing, err := s.Catalog.MyRating(ctx, token, int64(workID))
	if err != nil {
		return nil, upstream(err)
	}
	// A rating the caller wrote on another site is the same rating; this site
	// had none of its own, so writing one replaces it rather than conflicting.
	if existing != nil && s.Mirror(existing).WrittenHere {
		return nil, problem.New(problem.CodeAlreadyExists, "The caller has already rated this work; change that rating instead.")
	}
	if s.Sync != nil {
		s.Sync(ctx, workID, token, string(b.PlayStatus))
	}
	rating, err := s.Catalog.PutMyRating(ctx, token, int64(workID), catalogclient.RatingInput{
		Overall: b.Overall, AspectScores: aspectsInput(b.AspectScores), Recommend: string(b.Recommend),
		SpoilerLevel: string(b.SpoilerLevel), ShortSummary: summary,
	})
	if err != nil {
		return nil, upstream(err)
	}
	s.mirror(rating, b.GameTypes)
	s.reward(viewer.ID, existing, rating)
	id := int(rating.ID)
	s.Scan.ScanBg(gate.SubjectKindGalgameRating, strconv.Itoa(id), summary, int64(viewer.ID))
	out, p := s.detail(ctx, id)
	if p != nil {
		return nil, p
	}
	return &createRatingOutput{Location: "/api/v1/ratings/" + strconv.Itoa(id), Body: *out}, nil
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

func catalogModerationRequired() *problem.Problem {
	return problem.New(problem.CodeCatalogModerationRequired,
		"Deleting a rating removes it from every NextMoe site, which needs NextMoe catalog moderation standing.")
}

func permissionRequired(detail string) *problem.Problem {
	return problem.New(problem.CodePermissionRequired, detail)
}

type updateRatingInput struct {
	RatingID string `path:"rating_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Rating id."`
	Body     RatingPatch
}

// updateRating patches NextMoe's current copy, not the local one: an edit made
// on another site in the last minutes has not reached the mirror yet, and a
// PUT built from the mirror would quietly undo it.
func (s *Service) updateRating(ctx context.Context, in *updateRatingInput) (*ratingOutput, error) {
	if !s.writable() {
		return nil, problem.Internal(errUnconfigured)
	}
	viewer := v1.User(ctx)
	if p := s.requireActive(ctx, viewer); p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
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
	current, err := s.Catalog.MyRating(ctx, token, int64(r.WorkID))
	if err != nil {
		return nil, upstream(err)
	}
	if current == nil {
		return nil, notFound()
	}
	b := in.Body
	want := inputOf(current)
	if b.Recommend != nil {
		want.Recommend = string(*b.Recommend)
	}
	if b.Overall != nil {
		want.Overall = *b.Overall
	}
	if b.SpoilerLevel != nil {
		want.SpoilerLevel = string(*b.SpoilerLevel)
	}
	if b.AspectScores != nil {
		want.AspectScores = aspectsInput(b.AspectScores)
	}
	summaryChanged := b.ShortSummary != nil && *b.ShortSummary != current.ShortSummary
	if summaryChanged {
		if !utf8.ValidString(*b.ShortSummary) {
			return nil, invalidSummary()
		}
		if p := s.checkText(ctx, *b.ShortSummary, viewer.ID); p != nil {
			return nil, p
		}
		want.ShortSummary = *b.ShortSummary
	}
	// The rating carries its author's play state, so the state goes first and
	// the rating NextMoe answers afterwards already shows it.
	if b.PlayStatus != nil && s.Sync != nil {
		s.Sync(ctx, r.WorkID, token, string(*b.PlayStatus))
	}
	rating := current
	switch {
	case !sameInput(want, inputOf(current)):
		if rating, err = s.Catalog.PutMyRating(ctx, token, int64(r.WorkID), want); err != nil {
			return nil, upstream(err)
		}
	case b.PlayStatus != nil:
		if fresh, err := s.Catalog.MyRating(ctx, token, int64(r.WorkID)); err == nil && fresh != nil {
			rating = fresh
		}
	}
	s.mirror(rating, b.GameTypes)
	if summaryChanged {
		s.reward(viewer.ID, current, rating)
		s.Scan.ScanBg(gate.SubjectKindGalgameRating, strconv.FormatInt(rating.ID, 10), rating.ShortSummary, int64(viewer.ID))
	}
	out, p := s.detail(ctx, int(rating.ID))
	if p != nil {
		return nil, p
	}
	return &ratingOutput{Body: *out}, nil
}

type deleteRatingOutput struct{}

// deleteRating removes the rating from every NextMoe site. Staff delete through
// NextMoe's moderation, which judges the holder's own standing there; the
// forum's rating.delete_any alone does not reach it.
func (s *Service) deleteRating(ctx context.Context, in *ratingPathInput) (*deleteRatingOutput, error) {
	if !s.writable() {
		return nil, problem.Internal(errUnconfigured)
	}
	viewer := v1.User(ctx)
	if p := s.requireActive(ctx, viewer); p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	r, p := s.owned(ctx, in.RatingID)
	if p != nil {
		return nil, p
	}
	switch {
	case viewer.ID == r.UserID:
		if err := s.Catalog.DeleteMyRating(ctx, token, int64(r.WorkID)); err != nil {
			return nil, upstream(err)
		}
	case viewer.Can(perm.RatingDeleteAny) && !moderatesRatings(viewer):
		return nil, catalogModerationRequired()
	case moderatesRatings(viewer):
		err := s.Catalog.ModerationDeleteRating(ctx, token, int64(r.ID))
		var api *catalogclient.UserAPIError
		if errors.As(err, &api) && api.Status == http.StatusForbidden {
			return nil, catalogModerationRequired()
		}
		if err != nil && !errors.Is(err, catalogclient.ErrNotFound) {
			return nil, upstream(err)
		}
	default:
		return nil, permissionRequired("Only the author or staff holding rating.delete_any may delete a rating.")
	}
	if err := s.Store.DropRating(r.ID); err != nil {
		slog.Error("rating delete: local copy not dropped; the mirror job will drop it", "rating_id", r.ID, "error", err)
	}
	if r.WrittenHere && s.Award != nil {
		ref := moemoepoint.Ref("galgame_rating", r.ID)
		s.Award(r.UserID, -ratingReward(r.ShortSummary), moemoepoint.ReasonContentRemoved, ref, moemoepoint.KeyNonce(moemoepoint.ReasonContentRemoved, ref))
	}
	return &deleteRatingOutput{}, nil
}
