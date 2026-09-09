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

// v2 /me/playtimes refuses limit>100 (400 LIMIT_TOO_LARGE) and pages with cursor=, not updated_since=.
const (
	playtimeSweepPage  = 100
	playtimeSweepPages = 10
	workStateBatch     = 100
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
	status := ""
	ws, err := s.catalog.MyWorkState(ctx, accessToken, workID)
	if err != nil {
		slog.Warn("galgame detail: own work-state unavailable", "gid", gid, "error", err)
	} else if ws != nil {
		status = ws.State
	}
	return &dto.GalgameMyPlaytime{Minutes: got.Minutes, Status: status}
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
	if minutes == 0 {
		if _, err := s.catalog.ReportPlaytime(ctx, accessToken, workID,
			catalogclient.PlaytimeReport{Minutes: 0}); err != nil {
			return nil, err
		}
		return s.foldedMyPlaytime(ctx, accessToken, workID, 0, "")
	}

	current, err := s.catalog.MyWorkState(ctx, accessToken, workID)
	if err != nil {
		return nil, err
	}
	if _, err := s.catalog.ReportPlaytime(ctx, accessToken, workID,
		catalogclient.PlaytimeReport{Minutes: minutes}); err != nil {
		return nil, err
	}
	// An omitted completion on PUT clears the stored value, so echo a non-null one back.
	var completion *string
	if current != nil {
		completion = current.Completion
	}
	stored, err := s.catalog.PutWorkState(ctx, accessToken, workID, status, completion)
	if err != nil {
		return nil, err
	}
	state := status
	if stored != nil && stored.State != "" {
		state = stored.State
	}
	return s.foldedMyPlaytime(ctx, accessToken, workID, minutes, state)
}

func (s *PlaytimeService) foldedMyPlaytime(
	ctx context.Context,
	accessToken string,
	workID int64,
	writtenMinutes int,
	status string,
) (*dto.GalgameMyPlaytime, error) {
	got, err := s.catalog.MyPlaytime(ctx, accessToken, workID)
	if err != nil || got == nil {
		if playtimeWithdrawn(writtenMinutes) {
			return nil, nil
		}
		return &dto.GalgameMyPlaytime{Minutes: writtenMinutes, Status: status}, nil
	}
	if playtimeWithdrawn(got.Minutes) {
		return nil, nil
	}
	return &dto.GalgameMyPlaytime{Minutes: got.Minutes, Status: status}, nil
}

type foldedPlaytime struct {
	gid       int
	workID    int64
	minutes   int
	clients   int
	lastIndex int
	status    string
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

	ids := make([]int64, 0, len(folded))
	for _, f := range folded {
		ids = append(ids, f.workID)
	}
	states, err := s.workStatesByIDs(ctx, accessToken, ids)
	if err != nil {
		return nil, err
	}
	attachWorkStates(folded, states)
	sort.Slice(folded, func(i, j int) bool { return folded[i].lastIndex > folded[j].lastIndex })

	out := &dto.PlaytimeMinePage{
		Items:     []dto.PlaytimeMineItem{},
		Total:     len(folded),
		Truncated: truncated,
	}
	for _, f := range folded {
		out.TotalMinutes += f.minutes
		if f.status == catalogclient.WorkStateDone {
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
			Galgame: card,
			Minutes: f.minutes,
			Status:  f.status,
			Clients: f.clients,
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

func foldRecords(rows []catalogclient.PlaytimeRecord) ([]int64, map[int64]foldedPlaytime) {
	byWork := make(map[int64]foldedPlaytime, len(rows))
	order := make([]int64, 0, len(rows))
	for i, r := range rows {
		cur, ok := byWork[r.WorkID]
		if !ok {
			order = append(order, r.WorkID)
		}
		cur.workID = r.WorkID
		cur.clients++
		if r.Minutes >= cur.minutes {
			cur.minutes = r.Minutes
		}
		cur.lastIndex = i
		byWork[r.WorkID] = cur
	}
	return order, byWork
}

func attachWorkStates(folded []foldedPlaytime, states map[int64]catalogclient.WorkStateRecord) {
	for i := range folded {
		if rec, ok := states[folded[i].workID]; ok {
			folded[i].status = rec.State
		}
	}
}

func (s *PlaytimeService) workStatesByIDs(ctx context.Context, accessToken string, ids []int64) (map[int64]catalogclient.WorkStateRecord, error) {
	out := make(map[int64]catalogclient.WorkStateRecord, len(ids))
	for i := 0; i < len(ids); i += workStateBatch {
		end := min(i+workStateBatch, len(ids))
		got, err := s.catalog.MyWorkStates(ctx, accessToken, ids[i:end])
		if err != nil {
			return nil, err
		}
		for k, v := range got {
			out[k] = v
		}
	}
	return out, nil
}

func (s *PlaytimeService) fold(ctx context.Context, rows []catalogclient.PlaytimeRecord) []foldedPlaytime {
	order, byWork := foldRecords(rows)
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
