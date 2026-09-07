package handler

import (
	"strconv"

	"kun-galgame-api/internal/admin/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

type PurgeHandler struct {
	purgeService *service.PurgeService
}

func NewPurgeHandler(purgeService *service.PurgeService) *PurgeHandler {
	return &PurgeHandler{purgeService: purgeService}
}

func (h *PurgeHandler) GetUserContentStats(c fiber.Ctx) error {
	userID, appErr := parseUserID(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, h.purgeService.GetUserContentStats(c.Context(), userID))
}

func (h *PurgeHandler) PurgeUserContent(c fiber.Ctx) error {
	operator, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	userID, appErr := parseUserID(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	result, appErr := h.purgeService.PurgeUserContent(c.Context(), operator.ID, userID, middleware.GetAccessToken(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, result)
}

func parseUserID(c fiber.Ctx) (int, *errors.AppError) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return 0, errors.ErrBadRequest("非法的用户 ID")
	}
	return id, nil
}
