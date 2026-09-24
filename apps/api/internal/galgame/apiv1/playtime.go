package apiv1

import (
	"context"
	"log/slog"
	"sort"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/playstate"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"
)

// v2 /me/playtimes refuses limit>100 (400 LIMIT_TOO_LARGE) and pages with cursor=, not updated_since=.
const (
	playtimeSweepPage  = 100
	playtimeSweepPages = 10
)

type workPlaytimeInput struct {
	WorkID string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Work id."`
}

type putWorkPlaytimeInput struct {
	WorkID string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Work id."`
	Body   putWorkPlaytimeBody
}

type putWorkPlaytimeBody struct {
	Minutes   *int              `json:"minutes,omitempty" minimum:"0" maximum:"60000" doc:"Absolute cumulative minutes. 0 withdraws the duration. Omitted leaves minutes unchanged."`
	PlayState optionalPlayState `json:"play_state" required:"false" doc:"Play state to write. null deletes the work-state. Omitted leaves it unchanged. done is refused."`
}

type workViewerPlaytimeOutput struct {
	Body WorkViewerPlaytime
}

type listMyPlaytimesInput struct {
	Page        int  `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit       int  `query:"limit" minimum:"1" maximum:"100" default:"24" doc:"Page size. 1–100, default 24. Values above 100 are rejected, not clamped."`
	IncludeNSFW bool `query:"include_nsfw" default:"false" doc:"When true, adult works are included. Default false."`
}

type listMyPlaytimesOutput struct {
	Body WorkPlaytimeList
}

func (s *Service) putWorkPlaytime(ctx context.Context, in *putWorkPlaytimeInput) (*workViewerPlaytimeOutput, error) {
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	defer s.dropMyPlaytimeSweep(ctx, user.ID)
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	workID, ok := parseWorkID(in.WorkID)
	if !ok {
		return nil, notFound()
	}
	if _, p := s.catalogWork(ctx, workID); p != nil {
		return nil, p
	}
	if in.Body.Minutes == nil && !in.Body.PlayState.set {
		return nil, validationFailed(problem.AtPointer("", problem.ReasonRequired, "send minutes, play_state or both", nil))
	}
	if in.Body.PlayState.set && !in.Body.PlayState.null && in.Body.PlayState.value == "done" {
		return nil, validationFailed(problem.AtPointer("/play_state", problem.ReasonNotAllowedValue, "done is display-only", nil))
	}
	if s.catalog == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}

	if in.Body.Minutes != nil {
		if _, err := s.catalog.ReportPlaytime(ctx, token, int64(workID),
			catalogclient.PlaytimeReport{Minutes: *in.Body.Minutes}); err != nil {
			return nil, mapUserPlane(err, false)
		}
	}

	if in.Body.PlayState.set {
		if in.Body.PlayState.null {
			if err := s.catalog.DeleteWorkState(ctx, token, int64(workID)); err != nil {
				return nil, mapUserPlane(err, false)
			}
		} else {
			catState, completion, ok := playstate.ToCatalog(in.Body.PlayState.value)
			if !ok {
				allowed := playstate.All()
				return nil, validationFailed(problem.AtPointer("/play_state", problem.ReasonUnknownValue,
					"not in this field's closed vocabulary", &problem.FieldParams{Allowed: &allowed}))
			}
			// The user now picks 单线/主线/全线 directly, so their choice is
			// authoritative. Echoing a previously stored completion would
			// ignore the picker and wipe a different application's value
			// only by accident of GET-then-PUT.
			if _, err := s.catalog.PutWorkState(ctx, token, int64(workID), catState, completion); err != nil {
				return nil, mapUserPlane(err, false)
			}
		}
	}

	body, err := s.readViewerPlaytime(ctx, token, workID)
	if err != nil {
		return nil, err
	}
	return &workViewerPlaytimeOutput{Body: body}, nil
}

func (s *Service) deleteWorkPlaytime(ctx context.Context, in *workPlaytimeInput) (*workViewerPlaytimeOutput, error) {
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	defer s.dropMyPlaytimeSweep(ctx, user.ID)
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	workID, ok := parseWorkID(in.WorkID)
	if !ok {
		return nil, notFound()
	}
	if _, p := s.catalogWork(ctx, workID); p != nil {
		return nil, p
	}
	if s.catalog == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	if err := s.catalog.DeleteMyPlaytime(ctx, token, int64(workID)); err != nil {
		return nil, mapUserPlane(err, false)
	}
	if err := s.catalog.DeleteWorkState(ctx, token, int64(workID)); err != nil {
		return nil, mapUserPlane(err, false)
	}
	// Catalog's own playtime is the max across apps, so another app's minutes
	// survive the forum's withdraw; answering a hardcoded 0 would disagree with
	// the very next read of the work.
	body, err := s.readViewerPlaytime(ctx, token, workID)
	if err != nil {
		return nil, err
	}
	return &workViewerPlaytimeOutput{Body: body}, nil
}

func (s *Service) listMyPlaytimes(ctx context.Context, in *listMyPlaytimesInput) (*listMyPlaytimesOutput, error) {
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	if s.catalog == nil || s.hydrator == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	pg := browsePage(in.Page, in.Limit)
	if prob := pg.CheckDepth(); prob != nil {
		return nil, prob
	}

	sweep, err := s.myPlaytimeSweep(ctx, user.ID, token)
	if err != nil {
		return nil, mapUserPlane(err, false)
	}
	truncated := sweep.Truncated

	order, byWork := foldPlaytimeRecords(sweep.Playtimes)
	states, stateOrder := indexWorkStates(sweep.States)
	folded := assemblePlaytimes(order, byWork, stateOrder, states)

	ids := make([]int, 0, len(folded))
	for _, f := range folded {
		ids = append(ids, int(f.workID))
	}
	summaries, p := s.hydrator.ByIDs(ctx, ids, true)
	if p != nil {
		return nil, p
	}
	byID := map[int]workrepr.WorkSummary{}
	for _, sum := range summaries {
		id, ok := repr.ParseID(sum.ID)
		if ok {
			byID[id] = sum
		}
	}

	visible := make([]foldedPlaytime, 0, len(folded))
	for _, f := range folded {
		sum, ok := byID[int(f.workID)]
		if !ok {
			continue
		}
		if !in.IncludeNSFW && sum.IsNSFW {
			continue
		}
		f.summary = sum
		visible = append(visible, f)
	}

	totalMinutes := 0
	finished := 0
	for _, f := range visible {
		totalMinutes += f.minutes
		if playStateFinished(f.status) {
			finished++
		}
	}
	n, rel := collect.ClampTotal(len(visible))
	start := pg.Offset()
	items := []WorkPlaytime{}
	if start < len(visible) {
		end := min(start+pg.Limit, len(visible))
		for _, f := range visible[start:end] {
			var state *string
			if f.status != "" {
				st := f.status
				state = &st
			}
			items = append(items, WorkPlaytime{
				Object: "work_playtime", WorkSummary: f.summary, Minutes: f.minutes,
				PlayState: state, ClientCount: f.clients,
			})
		}
	}
	return &listMyPlaytimesOutput{Body: WorkPlaytimeList{
		Object: "list", Items: items, Total: n, TotalRelation: rel,
		TotalMinutes: totalMinutes, FinishedWorkCount: finished, IsTruncated: truncated,
	}}, nil
}

func (s *Service) readViewerPlaytime(ctx context.Context, token string, workID int) (WorkViewerPlaytime, error) {
	got, err := s.catalog.MyPlaytime(ctx, token, int64(workID))
	if err != nil {
		return WorkViewerPlaytime{}, mapUserPlane(err, false)
	}
	minutes := 0
	if got != nil {
		minutes = got.Minutes
	}
	if playtimeWithdrawn(minutes) {
		minutes = 0
	}
	var state *string
	ws, werr := s.catalog.MyWorkState(ctx, token, int64(workID))
	if werr != nil {
		slog.Warn("galgame playtime: own work-state unavailable after write", "work_id", workID, "upstream_status", upstreamStatus(werr), "err", werr)
	} else {
		state = playStateFromCatalog(ws)
	}
	return WorkViewerPlaytime{Minutes: minutes, PlayState: state}, nil
}

func (s *Service) sweepPlaytimes(ctx context.Context, token string) ([]catalogclient.PlaytimeRecord, bool, error) {
	var all []catalogclient.PlaytimeRecord
	cursor := ""
	for i := 0; i < playtimeSweepPages; i++ {
		rows, next, err := s.catalog.ListMyPlaytime(ctx, token, cursor, playtimeSweepPage)
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

func (s *Service) sweepWorkStates(ctx context.Context, token string) ([]catalogclient.WorkStateRecord, bool, error) {
	var all []catalogclient.WorkStateRecord
	cursor := ""
	for i := 0; i < playtimeSweepPages; i++ {
		rows, next, err := s.catalog.ListMyWorkStates(ctx, token, cursor, playtimeSweepPage)
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

func playStateFromCatalog(ws *catalogclient.WorkStateRecord) *string {
	if ws == nil {
		return nil
	}
	if ws.State == "" {
		return nil
	}
	flat := playstate.FromCatalog(ws.State, ws.Completion)
	if flat == "" {
		slog.Warn("playtime: unknown catalog work state, dropped", "work_id", ws.WorkID, "state", ws.State)
		return nil
	}
	return &flat
}

type foldedPlaytime struct {
	workID    int64
	minutes   int
	clients   int
	lastIndex int
	status    string
	summary   workrepr.WorkSummary
}

func foldPlaytimeRecords(rows []catalogclient.PlaytimeRecord) ([]int64, map[int64]foldedPlaytime) {
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

func assemblePlaytimes(
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
			if flat := playStateFromCatalog(&rec); flat != nil {
				f.status = *flat
			}
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
		flat := playStateFromCatalog(&rec)
		if flat == nil {
			continue
		}
		out = append(out, foldedPlaytime{workID: workID, minutes: 0, clients: 0, status: *flat})
	}
	return out
}
