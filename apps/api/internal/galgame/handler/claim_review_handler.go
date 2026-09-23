package handler

import (
	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

type ClaimReviewHandler struct {
	svc *service.ClaimReviewService
}

func NewClaimReviewHandler(svc *service.ClaimReviewService) *ClaimReviewHandler {
	return &ClaimReviewHandler{svc: svc}
}

func (h *ClaimReviewHandler) PendingQueue(c fiber.Ctx) error {
	page, appErr := h.svc.PendingQueue(c.Context(), collectQuery(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

type reviewRequest struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

func (h *ClaimReviewHandler) Review(c fiber.Ctx) error {
	token, appErr := userToken(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	workID, appErr := submissionWorkID(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req reviewRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, errors.ErrBadRequest("请求格式错误"))
	}
	res, appErr := h.svc.Review(c.Context(), token, workID, req.Action, req.Reason)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, res)
}
