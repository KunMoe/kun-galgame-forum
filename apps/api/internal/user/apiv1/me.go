package apiv1

import (
	"context"
	"errors"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/user/service"
	"kun-galgame-api/pkg/problem"
)

type getMeOutput struct {
	Body Me
}

func (s *Users) getMe(ctx context.Context, _ *struct{}) (*getMeOutput, error) {
	if prob := s.readyUsers(); prob != nil {
		return nil, prob
	}
	user := v1.User(ctx)
	if user == nil {
		return nil, problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	}
	st, err := s.users.Me(ctx, user.ID)
	if err != nil {
		if errors.Is(err, service.ErrUpstream) {
			return nil, unavailable(err)
		}
		return nil, problem.Internal(err)
	}
	return &getMeOutput{Body: Me{
		Object:                  "me",
		ID:                      repr.ID(user.ID),
		Moemoepoint:             st.Moemoepoint,
		HasCheckedInToday:       st.HasCheckedInToday,
		HasUnreadMessages:       st.HasUnreadMessages,
		IsCreator:               st.IsCreator,
		ToolsetUploadTodayBytes: st.ToolsetUploadTodayBytes,
	}}, nil
}
