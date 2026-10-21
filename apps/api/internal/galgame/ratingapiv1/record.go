package ratingapiv1

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"unicode/utf8"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/playstate"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"
)

// Catalog stores a report under this floor but leaves it out of every median,
// so the forum reads one as no playtime.
const playtimeFloor = 10

type WorkRecord struct {
	Object        string            `json:"object" enum:"work_record" maxLength:"11" doc:"Type discriminant. Always work_record."`
	WorkID        repr.DecimalID    `json:"work_id" doc:"The work."`
	PlayState     *AuthorPlayStatus `json:"play_state" doc:"The caller's play state on the work. null when none."`
	Minutes       int               `json:"minutes" minimum:"0" maximum:"60000" doc:"The most minutes any NextMoe application reported for the caller on the work. 0 when none, or when the report is below 10 minutes, which no median counts."`
	RatingSummary *RatingSummary    `json:"rating_summary" doc:"The caller's rating of the work, the same on every NextMoe site. null when none."`
}

type RecordRating struct {
	Recommend    Recommend     `json:"recommend"`
	Overall      int           `json:"overall" minimum:"1" maximum:"10" doc:"The overall score."`
	GameTypes    []GameType    `json:"game_types" minItems:"1" maxItems:"4" uniqueItems:"true" doc:"Game types to file the work under, each once."`
	SpoilerLevel SpoilerLevel  `json:"spoiler_level"`
	ShortSummary *string       `json:"short_summary,omitempty" maxLength:"2000" doc:"Short review, plain text, at most 2000 characters. Absent or empty means none. Free text; never use it as a decision input."`
	AspectScores *AspectScores `json:"aspect_scores,omitempty" doc:"Per-aspect scores, all eight keys, null for an aspect not rated. Absent means none rated."`
}

// WorkRecordPut replaces the whole record, as NextMoe does: a part left out is
// deleted. A client sends every part its form shows.
type WorkRecordPut struct {
	PlayState   *AuthorPlayStatus `json:"play_state" required:"false" doc:"The play state. Absent or null deletes it. done records a finish with the completion unstated; send it only to keep a done the record was read with."`
	Minutes     *int              `json:"minutes,omitempty" minimum:"1" maximum:"60000" doc:"This site's playtime report, in minutes. Absent withdraws it; reports from other applications stay. Below 10 is stored but counted by no median."`
	RatingInput *RecordRating     `json:"rating_input,omitempty" doc:"The rating. Absent deletes it."`
}

type workRecordInput struct {
	WorkID string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Work id."`
}

type putWorkRecordInput struct {
	WorkID string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Work id."`
	Body   WorkRecordPut
}

type workRecordOutput struct {
	Body WorkRecord
}

func (s *Service) recordWork(ctx context.Context, raw string) (int, string, *problem.Problem) {
	if !s.writable() {
		return 0, "", problem.Internal(errUnconfigured)
	}
	viewer := v1.User(ctx)
	if p := s.requireActive(ctx, viewer); p != nil {
		return 0, "", p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return 0, "", p
	}
	workID, ok := repr.ParseID(repr.DecimalID(raw))
	if !ok {
		return 0, "", notFound()
	}
	works, p := s.visibleWorks(ctx, []int{workID})
	if p != nil {
		return 0, "", p
	}
	if _, ok := works[workID]; !ok {
		return 0, "", notFound()
	}
	return workID, token, nil
}

func (s *Service) getWorkRecord(ctx context.Context, in *workRecordInput) (*workRecordOutput, error) {
	workID, token, p := s.recordWork(ctx, in.WorkID)
	if p != nil {
		return nil, p
	}
	rec, err := s.Catalog.WorkRecord(ctx, token, int64(workID))
	if err != nil {
		return nil, upstream(err)
	}
	// A rating written on another site minutes ago may not have reached the
	// mirror yet; the form still has to show it.
	if rec.Rating != nil {
		if _, err := s.Store.Get(int(rec.Rating.ID)); errors.Is(err, repository.ErrRatingNotFound) {
			s.mirror(rec.Rating, nil)
		}
	}
	out, p := s.recordView(ctx, workID, rec)
	if p != nil {
		return nil, p
	}
	return &workRecordOutput{Body: *out}, nil
}

