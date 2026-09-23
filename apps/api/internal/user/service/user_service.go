package service

import (
	"time"

	"kun-galgame-api/internal/galgame/client"
	galgameService "kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/userclient"

	"github.com/redis/go-redis/v9"
)

type UserService struct {
	stateRepo     *repository.StateRepository
	userStatsRepo *repository.UserStatsRepository
	rdb           *redis.Client
	galgameClient *client.GalgameClient
	galgameStats  *galgameService.GalgameUserStatsService
	userClient    *userclient.Client
	community     *communityclient.Client
	commentCache  *visiblePostsCache
	nowFn         func() time.Time
}

func NewUserService(
	stateRepo *repository.StateRepository,
	userStatsRepo *repository.UserStatsRepository,
	rdb *redis.Client,
	galgameClient *client.GalgameClient,
	galgameStats *galgameService.GalgameUserStatsService,
	userClient *userclient.Client,
	community *communityclient.Client,
) *UserService {
	return &UserService{
		stateRepo:     stateRepo,
		userStatsRepo: userStatsRepo,
		rdb:           rdb,
		galgameClient: galgameClient,
		galgameStats:  galgameStats,
		userClient:    userClient,
		community:     community,
		commentCache:  newVisiblePostsCache(),
	}
}

func (s *UserService) now() time.Time {
	if s != nil && s.nowFn != nil {
		return s.nowFn()
	}
	return time.Now()
}

func (s *UserService) WithClock(now func() time.Time) {
	if s != nil {
		s.nowFn = now
	}
}

func (s *UserService) ReplaceStatsRepo(r *repository.UserStatsRepository) {
	if s != nil {
		s.userStatsRepo = r
	}
}

func (s *UserService) ReplaceStateRepo(r *repository.StateRepository) {
	if s != nil {
		s.stateRepo = r
	}
}
