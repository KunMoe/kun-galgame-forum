package handler

import (
	"strconv"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/user/dto"
	"kun-galgame-api/internal/user/service"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type UserHandler struct {
	userService        *service.UserService
	userContentService *service.UserContentService
}

func NewUserHandler(
	userService *service.UserService,
	userContentService *service.UserContentService,
) *UserHandler {
	return &UserHandler{
		userService:        userService,
		userContentService: userContentService,
	}
}

func (h *UserHandler) GetProfile(c fiber.Ctx) error {
	userID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}
	profile, appErr := h.userService.GetUserProfile(c.Context(), userID)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, profile)
}

func (h *UserHandler) GetFloatingCard(c fiber.Ctx) error {
	var req dto.FloatingCardRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	card, appErr := h.userService.GetFloatingCard(c.Context(), req.UserID)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, card)
}

func (h *UserHandler) GetUserGalgames(c fiber.Ctx) error {
	userID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}
	var req dto.UserGalgamesRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	cards, total, appErr := h.userContentService.GetUserGalgameCards(c.Context(), userID, &req, utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.Paginated(c, cards, total)
}

func (h *UserHandler) GetUserTopics(c fiber.Ctx) error {
	userID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}
	var req dto.UserTopicsRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if req.Type == "topic_hide" {
		u := middleware.GetUser(c)
		if u == nil || (u.ID != userID && !u.Can(perm.TopicViewHidden)) {
			return response.Error(c, errors.ErrForbidden("您没有权限查看该用户的隐藏话题"))
		}
	}
	viewer := middleware.GetUser(c)
	canViewRestricted := viewer != nil && (viewer.ID == userID || viewer.Can(perm.TopicViewRestricted))
	items, total, appErr := h.userContentService.GetUserTopics(c.Context(), userID, &req, utils.IsSFW(c), viewer != nil, canViewRestricted)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, fiber.Map{"topics": items, "total": total})
}

func (h *UserHandler) GetUserReplies(c fiber.Ctx) error {
	userID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}
	var req dto.UserRepliesRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	items, total, appErr := h.userContentService.GetUserReplies(c.Context(), userID, &req, utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, fiber.Map{"replies": items, "total": total})
}

func (h *UserHandler) GetUserComments(c fiber.Ctx) error {
	userID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}
	var req dto.UserCommentsRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	items, total, appErr := h.userContentService.GetUserComments(c.Context(), userID, &req, utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, fiber.Map{"comments": items, "total": total})
}

func (h *UserHandler) GetUserResources(c fiber.Ctx) error {
	userID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}
	var req dto.UserResourcesRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	page, appErr := h.userContentService.GetUserResources(c.Context(), userID, &req, utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

func (h *UserHandler) GetUserGalgameComments(c fiber.Ctx) error {
	userID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}
	var req dto.UserGalgameCommentsRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	items, nextCursor, appErr := h.userContentService.GetUserGalgameComments(c.Context(), userID, &req, utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, fiber.Map{"comments": items, "next_cursor": nextCursor})
}

func (h *UserHandler) GetUserRatings(c fiber.Ctx) error {
	userID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的用户 ID"))
	}
	var req dto.UserRatingsRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	page, appErr := h.userContentService.GetUserRatings(c.Context(), userID, &req, utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

func (h *UserHandler) GetNotificationPreferences(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	prefs, appErr := h.userService.GetNotificationPreferences(user.ID)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, prefs)
}
func (h *UserHandler) UpdateNotificationPreferences(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.UpdateNotificationPreferenceRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	prefs, appErr := h.userService.UpdateNotificationPreferences(user.ID, req.MutedTypes)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, prefs)
}
