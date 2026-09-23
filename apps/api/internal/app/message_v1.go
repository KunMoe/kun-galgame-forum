package app

import (
	"kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/content"
	messageapiv1 "kun-galgame-api/internal/message/apiv1"
	msgRepo "kun-galgame-api/internal/message/repository"
)

func (a *App) newMessageV1() *messageapiv1.Service {
	var messages *msgRepo.MessageRepository
	var chats *msgRepo.ChatRepository
	if a.DB != nil {
		messages = msgRepo.NewMessageRepository(a.DB)
		chats = msgRepo.NewChatRepository(a.DB)
	}
	cdn := ""
	if a.Config != nil {
		cdn = a.Config.NextMoeAPI.ImageCDNBase
	}
	var convert *content.Converter
	if a.UserClient != nil {
		convert = &content.Converter{
			CDNBase:  cdn,
			SiteBase: apiv1.SiteOrigin,
			Images:   a.ImageMeta,
			Users:    a.UserClient.Users,
		}
	}
	return messageapiv1.New(messages, chats, a.UserClient, convert, cdn, a.Messages)
}
