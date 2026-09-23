package apiv1

import (
	"context"

	"kun-galgame-api/internal/admin/service"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
)

type userPermissionsInput struct {
	UserID string `path:"user_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"User id."`
}

type replaceUserPermissionsInput struct {
	UserID string `path:"user_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"User id."`
	Body   UserPermissionsReplace
}

type userPermissionsOutput struct {
	Body UserPermissions
}

func (s *Service) readyUsers() *problem.Problem {
	if s == nil || s.users == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

func (s *Service) getUserPermissions(ctx context.Context, in *userPermissionsInput) (*userPermissionsOutput, error) {
	u, prob := requireAdmin(ctx)
	if prob != nil {
		return nil, prob
	}
	if prob := s.readyUsers(); prob != nil {
		return nil, prob
	}
	uid, ok := repr.ParseID(repr.DecimalID(in.UserID))
	if !ok {
		return nil, notFound()
	}
	layer, err := s.users.View(ctx, uid)
	if err != nil {
		return nil, mapErr(err)
	}
	return &userPermissionsOutput{Body: toUserPermissions(layer, perm.Rank(u.Roles))}, nil
}

func (s *Service) replaceUserPermissions(ctx context.Context, in *replaceUserPermissionsInput) (*userPermissionsOutput, error) {
	u, prob := requireAdmin(ctx)
	if prob != nil {
		return nil, prob
	}
	if prob := s.readyUsers(); prob != nil {
		return nil, prob
	}
	uid, ok := repr.ParseID(repr.DecimalID(in.UserID))
	if !ok {
		return nil, notFound()
	}
	layer, err := s.users.Replace(ctx, operator(u), uid, fromOverrides(in.Body.Overrides))
	if err != nil {
		return nil, mapErr(err)
	}
	return &userPermissionsOutput{Body: toUserPermissions(layer, perm.Rank(u.Roles))}, nil
}

func toUserPermissions(l service.UserLayer, viewerRank int) UserPermissions {
	roles := make([]RankedRole, len(l.Roles))
	for i, r := range l.Roles {
		roles[i] = RankedRole(r)
	}
	return UserPermissions{
		Object:    "user_permissions",
		ID:        repr.ID(l.UserID),
		Roles:     roles,
		Baseline:  toPermissions(l.Baseline),
		Overrides: toOverrides(l.Overrides),
		Effective: toPermissions(l.Effective),
		IsLocked:  l.Locked,
		Viewer:    UserPermissionsViewer{CanEdit: !l.Locked && viewerRank > perm.Rank(l.Roles)},
	}
}
