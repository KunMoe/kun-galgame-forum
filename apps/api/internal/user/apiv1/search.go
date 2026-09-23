package apiv1

import (
	"context"
	"strings"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"
)

type listUsersInput struct {
	Q     string `query:"q" required:"true" minLength:"1" maxLength:"64" doc:"Name query. After trimming whitespace it must not be empty. Free text; never use it as a decision input."`
	Limit int    `query:"limit" minimum:"1" maximum:"20" default:"8" doc:"Page size. 1–20, default 8. Values above 20 are rejected, not clamped."`
}

type listUsersOutput struct {
	Body repr.List[repr.UserRef]
}

func (s *Users) listUsers(ctx context.Context, in *listUsersInput) (*listUsersOutput, error) {
	if prob := s.readyAccounts(); prob != nil {
		return nil, prob
	}
	if v1.User(ctx) == nil {
		return nil, problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	}
	q := strings.TrimSpace(in.Q)
	if q == "" {
		return nil, validationFailed(problem.AtParameter("q", problem.ReasonRequired,
			"q must contain at least 1 character after trimming whitespace", nil))
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 8
	}
	found, err := s.accounts.SearchUsers(ctx, q, limit)
	if err != nil {
		return nil, unavailable(err)
	}
	items := make([]repr.UserRef, 0, len(found))
	for _, u := range found {
		if u.Status != 0 {
			continue
		}
		items = append(items, repr.NewUserRef(s.cdn, u))
	}
	return &listUsersOutput{Body: repr.NewList(items, nil)}, nil
}
