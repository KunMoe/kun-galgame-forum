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

const (
	tagToolsets = "toolsets"
	tagUsers    = "users"

	activeCheck = "The caller is checked against the account service's current record first: a banned account is ACCOUNT_BANNED, and a failure of that lookup is SERVICE_UNAVAILABLE with nothing written. "
	bearerNote  = "Requests authenticated with a Bearer token never carry toolset permissions. "
)

func Register(svc *Service) func(huma.API) {
	return func(api huma.API) {
		registerToolsets(api, svc)
		registerResources(api, svc)
		registerUploads(api, svc)
		registerPracticality(api, svc)
	}
}

func registerToolsets(api huma.API, svc *Service) {
	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "listToolsets",
		Method:      http.MethodGet,
		Path:        "/toolsets",
		Summary:     "List toolsets",
		Description: "Lists toolsets as a page-number collection. Default sort is resource_updated_desc. " +
			"type, language, platform and version narrow the list; an unknown token is UNKNOWN_ENUM_VALUE. " +
			"q searches the name. Toolsets whose author is not renderable are omitted from both items and total.",
		Tags: []string{tagToolsets},
		Responses: problemResponses(map[int]string{
			400: "UNKNOWN_ENUM_VALUE, UNKNOWN_SORT, LIMIT_TOO_LARGE, or INVALID_PARAMETER.",
		}),
	}), svc.listToolsets)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "listUserToolsets",
		Method:      http.MethodGet,
		Path:        "/users/{user_id}/toolsets",
		Summary:     "List a user's toolsets",
		Description: "Lists toolsets the user authored, newest first. An unrenderable user is NOT_FOUND. An empty list is 200 with total 0.",
		Tags: []string{tagUsers},
		Responses: problemResponses(map[int]string{
			400: "LIMIT_TOO_LARGE or INVALID_PARAMETER.",
			404: "NOT_FOUND when the user does not exist or is not renderable.",
		}),
	}), svc.listUserToolsets)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "getToolset",
		Method:      http.MethodGet,
		Path:        "/toolsets/{toolset_id}",
		Summary:     "Get a toolset",
		Description: "Returns the toolset and counts one view. An unrenderable author is NOT_FOUND. viewer is null for an anonymous caller.",
		Tags:        []string{tagToolsets},
		Responses: problemResponses(map[int]string{
			404: "NOT_FOUND when the toolset does not exist or its author is not renderable.",
		}),
	}), svc.getToolset)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "getToolsetSource",
		Method:      http.MethodGet,
		Path:        "/toolsets/{toolset_id}/source",
		Summary:     "Get a toolset's edit source",
		Description: "Returns the stored Markdown of the description. Needs viewer.can_edit. " + bearerNote + activeCheck,
		Tags:        []string{tagToolsets},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without can_edit; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the toolset does not exist or its author is not renderable.",
		}),
	}), svc.getToolsetSource)

	huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
		OperationID:   "createToolset",
		Method:        http.MethodPost,
		Path:          "/toolsets",
		Summary:       "Create a toolset",
		DefaultStatus: http.StatusCreated,
		Description: "Creates a toolset and returns it. Idempotency-Key is required. " + activeCheck +
			"name is length-checked on the raw value; only whitespace is TOO_SHORT. " +
			"type, language, platform and version are required closed vocabularies. " +
			"Location is the new toolset's path.",
		Tags: []string{tagToolsets},
		Responses: problemResponses(map[int]string{
			400: "INVALID_PARAMETER when Idempotency-Key is missing or malformed.",
			403: "ACCOUNT_BANNED.",
			409: "IDEMPOTENCY_KEY_REUSED or IDEMPOTENCY_REQUEST_IN_PROGRESS.",
			422: "VALIDATION_FAILED or CONTENT_REJECTED.",
		}),
	})), svc.createToolset)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "updateToolset",
		Method:      http.MethodPatch,
		Path:        "/toolsets/{toolset_id}",
		Summary:     "Edit a toolset",
		Description: "Changes the fields that are sent and returns the toolset. Needs can_edit. " + bearerNote + activeCheck +
			"Only name, content_markdown and aliases, when sent, go through the trust-and-safety check. " +
			"aliases and homepage_urls, when present, replace the whole set.",
		Tags: []string{tagToolsets},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without can_edit; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the toolset does not exist or its author is not renderable.",
			409: "ALREADY_EXISTS when an alias collides.",
			422: "VALIDATION_FAILED or CONTENT_REJECTED.",
		}),
	}), svc.updateToolset)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID:   "deleteToolset",
		Method:        http.MethodDelete,
		Path:          "/toolsets/{toolset_id}",
		Summary:       "Delete a toolset",
		DefaultStatus: http.StatusNoContent,
		Description:   "Deletes a toolset and its resources, aliases, contributors, ratings and uploads. Needs can_delete. " + bearerNote + activeCheck + "Object-store failure is SERVICE_UNAVAILABLE and nothing is deleted.",
		Tags:          []string{tagToolsets},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without can_delete; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the toolset does not exist or its author is not renderable.",
		}),
	}), svc.deleteToolset)
}

