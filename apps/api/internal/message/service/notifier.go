package service

import (
	"fmt"
	"strings"

	msgModel "kun-galgame-api/internal/message/model"
	msgRepo "kun-galgame-api/internal/message/repository"

	"gorm.io/gorm"
)

type NotifyKind string

const (
	NotifyUpvoted   NotifyKind = "upvoted"
	NotifyLiked     NotifyKind = "liked"
	NotifyFavorite  NotifyKind = "favorite"
	NotifyReplied   NotifyKind = "replied"
	NotifyCommented NotifyKind = "commented"
	NotifyFollowed  NotifyKind = "followed"
	NotifySolution  NotifyKind = "solution"
	NotifyPinReply  NotifyKind = "pin-reply"
	NotifyMentioned NotifyKind = "mentioned"
	NotifyAdmin     NotifyKind = "admin"
	NotifyExpired   NotifyKind = "expired"
	NotifyRequested NotifyKind = "requested"
	NotifyMerged    NotifyKind = "merged"
	NotifyDeclined  NotifyKind = "declined"

	NotifyLotteryWon     NotifyKind = "lottery-won"
	NotifyLotteryClosed  NotifyKind = "lottery-closed"
	NotifyLotteryExpired NotifyKind = "lottery-expired"
	NotifyPollClosed     NotifyKind = "poll-closed"

	NotifyUserFollowed     NotifyKind = "user-followed"
	NotifyFolloweeTopic    NotifyKind = "followee-topic"
	NotifyFolloweeActivity NotifyKind = "followee-activity"
)

const notifyContentLimit = 233

type Spec struct {
	SenderID   int
	ReceiverID int
	Kind       NotifyKind
	Content    string

	TopicID    int
	ReplyFloor int
	CommentID  int
	WorkID     int
	ToolsetID  int
	WebsiteURL string
}

type Notifier interface {
	Emit(tx *gorm.DB, spec Spec) error
	EmitMany(tx *gorm.DB, specs []Spec) error
}

type notifier struct {
	repo *msgRepo.MessageRepository
}

func NewNotifier(repo *msgRepo.MessageRepository) Notifier {
	return &notifier{repo: repo}
}

func (n *notifier) Emit(tx *gorm.DB, spec Spec) error {
	if spec.ReceiverID == 0 || spec.SenderID == spec.ReceiverID {
		return nil
	}
	link := buildNotifyLink(spec)
	if link == "" {
		return nil
	}
	content := truncateNotifyContent(spec.Content)

	db := tx
	if db == nil {
		db = n.repo.DB()
	}

	var existing int64
	if err := db.Model(&msgModel.Message{}).
		Where(`sender_id = ? AND receiver_id = ? AND type = ? AND content = ? AND link = ?`,
			spec.SenderID, spec.ReceiverID, string(spec.Kind), content, link,
		).Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}

	return db.Create(&msgModel.Message{
		SenderID:   spec.SenderID,
		ReceiverID: spec.ReceiverID,
		Type:       string(spec.Kind),
		Content:    content,
		Link:       link,
	}).Error
}

func (n *notifier) EmitMany(tx *gorm.DB, specs []Spec) error {
	for _, s := range specs {
		if err := n.Emit(tx, s); err != nil {
			return err
		}
	}
	return nil
}

func BuildTopicLink(topicID, replyFloor, commentID int) string {
	switch {
	case commentID > 0:
		return fmt.Sprintf("/topic/%d?comment=%d", topicID, commentID)
	case replyFloor > 0:
		return fmt.Sprintf("/topic/%d?reply=%d", topicID, replyFloor)
	default:
		return fmt.Sprintf("/topic/%d", topicID)
	}
}

func buildNotifyLink(spec Spec) string {
	switch {
	case spec.TopicID > 0:
		return BuildTopicLink(spec.TopicID, spec.ReplyFloor, spec.CommentID)
	case spec.WorkID > 0:
		return fmt.Sprintf("/galgame/%d", spec.WorkID)
	case spec.ToolsetID > 0:
		return fmt.Sprintf("/toolset/%d", spec.ToolsetID)
	case spec.WebsiteURL != "":
		return "/website/" + spec.WebsiteURL
	}
	return ""
}

func truncateNotifyContent(s string) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) > notifyContentLimit {
		return string(r[:notifyContentLimit])
	}
	return s
}
