package apiv1

import (
	"context"
	"strconv"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	wallapiv1 "kun-galgame-api/internal/wall/apiv1"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/problem"
)

const (
	wallCommentSort  = "id_desc"
	wallCommentCalls = 3
)

func (s *Users) listUserWallComments(ctx context.Context, in *listUserWallCommentsInput) (*listUserWallCommentsOutput, error) {
	if prob := s.readyWallComments(); prob != nil {
		return nil, prob
	}
	ownerID, prob := s.requireRenderableOwner(ctx, in.UserID)
	if prob != nil {
		return nil, prob
	}
	limit := in.Limit
	if limit < 1 {
		limit = 24
	}
	fp := collect.Fingerprint(in.Relation, string(in.SubjectType), strconv.Itoa(ownerID))
	keys, curErr := collect.DecodeCursor(in.Cursor, wallCommentSort, fp)
	if curErr != nil {
		return nil, curErr
	}
	after := ""
	if keys != nil {
		if len(keys) != 1 || keys[0] == "" {
			return nil, invalidCursor()
		}
		if _, err := strconv.ParseInt(keys[0], 10, 64); err != nil {
			return nil, invalidCursor()
		}
		after = keys[0]
	}

	viewer := v1.User(ctx)
	var rows []communityclient.AuthorPostView
	var lastKey string
	var more bool
	var pageErr *problem.Problem
	switch in.Relation {
	case "authored":
		rows, lastKey, more, pageErr = s.pageAuthored(ctx, ownerID, after, limit, in.SubjectType)
	case "liked":
		rows, lastKey, more, pageErr = s.pageLiked(ctx, ownerID, after, limit, in.SubjectType)
	default:
		rows = []communityclient.AuthorPostView{}
	}
	if pageErr != nil {
		return nil, pageErr
	}
	items, p := s.wall.RenderAuthored(ctx, viewer, rows)
	if p != nil {
		return nil, p
	}
	var next *string
	if more && lastKey != "" {
		cur := collect.EncodeCursor(wallCommentSort, fp, lastKey)
		next = &cur
	}
	return &listUserWallCommentsOutput{Body: repr.NewList(items, next)}, nil
}

func (s *Users) pageAuthored(ctx context.Context, ownerID int, after string, limit int, subjectType wallapiv1.SubjectType) ([]communityclient.AuthorPostView, string, bool, *problem.Problem) {
	anchorKind := -1
	switch subjectType {
	case "galgame":
		anchorKind = communityclient.AnchorSiteGame
	case "":
		anchorKind = -1
	default:
		anchorKind = communityclient.AnchorSiteResource
	}

	kept := make([]communityclient.AuthorPostView, 0, limit)
	lastKey := after
	more := false
	for call := 0; call < wallCommentCalls && len(kept) < limit; call++ {
		page, err := s.community.AuthorPosts(ctx, int64(ownerID), after, limit, anchorKind)
		if err != nil {
			return nil, "", false, unavailable(err)
		}
		if len(page.Posts) == 0 {
			more = false
			break
		}
		for i, row := range page.Posts {
			lastKey = strconv.FormatInt(row.Post.ID, 10)
			after = lastKey
			if !matchWallSubject(row, subjectType) {
				continue
			}
			kept = append(kept, row)
			if len(kept) == limit {
				more = i+1 < len(page.Posts) || page.NextCursor != ""
				return kept, lastKey, more, nil
			}
		}
		if page.NextCursor == "" {
			more = false
			break
		}
		after = page.NextCursor
		lastKey = page.NextCursor
		more = true
	}
	return kept, lastKey, more, nil
}

func (s *Users) pageLiked(ctx context.Context, ownerID int, after string, limit int, subjectType wallapiv1.SubjectType) ([]communityclient.AuthorPostView, string, bool, *problem.Problem) {
	var afterID int64
	if after != "" {
		n, err := strconv.ParseInt(after, 10, 64)
		if err != nil || n < 1 {
			return nil, "", false, invalidCursor()
		}
		afterID = n
	}
	kept := make([]communityclient.AuthorPostView, 0, limit)
	lastKey := after
	more := false
	for call := 0; call < wallCommentCalls && len(kept) < limit; call++ {
		batch, err := s.content.ListUserLikedPosts(ownerID, afterID, limit)
		if err != nil {
			return nil, "", false, problem.Internal(err)
		}
		if len(batch) == 0 {
			more = false
			break
		}
		ids := make([]int64, len(batch))
		for i, row := range batch {
			ids[i] = row.PostID
		}
		resolved, err := s.community.ResolvePosts(ctx, ids)
		if err != nil {
			return nil, "", false, unavailable(err)
		}
		byID := make(map[int64]communityclient.AuthorPostView, len(resolved.Posts))
		for _, row := range resolved.Posts {
			byID[row.Post.ID] = row
		}
		for i, like := range batch {
			afterID = like.ID
			lastKey = strconv.FormatInt(like.ID, 10)
			row, ok := byID[like.PostID]
			if !ok || !matchWallSubject(row, subjectType) {
				continue
			}
			kept = append(kept, row)
			if len(kept) == limit {
				more = i+1 < len(batch) || len(batch) == limit
				return kept, lastKey, more, nil
			}
		}
		if len(batch) < limit {
			more = false
			break
		}
		more = true
	}
	return kept, lastKey, more, nil
}

func matchWallSubject(row communityclient.AuthorPostView, want wallapiv1.SubjectType) bool {
	if want == "" {
		return true
	}
	got, ok := wallapiv1.SubjectTypeOfAnchor(row.Thread.AnchorKind, row.Thread.AnchorID)
	return ok && got == want
}

func (s *Users) readyWallComments() *problem.Problem {
	if s == nil || s.accounts == nil || s.community == nil || s.wall == nil || s.content == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}
