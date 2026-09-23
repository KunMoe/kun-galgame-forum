package handler

import (
	communitytrust "kun-galgame-api/internal/community/trust"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/user/dto"
	"kun-galgame-api/internal/user/service"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type OAuthHandler struct {
	authService *service.AuthService
	secure      bool
	booster     *communitytrust.Reporter
}

func NewOAuthHandler(authService *service.AuthService, isProd bool, booster *communitytrust.Reporter) *OAuthHandler {
	return &OAuthHandler{authService: authService, secure: isProd, booster: booster}
}

func (h *OAuthHandler) Callback(c fiber.Ctx) error {
	var req dto.OAuthCallbackRequest
	if err := utils.ParseAndValidate(c, &req); err != nil {
		return response.Error(c, err)
	}

	session, appErr := h.authService.OAuthCallback(c.Context(), &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	if h.booster != nil {
		h.booster.Boost(session.User.ID, session.User.Roles)
	}

	c.Cookie(&fiber.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    session.Token,
		MaxAge:   int(middleware.SessionTTL.Seconds()),
		HTTPOnly: true,
		Secure:   h.secure,
		SameSite: "Lax",
		Path:     "/",
	})

	return response.OK(c, session.User)
}

func (h *OAuthHandler) Logout(c fiber.Ctx) error {
	token := c.Cookies(middleware.SessionCookieName)
	if token != "" {
		_ = h.authService.Logout(c.Context(), token)
	}

	c.Cookie(&fiber.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   h.secure,
		SameSite: "Lax",
		Path:     "/",
	})

	return response.OKMessage(c, "已登出")
}
