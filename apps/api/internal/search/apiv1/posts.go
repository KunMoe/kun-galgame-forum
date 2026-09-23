package apiv1

import (
	"context"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/search/repository"
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

type topicsOutput struct {
	Body repr.PageList[topicapiv1.TopicSummary]
}

type repliesOutput struct {
	Body repr.PageList[ReplySearchHit]
}

type commentsOutput struct {
	Body repr.PageList[CommentSearchHit]
}

func (s *Service) laneQuery(ctx context.Context, in *laneInput) (repository.V1Query, *problem.Problem) {
	if prob := s.ready(); prob != nil {
		return repository.V1Query{}, prob
	}
	keywords, prob := keywordsOf(in.Q)
	if prob != nil {
		return repository.V1Query{}, prob
	}
	if prob := in.CheckDepth(); prob != nil {
		return repository.V1Query{}, prob
	}
	return repository.V1Query{
		Keywords: keywords, Offset: in.Offset(), Limit: in.Limit,
		Authenticated: authenticated(ctx), IncludeNSFW: in.IncludeNSFW,
	}, nil
}

func (s *Service) searchTopics(ctx context.Context, in *laneInput) (*topicsOutput, error) {
	q, prob := s.laneQuery(ctx, in)
	if prob != nil {
		return nil, prob
	}
	if s.topics == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	rows, count, err := s.repo.SearchTopicRowsV1(q)
	if err != nil {
		return nil, problem.Internal(err)
	}
	items, prob := s.topics.Summaries(ctx, rows)
	if prob != nil {
		return nil, prob
	}
	total, relation := collect.ClampTotal(int(count))
	return &topicsOutput{Body: repr.NewPageList(items, total, relation)}, nil
}

func (s *Service) renderablePosts(ctx context.Context, rows []repository.PostRow) ([]repository.PostRow, map[int]userclient.User, *problem.Problem) {
	ids := make([]int, 0, len(rows)*2)
	for _, r := range rows {
		ids = append(ids, r.UserID, r.TopicUserID)
	}
	users, prob := s.lookupUsers(ctx, ids)
	if prob != nil {
		return nil, nil, prob
	}
	kept := make([]repository.PostRow, 0, len(rows))
	for _, r := range rows {
		author, ok := users[r.UserID]
		if ok && !userclient.IsRenderable(author) {
			continue
		}
		if owner, ok := users[r.TopicUserID]; ok && !userclient.IsRenderable(owner) {
			continue
		}
		kept = append(kept, r)
	}
	return kept, users, nil
}

func (s *Service) authorRef(users map[int]userclient.User, id int) repr.UserRef {
	if u, ok := users[id]; ok {
		return repr.NewUserRef(s.cdn, u)
	}
	return repr.DeletedUserRef(id)
}

func (s *Service) searchReplies(ctx context.Context, in *laneInput) (*repliesOutput, error) {
	q, prob := s.laneQuery(ctx, in)
	if prob != nil {
		return nil, prob
	}
	rows, count, err := s.repo.SearchReplyRowsV1(q)
	if err != nil {
		return nil, problem.Internal(err)
	}
	kept, users, prob := s.renderablePosts(ctx, rows)
	if prob != nil {
		return nil, prob
	}
	items := make([]ReplySearchHit, 0, len(kept))
	for _, r := range kept {
		items = append(items, ReplySearchHit{
			Object: "reply", ID: repr.ID(r.ID), TopicID: repr.ID(r.TopicID), TopicTitle: r.TopicTitle,
			Floor: max(r.Floor, 1), Excerpt: excerpt(r.Content, q.Keywords),
			Author: s.authorRef(users, r.UserID), CreatedAt: repr.Timestamp(r.Created),
		})
	}
	total, relation := collect.ClampTotal(int(count))
	return &repliesOutput{Body: repr.NewPageList(items, total, relation)}, nil
}

func (s *Service) searchComments(ctx context.Context, in *laneInput) (*commentsOutput, error) {
	q, prob := s.laneQuery(ctx, in)
	if prob != nil {
		return nil, prob
	}
	rows, count, err := s.repo.SearchCommentRowsV1(q)
	if err != nil {
		return nil, problem.Internal(err)
	}
	kept, users, prob := s.renderablePosts(ctx, rows)
	if prob != nil {
		return nil, prob
	}
	items := make([]CommentSearchHit, 0, len(kept))
	for _, r := range kept {
		items = append(items, CommentSearchHit{
			Object: "comment", ID: repr.ID(r.ID), TopicID: repr.ID(r.TopicID), TopicTitle: r.TopicTitle,
			Excerpt: excerpt(r.Content, q.Keywords),
			Author:  s.authorRef(users, r.UserID), CreatedAt: repr.Timestamp(r.Created),
		})
	}
	total, relation := collect.ClampTotal(int(count))
	return &commentsOutput{Body: repr.NewPageList(items, total, relation)}, nil
}
