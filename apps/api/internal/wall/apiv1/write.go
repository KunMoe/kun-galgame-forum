package apiv1

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/problem"
)

const maxMentions = 20

// checkBody applies the wall's limit to the value as sent (K19), before the
// stored form is derived from it.
func checkBody(spec subjectSpec, raw string) (string, *problem.Problem) {
	if strings.TrimSpace(raw) == "" {
		min := 1
		return "", validationFailed(problem.AtPointer("/content_markdown", problem.ReasonTooShort,
			"must contain at least 1 character after trimming whitespace", &problem.FieldParams{MinLength: &min}))
	}
	if utf8.RuneCountInString(raw) > spec.maxLength {
		max := spec.maxLength
		return "", validationFailed(problem.AtPointer("/content_markdown", problem.ReasonTooLong,
			fmt.Sprintf("must hold at most %d characters on a %s wall", max, spec.typ), &problem.FieldParams{MaxLength: &max}))
	}
	return markdown.NormalizeStoredContent(raw), nil
}

func unknownParent() *problem.Problem {
	return validationFailed(problem.AtPointer("/parent_comment_id", problem.ReasonUnknownReference,
		"the comment is not a visible comment on this wall", nil))
}

func (s *Service) parentOf(ctx context.Context, sub *subject, raw *repr.DecimalID, viewer *middleware.UserInfo) (*communityclient.PostView, *problem.Problem) {
	if raw == nil {
		return nil, nil
	}
	id, ok := repr.ParseID(*raw)
	if !ok {
		return nil, unknownParent()
	}
	res, err := s.community.ResolvePosts(ctx, []int64{int64(id)})
	if err != nil {
		return nil, upstreamProblem(err)
	}
	for _, ap := range res.Posts {
		if ap.Post.ID != int64(id) || ap.Thread.AnchorKind != sub.spec.anchorKind || ap.Thread.AnchorID != sub.spec.anchorID(sub.id) {
			continue
		}
		switch {
		case ap.Post.Status == communityclient.PostDeleted:
		case ap.Post.Status == communityclient.PostHeld && ap.Post.AuthorID != int64(viewer.ID):
		default:
			return &ap.Post, nil
		}
	}
	return nil, unknownParent()
}

// mentionIDs keeps the galgame wall's behaviour: the community service notifies
// the users a new comment names. The other walls never passed mentions on.
func (s *Service) mentionIDs(ctx context.Context, spec subjectSpec, authorID int, body string) ([]int64, *problem.Problem) {
	if spec.typ != "galgame" {
		return nil, nil
	}
	ids := dedupeMentions[struct{}](markdown.ExtractMentionIDs(body), authorID, nil)
	if len(ids) == 0 {
		return nil, nil
	}
	known, err := s.users.Users(ctx, ids)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	ids = dedupeMentions(ids, authorID, known)
	if len(ids) > maxMentions {
		max := maxMentions
		return nil, validationFailed(problem.AtPointer("/content_markdown", problem.ReasonTooManyItems,
			fmt.Sprintf("may mention at most %d users", max), &problem.FieldParams{MaxItems: &max}))
	}
	out := make([]int64, len(ids))
	for i, id := range ids {
		out[i] = int64(id)
	}
	return out, nil
}

