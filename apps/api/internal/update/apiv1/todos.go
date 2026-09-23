package apiv1

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	adminModel "kun-galgame-api/internal/admin/model"
	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
)

func (s *Service) listTodos(ctx context.Context, in *listTodosInput) (*listTodosOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	var status *int
	if in.State != "" {
		st, ok := stateFromToken(in.State)
		if !ok {
			return nil, problem.New(problem.CodeUnknownEnumValue, "state is not in this collection's vocabulary.",
				problem.AtParameter("state", problem.ReasonUnknownValue, "use a declared state", nil))
		}
		status = &st
	}
	fp := collect.Fingerprint("todos", in.State)
	keys, p := collect.DecodeCursor(in.Cursor, listSort, fp)
	if p != nil {
		return nil, p
	}
	pos, p := parsePos(keys)
	if p != nil {
		return nil, p
	}
	limit := limitOf(in.Limit)
	rows, err := s.store.ListTodos(status, pos, limit+1)
	if err != nil {
		return nil, problem.Internal(err)
	}
	var next *string
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[limit-1]
		next = nextCursor(fp, last.CreatedAt, last.ID)
	}
	var total *int
	if in.IncludeTotal {
		n, err := s.store.CountTodos(status)
		if err != nil {
			return nil, problem.Internal(err)
		}
		total = &n
	}
	items, p := s.renderTodos(ctx, rows, v1.User(ctx))
	if p != nil {
		return nil, p
	}
	return &listTodosOutput{Body: repr.NewCountedList(items, next, total)}, nil
}

func (s *Service) findTodo(raw string) (*adminModel.Todo, *problem.Problem) {
	id, ok := repr.ParseID(repr.DecimalID(raw))
	if !ok {
		return nil, notFound()
	}
	row, err := s.store.FindTodo(id)
	if err != nil {
		return nil, storeProblem(err)
	}
	return row, nil
}

func (s *Service) respondTodo(ctx context.Context, id int, viewer *middleware.UserInfo) (*todoOutput, error) {
	row, err := s.store.FindTodo(id)
	if err != nil {
		return nil, storeProblem(err)
	}
	item, p := s.renderTodo(ctx, row, viewer)
	if p != nil {
		return nil, p
	}
	return &todoOutput{Body: *item}, nil
}

func (s *Service) getTodo(ctx context.Context, in *todoIDInput) (*todoOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	row, p := s.findTodo(in.TodoID)
	if p != nil {
		return nil, p
	}
	item, p := s.renderTodo(ctx, row, v1.User(ctx))
	if p != nil {
		return nil, p
	}
	return &todoOutput{Body: *item}, nil
}

// screenText runs the trust gate on text the caller is submitting now; a hold
// is logged and still written.
func (s *Service) screenText(ctx context.Context, text string, authorID int) (string, []string, *problem.Problem) {
	id := int64(authorID)
	decision, matched := s.check.Decision(ctx, text, &id)
	if decision == gate.DecisionDeny {
		return decision, matched, contentRejected()
	}
	return decision, matched, nil
}

func (s *Service) afterWrite(decision string, matched []string, todoID, authorID int, text string) {
	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindTodo,
			"subject_id", todoID, "author_id", authorID, "matched", matched)
	}
	s.scan.ScanBg(gate.SubjectKindTodo, strconv.Itoa(todoID), text, int64(authorID))
}

func (s *Service) createTodo(ctx context.Context, in *createTodoInput) (*createTodoOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	text := markdown.NormalizeStoredContent(in.Body.Text)
	if blank(text) {
		return nil, tooShort("/text")
	}
	decision, matched, p := s.screenText(ctx, text, user.ID)
	if p != nil {
		return nil, p
	}
	row := adminModel.Todo{
		Type:    in.Body.Project,
		Status:  adminModel.TodoStatusPending,
		Content: text,
		UserID:  user.ID,
	}
	if err := s.store.CreateTodo(&row); err != nil {
		return nil, problem.Internal(err)
	}
	s.afterWrite(decision, matched, row.ID, user.ID, text)
	item, p := s.renderTodo(ctx, &row, user)
	if p != nil {
		return nil, p
	}
	return &createTodoOutput{
		Location: v1.Prefix + "/todos/" + strconv.Itoa(row.ID),
		Body:     *item,
	}, nil
}

