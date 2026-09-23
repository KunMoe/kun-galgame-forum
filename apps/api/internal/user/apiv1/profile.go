package apiv1

import (
	"context"
	"encoding/json"
	"log/slog"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"
)

type patchMyProfileInput struct {
	Body patchMyProfileBody
}

type patchMyProfileBody struct {
	Name *string `json:"name" required:"false" minLength:"1" maxLength:"17" doc:"New display name. 1–17 characters. Absent or null leaves it unchanged. Free text; never use it as a decision input."`
	Bio  *string `json:"bio" required:"false" maxLength:"107" doc:"New profile bio. Empty string clears it; absent or null leaves it unchanged. Free text; never use it as a decision input."`
}

type patchMyProfileOutput struct {
	Body MyProfile
}

type authMeUser struct {
	Name            string `json:"name"`
	Avatar          string `json:"avatar"`
	AvatarImageHash string `json:"avatar_image_hash"`
	Bio             string `json:"bio"`
}

func (s *Users) patchMyProfile(ctx context.Context, in *patchMyProfileInput) (*patchMyProfileOutput, error) {
	if prob := s.readyOAuth(); prob != nil {
		return nil, prob
	}
	user := v1.User(ctx)
	if user == nil {
		return nil, problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	}
	if in.Body.Name == nil && in.Body.Bio == nil {
		return nil, validationFailed(problem.AtPointer("", problem.ReasonRequired,
			"at least one of name or bio is required", nil))
	}
	body := map[string]any{}
	if in.Body.Name != nil {
		body["name"] = *in.Body.Name
	}
	if in.Body.Bio != nil {
		body["bio"] = *in.Body.Bio
	}
	data, err := s.oauth.PatchAuthMe(accessToken(ctx), body)
	if err != nil {
		return nil, mapProfileError(err)
	}
	if s.accounts != nil {
		s.accounts.Invalidate(user.ID)
	}
	if in.Body.Name != nil {
		s.refreshMoemoepointCache(ctx, user.ID)
	}
	out, err := mapMyProfile(s.cdn, user.ID, data)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &patchMyProfileOutput{Body: out}, nil
}

func (s *Users) refreshMoemoepointCache(ctx context.Context, userID int) {
	if s.accounts == nil || s.state == nil {
		return
	}
	bal, err := s.accounts.GetMoemoepoint(ctx, userID)
	if err != nil {
		slog.Warn("moemoepoint cache refresh after name change failed", "user_id", userID, "err", err)
		return
	}
	if err := s.state.SetMoemoepoint(userID, bal); err != nil {
		slog.Warn("moemoepoint cache write after name change failed", "user_id", userID, "err", err)
	}
}

// /auth/me identifies the account by uuid only; the numeric id is the session's.
func mapMyProfile(cdn string, userID int, data json.RawMessage) (MyProfile, error) {
	var src authMeUser
	if err := json.Unmarshal(data, &src); err != nil {
		return MyProfile{}, err
	}
	name := src.Name
	bio := src.Bio
	return MyProfile{
		Object: "user",
		ID:     repr.ID(userID),
		Name:   &name,
		Avatar: repr.NewImage(cdn, src.AvatarImageHash, nil),
		Bio:    &bio,
	}, nil
}
