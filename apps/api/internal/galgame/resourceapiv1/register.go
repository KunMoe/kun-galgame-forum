package apiv1

import (
	"context"
	"net/http"
	"strconv"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/problem"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
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

const (
	tagResources = "galgame-resources"
	tagWorks     = "works"

	activeCheck = "The caller is checked against the account service's current record first: a banned account is ACCOUNT_BANNED, and a failure of that lookup is SERVICE_UNAVAILABLE with nothing written. "
	bearerNote  = "Requests authenticated with a Bearer token never carry resource permissions. "
)

func withAccessToken(ctx huma.Context, next func(huma.Context)) {
	fc := humafiber.Unwrap(ctx)
	next(huma.WithValue(ctx, accessTokenCtxKey{}, middleware.GetAccessToken(fc)))
}

func accessToken(ctx context.Context) string {
	s, _ := ctx.Value(accessTokenCtxKey{}).(string)
	return s
}

func Register(svc *Service) func(huma.API) {
	return func(api huma.API) {
		registerBrowse(api, svc)
		registerWork(api, svc)
		registerWrite(api, svc)
		registerEngage(api, svc)
	}
}

func registerBrowse(api huma.API, svc *Service) {
	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "listGalgameResources",
		Method:      http.MethodGet,
		Path:        "/galgame-resources",
		Summary:     "List galgame download resources",
		Description: "Lists download resources as a page-number collection. Default sort is created_desc, default limit 50. " +
			"Resources whose author is not renderable are omitted from both items and total. " +
			"include_nsfw=false excludes resources on a local NSFW work. Secrets are never included.",
		Tags: []string{tagResources},
		Responses: problemResponses(map[int]string{
			400: "UNKNOWN_ENUM_VALUE, UNKNOWN_SORT, LIMIT_TOO_LARGE, or INVALID_PARAMETER.",
		}),
	}), svc.listGalgameResources)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "getGalgameResource",
		Method:      http.MethodGet,
		Path:        "/galgame-resources/{resource_id}",
		Summary:     "Get a galgame download resource",
		Description: "Returns the resource without download secrets and counts one view. An unrenderable author or unpublished work is NOT_FOUND. viewer is null for an anonymous caller.",
		Tags:        []string{tagResources},
		Responses: problemResponses(map[int]string{
			404: "NOT_FOUND when the resource does not exist, its author is not renderable, or the work is unpublished.",
		}),
	}), svc.getGalgameResource)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "createGalgameResourceDownload",
		Method:      http.MethodPost,
		Path:        "/galgame-resources/{resource_id}/downloads",
		Summary:     "Issue download secrets for a resource",
		Description: "Returns the stored download URLs and codes, and counts one download. Anonymous callers may use it. Secrets never appear on GET.",
		Tags:        []string{tagResources},
		Responses: problemResponses(map[int]string{
			404: "NOT_FOUND when the resource does not exist, its author is not renderable, or the work is unpublished.",
		}),
	}), svc.createGalgameResourceDownload)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "getGalgameResourceSource",
		Method:      http.MethodGet,
		Path:        "/galgame-resources/{resource_id}/source",
		Summary:     "Get a resource's edit source",
		Description: "Returns the editable fields of a resource, secrets included, without counting a download. " +
			"Needs viewer.can_edit. " + bearerNote + activeCheck,
		Tags: []string{tagResources},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without can_edit; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the resource does not exist, its author is not renderable, or the work is unpublished.",
		}),
	}), svc.getGalgameResourceSource)
}

func registerWork(api huma.API, svc *Service) {
	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "listWorkResources",
		Method:      http.MethodGet,
		Path:        "/works/{work_id}/resources",
		Summary:     "List a work's download resources",
		Description: "Lists download resources of a work as a page-number collection, valid first then newest. " +
			"An unknown, hidden or unpublished work is NOT_FOUND. An empty list is 200 with total 0. NSFW is not gated.",
		Tags: []string{tagWorks},
		Responses: problemResponses(map[int]string{
			400: "LIMIT_TOO_LARGE, INVALID_PARAMETER, or UNKNOWN_ENUM_VALUE.",
			404: "NOT_FOUND when the work does not exist, is hidden, or is unpublished.",
		}),
	}), svc.listWorkResources)

	huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
		OperationID:   "createWorkResource",
		Method:        http.MethodPost,
		Path:          "/works/{work_id}/resources",
		Summary:       "Create a download resource",
		DefaultStatus: http.StatusCreated,
		Description: "Creates a download resource on the work. Idempotency-Key is required. " + activeCheck +
			"A banned work is RESOURCE_PUBLISH_BANNED. Location is the new resource's path.",
		Tags:        []string{tagWorks},
		Middlewares: huma.Middlewares{withAccessToken},
		Responses: problemResponses(map[int]string{
			400: "INVALID_PARAMETER when Idempotency-Key is missing or malformed.",
			403: "RESOURCE_PUBLISH_BANNED or ACCOUNT_BANNED.",
			404: "NOT_FOUND when the work does not exist or is hidden.",
			409: "IDEMPOTENCY_KEY_REUSED or IDEMPOTENCY_REQUEST_IN_PROGRESS.",
			422: "VALIDATION_FAILED or CONTENT_REJECTED.",
		}),
	})), svc.createWorkResource)
}

