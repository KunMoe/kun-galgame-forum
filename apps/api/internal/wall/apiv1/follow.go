package apiv1

import (
	"context"
	"math"
	"strconv"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	websiteapiv1 "kun-galgame-api/internal/website/apiv1"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/problem"
)

type ReadSyncFunc func(userID int, threadID int64, lastRead int32) error

type WorkRefsFunc func(ctx context.Context, workIDs []int) (map[int]repr.WorkRef, error)

type WebsitesFunc func(websiteIDs []int) (map[int]websiteapiv1.WebsiteSummary, error)

func (s *Service) WithFollowing(readSync ReadSyncFunc, works WorkRefsFunc, websites WebsitesFunc) *Service {
	s.readSync, s.works, s.websites = readSync, works, websites
	return s
}

func (s *Service) followReady() bool {
	return s.ready() && s.readSync != nil && s.works != nil && s.websites != nil
}

type WallState struct {
	Object      string         `json:"object" enum:"wall_state" maxLength:"10" doc:"Type discriminant. Always wall_state."`
	ID          repr.DecimalID `json:"id" doc:"The wall's subject id, the same value as subject_id."`
	SubjectType SubjectType    `json:"subject_type"`
	SubjectID   repr.DecimalID `json:"subject_id" doc:"Id of the page whose wall this is."`
	IsFollowing bool           `json:"is_following" doc:"Whether the caller gets notified of every new comment on this wall."`
}

type FollowedWall struct {
	Object      string                       `json:"object" enum:"followed_wall" maxLength:"13" doc:"Type discriminant. Always followed_wall."`
	SubjectType SubjectType                  `json:"subject_type"`
	SubjectID   repr.DecimalID               `json:"subject_id" doc:"Id of the page whose wall this is."`
	Work        *repr.WorkRef                `json:"work" doc:"The work, for a galgame wall. null for every other kind of wall, and for a galgame wall whose work catalog no longer shows."`
	Website     *websiteapiv1.WebsiteSummary `json:"website" doc:"The site, for a website wall. null for every other kind of wall, and for a website that no longer exists."`
}

type wallInput struct {
	SubjectType SubjectType `path:"subject_type"`
	SubjectID   string      `path:"subject_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Id of the page whose wall this is."`
}

type wallStateOutput struct {
	Body WallState
}

type followedWallsInput struct {
	collect.Page
}

type followedWallsOutput struct {
	Body repr.List[FollowedWall]
}

const (
	followedSort        = "followed"
	followedUpstreamMax = 50
)

func (s *Service) wallFor(ctx context.Context, in *wallInput) (*middleware.UserInfo, *subject, *problem.Problem) {
	if !s.followReady() {
		return nil, nil, problem.Internal(errUnconfigured)
	}
	spec, ok := subjects[in.SubjectType]
	if !ok {
		return nil, nil, notFound()
	}
	id, ok := repr.ParseID(repr.DecimalID(in.SubjectID))
	if !ok {
		return nil, nil, notFound()
	}
	viewer := v1.User(ctx)
	sub, p := s.resolveSubject(ctx, spec, id, viewer)
	if p != nil {
		return nil, nil, p
	}
	return viewer, sub, nil
}

func wallState(sub *subject, following bool) WallState {
	return WallState{
		Object: "wall_state", ID: repr.ID(sub.id), SubjectType: sub.spec.typ,
		SubjectID: repr.ID(sub.id), IsFollowing: following,
	}
}

func (s *Service) threadOf(ctx context.Context, sub *subject) (int64, *problem.Problem) {
	page, err := s.community.GetComments(ctx, sub.spec.anchorKind, sub.spec.anchorID(sub.id), "", "1")
	if err != nil {
		return 0, upstreamProblem(err)
	}
	if page.Thread == nil {
		return 0, nil
	}
	return page.Thread.ID, nil
}

type standing struct {
	thread       int64
	threadRow    *communityclient.ThreadUserView
	anchorLevel  int32
	anchorExists bool
}

func (st standing) level() int32 {
	if st.threadRow != nil {
		return st.threadRow.NotificationLevel
	}
	if st.anchorExists {
		return st.anchorLevel
	}
	return communityclient.NotificationNormal
}

func (s *Service) standingOn(ctx context.Context, userID int, sub *subject) (standing, *problem.Problem) {
	var st standing
	thread, p := s.threadOf(ctx, sub)
	if p != nil {
		return st, p
	}
	st.thread = thread
	if thread > 0 {
		rows, err := s.community.ThreadStates(ctx, int64(userID), []int64{thread})
		if err != nil {
			return st, upstreamProblem(err)
		}
		if len(rows.States) > 0 {
			st.threadRow = &rows.States[0]
		}
	}
	anchors, err := s.community.AnchorStates(ctx, int64(userID), []communityclient.AnchorRef{
		{AnchorKind: sub.spec.anchorKind, AnchorID: sub.spec.anchorID(sub.id)},
	})
	if err != nil {
		return st, upstreamProblem(err)
	}
	if len(anchors.States) > 0 {
		st.anchorExists, st.anchorLevel = true, anchors.States[0].NotificationLevel
	}
	return st, nil
}

