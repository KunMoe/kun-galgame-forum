package app

import (
	galgameRepo "kun-galgame-api/internal/galgame/repository"
	galgameService "kun-galgame-api/internal/galgame/service"
	userapiv1 "kun-galgame-api/internal/user/apiv1"
	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/internal/user/repository"
	"kun-galgame-api/internal/user/service"
)

func (a *App) newUserV1() *userapiv1.Users {
	state := a.UserState
	users := a.UserService
	if users == nil && a.DB != nil {
		if state == nil {
			state = repository.NewStateRepository(a.DB)
		}
		galgameStats := galgameService.NewGalgameUserStatsService(nil, nil, galgameRepo.NewGalgameRepository(a.DB))
		users = service.NewUserService(
			state, repository.NewUserStatsRepository(a.DB), a.Redis, nil, galgameStats, a.UserClient, nil,
		)
		a.UserService = users
		if a.UserState == nil {
			a.UserState = state
		}
	}
	creators := a.CreatorService
	if creators == nil && a.DB != nil && a.UserClient != nil {
		creators = galgameService.NewCreatorService(
			galgameRepo.NewRatingStore(a.DB),
			galgameService.NewGalgameUserStatsService(nil, nil, galgameRepo.NewGalgameRepository(a.DB)),
			a.UserClient,
		)
		a.CreatorService = creators
	}
	oauthClient := a.OAuthClient
	if oauthClient == nil && a.Config != nil && a.Config.OAuth.ServerURL != "" {
		oauthClient = oauth.NewClient(a.Config.OAuth)
		a.OAuthClient = oauthClient
	}
	cdn := ""
	if a.Config != nil {
		cdn = a.Config.NextMoeAPI.ImageCDNBase
	}
	if state == nil {
		state = a.UserState
	}
	var content *repository.UserContentRepository
	if a.DB != nil {
		content = repository.NewUserContentRepository(a.DB)
	}
	return userapiv1.New(userapiv1.Deps{
		Users:    users,
		Creators: creators,
		OAuth:    oauthClient,
		Accounts: a.UserClient,
		Content:  content,
		Redis:    a.Redis,
		State:    state,
		CDN:      cdn,
	})
}
