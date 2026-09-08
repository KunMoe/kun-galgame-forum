package service

import (
	"context"
	"errors"
	"log/slog"
	"sort"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/catalogclient"
	apperrors "kun-galgame-api/pkg/errors"
)

// One sweep of the upstream sync face. Its page size is 200 and it orders by
// updated_at ascending, so a library larger than this cannot be shown newest
// first without pulling all of it.
const (
	playtimeSweepPage  = 200
	playtimeSweepPages = 5
)

// Catalog has no delete endpoint: a report is withdrawn by dropping it under
// the floor its aggregate counts from. So a sub-floor row is a WITHDRAWN
// report, not a very short session, and every read has to treat it as absent.
// Clearing a record and then opening the profile page printed
// "1 部作品 · 合计 · 已通关 1 部" — the withdrawn work still counted twice, with
// no duration left to render between the separators.
func playtimeWithdrawn(minutes int) bool {
	return minutes < catalogclient.PlaytimeMinutesFloor
}

func (s *GalgameService) hydrateMyPlaytime(ctx context.Context, gid int, accessToken string) *dto.GalgameMyPlaytime {
	if s.catalog == nil || accessToken == "" || s.galgameClient == nil {
		return nil
	}
	ids, appErr := s.galgameClient.CatalogWorkIDs(ctx, []int{gid})
	if appErr != nil {
		return nil
	}
	workID, ok := ids[gid]
	if !ok {
		return nil
	}
	got, err := s.catalog.MyPlaytime(ctx, accessToken, workID)
	if err != nil {
		// A token minted before playtime joined the authorize scope is the
		// ordinary case here, not a fault: the detail page just shows no
		// personal row until the user signs in again. It used to log nothing at
		// all, and that is how the 2026-09-08 folder-scope outage stayed
		// invisible on the sibling call sites for an hour — so it is counted now
		// rather than swallowed.
		if errors.Is(err, catalogclient.ErrInsufficientScope) {
			warnPlaytimeScope.warn("galgame detail: own playtime unavailable, token lacks playtime:read", "gid", gid)
		} else {
			slog.Warn("galgame detail: own playtime unavailable", "gid", gid, "error", err)
		}
		return nil
	}
	if got == nil || playtimeWithdrawn(got.Minutes) {
		return nil
	}
	return &dto.GalgameMyPlaytime{Minutes: got.Minutes, Status: got.Status, Clients: got.Clients}
}

type PlaytimeService struct {
	galgameService *GalgameService
	galgameClient  *client.GalgameClient
	catalog        *catalogclient.Client
	forumClientID  string
}

func NewPlaytimeService(
	galgameService *GalgameService,
	galgameClient *client.GalgameClient,
	catalog *catalogclient.Client,
	forumClientID string,
) *PlaytimeService {
	return &PlaytimeService{
		galgameService: galgameService,
		galgameClient:  galgameClient,
		catalog:        catalog,
		forumClientID:  forumClientID,
	}
}

func (s *PlaytimeService) Report(
	ctx context.Context,
	gid int,
	accessToken string,
	minutes int,
	status string,
) (*dto.GalgameMyPlaytime, error) {
	ids, appErr := s.galgameClient.CatalogWorkIDs(ctx, []int{gid})
	if appErr != nil {
		return nil, appErr
	}
	workID, ok := ids[gid]
	if !ok {
		return nil, apperrors.ErrNotFound("条目不存在")
	}
	if _, err := s.catalog.ReportPlaytime(ctx, accessToken, workID,
		catalogclient.PlaytimeReport{Minutes: minutes, Status: status}); err != nil {
		return nil, err
	}
	// Read the fold back rather than echoing what we just wrote: another
	// application of the same user may hold a larger number, and catalog keeps
	// the largest. Showing the forum's own figure would contradict the page.
	got, err := s.catalog.MyPlaytime(ctx, accessToken, workID)
	if err != nil || got == nil {
		if playtimeWithdrawn(minutes) {
			return nil, nil
		}
		return &dto.GalgameMyPlaytime{Minutes: minutes, Status: status, Clients: 1}, nil
	}
	if playtimeWithdrawn(got.Minutes) {
		return nil, nil
	}
	return &dto.GalgameMyPlaytime{Minutes: got.Minutes, Status: got.Status, Clients: got.Clients}, nil
}

type foldedPlaytime struct {
	gid          int
	minutes      int
	status       string
	lastPlayedAt string
	updatedAt    string
	clients      int
	external     bool
}

