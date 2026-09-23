package service

import (
	"fmt"

	msgModel "kun-galgame-api/internal/message/model"
	"kun-galgame-api/internal/moemoepoint"

	"gorm.io/gorm"
)

type InteractionHelpers struct{}

func (InteractionHelpers) AdjustMoemoepoint(_ *gorm.DB, userID, delta int, reason, ref string) {
	moemoepoint.Award(userID, delta, reason, ref, moemoepoint.KeyNonce(reason, ref))
}

func (InteractionHelpers) CreateGalgameMessageWithContent(
	tx *gorm.DB,
	senderID, receiverID int,
	msgType, content string,
	workID int,
) error {
	if senderID == receiverID || receiverID <= 0 {
		return nil
	}
	link := fmt.Sprintf("/galgame/%d", workID)

	var count int64
	tx.Model(&msgModel.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND type = ? AND link = ?",
			senderID, receiverID, msgType, link).
		Count(&count)
	if count > 0 {
		return nil
	}

	return tx.Create(&msgModel.Message{
		SenderID: senderID, ReceiverID: receiverID,
		Type: msgType, Content: content, Link: link, Status: "unread",
	}).Error
}

func (InteractionHelpers) CreateGalgameCommentMention(
	tx *gorm.DB,
	senderID, receiverID int,
	content string,
	workID, commentID int,
) error {
	if senderID == receiverID || receiverID <= 0 {
		return nil
	}
	// `thread` makes CommunityContainer treat `comment` as a legacy id and
	// resolve it through /comments/locate, so a community post id never scrolls.
	link := fmt.Sprintf("/galgame/%d?comment=%d", workID, commentID)

	var count int64
	tx.Model(&msgModel.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND type = ? AND link = ?",
			senderID, receiverID, "mentioned", link).
		Count(&count)
	if count > 0 {
		return nil
	}

	return tx.Create(&msgModel.Message{
		SenderID: senderID, ReceiverID: receiverID,
		Type: "mentioned", Content: content, Link: link, Status: "unread",
	}).Error
}
