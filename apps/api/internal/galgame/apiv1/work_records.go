package apiv1

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/playstate"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

const workRecordsScope = "user_work_records"

type WorkStateFilter string

func (WorkStateFilter) Schema(huma.Registry) *huma.Schema {
	s := repr.ClosedEnum(catalogclient.WorkStates()...)
	s.Description = "A play state on catalog's coarse axis. done covers every finish, whatever its completion."
	return s
}

type listUserWorkRecordsInput struct {
	UserID      string            `path:"user_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"User id."`
	State       []WorkStateFilter `query:"state" maxItems:"5" doc:"Only works in these play states, comma-separated. Omitted lists every state."`
	Cursor      string            `query:"cursor" pattern:"^cur_[A-Za-z0-9_-]+$" maxLength:"512" doc:"Opaque keyset cursor from a previous page of this collection."`
	Limit       int               `query:"limit" minimum:"1" maximum:"100" default:"24" doc:"Page size. 1–100, default 24. Values above 100 are rejected, not clamped."`
	IncludeNSFW bool              `query:"include_nsfw" default:"false" doc:"When true, adult works are included, in items and in state_counts. Default false."`
}

type UserWorkRecord struct {
	Object          string               `json:"object" enum:"work_record" maxLength:"11" doc:"Type discriminant. Always work_record."`
	WorkSummary     workrepr.WorkSummary `json:"work_summary" doc:"The work this record is about."`
	PlayState       *string              `json:"play_state" enum:"wish,doing,done_one_route,done_main,done_all,on_hold,dropped,done" maxLength:"14" doc:"The account's play state. done is a finish whose completion another application left unstated. Never null on this list, which holds only works with a play state."`
	StateUpdatedAt  repr.DateTime        `json:"state_updated_at" doc:"When the play state last changed. Items are ordered by it, newest first."`
	PlaytimeMinutes *int                 `json:"playtime_minutes" minimum:"10" doc:"The most minutes any application reported for the account. null when none is recorded or the report was withdrawn."`
	UserRating      *WorkRecordRating    `json:"user_rating" doc:"The account's rating of the work. null when it has none."`
}

type WorkRecordRating struct {
	ID      repr.DecimalID `json:"id" doc:"Rating id. GET /ratings/{rating_id} reads the rating."`
	Overall int            `json:"overall" minimum:"1" maximum:"10" doc:"The overall score."`
}

type WorkStateCounts struct {
	Wish    int `json:"wish" minimum:"0" doc:"Works the account wants to play."`
	Doing   int `json:"doing" minimum:"0" doc:"Works the account is playing."`
	Done    int `json:"done" minimum:"0" doc:"Works the account finished, whatever the completion."`
	OnHold  int `json:"on_hold" minimum:"0" doc:"Works the account put on hold."`
	Dropped int `json:"dropped" minimum:"0" doc:"Works the account dropped."`
}

type UserWorkRecordList struct {
	Object      string           `json:"object" enum:"list" maxLength:"4" doc:"Type discriminant. Always list."`
	Items       []UserWorkRecord `json:"items" doc:"Members of this page. Empty array, never null."`
	NextCursor  *string          `json:"next_cursor,omitempty" pattern:"^cur_[A-Za-z0-9_-]+$" maxLength:"512" doc:"Opaque keyset cursor. Omitted on the last page."`
	StateCounts *WorkStateCounts `json:"state_counts,omitempty" doc:"Works per play state, ignoring the state filter but not include_nsfw. Present on the first page only, the one requested without cursor. A work the forum cannot render is counted here but missing from items."`
}

type userWorkRecordListOutput struct {
	Body UserWorkRecordList
}

type PrivacySettings struct {
	Object              string         `json:"object" enum:"privacy_settings" maxLength:"16" doc:"Type discriminant. Always privacy_settings."`
	IsPlayRecordsPublic bool           `json:"is_play_records_public" doc:"Whether everyone can read the caller's play states and playtimes on every NextMoe site. true until the caller first turns it off. Ratings are public either way."`
	ChangedAt           *repr.DateTime `json:"changed_at" doc:"When the caller last changed a switch. null when never."`
}

type PrivacySettingsWrite struct {
	IsPlayRecordsPublic *bool `json:"is_play_records_public,omitempty" doc:"false keeps the caller's play states and playtimes to themselves on every NextMoe site; true makes them public again. Omitted leaves it unchanged."`
}

type privacySettingsOutput struct {
	Body PrivacySettings
}

type updatePrivacySettingsInput struct {
	Body PrivacySettingsWrite
}

