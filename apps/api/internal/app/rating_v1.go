package app

import (
	"kun-galgame-api/internal/galgame/client"
	ratingapiv1 "kun-galgame-api/internal/galgame/ratingapiv1"
	galgameRepo "kun-galgame-api/internal/galgame/repository"
	galgameService "kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

func newRatingV1(
	db *gorm.DB,
	galgame *client.GalgameClient,
	users *userclient.Client,
	check *gate.CheckService,
	scan *gate.ScanService,
	playtime *galgameService.PlaytimeService,
	cdn string,
) *ratingapiv1.Service {
	helpers := galgameService.InteractionHelpers{}
	return ratingapiv1.New(ratingapiv1.Deps{
		Store: galgameRepo.NewRatingStore(db),
		Rows:  galgame,
		Works: workrepr.NewHydrator(galgame, db, cdn),
		Users: users,
		Check: check,
		Scan:  scan,
		Award: moemoepoint.Award,
		Notify: func(tx *gorm.DB, senderID, receiverID int, preview string, workID int) error {
			return helpers.CreateGalgameMessageWithContent(tx, senderID, receiverID, "liked", preview, workID)
		},
		Sync: playtime.SyncWorkState,
		CDN:  cdn,
	})
}
