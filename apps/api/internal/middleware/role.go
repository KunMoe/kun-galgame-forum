package middleware

import (
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

func RequireModerator() fiber.Handler {
	return func(c fiber.Ctx) error {
		user, appErr := staffCandidate(c)
		if appErr == nil && !user.CanModerate() {
			appErr = errNoPermission()
		}
		if appErr != nil {
			return response.Error(c, appErr)
		}
		return c.Next()
	}
}

func RequirePermission(p perm.Permission) fiber.Handler {
	return func(c fiber.Ctx) error {
		user, appErr := staffCandidate(c)
		if appErr == nil && !user.Can(p) {
			appErr = errNoPermission()
		}
		if appErr != nil {
			return response.Error(c, appErr)
		}
		return c.Next()
	}
}

func staffCandidate(c fiber.Ctx) (*UserInfo, *errors.AppError) {
	user, appErr := MustGetUser(c)
	if appErr != nil {
		return nil, appErr
	}
	if user.viaBearer {
		return nil, errors.ErrForbidden("管理操作请在网页端进行")
	}
	return user, nil
}

func errNoPermission() *errors.AppError {
	return errors.ErrForbidden("您没有权限进行此操作")
}
