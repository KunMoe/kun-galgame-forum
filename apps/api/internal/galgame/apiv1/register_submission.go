package apiv1

import (
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"

	"github.com/danielgtaylor/huma/v2"
)

func (s *Service) registerSubmissions(api huma.API) {
	huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
		OperationID:   "createWorkSubmission",
		Method:        http.MethodPost,
		Path:          "/work-submissions",
		Summary:       "Submit a new work",
		DefaultStatus: http.StatusCreated,
		Description: "Mints a catalog work from the submitted names and claims it for this forum. The claim lands in pending, or straight in live for a submitter catalog trusts; state says which. " +
			"Idempotency-Key is required. When live works share a submitted title the mint is refused with DUPLICATE_SUSPECTS naming them and nothing is written; " +
			"re-send with is_duplicate_confirmed=true once the submitter has confirmed it is a different work. " +
			"A banner_hash becomes the work's cover after the mint; has_banner_attached=false means it did not, and the submission stands. " +
			"Location is the new submission's path.",
		Tags:        []string{"work-submissions"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "INVALID_PARAMETER when Idempotency-Key is missing or malformed.",
			403: "SCOPE_REQUIRED without catalog:edit; ACCOUNT_BANNED.",
			409: "DUPLICATE_SUSPECTS with suspects[]; ALREADY_EXISTS; IDEMPOTENCY_KEY_REUSED or IDEMPOTENCY_REQUEST_IN_PROGRESS.",
			422: "VALIDATION_FAILED when a title or display_name is blank or too long, a locale or original_language is not one of catalog's languages, titles and aliases exceed 100, an introduction locale repeats, or release_date is outside 1970–2200; CONTENT_REJECTED.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached or refuses for rate.",
		}),
	})), s.createWorkSubmission)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "getWorkSubmission",
		Method:      http.MethodGet,
		Path:        "/work-submissions/{work_id}",
		Summary:     "Get a work submission",
		Description: "The caller's own claim on the work. A cookie session holding galgame.claim.review may read anyone's. Someone else's claim and no claim are the same NOT_FOUND. " +
			"ETag is the claim's version, the If-Match for a write.",
		Tags:        []string{"work-submissions"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the caller may not see a claim on this work.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.getWorkSubmission)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "updateWorkSubmission",
		Method:      http.MethodPatch,
		Path:        "/work-submissions/{work_id}",
		Summary:     "Move a work submission",
		Description: "The submitter sends pending (from draft or declined) or draft (withdraw, from pending or live). " +
			"A cookie session holding galgame.claim.review sends live or declined (from pending), hidden (from any other state) or unban (from hidden); declined needs a note. " +
			"unban restores the state the claim was hidden from. The response is read back from catalog; its state is the outcome. " +
			"If-Match is forwarded; absent means no version check.",
		Tags:        []string{"work-submissions"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED for a reviewer state without galgame.claim.review, and always for a Bearer request; SCOPE_REQUIRED; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the caller may not see a claim on this work.",
			409: "INVALID_STATE_TRANSITION when the current state does not allow the target; detail names both.",
			412: "PRECONDITION_FAILED when If-Match is not the claim's current version.",
			422: "VALIDATION_FAILED when declined comes without a note.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.updateWorkSubmission)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID:   "deleteWorkSubmission",
		Method:        http.MethodDelete,
		Path:          "/work-submissions/{work_id}",
		Summary:       "Delete a draft submission",
		DefaultStatus: http.StatusNoContent,
		Description: "Deletes the caller's draft from catalog, then the forum's page for it. A page that already carries a resource is kept. " +
			"Only a draft can be deleted; withdraw a pending or live claim to draft first.",
		Tags:        []string{"work-submissions"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the claim is not the caller's.",
			409: "INVALID_STATE_TRANSITION when the claim is not a draft.",
			412: "PRECONDITION_FAILED when If-Match is not the claim's current version.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.deleteWorkSubmission)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listMyWorkSubmissions",
		Method:      http.MethodGet,
		Path:        "/me/work-submissions",
		Summary:     "List the caller's work submissions",
		Description: "Claims the caller submitted, newest activity first. state filters; absent lists every state.",
		Tags:        []string{"me"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "UNKNOWN_ENUM_VALUE for an unknown state; INVALID_CURSOR; LIMIT_TOO_LARGE.",
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.listMyWorkSubmissions)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listMyWorkSubmissionReviews",
		Method:      http.MethodGet,
		Path:        "/me/work-submission-reviews",
		Summary:     "List the submissions the caller reviewed",
		Description: "Claims the caller decided on and did not submit. Needs a cookie session holding galgame.claim.review.",
		Tags:        []string{"me"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "UNKNOWN_ENUM_VALUE for an unknown state; INVALID_CURSOR; LIMIT_TOO_LARGE.",
			403: "PERMISSION_REQUIRED without galgame.claim.review, and always for a Bearer request; SCOPE_REQUIRED; ACCOUNT_BANNED.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.listMyWorkSubmissionReviews)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listWorkSubmissions",
		Method:      http.MethodGet,
		Path:        "/work-submissions",
		Summary:     "List the review queue",
		Description: "Submissions awaiting or past review on this forum. state absent is pending; hidden lists the claims an unban can restore. " +
			"Queue rows carry no last_event. Needs a cookie session holding galgame.claim.review.",
		Tags:        []string{"work-submissions"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "UNKNOWN_ENUM_VALUE for an unknown state; INVALID_CURSOR; LIMIT_TOO_LARGE.",
			403: "PERMISSION_REQUIRED without galgame.claim.review, and always for a Bearer request; SCOPE_REQUIRED; ACCOUNT_BANNED.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.listWorkSubmissions)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listWorkSubmissionCandidates",
		Method:      http.MethodGet,
		Path:        "/work-submission-candidates",
		Summary:     "Search works before submitting one",
		Description: "Catalog works matching q that a submitter could publish resources on: unclaimed works, and works this forum claimed in live, draft or pending. " +
			"Works the forum displays as adult content are left out unless include_nsfw=true. " +
			"Filtering happens after catalog's page, so a page can be shorter than limit and there is no total.",
		Tags:        []string{"work-submissions"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "INVALID_PARAMETER when q is blank; INVALID_CURSOR; LIMIT_TOO_LARGE.",
			403: "ACCOUNT_BANNED.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.listWorkSubmissionCandidates)
}
