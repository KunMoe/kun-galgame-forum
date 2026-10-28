package apiv1

import (
	"context"
	"errors"
	"strconv"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/access"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

func notFound() *problem.Problem {
	return problem.New(problem.CodeNotFound, "Nothing visible exists at this URL.")
}

func parsePositiveID(s string) (int, bool) {
	return repr.ParseID(repr.DecimalID(s))
}

func (s *Service) visibleTopic(ctx context.Context, idStr string) (*model.Topic, *middleware.UserInfo, *problem.Problem) {
	if s == nil {
		return nil, nil, problem.Internal(errUnconfigured)
	}
	id, ok := parsePositiveID(idStr)
	if !ok {
		return nil, nil, notFound()
	}
	user := v1.User(ctx)
	topic, p := s.ReadableTopic(ctx, id, user)
	if p != nil {
		return nil, nil, p
	}
	return topic, user, nil
}

func (s *Service) ReadableTopic(ctx context.Context, id int, viewer *middleware.UserInfo) (*model.Topic, *problem.Problem) {
	topic, err := s.topics.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, notFound()
		}
		return nil, problem.Internal(err)
	}
	var grants []model.TopicAccessGrant
	if access.NeedsGrants(topic) {
		grants, err = s.topics.FindAccessGrants(topic.ID)
		if err != nil {
			return nil, problem.Internal(err)
		}
	}
	if !access.CanRead(topic, viewer, grants) {
		return nil, notFound()
	}
	if p := s.rejectUnrenderableAuthor(ctx, topic.UserID); p != nil {
		return nil, p
	}
	return topic, nil
}

func (s *Service) visibleReply(ctx context.Context, idStr string) (*model.Topic, *model.TopicReply, *middleware.UserInfo, *problem.Problem) {
	if s == nil {
		return nil, nil, nil, problem.Internal(errUnconfigured)
	}
	id, ok := parsePositiveID(idStr)
	if !ok {
		return nil, nil, nil, notFound()
	}
	row, err := s.replies.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, notFound()
		}
		return nil, nil, nil, problem.Internal(err)
	}
	if row.Status != 0 {
		return nil, nil, nil, notFound()
	}
	topic, user, p := s.visibleTopic(ctx, strconv.Itoa(row.TopicID))
	if p != nil {
		return nil, nil, nil, p
	}
	if p := s.rejectUnrenderableAuthor(ctx, row.UserID); p != nil {
		return nil, nil, nil, p
	}
	return topic, row, user, nil
}

func (s *Service) visibleComment(ctx context.Context, idStr string) (*model.Topic, *model.TopicReply, *model.TopicComment, *middleware.UserInfo, *problem.Problem) {
	if s == nil || s.comments == nil {
		return nil, nil, nil, nil, problem.Internal(errUnconfigured)
	}
	id, ok := parsePositiveID(idStr)
	if !ok {
		return nil, nil, nil, nil, notFound()
	}
	comment, err := s.comments.FindCommentByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, nil, notFound()
		}
		return nil, nil, nil, nil, problem.Internal(err)
	}
	if comment.Status != 0 {
		return nil, nil, nil, nil, notFound()
	}
	topic, reply, user, p := s.visibleReply(ctx, strconv.Itoa(comment.TopicReplyID))
	if p != nil {
		return nil, nil, nil, nil, p
	}
	if p := s.rejectUnrenderableAuthor(ctx, comment.UserID); p != nil {
		return nil, nil, nil, nil, p
	}
	return topic, reply, comment, user, nil
}

func (s *Service) rejectUnrenderableAuthor(ctx context.Context, userID int) *problem.Problem {
	users, p := s.lookupUsers(ctx, []int{userID})
	if p != nil {
		return p
	}
	if u, ok := users[userID]; ok && !userclient.IsRenderable(u) {
		return notFound()
	}
	return nil
}