func (s *Service) listUserWorkRecords(ctx context.Context, in *listUserWorkRecordsInput) (*userWorkRecordListOutput, error) {
	ownerID, ok := parseWorkID(in.UserID)
	if !ok {
		return nil, notFound()
	}
	if s.users == nil || s.catalog == nil || s.hydrator == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	_, renderable, p := s.lookupOwner(ctx, int64(ownerID))
	if p != nil {
		return nil, p
	}
	user := callerUser(ctx)
	isOwner := user != nil && user.ID == ownerID
	if !renderable && !isOwner {
		return nil, notFound()
	}

	states := make([]string, 0, len(in.State))
	for _, st := range in.State {
		if !slices.Contains(states, string(st)) {
			states = append(states, string(st))
		}
	}
	slices.Sort(states)
	fp := collect.Fingerprint(strconv.Itoa(ownerID), strings.Join(states, ","), strconv.FormatBool(in.IncludeNSFW))
	upstreamCursor, p := unwrapUpstreamCursor(in.Cursor, workRecordsScope, fp)
	if p != nil {
		return nil, p
	}

	token := ""
	if isOwner {
		if token, p = requireToken(ctx); p != nil {
			return nil, p
		}
	}
	firstPage := in.Cursor == ""
	q := catalogclient.WorkRecordsQuery{
		UserID:  int64(ownerID),
		States:  states,
		SFWOnly: !in.IncludeNSFW,
		Cursor:  upstreamCursor,
		Limit:   in.Limit,
		Counts:  firstPage && len(states) == 0,
	}
	page, err := s.catalog.WorkRecords(ctx, token, q)
	if err != nil {
		return nil, mapWorkRecordsErr(err, isOwner)
	}
	counts := page.StateCounts
	// Catalog's facet counts after the state filter like every v2 facet, so a
	// filtered tab's own page would read zero for every other tab.
	if firstPage && len(states) > 0 {
		all, err := s.catalog.WorkRecords(ctx, token, catalogclient.WorkRecordsQuery{
			UserID: int64(ownerID), SFWOnly: !in.IncludeNSFW, Limit: 1, Counts: true,
		})
		if err != nil {
			return nil, mapWorkRecordsErr(err, isOwner)
		}
		counts = all.StateCounts
	}

	ids := make([]int, 0, len(page.Items))
	for _, rec := range page.Items {
		ids = append(ids, int(rec.WorkID))
	}
	summaries, p := s.hydrator.ByIDs(ctx, ids, true)
	if p != nil {
		return nil, p
	}
	byID := make(map[int]workrepr.WorkSummary, len(summaries))
	for _, sum := range summaries {
		if id, ok := repr.ParseID(sum.ID); ok {
			byID[id] = sum
		}
	}

	items := make([]UserWorkRecord, 0, len(page.Items))
	for _, rec := range page.Items {
		sum, ok := byID[int(rec.WorkID)]
		if !ok || (!in.IncludeNSFW && sum.IsNSFW) {
			continue
		}
		state := playstate.FromCatalog(rec.State, rec.Completion)
		if state == "" {
			continue
		}
		item := UserWorkRecord{
			Object: "work_record", WorkSummary: sum, PlayState: &state,
			StateUpdatedAt: repr.Timestamp(rec.StateUpdatedAt),
		}
		if m := rec.PlaytimeMinutes; m != nil && !playtimeWithdrawn(*m) {
			item.PlaytimeMinutes = m
		}
		if r := rec.Rating; r != nil && r.Overall > 0 {
			item.UserRating = &WorkRecordRating{ID: repr.ID(int(r.ID)), Overall: r.Overall}
		}
		items = append(items, item)
	}

	out := UserWorkRecordList{
		Object:     "list",
		Items:      items,
		NextCursor: wrapUpstreamCursor(workRecordsScope, fp, page.NextCursor),
	}
	if counts != nil {
		out.StateCounts = &WorkStateCounts{
			Wish:    counts[catalogclient.WorkStateWish],
			Doing:   counts[catalogclient.WorkStateDoing],
			Done:    counts[catalogclient.WorkStateDone],
			OnHold:  counts[catalogclient.WorkStateOnHold],
			Dropped: counts[catalogclient.WorkStateDropped],
		}
	}
	return &userWorkRecordListOutput{Body: out}, nil
}

func mapWorkRecordsErr(err error, own bool) error {
	var api *catalogclient.UserAPIError
	if errors.As(err, &api) && api.Status == http.StatusForbidden && api.ProblemCode == catalogclient.CodePlayRecordsPrivate {
		return problem.New(problem.CodePlayRecordsPrivate, "The account keeps its play records private.")
	}
	return mapUserPlane(err, own)
}

func (s *Service) getMyPrivacySettings(ctx context.Context, _ *struct{}) (*privacySettingsOutput, error) {
	if _, p := s.requireActive(ctx); p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	if s.catalog == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	got, err := s.catalog.MyPrivacy(ctx, token)
	if err != nil {
		return nil, mapUserPlane(err, true)
	}
	return &privacySettingsOutput{Body: privacySettingsOf(got)}, nil
}

func (s *Service) updateMyPrivacySettings(ctx context.Context, in *updatePrivacySettingsInput) (*privacySettingsOutput, error) {
	if _, p := s.requireActive(ctx); p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	if in.Body.IsPlayRecordsPublic == nil {
		return nil, validationFailed(problem.AtPointer("", problem.ReasonRequired, "send at least one switch", nil))
	}
	if s.catalog == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	value := catalogclient.PlayRecordsPrivate
	if *in.Body.IsPlayRecordsPublic {
		value = catalogclient.PlayRecordsPublic
	}
	got, err := s.catalog.PatchMyPrivacy(ctx, token, value)
	if err != nil {
		return nil, mapUserPlane(err, true)
	}
	return &privacySettingsOutput{Body: privacySettingsOf(got)}, nil
}

func privacySettingsOf(p catalogclient.PrivacySettings) PrivacySettings {
	return PrivacySettings{
		Object:              "privacy_settings",
		IsPlayRecordsPublic: p.PlayRecords != catalogclient.PlayRecordsPrivate,
		ChangedAt:           repr.TimestampPtr(p.UpdatedAt),
	}
}
