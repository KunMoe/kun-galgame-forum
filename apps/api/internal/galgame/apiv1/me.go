package apiv1

import (
	"context"
	"errors"
	"log/slog"
	"slices"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/playstate"
	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"
)

type myWorkIDsInput struct {
	WorkIDs []repr.DecimalID `query:"work_ids" required:"true" maxItems:"100" doc:"Work ids to answer for, comma-separated. 1 to 100 of them."`
}

type listMyWorksOutput struct {
	Body repr.BatchList[MyWork]
}

type listMyCoverVotesOutput struct {
	Body repr.BatchList[MyCoverVote]
}

// readableRequest splits the requested ids into the readable ones, once each
// in request order, and the missing ones. Only readable ids go on to catalog's
// user plane, so a hidden work never answers even "you hold it".
func (s *Service) readableRequest(ctx context.Context, raw []repr.DecimalID) ([]int, []repr.DecimalID, *problem.Problem) {
	requested, p := parseWorkIDs(raw)
	if p != nil {
		return nil, nil, p
	}
	unique := make([]int, 0, len(requested))
	for _, id := range requested {
		if !slices.Contains(unique, id) {
			unique = append(unique, id)
		}
	}
	readable, p := s.readableWorks(ctx, unique)
	if p != nil {
		return nil, nil, p
	}
	ok := make([]int, 0, len(unique))
	missing := []repr.DecimalID{}
	for _, id := range unique {
		if readable[id] {
			ok = append(ok, id)
		} else {
			missing = append(missing, repr.ID(id))
		}
	}
	return ok, missing, nil
}

func (s *Service) listMyWorks(ctx context.Context, in *myWorkIDsInput) (*listMyWorksOutput, error) {
	if s == nil || s.works == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	user := v1.User(ctx)
	if user == nil {
		return nil, notFound()
	}
	ids, missing, p := s.readableRequest(ctx, in.WorkIDs)
	if p != nil {
		return nil, p
	}
	liked := map[int]bool{}
	if s.store != nil && s.store.Ready() {
		var err error
		liked, err = s.store.LikedSet(user.ID, ids)
		if err != nil {
			return nil, problem.Internal(err)
		}
	}
	libraries := s.myLibraries(ctx, user, ids)
	items := make([]MyWork, 0, len(ids))
	for _, id := range ids {
		items = append(items, MyWork{
			Object: "my_work", WorkID: repr.ID(id), HasLiked: liked[id], Library: libraries[id],
		})
	}
	return &listMyWorksOutput{Body: repr.NewBatchList(items, missing)}, nil
}

// A catalog failure leaves every library nil, which the page renders as
// unknown. has_favorited used to read false on any failure, so a throttled
// reader saw every heart go empty (2026-09-24).
func (s *Service) myLibraries(ctx context.Context, user *middleware.UserInfo, ids []int) map[int]*MyWorkLibrary {
	token := accessToken(ctx)
	if token == "" || s.catalog == nil || len(ids) == 0 {
		return nil
	}
	want := make([]int64, 0, len(ids))
	for _, id := range ids {
		want = append(want, int64(id))
	}
	rows, err := s.catalog.MyWorks(ctx, token, want)
	if err != nil {
		switch {
		// The App asks only for openid, profile and preferences, so every Bearer
		// request lands here until it adds folder:read.
		case errors.Is(err, catalogclient.ErrInsufficientScope) && user.ViaBearer():
			slog.Debug("me works: bearer token lacks folder:read, library unknown", "user_id", user.ID)
		case errors.Is(err, catalogclient.ErrInsufficientScope):
			service.WarnFoldersUnreadable(user.ID)
		default:
			slog.Warn("me works: catalog unreadable, library unknown", "user_id", user.ID, "upstream_status", upstreamStatus(err), "err", err)
		}
		return nil
	}
	out := make(map[int]*MyWorkLibrary, len(rows))
	for _, r := range rows {
		folders := slices.Clone(r.FolderIDs)
		slices.Sort(folders)
		lib := &MyWorkLibrary{CollectionIDs: make([]repr.DecimalID, 0, len(folders)), Playtime: libraryPlaytime(r)}
		for _, f := range folders {
			lib.CollectionIDs = append(lib.CollectionIDs, repr.ID(int(f)))
		}
		out[int(r.WorkID)] = lib
	}
	return out
}

func libraryPlaytime(r catalogclient.MyWork) *WorkViewerPlaytime {
	minutes := 0
	if r.Playtime != nil && r.Playtime.Minutes >= catalogclient.PlaytimeMinutesFloor {
		minutes = r.Playtime.Minutes
	}
	var state *string
	if ws := r.WorkState; ws != nil {
		if flat, ok := knownPlayState(ws); ok {
			state = &flat
		} else {
			slog.Warn("me works: unknown catalog work state, play_state dropped",
				"work_id", r.WorkID, "state", ws.State, "completion", ws.Completion)
		}
	}
	if minutes == 0 && state == nil {
		return nil
	}
	return &WorkViewerPlaytime{Minutes: minutes, PlayState: state}
}

// knownPlayState accepts only infra's closed vocabulary (fac31d2a repr):
// state wish|doing|done|on_hold|dropped, completion one_route|main|all|null.
// playstate.FromCatalog alone reads an unknown completion as "done".
func knownPlayState(ws *catalogclient.WorkStateRecord) (string, bool) {
	switch ws.State {
	case catalogclient.WorkStateWish, catalogclient.WorkStateDoing, catalogclient.WorkStateDone,
		catalogclient.WorkStateOnHold, catalogclient.WorkStateDropped:
	default:
		return "", false
	}
	if ws.Completion != nil {
		switch *ws.Completion {
		case "one_route", "main", "all":
		default:
			return "", false
		}
	}
	return playstate.FromCatalog(ws.State, ws.Completion), true
}

func (s *Service) listMyCoverVotes(ctx context.Context, in *myWorkIDsInput) (*listMyCoverVotesOutput, error) {
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	if s.works == nil || s.catalog == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	ids, missing, p := s.readableRequest(ctx, in.WorkIDs)
	if p != nil {
		return nil, p
	}
	votes, err := s.myCoverVotes(ctx, user.ID, token)
	if err != nil {
		return nil, mapUserPlane(err, true)
	}
	voted := map[int64]int64{}
	for _, v := range votes {
		voted[v.WorkID] = v.CoverID
	}
	items := make([]MyCoverVote, 0, len(ids))
	for _, id := range ids {
		item := MyCoverVote{Object: "my_cover_vote", WorkID: repr.ID(id)}
		if cover, ok := voted[int64(id)]; ok {
			c := repr.ID(int(cover))
			item.VotedCoverID = &c
		}
		items = append(items, item)
	}
	return &listMyCoverVotesOutput{Body: repr.NewBatchList(items, missing)}, nil
}

func (s *Service) readableWorks(ctx context.Context, ids []int) (map[int]bool, *problem.Problem) {
	out := map[int]bool{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, appErr := s.works.CatalogRowsByWorkIDs(ctx, ids, "names", "all")
	if appErr != nil {
		return nil, catalogUnavailable(appErr)
	}
	for _, id := range ids {
		row, ok := rows[id]
		if ok && client.CatalogItemRenderable(&row) {
			out[id] = true
		}
	}
	return out, nil
}
