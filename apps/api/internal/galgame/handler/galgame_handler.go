package handler

import (
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type GalgameHandler struct {
	galgameService *service.GalgameService
}

func NewGalgameHandler(galgameService *service.GalgameService) *GalgameHandler {
	return &GalgameHandler{galgameService: galgameService}
}

func (h *GalgameHandler) GetList(c fiber.Ctx) error {
	var req dto.GalgameListRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	page, appErr := h.galgameService.GetList(c.Context(), &req, utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

func (h *GalgameHandler) CollectedCalendar(c fiber.Ctx) error {
	return response.OK(c, h.galgameService.CollectedCalendar(utils.IsSFW(c)))
}
