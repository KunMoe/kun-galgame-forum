package apiv1

import (
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
)

func unavailable(desc string) map[string]*huma.Response {
	return map[string]*huma.Response{
		"503": {
			Description: desc,
			Content: map[string]*huma.MediaType{
				problem.ContentType: {Schema: &huma.Schema{Ref: v1.ProblemRef}},
			},
		},
	}
}

const upstreamDown = "SERVICE_UNAVAILABLE when the news service is not configured or cannot be reached."

func Register(s *Service) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listNewsItems",
			Method:      http.MethodGet,
			Path:        "/news-items",
			Summary:     "List news items",
			Description: "The partner news index, newest first. A cursor collection over the news service's own cursor; " +
				"the cursor is bound to every filter and to limit. limit is 1–50 because that is the news service's page cap.",
			Tags:      []string{"news"},
			Responses: unavailable(upstreamDown),
		}), s.listNewsItems)

		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listNewsSources",
			Method:      http.MethodGet,
			Path:        "/news-sources",
			Summary:     "List news partners",
			Description: "The whole partner directory, which is where a news item's news_source key resolves to a name and attribution. Not paged.",
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
