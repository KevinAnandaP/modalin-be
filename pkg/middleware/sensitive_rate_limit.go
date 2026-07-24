package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/google/uuid"
)

// SensitiveActionRateLimit protects upload, approval, and payout actions from
// accidental retries and brute-force abuse. For horizontally scaled production
// deployments, replace the in-memory store with the shared limiter store.
func SensitiveActionRateLimit() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        20,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			if id, ok := c.Locals("user_id").(uuid.UUID); ok && id != uuid.Nil {
				return id.String()
			}
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "too many sensitive requests; try again shortly"})
		},
	})
}
