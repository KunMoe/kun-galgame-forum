package apiv1

import (
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"

	"github.com/danielgtaylor/huma/v2"
)

func (s *Service) registerUserPlane(api huma.API) {
	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "getWorkCover",
		Method:      http.MethodGet,
		Path:        "/works/{work_id}/covers/{cover_id}",
		Summary:     "Get a work cover",
		Description: "Returns one cover of the work. The cover must belong to the work. " +
			"A failure to read vote tallies answers vote_count=0 and has_voted=false with a warning, still 200.",
		Tags:        []string{"works"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			401: "INVALID_CREDENTIAL for a bad Bearer.",
			404: "NOT_FOUND when the work or cover does not exist or is hidden, or the cover does not belong to the work; ENTITY_MERGED when the work was merged.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.getWorkCover)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "putWorkCoverVote",
		Method:      http.MethodPut,
		Path:        "/works/{work_id}/covers/{cover_id}/vote",
		Summary:     "Vote for a work cover",
		Description: "Sets the caller's vote on this cover. Voting again changes nothing. One ballot per work: a vote on another cover of the same work moves the ballot.",
		Tags:        []string{"works"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			403: "SCOPE_REQUIRED without catalog:edit; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the work or cover does not exist or is hidden, or the cover does not belong to the work.",
			503: "SERVICE_UNAVAILABLE when the catalog or the vote store cannot be reached.",
		}),
	}), s.putWorkCoverVote)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "deleteWorkCoverVote",
		Method:      http.MethodDelete,
		Path:        "/works/{work_id}/covers/{cover_id}/vote",
		Summary:     "Withdraw a work cover vote",
		Description: "Clears the caller's vote on this cover. Withdrawing when not voted changes nothing.",
		Tags:        []string{"works"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			403: "SCOPE_REQUIRED without catalog:edit; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the work or cover does not exist or is hidden, or the cover does not belong to the work.",
			503: "SERVICE_UNAVAILABLE when the catalog or the vote store cannot be reached.",
		}),
	}), s.deleteWorkCoverVote)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "putWorkPlaytime",
		Method:      http.MethodPut,
		Path:        "/works/{work_id}/playtime",
		Summary:     "Set the caller's playtime on a work",
		Description: "Writes minutes and/or play_state. At least one field is required. play_state done is refused. minutes 0 withdraws the duration without deleting the row.",
		Tags:        []string{"works"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the work does not exist or is hidden.",
			422: "VALIDATION_FAILED when both fields are omitted, minutes is outside 0–60000, or play_state is unknown or done.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.putWorkPlaytime)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "deleteWorkPlaytime",
		Method:      http.MethodDelete,
		Path:        "/works/{work_id}/playtime",
		Summary:     "Withdraw the caller's playtime on a work",
		Description: "Deletes the playtime this site reported for the caller on this work and clears the work-state. Repeating the delete is 200. The response is read back from catalog, so minutes another app reported can remain.",
		Tags:        []string{"works"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the work does not exist or is hidden.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.deleteWorkPlaytime)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listMyPlaytimes",
		Method:      http.MethodGet,
		Path:        "/me/playtimes",
		Summary:     "List the caller's playtimes",
		Description: "A page-number collection of the caller's playtimes. Totals use the same predicate as items, including include_nsfw.",
		Tags:        []string{"me"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "LIMIT_TOO_LARGE or INVALID_PARAMETER when page × limit is too deep.",
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.listMyPlaytimes)

	huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
		OperationID:   "createCollection",
		Method:        http.MethodPost,
		Path:          "/collections",
		Summary:       "Create a collection",
		DefaultStatus: http.StatusCreated,
		Description:   "Creates a collection and returns it. Idempotency-Key is required. visibility must be sent. Location is the new collection's path.",
		Tags:          []string{"collections"},
		Middlewares:   huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "INVALID_PARAMETER when Idempotency-Key is missing or malformed.",
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			409: "IDEMPOTENCY_KEY_REUSED or IDEMPOTENCY_REQUEST_IN_PROGRESS.",
			422: "CONTENT_REJECTED; VALIDATION_FAILED when title is blank, visibility is missing or unknown, or the account already holds as many collections as catalog allows.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	})), s.createCollection)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "getCollection",
		Method:      http.MethodGet,
		Path:        "/collections/{collection_id}",
		Summary:     "Get a collection",
		Description: "Returns collection metadata without its works. A private collection that is not the caller's, an unknown id, or an unrenderable owner who is not the caller is NOT_FOUND.",
		Tags:        []string{"collections"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			401: "INVALID_CREDENTIAL for a bad Bearer.",
			404: "NOT_FOUND when the collection is not visible.",
			503: "SERVICE_UNAVAILABLE when the catalog or the account service cannot be reached.",
		}),
	}), s.getCollection)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "updateCollection",
		Method:      http.MethodPatch,
		Path:        "/collections/{collection_id}",
		Summary:     "Update a collection",
		Description: "Partial update. An empty body is VALIDATION_FAILED. Staff edits of someone else's collection require a cookie session holding collection.edit_any and go through catalog moderation. Bearer requests never hold staff powers.",
		Tags:        []string{"collections"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the collection is not the caller's and the caller holds no staff power.",
			422: "CONTENT_REJECTED; VALIDATION_FAILED when the body is empty, title is blank, or is_default is false.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.updateCollection)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID:   "deleteCollection",
		Method:        http.MethodDelete,
		Path:          "/collections/{collection_id}",
		Summary:       "Delete a collection",
		DefaultStatus: http.StatusNoContent,
		Description:   "Deletes the collection. The owner cannot delete the default collection. Staff deletion of someone else's collection requires a cookie session holding collection.delete_any.",
		Tags:          []string{"collections"},
		Middlewares:   huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the collection is not the caller's and the caller holds no staff power.",
			409: "INVALID_STATE_TRANSITION when the owner deletes the default collection.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.deleteCollection)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "listCollectionWorks",
		Method:      http.MethodGet,
		Path:        "/collections/{collection_id}/works",
		Summary:     "List a collection's works",
		Description: "A page-number collection of works in the collection, newest added first. NSFW works are excluded from items and total unless include_nsfw=true. An id catalog does not render is dropped with a warning; total still counts the filtered population.",
		Tags:        []string{"collections"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "LIMIT_TOO_LARGE or INVALID_PARAMETER.",
			404: "NOT_FOUND when the collection is not visible.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot hydrate the page.",
		}),
	}), s.listCollectionWorks)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "getCollectionWork",
		Method:      http.MethodGet,
		Path:        "/collections/{collection_id}/works/{work_id}",
		Summary:     "Get a work in a collection",
		Description: "Returns the work when it is a member of the collection. Otherwise NOT_FOUND. An adult work with include_nsfw=false is NOT_FOUND.",
		Tags:        []string{"collections"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			404: "NOT_FOUND when the collection is not visible or the work is not a member.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot hydrate the work.",
		}),
	}), s.getCollectionWork)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "putCollectionWork",
		Method:      http.MethodPut,
		Path:        "/collections/{collection_id}/works/{work_id}",
		Summary:     "Add a work to a collection",
		Description: "Adds the work to the caller's collection. Adding again changes nothing. The collection must belong to the caller.",
		Tags:        []string{"collections"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the collection is not the caller's or the work does not exist.",
			422: "VALIDATION_FAILED when the collection is full.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.putCollectionWork)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "deleteCollectionWork",
		Method:      http.MethodDelete,
		Path:        "/collections/{collection_id}/works/{work_id}",
		Summary:     "Remove a work from a collection",
		Description: "Removes the work from the caller's collection. Removing when not a member changes nothing.",
		Tags:        []string{"collections"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the collection is not the caller's or the work does not exist.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.deleteCollectionWork)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listMyCollections",
		Method:      http.MethodGet,
		Path:        "/me/collections",
		Summary:     "List the caller's collections",
		Description: "A page-number collection of every collection the caller owns, including private ones. GET never creates a default collection and never mints an alias. work_id paints viewer.has_work.",
		Tags:        []string{"me"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "LIMIT_TOO_LARGE or INVALID_PARAMETER.",
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.listMyCollections)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listMyCollectionsForWork",
		Method:      http.MethodGet,
		Path:        "/me/works/{work_id}/collections",
		Summary:     "List the caller's collections against one work",
		Description: "A page-number collection of every collection the caller owns, including private ones, each with viewer.has_work for this work. The shape the collection picker needs: no preview covers, no owner, no description. GET never creates a default collection.",
		Tags:        []string{"me"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "LIMIT_TOO_LARGE or INVALID_PARAMETER.",
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the work does not exist or is hidden.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.listMyCollectionsForWork)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "listUserCollections",
		Method:      http.MethodGet,
		Path:        "/users/{user_id}/collections",
		Summary:     "List a user's collections",
		Description: "A page-number collection of a user's collections. Callers who are not the owner see only public collections. An unrenderable owner is NOT_FOUND.",
		Tags:        []string{"users"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			401: "INVALID_CREDENTIAL for a bad Bearer.",
			404: "NOT_FOUND when the account does not exist or is not renderable.",
			503: "SERVICE_UNAVAILABLE when the catalog or the account service cannot be reached.",
		}),
	}), s.listUserCollections)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "getCollectionAlias",
		Method:      http.MethodGet,
		Path:        "/collection-aliases/{alias_id}",
		Summary:     "Resolve a frozen collection alias",
		Description: "Maps a legacy forum collection id to the catalog folder id. Never creates a row. Visibility is the same as GET /collections/{collection_id}: a folder the caller cannot see is NOT_FOUND.",
		Tags:        []string{"collections"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			401: "INVALID_CREDENTIAL for a bad Bearer.",
			404: "NOT_FOUND when the alias is unknown or the folder is not visible.",
			503: "SERVICE_UNAVAILABLE when the catalog or the account service cannot be reached.",
		}),
	}), s.getCollectionAlias)
}
