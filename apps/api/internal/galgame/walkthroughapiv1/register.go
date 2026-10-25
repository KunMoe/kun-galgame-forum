package walkthroughapiv1

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
		tags := []string{"galgame-walkthroughs"}
		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listWalkthroughs",
			Method:      http.MethodGet,
			Path:        "/walkthroughs",
			Summary:     "List walkthroughs",
			Description: "Walkthrough summaries without the chart, by ascending id. work_id narrows to one work, character_id to the walkthroughs whose chart routes to that character. Walkthroughs of adult works only with include_nsfw=true. A cursor collection.",
			Tags:        tags,
			Responses: problemResponses(map[int]string{
				400: "INVALID_PARAMETER when an id or the cursor is malformed.",
				503: "SERVICE_UNAVAILABLE when catalog cannot be reached.",
			}),
		}), s.listWalkthroughs)
		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "getWalkthrough",
			Method:      http.MethodGet,
			Path:        "/walkthroughs/{walkthrough_id}",
			Summary:     "Get a walkthrough",
			Description: "A walkthrough's choice graph: entries, saves and endings as nodes, segments of choices as edges, and the characters each route and ending belongs to. Text is in the walkthrough's lang; zh-Hans machine translations sit in localized. A walkthrough of an adult work is NOT_FOUND unless include_nsfw=true.",
			Tags:        tags,
			Responses: problemResponses(map[int]string{
				404: "NOT_FOUND when no such walkthrough is visible.",
				503: "SERVICE_UNAVAILABLE when catalog cannot be reached.",
			}),
		}), s.getWalkthrough)
	}
}
