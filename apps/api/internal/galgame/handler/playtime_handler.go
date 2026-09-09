package handler

import (
	stderrors "errors"
	"log/slog"
	"net/http"

	"kun-galgame-api/internal/galgame/service"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type PlaytimeHandler struct {
	svc *service.PlaytimeService
}

func NewPlaytimeHandler(svc *service.PlaytimeService) *PlaytimeHandler {
	return &PlaytimeHandler{svc: svc}
}

var errPlaytimeDown = errors.New(errors.CodeBiz, "游玩时长服务暂不可用", http.StatusServiceUnavailable)

type reportPlaytimeRequest struct {
	Minutes int    `json:"minutes"`
	Status  string `json:"status"`
}

var playtimeStatuses = map[string]bool{
	catalogclient.WorkStateDoing:   true,
	catalogclient.WorkStateDone:    true,
	catalogclient.WorkStateOnHold:  true,
	catalogclient.WorkStateDropped: true,
}

func (h *PlaytimeHandler) Report(c fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}
	gid, appErr := parseGid(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req reportPlaytimeRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.Error(c, errors.ErrBadRequest("请求参数错误"))
	}
	if req.Minutes < 0 || req.Minutes > catalogclient.PlaytimeMinutesMax {
		return response.Error(c, errors.ErrBadRequest("游玩时长超出可记录的范围"))
	}
	if req.Status == "" {
		req.Status = catalogclient.WorkStateDoing
	}
	if !playtimeStatuses[req.Status] {
		return response.Error(c, errors.ErrBadRequest("未知的游玩状态"))
	}
	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}

	got, err := h.svc.Report(c.Context(), int(gid), token, req.Minutes, req.Status)
	if err != nil {
		return playtimeError(c, err)
	}
	return response.OK(c, got)
}

func (h *PlaytimeHandler) ListMine(c fiber.Ctx) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}
	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}
	page, limit := parseCollectionPage(c, 24)

	got, err := h.svc.ListMine(c.Context(), token, page, limit, utils.IsSFW(c))
	if err != nil {
		return playtimeError(c, err)
	}
	return response.OK(c, got)
}

func playtimeError(c fiber.Ctx, err error) error {
	var appErr *errors.AppError
	if stderrors.As(err, &appErr) {
		return response.Error(c, appErr)
	}
	var apiErr *catalogclient.UserAPIError
	switch {
	case stderrors.Is(err, catalogclient.ErrInsufficientScope):
		return response.Error(c, errors.ErrReauthRequired(
			"记录游玩时长需要新的授权，请退出登录后重新登录以授予该权限"))
	case stderrors.Is(err, catalogclient.ErrUnauthorized):
		return response.Error(c, errors.ErrAuthExpired())
	case stderrors.Is(err, catalogclient.ErrNotFound):
		return response.Error(c, errors.ErrNotFound("条目不存在"))
	case stderrors.Is(err, catalogclient.ErrNotConfigured):
		return response.Error(c, errPlaytimeDown)
	case stderrors.As(err, &apiErr):
		if apiErr.Status == http.StatusTooManyRequests {
			return response.Error(c, errors.ErrBadRequest("操作太频繁, 请稍后再试"))
		}
		if apiErr.Status == http.StatusBadRequest {
			return response.Error(c, errors.ErrBadRequest("游玩时长不被接受"))
		}
		slog.Error("galgame playtime: upstream error",
			"status", apiErr.Status, "code", apiErr.Code, "msg", apiErr.Message)
		return response.Error(c, errPlaytimeDown)
	default:
		slog.Warn("galgame playtime: catalog unreachable", "error", err)
		return response.Error(c, errPlaytimeDown)
	}
}
