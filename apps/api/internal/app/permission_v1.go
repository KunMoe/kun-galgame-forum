package app

import (
	adminRepo "kun-galgame-api/internal/admin/repository"
	adminService "kun-galgame-api/internal/admin/service"
	permissionapiv1 "kun-galgame-api/internal/permission/apiv1"
)

func (a *App) newPermissionV1() *permissionapiv1.Service {
	if a.DB == nil || a.UserClient == nil {
		return permissionapiv1.New(nil, nil, nil, nil, "")
	}
	roles := adminRepo.NewRolePermissionRepository(a.DB)
	users := adminRepo.NewUserPermissionRepository(a.DB)
	sync := adminService.NewPermissionOverrideSync(roles, users)
	cdn := ""
	if a.Config != nil {
		cdn = a.Config.NextMoeAPI.ImageCDNBase
	}
	return permissionapiv1.New(
		adminService.NewRolePermissionService(roles, sync),
		adminService.NewUserPermissionService(users, a.UserClient, sync),
		adminRepo.NewPermissionAuditRepository(a.DB),
		a.UserClient,
		cdn,
	)
}
