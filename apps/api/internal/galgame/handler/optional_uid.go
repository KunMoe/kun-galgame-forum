package handler

import (
	"kun-galgame-api/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func optionalUID(c fiber.Ctx) int {
	if user := middleware.GetUser(c); user != nil {
		return user.ID
	}
	return 0
}
