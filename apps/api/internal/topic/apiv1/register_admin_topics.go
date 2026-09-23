package apiv1

import (
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterAdminTopics(a *AdminTopics) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "listHiddenTopics",
			Method:      http.MethodGet,
			Path:        "/admin/hidden-topics",
			Summary:     "List hidden topics",
			Description: "The staff table of every hidden topic, whoever hid it, newest bump first with ties broken by descending id. " +
				"A page-number collection: page × limit may not exceed 10000, and total counts under the same filters as items. " +
				"It needs the topic.view_hidden permission, which a Bearer request never carries.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks topic.view_hidden.",
			}),
		}), a.listHiddenTopics)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getAdminTopic",
			Method:      http.MethodGet,
			Path:        "/admin/topics/{topic_id}",
			Summary:     "Get a topic's staff view",
			Description: "The topic as staff see it before purging it: what purgeTopic would delete with it, and how much lottery escrow it would hand back. " +
				"It needs the topic.delete_any permission.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks topic.delete_any.",
				404: "NOT_FOUND when the topic does not exist.",
			}),
		}), a.getAdminTopic)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID:   "purgeTopic",
			Method:        http.MethodDelete,
			Path:          "/admin/topics/{topic_id}",
			Summary:       "Purge a topic",
			DefaultStatus: http.StatusNoContent,
			Description: "Deletes the topic for good, with its replies, comments, polls, lotteries, favorites, view buckets and the notifications that link to it. " +
				"An open lottery's escrowed moemoepoint goes back to its author first. Hiding is the reversible alternative. " +
				"It needs the topic.delete_any permission.",
			Tags: []string{"topics"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller lacks topic.delete_any.",
				404: "NOT_FOUND when the topic does not exist.",
				409: "LOTTERY_DRAWN when one of the topic's lotteries is being drawn right now; retry once the draw ends.",
			}),
		}), a.purgeTopic)
	}
}
