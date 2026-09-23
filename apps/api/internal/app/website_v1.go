package app

import (
	websiteapiv1 "kun-galgame-api/internal/website/apiv1"
	websiteRepo "kun-galgame-api/internal/website/repository"
)

func (a *App) newWebsiteV1() *websiteapiv1.Service {
	cdn := ""
	if a.Config != nil {
		cdn = a.Config.NextMoeAPI.ImageCDNBase
	}
	return websiteapiv1.New(websiteRepo.NewStore(a.DB), a.UserClient, a.ImageMeta, cdn)
}
