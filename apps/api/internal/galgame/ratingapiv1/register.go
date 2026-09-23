package ratingapiv1

import (
	"net/http"
	"strconv"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

const (
	notFoundDesc   = "NOT_FOUND when no such rating exists, its author is banned, or catalog no longer shows its work."
	writeGate      = "ACCOUNT_BANNED when the caller's account is banned."
	unavailableMsg = "SERVICE_UNAVAILABLE when the account service or catalog cannot be reached."
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
		tags := []string{"galgame-ratings"}

		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "listRatings",
			Method:      http.MethodGet,
			Path:        "/ratings",
			Summary:     "List galgame ratings",
			Description: "A page-number collection: page × limit may not exceed 10000. " +
				"Ratings of adult works appear only with include_nsfw=true. " +
				"Ratings by banned authors and of works catalog no longer shows are left out of items but still counted in total, " +
				"so a page can hold fewer than limit items before the last page.",
			Tags: tags,
			Responses: problemResponses(map[int]string{
				400: "INVALID_PARAMETER when page × limit is too deep or an id is malformed; UNKNOWN_SORT or UNKNOWN_ENUM_VALUE for a value outside the vocabulary.",
				503: unavailableMsg,
			}),
		}), s.listRatings)

		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "getRating",
			Method:      http.MethodGet,
			Path:        "/ratings/{rating_id}",
			Summary:     "Get a galgame rating",
			Description: "Every read counts one view. A rating of an adult work is returned whatever the caller's content preference; " +
				"work.is_nsfw tells the client.",
			Tags: tags,
			Responses: problemResponses(map[int]string{
				404: notFoundDesc,
				503: unavailableMsg,
			}),
		}), s.getRating)

		huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
			OperationID:   "createRating",
			Method:        http.MethodPost,
			Path:          "/ratings",
			Summary:       "Rate a galgame",
			DefaultStatus: http.StatusCreated,
			Middlewares:   huma.Middlewares{withAccessToken},
			Description: "Writes the caller's rating of a work and returns it as getRating would. A user rates a work once; " +
				"change that rating with updateRating. work_id must name a work catalog shows (UNKNOWN_REFERENCE). " +
				"The author earns moemoepoints by the short summary's length, and the play status is copied to the caller's catalog work state.",
			Tags: tags,
			Responses: problemResponses(map[int]string{
				403: writeGate,
				409: "ALREADY_EXISTS when the caller has already rated the work; IDEMPOTENCY_KEY_REUSED or IDEMPOTENCY_REQUEST_IN_PROGRESS.",
				422: "VALIDATION_FAILED when a field is out of range or work_id names no work catalog shows; " +
					"CONTENT_REJECTED when the trust-and-safety check refuses the short summary.",
				503: unavailableMsg,
			}),
		})), s.createRating)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "updateRating",
			Method:      http.MethodPatch,
			Path:        "/ratings/{rating_id}",
			Summary:     "Update a galgame rating",
			Middlewares: huma.Middlewares{withAccessToken},
			Description: "Changes the fields sent and returns the rating as getRating would. Only the author may. " +
				"A changed short summary is checked again and moves the author's moemoepoints by the difference in reward; " +
				"a changed play status is copied to the caller's catalog work state.",
			Tags: tags,
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller is not the author; " + writeGate,
				404: notFoundDesc,
				422: "VALIDATION_FAILED when a field is out of range; CONTENT_REJECTED when the trust-and-safety check refuses the short summary.",
				503: unavailableMsg,
			}),
		}), s.updateRating)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID:   "deleteRating",
			Method:        http.MethodDelete,
			Path:          "/ratings/{rating_id}",
			Summary:       "Delete a galgame rating",
			DefaultStatus: http.StatusNoContent,
			Description: "Deletes the rating with its likes and comments; the author loses the moemoepoints it earned. " +
				"The author, the work page's creator and staff holding rating.delete_any may delete it; a Bearer request never holds staff powers.",
			Tags: tags,
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller may not delete the rating; " + writeGate,
				404: "NOT_FOUND when no such rating exists.",
				503: unavailableMsg,
			}),
		}), s.deleteRating)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "likeRating",
			Method:      http.MethodPut,
			Path:        "/ratings/{rating_id}/like",
			Summary:     "Like a galgame rating",
			Description: "Sets the caller's like. Liking an already liked rating changes nothing. The author earns 1 moemoepoint and is notified.",
			Tags:        tags,
			Responses: problemResponses(map[int]string{
				403: "SELF_LIKE_FORBIDDEN on the caller's own rating; " + writeGate,
				404: notFoundDesc,
				503: unavailableMsg,
			}),
		}), s.likeRating)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "unlikeRating",
			Method:      http.MethodDelete,
			Path:        "/ratings/{rating_id}/like",
			Summary:     "Remove a like from a galgame rating",
			Description: "Removes the caller's like. Removing a like that is not there changes nothing. The author loses the moemoepoint the like earned.",
			Tags:        tags,
			Responses: problemResponses(map[int]string{
				403: "SELF_LIKE_FORBIDDEN on the caller's own rating; " + writeGate,
				404: notFoundDesc,
				503: unavailableMsg,
			}),
		}), s.unlikeRating)
	}
}
