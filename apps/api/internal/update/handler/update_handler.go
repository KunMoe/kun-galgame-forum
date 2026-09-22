package handler

import (
	"log/slog"
	"strconv"

	adminModel "kun-galgame-api/internal/admin/model"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/internal/update/dto"
	"kun-galgame-api/internal/update/repository"
	"net/http"

	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/userclient"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type UpdateHandler struct {
	repo       *repository.UpdateRepository
	userClient *userclient.Client
	check      *gate.CheckService
	scan       *gate.ScanService
}

func NewUpdateHandler(
	repo *repository.UpdateRepository,
	userClient *userclient.Client,
	check *gate.CheckService,
	scan *gate.ScanService,
) *UpdateHandler {
	return &UpdateHandler{repo: repo, userClient: userClient, check: check, scan: scan}
}

// The guarded UPDATEs answer moved=false when the row left the state the
// handler had just read, which is a race, not a rule the caller broke.
var errStaleTodo = errors.New(errors.CodeBiz, "该待办的状态刚刚发生了变化, 请刷新后重试", http.StatusConflict)

func (h *UpdateHandler) GetHistory(c fiber.Ctx) error {
	var req dto.ListQuery
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	logs := h.repo.FindHistoryPaginated(req.Page, req.Limit)
	total := h.repo.CountHistory()

	return response.OK(c, fiber.Map{
		"updates": logs,
		"total":   total,
	})
}

func (h *UpdateHandler) CreateHistory(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.CreateHistoryRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	req.Content = markdown.NormalizeStoredContent(req.Content)

	log := adminModel.UpdateLog{
		Type: req.Type, Version: req.Version,
		Content: req.Content,
		UserID:  user.ID,
	}
	if err := h.repo.CreateHistory(&log); err != nil {
		return response.Error(c, errors.ErrInternal("创建更新日志失败"))
	}
	return response.OKMessage(c, "更新日志已创建")
}

func (h *UpdateHandler) UpdateHistory(c fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.UpdateHistoryRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	req.Content = markdown.NormalizeStoredContent(req.Content)

	fields := map[string]any{
		"type":    req.Type,
		"version": req.Version,
		"content": req.Content,
	}
	if err := h.repo.UpdateHistory(req.ID, fields); err != nil {
		return response.Error(c, errors.ErrInternal("更新日志失败"))
	}
	return response.OKMessage(c, "更新日志已更新")
}

func (h *UpdateHandler) DeleteHistory(c fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.DeleteHistoryRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if err := h.repo.DeleteHistory(req.ID); err != nil {
		return response.Error(c, errors.ErrInternal("删除更新日志失败"))
	}
	return response.OKMessage(c, "更新日志已删除")
}

func (h *UpdateHandler) GetTodos(c fiber.Ctx) error {
	var req dto.TodoListQuery
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	todos := h.repo.FindTodosPaginated(req.Page, req.Limit, req.Status)
	total := h.repo.CountTodos(req.Status)

	uids := userclient.CollectIDs(todos, func(t adminModel.Todo) int { return t.UserID })
	for _, t := range todos {
		if t.ClaimedUserID != nil {
			uids = append(uids, *t.ClaimedUserID)
		}
	}
	userMap := h.userClient.Hydrate(c.Context(), uids)

	items := make([]dto.TodoItem, 0, len(todos))
	for _, t := range todos {
		u := userMap[t.UserID]
		item := dto.TodoItemOf(t, dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar})
		if t.ClaimedUserID != nil {
			if cu, ok := userMap[*t.ClaimedUserID]; ok {
				item.ClaimedUser = &dto.UserBrief{ID: cu.ID, Name: cu.Name, Avatar: cu.Avatar}
			}
		}
		items = append(items, item)
	}

	return response.OK(c, fiber.Map{
		"todos": items,
		"total": total,
	})
}

func (h *UpdateHandler) CreateTodo(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.CreateTodoRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	req.Content = markdown.NormalizeStoredContent(req.Content)

	authorID := int64(user.ID)
	decision, matched := h.check.Decision(c.Context(), req.Content, &authorID)
	if decision == gate.DecisionDeny {
		return response.Error(c, gate.ErrContentBlocked())
	}

	todo := adminModel.Todo{
		Type:    req.Type,
		Status:  adminModel.TodoStatusPending,
		Content: req.Content,
		UserID:  user.ID,
	}
	if err := h.repo.CreateTodo(&todo); err != nil {
		return response.Error(c, errors.ErrInternal("创建待办失败"))
	}
	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindTodo,
			"subject_id", todo.ID, "author_id", user.ID, "matched", matched)
	}
	h.scan.ScanBg(gate.SubjectKindTodo, strconv.Itoa(todo.ID), req.Content, authorID)
	return response.OKMessage(c, "待办已创建")
}

func (h *UpdateHandler) UpdateTodo(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.UpdateTodoRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	todo, err := h.repo.FindTodoByID(req.ID)
	if err != nil {
		return response.Error(c, errors.ErrNotFound("待办不存在"))
	}
	if todo.UserID != user.ID {
		return response.Error(c, errors.ErrForbidden("只有发起者可以编辑该待办"))
	}
	if todo.Status == adminModel.TodoStatusDone || todo.Status == adminModel.TodoStatusDiscarded {
		return response.Error(c, errors.ErrForbidden("已完成或已废弃的待办不可编辑"))
	}

	req.Content = markdown.NormalizeStoredContent(req.Content)

	authorID := int64(user.ID)
	decision, matched := h.check.Decision(c.Context(), req.Content, &authorID)
	if decision == gate.DecisionDeny {
		return response.Error(c, gate.ErrContentBlocked())
	}

	fields := map[string]any{
		"type":    req.Type,
		"content": req.Content,
	}
	if err := h.repo.UpdateTodo(req.ID, fields); err != nil {
		return response.Error(c, errors.ErrInternal("更新待办失败"))
	}
	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindTodo,
			"subject_id", todo.ID, "author_id", user.ID, "matched", matched)
	}
	h.scan.ScanBg(gate.SubjectKindTodo, strconv.Itoa(todo.ID), req.Content, authorID)
	return response.OKMessage(c, "待办已更新")
}

