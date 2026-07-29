package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// RestrictedCORS permits browser requests only from explicitly configured origins.
func RestrictedCORS(origins []string) fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins: strings.Join(origins, ","),
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, PATCH, DELETE, OPTIONS",
	})
}