func registerResources(api huma.API, svc *Service) {
	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "getToolsetResource",
		Method:      http.MethodGet,
		Path:        "/toolsets/{toolset_id}/resources/{resource_id}",
		Summary:     "Get a toolset resource",
		Description: "Returns a resource without download secrets. A resource that does not belong to the toolset, or whose poster or the toolset's author is not renderable, is NOT_FOUND.",
		Tags:        []string{tagToolsets},
		Responses: problemResponses(map[int]string{
			404: "NOT_FOUND when the resource is not visible at this path.",
		}),
	}), svc.getToolsetResource)

	huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
		OperationID:   "createToolsetResource",
		Method:        http.MethodPost,
		Path:          "/toolsets/{toolset_id}/resources",
		Summary:       "Add a resource to a toolset",
		DefaultStatus: http.StatusCreated,
		Description: "Adds a resource. Any signed-in user may add a resource to another user's toolset and becomes a contributor. " +
			"Idempotency-Key is required. " + activeCheck +
			"file needs a completed artifact_id of the caller on this toolset; link needs url and size_label.",
		Tags: []string{tagToolsets},
		Responses: problemResponses(map[int]string{
			400: "INVALID_PARAMETER when Idempotency-Key is missing or malformed.",
			403: "ACCOUNT_BANNED.",
			404: "NOT_FOUND when the toolset does not exist or its author is not renderable.",
			409: "ALREADY_EXISTS when the artifact is already bound or the URL is taken; IDEMPOTENCY_KEY_REUSED or IDEMPOTENCY_REQUEST_IN_PROGRESS.",
			422: "VALIDATION_FAILED or CONTENT_REJECTED.",
		}),
	})), svc.createToolsetResource)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "updateToolsetResource",
		Method:      http.MethodPatch,
		Path:        "/toolsets/{toolset_id}/resources/{resource_id}",
		Summary:     "Edit a toolset resource",
		Description: "Changes the fields that are sent. Needs can_edit. " + bearerNote + activeCheck +
			"A file resource cannot change url, extraction_code or size_label.",
		Tags: []string{tagToolsets},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without can_edit; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the resource is not visible at this path.",
			409: "ALREADY_EXISTS when the new URL is taken.",
			422: "VALIDATION_FAILED or CONTENT_REJECTED.",
		}),
	}), svc.updateToolsetResource)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID:   "deleteToolsetResource",
		Method:        http.MethodDelete,
		Path:          "/toolsets/{toolset_id}/resources/{resource_id}",
		Summary:       "Delete a toolset resource",
		DefaultStatus: http.StatusNoContent,
		Description:   "Deletes a resource. Needs can_delete. " + bearerNote + activeCheck + "Object-store failure is SERVICE_UNAVAILABLE and the row stays.",
		Tags:          []string{tagToolsets},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without can_delete; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the resource is not visible at this path.",
		}),
	}), svc.deleteToolsetResource)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "createToolsetDownload",
		Method:      http.MethodPost,
		Path:        "/toolsets/{toolset_id}/resources/{resource_id}/downloads",
		Summary:     "Issue download secrets for a resource",
		Description: "Returns the download URL and codes, and counts one download. Anonymous callers may use it. A file resource's URL is a presigned artifact URL; failure is SERVICE_UNAVAILABLE.",
		Tags:        []string{tagToolsets},
		Responses: problemResponses(map[int]string{
			404: "NOT_FOUND when the resource is not visible at this path.",
		}),
	}), svc.createToolsetDownload)
}

