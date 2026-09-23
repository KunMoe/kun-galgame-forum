package calendarapiv1

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

func Register(svc *Service) func(huma.API) {
	return func(api huma.API) {
		tag := []string{"release-calendar"}
		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "getReleaseCalendarToday",
			Method:      http.MethodGet,
			Path:        "/release-calendar/today",
			Summary:     "Whether a work releases today",
			Description: "Computes has_release on the walked current month in Asia/Tokyo. A day-precise release_date equal to today counts; a month- or year-precise date does not.",
			Tags:        tag,
			Responses: problemResponses(map[int]string{
				400: "INVALID_PARAMETER when include_nsfw is not a boolean.",
				503: "SERVICE_UNAVAILABLE when the catalog cannot be reached, including a failure mid-walk.",
			}),
		}), svc.getReleaseCalendarToday)

		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listReleaseCalendarPending",
			Method:      http.MethodGet,
			Path:        "/release-calendar/pending",
			Summary:     "List pending releases in a year",
			Description: "Works whose release date is known only to year precision. The walk follows catalog's cursor, at most 2,000 works.",
			Tags:        tag,
			Responses: problemResponses(map[int]string{
				400: "INVALID_PARAMETER when year is malformed.",
				503: "SERVICE_UNAVAILABLE when the catalog cannot be reached, including a failure mid-walk.",
			}),
		}), svc.listReleaseCalendarPending)

		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listReleaseCalendarTBA",
			Method:      http.MethodGet,
			Path:        "/release-calendar/tba",
			Summary:     "List works with an unknown release date",
			Description: "Works catalog files as status unknown. The walk follows catalog's cursor, at most 2,000 works.",
			Tags:        tag,
			Responses: problemResponses(map[int]string{
				400: "INVALID_PARAMETER when include_nsfw is not a boolean.",
				503: "SERVICE_UNAVAILABLE when the catalog cannot be reached, including a failure mid-walk.",
			}),
		}), svc.listReleaseCalendarTBA)

		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listReleaseCalendarUpcoming",
			Method:      http.MethodGet,
			Path:        "/release-calendar/upcoming",
			Summary:     "List upcoming releases by month",
			Description: "Walks from the current Asia/Tokyo month to catalog's max_month, at most 24 months. A failed month is SERVICE_UNAVAILABLE for the whole response. Empty months are omitted.",
			Tags:        tag,
			Responses: problemResponses(map[int]string{
				400: "INVALID_PARAMETER when include_nsfw is not a boolean.",
				503: "SERVICE_UNAVAILABLE when any month's catalog walk fails.",
			}),
		}), svc.listReleaseCalendarUpcoming)

		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listReleaseCalendarMonth",
			Method:      http.MethodGet,
			Path:        "/release-calendar",
			Summary:     "List releases in a month",
			Description: "Walks catalog's cursor for the month, at most 2,000 works. is_truncated is true when the cap is hit. An id catalog does not render is dropped with a warning.",
			Tags:        tag,
			Responses: problemResponses(map[int]string{
				400: "INVALID_PARAMETER when month is malformed.",
				503: "SERVICE_UNAVAILABLE when the catalog cannot be reached, including a failure mid-walk.",
			}),
		}), svc.listReleaseCalendarMonth)
	}
}
