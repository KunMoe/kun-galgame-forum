package app

import (
	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/moemoepoint"
	toolsetapiv1 "kun-galgame-api/internal/toolset/apiv1"
	toolsetRepo "kun-galgame-api/internal/toolset/repository"
)

func (a *App) newToolsetV1() *toolsetapiv1.Service {
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
	var award toolsetapiv1.AwardFunc
	if a.TopicAward != nil {
		award = toolsetapiv1.AwardFunc(a.TopicAward)
	} else {
		award = moemoepoint.Award
	}
	return toolsetapiv1.New(
		toolsetRepo.NewStore(a.DB),
		a.UserClient,
		convert,
		a.TrustCheck,
		a.TrustScan,
		a.Artifact,
		a.FileStorage,
		award,
		cdn,
	)
}
