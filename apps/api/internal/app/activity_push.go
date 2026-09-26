package app

import (
	activityapiv1 "kun-galgame-api/internal/activity/apiv1"
	activitypush "kun-galgame-api/internal/activity/push"
	"kun-galgame-api/pkg/communityclient"

	"gorm.io/gorm"
)

func startActivityPush(db *gorm.DB, community *communityclient.Client, activities *activityapiv1.Service, redirectURI string) func() {
	return activitypush.New(db, community, activities, activitypush.Origin(redirectURI)).Start()
}
