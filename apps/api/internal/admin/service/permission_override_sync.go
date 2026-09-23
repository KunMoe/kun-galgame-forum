package service

import (
	"context"
	"log/slog"
	"time"

	"kun-galgame-api/internal/admin/model"
	"kun-galgame-api/pkg/perm"
)

type roleRowLister interface {
	ListAll(ctx context.Context) ([]model.RolePermissionOverride, error)
}

type userRowLister interface {
	ListAll(ctx context.Context) ([]model.UserPermissionOverride, error)
}

type PermissionOverrideSync struct {
	roleStore roleRowLister
	userStore userRowLister
}

func NewPermissionOverrideSync(roleStore roleRowLister, userStore userRowLister) *PermissionOverrideSync {
	return &PermissionOverrideSync{roleStore: roleStore, userStore: userStore}
}

func (s *PermissionOverrideSync) Load(ctx context.Context) error {
	roleRows, err := s.roleStore.ListAll(ctx)
	if err != nil {
		return err
	}
	userRows, err := s.userStore.ListAll(ctx)
	if err != nil {
		return err
	}
	perm.SetOverrides(roleRowsToOverrideMap(roleRows))
	perm.SetUserOverrides(userRowsToOverrideMap(userRows))
	return nil
}

func (s *PermissionOverrideSync) StartRefresher(interval time.Duration) func() {
	ticker := time.NewTicker(interval)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				ticker.Stop()
				return
			case <-ticker.C:
				if err := s.Load(context.Background()); err != nil {
					slog.Warn("权限覆盖刷新失败, 继续沿用上一次的有效权限", "error", err)
				}
			}
		}
	}()
	return func() { close(done) }
}

func roleRowsToOverrideMap(rows []model.RolePermissionOverride) map[string][]perm.Override {
	out := make(map[string][]perm.Override)
	for _, r := range rows {
		out[r.Role] = append(out[r.Role], perm.Override{
			Permission: perm.Permission(r.Permission),
			Effect:     r.Effect,
		})
	}
	return out
}

func userRowsToOverrideMap(rows []model.UserPermissionOverride) map[int][]perm.Override {
	out := make(map[int][]perm.Override)
	for _, r := range rows {
		out[r.UserID] = append(out[r.UserID], perm.Override{
			Permission: perm.Permission(r.Permission),
			Effect:     r.Effect,
		})
	}
	return out
}
