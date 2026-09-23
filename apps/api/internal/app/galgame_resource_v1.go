package app

import (
	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/content"
	galgameRepo "kun-galgame-api/internal/galgame/repository"
	resourceapiv1 "kun-galgame-api/internal/galgame/resourceapiv1"
	"kun-galgame-api/internal/moemoepoint"
)

func (a *App) newGalgameResourceV1() *resourceapiv1.Service {
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
	var award resourceapiv1.AwardFunc
	if a.TopicAward != nil {
		award = resourceapiv1.AwardFunc(a.TopicAward)
	} else {
		award = moemoepoint.Award
	}
	return resourceapiv1.New(
		galgameRepo.NewResourceV1Store(a.DB),
		a.UserClient,
		convert,
		a.TrustCheck,
		a.TrustScan,
		award,
		func() resourceapiv1.Catalog { return a.ResourceCatalog },
		func() resourceapiv1.ClaimFunc { return a.ResourceClaim },
		func() resourceapiv1.ShareChecker { return a.ResourceChecker },
		a.StoreLinks,
		cdn,
	)
}
