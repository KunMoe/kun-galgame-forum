package apiv1

import (
	"errors"
	"net/http"
	"strconv"

	"kun-galgame-api/internal/admin/repository"
	"kun-galgame-api/internal/admin/service"
	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"github.com/danielgtaylor/huma/v2"
)

var errUnconfigured = errors.New("permission v1 is not configured")

type Service struct {
	roles  *service.RolePermissionService
	users  *service.UserPermissionService
	audit  *repository.PermissionAuditRepository
	people *userclient.Client
	cdn    string
}

func New(roles *service.RolePermissionService, users *service.UserPermissionService,
	audit *repository.PermissionAuditRepository, people *userclient.Client, cdn string) *Service {
	return &Service{roles: roles, users: users, audit: audit, people: people, cdn: cdn}
}

func Register(s *Service) func(huma.API) {
	return func(api huma.API) {
		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getMyPermissions",
			Method:      http.MethodGet,
			Path:        "/me/permissions",
			Summary:     "List the caller's permissions",
			Description: "Every permission key the caller holds right now: their roles, the role overrides, and their personal overrides. " +
				"A Bearer request always gets an empty list. It only decides what to show; every write checks again.",
			Tags: []string{"permissions"},
		}), s.getMyPermissions)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getRolePermissionMatrix",
			Method:      http.MethodGet,
			Path:        "/admin/role-permissions",
			Summary:     "Get the role permission matrix",
			Description: "Each role's baseline, stored overrides and effective permissions, with the permission catalog. Admins only.",
			Tags:        []string{"permissions"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller is not an admin.",
			}),
		}), s.getRolePermissionMatrix)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "updateRolePermissionMatrix",
			Method:      http.MethodPatch,
			Path:        "/admin/role-permissions",
			Summary:     "Replace the overrides of some roles",
			Description: "Replaces the override set of every listed role in one transaction, and records one permission change per role. " +
				"The caller may change only roles ranked below their own and only keys they hold themselves, ren cannot be changed, " +
				"and the result must leave moderator holding nothing admin lacks. Other instances pick the change up within 60 seconds.",
			Tags: []string{"permissions"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller is not an admin.",
				422: "VALIDATION_FAILED, with one error per refused position: ren, a repeated role or key, a role at or above the caller, " +
					"a key outside the catalog, an override that changes nothing against the baseline, a key the caller does not hold, " +
					"and, at /changes, each key moderator would hold without admin.",
			}),
		}), s.updateRolePermissionMatrix)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "getUserPermissions",
			Method:      http.MethodGet,
			Path:        "/admin/user-permissions/{user_id}",
			Summary:     "Get a user's permissions",
			Description: "What the user's roles grant, the user's personal overrides, and the result. Roles come from the account service's current record. Admins only.",
			Tags:        []string{"permissions"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller is not an admin.",
				404: "NOT_FOUND when the account service has no such user.",
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
			}),
		}), s.getUserPermissions)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "replaceUserPermissions",
			Method:      http.MethodPut,
			Path:        "/admin/user-permissions/{user_id}",
			Summary:     "Replace a user's personal overrides",
			Description: "Replaces the user's personal override set and records a permission change. " +
				"The caller may change only users ranked below them and only keys they hold themselves; a ren holder cannot be changed. " +
				"The target's roles are read from the account service and nothing is written when it cannot be reached.",
			Tags: []string{"permissions"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller is not an admin.",
				404: "NOT_FOUND when the account service has no such user.",
				422: "VALIDATION_FAILED, with one error per refused position: parameter user_id for a ren holder or a user at or above the caller, " +
					"otherwise a repeated key, a key outside the catalog, an override that changes nothing against the user's roles, or a key the caller does not hold.",
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached.",
			}),
		}), s.replaceUserPermissions)

		huma.Register(api, v1.Required(huma.Operation{
			OperationID: "listPermissionChanges",
			Method:      http.MethodGet,
			Path:        "/admin/permission-changes",
			Summary:     "List permission changes",
			Description: "Every recorded replacement of a role's or a user's overrides, newest first with ties broken by descending id. " +
				"A page-number collection: page × limit may not exceed 10000. Admins only.",
			Tags: []string{"permissions"},
			Responses: problemResponses(map[int]string{
				403: "PERMISSION_REQUIRED when the caller is not an admin.",
				503: "SERVICE_UNAVAILABLE when the account service cannot be reached to name the people involved.",
			}),
		}), s.listPermissionChanges)
	}
}

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
