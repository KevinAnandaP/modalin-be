package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestSensitiveActionRateLimitUsesAuthenticatedUserAndRejectsBurst(t *testing.T) {
	app := fiber.New()
	userID := uuid.New()
	app.Use(func(c *fiber.Ctx) error { c.Locals("user_id", userID); return c.Next() })
	app.Post("/sensitive", SensitiveActionRateLimit(), func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })
	for i := 0; i < 20; i++ {
		response, err := app.Test(httptest.NewRequest("POST", "/sensitive", nil))
		if err != nil || response.StatusCode != fiber.StatusNoContent {
			t.Fatalf("request %d: response=%v err=%v", i, response, err)
		}
	}
	response, err := app.Test(httptest.NewRequest("POST", "/sensitive", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", response.StatusCode)
	}
}
