package app

import (
	docapiv1 "kun-galgame-api/internal/doc/apiv1"
	docRepo "kun-galgame-api/internal/doc/repository"
)

func (a *App) newDocV1() *docapiv1.Service {
	if a.DB == nil || a.UserClient == nil {
		return nil
	}
	cdn := ""
	if a.Config != nil {
		cdn = a.Config.NextMoeAPI.ImageCDNBase
	}
	convert := a.contentConverter(cdn)
	return docapiv1.New(docRepo.NewDocRepository(a.DB), a.UserClient, convert, a.ImageMeta, cdn)
}
