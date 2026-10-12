package apiv1

import (
	"context"
	"strconv"

	msgModel "kun-galgame-api/internal/message/model"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/topic/model"

	"gorm.io/gorm"
)

func (x *Interactions) setTopicReaction(ctx context.Context, in *topicReactionInput) (*topicEngagementOutput, error) {
	topic, user, p := x.visiblePublishedTopic(ctx, in.TopicID)
	if p != nil {
		return nil, p
	}
	tok := string(in.Reaction)
	if tok == "like" && user.ID == topic.UserID {
		return nil, selfLikeForbidden()
	}
	var jobs []pendingAward
	err := x.db.Transaction(func(tx *gorm.DB) error {
		jobs = jobs[:0]
		return x.applyTopicReactionSet(tx, topic, user.ID, tok, &jobs)
	})
	if err := x.afterCommit(err, jobs); err != nil {
		return nil, mapEngageErr(err)
	}
	return x.topicEngagementOut(ctx, topic.ID, user)
}

func (x *Interactions) removeTopicReaction(ctx context.Context, in *topicReactionInput) (*topicEngagementOutput, error) {
	topic, user, p := x.visiblePublishedTopic(ctx, in.TopicID)
	if p != nil {
		return nil, p
	}
	var jobs []pendingAward
	err := x.db.Transaction(func(tx *gorm.DB) error {
		jobs = jobs[:0]
		return x.applyTopicReactionRemove(tx, topic, user.ID, string(in.Reaction), &jobs)
	})
	if err := x.afterCommit(err, jobs); err != nil {
		return nil, mapEngageErr(err)
	}
	return x.topicEngagementOut(ctx, topic.ID, user)
}

func (x *Interactions) applyTopicReactionSet(tx *gorm.DB, topic *model.Topic, userID int, tok string, jobs *[]pendingAward) error {
	switch tok {
	case "like":
		if err := x.applyTopicReactionRemove(tx, topic, userID, "dislike", jobs); err != nil {
			return err
		}
	case "dislike":
		if err := x.applyTopicReactionRemove(tx, topic, userID, "like", jobs); err != nil {
			return err
		}
	}
	id, inserted, err := x.reads.topics.InsertTopicReaction(tx, topic.ID, userID, tok)
	if err != nil || !inserted {
		return err
	}
	switch tok {
	case "like":
		if err := x.reads.topics.AdjustTopicCount(tx, topic.ID, "like_count", 1); err != nil {
			return err
		}
		*jobs = append(*jobs, pendingAward{
			userID: topic.UserID,
			delta:  1,
			reason: moemoepoint.ReasonLiked,
			ref:    moemoepoint.Ref("topic", topic.ID),
			key:    moemoepoint.Key("liked", "topic_reaction_"+strconv.Itoa(id)),
		})
		return notifyTopicLink(tx, userID, topic.UserID, "liked", topicPreview(topic.Title), topic.ID, 0, msgModel.TopicRef(topic.ID))
	case "dislike":
		return x.reads.topics.AdjustTopicCount(tx, topic.ID, "dislike_count", 1)
	}
	return nil
}

func (x *Interactions) applyTopicReactionRemove(tx *gorm.DB, topic *model.Topic, userID int, tok string, jobs *[]pendingAward) error {
	id, deleted, err := x.reads.topics.DeleteTopicReaction(tx, topic.ID, userID, tok)
	if err != nil || !deleted {
		return err
	}
	switch tok {
	case "like":
		if err := x.reads.topics.AdjustTopicCount(tx, topic.ID, "like_count", -1); err != nil {
			return err
		}
		*jobs = append(*jobs, pendingAward{
			userID: topic.UserID,
			delta:  -1,
			reason: moemoepoint.ReasonLiked,
			ref:    moemoepoint.Ref("topic", topic.ID),
			key:    moemoepoint.Key("unliked", "topic_reaction_"+strconv.Itoa(id)),
		})
	case "dislike":
		return x.reads.topics.AdjustTopicCount(tx, topic.ID, "dislike_count", -1)
	}
	return nil
}

