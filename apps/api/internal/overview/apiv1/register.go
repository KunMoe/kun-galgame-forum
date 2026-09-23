package apiv1

import (
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"

	"github.com/danielgtaylor/huma/v2"
)

func Register(s *Service) func(huma.API) {
	return func(api huma.API) {
		dashboard := "It needs the admin.dashboard permission, which a Bearer request never carries."

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getAdminOverview",
			Method:      http.MethodGet,
			Path:        "/admin/overview",
			Summary:     "Get the site totals",
			Description: "How much of each kind of content the site holds right now. " + dashboard,
			Tags:        []string{"admin"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks admin.dashboard.",
			}),
		}), s.getOverview)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "listAdminOverviewDays",
			Method:      http.MethodGet,
			Path:        "/admin/overview/daily",
			Summary:     "List new content per day",
			Description: "One bucket per Asia/Shanghai calendar day: today and the days − 1 days before it, oldest first. " +
				"Every day is present, with zero counts when nothing was created. Today's bucket counts up to the moment of the request. " + dashboard,
			Tags: []string{"admin"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks admin.dashboard.",
			}),
		}), s.listOverviewDays)
	}
}
