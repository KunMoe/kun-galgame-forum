package app

import (
	appreleaseapiv1 "kun-galgame-api/internal/apprelease/apiv1"
	friendlinkapiv1 "kun-galgame-api/internal/friendlink/apiv1"
	friendRepo "kun-galgame-api/internal/friendlink/repository"
)

func (a *App) newFriendLinkV1() *friendlinkapiv1.Service {
	if a.DB == nil {
		return nil
	}
	cdn := ""
	if a.Config != nil {
		cdn = a.Config.NextMoeAPI.ImageCDNBase
	}
	return friendlinkapiv1.New(friendRepo.NewFriendLinkRepository(a.DB), a.ImageMeta, cdn)
}

func (a *App) newAppReleaseV1() *appreleaseapiv1.Service {
	if a.Config == nil {
		return appreleaseapiv1.New(nil)
	}
	return appreleaseapiv1.New(&a.Config.AppRelease)
}
