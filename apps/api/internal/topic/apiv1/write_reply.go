package apiv1

import (
	"context"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/infrastructure/markdown"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/internal/topic/service"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

func (w *Writes) createReply(ctx context.Context, in *createReplyInput) (*createReplyOutput, error) {
	if w == nil || w.reads == nil || w.db() == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	topic, user, p := w.reads.visibleTopic(ctx, in.TopicID)
	if p != nil {
		return nil, p
	}
	if strings.TrimSpace(in.Body.ContentMarkdown) == "" {
		return nil, validationFailed(tooShort("/content_markdown"))
	}
	body := markdown.NormalizeStoredContent(in.Body.ContentMarkdown)
	decision, matched, p := w.rejectContent(ctx, body, user.ID)
	if p != nil {
		return nil, p
	}

	var awards []pendingAward
	var replyID int
	err := w.db().Transaction(func(tx *gorm.DB) error {
		floor, err := repository.NextReplyFloor(tx, topic.ID)
		if err != nil {
			return err
		}
		row := &model.TopicReply{
			UserID:  user.ID,
			TopicID: topic.ID,
			Floor:   floor,
			Content: body,
		}
		if err := w.reads.replies.CreateReply(tx, row); err != nil {
			return err
		}
		if err := w.reads.topics.TouchStatusUpdateTime(tx, topic.ID, time.Now()); err != nil {
			return err
		}
		if err := service.RecomputeTopicCounts(tx, topic.ID); err != nil {
			return err
		}
		preview := truncatePreview(strings.TrimSpace(body), constants.TextPreviewLength)
		mentioned := service.MentionedIDs(body, mentionCap)
		levels, err := repository.SubscriptionLevels(tx, topic.ID, []int{topic.UserID})
		if err != nil {
			return err
		}
		if topic.UserID != user.ID {
			if levels[topic.UserID] == model.SubscriptionWatching {
				var h service.InteractionHelpers
				if err := h.CreateReplyMessage(tx, user.ID, topic.UserID, "replied", preview, topic.ID, row.Floor, 0); err != nil {
					return err
				}
			}
			awards = append(awards, repliedAward(topic.UserID, row.ID))
		}
		if err := w.notifyMentions(tx, user.ID, topic.ID, row.Floor, body); err != nil {
			return err
		}
		if err := repository.FanOutReply(tx, repository.ReplyFanOut{
			TopicID:    topic.ID,
			Floor:      row.Floor,
			ReplierID:  user.ID,
			Preview:    preview,
			Link:       msgService.BuildTopicLink(topic.ID, row.Floor, 0),
			NoticeType: string(msgService.NotifySubscribedTopic),
			Exclude:    append([]int{topic.UserID}, mentioned...),
			Notify:     fansOutReplies(topic),
		}); err != nil {
			return err
		}
		replyID = row.ID
		return nil
	})
	if p := txProblem(err); p != nil {
		return nil, p
	}
	if err != nil {
		return nil, problem.Internal(err)
	}
	w.flushAwards(awards)
	w.scanReply(decision, matched, replyID, user.ID, body)

	fresh, p := w.loadReply(replyID)
	if p != nil {
		return nil, p
	}
	mapped, p := w.reads.buildOneReply(ctx, topic, *fresh, user)
	if p != nil {
		return nil, p
	}
	if mapped == nil {
		return nil, notFound()
	}
	return &createReplyOutput{
		Location: "/api/v1/replies/" + strconv.Itoa(replyID),
		Body:     *mapped,
	}, nil
}

func (w *Writes) updateReply(ctx context.Context, in *updateReplyInput) (*updateReplyOutput, error) {
	if w == nil || w.reads == nil || w.db() == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	topic, row, user, p := w.reads.visibleReply(ctx, in.ReplyID)
	if p != nil {
		return nil, p
	}
	if !capsForReply(topic, row, user).Edit {
		return nil, permissionRequired()
	}
	if in.Body.ContentMarkdown == nil {
		mapped, p := w.reads.buildOneReply(ctx, topic, *row, user)
		if p != nil {
			return nil, p
		}
		if mapped == nil {
			return nil, notFound()
		}
		return &updateReplyOutput{Body: *mapped}, nil
	}
	if strings.TrimSpace(*in.Body.ContentMarkdown) == "" {
		return nil, validationFailed(tooShort("/content_markdown"))
	}
	body := markdown.NormalizeStoredContent(*in.Body.ContentMarkdown)
	if body == row.Content {
		mapped, p := w.reads.buildOneReply(ctx, topic, *row, user)
		if p != nil {
			return nil, p
		}
		if mapped == nil {
			return nil, notFound()
		}
		return &updateReplyOutput{Body: *mapped}, nil
	}
	decision, matched, p := w.rejectContent(ctx, body, row.UserID)
	if p != nil {
		return nil, p
	}
	now := time.Now()
	err := w.db().Transaction(func(tx *gorm.DB) error {
		if err := w.reads.replies.UpdateReplyContent(tx, row.ID, map[string]any{
			"content": body,
			"edited":  &now,
		}); err != nil {
			return err
		}
		return w.notifyMentions(tx, row.UserID, row.TopicID, row.Floor, body)
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	w.scanReply(decision, matched, row.ID, row.UserID, body)

	fresh, p := w.loadReply(row.ID)
	if p != nil {
		return nil, p
	}
	mapped, p := w.reads.buildOneReply(ctx, topic, *fresh, user)
	if p != nil {
		return nil, p
	}
	if mapped == nil {
		return nil, notFound()
	}
	return &updateReplyOutput{Body: *mapped}, nil
}

func (w *Writes) deleteReply(ctx context.Context, in *deleteReplyInput) (*struct{}, error) {
	if w == nil || w.reads == nil || w.db() == nil || w.state == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	topic, row, user, p := w.reads.visibleReply(ctx, in.ReplyID)
	if p != nil {
		return nil, p
	}
	if !capsForReply(topic, row, user).Delete {
		return nil, permissionRequired()
	}
	authorDelete := user.ID == row.UserID
	var awards []pendingAward
	err := w.db().Transaction(func(tx *gorm.DB) error {
		penalty := 3
		if authorDelete {
			var comments int64
			if err := tx.Model(&model.TopicComment{}).
				Where("topic_reply_id = ? AND status = 0", row.ID).
				Count(&comments).Error; err != nil {
				return err
			}
			penalty = 3 * (row.LikeCount + int(comments) + 1)
			state, err := w.state.LockForUpdate(tx, row.UserID)
			if err != nil {
				return err
			}
			if state.Moemoepoint < penalty {
				return txFail{p: moemoepointInsufficient(penalty)}
			}
		}
		if err := w.reads.replies.DeleteRepliesByIDs(tx, []int{row.ID}); err != nil {
			return err
		}
		if err := service.RecomputeTopicCounts(tx, row.TopicID); err != nil {
			return err
		}
		awards = append(awards, replyDeletedAward(row.UserID, row.ID, penalty))
		return nil
	})
	if p := txProblem(err); p != nil {
		return nil, p
	}
	if err != nil {
		return nil, problem.Internal(err)
	}
	w.flushAwards(awards)
	return nil, nil
}

func truncatePreview(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}

// Watchers are known only by id, and a role grant needs the reader's roles,
// so a restricted or hidden topic sends no folded notice: a wrong guess would
// mail its reply preview to someone who cannot open it.
func fansOutReplies(topic *model.Topic) bool {
	return topic.Status == 0 && (topic.AccessScope == "public" || topic.AccessScope == "login")
}
