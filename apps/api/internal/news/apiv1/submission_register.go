package apiv1

import (
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"

	"github.com/danielgtaylor/huma/v2"
)

func (s *Service) registerSubmissions(api huma.API) {
	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listMyNewsSubmissions",
		Method:      http.MethodGet,
		Path:        "/me/news-submissions",
		Summary:     "List the caller's news submissions",
		Description: "The caller's own submissions, pending included, newest first. A cursor collection over the catalog cursor; the cursor is bound to limit. next_cursor is omitted on the last page.",
		Tags:        []string{"me"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			http.StatusBadRequest:         "INVALID_CURSOR when the cursor is malformed or was issued for a different limit.",
			http.StatusForbidden:          "SCOPE_REQUIRED when the catalog token lacks the scope; ACCOUNT_BANNED when the caller is banned.",
			http.StatusTooManyRequests:    "QUOTA_EXCEEDED when the daily submission quota is exhausted; RATE_LIMITED otherwise. Retry-After is passed through when the catalog sends it.",
			http.StatusServiceUnavailable: "SERVICE_UNAVAILABLE when the catalog is not configured or cannot be reached.",
		}),
	}), s.listMyNewsSubmissions)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "getMyNewsSubmission",
		Method:      http.MethodGet,
		Path:        "/me/news-submissions/{news_submission_id}",
		Summary:     "Get one news submission",
		Description: "One submission belonging to the caller. Another account's item is NOT_FOUND.",
		Tags:        []string{"me"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			http.StatusForbidden:          "SCOPE_REQUIRED when the catalog token lacks the scope; ACCOUNT_BANNED when the caller is banned.",
			http.StatusNotFound:           "NOT_FOUND when the submission is not the caller's.",
			http.StatusTooManyRequests:    "QUOTA_EXCEEDED when the daily submission quota is exhausted; RATE_LIMITED otherwise. Retry-After is passed through when the catalog sends it.",
			http.StatusServiceUnavailable: "SERVICE_UNAVAILABLE when the catalog is not configured or cannot be reached.",
		}),
	}), s.getMyNewsSubmission)

	huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
		OperationID:   "createNewsSubmission",
		Method:        http.MethodPost,
		Path:          "/me/news-submissions",
		Summary:       "Submit a news item",
		DefaultStatus: http.StatusCreated,
		Description:   "Submits a community news item. It stays pending until a NextMoe moderator publishes it. Send content_markdown or source_url. Idempotency-Key is required. Location is the new submission's path.",
		Tags:          []string{"me"},
		Middlewares:   huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			http.StatusBadRequest:          "INVALID_PARAMETER when Idempotency-Key is missing or malformed.",
			http.StatusForbidden:           "SCOPE_REQUIRED when the catalog token lacks the scope; ACCOUNT_BANNED when the caller is banned.",
			http.StatusConflict:            "IDEMPOTENCY_KEY_REUSED or IDEMPOTENCY_REQUEST_IN_PROGRESS.",
			http.StatusUnprocessableEntity: "CONTENT_REJECTED when the trust check refuses the text; VALIDATION_FAILED when title or preview is blank, or neither content_markdown nor source_url is sent.",
			http.StatusTooManyRequests:     "QUOTA_EXCEEDED when the daily submission quota is exhausted; RATE_LIMITED otherwise. Retry-After is passed through when the catalog sends it.",
			http.StatusServiceUnavailable:  "SERVICE_UNAVAILABLE when the catalog is not configured or cannot be reached.",
		}),
	})), s.createNewsSubmission)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "updateNewsSubmission",
		Method:      http.MethodPatch,
		Path:        "/me/news-submissions/{news_submission_id}",
		Summary:     "Update a news submission",
		Description: "Partial update of the caller's submission. Editing a published item returns it to pending until a NextMoe moderator publishes it again. Only a published item can be withdrawn. Withdrawal is state withdrawn sent on its own.",
		Tags:        []string{"me"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			http.StatusForbidden:           "SCOPE_REQUIRED when the catalog token lacks the scope; ACCOUNT_BANNED when the caller is banned.",
			http.StatusNotFound:            "NOT_FOUND when the submission is not the caller's.",
			http.StatusConflict:            "INVALID_STATE_TRANSITION when the withdrawal or edit is not legal from the current state.",
			http.StatusUnprocessableEntity: "CONTENT_REJECTED when the trust check refuses the text; VALIDATION_FAILED when the body is empty, state is sent with another field, a sent title or preview is blank, or the item would have neither content_markdown nor source_url.",
			http.StatusTooManyRequests:     "QUOTA_EXCEEDED when the daily submission quota is exhausted; RATE_LIMITED otherwise. Retry-After is passed through when the catalog sends it.",
			http.StatusServiceUnavailable:  "SERVICE_UNAVAILABLE when the catalog is not configured or cannot be reached.",
		}),
	}), s.updateNewsSubmission)
}
