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

const wallNotFound = "NOT_FOUND when the comment does not exist, is on a page this forum does not host a wall for, " +
	"was written by a banned user, is held for review and the caller is not its author, " +
	"or is on a wall listWallComments would refuse the caller; these are indistinguishable."

const subjectNotFound = "NOT_FOUND when the page does not exist or this forum would not show it: " +
	"a galgame the catalog does not resolve, or a rating, resource, quiz or toolset whose owner is banned."

const quizGate = "QUIZ_ANSWER_REQUIRED when the wall belongs to a quiz that hides its game or carries spoilers " +
	"and the caller is neither its author, nor one who answered it, nor staff holding that wall's edit or delete permission. "

func Register(svc *Service) func(huma.API) {
	return func(api huma.API) {
		tags := []string{"wall-comments"}

		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "listWallComments",
			Method:      http.MethodGet,
			Path:        "/wall-comments",
			Summary:     "List a page's comment wall",
			Description: "Lists the comments on the wall of one page — a galgame, a galgame rating, resource or quiz, a toolset or a website — " +
				"as a cursor page in the order they were posted. Replies appear in that order too; group them by root_comment_id. " +
				"Comments by banned users and comments held for review that the caller did not write are left out without reading on, " +
				"so a page can hold fewer than limit items; continue while next_cursor is present. " +
				"A deleted comment stays as a tombstone so its replies keep their parent.",
			Tags: tags,
			Responses: problemResponses(map[int]string{
				400: "INVALID_PARAMETER when subject_type or subject_id is missing or subject_id is malformed; " +
					"UNKNOWN_ENUM_VALUE for an undeclared subject_type; LIMIT_TOO_LARGE; INVALID_CURSOR, including a cursor from another wall.",
				403: quizGate + "ACCOUNT_BANNED.",
				404: subjectNotFound,
			}),
		}), svc.listWallComments)

		huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
			OperationID:   "createWallComment",
			Method:        http.MethodPost,
			Path:          "/wall-comments",
			Summary:       "Comment on a page's wall",
			DefaultStatus: http.StatusCreated,
			Description: "Posts a comment on a page's wall and returns it as getWallComment would to its author; " +
				"the community service may hold it for review, and then state is held. " +
				"addressee is derived, not sent: the parent's author for a reply, the rating's author for a top-level comment on a rating wall. " +
				"The owner of a resource, toolset or quiz is notified of a top-level comment unless they already follow the wall. " +
				"On a galgame wall, users mentioned in the body are notified.",
			Tags: tags,
			Responses: problemResponses(map[int]string{
				403: quizGate + "SCOPE_REQUIRED or ACCOUNT_BANNED.",
				404: subjectNotFound,
				409: "INVALID_STATE_TRANSITION when the wall is closed; IDEMPOTENCY_KEY_REUSED or IDEMPOTENCY_REQUEST_IN_PROGRESS.",
				422: "VALIDATION_FAILED when the body is blank or over the wall's limit, mentions more than 20 users, " +
					"or parent_comment_id is not a visible comment on this wall; CONTENT_REJECTED when the word list refuses the body.",
				429: "RATE_LIMITED when the community service's new-account limit refuses the comment.",
			}),
		})), svc.createWallComment)

		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "getWallComment",
			Method:      http.MethodGet,
			Path:        "/wall-comments/{wall_comment_id}",
			Summary:     "Get a wall comment",
			Description: "Returns one wall comment, for a permalink; subject_type and subject_id say which page it is on. " +
				"A deleted comment comes back as its tombstone. " + wallNotFound,
			Tags: tags,
			Responses: problemResponses(map[int]string{
				403: quizGate + "ACCOUNT_BANNED.",
			}),
		}), svc.getWallComment)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getWallCommentSource",
			Method:      http.MethodGet,
			Path:        "/wall-comments/{wall_comment_id}/source",
			Summary:     "Get a wall comment's editable source",
			Description: "Returns the stored Markdown of the comment, to fill an edit form. It needs can_edit. " + wallNotFound,
			Tags:        tags,
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller may read but not edit the comment; " + quizGate + "SCOPE_REQUIRED or ACCOUNT_BANNED.",
			}),
		}), svc.getWallCommentSource)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "updateWallComment",
			Method:      http.MethodPatch,
			Path:        "/wall-comments/{wall_comment_id}",
			Summary:     "Update a wall comment",
			Description: "Replaces the comment body and returns the comment as getWallComment would. It needs can_edit. " +
				"A body equal to the stored one changes nothing. Staff editing someone else's comment sets is_edited_by_moderator. " +
				"On a galgame wall, users newly mentioned by the edit are notified. " + wallNotFound,
			Tags: tags,
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller may read but not edit the comment; " + quizGate + "SCOPE_REQUIRED or ACCOUNT_BANNED.",
				409: "INVALID_STATE_TRANSITION when the comment is a tombstone.",
				422: "VALIDATION_FAILED when the body is blank or over the wall's limit; CONTENT_REJECTED when the word list refuses the body.",
				429: "RATE_LIMITED when the community service's new-account limit refuses the edit.",
			}),
		}), svc.updateWallComment)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID:   "deleteWallComment",
			Method:        http.MethodDelete,
			Path:          "/wall-comments/{wall_comment_id}",
			Summary:       "Delete a wall comment",
			DefaultStatus: http.StatusNoContent,
			Description: "Turns the comment into a tombstone; its replies stay. It needs can_delete. " +
				"Deleting a tombstone again succeeds and changes nothing. " + wallNotFound,
			Tags: tags,
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller may read but not delete the comment; " + quizGate + "SCOPE_REQUIRED or ACCOUNT_BANNED.",
			}),
		}), svc.deleteWallComment)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "likeWallComment",
			Method:      http.MethodPut,
			Path:        "/wall-comments/{wall_comment_id}/like",
			Summary:     "Like a wall comment",
			Description: "Sets the caller's like and returns the comment. Liking an already liked comment changes nothing. " +
				"The author earns 1 moemoepoint. " + wallNotFound,
			Tags: tags,
			Responses: problemResponses(map[int]string{
				403: "SELF_LIKE_FORBIDDEN on the caller's own comment; " + quizGate + "SCOPE_REQUIRED or ACCOUNT_BANNED.",
				409: "INVALID_STATE_TRANSITION when the comment is a tombstone.",
			}),
		}), svc.likeWallComment)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "unlikeWallComment",
			Method:      http.MethodDelete,
			Path:        "/wall-comments/{wall_comment_id}/like",
			Summary:     "Remove a like from a wall comment",
			Description: "Removes the caller's like and returns the comment. Removing a like that is not there changes nothing. " +
				"The author loses the moemoepoint the like earned. " + wallNotFound,
			Tags: tags,
			Responses: problemResponses(map[int]string{
				403: "SELF_LIKE_FORBIDDEN on the caller's own comment; " + quizGate + "SCOPE_REQUIRED or ACCOUNT_BANNED.",
				409: "INVALID_STATE_TRANSITION when the comment is a tombstone.",
			}),
		}), svc.unlikeWallComment)

		huma.Register(api, v1.IdempotencyOptional(v1.Required(huma.Operation{
			OperationID:   "flagWallComment",
			Method:        http.MethodPost,
			Path:          "/wall-comments/{wall_comment_id}/flags",
			Summary:       "Flag a wall comment",
			DefaultStatus: http.StatusNoContent,
			Description: "Reports the comment to the moderators. A flag has no id and cannot be read back; " +
				"flagging the same comment again is accepted and counts once. " + wallNotFound,
			Tags: tags,
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED on the caller's own comment; " + quizGate + "SCOPE_REQUIRED or ACCOUNT_BANNED.",
				409: "INVALID_STATE_TRANSITION when the comment is a tombstone; IDEMPOTENCY_KEY_REUSED or IDEMPOTENCY_REQUEST_IN_PROGRESS.",
			}),
		})), svc.flagWallComment)
	}
}
