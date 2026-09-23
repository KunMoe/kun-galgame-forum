package apiv1

import (
	"context"
	"strconv"
	"strings"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

type listUsersInput struct {
	Q     string           `query:"q" required:"false" maxLength:"50" doc:"Name query, at most 50 characters (the account service's own limit). After trimming whitespace it must not be empty. Exactly one of q or ids is required. Free text; never use it as a decision input."`
	IDs   []repr.DecimalID `query:"ids" required:"false" maxItems:"100" doc:"User ids to resolve, comma-separated. 1 to 100 of them. Exactly one of q or ids is required."`
	Limit int              `query:"limit" minimum:"1" maximum:"20" default:"8" doc:"Page size for q. 1–20, default 8. Values above 20 are rejected, not clamped. Ignored when ids is set."`
}

type listUsersOutput struct {
	Body repr.BatchList[repr.UserRef]
}

func (s *Users) listUsers(ctx context.Context, in *listUsersInput) (*listUsersOutput, error) {
	if prob := s.readyAccounts(); prob != nil {
		return nil, prob
	}
	if v1.User(ctx) == nil {
		return nil, problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	}
	q := strings.TrimSpace(in.Q)
	hasQ := q != ""
	hasIDs := len(in.IDs) > 0
	if hasQ && hasIDs {
		return nil, validationFailed(problem.AtParameter("ids", problem.ReasonInconsistentWith,
			"q", nil))
	}
	if hasIDs {
		return s.listUsersByIDs(ctx, in.IDs)
	}
	if q == "" {
		return nil, validationFailed(problem.AtParameter("q", problem.ReasonRequired,
			"q must contain at least 1 character after trimming whitespace", nil))
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 8
	}
	found, err := s.accounts.SearchUsers(ctx, q, limit)
	if userclient.IsInvalidParam(err) {
		return nil, problem.New(problem.CodeInvalidParameter, "The account service refused the name query.",
			problem.AtParameter("q", problem.ReasonNotAllowedValue, "the account service refused this name query", nil))
	}
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
	return &listUsersOutput{Body: repr.NewBatchList(items, nil)}, nil
}

func (s *Users) listUsersByIDs(ctx context.Context, parts []repr.DecimalID) (*listUsersOutput, error) {
	requested, prob := parseUserIDs(parts)
	if prob != nil {
		return nil, prob
	}
	unique := make([]int, 0, len(requested))
	seen := make(map[int]bool, len(requested))
	for _, id := range requested {
		if seen[id] {
			continue
		}
		seen[id] = true
		unique = append(unique, id)
	}
	found, err := s.accounts.Users(ctx, unique)
	if err != nil {
		return nil, unavailable(err)
	}
	items := make([]repr.UserRef, 0, len(unique))
	missing := []repr.DecimalID{}
	emitted := map[int]bool{}
	for _, id := range requested {
		if emitted[id] {
			continue
		}
		emitted[id] = true
		u, ok := found[id]
		if !ok || !userclient.IsRenderable(u) {
			missing = append(missing, repr.ID(id))
			continue
		}
		items = append(items, repr.NewUserRef(s.cdn, u))
	}
	return &listUsersOutput{Body: repr.NewBatchList(items, missing)}, nil
}

func parseUserIDs(parts []repr.DecimalID) ([]int, *problem.Problem) {
	out := make([]int, 0, len(parts))
	for i, part := range parts {
		id, ok := repr.ParseID(repr.DecimalID(strings.TrimSpace(string(part))))
		if !ok {
			return nil, validationFailed(problem.AtParameter("ids", problem.ReasonInvalidFormat,
				"every id must be a positive decimal integer; item "+strconv.Itoa(i)+" is not", nil))
		}
		out = append(out, id)
	}
	return out, nil
}
