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
	updateLogsTag = "update-logs"
	todosTag      = "todos"
)

func Register(svc *Service) func(huma.API) {
	return func(api huma.API) {
		registerUpdateLogs(api, svc)
		registerTodos(api, svc)
	}
}

func registerUpdateLogs(api huma.API, svc *Service) {
	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "listUpdateLogs",
		Method:      http.MethodGet,
		Path:        "/update-logs",
		Summary:     "List update log entries",
		Description: "Lists the site's changelog as a cursor page, newest first, ties broken by descending id. There is one sort, no sort parameter and no filter.",
		Tags:        []string{updateLogsTag},
		Responses: problemResponses(map[int]string{
			400: "INVALID_CURSOR or LIMIT_TOO_LARGE.",
		}),
	}), svc.listUpdateLogs)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "getUpdateLog",
		Method:      http.MethodGet,
		Path:        "/update-logs/{update_log_id}",
		Summary:     "Get an update log entry",
		Description: "Returns one changelog entry.",
		Tags:        []string{updateLogsTag},
		Responses: problemResponses(map[int]string{
			404: "NOT_FOUND when the entry does not exist.",
		}),
	}), svc.getUpdateLog)

	huma.Register(api, v1.IdempotencyOptional(v1.Required(huma.Operation{
		OperationID:   "createUpdateLog",
		Method:        http.MethodPost,
		Path:          "/update-logs",
		Summary:       "Create an update log entry",
		DefaultStatus: http.StatusCreated,
		Description: "Creates a changelog entry and returns it. Needs update_log.create, which a Bearer request never carries. " +
			"The caller is checked against the account service's current record first: a banned account is ACCOUNT_BANNED, and a failure of that lookup is SERVICE_UNAVAILABLE with nothing written. " +
			"release_version is trimmed after its length check; text is stored after NormalizeStoredContent, which turns a pasted image-service URL into an /image/<hash> token. Either one blank is VALIDATION_FAILED TOO_SHORT. " +
			"There is no trust-and-safety check. Location is the canonical path of the new entry.",
		Tags: []string{updateLogsTag},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without update_log.create; ACCOUNT_BANNED.",
			422: "VALIDATION_FAILED when release_version or text is blank or too long, or change_type is not in the vocabulary.",
		}),
	})), svc.createUpdateLog)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "updateUpdateLog",
		Method:      http.MethodPatch,
		Path:        "/update-logs/{update_log_id}",
		Summary:     "Edit an update log entry",
		Description: "Changes the fields that are sent and returns the entry. Needs update_log.edit, checked before the entry is looked up. " +
			"Fields are checked as in createUpdateLog. An empty object writes nothing and returns the entry.",
		Tags: []string{updateLogsTag},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without update_log.edit; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the entry does not exist.",
			422: "VALIDATION_FAILED when a sent field is blank, too long, or outside its vocabulary.",
		}),
	}), svc.updateUpdateLog)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID:   "deleteUpdateLog",
		Method:        http.MethodDelete,
		Path:          "/update-logs/{update_log_id}",
		Summary:       "Delete an update log entry",
		DefaultStatus: http.StatusNoContent,
		Description:   "Deletes a changelog entry and its home-feed card. Needs update_log.delete, checked before the entry is looked up.",
		Tags:          []string{updateLogsTag},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without update_log.delete; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the entry does not exist.",
		}),
	}), svc.deleteUpdateLog)
}