func (h *UpdateHandler) ClaimTodo(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.TodoActionRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	todo, err := h.repo.FindTodoByID(req.TodoID)
	if err != nil {
		return response.Error(c, errors.ErrNotFound("待办不存在"))
	}
	if todo.Status != adminModel.TodoStatusPending {
		return response.Error(c, errors.ErrForbidden("该待办已被认领或已结束"))
	}
	moved, err := h.repo.ClaimTodo(req.TodoID, user.ID)
	if err != nil {
		return response.Error(c, errors.ErrInternal("认领待办失败"))
	}
	if !moved {
		return response.Error(c, errStaleTodo)
	}
	return response.OKMessage(c, "已认领该待办")
}

// Ending a claimed todo — completing or discarding it — is the claimer's call or
// a board editor's. Only update_log.edit holders can claim in the first place
// (see setupRoutes), so this reads as "the claimer, or one of their peers".
func canEndClaimedTodo(todo *adminModel.Todo, user *middleware.UserInfo) bool {
	if todo.ClaimedUserID != nil && *todo.ClaimedUserID == user.ID {
		return true
	}
	return user.Can(perm.UpdateLogEdit)
}

func (h *UpdateHandler) CompleteTodo(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.TodoActionRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	todo, err := h.repo.FindTodoByID(req.TodoID)
	if err != nil {
		return response.Error(c, errors.ErrNotFound("待办不存在"))
	}
	if todo.Status != adminModel.TodoStatusClaimed {
		return response.Error(c, errors.ErrForbidden("该待办未处于进行中状态"))
	}
	if !canEndClaimedTodo(todo, user) {
		return response.Error(c, errors.ErrForbidden("只有认领者或看板编辑者可以完成该待办"))
	}
	moved, err := h.repo.CompleteTodo(req.TodoID)
	if err != nil {
		return response.Error(c, errors.ErrInternal("完成待办失败"))
	}
	if !moved {
		return response.Error(c, errStaleTodo)
	}
	return response.OKMessage(c, "待办已完成")
}

func (h *UpdateHandler) DiscardTodo(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.TodoActionRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	todo, err := h.repo.FindTodoByID(req.TodoID)
	if err != nil {
		return response.Error(c, errors.ErrNotFound("待办不存在"))
	}

	switch todo.Status {
	case adminModel.TodoStatusPending:
		if todo.UserID != user.ID {
			return response.Error(c, errors.ErrForbidden("只有发起者可以废弃该待办"))
		}
	case adminModel.TodoStatusClaimed:
		if !canEndClaimedTodo(todo, user) {
			return response.Error(c, errors.ErrForbidden("只有认领者或看板编辑者可以废弃该待办"))
		}
	default:
		return response.Error(c, errors.ErrForbidden("该待办不可废弃"))
	}

	moved, err := h.repo.DiscardTodo(req.TodoID, todo.Status)
	if err != nil {
		return response.Error(c, errors.ErrInternal("废弃待办失败"))
	}
	if !moved {
		return response.Error(c, errStaleTodo)
	}
	return response.OKMessage(c, "待办已废弃")
}

func (h *UpdateHandler) ReleaseTodo(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.TodoActionRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	todo, err := h.repo.FindTodoByID(req.TodoID)
	if err != nil {
		return response.Error(c, errors.ErrNotFound("待办不存在"))
	}
	if todo.Status != adminModel.TodoStatusClaimed {
		return response.Error(c, errors.ErrForbidden("该待办未处于进行中状态"))
	}
	if todo.ClaimedUserID == nil || *todo.ClaimedUserID != user.ID {
		return response.Error(c, errors.ErrForbidden("只有认领者可以放弃该待办"))
	}
	moved, err := h.repo.ReleaseTodo(req.TodoID)
	if err != nil {
		return response.Error(c, errors.ErrInternal("放弃待办失败"))
	}
	if !moved {
		return response.Error(c, errStaleTodo)
	}
	return response.OKMessage(c, "已放弃该待办, 任务回到待处理")
}

func (h *UpdateHandler) ReopenTodo(c fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.TodoActionRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	todo, err := h.repo.FindTodoByID(req.TodoID)
	if err != nil {
		return response.Error(c, errors.ErrNotFound("待办不存在"))
	}
	if todo.Status != adminModel.TodoStatusDiscarded {
		return response.Error(c, errors.ErrForbidden("该待办未处于已废弃状态"))
	}
	moved, err := h.repo.ReopenTodo(req.TodoID)
	if err != nil {
		return response.Error(c, errors.ErrInternal("重新启用待办失败"))
	}
	if !moved {
		return response.Error(c, errStaleTodo)
	}
	return response.OKMessage(c, "待办已重新启用, 回到待处理")
}

func (h *UpdateHandler) DeleteTodo(c fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}

	var req dto.DeleteTodoRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if err := h.repo.DeleteTodo(req.ID); err != nil {
		return response.Error(c, errors.ErrInternal("删除待办失败"))
	}
	return response.OKMessage(c, "待办已删除")
}
