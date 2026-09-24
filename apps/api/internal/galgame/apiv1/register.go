package apiv1

import (
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

func Register(svc *Service) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listWorkCollectedMonths",
			Method:      http.MethodGet,
			Path:        "/works/collected-months",
			Summary:     "List months that have collected works",
			Description: "Months in which the forum first listed a published work that has a resource, under the same population as GET /works with its default filters. " +
				"Years descend, months inside a year ascend. A query or scan failure is INTERNAL_ERROR, never an empty list.",
			Tags: []string{"works"},
			Responses: problemResponses(map[int]string{
				400: "INVALID_PARAMETER when include_nsfw is not a boolean.",
			}),
		}), svc.listWorkCollectedMonths)

		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listWorks",
			Method:      http.MethodGet,
			Path:        "/works",
			Summary:     "List works on the forum",
			Description: "A page-number collection of published forum works. Default sort is resource_updated_desc, default limit 24. " +
				"NSFW works are excluded before paging unless include_nsfw=true; a work whose content_limit has not been synced yet counts as adult until it is. " +
				"Default pages require at least one forum resource; include_resourceless=true lists every published work. " +
				"An id catalog does not render is dropped with a warning, so a page may be shorter than limit; total still counts the SQL population.",
			Tags: []string{"works"},
			Responses: problemResponses(map[int]string{
				400: "UNKNOWN_ENUM_VALUE, UNKNOWN_SORT, LIMIT_TOO_LARGE, or INVALID_PARAMETER when page × limit is too deep, a date or month is malformed, or a boolean or array is the wrong shape.",
				503: "SERVICE_UNAVAILABLE when the catalog cannot hydrate the page.",
			}),
		}), svc.listWorks)

		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listLibraryWorks",
			Method:      http.MethodGet,
			Path:        "/library-works",
			Summary:     "List works from the catalog library",
			Description: "A page-number collection from catalog's work search population. Default sort is popularity_desc, default limit 24. " +
				"q is optional. Forum resource-axis, host, collection-date and rating filters are not parameters of this collection. " +
				"An id catalog does not render is dropped with a warning, so a page may be shorter than limit; total still counts catalog's population.",
			Tags: []string{"works"},
			Responses: problemResponses(map[int]string{
				400: "UNKNOWN_SORT, LIMIT_TOO_LARGE, or INVALID_PARAMETER when q is blank after trimming, a date is malformed, or page × limit is too deep.",
				503: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
			}),
		}), svc.listLibraryWorks)

		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "getWork",
			Method:      http.MethodGet,
			Path:        "/works/{work_id}",
			Summary:     "Get a work",
			Description: "Returns the work. Hidden or unknown works are NOT_FOUND; a merged work is ENTITY_MERGED with current_id. " +
				"Local published is not required. include_nsfw=false strips adult tags. Each read adds one view without touching updated_at. " +
				"viewer is null for an anonymous caller.",
			Tags:        []string{"works"},
			Middlewares: huma.Middlewares{withAccessToken},
			Responses: problemResponses(map[int]string{
				400: "INVALID_PARAMETER when include_nsfw is not a boolean.",
				404: "NOT_FOUND when the work does not exist or is hidden; ENTITY_MERGED when it was merged, with current_id.",
				503: "SERVICE_UNAVAILABLE when the catalog or the account service cannot be reached.",
			}),
		}), svc.getWork)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "putWorkLike",
			Method:      http.MethodPut,
			Path:        "/works/{work_id}/like",
			Summary:     "Like a work",
			Description: "Sets the caller's like. Liking again changes nothing. Liking one's own work is SELF_LIKE_FORBIDDEN.",
			Tags:        []string{"works"},
			Responses: problemResponses(map[int]string{
				403: "SELF_LIKE_FORBIDDEN or ACCOUNT_BANNED.",
				404: "NOT_FOUND when the work does not exist or is hidden.",
				503: "SERVICE_UNAVAILABLE when the catalog, the account service or moemoepoint cannot be reached.",
			}),
		}), svc.putWorkLike)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "deleteWorkLike",
			Method:      http.MethodDelete,
			Path:        "/works/{work_id}/like",
			Summary:     "Unlike a work",
			Description: "Clears the caller's like. Unliking when not liked changes nothing.",
			Tags:        []string{"works"},
			Responses: problemResponses(map[int]string{
				403: "ACCOUNT_BANNED.",
				404: "NOT_FOUND when the work does not exist or is hidden.",
				503: "SERVICE_UNAVAILABLE when the catalog, the account service or moemoepoint cannot be reached.",
			}),
		}), svc.deleteWorkLike)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "listMyWorkStates",
			Method:      http.MethodGet,
			Path:        "/me/work-states",
			Summary:     "Batch-read the caller's work like and favorite states",
			Description: "Answers, for each work id named in work_ids, whether the caller liked it and whether they hold it in a folder. " +
				"It is a batch read and is not paginated: work_ids is required, holds 1 to 100 ids. " +
				"An id catalog does not know or has hidden is missing. " +
				"has_favorited is false when the caller's folders cannot be read; has_liked is always answered.",
			Tags:        []string{"me"},
			Middlewares: huma.Middlewares{withAccessToken},
			Responses: problemResponses(map[int]string{
				400: "INVALID_PARAMETER when work_ids is absent, empty, holds more than 100 ids, or holds something that is not a positive decimal integer.",
				403: "ACCOUNT_BANNED.",
				503: "SERVICE_UNAVAILABLE when the catalog cannot say which works exist.",
			}),
		}), svc.listMyWorkStates)

		svc.registerUserPlane(api)
		svc.registerEdit(api)
		svc.registerSubmissions(api)

		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listWorkMoyuPatches",
			Method:      http.MethodGet,
			Path:        "/works/{work_id}/moyu-patches",
			Summary:     "List a work's patches on moyu",
			Description: "Lists the pages www.moyu.moe, the KUN Galgame patch site, holds for the work, each with its live resources. " +
				"Usually one page: moyu dedupes on the VNDB string, so a game that arrived under two spellings has two, and the page a reader should land on comes first. " +
				"The whole set in one response; it is never paged. An empty list means moyu has nothing for the work. " +
				"No download link, share code or password is carried; send a reader to web_url. " +
				"An answer may be up to 30 minutes old. NOT_FOUND when the work does not exist.",
			Tags: []string{"works"},
			Responses: map[string]*huma.Response{
				"503": {
					Description: "SERVICE_UNAVAILABLE: www.moyu.moe, the catalog or the account service cannot be reached.",
					Content: map[string]*huma.MediaType{
						problem.ContentType: {Schema: &huma.Schema{Ref: v1.ProblemRef}},
					},
				},
			},
		}), svc.listGalgameMoyuPatches)
	}
}