func (s *Service) putWorkRecord(ctx context.Context, in *putWorkRecordInput) (*workRecordOutput, error) {
	workID, token, p := s.recordWork(ctx, in.WorkID)
	if p != nil {
		return nil, p
	}
	viewer := v1.User(ctx)
	b := in.Body
	put := catalogclient.WorkRecordInput{}
	if b.PlayState != nil {
		state, completion, _ := playstate.ToCatalog(string(*b.PlayState))
		put.WorkState = &catalogclient.WorkRecordState{State: state}
		if completion != nil {
			put.WorkState.Completion = *completion
		}
	}
	if b.Minutes != nil {
		put.Playtime = &catalogclient.WorkRecordPlaytime{Minutes: *b.Minutes}
	}
	summary := ""
	if r := b.RatingInput; r != nil {
		if r.ShortSummary != nil {
			summary = *r.ShortSummary
		}
		if !utf8.ValidString(summary) {
			return nil, invalidSummary()
		}
		put.Rating = &catalogclient.RatingInput{
			Overall: r.Overall, AspectScores: aspectsInput(r.AspectScores), Recommend: string(r.Recommend),
			SpoilerLevel: string(r.SpoilerLevel), ShortSummary: summary,
		}
	}

	before, err := s.Catalog.WorkRecord(ctx, token, int64(workID))
	if err != nil {
		return nil, upstream(err)
	}
	summaryChanged := put.Rating != nil && (before.Rating == nil || before.Rating.ShortSummary != summary)
	if summaryChanged {
		if p := s.checkText(ctx, summary, viewer.ID); p != nil {
			return nil, p
		}
	}
	after, err := s.Catalog.PutWorkRecord(ctx, token, int64(workID), put)
	if err != nil {
		return nil, upstream(err)
	}

	if after.Rating != nil {
		var types []GameType
		if b.RatingInput != nil {
			types = b.RatingInput.GameTypes
		}
		s.mirror(after.Rating, types)
	} else if err := s.Store.DropUserRating(viewer.ID, workID); err != nil {
		slog.Error("work record: local rating not dropped; the mirror job will drop it", "work_id", workID, "error", err)
	}
	s.reward(viewer.ID, before.Rating, after.Rating)
	if summaryChanged && after.Rating != nil {
		s.Scan.ScanBg(gate.SubjectKindGalgameRating, strconv.FormatInt(after.Rating.ID, 10), summary, int64(viewer.ID))
	}
	if s.RecordWritten != nil {
		s.RecordWritten(ctx, viewer.ID)
	}
	out, p := s.recordView(ctx, workID, after)
	if p != nil {
		return nil, p
	}
	return &workRecordOutput{Body: *out}, nil
}

func (s *Service) recordView(ctx context.Context, workID int, rec *catalogclient.WorkRecord) (*WorkRecord, *problem.Problem) {
	out := &WorkRecord{Object: "work_record", WorkID: repr.ID(workID)}
	if ws := rec.WorkState; ws != nil {
		if flat := playstate.FromCatalog(ws.State, ws.Completion); flat != "" {
			v := AuthorPlayStatus(flat)
			out.PlayState = &v
		}
	}
	if pt := rec.Playtime; pt != nil && pt.Minutes >= playtimeFloor {
		out.Minutes = pt.Minutes
	}
	if rec.Rating == nil {
		return out, nil
	}
	row, err := s.Store.Get(int(rec.Rating.ID))
	if errors.Is(err, repository.ErrRatingNotFound) {
		return out, nil
	}
	if err != nil {
		return nil, problem.Internal(err)
	}
	items, p := s.summaries(ctx, []repository.RatingRecord{row}, false)
	if p != nil {
		return nil, p
	}
	if len(items) == 1 {
		out.RatingSummary = &items[0]
	}
	return out, nil
}
