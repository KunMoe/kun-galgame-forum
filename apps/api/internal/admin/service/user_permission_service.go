package service

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"kun-galgame-api/internal/admin/model"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/userclient"
)

type userOverrideStore interface {
	ListForUser(ctx context.Context, userID int) ([]model.UserPermissionOverride, error)
	Replace(ctx context.Context, userID, operatorUID int, plan func([]model.UserPermissionOverride) ([]model.UserPermissionOverride, error)) error
}

type userLookup interface {
	User(ctx context.Context, id int) (userclient.User, bool, error)
}

type UserPermissionService struct {
	repo       userOverrideStore
	userClient userLookup
	reload     reloader
}

func NewUserPermissionService(repo userOverrideStore, userClient userLookup, reload reloader) *UserPermissionService {
	return &UserPermissionService{repo: repo, userClient: userClient, reload: reload}
}

func (s *UserPermissionService) View(ctx context.Context, uid int) (UserLayer, error) {
	roles, err := s.targetRoles(ctx, uid)
	if err != nil {
		return UserLayer{}, err
	}
	rows, err := s.repo.ListForUser(ctx, uid)
	if err != nil {
		return UserLayer{}, err
	}
	return buildUserLayer(uid, roles, rows), nil
}

func (s *UserPermissionService) Replace(ctx context.Context, op Operator, uid int, overrides []perm.Override) (UserLayer, error) {
	roles, err := s.targetRoles(ctx, uid)
	if err != nil {
		return UserLayer{}, err
	}
	err = s.repo.Replace(ctx, uid, op.ID, func(current []model.UserPermissionOverride) ([]model.UserPermissionOverride, error) {
		if vs := validateUserReplace(op, roles, current, overrides); len(vs) > 0 {
			return nil, &ValidationError{Violations: vs}
		}
		rows := make([]model.UserPermissionOverride, len(overrides))
		for i, ov := range overrides {
			rows[i] = model.UserPermissionOverride{Permission: string(ov.Permission), Effect: ov.Effect}
		}
		return rows, nil
	})
	if err != nil {
		return UserLayer{}, err
	}
	if err := s.reload.Load(ctx); err != nil {
		slog.Warn("reloading permission overrides after a user write failed; the refresher will converge", "error", err)
	}
	return s.View(ctx, uid)
}

func (s *UserPermissionService) targetRoles(ctx context.Context, uid int) ([]string, error) {
	u, found, err := s.userClient.User(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUserLookup, err)
	}
	if !found {
		return nil, ErrUserNotFound
	}
	roles := make([]string, 0, len(matrixRoles))
	for _, r := range matrixRoles {
		if slices.Contains(u.Roles, r) {
			roles = append(roles, r)
		}
	}
	return roles, nil
}

// Computed from the rows just read rather than perm.CanUser, whose in-process
// table can be 60 seconds stale on another instance.
func buildUserLayer(uid int, roles []string, rows []model.UserPermissionOverride) UserLayer {
	locked := slices.Contains(roles, roleRen)
	effects := make(map[perm.Permission]string, len(rows))
	overrides := make([]perm.Override, 0, len(rows))
	for _, r := range rows {
		effects[perm.Permission(r.Permission)] = r.Effect
		overrides = append(overrides, perm.Override{Permission: perm.Permission(r.Permission), Effect: r.Effect})
	}
	catalog := perm.Catalog()
	baseline := make([]perm.Permission, 0, len(catalog))
	effective := make([]perm.Permission, 0, len(catalog))
	for _, p := range catalog {
		inRole := perm.Can(roles, p)
		if inRole {
			baseline = append(baseline, p)
		}
		switch {
		case locked, effects[p] == perm.EffectGrant, inRole && effects[p] != perm.EffectRevoke:
			effective = append(effective, p)
		}
	}
	return UserLayer{
		UserID:    uid,
		Roles:     roles,
		Baseline:  baseline,
		Overrides: inCatalogOrder(overrides),
		Effective: effective,
		Locked:    locked,
	}
}
