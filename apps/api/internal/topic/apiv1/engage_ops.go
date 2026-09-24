package apiv1

import (
	"context"
	"errors"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

type Interactions struct {
	reads  *Service
	db     *gorm.DB
	notify msgService.Notifier
	award  AwardFunc
}

func NewInteractions(reads *Service, db *gorm.DB, notify msgService.Notifier, award AwardFunc) *Interactions {
	if db == nil && reads != nil && reads.topics != nil {
		db = reads.topics.DB()
	}
	if award == nil {
		award = moemoepoint.Award
	}
	return &Interactions{reads: reads, db: db, notify: notify, award: award}
}

type topicReactionInput struct {
	TopicID  string        `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
	Reaction ReactionInput `path:"reaction"`
}

type topicEngagementOutput struct {
	Body TopicEngagement
}

type topicFavoriteInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
}

type upvoteTopicInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
	Body    UpvoteCreate
}

type upvoteTopicOutput struct {
	Body TopicUpvote
}

type listTopicUpvotesInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
	collect.Page
}

type listTopicUpvotesOutput struct {
	Body repr.List[TopicUpvote]
}

type listTopicReactionsInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
	collect.Page
}

type listReactionsOutput struct {
	Body repr.List[Reaction]
}

type setTopicReplyInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
	Body    ReplyChoice
}

type clearTopicReplyInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
}

type topicOutput struct {
	Body Topic
}

type replyReactionInput struct {
	ReplyID  string        `path:"reply_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Reply id."`
	Reaction ReactionInput `path:"reaction"`
}

type replyEngagementOutput struct {
	Body ReplyEngagement
}

type listReplyReactionsInput struct {
	ReplyID string `path:"reply_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Reply id."`
	collect.Page
}

type engageError struct {
	p *problem.Problem
}

func (e engageError) Error() string {
	if e.p == nil {
		return ""
	}
	return e.p.Error()
}

func (x *Interactions) ready() *problem.Problem {
	if x == nil || x.reads == nil || x.db == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

func (x *Interactions) visiblePublishedTopic(ctx context.Context, idStr string) (*model.Topic, *middleware.UserInfo, *problem.Problem) {
	if p := x.ready(); p != nil {
		return nil, nil, p
	}
	topic, user, p := x.reads.visibleTopic(ctx, idStr)
	if p != nil {
		return nil, nil, p
	}
	if topic.Status != 0 {
		return nil, nil, notFound()
	}
	return topic, user, nil
}

func (x *Interactions) visiblePublishedReply(ctx context.Context, idStr string) (*model.Topic, *model.TopicReply, *middleware.UserInfo, *problem.Problem) {
	if p := x.ready(); p != nil {
		return nil, nil, nil, p
	}
	topic, reply, user, p := x.reads.visibleReply(ctx, idStr)
	if p != nil {
		return nil, nil, nil, p
	}
	if topic.Status != 0 {
		return nil, nil, nil, notFound()
	}
	return topic, reply, user, nil
}

func (x *Interactions) afterCommit(err error, jobs []pendingAward) error {
	if err != nil {
		return err
	}
	for _, j := range jobs {
		x.award(j.userID, j.delta, j.reason, j.ref, j.key)
	}
	return nil
}

func mapEngageErr(err error) error {
	if err == nil {
		return nil
	}
	var ee engageError
	if errors.As(err, &ee) && ee.p != nil {
		return ee.p
	}
	return problem.Internal(err)
}

func selfLikeForbidden() *problem.Problem {
	return problem.New(problem.CodeSelfLikeForbidden, "Users cannot like their own topics, replies or comments.")
}

func selfUpvoteForbidden() *problem.Problem {
	return problem.New(problem.CodeSelfUpvoteForbidden, "Users cannot upvote their own topics.")
}

func unknownReply() *problem.Problem {
	return problem.New(
		problem.CodeValidationFailed,
		"The reply is not a visible reply of this topic.",
		problem.AtPointer("/reply_id", problem.ReasonUnknownReference, "the reply is not a visible reply of this topic", nil),
	)
}
