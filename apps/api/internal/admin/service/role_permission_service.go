package service

import (
	"context"
	"log/slog"

	"kun-galgame-api/internal/admin/model"
	"kun-galgame-api/pkg/perm"
)

type overrideStore interface {
	ListAll(ctx context.Context) ([]model.RolePermissionOverride, error)
	Replace(ctx context.Context, operatorUID int, plan func([]model.RolePermissionOverride) ([]model.RoleReplacement, error)) error
}

type reloader interface {
	Load(ctx context.Context) error
}

type RolePermissionService struct {
	repo   overrideStore
	reload reloader
}

func NewRolePermissionService(repo overrideStore, reload reloader) *RolePermissionService {
	return &RolePermissionService{repo: repo, reload: reload}
}

func (s *RolePermissionService) Matrix(ctx context.Context) (Matrix, error) {
	rows, err := s.repo.ListAll(ctx)
	if err != nil {
		return Matrix{}, err
	}
	return buildMatrix(rows), nil
}

// Apply replaces the overrides of every role in changes at once. Moderator ⊆
// admin spans two roles, so each change is judged against the others' new
// state, never against what is stored for them.
func (s *RolePermissionService) Apply(ctx context.Context, op Operator, changes []RoleChange) (Matrix, error) {
	err := s.repo.Replace(ctx, op.ID, func(current []model.RolePermissionOverride) ([]model.RoleReplacement, error) {
		if vs := validateRoleChanges(op, current, changes); len(vs) > 0 {
			return nil, &ValidationError{Violations: vs}
		}
		out := make([]model.RoleReplacement, len(changes))
		for i, ch := range changes {
			rows := make([]model.RolePermissionOverride, len(ch.Overrides))
			for j, ov := range ch.Overrides {
				rows[j] = model.RolePermissionOverride{Role: ch.Role, Permission: string(ov.Permission), Effect: ov.Effect}
			}
			out[i] = model.RoleReplacement{Role: ch.Role, Rows: rows}
		}
		return out, nil
	})
	if err != nil {
		return Matrix{}, err
	}
	if err := s.reload.Load(ctx); err != nil {
		slog.Warn("reloading permission overrides after a role write failed; the refresher will converge", "error", err)
	}
	return s.Matrix(ctx)
}

func buildMatrix(rows []model.RolePermissionOverride) Matrix {
	byRole := make(map[string][]perm.Override)
	for _, r := range rows {
		byRole[r.Role] = append(byRole[r.Role], perm.Override{Permission: perm.Permission(r.Permission), Effect: r.Effect})
	}
	layers := make([]RoleLayer, 0, len(matrixRoles))
	for _, role := range matrixRoles {
		overrides := byRole[role]
		if role == roleRen {
			overrides = nil
		}
		layers = append(layers, RoleLayer{
			Role:      role,
			Baseline:  perm.Baseline(role),
			Overrides: inCatalogOrder(overrides),
			Effective: perm.EffectiveSet(role, overrides),
			Locked:    role == roleRen,
		})
	}
	return Matrix{Catalog: perm.Catalog(), Roles: layers}
}

func inCatalogOrder(overrides []perm.Override) []perm.Override {
	byPerm := make(map[perm.Permission]perm.Override, len(overrides))
	for _, ov := range overrides {
		byPerm[ov.Permission] = ov
	}
	out := make([]perm.Override, 0, len(overrides))
	for _, p := range perm.Catalog() {
		if ov, ok := byPerm[p]; ok {
			out = append(out, ov)
		}
	}
	return out
}
