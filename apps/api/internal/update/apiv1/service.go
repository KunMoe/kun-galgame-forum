package apiv1

import (
	"context"
	"errors"
	"strings"

	adminModel "kun-galgame-api/internal/admin/model"
	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/internal/update/repository"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

var errUnconfigured = errors.New("apiv1 update: service is not configured")

type Service struct {
	store *repository.Store
	users *userclient.Client
	check *gate.CheckService
	scan  *gate.ScanService
	cdn   string
}

func New(store *repository.Store, users *userclient.Client, check *gate.CheckService, scan *gate.ScanService, cdn string) *Service {
	return &Service{store: store, users: users, check: check, scan: scan, cdn: cdn}
}

func (s *Service) ready() *problem.Problem {
	if s == nil || !s.store.Ready() || s.users == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

// requireActive checks the writer against OAuth's current record. A session
// only learns of a ban when its token is refreshed, so without this a banned
// user keeps writing until then.
func (s *Service) requireActive(ctx context.Context) (*middleware.UserInfo, *problem.Problem) {
	user := v1.User(ctx)
	if user == nil {
		return nil, problem.New(problem.CodeMissingCredential, "The request has no credentials.")
	}
	users, err := s.users.Users(ctx, []int{user.ID})
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	if u, ok := users[user.ID]; ok && !userclient.IsRenderable(u) {
		return nil, problem.New(problem.CodeAccountBanned, "The signed-in user's account is banned.")
	}
	return user, nil
}

// userRefs never fails the read: an account service outage renders every
// person with name null (K17 binds writes only).
func (s *Service) userRefs(ctx context.Context, ids []int) map[int]repr.UserRef {
	out := make(map[int]repr.UserRef, len(ids))
	users, _ := s.users.Users(ctx, ids)
	for _, id := range ids {
		if u, ok := users[id]; ok {
			out[id] = repr.NewUserRef(s.cdn, u)
		} else {
			out[id] = repr.DeletedUserRef(id)
		}
	}
	return out
}

func (s *Service) todoUserIDs(rows []adminModel.Todo) []int {
	ids := make([]int, 0, len(rows)*2)
	for _, t := range rows {
		ids = append(ids, t.UserID)
		if t.ClaimedUserID != nil {
			ids = append(ids, *t.ClaimedUserID)
		}
	}
	return ids
}

func (s *Service) renderTodos(ctx context.Context, rows []adminModel.Todo, viewer *middleware.UserInfo) ([]Todo, *problem.Problem) {
	refs := s.userRefs(ctx, s.todoUserIDs(rows))
	out := make([]Todo, 0, len(rows))
	for i := range rows {
		t := &rows[i]
		state, ok := stateToken(t.Status)
		if !ok {
			return nil, problem.Internal(errors.New("apiv1 update: todo has a status outside the state machine"))
		}
		item := Todo{
			Object:    "todo",
			ID:        repr.ID(t.ID),
			Project:   t.Type,
			State:     state,
			Text:      t.Content,
			Author:    refs[t.UserID],
			CreatedAt: repr.Timestamp(t.CreatedAt),
			UpdatedAt: repr.Timestamp(t.UpdatedAt),
			Viewer:    todoViewer(t, viewer),
		}
		if t.ClaimedUserID != nil {
			ref := refs[*t.ClaimedUserID]
			item.Claimer = &ref
		}
		if t.Status == adminModel.TodoStatusDone {
			item.CompletedAt = repr.TimestampPtr(t.CompletedTime)
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) renderTodo(ctx context.Context, t *adminModel.Todo, viewer *middleware.UserInfo) (*Todo, *problem.Problem) {
	items, p := s.renderTodos(ctx, []adminModel.Todo{*t}, viewer)
	if p != nil {
		return nil, p
	}
	return &items[0], nil
}

func renderUpdateLog(l *adminModel.UpdateLog, viewer *middleware.UserInfo) UpdateLog {
	out := UpdateLog{
		Object:         "update_log",
		ID:             repr.ID(l.ID),
		ChangeType:     l.Type,
		ReleaseVersion: l.Version,
		Text:           l.Content,
		CreatedAt:      repr.Timestamp(l.CreatedAt),
		UpdatedAt:      repr.Timestamp(l.UpdatedAt),
	}
	if viewer != nil {
		out.Viewer = &UpdateLogViewer{
			CanEdit:   viewer.Can(perm.UpdateLogEdit),
			CanDelete: viewer.Can(perm.UpdateLogDelete),
		}
	}
	return out
}

func blank(s string) bool {
	return strings.TrimSpace(s) == ""
}

func tooShort(pointer string) *problem.Problem {
	one := 1
	return validationFailed(problem.AtPointer(pointer, problem.ReasonTooShort, "must contain a non-whitespace character", &problem.FieldParams{MinLength: &one}))
}

func storeProblem(err error) *problem.Problem {
	if errors.Is(err, repository.ErrNotFound) {
		return notFound()
	}
	return problem.Internal(err)
}