func dedupeMentions[U any](ids []int, authorID int, known map[int]U) []int {
	seen := make(map[int]bool, len(ids))
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 || id == authorID || seen[id] {
			continue
		}
		if known != nil {
			if _, ok := known[id]; !ok {
				continue
			}
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func (s *Service) createWallComment(ctx context.Context, in *createInput) (*createOutput, error) {
	if !s.ready() {
		return nil, problem.Internal(errUnconfigured)
	}
	viewer := v1.User(ctx)
	if p := s.requireActive(ctx, viewer); p != nil {
		return nil, p
	}
	spec, ok := subjects[in.Body.SubjectType]
	if !ok {
		return nil, validationFailed(problem.AtPointer("/subject_type", problem.ReasonUnknownValue, "use a declared subject type", nil))
	}
	sid, ok := repr.ParseID(in.Body.SubjectID)
	if !ok {
		return nil, validationFailed(problem.AtPointer("/subject_id", problem.ReasonInvalidFormat, "must be a positive decimal integer", nil))
	}
	sub, p := s.resolveSubject(ctx, spec, sid, viewer)
	if p != nil {
		return nil, p
	}
	body, p := checkBody(spec, in.Body.ContentMarkdown)
	if p != nil {
		return nil, p
	}
	parent, p := s.parentOf(ctx, sub, in.Body.ParentCommentID, viewer)
	if p != nil {
		return nil, p
	}
	mentions, p := s.mentionIDs(ctx, spec, viewer.ID, body)
	if p != nil {
		return nil, p
	}

	req := communityclient.CommentRequest{
		AnchorKind: spec.anchorKind, AnchorID: spec.anchorID(sid),
		ContentRating: communityclient.RatingAll, AuthorID: int64(viewer.ID), Body: body,
		MentionUserIDs: mentions,
	}
	if parent != nil {
		req.ReplyToPostID = parent.ID
	} else if spec.typ == "galgame_rating" {
		req.TargetUserID = int64(sub.ownerID)
	}
	res, err := s.community.CommentOnAnchor(ctx, req)
	if err != nil {
		return nil, upstreamProblem(err)
	}
	s.afterCreate(ctx, sub, viewer.ID, body, &res.Post)

	item, p := s.renderOne(ctx, sub, res.Post, viewer)
	if p != nil {
		return nil, p
	}
	return &createOutput{
		Location: "/api/v1/wall-comments/" + strconv.FormatInt(res.Post.ID, 10),
		Body:     *item,
	}, nil
}

func (s *Service) updateWallComment(ctx context.Context, in *updateInput) (*commentOutput, error) {
	if !s.ready() {
		return nil, problem.Internal(errUnconfigured)
	}
	viewer := v1.User(ctx)
	if p := s.requireActive(ctx, viewer); p != nil {
		return nil, p
	}
	loc, p := s.locate(ctx, in.WallCommentID, viewer)
	if p != nil {
		return nil, p
	}
	if loc.post.Status == communityclient.PostDeleted {
		return nil, tombstoned()
	}
	if !caps(loc.sub, loc.post, viewer).CanEdit {
		return nil, permissionRequired()
	}
	body, p := checkBody(loc.sub.spec, in.Body.ContentMarkdown)
	if p != nil {
		return nil, p
	}
	post := loc.post
	if body != post.ContentRaw {
		isAuthor := int64(viewer.ID) == post.AuthorID
		edited, err := s.community.EditPost(ctx, post.ID, communityclient.EditPostRequest{
			AuthorID: int64(viewer.ID), Body: body, AsModerator: !isAuthor,
		})
		if err != nil {
			return nil, upstreamProblem(err)
		}
		if loc.sub.spec.typ == "galgame" {
			s.notifyNewMentions(viewer.ID, loc.sub.id, post.ContentRaw, body, post.ID)
		}
		post = *edited
	}
	item, p := s.renderOne(ctx, loc.sub, post, viewer)
	if p != nil {
		return nil, p
	}
	return &commentOutput{Body: *item}, nil
}

func (s *Service) deleteWallComment(ctx context.Context, in *commentInput) (*struct{}, error) {
	if !s.ready() {
		return nil, problem.Internal(errUnconfigured)
	}
	viewer := v1.User(ctx)
	if p := s.requireActive(ctx, viewer); p != nil {
		return nil, p
	}
	loc, p := s.locate(ctx, in.WallCommentID, viewer)
	if p != nil {
		return nil, p
	}
	if loc.post.Status == communityclient.PostDeleted {
		return nil, nil
	}
	if !caps(loc.sub, loc.post, viewer).CanDelete {
		return nil, permissionRequired()
	}
	isAuthor := int64(viewer.ID) == loc.post.AuthorID
	if err := s.community.DeletePost(ctx, loc.post.ID, int64(viewer.ID), !isAuthor); err != nil {
		return nil, upstreamProblem(err)
	}
	s.afterDelete(loc.sub, loc.post.ID)
	return nil, nil
}
