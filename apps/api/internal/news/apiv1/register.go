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

func unavailable(desc string) map[string]*huma.Response {
	return problemResponses(map[int]string{http.StatusServiceUnavailable: desc})
}

const upstreamDown = "SERVICE_UNAVAILABLE when the news service is not configured or cannot be reached."

func Register(s *Service) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listNewsItems",
			Method:      http.MethodGet,
			Path:        "/news-items",
			Summary:     "List news items",
			Description: "The news index, partner items and published community submissions, newest first. A cursor collection over the news service's own cursor; " +
				"the cursor is bound to every filter and to limit. limit is 1–50 because that is the news service's page cap.",
			Tags:      []string{"news"},
			Responses: unavailable(upstreamDown),
		}), s.listNewsItems)

		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "getNewsItem",
			Method:      http.MethodGet,
			Path:        "/news-items/{news_item_id}",
			Summary:     "Get a news item",
			Description: "One published news item. Pending and withdrawn items are NOT_FOUND, the same as an id that was never issued. " +
				"content is the item's own text when has_body is true, and an empty document otherwise.",
			Tags: []string{"news"},
			Responses: problemResponses(map[int]string{
				http.StatusNotFound:           "NOT_FOUND when the item never existed, is still pending, or was withdrawn.",
				http.StatusServiceUnavailable: upstreamDown,
			}),
		}), s.getNewsItem)

		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listNewsSources",
			Method:      http.MethodGet,
			Path:        "/news-sources",
			Summary:     "List news sources",
			Description: "The whole source directory, partners and community, which is where a news item's news_source key resolves to a name and attribution. Not paged.",
			Tags:        []string{"news"},
			Responses:   unavailable(upstreamDown),
		}), s.listNewsSources)

		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "getNewsArchive",
			Method:      http.MethodGet,
			Path:        "/news-archive",
			Summary:     "Get the news archive index",
			Description: "Item counts by year, and by month for the one year asked about, under the same filters as listNewsItems. Years and months are cut on Asia/Shanghai.",
			Tags:        []string{"news"},
			Responses:   unavailable(upstreamDown),
		}), s.getNewsArchive)

		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "getNewsMonth",
			Method:      http.MethodGet,
			Path:        "/news-archive/{year}/{month}",
			Summary:     "Get one month of news",
			Description: "How many items one month holds and how they fall across its days, every day listed. The items themselves are listNewsMonthItems.",
			Tags:        []string{"news"},
			Responses:   unavailable(upstreamDown),
		}), s.getNewsMonth)

		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listNewsMonthItems",
			Method:      http.MethodGet,
			Path:        "/news-archive/{year}/{month}/items",
			Summary:     "List one month of news",
			Description: "One month of news items, newest first, as a page-number collection: a month is a reference page, and a reader who wants its third week should not scroll the first two. " +
				"total counts the month after the day filter.",
			Tags:      []string{"news"},
			Responses: unavailable(upstreamDown),
		}), s.listNewsMonthItems)
	}
}
