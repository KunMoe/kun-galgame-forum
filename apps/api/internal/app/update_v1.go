package app

import (
	updateapiv1 "kun-galgame-api/internal/update/apiv1"
	updateRepo "kun-galgame-api/internal/update/repository"
)

func (a *App) newUpdateV1() *updateapiv1.Service {
	cdn := ""
	if a.Config != nil {
		cdn = a.Config.NextMoeAPI.ImageCDNBase
	}
	return updateapiv1.New(updateRepo.NewStore(a.DB), a.UserClient, a.TrustCheck, a.TrustScan, cdn)
}
