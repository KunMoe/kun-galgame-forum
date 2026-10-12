package apiv1

import (
	"context"
	"errors"
	"strconv"
	"time"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/constants"
	msgModel "kun-galgame-api/internal/message/model"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

func sameID(p *int, id int) bool {
	return p != nil && *p == id
}

func (x *Interactions) choiceReply(ctx context.Context, topicID int, raw repr.DecimalID) (*model.TopicReply, error) {
	id, ok := parsePositiveID(string(raw))
	if !ok {
		return nil, unknownReply()
	}
	row, err := x.reads.replies.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, unknownReply()
		}
		return nil, problem.Internal(err)
	}
	if row.TopicID != topicID || row.Status != 0 {
		return nil, unknownReply()
	}
	users, p := x.reads.lookupUsers(ctx, []int{row.UserID})
	if p != nil {
		return nil, p
	}
	u, ok := users[row.UserID]
	if !ok || !userclient.IsRenderable(u) {
		return nil, unknownReply()
	}
	return row, nil
}

func (x *Interactions) setBestAnswer(ctx context.Context, in *setTopicReplyInput) (*topicOutput, error) {
	topic, user, p := x.visiblePublishedTopic(ctx, in.TopicID)
	if p != nil {
		return nil, p
	}
	if !capsForTopic(topic, user).SetBestAnswer {
		return nil, permissionRequired()
	}
	reply, err := x.choiceReply(ctx, topic.ID, in.Body.ReplyID)
	if err != nil {
		return nil, err
	}
	if sameID(topic.BestAnswerID, reply.ID) {
		return x.topicOut(ctx, topic.ID, user)
	}
	var prev *model.TopicReply
	if topic.BestAnswerID != nil {
		row, findErr := x.reads.replies.FindByID(*topic.BestAnswerID)
		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return nil, problem.Internal(findErr)
		}
		if findErr == nil {
			prev = row
		}
	}
	var jobs []pendingAward
	err = x.db.Transaction(func(tx *gorm.DB) error {
		jobs = jobs[:0]
		id := reply.ID
		if err := x.reads.topics.SetBestAnswerID(tx, topic.ID, &id, true, time.Now()); err != nil {
			return err
		}
		if reply.UserID != topic.UserID {
			jobs = append(jobs, pendingAward{
				userID: reply.UserID,
				delta:  constants.RewardBestAnswer,
				reason: moemoepoint.ReasonContentApproved,
				ref:    moemoepoint.Ref("topic_reply", reply.ID),
				key:    moemoepoint.KeyNonce("best_answer_set", "topic_reply_"+strconv.Itoa(reply.ID)),
			})
			if err := x.emitSolution(tx, user.ID, reply.UserID, replyPlainPreview(*reply), reply); err != nil {
				return err
			}
		}
		if prev != nil && prev.UserID != topic.UserID {
			jobs = append(jobs, pendingAward{
				userID: prev.UserID,
				delta:  -constants.RewardBestAnswer,
				reason: moemoepoint.ReasonContentRemoved,
				ref:    moemoepoint.Ref("topic_reply", prev.ID),
				key:    moemoepoint.KeyNonce("best_answer_cleared", "topic_reply_"+strconv.Itoa(prev.ID)),
			})
		}
		return nil
	})
	if err := x.afterCommit(err, jobs); err != nil {
		return nil, mapEngageErr(err)
	}
	return x.topicOut(ctx, topic.ID, user)
}

func (x *Interactions) clearBestAnswer(ctx context.Context, in *clearTopicReplyInput) (*topicOutput, error) {
	topic, user, p := x.visiblePublishedTopic(ctx, in.TopicID)
	if p != nil {
		return nil, p
	}
	if !capsForTopic(topic, user).SetBestAnswer {
		return nil, permissionRequired()
	}
	if topic.BestAnswerID == nil {
		return x.topicOut(ctx, topic.ID, user)
	}
	prev, findErr := x.reads.replies.FindByID(*topic.BestAnswerID)
	if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return nil, problem.Internal(findErr)
	}
	var jobs []pendingAward
	err := x.db.Transaction(func(tx *gorm.DB) error {
		jobs = jobs[:0]
		if err := x.reads.topics.SetBestAnswerID(tx, topic.ID, nil, false, time.Time{}); err != nil {
			return err
		}
		if prev != nil && prev.UserID != topic.UserID {
			jobs = append(jobs, pendingAward{
				userID: prev.UserID,
				delta:  -constants.RewardBestAnswer,
				reason: moemoepoint.ReasonContentRemoved,
				ref:    moemoepoint.Ref("topic_reply", prev.ID),
				key:    moemoepoint.KeyNonce("best_answer_cleared", "topic_reply_"+strconv.Itoa(prev.ID)),
			})
		}
		return nil
	})
	if err := x.afterCommit(err, jobs); err != nil {
		return nil, mapEngageErr(err)
	}
	return x.topicOut(ctx, topic.ID, user)
}

func (x *Interactions) pinReply(ctx context.Context, in *setTopicReplyInput) (*topicOutput, error) {
	topic, user, p := x.visiblePublishedTopic(ctx, in.TopicID)
	if p != nil {
		return nil, p
	}
	if !capsForTopic(topic, user).PinReply {
		return nil, permissionRequired()
	}
	reply, err := x.choiceReply(ctx, topic.ID, in.Body.ReplyID)
	if err != nil {
		return nil, err
	}
	if sameID(topic.PinnedReplyID, reply.ID) {
		return x.topicOut(ctx, topic.ID, user)
	}
	err = x.db.Transaction(func(tx *gorm.DB) error {
		id := reply.ID
		if err := x.reads.topics.SetPinnedReplyID(tx, topic.ID, &id); err != nil {
			return err
		}
		if user.ID == reply.UserID {
			return nil
		}
		return notifyTopicLink(tx, user.ID, reply.UserID, "pin-reply", replyPlainPreview(*reply), topic.ID, reply.Floor, msgModel.ReplyRef(reply.ID, topic.ID))
	})
	if err != nil {
		return nil, mapEngageErr(err)
	}
	return x.topicOut(ctx, topic.ID, user)
}

func (x *Interactions) unpinReply(ctx context.Context, in *clearTopicReplyInput) (*topicOutput, error) {
	topic, user, p := x.visiblePublishedTopic(ctx, in.TopicID)
	if p != nil {
		return nil, p
	}
	if !capsForTopic(topic, user).PinReply {
		return nil, permissionRequired()
	}
	if topic.PinnedReplyID == nil {
		return x.topicOut(ctx, topic.ID, user)
	}
	if err := x.reads.topics.SetPinnedReplyID(x.db, topic.ID, nil); err != nil {
		return nil, problem.Internal(err)
	}
	return x.topicOut(ctx, topic.ID, user)
}
