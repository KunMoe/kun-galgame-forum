package app

import (
	"kun-galgame-api/internal/moemoepoint"
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"
	topicRepo "kun-galgame-api/internal/topic/repository"
)

func (a *App) newTopicV1Admin(reads *topicapiv1.Service) *topicapiv1.AdminTopics {
	award := a.TopicAward
	if award == nil {
		award = moemoepoint.Award
	}
	var lots *topicRepo.LotteryRepository
	if a.LotteryService != nil {
		lots = a.LotteryService.Repo()
	}
	return topicapiv1.NewAdminTopics(reads, lots, award)
}
