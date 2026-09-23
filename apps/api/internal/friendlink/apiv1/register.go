package apiv1

import (
	"net/http"
	"strconv"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

func problemResponses(byStatus map[int]string) map[string]*huma.Response {
	out := make(map[string]*huma.Response, len(byStatus))
	for status, desc := range byStatus {
		out[strconv.Itoa(status)] = &huma.Response{
			Description: desc,
			Content: map[string]*huma.MediaType{
				problem.ContentType: {Schema: &huma.Schema{Ref: v1.ProblemRef}},
			},
		}
	}
	return out
}

func Register(s *Service) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listFriendLinks",
			Method:      http.MethodGet,
			Path:        "/friend-links",
			Summary:     "List friend links",
			Description: "Every friend link, shelf by shelf (official, galgame, others), each shelf in the order staff set with putFriendLinkOrder. " +
				"A cursor collection; the cursor is bound to friend_link_category.",
			Tags: []string{"friend-links"},
		}), s.listFriendLinks)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getFriendLink",
			Method:      http.MethodGet,
			Path:        "/admin/friend-links/{friend_link_id}",
			Summary:     "Get a friend link",
			Description: "One friend link, as the editor loads it. It needs the friend_link.edit permission, which a Bearer request never carries.",
			Tags:        []string{"friend-links"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks friend_link.edit.",
				404: "NOT_FOUND when the link does not exist.",
			}),
		}), s.getFriendLink)

		huma.Register(api, v1.IdempotencyOptional(v1.Required(huma.Operation{
			OperationID:   "createFriendLink",
			Method:        http.MethodPost,
			Path:          "/admin/friend-links",
			Summary:       "Create a friend link",
			DefaultStatus: http.StatusCreated,
			Description:   "Adds a link at the end of its shelf. It needs the friend_link.create permission.",
			Tags:          []string{"friend-links"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks friend_link.create.",
			}),
		})), s.createFriendLink)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "updateFriendLink",
			Method:      http.MethodPatch,
			Path:        "/admin/friend-links/{friend_link_id}",
			Summary:     "Update a friend link",
			Description: "Changes the fields sent. A link that changes shelf goes to the end of the new one. It needs the friend_link.edit permission.",
			Tags:        []string{"friend-links"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks friend_link.edit.",
				404: "NOT_FOUND when the link does not exist.",
			}),
		}), s.updateFriendLink)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID:   "deleteFriendLink",
			Method:        http.MethodDelete,
			Path:          "/admin/friend-links/{friend_link_id}",
			Summary:       "Delete a friend link",
			DefaultStatus: http.StatusNoContent,
			Description:   "Deletes the link for good. It needs the friend_link.delete permission.",
			Tags:          []string{"friend-links"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks friend_link.delete.",
				404: "NOT_FOUND when the link does not exist.",
			}),
		}), s.deleteFriendLink)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID:   "putFriendLinkOrder",
			Method:        http.MethodPut,
			Path:          "/admin/friend-link-order",
			Summary:       "Set a shelf's friend-link order",
			DefaultStatus: http.StatusNoContent,
			Description: "Replaces the display order of one shelf. The list must name every link on that shelf exactly once, " +
				"so a list made before someone else changed the shelf is refused rather than half applied. It needs the friend_link.edit permission.",
			Tags: []string{"friend-links"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks friend_link.edit.",
			}),
		}), s.putFriendLinkOrder)
	}
}