func (s *Service) getWallState(ctx context.Context, in *wallInput) (*wallStateOutput, error) {
	viewer, sub, p := s.wallFor(ctx, in)
	if p != nil {
		return nil, p
	}
	st, p := s.standingOn(ctx, viewer.ID, sub)
	if p != nil {
		return nil, p
	}
	return &wallStateOutput{Body: wallState(sub, st.level() == communityclient.NotificationWatching)}, nil
}

func (s *Service) followWall(ctx context.Context, in *wallInput) (*wallStateOutput, error) {
	return s.setFollowing(ctx, in, true)
}

func (s *Service) unfollowWall(ctx context.Context, in *wallInput) (*wallStateOutput, error) {
	return s.setFollowing(ctx, in, false)
}

// Unfollowing writes normal, never muted: muted would also silence the replies
// and mentions addressed to the caller.
func (s *Service) setFollowing(ctx context.Context, in *wallInput, following bool) (*wallStateOutput, error) {
	viewer, sub, p := s.wallFor(ctx, in)
	if p != nil {
		return nil, p
	}
	level := int32(communityclient.NotificationNormal)
	if following {
		level = communityclient.NotificationWatching
	}
	thread, p := s.threadOf(ctx, sub)
	if p != nil {
		return nil, p
	}
	if _, err := s.community.SetAnchorNotification(ctx, int64(viewer.ID), sub.spec.anchorKind, sub.spec.anchorID(sub.id), level); err != nil {
		return nil, upstreamProblem(err)
	}
	if thread > 0 {
		if _, err := s.community.SetThreadNotification(ctx, thread, int64(viewer.ID), level); err != nil {
			return nil, upstreamProblem(err)
		}
	}
	return &wallStateOutput{Body: wallState(sub, following)}, nil
}

// A receipt only marks a wall the caller already has standing on (a thread
// row, or a followed anchor); for any other wall it would create state
// upstream for a page the caller merely opened.
func (s *Service) markWallRead(ctx context.Context, in *wallInput) (*wallStateOutput, error) {
	viewer, sub, p := s.wallFor(ctx, in)
	if p != nil {
		return nil, p
	}
	st, p := s.standingOn(ctx, viewer.ID, sub)
	if p != nil {
		return nil, p
	}
	watchingAnchor := st.anchorExists && st.anchorLevel == communityclient.NotificationWatching
	if st.thread > 0 && (st.threadRow != nil || watchingAnchor) {
		view, err := s.community.MarkThreadRead(ctx, st.thread, int64(viewer.ID), math.MaxInt32)
		if err != nil {
			return nil, upstreamProblem(err)
		}
		st.threadRow = view
		if err := s.readSync(viewer.ID, st.thread, view.LastReadPostNumber); err != nil {
			return nil, problem.Internal(err)
		}
	}
	return &wallStateOutput{Body: wallState(sub, st.level() == communityclient.NotificationWatching)}, nil
}

func (s *Service) listFollowedWalls(ctx context.Context, in *followedWallsInput) (*followedWallsOutput, error) {
	if !s.followReady() {
		return nil, problem.Internal(errUnconfigured)
	}
	viewer := v1.User(ctx)
	fp := collect.Fingerprint(strconv.Itoa(viewer.ID))
	keys, curErr := collect.DecodeCursor(in.Cursor, followedSort, fp)
	if curErr != nil {
		return nil, curErr
	}
	upstreamCursor := ""
	if keys != nil {
		if len(keys) != 1 || keys[0] == "" {
			return nil, invalidCursor()
		}
		upstreamCursor = keys[0]
	}
	page, err := s.community.ListAnchorSubscriptions(ctx, int64(viewer.ID), -1, upstreamCursor, min(in.Limit, followedUpstreamMax))
	if err != nil {
		return nil, upstreamProblem(err)
	}

	type followed struct {
		spec subjectSpec
		id   int
	}
	var rows []followed
	var workIDs, websiteIDs []int
	for _, sub := range page.Subscriptions {
		if sub.NotificationLevel != communityclient.NotificationWatching {
			continue
		}
		spec, id, ok := subjectFromAnchor(sub.AnchorKind, sub.AnchorID)
		if !ok {
			continue
		}
		rows = append(rows, followed{spec, id})
		switch spec.typ {
		case "galgame":
			workIDs = append(workIDs, id)
		case "website":
			websiteIDs = append(websiteIDs, id)
		}
	}
	works := map[int]repr.WorkRef{}
	if len(workIDs) > 0 {
		works, err = s.works(ctx, workIDs)
		if err != nil {
			return nil, problem.Unavailable(err)
		}
	}
	websites := map[int]websiteapiv1.WebsiteSummary{}
	if len(websiteIDs) > 0 {
		websites, err = s.websites(websiteIDs)
		if err != nil {
			return nil, problem.Internal(err)
		}
	}
	items := make([]FollowedWall, 0, len(rows))
	for _, row := range rows {
		item := FollowedWall{Object: "followed_wall", SubjectType: row.spec.typ, SubjectID: repr.ID(row.id)}
		if w, ok := works[row.id]; ok && row.spec.typ == "galgame" {
			item.Work = &w
		}
		if site, ok := websites[row.id]; ok && row.spec.typ == "website" {
			item.Website = &site
		}
		items = append(items, item)
	}
	var next *string
	if page.NextCursor != "" {
		cur := collect.EncodeCursor(followedSort, fp, page.NextCursor)
		next = &cur
	}
	return &followedWallsOutput{Body: repr.NewList(items, next)}, nil
}
