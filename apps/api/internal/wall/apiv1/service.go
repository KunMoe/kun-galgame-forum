package apiv1

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/wall/repository"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

type AwardFunc func(userID, delta int, reason, ref, key string)

type GalgameResolver func(ctx context.Context, workID int) (bool, error)

type Service struct {
	store     *repository.Store
	community *communityclient.Client
	users     *userclient.Client
	convert   *content.Converter
	galgame   GalgameResolver
	award     AwardFunc
	cdn       string
}

func New(
	store *repository.Store,
	community *communityclient.Client,
	users *userclient.Client,
	convert *content.Converter,
	galgame GalgameResolver,
	award AwardFunc,
	cdn string,
) *Service {
	return &Service{
		store: store, community: community, users: users,
		convert: convert, galgame: galgame, award: award, cdn: cdn,
	}
}

var errUnconfigured = errors.New("apiv1 walls: service is not configured")

func (s *Service) ready() bool {
	return s != nil && s.store != nil && s.community != nil && s.users != nil && s.convert != nil && s.galgame != nil
}

type subject struct {
	spec        subjectSpec
	id          int
	ownerID     int
	websiteSlug string
	isNSFW      bool
}

func notFound() *problem.Problem {
	return problem.New(problem.CodeNotFound, "Nothing visible exists at this URL.")
}

func quizAnswerRequired() *problem.Problem {
	return problem.New(problem.CodeQuizAnswerRequired, "The quiz hides its game or carries spoilers; answer it to read and write its comment wall.")
}

func permissionRequired() *problem.Problem {
	return problem.New(problem.CodePermissionRequired, "The token lacks the permission this decision needs.")
}

func tombstoned() *problem.Problem {
	return problem.New(problem.CodeInvalidStateTransition, "The comment is deleted; a tombstone cannot be edited, liked or flagged.")
}

func validationFailed(fields ...problem.FieldError) *problem.Problem {
	return problem.New(problem.CodeValidationFailed, "The request is syntactically valid but semantically not.", fields...)
}

func viewerID(u *middleware.UserInfo) int {
	if u == nil {
		return 0
	}
	return u.ID
}

func (s *Service) resolveSubject(ctx context.Context, spec subjectSpec, id int, viewer *middleware.UserInfo) (*subject, *problem.Problem) {
	sub := &subject{spec: spec, id: id}
	var err error
	switch spec.typ {
	case "galgame":
		found, gerr := s.galgame(ctx, id)
		if gerr != nil {
			return nil, problem.Unavailable(gerr)
		}
		if !found {
			return nil, notFound()
		}
		return sub, nil
	case "website":
		w, werr := s.store.Website(id)
		if werr != nil {
			return nil, storeProblem(werr)
		}
		sub.websiteSlug = w.URL
		sub.isNSFW = w.AgeLimit != "all"
		return sub, nil
	case "galgame_rating":
		sub.ownerID, err = s.store.RatingAuthor(id)
	case "galgame_resource":
		sub.ownerID, err = s.store.ResourceOwner(id)
	case "toolset":
		sub.ownerID, err = s.store.ToolsetOwner(id)
	case "galgame_quiz":
		q, qerr := s.store.Quiz(id)
		if qerr != nil {
			return nil, storeProblem(qerr)
		}
		sub.ownerID = q.UserID
		if p := s.userRenderable(ctx, sub.ownerID); p != nil {
			return nil, p
		}
		if q.HideGalgame || (q.SpoilerLevel != "" && q.SpoilerLevel != "none") {
			uid := viewerID(viewer)
			if uid == 0 {
				return nil, quizAnswerRequired()
			}
			if uid != q.UserID && !viewer.Can(spec.editPerm) && !viewer.Can(spec.deletePerm) {
				answered, aerr := s.store.HasAnswered(id, uid)
				if aerr != nil {
					return nil, problem.Internal(aerr)
				}
				if !answered {
					return nil, quizAnswerRequired()
				}
			}
		}
		return sub, nil
	}
	if err != nil {
		return nil, storeProblem(err)
	}
	if p := s.userRenderable(ctx, sub.ownerID); p != nil {
		return nil, p
	}
	return sub, nil
}

