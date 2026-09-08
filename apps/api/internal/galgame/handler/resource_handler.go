package handler

import (
	"strconv"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type ResourceHandler struct {
	resourceService *service.ResourceService
}

func NewResourceHandler(resourceService *service.ResourceService) *ResourceHandler {
	return &ResourceHandler{resourceService: resourceService}
}

func (h *ResourceHandler) GetResourceList(c fiber.Ctx) error {
	var req dto.ResourceListRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	page, appErr := h.resourceService.GetResourceList(c.Context(), &req, optionalUID(c), utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

func (h *ResourceHandler) GetResourceDetail(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的资源 ID"))
	}

	currentUID := optionalUID(c)
	detail, notFound, appErr := h.resourceService.GetResourceDetail(c.Context(), id, currentUID)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	if notFound != nil {
		return response.OK(c, "not found")
	}
	return response.OK(c, detail)
}

func (h *ResourceHandler) GetResourceDownloadDetail(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的资源 ID"))
	}

	currentUID := optionalUID(c)
	detail, appErr := h.resourceService.GetResourceDownloadDetail(c.Context(), id, currentUID)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

func (h *ResourceHandler) GetGalgameResources(c fiber.Ctx) error {
	var req dto.GalgameResourcesRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	cards, appErr := h.resourceService.GetGalgameResources(c.Context(), &req, optionalUID(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, cards)
}

func optionalUID(c fiber.Ctx) int {
	if user := middleware.GetUser(c); user != nil {
		return user.ID
	}
	return 0
}

func (h *ResourceHandler) CreateResource(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.CreateGalgameResourceRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.resourceService.CreateResource(
		c.Context(), user.ID, middleware.GetAccessToken(c), &req,
	); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "资源创建成功")
}

func (h *ResourceHandler) UpdateResource(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.UpdateGalgameResourceRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.resourceService.UpdateResource(c.Context(), user.ID, perm.CanUser(user.ID, user.Roles, perm.ResourceEditAny), &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "资源更新成功")
}

func (h *ResourceHandler) DeleteResource(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.DeleteGalgameResourceRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.resourceService.DeleteResource(user.ID, perm.CanUser(user.ID, user.Roles, perm.ResourceDeleteAny), req.GalgameResourceID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "资源已删除")
}

type setResourcePublishBanRequest struct {
	Banned bool `json:"banned"`
}

func (h *ResourceHandler) SetResourcePublishBan(c fiber.Ctx) error {
	gid, err := strconv.Atoi(c.Params("gid"))
	if err != nil || gid <= 0 {
		return response.Error(c, errors.ErrBadRequest("无效的 Galgame ID"))
	}
	var req setResourcePublishBanRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, errors.ErrBadRequest("请求格式错误"))
	}
	if appErr := h.resourceService.SetResourcePublishBan(gid, req.Banned); appErr != nil {
		return response.Error(c, appErr)
	}
	if req.Banned {
		return response.OKMessage(c, "已禁止在本游戏下发布资源")
	}
	return response.OKMessage(c, "已解除本游戏的资源发布禁止")
}

func (h *ResourceHandler) ToggleLike(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.ToggleResourceLikeRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.resourceService.ToggleLike(user.ID, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "操作成功")
}

func (h *ResourceHandler) MarkValid(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.ResourceStatusRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	canEditAny := perm.CanUser(user.ID, user.Roles, perm.ResourceEditAny)
	if appErr := h.resourceService.MarkValid(user.ID, canEditAny, req.GalgameResourceID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "资源已标记为有效")
}

func (h *ResourceHandler) MarkExpired(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.ResourceStatusRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	res, appErr := h.resourceService.MarkExpired(c.Context(), user.ID, req.GalgameResourceID)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, res)
}
