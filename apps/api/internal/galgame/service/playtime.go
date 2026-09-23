package service

import (
	"context"
	"errors"
	"log/slog"
	"sort"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/playstate"
	"kun-galgame-api/pkg/catalogclient"
	apperrors "kun-galgame-api/pkg/errors"
)

// v2 /me/playtimes refuses limit>100 (400 LIMIT_TOO_LARGE) and pages with cursor=, not updated_since=.
const (
	playtimeSweepPage  = 100
	playtimeSweepPages = 10
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

func playStateFinished(status string) bool {
	switch status {
	case playstate.DoneOneRoute, playstate.DoneMain, playstate.DoneAll:
		return true
	default:
		return false
	}
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
	workID int,
	accessToken string,
	minutes *int,
	status *string,
) (*dto.GalgameMyPlaytime, error) {
	id := int64(workID)

	writtenMinutes := 0
	if minutes != nil {
		if _, err := s.catalog.ReportPlaytime(ctx, accessToken, id,
			catalogclient.PlaytimeReport{Minutes: *minutes}); err != nil {
			return nil, err
		}
		writtenMinutes = *minutes
	}

	flat := ""
	if status != nil {
		if *status == "" {
			if err := s.catalog.DeleteWorkState(ctx, accessToken, id); err != nil {
				return nil, err
			}
		} else {
			catState, completion, ok := playstate.ToCatalog(*status)
			if !ok {
				return nil, apperrors.ErrBadRequest("未知的游玩状态")
			}
			// The user now picks 单线/主线/全线 directly, so their choice is
			// authoritative. Echoing a previously stored completion would
			// ignore the picker and wipe a different application's value
			// only by accident of GET-then-PUT.
			stored, err := s.catalog.PutWorkState(ctx, accessToken, id, catState, completion)
			if err != nil {
				return nil, err
			}
			if stored != nil {
				flat = playstate.FromCatalog(stored.State, stored.Completion)
			} else {
				flat = *status
			}
		}
	} else {
		ws, err := s.catalog.MyWorkState(ctx, accessToken, id)
		if err != nil {
			slog.Warn("galgame playtime: own work-state unavailable after report", "error", err)
		} else if ws != nil {
			flat = playstate.FromCatalog(ws.State, ws.Completion)
		}
	}

	return s.foldedMyPlaytime(ctx, accessToken, id, writtenMinutes, flat)
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

func (s *PlaytimeService) foldedMyPlaytime(
	ctx context.Context,
	accessToken string,
	workID int64,
	writtenMinutes int,
	status string,
) (*dto.GalgameMyPlaytime, error) {
	got, err := s.catalog.MyPlaytime(ctx, accessToken, workID)
	minutes := writtenMinutes
	if err == nil && got != nil {
		minutes = got.Minutes
	}
	if playtimeWithdrawn(minutes) {
		minutes = 0
	}
	if minutes == 0 && status == "" {
		return nil, nil
	}
	return &dto.GalgameMyPlaytime{Minutes: minutes, Status: status}, nil
}

type foldedPlaytime struct {
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
	playtimeRows, playtimeTruncated, err := s.sweep(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	stateRows, stateTruncated, err := s.sweepWorkStates(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	order, byWork := foldRecords(playtimeRows)
	states, stateOrder := indexWorkStates(stateRows)

	folded := assembleMine(order, byWork, stateOrder, states)

	out := &dto.PlaytimeMinePage{
		Items:     []dto.PlaytimeMineItem{},
		Total:     len(folded),
		Truncated: playtimeTruncated || stateTruncated,
	}
	for _, f := range folded {
		out.TotalMinutes += f.minutes
		if playStateFinished(f.status) {
			out.FinishedWorks++
		}
	}

	start := (page - 1) * limit
	if start >= len(folded) {
		return out, nil
	}
	end := min(start+limit, len(folded))
	window := folded[start:end]

	workIDs := make([]int, 0, len(window))
	for _, f := range window {
		workIDs = append(workIDs, int(f.workID))
	}
	cards, appErr := s.galgameService.HydrateCardsByIDs(ctx, workIDs, isSFW)
	if appErr != nil {
		return nil, appErr
	}
	byID := make(map[int]dto.GalgameListCard, len(cards))
	for _, c := range cards {
		byID[c.ID] = c
	}
	for _, f := range window {
		card, ok := byID[int(f.workID)]
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

func (s *PlaytimeService) sweepWorkStates(ctx context.Context, accessToken string) ([]catalogclient.WorkStateRecord, bool, error) {
	var all []catalogclient.WorkStateRecord
	cursor := ""
	for i := 0; i < playtimeSweepPages; i++ {
		rows, next, err := s.catalog.ListMyWorkStates(ctx, accessToken, cursor, playtimeSweepPage)
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

func indexWorkStates(rows []catalogclient.WorkStateRecord) (map[int64]catalogclient.WorkStateRecord, []int64) {
	byWork := make(map[int64]catalogclient.WorkStateRecord, len(rows))
	order := make([]int64, 0, len(rows))
	for _, r := range rows {
		if _, ok := byWork[r.WorkID]; !ok {
			order = append(order, r.WorkID)
		}
		byWork[r.WorkID] = r
	}
	return byWork, order
}

func assembleMine(
	playtimeOrder []int64,
	byWork map[int64]foldedPlaytime,
	stateOrder []int64,
	states map[int64]catalogclient.WorkStateRecord,
) []foldedPlaytime {
	derived := make([]foldedPlaytime, 0, len(playtimeOrder))
	seen := make(map[int64]struct{}, len(playtimeOrder)+len(stateOrder))
	for _, workID := range playtimeOrder {
		if workID <= 0 {
			continue
		}
		f := byWork[workID]
		if rec, ok := states[workID]; ok {
			f.status = playstate.FromCatalog(rec.State, rec.Completion)
		}
		if playtimeWithdrawn(f.minutes) {
			if f.status == "" {
				continue
			}
			f.minutes = 0
		}
		derived = append(derived, f)
		seen[workID] = struct{}{}
	}
	sort.Slice(derived, func(i, j int) bool { return derived[i].lastIndex > derived[j].lastIndex })

	out := derived
	for _, workID := range stateOrder {
		if _, ok := seen[workID]; ok {
			continue
		}
		if workID <= 0 {
			continue
		}
		rec := states[workID]
		status := playstate.FromCatalog(rec.State, rec.Completion)
		if status == "" {
			continue
		}
		out = append(out, foldedPlaytime{
			workID:  workID,
			minutes: 0,
			clients: 0,
			status:  status,
		})
	}
	return out
}
