package apiv1

import (
	"context"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/community/anchor"
	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	galgameService "kun-galgame-api/internal/galgame/service"
	wallapiv1 "kun-galgame-api/internal/wall/apiv1"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

const wallCursorSort = "wall_comments"

var wallResourceTypes = map[string]wallapiv1.SubjectType{
	"rating":   "galgame_rating",
	"resource": "galgame_resource",
	"quiz":     "galgame_quiz",
	"toolset":  "toolset",
	"website":  "website",
}

type wallOutput struct {
	Body repr.List[WallCommentSearchHit]
}

func (s *Service) searchWallComments(ctx context.Context, in *wallInput) (*wallOutput, error) {
	if prob := s.ready(); prob != nil {
		return nil, prob
	}
	q := strings.TrimSpace(in.Q)
	if n := utf8.RuneCountInString(q); n < 2 {
		minLen := 2
		return nil, problem.New(problem.CodeInvalidParameter, "q is shorter than 2 characters after trimming.",
			problem.AtParameter("q", problem.ReasonTooShort, "q needs at least 2 non-space characters",
				&problem.FieldParams{MinLength: &minLen}))
	}
	fingerprint := collect.Fingerprint(q)
	keys, prob := collect.DecodeCursor(in.Cursor, wallCursorSort, fingerprint)
	if prob != nil {
		return nil, prob
	}
	upstream := ""
	if len(keys) == 1 {
		upstream = keys[0]
	}
	if s.community == nil || s.anchors == nil || !s.community.Configured() {
		return nil, problem.Unavailable(errUnconfigured)
	}
	page, err := s.community.SearchPosts(ctx, q, communityclient.KindComments, upstream, in.Limit)
	if err != nil {
		return nil, problem.Unavailable(err)
	}

	refs := make([]anchor.Ref, 0, len(page.Posts))
	uids := make([]int, 0, len(page.Posts))
	for _, row := range page.Posts {
		refs = append(refs, anchor.Ref{Kind: row.Thread.AnchorKind, ID: row.Thread.AnchorID})
		uids = append(uids, int(row.Post.AuthorID))
	}
	targets := s.anchors.Resolve(refs)
	users, prob := s.lookupUsers(ctx, uids)
	if prob != nil {
		return nil, prob
	}
	works := s.wallWorks(ctx, targets)

	keywords := strings.Fields(q)
	items := make([]WallCommentSearchHit, 0, len(page.Posts))
	for _, row := range page.Posts {
		ref := anchor.Ref{Kind: row.Thread.AnchorKind, ID: row.Thread.AnchorID}
		target, ok := targets[ref]
		if !ok {
			continue
		}
		subjectType, subjectID, ok := wallSubject(ref)
		if !ok {
			continue
		}
		if author, known := users[int(row.Post.AuthorID)]; known && !userclient.IsRenderable(author) {
			continue
		}
		created, err := time.Parse(time.RFC3339, row.Post.CreatedAt)
		if err != nil {
			return nil, problem.Internal(err)
		}
		hit := WallCommentSearchHit{
			Object: "wall_comment", ID: repr.DecimalID(strconv.FormatInt(row.Post.ID, 10)),
			SubjectType: subjectType, SubjectID: repr.DecimalID(subjectID), SubjectPath: target.Link,
			Excerpt: excerpt(row.Post.ContentRaw, keywords),
			Author:  s.authorRef(users, int(row.Post.AuthorID)), CreatedAt: repr.Timestamp(created),
		}
		if work, ok := works[target.WorkID]; ok && target.WorkID > 0 {
			hit.Work = &work
		}
		items = append(items, hit)
	}
	var next *string
	if page.NextCursor != "" {
		cur := collect.EncodeCursor(wallCursorSort, fingerprint, page.NextCursor)
		next = &cur
	}
	return &wallOutput{Body: repr.NewList(items, next)}, nil
}

func wallSubject(ref anchor.Ref) (wallapiv1.SubjectType, string, bool) {
	switch ref.Kind {
	case communityclient.AnchorSiteGame:
		return "galgame", ref.ID, true
	case communityclient.AnchorSiteResource:
		source, id, ok := strings.Cut(ref.ID, ":")
		typ, known := wallResourceTypes[source]
		return typ, id, ok && known
	}
	return "", "", false
}

func (s *Service) wallWorks(ctx context.Context, targets map[anchor.Ref]anchor.Target) map[int]repr.WorkRef {
	out := map[int]repr.WorkRef{}
	if s.galgame == nil {
		return out
	}
	ids := make([]int, 0, len(targets))
	for _, t := range targets {
		if t.WorkID > 0 {
			ids = append(ids, t.WorkID)
		}
	}
	if len(ids) == 0 {
		return out
	}
	rows, appErr := s.galgame.CatalogRowsByWorkIDs(ctx, ids, galgameService.CatalogCardInclude, "all")
	if appErr != nil {
		return out
	}
	for id := range rows {
		row := rows[id]
		out[id] = galgameapiv1.WorkRefOf(ctx, &row, s.cdn)
	}
	return out
}