func (x *Interactions) setReplyReaction(ctx context.Context, in *replyReactionInput) (*replyEngagementOutput, error) {
	topic, reply, user, p := x.visiblePublishedReply(ctx, in.ReplyID)
	if p != nil {
		return nil, p
	}
	tok := string(in.Reaction)
	if tok == "like" && user.ID == reply.UserID {
		return nil, selfLikeForbidden()
	}
	var jobs []pendingAward
	err := x.db.Transaction(func(tx *gorm.DB) error {
		jobs = jobs[:0]
		return x.applyReplyReactionSet(tx, topic, reply, user.ID, tok, &jobs)
	})
	if err := x.afterCommit(err, jobs); err != nil {
		return nil, mapEngageErr(err)
	}
	return x.replyEngagementOut(ctx, topic, reply.ID, user)
}

func (x *Interactions) removeReplyReaction(ctx context.Context, in *replyReactionInput) (*replyEngagementOutput, error) {
	topic, reply, user, p := x.visiblePublishedReply(ctx, in.ReplyID)
	if p != nil {
		return nil, p
	}
	var jobs []pendingAward
	err := x.db.Transaction(func(tx *gorm.DB) error {
		jobs = jobs[:0]
		return x.applyReplyReactionRemove(tx, reply, user.ID, string(in.Reaction), &jobs)
	})
	if err := x.afterCommit(err, jobs); err != nil {
		return nil, mapEngageErr(err)
	}
	return x.replyEngagementOut(ctx, topic, reply.ID, user)
}

func (x *Interactions) applyReplyReactionSet(tx *gorm.DB, topic *model.Topic, reply *model.TopicReply, userID int, tok string, jobs *[]pendingAward) error {
	switch tok {
	case "like":
		if err := x.applyReplyReactionRemove(tx, reply, userID, "dislike", jobs); err != nil {
			return err
		}
	case "dislike":
		if err := x.applyReplyReactionRemove(tx, reply, userID, "like", jobs); err != nil {
			return err
		}
	}
	id, inserted, err := x.reads.replies.InsertReplyReaction(tx, reply.ID, userID, tok)
	if err != nil || !inserted {
		return err
	}
	switch tok {
	case "like":
		if err := x.reads.replies.AdjustReplyCount(tx, reply.ID, "like_count", 1); err != nil {
			return err
		}
		*jobs = append(*jobs, pendingAward{
			userID: reply.UserID,
			delta:  1,
			reason: moemoepoint.ReasonLiked,
			ref:    moemoepoint.Ref("topic_reply", reply.ID),
			key:    moemoepoint.Key("liked", "topic_reply_reaction_"+strconv.Itoa(id)),
		})
		return notifyTopicLink(tx, userID, reply.UserID, "liked", replyContentPreview(reply.Content), topic.ID, reply.Floor, msgModel.ReplyRef(reply.ID, topic.ID))
	case "dislike":
		return x.reads.replies.AdjustReplyCount(tx, reply.ID, "dislike_count", 1)
	}
	return nil
}

func (x *Interactions) applyReplyReactionRemove(tx *gorm.DB, reply *model.TopicReply, userID int, tok string, jobs *[]pendingAward) error {
	id, deleted, err := x.reads.replies.DeleteReplyReaction(tx, reply.ID, userID, tok)
	if err != nil || !deleted {
		return err
	}
	switch tok {
	case "like":
		if err := x.reads.replies.AdjustReplyCount(tx, reply.ID, "like_count", -1); err != nil {
			return err
		}
		*jobs = append(*jobs, pendingAward{
			userID: reply.UserID,
			delta:  -1,
			reason: moemoepoint.ReasonLiked,
			ref:    moemoepoint.Ref("topic_reply", reply.ID),
			key:    moemoepoint.Key("unliked", "topic_reply_reaction_"+strconv.Itoa(id)),
		})
	case "dislike":
		return x.reads.replies.AdjustReplyCount(tx, reply.ID, "dislike_count", -1)
	}
	return nil
}
