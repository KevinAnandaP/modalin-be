package router

import (
	healthHandler "modalin-be/internal/health/handler"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
	// Root route
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Welcome to Modalin Backend API",
			"version": "1.0.0",
		})
	})

	// Initialize Handlers
	health := healthHandler.NewHealthHandler()

	// Register Routes
	app.Get("/health", health.CheckHealth)

	// API group for future endpoints
	api := app.Group("/api")
	v1 := api.Group("/v1")

	// Register API v1 Routes
	v1.Get("/health", health.CheckHealth)
	v1.Get("/ping", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "pong",
		})
	})
}
