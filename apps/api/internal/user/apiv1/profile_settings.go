package apiv1

import (
	"context"
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

const profileSettingsPath = "/me/profile-settings"

type ProfileSettings struct {
	Object           string `json:"object" enum:"profile_settings" maxLength:"16" doc:"Type discriminant. Always profile_settings."`
	IsSignInRequired bool   `json:"is_sign_in_required" doc:"Whether the caller's profile is shown to signed-in users only. false until the caller first sets it."`
}

type ProfileSettingsWrite struct {
	IsSignInRequired bool `json:"is_sign_in_required" doc:"true shows the caller's profile to signed-in users only; false shows it to everyone."`
}

type profileSettingsOutput struct {
	Body ProfileSettings
}

type putProfileSettingsInput struct {
	Body ProfileSettingsWrite
}

func (s *Users) registerProfileSettings(api huma.API) {
	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "getProfileSettings",
		Method:      http.MethodGet,
		Path:        profileSettingsPath,
		Summary:     "Get who may read the caller's profile",
		Description: "Returns whether the caller's profile is shown to signed-in users only. An account with no forum state reads false.",
		Tags:        []string{"users"},
	}), s.getProfileSettings)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "putProfileSettings",
		Method:      http.MethodPut,
		Path:        profileSettingsPath,
		Summary:     "Set who may read the caller's profile",
		Description: "While is_sign_in_required is true, getUser, every /users/{user_id}/* list, and listRatings and listQuizzes filtered by the caller as author_id " +
			"answer an anonymous caller PROFILE_SIGN_IN_REQUIRED. Every signed-in caller still reads them. " +
			"The caller's content stays where it was published: topics, replies, comments and resources still show the caller as their author.",
		Tags: []string{"users"},
		Responses: problemResponses(map[int]string{
			422: "VALIDATION_FAILED when is_sign_in_required is missing or not a boolean.",
		}),
	}), s.putProfileSettings)
}

func (s *Users) getProfileSettings(ctx context.Context, _ *struct{}) (*profileSettingsOutput, error) {
	if prob := s.readyUsers(); prob != nil {
		return nil, prob
	}
	required, err := s.users.ProfileSignInRequired(ctx, v1.User(ctx).ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return profileSettings(required), nil
}

func (s *Users) putProfileSettings(ctx context.Context, in *putProfileSettingsInput) (*profileSettingsOutput, error) {
	if prob := s.readyUsers(); prob != nil {
		return nil, prob
	}
	required := in.Body.IsSignInRequired
	if err := s.users.SetProfileSignInRequired(v1.User(ctx).ID, required); err != nil {
		return nil, problem.Internal(err)
	}
	return profileSettings(required), nil
}

func profileSettings(required bool) *profileSettingsOutput {
	return &profileSettingsOutput{Body: ProfileSettings{Object: "profile_settings", IsSignInRequired: required}}
}
