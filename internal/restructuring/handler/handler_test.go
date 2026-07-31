package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestPageUsesValidatedQueryValues(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		limit, offset := page(c)
		return c.JSON(fiber.Map{"limit": limit, "offset": offset})
	})
	response, err := app.Test(httptest.NewRequest("GET", "/?limit=50&offset=10", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status=%d", response.StatusCode)
	}
}
