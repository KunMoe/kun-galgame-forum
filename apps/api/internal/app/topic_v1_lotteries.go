package app

import (
	"kun-galgame-api/internal/moemoepoint"
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"
	topicService "kun-galgame-api/internal/topic/service"
)

func (a *App) newTopicV1Lotteries(reads *topicapiv1.Service) *topicapiv1.Lotteries {
	award := a.TopicAward
	if award == nil {
		award = moemoepoint.Award
	}
	if a.LotteryService != nil {
		a.LotteryService.SetAward(topicService.AwardFunc(award))
	}
	return topicapiv1.NewLotteries(reads, a.LotteryService, a.TrustCheck, a.TrustScan, award, a.ImageMeta)
}
