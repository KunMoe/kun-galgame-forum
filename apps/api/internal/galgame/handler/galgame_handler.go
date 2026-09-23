package handler

import (
	"strconv"
	"strings"

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
	workID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的 Galgame ID"))
	}

	detail, appErr := h.galgameService.GetDetail(
		c.Context(), workID, optionalUID(c), middleware.GetAccessToken(c), utils.IsSFW(c),
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

func (h *GalgameHandler) CollectedCalendar(c fiber.Ctx) error {
	return response.OK(c, h.galgameService.CollectedCalendar(utils.IsSFW(c)))
}

func (h *GalgameHandler) ToggleLike(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	workID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的 Galgame ID"))
	}

	if appErr := h.galgameService.ToggleLike(c.Context(), user.ID, workID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "操作成功")
}

func (h *GalgameHandler) MyInteractions(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, h.galgameService.GetMyInteractions(
		c.Context(), user.ID, middleware.GetAccessToken(c), parseCSVInts(c.Query("work_ids"), 100),
	))
}

func parseCSVInts(raw string, max int) []int {
	if raw == "" || max < 1 {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	seen := map[int]struct{}{}
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil || n <= 0 {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
		if len(out) >= max {
			break
		}
	}
	return out
}