func registerTodos(api huma.API, svc *Service) {
	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "listTodos",
		Method:      http.MethodGet,
		Path:        "/todos",
		Summary:     "List the todo board",
		Description: "Lists tasks as a cursor page, newest first, ties broken by descending id. There is one sort and no sort parameter. " +
			"state keeps only tasks in that state; absent lists every state. The cursor is bound to state; reusing it with another state is INVALID_CURSOR. " +
			"include_total=true adds total, counted under the same state filter as items. Tasks by banned authors are listed like any other.",
		Tags: []string{todosTag},
		Responses: problemResponses(map[int]string{
			400: "INVALID_CURSOR, LIMIT_TOO_LARGE, or UNKNOWN_ENUM_VALUE.",
		}),
	}), svc.listTodos)

	huma.Register(api, v1.Optional(huma.Operation{
		OperationID: "getTodo",
		Method:      http.MethodGet,
		Path:        "/todos/{todo_id}",
		Summary:     "Get a todo",
		Description: "Returns one task.",
		Tags:        []string{todosTag},
		Responses: problemResponses(map[int]string{
			404: "NOT_FOUND when the task does not exist.",
		}),
	}), svc.getTodo)

	huma.Register(api, v1.IdempotencyRequired(v1.Required(huma.Operation{
		OperationID:   "createTodo",
		Method:        http.MethodPost,
		Path:          "/todos",
		Summary:       "Open a todo",
		DefaultStatus: http.StatusCreated,
		Description: "Opens a task on the board and returns it. Any signed-in user may; Idempotency-Key is required. A new task is always pending with no claimer. " +
			"The caller is checked against the account service's current record first: a banned account is ACCOUNT_BANNED, and a failure of that lookup is SERVICE_UNAVAILABLE with nothing written. " +
			"text is length-checked on the raw value and stored after NormalizeStoredContent; a value that is then blank is VALIDATION_FAILED TOO_SHORT. " +
			"text goes through the trust-and-safety check: a refusal is CONTENT_REJECTED with nothing written, a hold is written. Location is the canonical path of the new task.",
		Tags: []string{todosTag},
		Responses: problemResponses(map[int]string{
			403: "ACCOUNT_BANNED.",
			409: "IDEMPOTENCY_KEY_REUSED or IDEMPOTENCY_REQUEST_IN_PROGRESS.",
			422: "VALIDATION_FAILED when text is blank or too long or project is outside its vocabulary; CONTENT_REJECTED when the trust-and-safety check refuses text.",
		}),
	})), svc.createTodo)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "updateTodo",
		Method:      http.MethodPatch,
		Path:        "/todos/{todo_id}",
		Summary:     "Edit a todo or move it to another state",
		Description: "With state, moves the task; state is never sent with another field (VALIDATION_FAILED INCONSISTENT_WITH at /state). The transitions are: " +
			"pending to in_progress (claim; update_log.edit; the caller becomes the claimer), " +
			"pending to discarded (the author), " +
			"in_progress to done (the claimer or update_log.edit; sets completed_at), " +
			"in_progress to discarded (the claimer or update_log.edit), " +
			"in_progress to pending (release; the claimer only; clears the claimer), " +
			"discarded to pending (reopen; update_log.reopen; clears the claimer). " +
			"done is final. Any other pair, including the state the task is already in, is INVALID_STATE_TRANSITION, checked before permissions; a listed pair the caller may not make is PERMISSION_REQUIRED. " +
			"The write is guarded on the state that was read, so losing a race to another caller is INVALID_STATE_TRANSITION too. " +
			"Without state, changes project and text: the author only, and only while the task is pending or in_progress (otherwise INVALID_STATE_TRANSITION, checked first). " +
			"text is checked as in createTodo; the trust-and-safety check runs only when the stored text changes. An empty object writes nothing. " +
			"Every flag in viewer is exactly the gate this operation applies. Update_log permissions are never carried by a Bearer request.",
		Tags: []string{todosTag},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED when the caller may not make this transition or edit; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the task does not exist.",
			409: "INVALID_STATE_TRANSITION when the transition is not in the state machine, the task is done or discarded and cannot be edited, or another caller moved it first.",
			422: "VALIDATION_FAILED when state is sent with another field, text is blank or too long, or a value is outside its vocabulary; CONTENT_REJECTED when the trust-and-safety check refuses changed text.",
		}),
	}), svc.updateTodo)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID:   "deleteTodo",
		Method:        http.MethodDelete,
		Path:          "/todos/{todo_id}",
		Summary:       "Delete a todo",
		DefaultStatus: http.StatusNoContent,
		Description:   "Deletes a task and its home-feed card, in any state. Needs update_log.delete, checked before the task is looked up.",
		Tags:          []string{todosTag},
		Responses: problemResponses(map[int]string{
			403: "PERMISSION_REQUIRED without update_log.delete; ACCOUNT_BANNED.",
			404: "NOT_FOUND when the task does not exist.",
		}),
	}), svc.deleteTodo)
}
