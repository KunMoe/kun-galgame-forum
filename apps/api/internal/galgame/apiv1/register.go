package apiv1

import (
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

func Register(svc *Service) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Optional(huma.Operation{
			OperationID: "getWork",
			Method:      http.MethodGet,
			Path:        "/works/{work_id}",
			Summary:     "Get a work",
			Description: "Returns a WorkRef for the catalog work. Hidden or unknown works are NOT_FOUND. Local published is not required.",
			Tags:        []string{"works"},
			Responses: map[string]*huma.Response{
				"404": {
					Description: "NOT_FOUND when the work does not exist or is hidden.",
					Content: map[string]*huma.MediaType{
						problem.ContentType: {Schema: &huma.Schema{Ref: v1.ProblemRef}},
					},
				},
				"503": {
					Description: "SERVICE_UNAVAILABLE when the catalog cannot be reached.",
					Content: map[string]*huma.MediaType{
						problem.ContentType: {Schema: &huma.Schema{Ref: v1.ProblemRef}},
					},
				},
			},
		}), svc.getWork)

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
