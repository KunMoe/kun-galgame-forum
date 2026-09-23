package handler

import (
	"strconv"

	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

type SubmissionHandler struct {
	svc *service.SubmissionService
}

func NewSubmissionHandler(svc *service.SubmissionService) *SubmissionHandler {
	return &SubmissionHandler{svc: svc}
}

func submissionWorkID(c fiber.Ctx) (int, *errors.AppError) {
	workID, err := strconv.Atoi(c.Params("id"))
	if err != nil || workID <= 0 {
		return 0, errors.ErrBadRequest("无效的 Galgame ID")
	}
	return workID, nil
}

func (h *SubmissionHandler) Submit(c fiber.Ctx) error {
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	user := middleware.GetUser(c)
	if user == nil {
		return response.Error(c, errors.ErrAuthExpired())
	}
	var form service.SubmissionForm
	if err := c.Bind().Body(&form); err != nil {
		return response.Error(c, errors.ErrBadRequest("请求格式错误"))
	}
	res, appErr := h.svc.Submit(c.Context(), token, user.ID, &form)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, res)
}

func (h *SubmissionHandler) Resubmit(c fiber.Ctx) error {
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	workID, appErr := submissionWorkID(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	res, appErr := h.svc.Resubmit(c.Context(), token, workID)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, res)
}

func (h *SubmissionHandler) Withdraw(c fiber.Ctx) error {
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	workID, appErr := submissionWorkID(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	if _, appErr := h.svc.Withdraw(c.Context(), token, workID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "撤回成功")
}

func (h *SubmissionHandler) DeleteDraft(c fiber.Ctx) error {
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	workID, appErr := submissionWorkID(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.svc.DeleteDraft(c.Context(), token, workID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "删除成功")
}

func (h *SubmissionHandler) ListMine(c fiber.Ctx) error {
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	page, appErr := h.svc.ListMine(c.Context(), token, collectQuery(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

func (h *SubmissionHandler) ListAudit(c fiber.Ctx) error {
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	page, appErr := h.svc.ListAudit(c.Context(), token, collectQuery(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

func (h *SubmissionHandler) SearchWithPending(c fiber.Ctx) error {
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	page, appErr := h.svc.SearchWithPending(c.Context(), token, collectQuery(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}