func registerWrite(api huma.API, svc *Service) {
	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "updateGalgameResource",
		Method:      http.MethodPatch,
		Path:        "/galgame-resources/{resource_id}",
		Summary:     "Edit a download resource",
		Description: "Changes the fields that are sent and returns the resource. Needs can_edit. " + bearerNote + activeCheck +
			"Only content_markdown and download_urls, when sent, go through the trust-and-safety check. " +
			"download_urls and axis arrays, when present, replace the whole set. state may only be valid.",
		Tags: []string{tagResources},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without can_edit; RESOURCE_PUBLISH_BANNED; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the resource does not exist, its author is not renderable, or the work is unpublished.",
			422: "VALIDATION_FAILED or CONTENT_REJECTED.",
		}),
	}), svc.updateGalgameResource)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID:   "deleteGalgameResource",
		Method:        http.MethodDelete,
		Path:          "/galgame-resources/{resource_id}",
		Summary:       "Delete a download resource",
		DefaultStatus: http.StatusNoContent,
		Description:   "Deletes a resource. Needs can_delete. " + bearerNote + activeCheck,
		Tags:          []string{tagResources},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without can_delete; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the resource does not exist, its author is not renderable, or the work is unpublished.",
		}),
	}), svc.deleteGalgameResource)
}

func registerEngage(api huma.API, svc *Service) {
	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "putGalgameResourceLike",
		Method:      http.MethodPut,
		Path:        "/galgame-resources/{resource_id}/like",
		Summary:     "Like a download resource",
		Description: "Sets the caller's like. Liking again changes nothing. Liking one's own resource is SELF_LIKE_FORBIDDEN. " + activeCheck,
		Tags:        []string{tagResources},
		Responses: problemResponses(map[int]string{
			403: "SELF_LIKE_FORBIDDEN or ACCOUNT_BANNED.",
			404: "NOT_FOUND when the resource does not exist, its author is not renderable, or the work is unpublished.",
		}),
	}), svc.putGalgameResourceLike)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "deleteGalgameResourceLike",
		Method:      http.MethodDelete,
		Path:        "/galgame-resources/{resource_id}/like",
		Summary:     "Unlike a download resource",
		Description: "Clears the caller's like. Unliking when not liked changes nothing. " + activeCheck,
		Tags:        []string{tagResources},
		Responses: problemResponses(map[int]string{
			403: "ACCOUNT_BANNED.",
			404: "NOT_FOUND when the resource does not exist, its author is not renderable, or the work is unpublished.",
		}),
	}), svc.deleteGalgameResourceLike)

	huma.Register(api, v1.IdempotencyOptional(v1.Required(huma.Operation{
		OperationID: "createGalgameResourceExpiryReport",
		Method:      http.MethodPost,
		Path:        "/galgame-resources/{resource_id}/expiry-reports",
		Summary:     "Report a resource as expired",
		Description: "Checks the stored links and may mark the resource expired. Already expired is 200 with no side effects. " + activeCheck,
		Tags:        []string{tagResources},
		Responses: problemResponses(map[int]string{
			404: "NOT_FOUND when the resource does not exist, its author is not renderable, or the work is unpublished.",
			409: "IDEMPOTENCY_KEY_REUSED or IDEMPOTENCY_REQUEST_IN_PROGRESS.",
		}),
	})), svc.createGalgameResourceExpiryReport)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "putWorkResourcePublishBan",
		Method:      http.MethodPut,
		Path:        "/works/{work_id}/resource-publish-ban",
		Summary:     "Ban publishing download resources on a work",
		Description: "Sets the resource-publish ban. Needs galgame.ban_resource_publish. " + bearerNote + activeCheck +
			"An unknown catalog work is NOT_FOUND and writes no local row.",
		Tags: []string{tagWorks},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED, including on a Bearer request; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the work does not exist or is hidden.",
		}),
	}), svc.putWorkResourcePublishBan)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "deleteWorkResourcePublishBan",
		Method:      http.MethodDelete,
		Path:        "/works/{work_id}/resource-publish-ban",
		Summary:     "Lift the resource-publish ban on a work",
		Description: "Clears the resource-publish ban. Needs galgame.ban_resource_publish. " + bearerNote + activeCheck +
			"A missing local row is 200 with is_resource_publish_banned false.",
		Tags: []string{tagWorks},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED, including on a Bearer request; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the work does not exist or is hidden.",
		}),
	}), svc.deleteWorkResourcePublishBan)
}
