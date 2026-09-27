package apiv1

import (
	"context"
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"

	"github.com/danielgtaylor/huma/v2"
)

const activitySettingsPath = "/me/activity-settings"

type ActivitySettings struct {
	Object   string `json:"object" enum:"activity_settings" maxLength:"17" doc:"Type discriminant. Always activity_settings."`
	IsHidden bool   `json:"is_hidden" doc:"Whether the caller's activity is hidden from everyone else on every NextMoe site. false until the caller first sets it."`
}

type ActivitySettingsWrite struct {
	IsHidden bool `json:"is_hidden" doc:"true hides the caller's activity from everyone else; false shows it again."`
}

type activitySettingsOutput struct {
	Body ActivitySettings
}

type putActivitySettingsInput struct {
	Body ActivitySettingsWrite
}

func registerActivitySettings(api huma.API, s *Service) {
	tags := []string{"activities"}
	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "getActivitySettings",
		Method:      http.MethodGet,
		Path:        activitySettingsPath,
		Summary:     "Get whether the caller hides their activity",
		Description: "The switch is stored by NextMoe community and shared by every NextMoe site.",
		Tags:        tags,
		Responses: problemResponses(map[int]string{
			503: followingDown,
		}),
	}), s.getActivitySettings)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "putActivitySettings",
		Method:      http.MethodPut,
		Path:        activitySettingsPath,
		Summary:     "Hide or show the caller's activity",
		Description: "While hidden, on every NextMoe site: the caller's groups leave every follower's following feed and unseen count, " +
			"listActivityGroupItems answers NOT_FOUND for them to anyone but the caller, and publishing notifies no follower. " +
			"Turning it on also withdraws every followee_activity_published notification the caller raised, read ones included. " +
			"Turning it off shows the activity again at once; withdrawn notifications do not come back, and nothing published while hidden ever notifies. " +
			"The caller's profile and their own view of their groups are unaffected.",
		Tags: tags,
		Responses: problemResponses(map[int]string{
			422: "VALIDATION_FAILED when is_hidden is missing or not a boolean.",
			503: followingDown,
		}),
	}), s.putActivitySettings)
}

func (s *Service) getActivitySettings(ctx context.Context, _ *struct{}) (*activitySettingsOutput, error) {
	if prob := s.readyFollowing(); prob != nil {
		return nil, prob
	}
	got, err := s.community.GetActivitySettings(ctx, int64(v1.User(ctx).ID))
	if err != nil {
		return nil, communityReadProblem(err, false)
	}
	return &activitySettingsOutput{Body: ActivitySettings{Object: "activity_settings", IsHidden: got.Hidden}}, nil
}

func (s *Service) putActivitySettings(ctx context.Context, in *putActivitySettingsInput) (*activitySettingsOutput, error) {
	if prob := s.readyFollowing(); prob != nil {
		return nil, prob
	}
	got, err := s.community.PutActivitySettings(ctx, int64(v1.User(ctx).ID), in.Body.IsHidden)
	if err != nil {
		return nil, communityReadProblem(err, false)
	}
	return &activitySettingsOutput{Body: ActivitySettings{Object: "activity_settings", IsHidden: got.Hidden}}, nil
}
