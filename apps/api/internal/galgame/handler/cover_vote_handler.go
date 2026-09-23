package handler

import (
	stderrors "errors"
	"log/slog"
	"net/http"
	"strconv"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

type CoverVoteHandler struct {
	catalog       *catalogclient.Client
	galgameClient *client.GalgameClient
}

func NewCoverVoteHandler(catalog *catalogclient.Client, galgameClient *client.GalgameClient) *CoverVoteHandler {
	return &CoverVoteHandler{catalog: catalog, galgameClient: galgameClient}
}

var errVoteDown = errors.New(errors.CodeBiz, "封面投票服务暂不可用", http.StatusServiceUnavailable)

func (h *CoverVoteHandler) Vote(c fiber.Ctx) error {
	return h.cast(c, true)
}

func (h *CoverVoteHandler) Unvote(c fiber.Ctx) error {
	return h.cast(c, false)
}

func (h *CoverVoteHandler) cast(c fiber.Ctx, vote bool) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}
	workID, appErr := parseWorkID(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	coverID, err := strconv.ParseInt(c.Params("coverId"), 10, 64)
	if err != nil || coverID <= 0 {
		return response.Error(c, errors.ErrBadRequest("无效的封面 ID"))
	}
	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}
	if h.catalog == nil || h.galgameClient == nil {
		return response.Error(c, errVoteDown)
	}

	var result *catalogclient.CoverVoteResult
	var voteErr error
	if vote {
		result, voteErr = h.catalog.VoteCover(c.Context(), token, workID, coverID)
	} else {
		result, voteErr = h.catalog.UnvoteCover(c.Context(), token, workID, coverID)
	}
	if voteErr != nil {
		return coverVoteError(c, voteErr)
	}
	return response.OK(c, fiber.Map{
		"cover_id":   result.CoverID,
		"vote_count": result.VoteCount,
		"voted":      result.Voted,
	})
}

func coverVoteError(c fiber.Ctx, err error) error {
	var apiErr *catalogclient.UserAPIError
	switch {
	case stderrors.Is(err, catalogclient.ErrInsufficientScope):
		return response.Error(c, errors.ErrReauthRequired(
			"投票需要新的授权，请退出登录后重新登录以授予该权限"))
	case stderrors.Is(err, catalogclient.ErrUnauthorized):
		return response.Error(c, errors.ErrAuthExpired())
	case stderrors.Is(err, catalogclient.ErrNotFound):
		return response.Error(c, errors.ErrNotFound("条目或封面不存在"))
	case stderrors.Is(err, catalogclient.ErrNotConfigured):
		return response.Error(c, errVoteDown)
	case stderrors.As(err, &apiErr):
		if apiErr.Status == http.StatusForbidden {
			return response.Error(c, errors.ErrForbidden("你没有权限执行此操作"))
		}
		slog.Error("galgame cover vote: upstream error",
			"status", apiErr.Status, "code", apiErr.Code, "msg", apiErr.Message)
		return response.Error(c, errVoteDown)
	default:
		slog.Warn("galgame cover vote: catalog unreachable", "error", err)
		return response.Error(c, errVoteDown)
	}
}
