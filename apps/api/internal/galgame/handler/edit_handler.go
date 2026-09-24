package handler

import (
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/errors"

	"github.com/gofiber/fiber/v3"
)

func userToken(c fiber.Ctx) (string, *errors.AppError) {
	token := middleware.GetAccessToken(c)
	if token == "" {
		return "", errors.ErrAuthExpired()
	}
	return token, nil
}
