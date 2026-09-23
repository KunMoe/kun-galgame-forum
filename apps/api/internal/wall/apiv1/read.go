package apiv1

import (
	"context"
	"strconv"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"
)

const (
	listSort = "post_number_asc"
	// The community service answers at most 100 posts; above that it silently
	// falls back to its default page size.
	upstreamMaxLimit = 100
)

func (s *Service) listWallComments(ctx context.Context, in *listInput) (*listOutput, error) {
	if !s.ready() {
		return nil, problem.Internal(errUnconfigured)
	}
	spec, ok := subjects[in.SubjectType]
	if !ok {
		return nil, problem.New(problem.CodeUnknownEnumValue, "subject_type is not in this collection's vocabulary.",
			problem.AtParameter("subject_type", problem.ReasonUnknownValue, "use a declared subject type", nil))
	}
	sid, ok := repr.ParseID(repr.DecimalID(in.SubjectID))
	if !ok {
		return nil, problem.New(problem.CodeInvalidParameter, "subject_id is not a positive decimal integer.",
			problem.AtParameter("subject_id", problem.ReasonInvalidFormat, "must be a positive decimal integer", nil))
	}
	fp := collect.Fingerprint(string(spec.typ), strconv.Itoa(sid))
	keys, curErr := collect.DecodeCursor(in.Cursor, listSort, fp)
	if curErr != nil {
		return nil, curErr
	}
	after := ""
	if keys != nil {
		n, err := strconv.Atoi(keys[0])
		if len(keys) != 1 || err != nil || n < 1 {
			return nil, invalidCursor()
		}
		after = keys[0]
	}

	viewer := v1.User(ctx)
	sub, p := s.resolveSubject(ctx, spec, sid, viewer)
	if p != nil {
		return nil, p
	}

	fetch := min(in.Limit+1, upstreamMaxLimit)
	page, err := s.community.GetComments(ctx, spec.anchorKind, spec.anchorID(sid), after, strconv.Itoa(fetch))
	if err != nil {
		return nil, upstreamProblem(err)
	}
	posts := page.Posts
	more := false
	if len(posts) > in.Limit {
		posts, more = posts[:in.Limit], true
	} else if fetch == in.Limit {
		more = page.NextCursor != ""
	}
	rendered, p := s.render(ctx, sub, posts, viewer)
	if p != nil {
		return nil, p
	}
	items := make([]WallComment, 0, len(rendered))
	for _, item := range rendered {
		if item != nil {
			items = append(items, *item)
		}
	}
	var next *string
	if more && len(posts) > 0 {
		cur := collect.EncodeCursor(listSort, fp, strconv.Itoa(int(posts[len(posts)-1].PostNumber)))
		next = &cur
	}
	return &listOutput{Body: repr.NewList(items, next)}, nil
}

func invalidCursor() *problem.Problem {
	return problem.New(
		problem.CodeInvalidCursor,
		"The cursor cannot be parsed or is no longer valid.",
		problem.AtParameter("cursor", problem.ReasonInvalidFormat, "pass the next_cursor from a previous page of this collection", nil),
	)
}

func (s *Service) getWallComment(ctx context.Context, in *commentInput) (*commentOutput, error) {
	if !s.ready() {
		return nil, problem.Internal(errUnconfigured)
	}
	viewer := v1.User(ctx)
	loc, p := s.locate(ctx, in.WallCommentID, viewer)
	if p != nil {
		return nil, p
	}
	item, p := s.renderOne(ctx, loc.sub, loc.post, viewer)
	if p != nil {
		return nil, p
	}
	return &commentOutput{Body: *item}, nil
}

func (s *Service) getWallCommentSource(ctx context.Context, in *commentInput) (*sourceOutput, error) {
	if !s.ready() {
		return nil, problem.Internal(errUnconfigured)
	}
	viewer := v1.User(ctx)
	loc, p := s.locate(ctx, in.WallCommentID, viewer)
	if p != nil {
		return nil, p
	}
	if !caps(loc.sub, loc.post, viewer).CanEdit {
		return nil, permissionRequired()
	}
	return &sourceOutput{Body: WallCommentSource{
		Object:          "wall_comment_source",
		WallCommentID:   repr.DecimalID(in.WallCommentID),
		ContentMarkdown: loc.post.ContentRaw,
	}}, nil
}
