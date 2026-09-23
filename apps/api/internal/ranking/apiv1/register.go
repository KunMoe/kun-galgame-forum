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

const topN = "A top-N list rather than a paged collection: it has no page, cursor or total. "

func Register(s *Service) func(huma.API) {
	return func(api huma.API) {
		tags := []string{"rankings"}

		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listWorkRanking",
			Method:      http.MethodGet,
			Path:        "/rankings/works",
			Summary:     "Rank works",
			Description: topN + "Only published works; without include_resourceless, only works that have a resource. " +
				"Works the catalog does not return are dropped and the places renumbered.",
			Tags: tags,
			Responses: problemResponses(map[int]string{
				400: "UNKNOWN_SORT, LIMIT_TOO_LARGE or INVALID_PARAMETER.",
				503: "SERVICE_UNAVAILABLE when the catalog or the account service is unreachable.",
			}),
		}), s.listWorkRanking)

		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listTopicRanking",
			Method:      http.MethodGet,
			Path:        "/rankings/topics",
			Summary:     "Rank topics",
			Description: topN + "Only topics anonymous visitors can list: not hidden and public. " +
				"Topics whose author is banned or deleted are dropped and the places renumbered.",
			Tags: tags,
			Responses: problemResponses(map[int]string{
				400: "UNKNOWN_SORT, LIMIT_TOO_LARGE or INVALID_PARAMETER.",
				503: "SERVICE_UNAVAILABLE when the account service is unreachable.",
			}),
		}), s.listTopicRanking)

		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listUserRanking",
			Method:      http.MethodGet,
			Path:        "/rankings/users",
			Summary:     "Rank users",
			Description: topN + "Counts include only what anonymous visitors can read. " +
				"Banned and deleted accounts are dropped and the places renumbered.",
			Tags: tags,
			Responses: problemResponses(map[int]string{
				400: "UNKNOWN_SORT or LIMIT_TOO_LARGE.",
				503: "SERVICE_UNAVAILABLE when the account service is unreachable.",
			}),
		}), s.listUserRanking)
	}
}
