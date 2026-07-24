package main

import (
	"log"

	"modalin-be/internal/campaign/job"
	"modalin-be/internal/router"
	"modalin-be/pkg/config"
	"modalin-be/pkg/database"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// 1. Load Configurations
	config.LoadConfig()
	if config.AppConfig.JWTSecret == "" {
		log.Fatal("JWT_SECRET must be configured")
	}

	// 2. Connect Database
	database.ConnectDB()
	job.StartRepaymentOverdueScheduler(database.DB)

	// 3. Initialize Fiber Application
	app := fiber.New(fiber.Config{
		AppName: "Modalin Backend API v1.0",
	})

	// 4. Middlewares
	app.Use(logger.New())  // Request logging
	app.Use(recover.New()) // Panic recovery
	app.Use(cors.New())    // CORS configuration (crucial for Vue frontend)
	app.Get("/uploads/fund-usage-proofs/*", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusForbidden) })
	app.Get("/uploads/financial-proofs/*", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusForbidden) })
	app.Get("/uploads/monthly-progress-proofs/*", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusForbidden) })
	app.Get("/uploads/revenue-report-proofs/*", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusForbidden) })
	app.Get("/uploads/repayment-proofs/*", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusForbidden) })
	app.Get("/uploads/disbursement-proofs/*", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusForbidden) })
	app.Get("/uploads/verification-reports/*", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusForbidden) })
	app.Static("/uploads", config.AppConfig.UploadDir)

	// 5. Setup Routes
	router.SetupRoutes(app)

	// 6. Start Server
	port := config.AppConfig.Port
	log.Printf("Server starting on port %s...", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
