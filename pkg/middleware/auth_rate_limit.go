package middleware

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// LoginRateLimit protects one credential pair from online password guessing.
// Its in-memory state is suitable only for a single API instance; use a shared
// Fiber limiter store when the API is horizontally scaled.
func LoginRateLimit() fiber.Handler {
	return publicAuthRateLimit("login", 10, time.Minute, true)
}

func RegistrationRateLimit() fiber.Handler {
	return publicAuthRateLimit("register", 5, time.Minute, false)
}

func GoogleAuthRateLimit() fiber.Handler {
	return publicAuthRateLimit("google", 10, time.Minute, false)
}

func publicAuthRateLimit(action string, max int, expiration time.Duration, includeEmail bool) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: expiration,
		KeyGenerator: func(c *fiber.Ctx) string {
			key := action + ":" + c.IP()
			if includeEmail {
				var body struct {
					Email string `json:"email"`
				}
				if json.Unmarshal(c.Body(), &body) == nil {
					if email := strings.ToLower(strings.TrimSpace(body.Email)); email != "" {
						key += ":" + email
					}
				}
			}
			return key
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "too many authentication attempts; try again shortly"})
		},
	})
}
