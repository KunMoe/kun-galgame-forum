package service

import (
	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/infrastructure/markdown"
	msgModel "kun-galgame-api/internal/message/model"
	msgService "kun-galgame-api/internal/message/service"

	"gorm.io/gorm"
)

type InteractionHelpers struct{}

func (InteractionHelpers) CreateTopicMessageWithContent(
	tx *gorm.DB,
	senderID, receiverID int,
	msgType, content string,
	topicID, replyFloor, commentID int,
) error {
	if senderID == receiverID || receiverID <= 0 {
		return nil
	}
	link := msgService.BuildTopicLink(topicID, replyFloor, commentID)
	return createDedupMessage(tx, senderID, receiverID, msgType, content, link)
}

func (InteractionHelpers) CreateReplyMessage(
	tx *gorm.DB,
	senderID, receiverID int,
	msgType, content string,
	topicID, replyFloor, commentID int,
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
	}).Error
}

func createDedupMessage(tx *gorm.DB, senderID, receiverID int, msgType, content, link string) error {
	if senderID == receiverID || receiverID <= 0 {
		return nil
	}
	var count int64
	tx.Model(&msgModel.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND type = ? AND link = ?",
			senderID, receiverID, msgType, link).
		Count(&count)
	if count > 0 {
		return nil
	}
	return tx.Create(&msgModel.Message{
		SenderID:   senderID,
		ReceiverID: receiverID,
		Type:       msgType,
		Content:    content,
		Link:       link,
		Status:     "unread",
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

func (h InteractionHelpers) NotifyMentionsLimited(tx *gorm.DB, senderID, topicID, replyFloor, commentID int, content string, limit int, muted map[int]bool) error {
	preview := truncate(markdown.StripReferenceTokens(content), constants.TextPreviewLength)
	for _, uid := range MentionedIDs(content, limit) {
		if muted[uid] {
			continue
		}
		if err := h.CreateTopicMessageWithContent(tx, senderID, uid, "mentioned", preview, topicID, replyFloor, commentID); err != nil {
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
