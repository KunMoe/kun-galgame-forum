package apiv1

import (
	"context"
	"net/http"
	"unicode/utf8"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"github.com/danielgtaylor/huma/v2"
)

const excerptRunes = 200

func (s *Users) registerLists(api huma.API) {
	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "listUserTopics",
		Method:      http.MethodGet,
		Path:        "/users/{user_id}/topics",
		Summary:     "List a user's topics",
		Description: "Lists topics related to a user as a page-number collection, newest first with ties broken by descending id. " +
			"relation is required and closed. Hidden topics never appear except under relation=hidden, which is only the owner or a caller holding topic.view_hidden. " +
			"Restricted topics never appear, including for a caller who can view them elsewhere. " +
			"NOT_FOUND when the account does not exist or is not renderable.",
		Tags: []string{"users"},
		Responses: problemResponses(map[int]string{
			400: "UNKNOWN_ENUM_VALUE when relation is not in this collection's vocabulary. LIMIT_TOO_LARGE when limit is greater than 100. INVALID_PARAMETER when page × limit exceeds 10000.",
			403: "PERMISSION_REQUIRED when relation is hidden and the caller is neither the owner nor holding topic.view_hidden.",
			503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
		}),
	}), s.listUserTopics)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "listUserReplies",
		Method:      http.MethodGet,
		Path:        "/users/{user_id}/replies",
		Summary:     "List a user's replies",
		Description: "Lists replies related to a user as a page-number collection, newest first with ties broken by descending id. " +
			"relation is required and closed. Replies whose parent topic is hidden or restricted never appear. " +
			"NOT_FOUND when the account does not exist or is not renderable.",
		Tags: []string{"users"},
		Responses: problemResponses(map[int]string{
			400: "UNKNOWN_ENUM_VALUE when relation is not in this collection's vocabulary. LIMIT_TOO_LARGE when limit is greater than 100. INVALID_PARAMETER when page × limit exceeds 10000.",
			503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
		}),
	}), s.listUserReplies)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "listUserComments",
		Method:      http.MethodGet,
		Path:        "/users/{user_id}/comments",
		Summary:     "List a user's comments",
		Description: "Lists topic comments related to a user as a page-number collection, newest first with ties broken by descending id. " +
			"relation is required and closed. Comments whose parent topic is hidden or restricted never appear. " +
			"NOT_FOUND when the account does not exist or is not renderable.",
		Tags: []string{"users"},
		Responses: problemResponses(map[int]string{
			400: "UNKNOWN_ENUM_VALUE when relation is not in this collection's vocabulary. LIMIT_TOO_LARGE when limit is greater than 100. INVALID_PARAMETER when page × limit exceeds 10000.",
			503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
		}),
	}), s.listUserComments)

	s.registerWorkLists(api)
}

func (s *Users) readyLists() *problem.Problem {
	if s == nil || s.accounts == nil || s.content == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

func (s *Users) requireRenderableOwner(ctx context.Context, idStr string) (int, *problem.Problem) {
	id, ok := repr.ParseID(repr.DecimalID(idStr))
	if !ok {
		return 0, notFound()
	}
	u, found, err := s.accounts.User(ctx, id)
	if err != nil {
		return 0, unavailable(err)
	}
	if !found || !userclient.IsRenderable(u) {
		return 0, notFound()
	}
	return id, nil
}

func (s *Users) listUserTopics(ctx context.Context, in *listUserTopicsInput) (*listUserTopicsOutput, error) {
	ownerID, page, viewer, prob := s.prepareUserList(ctx, in.UserListPage)
	if prob != nil {
		return nil, prob
	}
	if in.Relation == "hidden" {
		if viewer == nil || (viewer.ID != ownerID && !viewer.Can(perm.TopicViewHidden)) {
			return nil, permissionRequired()
		}
	}
	rows, count, err := s.content.ListUserTopics(repository.UserListQuery{
		OwnerID: ownerID, Relation: in.Relation, IncludeNSFW: in.IncludeNSFW,
		Authenticated: viewer != nil, Offset: page.Offset(), Limit: page.Limit,
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	items := make([]UserTopicItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, UserTopicItem{
			Object:    "topic",
			ID:        repr.ID(row.ID),
			Title:     row.Title,
			CreatedAt: repr.Timestamp(row.Created),
		})
	}
	total, relation := collect.ClampTotal(count)
	return &listUserTopicsOutput{Body: repr.NewPageList(items, total, relation)}, nil
}

func (s *Users) listUserReplies(ctx context.Context, in *listUserRepliesInput) (*listUserRepliesOutput, error) {
	ownerID, page, viewer, prob := s.prepareUserList(ctx, in.UserListPage)
	if prob != nil {
		return nil, prob
	}
	rows, count, err := s.content.ListUserReplies(repository.UserListQuery{
		OwnerID: ownerID, Relation: in.Relation, IncludeNSFW: in.IncludeNSFW,
		Authenticated: viewer != nil, Offset: page.Offset(), Limit: page.Limit,
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	items := make([]UserReplyItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, UserReplyItem{
			Object:    "reply",
			ID:        repr.ID(row.ID),
			TopicID:   repr.ID(row.TopicID),
			Floor:     row.Floor,
			Excerpt:   excerptPlain(markdown.ToPlainText(row.Content, utf8.RuneCountInString(row.Content))),
			CreatedAt: repr.Timestamp(row.Created),
		})
	}
	total, relation := collect.ClampTotal(count)
	return &listUserRepliesOutput{Body: repr.NewPageList(items, total, relation)}, nil
}

func (s *Users) listUserComments(ctx context.Context, in *listUserCommentsInput) (*listUserCommentsOutput, error) {
	ownerID, page, viewer, prob := s.prepareUserList(ctx, in.UserListPage)
	if prob != nil {
		return nil, prob
	}
	rows, count, err := s.content.ListUserComments(repository.UserListQuery{
		OwnerID: ownerID, Relation: in.Relation, IncludeNSFW: in.IncludeNSFW,
		Authenticated: viewer != nil, Offset: page.Offset(), Limit: page.Limit,
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	items := make([]UserCommentItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, UserCommentItem{
			Object:    "comment",
			ID:        repr.ID(row.ID),
			TopicID:   repr.ID(row.TopicID),
			Excerpt:   excerptPlain(row.Content),
			CreatedAt: repr.Timestamp(row.Created),
		})
	}
	total, relation := collect.ClampTotal(count)
	return &listUserCommentsOutput{Body: repr.NewPageList(items, total, relation)}, nil
}

func (s *Users) prepareUserList(ctx context.Context, in UserListPage) (int, collect.PageNumber, *middleware.UserInfo, *problem.Problem) {
	if prob := s.readyLists(); prob != nil {
		return 0, collect.PageNumber{}, nil, prob
	}
	ownerID, prob := s.requireRenderableOwner(ctx, in.UserID)
	if prob != nil {
		return 0, collect.PageNumber{}, nil, prob
	}
	page := in.number()
	if prob := page.CheckDepth(); prob != nil {
		return 0, collect.PageNumber{}, nil, prob
	}
	return ownerID, page, v1.User(ctx), nil
}

func excerptPlain(s string) string {
	if utf8.RuneCountInString(s) <= excerptRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:excerptRunes-1]) + "…"
}
