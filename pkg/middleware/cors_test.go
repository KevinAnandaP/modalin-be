package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestRestrictedCORSAllowsOnlyConfiguredOrigins(t *testing.T) {
	app := fiber.New()
	app.Use(RestrictedCORS([]string{"https://modalin.my.id"}))
	app.Get("/ping", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })

	allowed := httptest.NewRequest("GET", "/ping", nil)
	allowed.Header.Set(fiber.HeaderOrigin, "https://modalin.my.id")
	response, err := app.Test(allowed)
	if err != nil {
		t.Fatal(err)
	}
	if got := response.Header.Get(fiber.HeaderAccessControlAllowOrigin); got != "https://modalin.my.id" {
		t.Fatalf("allowed origin header = %q", got)
	}

	blocked := httptest.NewRequest("GET", "/ping", nil)
	blocked.Header.Set(fiber.HeaderOrigin, "https://attacker.example")
	response, err = app.Test(blocked)
	if err != nil {
		t.Fatal(err)
	}
	if got := response.Header.Get(fiber.HeaderAccessControlAllowOrigin); got != "" {
		t.Fatalf("blocked origin header = %q, want empty", got)
	}
}
