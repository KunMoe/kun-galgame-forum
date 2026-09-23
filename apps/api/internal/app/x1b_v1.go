package app

import (
	sectionapiv1 "kun-galgame-api/internal/section/apiv1"
	sectionRepo "kun-galgame-api/internal/section/repository"
)

func (a *App) newSectionV1() *sectionapiv1.Service {
	if a.DB == nil || a.UserClient == nil {
		return nil
	}
	return sectionapiv1.New(sectionRepo.NewSectionRepository(a.DB), a.UserClient)
}
