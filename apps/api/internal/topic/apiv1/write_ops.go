package apiv1

import (
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/trust/gate"
	userRepo "kun-galgame-api/internal/user/repository"
)

type AwardFunc func(userID, delta int, reason, ref, idempotencyKey string)

type Writes struct {
	reads  *Service
	state  *userRepo.StateRepository
	check  *gate.CheckService
	scan   *gate.ScanService
	award  AwardFunc
	notify msgService.Notifier
}

func NewWrites(
	reads *Service,
	state *userRepo.StateRepository,
	check *gate.CheckService,
	scan *gate.ScanService,
	award AwardFunc,
	notify msgService.Notifier,
) *Writes {
	if check == nil {
		check = gate.NewCheckService(nil)
	}
	if scan == nil {
		scan = gate.NewScanService(nil)
	}
	if award == nil {
		award = moemoepoint.Award
	}
	if state == nil && reads != nil && reads.topics != nil {
		state = userRepo.NewStateRepository(reads.topics.DB())
	}
	return &Writes{reads: reads, state: state, check: check, scan: scan, award: award, notify: notify}
}

type createTopicInput struct {
	Body TopicCreate
}

type createTopicOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new topic, such as /api/v1/topics/4121."`
	Body     Topic
}

type updateTopicInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
	Body    TopicPatch
}

type updateTopicOutput struct {
	Body Topic
}

type getTopicSourceInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
}

type getTopicSourceOutput struct {
	Body TopicSource
}

type createReplyInput struct {
	TopicID string `path:"topic_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Topic id."`
	Body    ReplyCreate
}

type createReplyOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new reply, such as /api/v1/replies/16335."`
	Body     Reply
}

type updateReplyInput struct {
	ReplyID string `path:"reply_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Reply id."`
	Body    ReplyPatch
}

type updateReplyOutput struct {
	Body Reply
}

type deleteReplyInput struct {
	ReplyID string `path:"reply_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Reply id."`
}

type getReplySourceInput struct {
	ReplyID string `path:"reply_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Reply id."`
}

type getReplySourceOutput struct {
	Body ReplySource
}

type createCommentInput struct {
	ReplyID string `path:"reply_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Reply id."`
	Body    CommentCreate
}

type createCommentOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new comment, such as /api/v1/comments/2087."`
	Body     Comment
}

type commentInput struct {
	CommentID string `path:"comment_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Comment id."`
}

type updateCommentInput struct {
	CommentID string `path:"comment_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Comment id."`
	Body      CommentPatch
}

type commentOutput struct {
	Body Comment
}

type getCommentSourceOutput struct {
	Body CommentSource
}
