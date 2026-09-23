package apiv1

import (
	"context"
	"errors"

	"kun-galgame-api/internal/admin/service"
	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
)

func permissionRequired() *problem.Problem {
	return problem.New(problem.CodePermissionRequired, "The token lacks the permission this decision needs.")
}

func notFound() *problem.Problem {
	return problem.New(problem.CodeNotFound, "Nothing visible exists at this URL.")
}

// A role gate on purpose, not a permission key: overrides must never be able
// to lock admins out of the surface that repairs overrides.
func requireAdmin(ctx context.Context) (*middleware.UserInfo, *problem.Problem) {
	u := v1.User(ctx)
	if u == nil || !u.CanAdminister() {
		return nil, permissionRequired()
	}
	return u, nil
}

func operator(u *middleware.UserInfo) service.Operator {
	return service.Operator{ID: u.ID, Rank: perm.Rank(u.Roles), Holds: u.Can}
}

func mapErr(err error) *problem.Problem {
	var ve *service.ValidationError
	switch {
	case errors.As(err, &ve):
		fields := make([]problem.FieldError, len(ve.Violations))
		for i, v := range ve.Violations {
			if v.Parameter != "" {
				fields[i] = problem.AtParameter(v.Parameter, v.Reason, v.Detail, nil)
			} else {
				fields[i] = problem.AtPointer(v.Pointer, v.Reason, v.Detail, nil)
			}
		}
		return problem.New(problem.CodeValidationFailed, "The request is syntactically valid but semantically not.", fields...)
	case errors.Is(err, service.ErrUserNotFound):
		return notFound()
	case errors.Is(err, service.ErrUserLookup):
		return problem.Unavailable(err)
	}
	return problem.Internal(err)
}

func toPermissions(ps []perm.Permission) []Permission {
	out := make([]Permission, len(ps))
	for i, p := range ps {
		out[i] = Permission(p)
	}
	return out
}

func toOverrides(ovs []perm.Override) []PermissionOverride {
	out := make([]PermissionOverride, len(ovs))
	for i, ov := range ovs {
		out[i] = PermissionOverride{Permission: Permission(ov.Permission), Effect: ov.Effect}
	}
	return out
}

func fromOverrides(ovs []PermissionOverride) []perm.Override {
	out := make([]perm.Override, len(ovs))
	for i, ov := range ovs {
		out[i] = perm.Override{Permission: perm.Permission(ov.Permission), Effect: ov.Effect}
	}
	return out
}

type permissionSetOutput struct {
	Body PermissionSet
}

func (s *Service) getMyPermissions(ctx context.Context, _ *struct{}) (*permissionSetOutput, error) {
	u := v1.User(ctx)
	held := make([]Permission, 0)
	for _, p := range perm.Catalog() {
		if u.Can(p) {
			held = append(held, Permission(p))
		}
	}
	return &permissionSetOutput{Body: PermissionSet{Object: "permission_set", Permissions: held}}, nil
}
