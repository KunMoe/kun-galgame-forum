package apiv1

import (
	"log/slog"

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/message/notifytype"
	"kun-galgame-api/internal/message/repository"
	"kun-galgame-api/pkg/userclient"
)

func (s *Service) mapNotification(row repository.V1NotificationRow, users map[int]userclient.User) (Notification, bool) {
	tok, ok := notifytype.FromDB(row.Type)
	if !ok {
		slog.Warn("dropping notification with unknown type", "id", row.ID, "type", row.Type)
		return Notification{}, false
	}
	actor, keep := s.userRef(users, row.SenderID)
	if !keep {
		return Notification{}, false
	}
	origin := "local"
	if row.CommunityNotificationID != nil {
		origin = "community"
	}
	return Notification{
		Object:           "notification",
		ID:               repr.ID(row.ID),
		NotificationType: tok,
		Actor:            actor,
		ActorCount:       countAtLeastOne(row.ActorCount),
		ItemCount:        countAtLeastOne(row.ItemCount),
		Path:             notificationPath(row.Link),
		ExcerptMarkdown:  truncateRunes(row.Content, excerptRuneLimit),
		Origin:           origin,
		IsRead:           row.Status == "read",
		CreatedAt:        repr.Timestamp(row.Created),
	}, true
}

func (s *Service) mapDirectMessage(
	row repository.V1ChatMessageRow,
	sender repr.UserRef,
	doc content.ContentDocument,
	callerID int,
) DirectMessage {
	state := "sent"
	if row.IsRecall {
		state = "recalled"
		doc = content.NewDocument(nil)
	}
	return DirectMessage{
		Object:     "direct_message",
		ID:         repr.ID(row.ID),
		Sender:     sender,
		State:      state,
		Content:    doc,
		CreatedAt:  repr.Timestamp(row.Created),
		RecalledAt: repr.TimestampPtr(row.RecallTime),
		Viewer:     DirectMessageViewer{IsMine: row.SenderID == callerID},
	}
}

func lastMessageRow(row repository.V1ConversationRow) repository.V1ChatMessageRow {
	return repository.V1ChatMessageRow{
		ID:         row.LastID,
		SenderID:   row.LastSenderID,
		Content:    row.LastContent,
		IsRecall:   row.LastIsRecall,
		RecallTime: row.LastRecallTime,
		Created:    row.LastCreated,
	}
}
