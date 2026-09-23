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

// RenderAuthored renders posts from any mix of walls — the authored and the
// liked lists alike — and drops each post whose wall the viewer may not read.
func (s *Service) RenderAuthored(ctx context.Context, viewer *middleware.UserInfo, rows []communityclient.AuthorPostView) ([]WallComment, *problem.Problem) {
	if !s.ready() || s.galgames == nil {
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
	var workIDs []int
	for _, key := range order {
		if key.typ == "galgame" {
			workIDs = append(workIDs, key.id)
		}
	}
	// resolveSubject answers a galgame wall with an uncached catalog detail
	// call; one per wall, in series, is seconds per profile page and spends
	// the forum's single egress IP against catalog's per-IP limiter.
	if len(workIDs) > 0 {
		found, err := s.galgames(ctx, workIDs)
		if err != nil {
			return nil, problem.Unavailable(err)
		}
		for _, id := range workIDs {
			key := wallKey{typ: "galgame", id: id}
			if found[id] {
				walls[key] = &resolved{sub: &subject{spec: specs[key], id: id}}
			} else {
				walls[key] = &resolved{skip: true}
			}
		}
	}
	for _, key := range order {
		if key.typ == "galgame" {
			continue
		}
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
