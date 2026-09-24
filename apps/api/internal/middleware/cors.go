package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3/middleware/cors"
)

func CORS(allowOrigins string) cors.Config {
	return cors.Config{
		AllowOrigins:     strings.Split(allowOrigins, ","),
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "Idempotency-Key", "If-Match", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID", "Idempotency-Replayed", "Location", "ETag", "Retry-After", "Deprecation", "Sunset", "Link"},
		AllowCredentials: true,
		MaxAge:           86400,
	}
}
