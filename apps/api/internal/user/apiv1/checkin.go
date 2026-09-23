package apiv1

import (
	"context"
	"errors"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/user/service"
	"kun-galgame-api/pkg/problem"
)

type createCheckInOutput struct {
	Body CheckIn
}

func (s *Users) createCheckIn(ctx context.Context, _ *struct{}) (*createCheckInOutput, error) {
	if prob := s.readyUsers(); prob != nil {
		return nil, prob
	}
	user := v1.User(ctx)
	if user == nil {
		return nil, problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	}
	result, err := s.users.CheckIn(ctx, user.ID)
	if err != nil {
		if errors.Is(err, service.ErrAlreadyCheckedIn) {
			return nil, problem.New(problem.CodeAlreadyExists,
				"The same subject already has a live record for this target.")
		}
		if errors.Is(err, service.ErrUpstream) {
			return nil, unavailable(err)
		}
		return nil, problem.Internal(err)
	}
	return &createCheckInOutput{Body: CheckIn{
		Object:             "check_in",
		CheckInDate:        repr.CalendarDate(result.Date),
		MoemoepointAwarded: result.Awarded,
		Moemoepoint:        result.Balance,
	}}, nil
}
