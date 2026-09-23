package app

import (
	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/moemoepoint"
	quizapiv1 "kun-galgame-api/internal/quiz/apiv1"
	quizRepo "kun-galgame-api/internal/quiz/repository"
)

func (a *App) newQuizV1() *quizapiv1.Service {
	if a.DB == nil || a.UserClient == nil {
		return nil
	}
	cdn := ""
	if a.Config != nil {
		cdn = a.Config.NextMoeAPI.ImageCDNBase
	}
	convert := &content.Converter{
		CDNBase:  cdn,
		SiteBase: apiv1.SiteOrigin,
		Images:   a.ImageMeta,
		Users:    a.UserClient.Users,
	}
	var award quizapiv1.AwardFunc
	if a.TopicAward != nil {
		award = quizapiv1.AwardFunc(a.TopicAward)
	} else {
		award = moemoepoint.Award
	}
	return quizapiv1.New(
		quizRepo.NewStore(a.DB),
		a.UserClient,
		convert,
		a.TrustCheck,
		a.TrustScan,
		award,
		func() quizapiv1.Catalog { return a.QuizCatalog },
		cdn,
	)
}
