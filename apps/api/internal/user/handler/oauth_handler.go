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

func (h *OAuthHandler) Me(c fiber.Ctx) error {
	noStore(c)
	user, err := middleware.MustGetUser(c)
	if err != nil {
		return response.Error(c, err)
	}

	profile, appErr := h.authService.GetProfile(c.Context(), user.ID)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	enrichProfileFromSession(profile, user)

	return response.OK(c, profile)
}

func enrichProfileFromSession(p *dto.UserProfile, u *middleware.UserInfo) {
	p.Sub = u.Sub
	p.Roles = u.Roles
	p.AdultConfirmed = u.AdultConfirmed
	p.NSFWDisplay = u.NSFWDisplay
}

// Cloudflare rewrites a downstream's own max-age into a much longer one, so a
// cached preference response shows the reader a stance they already changed on
// another device. The upstream contract sets no-store on the same faces.
func noStore(c fiber.Ctx) { c.Set("Cache-Control", "no-store") }
