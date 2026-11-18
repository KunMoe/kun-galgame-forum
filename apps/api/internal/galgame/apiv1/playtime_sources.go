package apiv1

import (
	"context"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"
)

type PlaytimeSource struct {
	Object       string         `json:"object" enum:"playtime_source" maxLength:"15" doc:"Type discriminant. Always playtime_source."`
	ClientID     string         `json:"client_id" maxLength:"128" pattern:"^[A-Za-z0-9._~-]+$" doc:"The OAuth client of the app that reported these minutes."`
	AppName      *string        `json:"app_name" maxLength:"128" doc:"That app's display name. null once the app is gone; show a localized label. Free text; never use it as a decision input."`
	Minutes      int            `json:"minutes" minimum:"0" doc:"Minutes that app reported."`
	LastPlayedAt *repr.DateTime `json:"last_played_at" doc:"When that app last saw the game played. null when it did not say."`
	UpdatedAt    repr.DateTime  `json:"updated_at" doc:"When that app last reported."`
	IsSelf       bool           `json:"is_self" doc:"Whether this site reported the row. Writing a playtime here changes only this row."`
}

type playtimeSourcesOutput struct {
	Body repr.List[PlaytimeSource]
}

type deletePlaytimeSourceInput struct {
	WorkID string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Work id."`
	Client string `path:"client" pattern:"^[A-Za-z0-9._~-]+$" maxLength:"128" doc:"client_id of the app whose minutes to delete."`
}

type playtimeSourceCall struct {
	userID int
	token  string
	workID int64
}

// playtimeSourceWork resolves what all three source routes need before
// calling catalog: an active caller, their token, and a visible work.
func (s *Service) playtimeSourceWork(ctx context.Context, rawWorkID string) (playtimeSourceCall, *problem.Problem) {
	user, p := s.requireActive(ctx)
	if p != nil {
		return playtimeSourceCall{}, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return playtimeSourceCall{}, p
	}
	workID, ok := parseWorkID(rawWorkID)
	if !ok {
		return playtimeSourceCall{}, notFound()
	}
	if _, p := s.catalogWork(ctx, workID); p != nil {
		return playtimeSourceCall{}, p
	}
	if s.catalog == nil {
		return playtimeSourceCall{}, problem.Unavailable(errUnconfigured)
	}
	return playtimeSourceCall{userID: user.ID, token: token, workID: int64(workID)}, nil
}

func (s *Service) listWorkPlaytimeSources(ctx context.Context, in *workPlaytimeInput) (*playtimeSourcesOutput, error) {
	call, p := s.playtimeSourceWork(ctx, in.WorkID)
	if p != nil {
		return nil, p
	}
	rows, err := s.catalog.MyPlaytimeSources(ctx, call.token, call.workID)
	if err != nil {
		return nil, mapUserPlane(err, true)
	}
	items := make([]PlaytimeSource, 0, len(rows))
	for _, r := range rows {
		item := PlaytimeSource{
			Object: "playtime_source", ClientID: r.ClientID, AppName: r.AppName, Minutes: r.Minutes,
			UpdatedAt: repr.Timestamp(r.UpdatedAt), IsSelf: r.Self,
		}
		if r.LastPlayedAt != nil {
			at := repr.Timestamp(*r.LastPlayedAt)
			item.LastPlayedAt = &at
		}
		items = append(items, item)
	}
	return &playtimeSourcesOutput{Body: repr.NewList(items, nil)}, nil
}

func (s *Service) deleteWorkPlaytimeSource(ctx context.Context, in *deletePlaytimeSourceInput) (*workViewerPlaytimeOutput, error) {
	return s.deletePlaytimeSources(ctx, in.WorkID, func(token string, workID int64) error {
		return s.catalog.DeleteMyPlaytimeSource(ctx, token, workID, in.Client)
	})
}

func (s *Service) deleteWorkPlaytimeSources(ctx context.Context, in *workPlaytimeInput) (*workViewerPlaytimeOutput, error) {
	return s.deletePlaytimeSources(ctx, in.WorkID, func(token string, workID int64) error {
		return s.catalog.DeleteMyPlaytimeSources(ctx, token, workID)
	})
}

// The answer is read back from catalog: deleting one app's row leaves the
// merged minutes at the most any remaining app reported.
func (s *Service) deletePlaytimeSources(ctx context.Context, rawWorkID string,
	del func(token string, workID int64) error) (*workViewerPlaytimeOutput, error) {

	call, p := s.playtimeSourceWork(ctx, rawWorkID)
	if p != nil {
		return nil, p
	}
	defer s.dropMyPlaytimeSweep(ctx, call.userID)
	if err := del(call.token, call.workID); err != nil {
		return nil, mapUserPlane(err, true)
	}
	body, err := s.readViewerPlaytime(ctx, call.token, int(call.workID))
	if err != nil {
		return nil, err
	}
	return &workViewerPlaytimeOutput{Body: body}, nil
}
