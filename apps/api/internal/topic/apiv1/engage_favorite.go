package apiv1

import (
	"context"
	"strconv"

	msgModel "kun-galgame-api/internal/message/model"
	"kun-galgame-api/internal/moemoepoint"

	"gorm.io/gorm"
)

func (x *Interactions) favoriteTopic(ctx context.Context, in *topicFavoriteInput) (*topicEngagementOutput, error) {
	topic, user, p := x.visiblePublishedTopic(ctx, in.TopicID)
	if p != nil {
		return nil, p
	}
	var jobs []pendingAward
	err := x.db.Transaction(func(tx *gorm.DB) error {
		jobs = jobs[:0]
		id, inserted, err := x.reads.topics.InsertTopicFavorite(tx, topic.ID, user.ID)
		if err != nil || !inserted {
			return err
		}
		if err := x.reads.topics.AdjustTopicCount(tx, topic.ID, "favorite_count", 1); err != nil {
			return err
		}
		if user.ID == topic.UserID {
			return nil
		}
		jobs = append(jobs, pendingAward{
			userID: topic.UserID,
			delta:  1,
			reason: moemoepoint.ReasonLiked,
			ref:    moemoepoint.Ref("topic", topic.ID),
			key:    moemoepoint.Key("favorited", "topic_favorite_"+strconv.Itoa(id)),
		})
		return notifyTopicLink(tx, user.ID, topic.UserID, "favorite", topicPreview(topic.Title), topic.ID, 0, msgModel.TopicRef(topic.ID))
	})
	if err := x.afterCommit(err, jobs); err != nil {
		return nil, mapEngageErr(err)
	}
	return x.topicEngagementOut(ctx, topic.ID, user)
}

func (x *Interactions) unfavoriteTopic(ctx context.Context, in *topicFavoriteInput) (*topicEngagementOutput, error) {
	topic, user, p := x.visiblePublishedTopic(ctx, in.TopicID)
	if p != nil {
		return nil, p
	}
	var jobs []pendingAward
	err := x.db.Transaction(func(tx *gorm.DB) error {
		jobs = jobs[:0]
		id, deleted, err := x.reads.topics.DeleteTopicFavoriteRow(tx, topic.ID, user.ID)
		if err != nil || !deleted {
			return err
		}
		if err := x.reads.topics.AdjustTopicCount(tx, topic.ID, "favorite_count", -1); err != nil {
			return err
		}
		if user.ID == topic.UserID {
			return nil
		}
		jobs = append(jobs, pendingAward{
			userID: topic.UserID,
			delta:  -1,
			reason: moemoepoint.ReasonLiked,
			ref:    moemoepoint.Ref("topic", topic.ID),
			key:    moemoepoint.Key("unfavorited", "topic_favorite_"+strconv.Itoa(id)),
		})
		return nil
	})
	if err := x.afterCommit(err, jobs); err != nil {
		return nil, mapEngageErr(err)
	}
	return x.topicEngagementOut(ctx, topic.ID, user)
}
