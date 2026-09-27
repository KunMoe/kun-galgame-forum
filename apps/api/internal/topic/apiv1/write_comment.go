package apiv1

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/service"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

const (
	commentTextLimit = 1000
	// varchar(1007). NormalizeStoredContent rewrites a sticker URL into a
	// longer /image/<hash>_320 token, so a body inside the request limit can
	// still overflow the column.
	commentStoredLimit = 1007
	commentDeleteCost  = 3
)

func commentTooLong() problem.FieldError {
	max := commentTextLimit
	return problem.AtPointer("/text", problem.ReasonTooLong,
		fmt.Sprintf("must hold at most %d characters once its image references are rewritten", commentTextLimit),
		&problem.FieldParams{MaxLength: &max})
}

func unknownParentComment() problem.FieldError {
	return problem.AtPointer("/parent_comment_id", problem.ReasonUnknownReference,
		"the comment is not a visible comment of this reply", nil)
}

func commentedAward(authorID, commentID int) pendingAward {
	return pendingAward{
		userID: authorID,
		delta:  constants.RewardReply,
		reason: moemoepoint.ReasonContentApproved,
		ref:    moemoepoint.Ref("topic_comment", commentID),
		key:    moemoepoint.Key("commented", fmt.Sprintf("topic_comment_%d", commentID)),
	}
}

func commentDeletedAward(authorID, commentID, penalty int) pendingAward {
	return pendingAward{
		userID: authorID,
		delta:  -penalty,
		reason: moemoepoint.ReasonContentRemoved,
		ref:    moemoepoint.Ref("topic_comment", commentID),
		key:    moemoepoint.Key("comment_deleted", fmt.Sprintf("topic_comment_%d", commentID)),
	}
}

func (w *Writes) scanComment(decision string, matched []string, commentID, authorID int, text string) {
	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindTopicComment, "subject_id", commentID, "author_id", authorID, "matched", matched)
	}
	w.scan.ScanBg(gate.SubjectKindTopicComment, strconv.Itoa(commentID), text, int64(authorID))
}

func storeCommentText(text string) (string, *problem.Problem) {
	if strings.TrimSpace(text) == "" {
		return "", validationFailed(tooShort("/text"))
	}
	body := markdown.NormalizeStoredContent(text)
	if utf8.RuneCountInString(body) > commentStoredLimit {
		return "", validationFailed(commentTooLong())
	}
	return body, nil
}

func (w *Writes) createComment(ctx context.Context, in *createCommentInput) (*createCommentOutput, error) {
	if w == nil || w.reads == nil || w.db() == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	topic, reply, user, p := w.reads.visibleReply(ctx, in.ReplyID)
	if p != nil {
		return nil, p
	}
	body, p := storeCommentText(in.Body.Text)
	if p != nil {
		return nil, p
	}
	parent, p := w.reads.parentComment(reply, in.Body.ParentCommentID)
	if p != nil {
		return nil, p
	}
	inReplyTo := reply.UserID
	var parentID *int
	if parent != nil {
		inReplyTo = parent.UserID
		parentID = &parent.ID
	}
	decision, matched, p := w.rejectContent(ctx, body, user.ID)
	if p != nil {
		return nil, p
	}

	var awards []pendingAward
	var commentID int
	err := w.db().Transaction(func(tx *gorm.DB) error {
		awards = awards[:0]
		id, _, err := w.reads.comments.InsertComment(tx, reply.ID, topic.ID, user.ID, inReplyTo, parentID, body)
		if err != nil {
			return err
		}
		commentID = id
		if err := w.reads.topics.TouchStatusUpdateTime(tx, topic.ID, time.Now()); err != nil {
			return err
		}
		if err := service.RecomputeTopicCounts(tx, topic.ID); err != nil {
			return err
		}
		if inReplyTo == user.ID {
			return nil
		}
		awards = append(awards, commentedAward(inReplyTo, id))
		muted, err := mutedReaders(tx, topic.ID, []int{inReplyTo})
		if err != nil || muted[inReplyTo] {
			return err
		}
		var h service.InteractionHelpers
		preview := truncatePreview(strings.TrimSpace(body), constants.TextPreviewLength)
		return h.CreateReplyMessage(tx, user.ID, inReplyTo, "commented", preview, topic.ID, 0, id)
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	w.flushAwards(awards)
	w.scanComment(decision, matched, commentID, user.ID, body)

	mapped, p := w.reads.buildOneComment(ctx, topic, commentID, user)
	if p != nil {
		return nil, p
	}
	return &createCommentOutput{
		Location: "/api/v1/comments/" + strconv.Itoa(commentID),
		Body:     *mapped,
	}, nil
}

func (s *Service) parentComment(reply *model.TopicReply, raw *repr.DecimalID) (*model.TopicComment, *problem.Problem) {
	if raw == nil {
		return nil, nil
	}
	id, ok := repr.ParseID(*raw)
	if !ok {
		return nil, validationFailed(unknownParentComment())
	}
	row, err := s.comments.FindCommentByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, validationFailed(unknownParentComment())
		}
		return nil, problem.Internal(err)
	}
	if row.Status != 0 || row.TopicReplyID != reply.ID {
		return nil, validationFailed(unknownParentComment())
	}
	return row, nil
}

