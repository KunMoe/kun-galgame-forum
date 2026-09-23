package handler

import (
	"net/url"

	"github.com/gofiber/fiber/v3"
)

func collectQuery(c fiber.Ctx) url.Values {
	q := make(url.Values)
	for k, v := range c.Queries() {
		q.Set(k, v)
	}
	return q
}
