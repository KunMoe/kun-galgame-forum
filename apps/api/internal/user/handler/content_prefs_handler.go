package handler

import (
	"encoding/json"
	stderrors "errors"
	"log/slog"

	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/user/dto"
	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/userclient"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

type ContentPrefsHandler struct {
	oauthClient *oauth.Client
	userClient  *userclient.Client
	rdb         *redis.Client
}

func NewContentPrefsHandler(
	oauthClient *oauth.Client,
	userClient *userclient.Client,
	rdb *redis.Client,
) *ContentPrefsHandler {
	return &ContentPrefsHandler{oauthClient: oauthClient, userClient: userClient, rdb: rdb}
}

// Cloudflare rewrites a downstream's own max-age into a much longer one, so a
// cached preference response shows the reader a stance they already changed on
// another device. The upstream contract sets no-store on the same three faces.
func noStore(c fiber.Ctx) { c.Set("Cache-Control", "no-store") }

func (h *ContentPrefsHandler) UpdateNSFWDisplay(c fiber.Ctx) error {
	noStore(c)
	user, appErr := middleware.MustGetUser(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	var req dto.UpdateNSFWDisplayRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}

	state, err := h.oauthClient.PutAuthMeNSFW(token, req.NSFWDisplay)
	if err != nil {
		return response.Error(c, mapContentPrefsError(err))
	}

	adultConfirmed := state.AdultConfirmedAt != nil && *state.AdultConfirmedAt != ""
	if session := c.Cookies(middleware.SessionCookieName); session != "" {
		if serr := middleware.SetSessionContentStance(
			c.Context(), h.rdb, session, adultConfirmed, state.NSFWDisplay,
		); serr != nil {
			slog.Warn("写回 session 内容分级失败, 本次改动要等刷新才生效", "error", serr)
		}
	}
	h.userClient.Invalidate(user.ID)

	return response.OK(c, dto.NSFWDisplayResponse{
		NSFWDisplay:    state.NSFWDisplay,
		AdultConfirmed: adultConfirmed,
	})
}

func (h *ContentPrefsHandler) GetPreferences(c fiber.Ctx) error {
	noStore(c)
	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}
	data, err := h.oauthClient.GetPreferences(token)
	if err != nil {
		return response.Error(c, mapContentPrefsError(err))
	}
	return respondPreferences(c, data)
}

func (h *ContentPrefsHandler) UpdatePreferences(c fiber.Ctx) error {
	noStore(c)
	var req dto.UpdatePreferencesRequest
	if appErr := utils.ParseAndValidate(c, &req); appErr != nil {
		return response.Error(c, appErr)
	}
	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}
	data, err := h.oauthClient.PutPreferences(token, req.Doc, c.Get("If-Match"))
	if err != nil {
		return response.Error(c, mapContentPrefsError(err))
	}
	return respondPreferences(c, data)
}

func respondPreferences(c fiber.Ctx, data json.RawMessage) error {
	var out dto.PreferencesResponse
	if jerr := json.Unmarshal(data, &out); jerr != nil {
		return response.Error(c, errors.ErrInternal("解析云端偏好响应失败"))
	}
	return response.OK(c, out)
}

func mapContentPrefsError(err error) *errors.AppError {
	var oe *oauth.Error
	if stderrors.As(err, &oe) {
		switch oe.Code {
		case oauth.CodeAdultConfirmationNeeded:
			return errors.ErrAdultConfirmationRequired()
		case oauth.CodePreferencesScopeMissing:
			return errors.ErrCloudPreferencesUnavailable()
		case oauth.CodePreferencesConflict:
			return errors.ErrCloudPreferencesConflict()
		}
	}
	return mapOAuthError(err)
}
