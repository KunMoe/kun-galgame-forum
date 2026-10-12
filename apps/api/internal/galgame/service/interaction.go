package service

import (
	"fmt"

	msgModel "kun-galgame-api/internal/message/model"
	msgRepo "kun-galgame-api/internal/message/repository"

	"gorm.io/gorm"
)

type InteractionHelpers struct{}

func (InteractionHelpers) CreateRatingLikedMessage(
	tx *gorm.DB,
	senderID, receiverID int,
	content string,
	workID, ratingID int,
) error {
	if senderID == receiverID || receiverID <= 0 {
		return nil
	}
	return msgRepo.InsertOnce(tx, &msgModel.Message{
		SenderID: senderID, ReceiverID: receiverID,
		Type: "liked", Content: content, Link: fmt.Sprintf("/galgame/%d", workID), Status: "unread",
		Ref: msgModel.RatingRef(ratingID, workID),
	})
}

func (InteractionHelpers) CreateGalgameCommentMention(
	tx *gorm.DB,
	senderID, receiverID int,
	content string,
	workID int,
	postID int64,
) error {
	if senderID == receiverID || receiverID <= 0 {
		return nil
	}
	// `thread` makes CommunityContainer treat `comment` as a legacy id and
	// resolve it through /comments/locate, so a community post id never scrolls.
	return msgRepo.InsertOnce(tx, &msgModel.Message{
		SenderID: senderID, ReceiverID: receiverID,
		Type: "mentioned", Content: content, Link: fmt.Sprintf("/galgame/%d?comment=%d", workID, postID), Status: "unread",
		Ref: msgModel.CommunityPostRef(postID, msgModel.WorkRef(workID)),
	})
}
