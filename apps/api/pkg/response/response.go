package response

import (
	"kun-galgame-api/pkg/errors"

	"github.com/gofiber/fiber/v3"
)

func OK(c fiber.Ctx, data any) error {
	return c.JSON(fiber.Map{
		"code":    errors.CodeOK,
		"message": "成功",
		"data":    data,
	})
}

func OKMessage(c fiber.Ctx, msg string) error {
	return c.JSON(fiber.Map{
		"code":    errors.CodeOK,
		"message": msg,
	})
}

func Error(c fiber.Ctx, err *errors.AppError) error {
	body := fiber.Map{
		"code":    err.Code,
		"message": err.Message,
	}
	if len(err.Errors) > 0 {
		body["errors"] = err.Errors
	}
	return c.Status(err.StatusCode).JSON(body)
}

func Paginated(c fiber.Ctx, items any, total int64) error {
	return c.JSON(fiber.Map{
		"code":    errors.CodeOK,
		"message": "成功",
		"data": fiber.Map{
			"items": items,
			"total": total,
		},
	})
}
