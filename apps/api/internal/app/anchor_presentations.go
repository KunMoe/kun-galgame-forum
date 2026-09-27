package app

import (
	activityapiv1 "kun-galgame-api/internal/activity/apiv1"
	activitypush "kun-galgame-api/internal/activity/push"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

func startAnchorPresentations(db *gorm.DB, community *communityclient.Client, users *userclient.Client, catalog activityapiv1.Catalog, cdn, redirectURI string) func() {
	return activitypush.NewPresenter(db, community, users, catalog, cdn, activitypush.Origin(redirectURI)).Start()
}