func (s *Service) userRenderable(ctx context.Context, userID int) *problem.Problem {
	users, err := s.users.Users(ctx, []int{userID})
	if err != nil {
		return problem.Unavailable(err)
	}
	if u, ok := users[userID]; ok && !userclient.IsRenderable(u) {
		return notFound()
	}
	return nil
}

// requireActive checks the writer against OAuth's current record. A session
// only learns of a ban when its token is refreshed, so without this a banned
// user keeps writing until then.
func (s *Service) requireActive(ctx context.Context, viewer *middleware.UserInfo) *problem.Problem {
	users, err := s.users.Users(ctx, []int{viewer.ID})
	if err != nil {
		return problem.Unavailable(err)
	}
	if u, ok := users[viewer.ID]; ok && !userclient.IsRenderable(u) {
		return problem.New(problem.CodeAccountBanned, "The signed-in user's account is banned.")
	}
	return nil
}

func storeProblem(err error) *problem.Problem {
	if errors.Is(err, repository.ErrNotFound) {
		return notFound()
	}
	return problem.Internal(err)
}

type located struct {
	sub  *subject
	post communityclient.PostView
}

// locate finds a post and the wall it is on. The wall comes from the community
// service's own record of the thread, never from anything in the request: the
// legacy delete took the wall from the URL and checked the permission for that
// wall against a post that could sit on any other.
func (s *Service) locate(ctx context.Context, rawID string, viewer *middleware.UserInfo) (*located, *problem.Problem) {
	id, ok := repr.ParseID(repr.DecimalID(rawID))
	if !ok {
		return nil, notFound()
	}
	res, err := s.community.ResolvePosts(ctx, []int64{int64(id)})
	if err != nil {
		return nil, upstreamProblem(err)
	}
	for _, ap := range res.Posts {
		if ap.Post.ID != int64(id) {
			continue
		}
		spec, sid, known := subjectFromAnchor(ap.Thread.AnchorKind, ap.Thread.AnchorID)
		if !known {
			return nil, notFound()
		}
		sub, p := s.resolveSubject(ctx, spec, sid, viewer)
		if p != nil {
			return nil, p
		}
		if ap.Post.Status == communityclient.PostHeld && int64(viewerID(viewer)) != ap.Post.AuthorID {
			return nil, notFound()
		}
		if p := s.userRenderable(ctx, int(ap.Post.AuthorID)); p != nil {
			return nil, p
		}
		return &located{sub: sub, post: ap.Post}, nil
	}
	return nil, notFound()
}

// upstreamProblem maps a community service failure. Its 4xx bodies carry an
// int code shared by unrelated failures, so the reason text is the only thing
// that tells a word-list rejection from any other 422.
func upstreamProblem(err error) *problem.Problem {
	switch {
	case errors.Is(err, communityclient.ErrRateLimited):
		return problem.New(problem.CodeRateLimited, "The community service refused the write under its new-account rate limit.")
	case errors.Is(err, communityclient.ErrNotConfigured), errors.Is(err, communityclient.ErrForbidden):
		return problem.Unavailable(err)
	}
	var apiErr *communityclient.APIError
	if !errors.As(err, &apiErr) {
		return problem.Unavailable(err)
	}
	switch {
	case apiErr.Status == 404:
		return notFound()
	case apiErr.Status == 422 && strings.Contains(apiErr.Msg, "word list"):
		return problem.New(problem.CodeContentRejected, "The community service's word list refused the submitted text. Nothing was written.")
	case apiErr.Status == 409 && (strings.Contains(apiErr.Msg, "not open") || strings.Contains(apiErr.Msg, "not editable")):
		return problem.New(problem.CodeInvalidStateTransition, "The wall is closed or the comment can no longer change.")
	case apiErr.Status >= 500:
		return problem.Unavailable(err)
	}
	slog.Error("community upstream refused a request this service should not have sent", "status", apiErr.Status, "code", apiErr.Code, "msg", apiErr.Msg)
	return problem.Internal(err)
}
