package apiv1

import (
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/constants"
	msgModel "kun-galgame-api/internal/message/model"
	msgRepo "kun-galgame-api/internal/message/repository"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
)

func truncateRunes(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}

func topicPreview(title string) string {
	return truncateRunes(title, constants.TextPreviewLength)
}

func replyContentPreview(content string) string {
	return truncateRunes(content, constants.TextPreviewLength)
}

func replyPlainPreview(reply model.TopicReply) string {
	return content.PlainText(reply.Content, 500)
}

func dedupMessage(tx *gorm.DB, senderID, receiverID int, msgType, content, link string, ref msgModel.Ref) error {
	if senderID == receiverID || receiverID <= 0 {
		return nil
	}
	return msgRepo.InsertOnce(tx, &msgModel.Message{
		SenderID:   senderID,
		ReceiverID: receiverID,
		Type:       msgType,
		Content:    content,
		Link:       link,
		Status:     "unread",
		Ref:        ref,
	})
}

func notifyTopicLink(tx *gorm.DB, senderID, receiverID int, msgType, content string, topicID, replyFloor int, ref msgModel.Ref) error {
	return dedupMessage(tx, senderID, receiverID, msgType, content, msgService.BuildTopicLink(topicID, replyFloor, 0), ref)
}

func (x *Interactions) emitSolution(tx *gorm.DB, senderID, receiverID int, content string, reply *model.TopicReply) error {
	if x.notify == nil {
		return nil
	}
	return x.notify.Emit(tx, msgService.Spec{
		SenderID:   senderID,
		ReceiverID: receiverID,
		Kind:       msgService.NotifySolution,
		Content:    content,
		TopicID:    reply.TopicID,
		ReplyFloor: reply.Floor,
		Ref:        msgModel.ReplyRef(reply.ID, reply.TopicID),
	})
}
