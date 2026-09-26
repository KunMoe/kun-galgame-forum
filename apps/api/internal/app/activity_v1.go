package app

import (
	activityapiv1 "kun-galgame-api/internal/activity/apiv1"
	activityRepo "kun-galgame-api/internal/activity/repository"
	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

func newActivityV1(db *gorm.DB, catalog activityapiv1.Catalog, users *userclient.Client, community *communityclient.Client, images func([]string) map[string]imageclient.ImageMeta, cdn string) *activityapiv1.Service {
	convert := &content.Converter{CDNBase: cdn, SiteBase: apiv1.SiteOrigin, Images: images, Users: users.Users}
	return activityapiv1.New(activityRepo.NewActivityRepository(db), catalog, users, community, convert, cdn)
}
