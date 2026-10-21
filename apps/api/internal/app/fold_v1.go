package app

import (
	"kun-galgame-api/internal/galgame/foldapiv1"
	galgameRepo "kun-galgame-api/internal/galgame/repository"
)

func (a *App) newWorkMergeFoldV1() *foldapiv1.Service {
	if a.DB == nil {
		return foldapiv1.New(nil)
	}
	return foldapiv1.New(galgameRepo.NewGalgameMergeRepository(a.DB))
}
