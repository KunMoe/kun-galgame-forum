package app

import (
	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/content"
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
	convert := &content.Converter{
		CDNBase:  cdn,
		SiteBase: apiv1.SiteOrigin,
		Images:   a.ImageMeta,
		Users:    a.UserClient.Users,
	}
	return docapiv1.New(docRepo.NewDocRepository(a.DB), a.UserClient, convert, a.ImageMeta, cdn)
}
