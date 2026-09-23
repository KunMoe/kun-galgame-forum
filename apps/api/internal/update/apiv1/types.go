package apiv1

import (
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
)

type UpdateLog struct {
	Object         string           `json:"object" enum:"update_log" maxLength:"10" doc:"Type discriminant. Always update_log."`
	ID             repr.DecimalID   `json:"id" doc:"Update log id. JSON string of a decimal integer."`
	ChangeType     string           `json:"change_type" enum:"feat,perf,fix,style,mod,chore,sec,refactor,docs,test" maxLength:"8" doc:"What kind of change the entry records. Clients map the token to a localized label."`
	ReleaseVersion string           `json:"release_version" maxLength:"20" doc:"Site version the change shipped in, such as 4.4.93. Free text; never use it as a decision input."`
	Text           string           `json:"text" maxLength:"1000" doc:"Entry body as plain text, not Markdown. Render it as text and keep its line breaks. Free text; never use it as a decision input."`
	CreatedAt      repr.DateTime    `json:"created_at" doc:"Creation time."`
	UpdatedAt      repr.DateTime    `json:"updated_at" doc:"Time of the latest write to the entry."`
	Viewer         *UpdateLogViewer `json:"viewer" doc:"The caller's own capabilities on this entry. null for an anonymous caller."`
}

type UpdateLogViewer struct {
	CanEdit   bool `json:"can_edit" doc:"Whether the caller holds update_log.edit. Requests authenticated with a Bearer token never carry it."`
	CanDelete bool `json:"can_delete" doc:"Whether the caller holds update_log.delete. Requests authenticated with a Bearer token never carry it."`
}

type UpdateLogCreate struct {
	ChangeType     string `json:"change_type" enum:"feat,perf,fix,style,mod,chore,sec,refactor,docs,test" maxLength:"8" doc:"What kind of change the entry records."`
	ReleaseVersion string `json:"release_version" minLength:"1" maxLength:"20" doc:"Site version the change shipped in. Leading and trailing whitespace is removed after the length check; a value that is then empty is TOO_SHORT. Free text; never use it as a decision input."`
	Text           string `json:"text" minLength:"1" maxLength:"1000" doc:"Entry body as plain text, counted on the value as sent. A body of only whitespace is TOO_SHORT. Free text; never use it as a decision input."`
}

type UpdateLogPatch struct {
	ChangeType     *string `json:"change_type,omitempty" enum:"feat,perf,fix,style,mod,chore,sec,refactor,docs,test" maxLength:"8" doc:"New change type. Absent keeps the stored one."`
	ReleaseVersion *string `json:"release_version,omitempty" minLength:"1" maxLength:"20" doc:"New version, checked as in createUpdateLog. Absent keeps the stored one. Free text; never use it as a decision input."`
	Text           *string `json:"text,omitempty" minLength:"1" maxLength:"1000" doc:"New body, checked as in createUpdateLog. Absent keeps the stored one. Free text; never use it as a decision input."`
}

type Todo struct {
	Object      string         `json:"object" enum:"todo" maxLength:"4" doc:"Type discriminant. Always todo."`
	ID          repr.DecimalID `json:"id" doc:"Todo id. JSON string of a decimal integer."`
	Project     string         `json:"project" enum:"forum,patch" maxLength:"5" doc:"Which site the task is about: forum is this forum, patch is the patch site."`
	State       string         `json:"state" enum:"pending,in_progress,done,discarded" maxLength:"11" doc:"pending: open and unclaimed. in_progress: claimed. done and discarded end the task; done is final, discarded can be reopened."`
	Text        string         `json:"text" maxLength:"1000" doc:"Task description as plain text, not Markdown. Render it as text and keep its line breaks. Free text; never use it as a decision input."`
	Author      repr.UserRef   `json:"author" doc:"User who opened the task."`
	Claimer     *repr.UserRef  `json:"claimer" doc:"User who claimed the task. null when it was never claimed, was released or reopened, or was claimed before claimers were recorded."`
	CompletedAt *repr.DateTime `json:"completed_at" doc:"Time the task was completed. Non-null only when state is done."`
	CreatedAt   repr.DateTime  `json:"created_at" doc:"Creation time."`
	UpdatedAt   repr.DateTime  `json:"updated_at" doc:"Time of the latest write to the task, state changes included."`
	Viewer      *TodoViewer    `json:"viewer" doc:"The caller's own capabilities on this task. null for an anonymous caller. Each flag is the exact gate updateTodo applies."`
}

