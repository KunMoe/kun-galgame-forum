package app

import authapiv1 "kun-galgame-api/internal/auth/apiv1"

func (a *App) newAuthV1() *authapiv1.Service {
	cdn := ""
	if a.Config != nil {
		cdn = a.Config.NextMoeAPI.ImageCDNBase
	}
	if a.UserClient == nil {
		return authapiv1.New(nil, cdn)
	}
	return authapiv1.New(a.UserClient, cdn)
}
