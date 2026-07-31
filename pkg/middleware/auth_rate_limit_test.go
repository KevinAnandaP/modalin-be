package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestLoginRateLimitRejectsBurstForSameIPAndEmail(t *testing.T) {
	app := fiber.New()
	app.Post("/login", LoginRateLimit(), func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })
	for i := 0; i < 10; i++ {
		response, err := app.Test(httptest.NewRequest("POST", "/login", strings.NewReader(`{"email":"person@example.com"}`)))
		if err != nil || response.StatusCode != fiber.StatusNoContent {
			t.Fatalf("request %d: response=%v err=%v", i, response, err)
		}
	}
	response, err := app.Test(httptest.NewRequest("POST", "/login", strings.NewReader(`{"email":"person@example.com"}`)))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", response.StatusCode)
	}
}

func TestRegistrationAndGoogleRateLimitsRejectBursts(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		limit fiber.Handler
		max   int
	}{
		{name: "registration", path: "/register", limit: RegistrationRateLimit(), max: 5},
		{name: "Google", path: "/google", limit: GoogleAuthRateLimit(), max: 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post(tt.path, tt.limit, func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })
			for i := 0; i < tt.max; i++ {
				response, err := app.Test(httptest.NewRequest("POST", tt.path, nil))
				if err != nil || response.StatusCode != fiber.StatusNoContent {
					t.Fatalf("request %d: response=%v err=%v", i, response, err)
				}
			}
			response, err := app.Test(httptest.NewRequest("POST", tt.path, nil))
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != fiber.StatusTooManyRequests {
				t.Fatalf("expected 429, got %d", response.StatusCode)
			}
		})
	}
}