func registerUploads(api huma.API, svc *Service) {
	huma.Register(api, v1.IdempotencyOptional(v1.Required(huma.Operation{
		OperationID:   "createToolsetUpload",
		Method:        http.MethodPost,
		Path:          "/toolsets/{toolset_id}/uploads",
		Summary:       "Start a toolset file upload",
		DefaultStatus: http.StatusCreated,
		Description: "Creates an upload session for a .7z, .zip or .rar file. " + activeCheck +
			"Over the daily quota is QUOTA_EXCEEDED with Retry-After until the next daily reset.",
		Tags: []string{tagToolsets},
		Responses: problemResponses(map[int]string{
			403: "ACCOUNT_BANNED.",
			404: "NOT_FOUND when the toolset does not exist or its author is not renderable.",
			409: "IDEMPOTENCY_KEY_REUSED or IDEMPOTENCY_REQUEST_IN_PROGRESS.",
			422: "VALIDATION_FAILED.",
			429: "QUOTA_EXCEEDED when the daily upload quota is exhausted.",
		}),
	})), svc.createToolsetUpload)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "getToolsetUpload",
		Method:      http.MethodGet,
		Path:        "/toolsets/{toolset_id}/uploads/{upload_id}",
		Summary:     "Resume a toolset file upload",
		Description: "Returns the upload session with fresh part URLs. Only the owner can read it. " + activeCheck,
		Tags:        []string{tagToolsets},
		Responses: problemResponses(map[int]string{
			403: "ACCOUNT_BANNED.",
			404: "NOT_FOUND when the upload is not the caller's session on this toolset.",
		}),
	}), svc.getToolsetUpload)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "updateToolsetUpload",
		Method:      http.MethodPatch,
		Path:        "/toolsets/{toolset_id}/uploads/{upload_id}",
		Summary:     "Complete a toolset file upload",
		Description: "Marks the upload completed. state must be completed. " + activeCheck +
			"completed_at is written once, in the same transaction as the daily quota.",
		Tags: []string{tagToolsets},
		Responses: problemResponses(map[int]string{
			403: "ACCOUNT_BANNED.",
			404: "NOT_FOUND when the upload is not the caller's session on this toolset.",
			409: "INVALID_STATE_TRANSITION when state is not completed.",
		}),
	}), svc.updateToolsetUpload)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID:   "deleteToolsetUpload",
		Method:        http.MethodDelete,
		Path:          "/toolsets/{toolset_id}/uploads/{upload_id}",
		Summary:       "Abort a toolset file upload",
		DefaultStatus: http.StatusNoContent,
		Description:   "Aborts a pending upload. A completed upload is INVALID_STATE_TRANSITION. Object-store failure is SERVICE_UNAVAILABLE and the row stays. " + activeCheck,
		Tags:          []string{tagToolsets},
		Responses: problemResponses(map[int]string{
			403: "ACCOUNT_BANNED.",
			404: "NOT_FOUND when the upload is not the caller's session on this toolset.",
			409: "INVALID_STATE_TRANSITION when the upload is already completed.",
		}),
	}), svc.deleteToolsetUpload)
}

func registerPracticality(api huma.API, svc *Service) {
	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "putToolsetPracticality",
		Method:      http.MethodPut,
		Path:        "/toolsets/{toolset_id}/practicality",
		Summary:     "Rate a toolset's practicality",
		Description: "Sets the caller's 1–5 rating. Rating again with the same value changes nothing. " + activeCheck,
		Tags:        []string{tagToolsets},
		Responses: problemResponses(map[int]string{
			403: "ACCOUNT_BANNED.",
			404: "NOT_FOUND when the toolset does not exist or its author is not renderable.",
			422: "VALIDATION_FAILED.",
		}),
	}), svc.putToolsetPracticality)
}