func (s *Service) updateTodo(ctx context.Context, in *patchTodoInput) (*todoOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	row, p := s.findTodo(in.TodoID)
	if p != nil {
		return nil, p
	}
	if in.Body.State != nil {
		if in.Body.Project != nil || in.Body.Text != nil {
			other := "project"
			if in.Body.Project == nil {
				other = "text"
			}
			return nil, validationFailed(problem.AtPointer("/state", problem.ReasonInconsistentWith,
				"state cannot be sent together with "+other, nil))
		}
		return s.moveTodo(ctx, row, user, *in.Body.State)
	}
	return s.editTodo(ctx, row, user, in.Body)
}

func (s *Service) moveTodo(ctx context.Context, row *adminModel.Todo, user *middleware.UserInfo, target string) (*todoOutput, error) {
	to, ok := stateFromToken(target)
	if !ok {
		return nil, invalidTransition(row.Status, target)
	}
	allowed, ok := transitions[transition{row.Status, to}]
	if !ok {
		return nil, invalidTransition(row.Status, target)
	}
	if !allowed(row, user) {
		return nil, permissionRequired("The caller may not move this todo to " + target + ".")
	}
	fields := map[string]any{}
	switch to {
	case adminModel.TodoStatusClaimed:
		fields["claimed_user_id"] = user.ID
	case adminModel.TodoStatusDone:
		fields["completed_time"] = time.Now()
	case adminModel.TodoStatusDiscarded:
		fields["completed_time"] = nil
	case adminModel.TodoStatusPending:
		fields["claimed_user_id"] = nil
		fields["completed_time"] = nil
	}
	moved, err := s.store.TransitionTodo(row.ID, row.Status, to, fields)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if !moved {
		fresh, err := s.store.FindTodo(row.ID)
		if err != nil {
			return nil, storeProblem(err)
		}
		return nil, invalidTransition(fresh.Status, target)
	}
	return s.respondTodo(ctx, row.ID, user)
}

func (s *Service) editTodo(ctx context.Context, row *adminModel.Todo, user *middleware.UserInfo, body TodoPatch) (*todoOutput, error) {
	if !isEditable(row.Status) {
		return nil, notEditable(row.Status)
	}
	if !isAuthor(row, user) {
		return nil, permissionRequired("Only the todo's author may edit it.")
	}
	fields := map[string]any{}
	if body.Project != nil {
		fields["type"] = *body.Project
	}
	var decision string
	var matched []string
	screened := ""
	if body.Text != nil {
		text := markdown.NormalizeStoredContent(*body.Text)
		if blank(text) {
			return nil, tooShort("/text")
		}
		if text != row.Content {
			d, m, p := s.screenText(ctx, text, user.ID)
			if p != nil {
				return nil, p
			}
			decision, matched, screened = d, m, text
		}
		fields["content"] = text
	}
	if len(fields) > 0 {
		moved, err := s.store.PatchTodo(row.ID, []int{adminModel.TodoStatusPending, adminModel.TodoStatusClaimed}, fields)
		if err != nil {
			return nil, problem.Internal(err)
		}
		if !moved {
			fresh, err := s.store.FindTodo(row.ID)
			if err != nil {
				return nil, storeProblem(err)
			}
			return nil, notEditable(fresh.Status)
		}
		if screened != "" {
			s.afterWrite(decision, matched, row.ID, user.ID, screened)
		}
	}
	return s.respondTodo(ctx, row.ID, user)
}

func (s *Service) deleteTodo(ctx context.Context, in *todoIDInput) (*noContentOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	if _, p := s.staff(ctx, perm.UpdateLogDelete, "Deleting a todo needs update_log.delete."); p != nil {
		return nil, p
	}
	id, ok := repr.ParseID(repr.DecimalID(in.TodoID))
	if !ok {
		return nil, notFound()
	}
	if err := s.store.DeleteTodo(id); err != nil {
		return nil, storeProblem(err)
	}
	return &noContentOutput{}, nil
}
