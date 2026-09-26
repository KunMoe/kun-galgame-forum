package app

import (
	"log/slog"
	"time"

	adminRepo "kun-galgame-api/internal/admin/repository"
	galgameRepo "kun-galgame-api/internal/galgame/repository"
	galgameService "kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/galgame/workrepr"
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
	var paths *repository.MoemoepointPathRepository
	if a.DB != nil {
		content = repository.NewUserContentRepository(a.DB)
		paths = repository.NewMoemoepointPathRepository(a.DB)
	}
	var works *workrepr.Hydrator
	if a.DB != nil && a.ResourceCatalog != nil {
		works = workrepr.NewHydrator(a.ResourceCatalog, a.DB, cdn)
	}
	return userapiv1.New(userapiv1.Deps{
		Users:       users,
		Creators:    creators,
		OAuth:       oauthClient,
		Accounts:    a.UserClient,
		Content:     content,
		Paths:       paths,
		Works:       works,
		Resources:   a.newGalgameResourceV1(),
		Community:   a.Community,
		Wall:        a.WallV1,
		Contributed: a.ContributedWorkIDs,
		Redis:       a.Redis,
		State:       state,
		Purge:       a.AdminPurge,
		CDN:         cdn,
	})
}

func expirePurgeArchive(repo *adminRepo.PurgeRepository) func() {
	return func() {
		n, err := repo.ExpireArchive(time.Now().Add(-adminRepo.ArchiveRetention), 5000)
		if err != nil {
			slog.Error("清空存档过期清理失败", "deleted", n, "error", err)
			return
		}
		slog.Info("清空存档过期清理完成", "deleted", n)
	}
}