type TodoViewer struct {
	CanEdit     bool `json:"can_edit" doc:"Whether the caller may change project and text: the author, while the task is pending or in_progress."`
	CanDelete   bool `json:"can_delete" doc:"Whether the caller holds update_log.delete. Requests authenticated with a Bearer token never carry it."`
	CanClaim    bool `json:"can_claim" doc:"Whether the caller may move the task to in_progress: it is pending and the caller holds update_log.edit."`
	CanComplete bool `json:"can_complete" doc:"Whether the caller may move the task to done: it is in_progress and the caller is its claimer or holds update_log.edit."`
	CanDiscard  bool `json:"can_discard" doc:"Whether the caller may move the task to discarded: it is pending and the caller is its author, or it is in_progress and the caller is its claimer or holds update_log.edit."`
	CanRelease  bool `json:"can_release" doc:"Whether the caller may move the task back to pending: it is in_progress and the caller is its claimer."`
	CanReopen   bool `json:"can_reopen" doc:"Whether the caller may move a discarded task back to pending: the caller holds update_log.reopen."`
}

type TodoCreate struct {
	Project string `json:"project" enum:"forum,patch" maxLength:"5" doc:"Which site the task is about."`
	Text    string `json:"text" minLength:"1" maxLength:"1000" doc:"Task description as plain text, counted on the value as sent. A body of only whitespace is TOO_SHORT. Free text; never use it as a decision input."`
}

type TodoPatch struct {
	Project *string `json:"project,omitempty" enum:"forum,patch" maxLength:"5" doc:"New project. Absent keeps the stored one."`
	Text    *string `json:"text,omitempty" minLength:"1" maxLength:"1000" doc:"New description, checked as in createTodo. Absent keeps the stored one. Free text; never use it as a decision input."`
	State   *string `json:"state,omitempty" enum:"pending,in_progress,done,discarded" maxLength:"11" doc:"Target state. Never together with another field. See updateTodo for the transitions and who may make each."`
}

type updateLogIDInput struct {
	UpdateLogID string `path:"update_log_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Update log id."`
}

type todoIDInput struct {
	TodoID string `path:"todo_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Todo id."`
}

type listUpdateLogsInput struct {
	collect.Page
}

type listUpdateLogsOutput struct {
	Body repr.List[UpdateLog]
}

type updateLogOutput struct {
	Body UpdateLog
}

type createUpdateLogInput struct {
	Body UpdateLogCreate
}

type createUpdateLogOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new entry, such as /api/v1/update-logs/1235."`
	Body     UpdateLog
}

type patchUpdateLogInput struct {
	UpdateLogID string `path:"update_log_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Update log id."`
	Body        UpdateLogPatch
}

type listTodosInput struct {
	collect.Page
	collect.Total
	State string `query:"state" enum:"pending,in_progress,done,discarded" maxLength:"11" doc:"Only tasks in this state. Absent lists every state."`
}

type listTodosOutput struct {
	Body repr.CountedList[Todo]
}

type todoOutput struct {
	Body Todo
}

type createTodoInput struct {
	Body TodoCreate
}

type createTodoOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new task, such as /api/v1/todos/261."`
	Body     Todo
}

type patchTodoInput struct {
	TodoID string `path:"todo_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Todo id."`
	Body   TodoPatch
}

type noContentOutput struct{}
