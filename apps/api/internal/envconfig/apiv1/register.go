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
			Content:     map[string]*huma.MediaType{problem.ContentType: {Schema: &huma.Schema{Ref: v1.ProblemRef}}},
		}
	}
	return out
}

const denied = "PERMISSION_REQUIRED without setting.manage, which a Bearer request never carries."

func Register(s *Service) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getAdminEnvConfig",
			Method:      http.MethodGet,
			Path:        "/admin/settings/env",
			Summary:     "Read the environment variables and the config store",
			Description: "Every variable the running API read with where its value came from, every stored value, and what needs doing about each. " +
				"Secrets appear only as a fingerprint and their last characters. Needs setting.manage.",
			Tags:      []string{"admin"},
			Responses: problemResponses(map[int]string{403: denied}),
		}), s.getEnvConfig)
		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "putAdminEnvVariable",
			Method:      http.MethodPut,
			Path:        "/admin/settings/env/variables/{variable}",
			Summary:     "Store an environment variable",
			Description: "Seals the value into the config store and records the change. The API reads the store when it starts, so the value takes effect on the next restart, " +
				"and only while the environment does not set the same variable. Needs setting.manage.",
			Tags: []string{"admin"},
			Responses: problemResponses(map[int]string{
				403: denied,
				422: "VALIDATION_FAILED at the variable parameter for a bootstrap or topology variable, which stays in the compose file.",
				503: "SERVICE_UNAVAILABLE when the API started without the store key.",
			}),
		}), s.putEnvVariable)
		huma.Register(api, v1.Required(huma.Operation{
			OperationID:   "deleteAdminEnvVariable",
			Method:        http.MethodDelete,
			Path:          "/admin/settings/env/variables/{variable}",
			Summary:       "Remove a stored environment variable",
			Description:   "Deletes the stored value and records the change. After the next restart the variable falls back to the environment, then the code default. Needs setting.manage.",
			Tags:          []string{"admin"},
			DefaultStatus: http.StatusNoContent,
			Responses: problemResponses(map[int]string{
				403: denied,
				404: "NOT_FOUND when nothing is stored for the variable.",
				503: "SERVICE_UNAVAILABLE when the API started without the store key.",
			}),
		}), s.deleteEnvVariable)
		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "listAdminEnvChanges",
			Method:      http.MethodGet,
			Path:        "/admin/settings/env/changes",
			Summary:     "List changes to the config store",
			Description: "Every write and delete, newest first. Needs setting.manage.",
			Tags:        []string{"admin"},
			Responses:   problemResponses(map[int]string{403: denied}),
		}), s.listEnvChanges)
	}
}