func (s *PlaytimeService) ListMine(
	ctx context.Context,
	accessToken string,
	page, limit int,
	isSFW bool,
) (*dto.PlaytimeMinePage, error) {
	rows, truncated, err := s.sweep(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	folded := s.fold(ctx, rows)
	sort.Slice(folded, func(i, j int) bool { return folded[i].updatedAt > folded[j].updatedAt })

	out := &dto.PlaytimeMinePage{
		Items:     []dto.PlaytimeMineItem{},
		Total:     len(folded),
		Truncated: truncated,
	}
	for _, f := range folded {
		out.TotalMinutes += f.minutes
		if f.status == catalogclient.PlaytimeStatusFinished {
			out.FinishedWorks++
		}
	}

	start := (page - 1) * limit
	if start >= len(folded) {
		return out, nil
	}
	end := min(start+limit, len(folded))
	window := folded[start:end]

	gids := make([]int, 0, len(window))
	for _, f := range window {
		gids = append(gids, f.gid)
	}
	cards, appErr := s.galgameService.HydrateCardsByIDs(ctx, gids, isSFW)
	if appErr != nil {
		return nil, appErr
	}
	byID := make(map[int]dto.GalgameListCard, len(cards))
	for _, c := range cards {
		byID[c.ID] = c
	}
	for _, f := range window {
		card, ok := byID[f.gid]
		if !ok {
			continue
		}
		out.Items = append(out.Items, dto.PlaytimeMineItem{
			Galgame:      card,
			Minutes:      f.minutes,
			Status:       f.status,
			LastPlayedAt: f.lastPlayedAt,
			UpdatedAt:    f.updatedAt,
			Clients:      f.clients,
			External:     f.external,
		})
	}
	return out, nil
}

func (s *PlaytimeService) sweep(ctx context.Context, accessToken string) ([]catalogclient.PlaytimeRecord, bool, error) {
	var all []catalogclient.PlaytimeRecord
	cursor := ""
	for i := 0; i < playtimeSweepPages; i++ {
		rows, next, err := s.catalog.ListMyPlaytime(ctx, accessToken, cursor, playtimeSweepPage)
		if err != nil {
			return nil, false, err
		}
		all = append(all, rows...)
		if len(rows) < playtimeSweepPage || next == "" {
			return all, false, nil
		}
		cursor = next
	}
	return all, true, nil
}

// foldRecords collapses the per-(work, client) rows the sync face returns into
// one row per work, the same way catalog's own self-read does: MAX minutes, and
// finished wins over every other status.
func foldRecords(rows []catalogclient.PlaytimeRecord, forumClientID string) ([]int64, map[int64]foldedPlaytime) {
	byWork := make(map[int64]foldedPlaytime, len(rows))
	order := make([]int64, 0, len(rows))
	for _, r := range rows {
		cur, ok := byWork[r.WorkID]
		if !ok {
			order = append(order, r.WorkID)
		}
		cur.clients++
		if r.Minutes >= cur.minutes {
			cur.minutes = r.Minutes
			cur.external = r.ClientID != forumClientID
			if cur.status != catalogclient.PlaytimeStatusFinished {
				cur.status = r.Status
			}
		}
		if r.Status == catalogclient.PlaytimeStatusFinished {
			cur.status = catalogclient.PlaytimeStatusFinished
		}
		if r.LastPlayedAt != nil && *r.LastPlayedAt > cur.lastPlayedAt {
			cur.lastPlayedAt = *r.LastPlayedAt
		}
		if r.UpdatedAt > cur.updatedAt {
			cur.updatedAt = r.UpdatedAt
		}
		byWork[r.WorkID] = cur
	}
	return order, byWork
}

func (s *PlaytimeService) fold(ctx context.Context, rows []catalogclient.PlaytimeRecord) []foldedPlaytime {
	order, byWork := foldRecords(rows, s.forumClientID)
	gidByWork, appErr := s.galgameClient.GIDsByCatalogIDs(ctx, order)
	if appErr != nil {
		return nil
	}
	out := make([]foldedPlaytime, 0, len(order))
	for _, workID := range order {
		gid, ok := gidByWork[workID]
		if !ok || gid <= 0 {
			continue
		}
		f := byWork[workID]
		if playtimeWithdrawn(f.minutes) {
			continue
		}
		f.gid = gid
		out = append(out, f)
	}
	return out
}
