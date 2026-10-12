package apiv1

import (
	"context"
	"strconv"

	"kun-galgame-api/internal/constants"
	msgModel "kun-galgame-api/internal/message/model"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/moemoepoint"

	"gorm.io/gorm"
)

func (x *Interactions) likeComment(ctx context.Context, in *commentInput) (*commentOutput, error) {
	if p := x.ready(); p != nil {
		return nil, p
	}
	topic, _, comment, user, p := x.reads.visibleComment(ctx, in.CommentID)
	if p != nil {
		return nil, p
	}
	if !capsForComment(topic, comment.UserID, user).Like {
		return nil, selfLikeForbidden()
	}
	var jobs []pendingAward
	err := x.db.Transaction(func(tx *gorm.DB) error {
		jobs = jobs[:0]
		id, inserted, err := x.reads.comments.InsertCommentLike(tx, comment.ID, user.ID)
		if err != nil || !inserted {
			return err
		}
		jobs = append(jobs, pendingAward{
			userID: comment.UserID,
			delta:  1,
			reason: moemoepoint.ReasonLiked,
			ref:    moemoepoint.Ref("topic_comment", comment.ID),
			key:    moemoepoint.Key("liked", "topic_comment_like_"+strconv.Itoa(id)),
		})
		preview := truncateRunes(comment.Content, constants.TextPreviewLength)
		return dedupMessage(tx, user.ID, comment.UserID, "liked", preview,
			msgService.BuildTopicLink(comment.TopicID, 0, comment.ID), msgModel.TopicCommentRef(comment.ID, comment.TopicID))
	})
	if err := x.afterCommit(err, jobs); err != nil {
		return nil, mapEngageErr(err)
	}
	return x.reads.commentOut(ctx, topic, comment.ID, user)
}

func (x *Interactions) unlikeComment(ctx context.Context, in *commentInput) (*commentOutput, error) {
	if p := x.ready(); p != nil {
		return nil, p
	}
	topic, _, comment, user, p := x.reads.visibleComment(ctx, in.CommentID)
	if p != nil {
		return nil, p
	}
	var jobs []pendingAward
	err := x.db.Transaction(func(tx *gorm.DB) error {
		jobs = jobs[:0]
		id, deleted, err := x.reads.comments.DeleteCommentLikeRow(tx, comment.ID, user.ID)
		if err != nil || !deleted {
			return err
		}
		jobs = append(jobs, pendingAward{
			userID: comment.UserID,
			delta:  -1,
			reason: moemoepoint.ReasonLiked,
			ref:    moemoepoint.Ref("topic_comment", comment.ID),
			key:    moemoepoint.Key("unliked", "topic_comment_like_"+strconv.Itoa(id)),
		})
		return nil
	})
	if err := x.afterCommit(err, jobs); err != nil {
		return nil, mapEngageErr(err)
	}
	return x.reads.commentOut(ctx, topic, comment.ID, user)
}
