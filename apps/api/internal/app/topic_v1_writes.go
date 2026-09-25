package app

import (
	msgRepo "kun-galgame-api/internal/message/repository"
	msgService "kun-galgame-api/internal/message/service"
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"
	"kun-galgame-api/internal/trust/gate"
	userRepo "kun-galgame-api/internal/user/repository"
)

func (a *App) newTopicV1Writes(reads *topicapiv1.Service) *topicapiv1.Writes {
	check := a.TrustCheck
	scan := a.TrustScan
	if check == nil {
		check = gate.NewCheckService(nil)
	}
	if scan == nil {
		scan = gate.NewScanService(nil)
	}
	state := a.UserState
	if state == nil && a.DB != nil {
		state = userRepo.NewStateRepository(a.DB)
	}
	notify := a.Notifier
	if notify == nil && a.DB != nil {
		notify = msgService.NewNotifier(msgRepo.NewMessageRepository(a.DB))
	}
	return topicapiv1.NewWrites(reads, state, check, scan, a.TopicAward, notify, a.Community)
}
