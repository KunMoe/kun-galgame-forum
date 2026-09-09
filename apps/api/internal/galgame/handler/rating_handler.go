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

type RatingHandler struct {
	ratingService   *service.RatingService
	playtimeService *service.PlaytimeService
}

func NewRatingHandler(ratingService *service.RatingService, playtimeService *service.PlaytimeService) *RatingHandler {
	return &RatingHandler{ratingService: ratingService, playtimeService: playtimeService}
}

func (h *RatingHandler) GetAllRatings(c fiber.Ctx) error {
	var req dto.RatingListRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}

	page, appErr := h.ratingService.GetAllRatings(c.Context(), &req, utils.IsSFW(c))
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, page)
}

func (h *RatingHandler) GetRatingDetail(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, errors.ErrBadRequest("无效的评分 ID"))
	}

	currentUID := optionalUID(c)
	detail, appErr := h.ratingService.GetRatingDetail(c.Context(), id, currentUID)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OK(c, detail)
}

func (h *RatingHandler) CreateRating(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.CreateRatingRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	created, appErr := h.ratingService.CreateRating(c.Context(), user.ID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	err := response.OK(c, created)
	h.playtimeService.SyncWorkState(c.Context(), created.Galgame.ID, middleware.GetAccessToken(c), req.PlayStatus)
	return err
}

func (h *RatingHandler) UpdateRating(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.UpdateRatingRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	gid, appErr := h.ratingService.UpdateRating(c.Context(), user.ID, &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	err := response.OKMessage(c, "评分更新成功")
	h.playtimeService.SyncWorkState(c.Context(), gid, middleware.GetAccessToken(c), req.PlayStatus)
	return err
}

func (h *RatingHandler) DeleteRating(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.DeleteRatingRequest
	if appErr := utils.ParseQueryAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.ratingService.DeleteRating(user.ID, perm.CanUser(user.ID, user.Roles, perm.RatingDeleteAny), req.GalgameRatingID); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "评分已删除")
}

func (h *RatingHandler) ToggleLike(c fiber.Ctx) error {
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.ToggleRatingLikeRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	if appErr := h.ratingService.ToggleRatingLike(user.ID, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	return response.OKMessage(c, "操作成功")
}
