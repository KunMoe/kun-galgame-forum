package apiv1

import (
	"context"
	"strconv"
	"time"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

func postState(status int32) string {
	switch status {
	case communityclient.PostHeld:
		return "held"
	case communityclient.PostDeleted:
		return "deleted"
	default:
		return "visible"
	}
}

func caps(sub *subject, p communityclient.PostView, viewer *middleware.UserInfo) *WallCommentViewer {
	if viewer == nil {
		return nil
	}
	if p.Status == communityclient.PostDeleted {
		return &WallCommentViewer{}
	}
	isAuthor := int64(viewer.ID) == p.AuthorID
	return &WallCommentViewer{
		CanEdit:   isAuthor || viewer.Can(sub.spec.editPerm),
		CanDelete: isAuthor || viewer.Can(sub.spec.deletePerm) || (sub.spec.hasOwner && sub.ownerID == viewer.ID),
		CanLike:   !isAuthor,
		CanFlag:   !isAuthor,
	}
}

// render maps posts of one wall in order. A post by a banned author, or a held
// post the caller did not write, maps to nil and is left out.
func (s *Service) render(ctx context.Context, sub *subject, posts []communityclient.PostView, viewer *middleware.UserInfo) ([]*WallComment, *problem.Problem) {
	out := make([]*WallComment, len(posts))
	if len(posts) == 0 {
		return out, nil
	}
	ids := make([]int, 0, len(posts)*2)
	postIDs := make([]int64, len(posts))
	sources := make([]string, len(posts))
	for i, p := range posts {
		ids = append(ids, int(p.AuthorID))
		if p.TargetUserID != 0 {
			ids = append(ids, int(p.TargetUserID))
		}
		postIDs[i] = p.ID
		if p.Status != communityclient.PostDeleted {
			sources[i] = p.ContentRaw
		}
	}
	users, err := s.users.Users(ctx, ids)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	likes, err := s.store.LikeCounts(postIDs)
	if err != nil {
		return nil, problem.Internal(err)
	}
	liked, err := s.store.LikedSet(viewerID(viewer), postIDs)
	if err != nil {
		return nil, problem.Internal(err)
	}
	docs, err := s.convert.Convert(ctx, sources)
	if err != nil {
		return nil, problem.Internal(err)
	}

	ref := func(id int64) repr.UserRef {
		if u, ok := users[int(id)]; ok && userclient.IsRenderable(u) {
			return repr.NewUserRef(s.cdn, u)
		}
		return repr.DeletedUserRef(int(id))
	}
	uid := int64(viewerID(viewer))
	for i, p := range posts {
		if u, ok := users[int(p.AuthorID)]; ok && !userclient.IsRenderable(u) {
			continue
		}
		if p.Status == communityclient.PostHeld && p.AuthorID != uid {
			continue
		}
		created, err := parseUpstreamTime(p.CreatedAt)
		if err != nil {
			return nil, problem.Internal(err)
		}
		item := &WallComment{
			Object:              "wall_comment",
			ID:                  repr.DecimalID(strconv.FormatInt(p.ID, 10)),
			SubjectType:         sub.spec.typ,
			SubjectID:           repr.ID(sub.id),
			ParentCommentID:     optionalID(p.ReplyToPostID),
			RootCommentID:       optionalID(p.RootPostID),
			Author:              ref(p.AuthorID),
			State:               postState(p.Status),
			Content:             docs[i],
			LikeCount:           likes[p.ID],
			CreatedAt:           repr.Timestamp(created),
			IsEditedByModerator: p.EditedByModerator,
			Viewer:              caps(sub, p, viewer),
		}
		if p.TargetUserID != 0 {
			addressee := ref(p.TargetUserID)
			item.Addressee = &addressee
		}
		if p.EditedAt != "" {
			edited, err := parseUpstreamTime(p.EditedAt)
			if err != nil {
				return nil, problem.Internal(err)
			}
			item.EditedAt = repr.TimestampPtr(&edited)
		}
		if item.Viewer != nil {
			item.Viewer.HasLiked = liked[p.ID]
		}
		out[i] = item
	}
	return out, nil
}

func (s *Service) renderOne(ctx context.Context, sub *subject, p communityclient.PostView, viewer *middleware.UserInfo) (*WallComment, *problem.Problem) {
	items, prob := s.render(ctx, sub, []communityclient.PostView{p}, viewer)
	if prob != nil {
		return nil, prob
	}
	if items[0] == nil {
		return nil, notFound()
	}
	return items[0], nil
}

func optionalID(id int64) *repr.DecimalID {
	if id == 0 {
		return nil
	}
	d := repr.DecimalID(strconv.FormatInt(id, 10))
	return &d
}

func parseUpstreamTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, s)
}
