package apiv1

import (
	"context"
	"errors"
	"log/slog"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"
)

type listMyWorkStatesInput struct {
	WorkIDs []repr.DecimalID `query:"work_ids" required:"true" maxItems:"100" doc:"Work ids to answer for, comma-separated. 1 to 100 of them."`
}

type listMyWorkStatesOutput struct {
	Body repr.BatchList[WorkState]
}

func (s *Service) listMyWorkStates(ctx context.Context, in *listMyWorkStatesInput) (*listMyWorkStatesOutput, error) {
	if s == nil || s.works == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	user := v1.User(ctx)
	if user == nil {
		return nil, notFound()
	}
	requested, p := parseWorkIDs(in.WorkIDs)
	if p != nil {
		return nil, p
	}
	unique := make([]int, 0, len(requested))
	seen := map[int]bool{}
	for _, id := range requested {
		if seen[id] {
			continue
		}
		seen[id] = true
		unique = append(unique, id)
	}
	readable, p := s.readableWorks(ctx, unique)
	if p != nil {
		return nil, p
	}
	liked := map[int]bool{}
	if s.store != nil && s.store.Ready() {
		var err error
		liked, err = s.store.LikedSet(user.ID, unique)
		if err != nil {
			return nil, problem.Internal(err)
		}
	}
	favorited, p := s.favoritedSet(ctx, accessToken(ctx), unique, user.ID)
	if p != nil {
		return nil, p
	}
	items := make([]WorkState, 0, len(unique))
	missing := []repr.DecimalID{}
	emitted := map[int]bool{}
	for _, id := range requested {
		if !readable[id] {
			if !emitted[id] {
				emitted[id] = true
				missing = append(missing, repr.ID(id))
			}
			continue
		}
		if emitted[id] {
			continue
		}
		emitted[id] = true
		items = append(items, WorkState{
			Object: "work_state", WorkID: repr.ID(id),
			HasLiked: liked[id], HasFavorited: favorited[id],
		})
	}
	return &listMyWorkStatesOutput{Body: repr.NewBatchList(items, missing)}, nil
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

// Walking every folder membership to paint hearts is what spent user 90769's
// 10k/day quota on 2026-09-20 (3,560 items, 36 catalog pages per call).
// Holdings answers the same question for the ids on screen, in one call.
func (s *Service) favoritedSet(ctx context.Context, token string, ids []int, userID int) (map[int]bool, *problem.Problem) {
	out := map[int]bool{}
	if token == "" || s.catalog == nil || len(ids) == 0 {
		return out, nil
	}
	want := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id > 0 {
			want = append(want, int64(id))
		}
	}
	holdings, err := s.catalog.MyFolderHoldings(ctx, token, want)
	if err != nil {
		if errors.Is(err, catalogclient.ErrInsufficientScope) {
			service.WarnFoldersUnreadable(userID)
		} else {
			slog.Warn("galgame: my folders unreadable", "user_id", userID, "upstream_status", upstreamStatus(err), "err", err)
		}
		return out, nil
	}
	for _, h := range holdings {
		if h.WorkID > 0 {
			out[int(h.WorkID)] = true
		}
	}
	return out, nil
}
