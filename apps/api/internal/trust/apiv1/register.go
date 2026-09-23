package apiv1

import (
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"

	"github.com/danielgtaylor/huma/v2"
)

const upstreamDown = "SERVICE_UNAVAILABLE when the trust service or the account service is unreachable, unconfigured, or answers with anything this operation does not map."

func Register(s *Service) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "listReportReasons",
			Method:      http.MethodGet,
			Path:        "/report-reasons",
			Summary:     "List report reasons",
			Description: "The reasons a report can be filed under, as the trust service currently offers them to this forum. " +
				"The list is data that the trust service's administrators maintain; this forum refreshes it every five minutes.",
			Tags: []string{"reports"},
			Responses: problemResponses(map[int]string{
				503: "SERVICE_UNAVAILABLE when the trust service is unreachable or unconfigured.",
			}),
		}), s.listReportReasons)

		huma.Register(api, v1.IdempotencyOptional(v1.Required(huma.Operation{
			OperationID:   "createReport",
			Method:        http.MethodPost,
			Path:          "/reports",
			Summary:       "Report content",
			DefaultStatus: http.StatusNoContent,
			Description: "Files a report with the trust-and-safety service on behalf of the caller. " +
				"A report has no id the reporter can read back. Reporting the same content again is accepted and counts once.",
			Tags: []string{"reports"},
			Responses: problemResponses(map[int]string{
				409: "IDEMPOTENCY_KEY_REUSED or IDEMPOTENCY_REQUEST_IN_PROGRESS.",
				422: "VALIDATION_FAILED when subject_kind is not a kind this forum registers, reason_key is not one the trust service offers, or subject_url is not on this forum.",
				429: "RATE_LIMITED when the caller has filed too many reports recently.",
				503: upstreamDown,
			}),
		})), s.createReport)

		tags := []string{"moderation"}
		reviewer := "It needs the trust.review permission, which a Bearer request never carries."

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "listReviewItems",
			Method:      http.MethodGet,
			Path:        "/admin/review-items",
			Summary:     "List the review inbox",
			Description: "This forum's items in the trust-and-safety review inbox, highest priority first with ties broken by descending id. " +
				"A page-number collection: page × limit may not exceed 10000, and total counts under the same filter as items. " + reviewer,
			Tags:        tags,
			Middlewares: huma.Middlewares{withAccessToken},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks trust.review, or the trust service refuses the caller's moderation role.",
				503: upstreamDown,
			}),
		}), s.listReviewItems)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getReviewItem",
			Method:      http.MethodGet,
			Path:        "/admin/review-items/{review_item_id}",
			Summary:     "Get a review item",
			Description: "One review item with the reports behind it. " + reviewer,
			Tags:        tags,
			Middlewares: huma.Middlewares{withAccessToken},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks trust.review, or the trust service refuses the caller's moderation role.",
				404: "NOT_FOUND when the item does not exist or belongs to another site.",
				503: upstreamDown,
			}),
		}), s.getReviewItem)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "updateReviewItem",
			Method:      http.MethodPatch,
			Path:        "/admin/review-items/{review_item_id}",
			Summary:     "Claim or decide a review item",
			Description: "state claimed takes a pending item for the caller. state actioned or dismissed decides a pending or claimed item; " +
				"actioned writes a disposition that the trust service delivers back to this forum to enforce. " +
				"Answers with the item as it stands afterwards. " + reviewer,
			Tags:        tags,
			Middlewares: huma.Middlewares{withAccessToken},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks trust.review, or the trust service refuses the caller's moderation role.",
				404: "NOT_FOUND when the item does not exist or belongs to another site.",
				409: "INVALID_STATE_TRANSITION when the item is already claimed or decided; detail names its current state.",
				422: "VALIDATION_FAILED when an actioned decision lacks action or reason_code, or when action, reason_code or statement come with any other state.",
				503: upstreamDown,
			}),
		}), s.updateReviewItem)
	}
}
