package apiv1

import (
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"

	"github.com/danielgtaylor/huma/v2"
)

func (s *Service) registerEdit(api huma.API) {
	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "getWorkEditForm",
		Method:      http.MethodGet,
		Path:        "/works/{work_id}/edit-form",
		Summary:     "Get a work's edit form",
		Description: "Current field values, the editable-field schema and the closed vocabularies the fields name. " +
			"The schema is the same for every caller: whether a proposal can be decided is on the proposal. " +
			"A vocabulary read failure answers vocabularies=[] with a warning, still 200. Hidden or unknown works are NOT_FOUND.",
		Tags:        []string{"works"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			403: "SCOPE_REQUIRED without catalog:edit; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the work does not exist or is hidden; ENTITY_MERGED when it was merged, with current_id.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.getWorkEditForm)

	huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
		OperationID:   "createWorkEditProposal",
		Method:        http.MethodPost,
		Path:          "/works/{work_id}/edit-proposals",
		Summary:       "Propose an edit to a work",
		DefaultStatus: http.StatusCreated,
		Description: "Files a proposal. Idempotency-Key is required. Every patch key starts with catalog.work. " +
			"A proposal catalog merges at once answers state=merged with its revision; revision is null when it did not merge, " +
			"and also when the merged revision could not be read back, in which case the proposal still stands. " +
			"Location is the proposal's path.",
		Tags:        []string{"works"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "INVALID_PARAMETER when Idempotency-Key is missing or malformed.",
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the work does not exist or is hidden; ENTITY_MERGED when it was merged.",
			409: "IDEMPOTENCY_KEY_REUSED or IDEMPOTENCY_REQUEST_IN_PROGRESS.",
			422: "VALIDATION_FAILED when patch is empty, a key lacks the catalog.work. prefix, or catalog refuses a value.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached, or its quota is used up.",
		}),
	})), s.createWorkEditProposal)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "listWorkEditProposals",
		Method:      http.MethodGet,
		Path:        "/works/{work_id}/edit-proposals",
		Summary:     "List a work's edit proposals",
		Description: "A cursor collection of the proposals filed on this site for the work, newest first. state defaults to open. " +
			"Items carry no patch: the patch is on GET /edit-proposals/{proposal_id} for those who may read it. " +
			"viewer is null for an anonymous caller. Hidden or unknown works are NOT_FOUND.",
		Tags:        []string{"works"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "UNKNOWN_ENUM_VALUE, LIMIT_TOO_LARGE or INVALID_CURSOR.",
			401: "INVALID_CREDENTIAL for a bad Bearer.",
			404: "NOT_FOUND when the work does not exist or is hidden; ENTITY_MERGED when it was merged.",
			503: "SERVICE_UNAVAILABLE when the catalog or the account service cannot be reached.",
		}),
	}), s.listWorkEditProposals)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "listWorkEditRevisions",
		Method:      http.MethodGet,
		Path:        "/works/{work_id}/edit-revisions",
		Summary:     "List a work's edit revisions",
		Description: "A page-number collection of the work's revision chain, newest first. Hidden or unknown works are NOT_FOUND.",
		Tags:        []string{"works"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "LIMIT_TOO_LARGE or INVALID_PARAMETER when page × limit is too deep.",
			401: "INVALID_CREDENTIAL for a bad Bearer.",
			404: "NOT_FOUND when the work does not exist or is hidden; ENTITY_MERGED when it was merged.",
			503: "SERVICE_UNAVAILABLE when the catalog or the account service cannot be reached.",
		}),
	}), s.listWorkEditRevisions)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "getWorkEditRevisionDiff",
		Method:      http.MethodGet,
		Path:        "/works/{work_id}/edit-revisions/diff",
		Summary:     "Diff two revisions of a work",
		Description: "The field-level changes from from_seq to to_seq. Both are required and at least 1. A seq the work does not have is NOT_FOUND.",
		Tags:        []string{"works"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "INVALID_PARAMETER when from_seq or to_seq is missing or below 1.",
			401: "INVALID_CREDENTIAL for a bad Bearer.",
			404: "NOT_FOUND when the work does not exist, is hidden, or lacks either seq; ENTITY_MERGED when it was merged.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.getWorkEditRevisionDiff)

	huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
		OperationID:   "createWorkEditRevert",
		Method:        http.MethodPost,
		Path:          "/works/{work_id}/edit-reverts",
		Summary:       "Revert a work to a revision",
		DefaultStatus: http.StatusCreated,
		Description: "Files a proposal that restores the work to to_seq. Idempotency-Key is required. " +
			"When catalog merges it at once, revision is the new revision; otherwise revision is null and the proposal waits for review. " +
			"Location is the proposal's path.",
		Tags:        []string{"works"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "INVALID_PARAMETER when Idempotency-Key is missing or malformed.",
			403: "SCOPE_REQUIRED; PERMISSION_REQUIRED when catalog refuses the revert; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the work does not exist, is hidden, or has no revision to_seq; ENTITY_MERGED when it was merged.",
			409: "IDEMPOTENCY_KEY_REUSED or IDEMPOTENCY_REQUEST_IN_PROGRESS.",
			422: "VALIDATION_FAILED when to_seq is below 1.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached, or its quota is used up.",
		}),
	})), s.createWorkEditRevert)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listMyEditProposals",
		Method:      http.MethodGet,
		Path:        "/me/edit-proposals",
		Summary:     "List the caller's edit proposals",
		Description: "A cursor collection of the proposals the caller filed on this site, newest first. work_id narrows it to one work; state to one state.",
		Tags:        []string{"me"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "UNKNOWN_ENUM_VALUE, LIMIT_TOO_LARGE or INVALID_CURSOR.",
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			503: "SERVICE_UNAVAILABLE when the catalog or the account service cannot be reached.",
		}),
	}), s.listMyEditProposals)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listEditProposals",
		Method:      http.MethodGet,
		Path:        "/edit-proposals",
		Summary:     "List the edit review queue",
		Description: "A cursor collection of this site's proposals, newest first. state defaults to open. " +
			"Requires a cookie session holding galgame.edit_proposal.review; requests authenticated with a Bearer token never hold staff powers.",
		Tags:        []string{"edit-proposals"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "UNKNOWN_ENUM_VALUE, LIMIT_TOO_LARGE or INVALID_CURSOR.",
			403: "PERMISSION_REQUIRED; SCOPE_REQUIRED; ACCOUNT_BANNED.",
			503: "SERVICE_UNAVAILABLE when the catalog or the account service cannot be reached.",
		}),
	}), s.listEditProposals)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "getEditProposal",
		Method:      http.MethodGet,
		Path:        "/edit-proposals/{proposal_id}",
		Summary:     "Get an edit proposal",
		Description: "The proposal with its patch and amendments. The proposer, the work's owner and reviewers may read it; anyone else is NOT_FOUND. " +
			"ETag is catalog's validator; send it back as If-Match to amend, withdraw or decide.",
		Tags:        []string{"edit-proposals"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the proposal does not exist, belongs to another site, or the caller may not read it.",
			503: "SERVICE_UNAVAILABLE when the catalog or the account service cannot be reached.",
		}),
	}), s.getEditProposal)

	huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
		OperationID:   "createEditProposalAmendment",
		Method:        http.MethodPost,
		Path:          "/edit-proposals/{proposal_id}/amendments",
		Summary:       "Amend an edit proposal",
		DefaultStatus: http.StatusCreated,
		Description: "Appends a correction to an open proposal. Idempotency-Key is required. set or unset must name at least one key. " +
			"If-Match is forwarded to catalog; without it the write is unconditional. Location is the proposal's path; ETag is its new validator.",
		Tags:        []string{"edit-proposals"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "INVALID_PARAMETER when Idempotency-Key is missing or malformed.",
			403: "PERMISSION_REQUIRED when catalog refuses the caller; SCOPE_REQUIRED; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the proposal does not exist or belongs to another site.",
			409: "INVALID_STATE_TRANSITION when the proposal is not open; IDEMPOTENCY_KEY_REUSED or IDEMPOTENCY_REQUEST_IN_PROGRESS.",
			412: "PRECONDITION_FAILED when If-Match does not match.",
			422: "VALIDATION_FAILED when set and unset are both empty, a key lacks the catalog.work. prefix, or catalog refuses a value.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached, or its quota is used up.",
		}),
	})), s.createEditProposalAmendment)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "updateEditProposal",
		Method:      http.MethodPatch,
		Path:        "/edit-proposals/{proposal_id}",
		Summary:     "Merge, decline or withdraw an edit proposal",
		Description: "Moves an open proposal. withdrawn is the proposer's; merged and declined need a cookie session holding galgame.edit_proposal.review, or owning the work, and catalog decides. " +
			"declined requires a note. The response is the proposal read back from catalog. If-Match is forwarded; without it the write is unconditional.",
		Tags:        []string{"edit-proposals"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED for a Bearer decision, a caller without review standing, or someone else's withdrawal; SCOPE_REQUIRED; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the proposal does not exist, belongs to another site, or the caller may not read it.",
			409: "INVALID_STATE_TRANSITION when the proposal is not open.",
			412: "PRECONDITION_FAILED when If-Match does not match.",
			422: "VALIDATION_FAILED when declining without a note.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached, or its quota is used up.",
		}),
	}), s.updateEditProposal)
}
