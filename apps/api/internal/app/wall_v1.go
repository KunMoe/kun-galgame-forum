package app

import (
	"context"

	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	"kun-galgame-api/internal/galgame/client"
	msgRepo "kun-galgame-api/internal/message/repository"
	"kun-galgame-api/internal/moemoepoint"
	wallapiv1 "kun-galgame-api/internal/wall/apiv1"
	wallRepo "kun-galgame-api/internal/wall/repository"
	websiteapiv1 "kun-galgame-api/internal/website/apiv1"
	websiteRepo "kun-galgame-api/internal/website/repository"
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
	exist := func(ctx context.Context, workIDs []int) (map[int]bool, error) {
		rows, appErr := galgame.CatalogRowsByWorkIDs(ctx, workIDs, "", "all")
		if appErr != nil {
			return nil, appErr
		}
		out := make(map[int]bool, len(rows))
		for id := range rows {
			out[id] = true
		}
		return out, nil
	}
	works := func(ctx context.Context, workIDs []int) (map[int]repr.WorkRef, error) {
		rows, appErr := galgame.CatalogRowsByWorkIDs(ctx, workIDs, "names,covers", "all")
		if appErr != nil {
			return nil, appErr
		}
		out := make(map[int]repr.WorkRef, len(rows))
		for id := range rows {
			row := rows[id]
			out[id] = galgameapiv1.WorkRefOf(ctx, &row, cdn)
		}
		return out, nil
	}
	return wallapiv1.New(wallRepo.NewStore(db), community, users, convert, resolve, exist, moemoepoint.Award, cdn).
		WithFollowing(msgRepo.NewMessageRepository(db).MarkCommunityThreadRead, works,
			websiteapiv1.New(websiteRepo.NewStore(db), users, images, cdn).SummariesByIDs)
}