func (w *Writes) updateComment(ctx context.Context, in *updateCommentInput) (*commentOutput, error) {
	if w == nil || w.reads == nil || w.db() == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	topic, _, comment, user, p := w.reads.visibleComment(ctx, in.CommentID)
	if p != nil {
		return nil, p
	}
	if !capsForComment(topic, comment.UserID, user).Edit {
		return nil, permissionRequired()
	}
	if in.Body.Text == nil {
		return w.reads.commentOut(ctx, topic, comment.ID, user)
	}
	body, p := storeCommentText(*in.Body.Text)
	if p != nil {
		return nil, p
	}
	// K18: an unchanged body is not new text, so it is not checked again.
	if body == comment.Content {
		return w.reads.commentOut(ctx, topic, comment.ID, user)
	}
	decision, matched, p := w.rejectContent(ctx, body, comment.UserID)
	if p != nil {
		return nil, p
	}
	now := time.Now()
	if err := w.reads.comments.UpdateCommentBody(w.db(), comment.ID, body, now); err != nil {
		return nil, problem.Internal(err)
	}
	w.scanComment(decision, matched, comment.ID, comment.UserID, body)
	return w.reads.commentOut(ctx, topic, comment.ID, user)
}

func (w *Writes) deleteComment(ctx context.Context, in *commentInput) (*struct{}, error) {
	if w == nil || w.reads == nil || w.db() == nil || w.state == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	topic, _, comment, user, p := w.reads.visibleComment(ctx, in.CommentID)
	if p != nil {
		return nil, p
	}
	canModerate := user.Can(perm.CommentTopicDelete)
	if !capsForComment(topic, comment.UserID, user).Delete {
		return nil, permissionRequired()
	}
	likes, err := w.reads.comments.CountCommentLikes(comment.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	penalty := commentDeleteCost
	if comment.UserID == user.ID && !canModerate {
		penalty = commentDeleteCost * (int(likes) + 1)
	}

	var awards []pendingAward
	err = w.db().Transaction(func(tx *gorm.DB) error {
		awards = awards[:0]
		// A moderator could not remove a comment whose author was too poor to
		// pay for it: the balance was a hard gate and the deletion rolled back.
		// The charge is best effort now, capped at the cached balance.
		state, err := w.state.LockForUpdate(tx, comment.UserID)
		if err != nil {
			return err
		}
		charge := min(penalty, state.Moemoepoint)
		if err := w.reads.comments.DeleteComment(tx, comment.ID); err != nil {
			return err
		}
		if err := service.RecomputeTopicCounts(tx, comment.TopicID); err != nil {
			return err
		}
		if charge > 0 {
			awards = append(awards, commentDeletedAward(comment.UserID, comment.ID, charge))
		}
		return nil
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	w.flushAwards(awards)
	return nil, nil
}

func (w *Writes) getCommentSource(ctx context.Context, in *commentInput) (*getCommentSourceOutput, error) {
	if w == nil || w.reads == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	topic, _, comment, user, p := w.reads.visibleComment(ctx, in.CommentID)
	if p != nil {
		return nil, p
	}
	if !capsForComment(topic, comment.UserID, user).Edit {
		return nil, permissionRequired()
	}
	return &getCommentSourceOutput{Body: CommentSource{
		Object:    "comment_source",
		CommentID: repr.ID(comment.ID),
		ReplyID:   repr.ID(comment.TopicReplyID),
		Text:      comment.Content,
	}}, nil
}
