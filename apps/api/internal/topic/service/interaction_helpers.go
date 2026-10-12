package service

import (
	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/infrastructure/markdown"
	msgModel "kun-galgame-api/internal/message/model"
	msgRepo "kun-galgame-api/internal/message/repository"
	msgService "kun-galgame-api/internal/message/service"

	"gorm.io/gorm"
)

type InteractionHelpers struct{}

func (InteractionHelpers) CreateTopicMessageWithContent(
	tx *gorm.DB,
	senderID, receiverID int,
	msgType, content string,
	topicID, replyFloor, commentID int,
	ref msgModel.Ref,
) error {
	if senderID == receiverID || receiverID <= 0 {
		return nil
	}
	return msgRepo.InsertOnce(tx, &msgModel.Message{
		SenderID:   senderID,
		ReceiverID: receiverID,
		Type:       msgType,
		Content:    content,
		Link:       msgService.BuildTopicLink(topicID, replyFloor, commentID),
		Status:     "unread",
		Ref:        ref,
	})
}

func (InteractionHelpers) CreateReplyMessage(
	tx *gorm.DB,
	senderID, receiverID int,
	msgType, content string,
	topicID, replyFloor, commentID int,
	ref msgModel.Ref,
) error {
	if senderID == receiverID || receiverID <= 0 {
		return nil
	}
	link := msgService.BuildTopicLink(topicID, replyFloor, commentID)
	return tx.Create(&msgModel.Message{
		SenderID:   senderID,
		ReceiverID: receiverID,
		Type:       msgType,
		Content:    content,
		Link:       link,
		Status:     "unread",
		Ref:        ref,
	}).Error
}

func recomputeTopicCounts(tx *gorm.DB, topicID int) error {
	return tx.Exec(`
		UPDATE topic SET
			reply_count   = (SELECT COUNT(*) FROM topic_reply  WHERE topic_id = ? AND status = 0),
			comment_count = (SELECT COUNT(*) FROM topic_comment WHERE topic_id = ? AND status = 0)
		WHERE id = ?`, topicID, topicID, topicID).Error
}

func RecomputeTopicCounts(tx *gorm.DB, topicID int) error {
	return recomputeTopicCounts(tx, topicID)
}

func (h InteractionHelpers) NotifyMentionsLimited(tx *gorm.DB, senderID, topicID, replyFloor, commentID int, ref msgModel.Ref, content string, limit int, muted map[int]bool) error {
	preview := truncate(markdown.StripReferenceTokens(content), constants.TextPreviewLength)
	for _, uid := range MentionedIDs(content, limit) {
		if muted[uid] {
			continue
		}
		if err := h.CreateTopicMessageWithContent(tx, senderID, uid, "mentioned", preview, topicID, replyFloor, commentID, ref); err != nil {
			return err
		}
	}
	return nil
}

func MentionedIDs(content string, limit int) []int {
	ids := markdown.ExtractMentionIDs(content)
	if limit > 0 && len(ids) > limit {
		ids = ids[:limit]
	}
	return ids
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}
