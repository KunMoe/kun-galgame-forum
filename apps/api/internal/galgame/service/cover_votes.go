package service

import (
	"context"
	"errors"
	"log/slog"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/catalogclient"
)

func (s *GalgameService) hydrateCoverVotes(ctx context.Context, workID int, accessToken string, covers []dto.GalgameCover) {
	if s.catalog == nil || len(covers) == 0 {
		return
	}
	var (
		tallies []catalogclient.CoverTally
		err     error
	)
	if accessToken != "" {
		tallies, err = s.catalog.WorkCoversUser(ctx, accessToken, int64(workID))
		// The user lane answers 401 for every reader, not the 403 SCOPE_REQUIRED
		// this fallback was written for, so signed-in readers saw no tallies at all
		// while signed-out readers saw them. Degrading loses only the `voted` flag.
		if errors.Is(err, catalogclient.ErrInsufficientScope) || errors.Is(err, catalogclient.ErrUnauthorized) {
			tallies, err = s.catalog.WorkCoverVotes(ctx, int64(workID))
		}
	} else {
		tallies, err = s.catalog.WorkCoverVotes(ctx, int64(workID))
	}
	if err != nil {
		slog.Warn("galgame detail: cover vote tallies unavailable", "work_id", workID, "error", err)
		return
	}
	byHash := make(map[string]int, len(tallies))
	for i, t := range tallies {
		if t.ImageHash != "" {
			byHash[t.ImageHash] = i
		}
	}
	for i := range covers {
		t, ok := byHash[covers[i].ImageHash]
		if !ok {
			continue
		}
		covers[i].ID = tallies[t].ID
		covers[i].VoteCount = tallies[t].VoteCount
		covers[i].Voted = tallies[t].Voted
	}
}
