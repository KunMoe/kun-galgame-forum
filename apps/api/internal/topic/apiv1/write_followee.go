package apiv1

import (
	"context"
	"log/slog"
	"time"

	msgService "kun-galgame-api/internal/message/service"
)

const followeeNotifyTimeout = 30 * time.Second
const followeeNotifyPage = 100
const followeeNotifyMaxPages = 50

func (w *Writes) notifyFollowersOfTopic(authorID, topicID int, scope string) {
	if scope != "public" || w.notify == nil || w.community == nil || !w.community.Configured() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), followeeNotifyTimeout)
		defer cancel()
		if err := w.fanOutFolloweeTopic(ctx, authorID, topicID); err != nil {
			slog.Warn("followee topic notify", "topic_id", topicID, "author_id", authorID, "err", err)
		}
	}()
}

func (w *Writes) fanOutFolloweeTopic(ctx context.Context, authorID, topicID int) error {
	cursor := ""
	for page := 0; page < followeeNotifyMaxPages; page++ {
		list, err := w.community.ListFollowers(ctx, int64(authorID), cursor, followeeNotifyPage)
		if err != nil {
			return err
		}
		for _, row := range list.Users {
			if err := w.notify.Emit(nil, msgService.Spec{
				SenderID:   authorID,
				ReceiverID: int(row.UserID),
				Kind:       msgService.NotifyFolloweeTopic,
				TopicID:    topicID,
			}); err != nil {
				slog.Warn("followee topic notify emit", "topic_id", topicID, "receiver_id", row.UserID, "err", err)
			}
		}
		if list.NextCursor == "" {
			return nil
		}
		cursor = list.NextCursor
	}
	return nil
}
