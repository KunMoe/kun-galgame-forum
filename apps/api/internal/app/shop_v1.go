package app

import (
	shopapiv1 "kun-galgame-api/internal/shop/apiv1"
	"kun-galgame-api/internal/user/repository"
)

func (a *App) newShopV1() *shopapiv1.Service {
	if a.UserClient == nil {
		return nil
	}
	state := a.UserState
	if state == nil && a.DB != nil {
		state = repository.NewStateRepository(a.DB)
	}
	var mirror func(userID, balance int) error
	if state != nil {
		mirror = state.SetMoemoepoint
	}
	return shopapiv1.New(a.UserClient, mirror)
}
