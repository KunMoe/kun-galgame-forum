package apiv1

import (
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"

	"github.com/danielgtaylor/huma/v2"
)

func (s *Service) registerCollectionLists(api huma.API) {
	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "listCollections",
		Method:      http.MethodGet,
		Path:        "/collections",
		Summary:     "List the collection directory",
		Description: "A cursor collection of public collections that read as curated lists: not an owner's default, a non-blank title, and at least 3 works. " +
			"Without include_nsfw, collections whose content_rating is r18 are left out. contains_work_id narrows the list to collections holding that work.",
		Tags:        []string{"collections"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "LIMIT_TOO_LARGE; INVALID_CURSOR, including a cursor from another sort, filter or limit; INVALID_PARAMETER.",
			401: "INVALID_CREDENTIAL for a bad Bearer.",
			503: "SERVICE_UNAVAILABLE when the catalog or the account service cannot be reached.",
		}),
	}), s.listCollections)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listFollowedCollections",
		Method:      http.MethodGet,
		Path:        "/me/followed-collections",
		Summary:     "List the collections the caller follows",
		Description: "A cursor collection, newest follow first. A followed collection its owner has made private drops out of the list and returns if it is made public again.",
		Tags:        []string{"me"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "LIMIT_TOO_LARGE; INVALID_CURSOR.",
			403: "ACCOUNT_BANNED.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.listFollowedCollections)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "putCollectionFollow",
		Method:      http.MethodPut,
		Path:        "/collections/{collection_id}/follow",
		Summary:     "Follow a collection",
		Description: "Follows a public collection. Following again changes nothing. An account follows at most 1000 collections.",
		Tags:        []string{"collections"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			403: "ACCOUNT_BANNED.",
			404: "NOT_FOUND when the collection is not public.",
			422: "VALIDATION_FAILED for the caller's own collection or past the follow cap.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.followCollection)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "deleteCollectionFollow",
		Method:      http.MethodDelete,
		Path:        "/collections/{collection_id}/follow",
		Summary:     "Unfollow a collection",
		Description: "Stops following the collection. Unfollowing when not following changes nothing, also for a collection that has since gone private.",
		Tags:        []string{"collections"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			403: "ACCOUNT_BANNED.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.unfollowCollection)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "listCollectionEntries",
		Method:      http.MethodGet,
		Path:        "/collections/{collection_id}/entries",
		Summary:     "List a collection's works with the owner's notes",
		Description: "A page-number collection of the works in the collection, each with the owner's note. Default sort is the owner's own order. " +
			"Adult works are excluded from items and total unless include_nsfw=true.",
		Tags:        []string{"collections"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			401: "INVALID_CREDENTIAL for a bad Bearer.",
			404: "NOT_FOUND when the collection is not visible.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.listCollectionEntries)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "listRelatedCollections",
		Method:      http.MethodGet,
		Path:        "/collections/{collection_id}/related",
		Summary:     "List collections related to a collection",
		Description: "Up to 8 public collections from the directory, first those holding the same leading works, then the most followed. Not paginated. A private collection has none.",
		Tags:        []string{"collections"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			401: "INVALID_CREDENTIAL for a bad Bearer.",
			404: "NOT_FOUND when the collection is not visible.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.listRelatedCollections)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "patchCollectionWork",
		Method:      http.MethodPatch,
		Path:        "/collections/{collection_id}/works/{work_id}",
		Summary:     "Change a work's note or place in a collection",
		Description: "Sets the owner's note on the work, moves it in the owner's order, or both. after_work_id and before_work_id are exclusive. " +
			"Staff holding collection.edit_any may rewrite or clear the note on someone else's collection through a cookie session; they cannot move works.",
		Tags:        []string{"collections"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the collection is not the caller's or does not hold the work.",
			422: "CONTENT_REJECTED; VALIDATION_FAILED when nothing is sent, both anchors are sent, or the anchor is not another work in the collection.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.patchCollectionWork)

	huma.Register(api, v1.IdempotencyOptional(v1.Required(huma.Operation{
		OperationID:   "createCollectionReport",
		Method:        http.MethodPost,
		Path:          "/collection-reports",
		Summary:       "Report a collection",
		DefaultStatus: http.StatusNoContent,
		Description:   "Files a report on a public collection's title and description, or with work_id on the owner's note on that work, with the platform's moderators.",
		Tags:          []string{"collections"},
		Middlewares:   huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			403: "ACCOUNT_BANNED.",
			404: "NOT_FOUND when the collection is not public.",
			409: "IDEMPOTENCY_KEY_REUSED or IDEMPOTENCY_REQUEST_IN_PROGRESS.",
			422: "VALIDATION_FAILED when work_id names a work the collection does not hold or that carries no note.",
			429: "RATE_LIMITED when this account has filed too many reports.",
			503: "SERVICE_UNAVAILABLE when the moderation inbox cannot be reached.",
		}),
	})), s.createCollectionReport)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "listCollectionTags",
		Method:      http.MethodGet,
		Path:        "/collection-tags",
		Summary:     "List the directory's collection tags",
		Description: "Tags carried by directory collections, most used first, each with how many collections carry it. Not paginated.",
		Tags:        []string{"collections"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			401: "INVALID_CREDENTIAL for a bad Bearer.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.listCollectionTags)

	huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
		OperationID:   "copyCollection",
		Method:        http.MethodPost,
		Path:          "/collection-copies",
		Summary:       "Copy a collection",
		DefaultStatus: http.StatusCreated,
		Description: "Copies a public, unlisted or own collection into a new private collection of the caller's, with its works, order, notes, tags and cover. " +
			"The copy's description starts with the source's title. Location is the new collection's path.",
		Tags:        []string{"collections"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the source is someone else's private collection or does not exist.",
			409: "IDEMPOTENCY_KEY_REUSED or IDEMPOTENCY_REQUEST_IN_PROGRESS.",
			422: "CONTENT_REJECTED; VALIDATION_FAILED when the account already holds as many collections as catalog allows.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	})), s.copyCollection)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "getFollowedCollectionUpdates",
		Method:      http.MethodGet,
		Path:        "/me/followed-collections/updates",
		Summary:     "Count followed collections with new works",
		Description: "How many followed collections gained a work since the caller last opened them. Shared across NextMoe sites.",
		Tags:        []string{"me"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			403: "ACCOUNT_BANNED.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.getFollowedCollectionUpdates)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID:   "putFollowedCollectionReadMarker",
		Method:        http.MethodPut,
		Path:          "/collections/{collection_id}/read-marker",
		Summary:       "Mark a followed collection seen",
		DefaultStatus: http.StatusNoContent,
		Description:   "Clears the collection's new-work count for the caller. Sent when the caller opens a followed collection.",
		Tags:          []string{"me"},
		Middlewares:   huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			403: "ACCOUNT_BANNED.",
			404: "NOT_FOUND when the caller does not follow the collection.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.putFollowedCollectionReadMarker)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listCollectionProgress",
		Method:      http.MethodGet,
		Path:        "/me/collection-progress",
		Summary:     "Batch-read the caller's progress through collections",
		Description: "For each named collection, how many of its works the caller has played, is playing, wants, and keeps elsewhere. A collection the caller cannot see is in missing.",
		Tags:        []string{"me"},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "INVALID_PARAMETER when collection_ids is empty, holds more than 50 ids or something that is not an id.",
			403: "SCOPE_REQUIRED; ACCOUNT_BANNED.",
			503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
		}),
	}), s.listCollectionProgress)
}
