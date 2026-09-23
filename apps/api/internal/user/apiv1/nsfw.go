package apiv1

import (
	"context"
	"log/slog"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/problem"
)

type putNsfwDisplayInput struct {
	Body putNsfwDisplayBody
}

type putNsfwDisplayBody struct {
	NsfwDisplay string `json:"nsfw_display" enum:"hide,blur,show" maxLength:"4" doc:"How adult content is shown: hide, blur, or show."`
}

type putNsfwDisplayOutput struct {
	Body NsfwDisplay
}

func (s *Users) putNsfwDisplay(ctx context.Context, in *putNsfwDisplayInput) (*putNsfwDisplayOutput, error) {
	if prob := s.readyOAuth(); prob != nil {
		return nil, prob
	}
	user := v1.User(ctx)
	if user == nil {
		return nil, problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	}
	state, err := s.oauth.PutAuthMeNSFW(accessToken(ctx), in.Body.NsfwDisplay)
	if err != nil {
		return nil, unavailable(err)
	}
	if cookie := sessionCookie(ctx); cookie != "" && s.redis != nil && !user.ViaBearer() {
		if serr := middleware.SetSessionContentStance(ctx, s.redis, cookie, true, state.NSFWDisplay); serr != nil {
			slog.Warn("session content stance writeback failed", "err", serr)
		}
	}
	if s.accounts != nil {
		s.accounts.Invalidate(user.ID)
	}
	return &putNsfwDisplayOutput{Body: NsfwDisplay{
		Object:      "nsfw_display",
		NsfwDisplay: state.NSFWDisplay,
	}}, nil
}
