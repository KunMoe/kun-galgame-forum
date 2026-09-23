package handler

import (
	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

type GalgameProxyHandler struct {
	galgameProxyService *service.GalgameProxyService
}

func NewGalgameProxyHandler(galgameProxyService *service.GalgameProxyService) *GalgameProxyHandler {
	return &GalgameProxyHandler{galgameProxyService: galgameProxyService}
}

func (h *GalgameProxyHandler) GetGalgameLinks(c fiber.Ctx) error {
	links, appErr := h.galgameProxyService.GetGalgameLinks(c.Context(), c.Params("id"))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, links)
}
