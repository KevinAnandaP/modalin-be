package router

import (
	authHandler "modalin-be/internal/auth/handler"
	authRepository "modalin-be/internal/auth/repository"
	authService "modalin-be/internal/auth/service"
	healthHandler "modalin-be/internal/health/handler"
	"modalin-be/pkg/config"
	"modalin-be/pkg/database"
	"modalin-be/pkg/middleware"

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
	auth := authHandler.NewAuthHandler(authService.NewAuthService(authRepository.NewAuthRepository(database.DB), config.AppConfig.JWTSecret))

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

	authRoutes := v1.Group("/auth")
	authRoutes.Post("/register", auth.Register)
	authRoutes.Post("/login", auth.Login)
	authRoutes.Get("/me", middleware.JWTProtected(config.AppConfig.JWTSecret), auth.Me)
	authRoutes.Post("/roles", middleware.JWTProtected(config.AppConfig.JWTSecret), auth.RequestRole)
}
