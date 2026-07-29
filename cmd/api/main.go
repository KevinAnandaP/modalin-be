package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"modalin-be/internal/campaign/job"
	"modalin-be/internal/router"
	"modalin-be/pkg/config"
	"modalin-be/pkg/database"
	"modalin-be/pkg/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	// 1. Load Configurations
	config.LoadConfig()
	if err := config.AppConfig.Validate(); err != nil {
		log.Fatal(err)
	}

	// 2. Connect Database
	database.ConnectDB()
	job.StartRepaymentOverdueScheduler(ctx, database.DB)

	// 3. Initialize Fiber Application
	app := newApp()

	// 5. Setup Routes
	router.SetupRoutes(app)

	// 6. Start Server
	port := config.AppConfig.Port
	log.Printf("Server starting on port %s...", port)
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- app.Listen(":" + port) }()
	select {
	case err := <-serverErrors:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return shutdownApp(shutdownCtx, app)
	}
}

func newApp() *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "Modalin Backend API v1.0",
		BodyLimit:    6 * 1024 * 1024,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
		ErrorHandler: jsonErrorHandler,
	})
	app.Use(requestid.New())
	app.Use(middleware.PropagateRequestID())
	app.Use(middleware.JSONRequestLogger(nil))
	app.Use(middleware.DefaultRequestMetrics.Middleware())
	app.Use(recover.New())
	app.Use(middleware.RestrictedCORS(config.AppConfig.CORSAllowedOrigins))
	return app
}

func shutdownApp(ctx context.Context, app *fiber.App) error {
	if err := app.ShutdownWithContext(ctx); err != nil {
		return err
	}
	return database.CloseDB()
}

func jsonErrorHandler(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	if fiberError := new(fiber.Error); errors.As(err, &fiberError) {
		status = fiberError.Code
	}
	requestID, _ := c.Locals("requestid").(string)
	message := "internal server error"
	if status >= 400 && status < 500 {
		message = "request rejected"
	}
	return c.Status(status).JSON(fiber.Map{"error": message, "request_id": requestID})
}
