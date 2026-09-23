package app

import (
	"context"

	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/moemoepoint"
	wallapiv1 "kun-galgame-api/internal/wall/apiv1"
	wallRepo "kun-galgame-api/internal/wall/repository"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

func newWallV1(
	db *gorm.DB,
	community *communityclient.Client,
	users *userclient.Client,
	galgame *client.GalgameClient,
	images func(hashes []string) map[string]imageclient.ImageMeta,
	cdn string,
) *wallapiv1.Service {
	convert := &content.Converter{CDNBase: cdn, SiteBase: apiv1.SiteOrigin, Images: images, Users: users.Users}
	resolve := func(ctx context.Context, workID int) (bool, error) {
		_, found, appErr := galgame.CatalogWorkDetail(ctx, workID)
		if appErr != nil {
			return false, appErr
		}
		return found, nil
	}
	return wallapiv1.New(wallRepo.NewStore(db), community, users, convert, resolve, moemoepoint.Award, cdn)
}
