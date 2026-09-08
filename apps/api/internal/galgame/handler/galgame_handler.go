package handler

import (
	"strconv"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
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

func (h *GalgameHandler) GetDetail(c fiber.Ctx) error {
	gid, err := strconv.Atoi(c.Params("gid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的 Galgame ID"))
	}

	detail, appErr := h.galgameService.GetDetail(
		c.Context(), gid, optionalUID(c), middleware.GetAccessToken(c), utils.IsSFW(c),
	)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
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

func (h *GalgameHandler) ToggleLike(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	gid, err := strconv.Atoi(c.Params("gid"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的 Galgame ID"))
	}

	if appErr := h.galgameService.ToggleLike(c.Context(), user.ID, gid); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "操作成功")
}

func (h *GalgameHandler) MyInteractions(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, h.galgameService.GetMyInteractions(c.Context(), user.ID, middleware.GetAccessToken(c)))
}
