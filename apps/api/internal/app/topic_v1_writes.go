package app

import (
	"kun-galgame-api/internal/envconfig"
	envconfigapiv1 "kun-galgame-api/internal/envconfig/apiv1"
	msgRepo "kun-galgame-api/internal/message/repository"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/setting"
	settingapiv1 "kun-galgame-api/internal/setting/apiv1"
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"
	topicRepo "kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/internal/trust/gate"
	userRepo "kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/config"
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
	w := topicapiv1.NewWrites(reads, state, check, scan, a.TopicAward, notify)
	topicAI := a.TopicAI
	if topicAI == nil && a.DB != nil {
		topicAI = topicapiv1.NewTopicAI(nil, a.settingsStore(), topicRepo.NewAIRepository(a.DB))
	}
	w.SetAI(topicAI)
	return w
}

func (a *App) settingsStore() *setting.Store {
	if a.Settings == nil && a.DB != nil {
		a.Settings = setting.NewStore(a.DB)
	}
	return a.Settings
}

func (a *App) newSettingV1() *settingapiv1.Service {
	if a.DB == nil {
		return nil
	}
	return settingapiv1.New(a.settingsStore(), a.Config != nil && a.Config.AI.BaseURL != "" && a.Config.AI.Token != "", a.Platform)
}

func (a *App) newEnvConfigV1() *envconfigapiv1.Service {
	if a.DB == nil {
		return nil
	}
	cdn := ""
	if a.Config != nil {
		cdn = a.Config.NextMoeAPI.ImageCDNBase
	}
	return envconfigapiv1.New(envconfig.New(a.DB, config.StoreKey()), a.UserClient, cdn)
}
