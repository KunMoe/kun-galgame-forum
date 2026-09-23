package apiv1

import (
	"context"

	"kun-galgame-api/internal/admin/service"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
)

type matrixOutput struct {
	Body RolePermissionMatrix
}

type patchMatrixInput struct {
	Body RolePermissionMatrixPatch
}

func (s *Service) readyRoles() *problem.Problem {
	if s == nil || s.roles == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

func (s *Service) getRolePermissionMatrix(ctx context.Context, _ *struct{}) (*matrixOutput, error) {
	u, prob := requireAdmin(ctx)
	if prob != nil {
		return nil, prob
	}
	if prob := s.readyRoles(); prob != nil {
		return nil, prob
	}
	m, err := s.roles.Matrix(ctx)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &matrixOutput{Body: toMatrix(m, perm.Rank(u.Roles))}, nil
}

func (s *Service) updateRolePermissionMatrix(ctx context.Context, in *patchMatrixInput) (*matrixOutput, error) {
	u, prob := requireAdmin(ctx)
	if prob != nil {
		return nil, prob
	}
	if prob := s.readyRoles(); prob != nil {
		return nil, prob
	}
	changes := make([]service.RoleChange, len(in.Body.Changes))
	for i, ch := range in.Body.Changes {
		changes[i] = service.RoleChange{Role: string(ch.Role), Overrides: fromOverrides(ch.Overrides)}
	}
	m, err := s.roles.Apply(ctx, operator(u), changes)
	if err != nil {
		return nil, mapErr(err)
	}
	return &matrixOutput{Body: toMatrix(m, perm.Rank(u.Roles))}, nil
}

func toMatrix(m service.Matrix, viewerRank int) RolePermissionMatrix {
	layers := make([]RolePermissions, len(m.Roles))
	for i, l := range m.Roles {
		layers[i] = RolePermissions{
			Object:    "role_permissions",
			Role:      RankedRole(l.Role),
			Baseline:  toPermissions(l.Baseline),
			Overrides: toOverrides(l.Overrides),
			Effective: toPermissions(l.Effective),
			IsLocked:  l.Locked,
			Viewer:    RolePermissionsViewer{CanEdit: !l.Locked && viewerRank > perm.RoleRank(l.Role)},
		}
	}
	return RolePermissionMatrix{Object: "role_permission_matrix", Catalog: toPermissions(m.Catalog), RolePermissions: layers}
}
