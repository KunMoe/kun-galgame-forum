package apiv1

import (
	"context"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/problem"
)

type wallKey struct {
	typ SubjectType
	id  int
}

// A profile list that skipped this gate leaked comments on spoiler quizzes.
func (s *Service) RenderAuthored(ctx context.Context, viewer *middleware.UserInfo, rows []communityclient.AuthorPostView) ([]WallComment, *problem.Problem) {
	if !s.ready() {
		return nil, problem.Internal(errUnconfigured)
	}
	if len(rows) == 0 {
		return []WallComment{}, nil
	}

	type resolved struct {
		sub  *subject
		skip bool
	}
	specs := map[wallKey]subjectSpec{}
	order := make([]wallKey, 0)
	idxKey := make([]wallKey, len(rows))
	known := make([]bool, len(rows))

	for i, row := range rows {
		spec, id, ok := subjectFromAnchor(row.Thread.AnchorKind, row.Thread.AnchorID)
		if !ok {
			continue
		}
		key := wallKey{typ: spec.typ, id: id}
		idxKey[i] = key
		known[i] = true
		if _, seen := specs[key]; seen {
			continue
		}
		specs[key] = spec
		order = append(order, key)
	}

	walls := map[wallKey]*resolved{}
	for _, key := range order {
		sub, p := s.resolveSubject(ctx, specs[key], key.id, viewer)
		if p != nil {
			if p.Code == problem.CodeNotFound || p.Code == problem.CodeQuizAnswerRequired {
				walls[key] = &resolved{skip: true}
				continue
			}
			return nil, p
		}
		walls[key] = &resolved{sub: sub}
	}

	type group struct {
		sub   *subject
		posts []communityclient.PostView
		idxs  []int
	}
	groups := map[wallKey]*group{}
	groupOrder := make([]wallKey, 0)
	for i, row := range rows {
		if !known[i] {
			continue
		}
		key := idxKey[i]
		rw := walls[key]
		if rw == nil || rw.skip {
			continue
		}
		g, ok := groups[key]
		if !ok {
			g = &group{sub: rw.sub}
			groups[key] = g
			groupOrder = append(groupOrder, key)
		}
		g.posts = append(g.posts, row.Post)
		g.idxs = append(g.idxs, i)
	}

	byIndex := make([]*WallComment, len(rows))
	for _, key := range groupOrder {
		g := groups[key]
		rendered, p := s.render(ctx, g.sub, g.posts, viewer)
		if p != nil {
			return nil, p
		}
		for j, item := range rendered {
			byIndex[g.idxs[j]] = item
		}
	}

	out := make([]WallComment, 0, len(rows))
	for i := range rows {
		if byIndex[i] != nil {
			out = append(out, *byIndex[i])
		}
	}
	return out, nil
}
