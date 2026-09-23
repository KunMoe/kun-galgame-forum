package apiv1

import (
	"context"
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	galgameRepo "kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/user/repository"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

func (s *Users) registerWorkLists(api huma.API) {
	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "listUserWorks",
		Method:      http.MethodGet,
		Path:        "/users/{user_id}/works",
		Summary:     "List a user's works",
		Description: "Lists works related to a user as a page-number collection. " +
			"published and liked are newest first with ties broken by descending id. contributed keeps the catalog proposal order. " +
			"A work catalog no longer renders is omitted from the page while total still counts it. " +
			"NOT_FOUND when the account does not exist or is not renderable.",
		Tags: []string{"users"},
		Responses: problemResponses(map[int]string{
			400: "UNKNOWN_ENUM_VALUE when relation is not in this collection's vocabulary. LIMIT_TOO_LARGE when limit is greater than 100. INVALID_PARAMETER when page × limit exceeds 10000.",
			503: "SERVICE_UNAVAILABLE when the account service or catalog cannot be reached.",
		}),
	}), s.listUserWorks)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "listUserGalgameResources",
		Method:      http.MethodGet,
		Path:        "/users/{user_id}/galgame-resources",
		Summary:     "List a user's galgame resources",
		Description: "Lists download resources related to a user as a page-number collection, newest first with ties broken by descending id. " +
			"Items never include download links, extraction codes or archive passwords. " +
			"NOT_FOUND when the account does not exist or is not renderable.",
		Tags: []string{"users"},
		Responses: problemResponses(map[int]string{
			400: "UNKNOWN_ENUM_VALUE when relation or state is not in this collection's vocabulary. LIMIT_TOO_LARGE when limit is greater than 100. INVALID_PARAMETER when page × limit exceeds 10000.",
			503: "SERVICE_UNAVAILABLE when the account service or catalog cannot be reached.",
		}),
	}), s.listUserGalgameResources)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "listUserWallComments",
		Method:      http.MethodGet,
		Path:        "/users/{user_id}/wall-comments",
		Summary:     "List a user's wall comments",
		Description: "Lists wall comments related to a user as a cursor collection. " +
			"relation=authored is ordered by community post id, newest first; imported historical posts may have non-monotonic times. " +
			"relation=liked is ordered by when the user liked the comment, newest first. " +
			"A page may be shorter than limit while next_cursor is present. " +
			"NOT_FOUND when the account does not exist or is not renderable.",
		Tags: []string{"users"},
		Responses: problemResponses(map[int]string{
			400: "UNKNOWN_ENUM_VALUE when relation or subject_type is not in this collection's vocabulary. LIMIT_TOO_LARGE when limit is greater than 100. INVALID_CURSOR when the cursor is malformed or was issued for different filters.",
			503: "SERVICE_UNAVAILABLE when the account service or community service cannot be reached.",
		}),
	}), s.listUserWallComments)
}

func (s *Users) listUserWorks(ctx context.Context, in *listUserWorksInput) (*listUserWorksOutput, error) {
	if prob := s.readyWorks(); prob != nil {
		return nil, prob
	}
	ownerID, page, _, prob := s.prepareOwner(ctx, in.UserID, in.number())
	if prob != nil {
		return nil, prob
	}
	if in.Relation == "contributed" {
		return s.listContributedWorks(ctx, ownerID, page, in.IncludeNSFW)
	}
	ids, count, err := s.content.ListUserWorkIDs(repository.UserWorkQuery{
		OwnerID: ownerID, Relation: in.Relation, IncludeNSFW: in.IncludeNSFW,
		Offset: page.Offset(), Limit: page.Limit,
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	items, p := s.works.ByIDs(ctx, ids, in.IncludeNSFW)
	if p != nil {
		return nil, p
	}
	total, relation := collect.ClampTotal(count)
	return &listUserWorksOutput{Body: repr.NewPageList(items, total, relation)}, nil
}

// A contributed work often has no local row to carry content_limit, so a local
// NSFW filter counted works that catalog then refused to hydrate: total said 43
// while the SFW pages held 7. Catalog decides for the whole list, then it pages.
func (s *Users) listContributedWorks(ctx context.Context, ownerID int, page collect.PageNumber, includeNSFW bool) (*listUserWorksOutput, error) {
	if s.contributed == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	ids, err := s.contributed(ctx, int64(ownerID))
	if err != nil {
		return nil, unavailable(err)
	}
	all, p := s.works.ByIDs(ctx, ids, includeNSFW)
	if p != nil {
		return nil, p
	}
	total, relation := collect.ClampTotal(len(all))
	return &listUserWorksOutput{Body: repr.NewPageList(pageOf(all, page), total, relation)}, nil
}

func (s *Users) listUserGalgameResources(ctx context.Context, in *listUserGalgameResourcesInput) (*listUserGalgameResourcesOutput, error) {
	if prob := s.readyResources(); prob != nil {
		return nil, prob
	}
	ownerID, page, _, prob := s.prepareOwner(ctx, in.UserID, in.number())
	if prob != nil {
		return nil, prob
	}
	filter := galgameRepo.ResourceListFilter{
		IncludeNSFW: in.IncludeNSFW,
		State:       in.State,
	}
	switch in.Relation {
	case "published":
		filter.UploaderID = ownerID
	case "liked":
		filter.LikedBy = ownerID
	}
	body, p := s.resources.ListForUser(ctx, filter, page.Page, page.Limit)
	if p != nil {
		return nil, p
	}
	return &listUserGalgameResourcesOutput{Body: body}, nil
}

func (s *Users) readyWorks() *problem.Problem {
	if s == nil || s.accounts == nil || s.content == nil || s.works == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

func (s *Users) readyResources() *problem.Problem {
	if s == nil || s.accounts == nil || s.resources == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

func (s *Users) prepareOwner(ctx context.Context, userID string, page collect.PageNumber) (int, collect.PageNumber, *middleware.UserInfo, *problem.Problem) {
	ownerID, prob := s.requireRenderableOwner(ctx, userID)
	if prob != nil {
		return 0, collect.PageNumber{}, nil, prob
	}
	if prob := page.CheckDepth(); prob != nil {
		return 0, collect.PageNumber{}, nil, prob
	}
	return ownerID, page, v1.User(ctx), nil
}

func pageOf[T any](all []T, page collect.PageNumber) []T {
	start := min(page.Offset(), len(all))
	end := min(start+page.Limit, len(all))
	return all[start:end]
}
