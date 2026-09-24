package service

import (
	"context"
	"errors"
	"log/slog"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/playstate"
	"kun-galgame-api/pkg/catalogclient"
)

type PlaytimeService struct {
	galgameClient *client.GalgameClient
	catalog       *catalogclient.Client
}

func NewPlaytimeService(galgameClient *client.GalgameClient, catalog *catalogclient.Client) *PlaytimeService {
	return &PlaytimeService{galgameClient: galgameClient, catalog: catalog}
}

// SyncWorkState mirrors a published rating's play state onto catalog. The
// rating is already committed when this runs, so every failure is logged and
// swallowed: a catalog outage must not fail a rating the user already wrote.
func (s *PlaytimeService) SyncWorkState(ctx context.Context, workID int, accessToken, playStatus string) {
	if accessToken == "" {
		return
	}
	catState, completion, ok := playstate.ToCatalog(playStatus)
	if !ok {
		slog.Warn("galgame rating: skip work-state sync, unknown play status", "work_id", workID, "play_status", playStatus)
		return
	}
	if s.catalog == nil || s.galgameClient == nil {
		slog.Warn("galgame rating: skip work-state sync, catalog unavailable", "work_id", workID)
		return
	}
	if _, err := s.catalog.PutWorkState(ctx, accessToken, int64(workID), catState, completion); err != nil {
		if errors.Is(err, catalogclient.ErrInsufficientScope) {
			warnRatingWorkStateScope.warn("galgame rating: work-state sync unavailable, token lacks scope", "work_id", workID)
			return
		}
		slog.Warn("galgame rating: work-state sync failed", "work_id", workID, "error", err)
	}
}
