package app

import (
	"log/slog"
	"time"

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
		convert = a.contentConverter(cdn)
	}
	return messageapiv1.New(messages, chats, a.UserClient, convert, cdn, a.Messages, a.Config != nil && a.Config.Chat.Enabled)
}

func expireReadNotifications(repo *msgRepo.MessageRepository, days int) func() {
	if days <= 0 {
		return nil
	}
	return func() {
		n, err := repo.DeleteReadBefore(time.Now().AddDate(0, 0, -days), 5000)
		if err != nil {
			slog.Error("已读通知过期清理失败", "deleted", n, "error", err)
			return
		}
		slog.Info("已读通知过期清理完成", "deleted", n, "retention_days", days)
	}
}
