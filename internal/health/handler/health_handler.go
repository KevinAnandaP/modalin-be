package handler

import (
	"modalin-be/pkg/database"

	"github.com/gofiber/fiber/v2"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) CheckHealth(c *fiber.Ctx) error {
	return h.Ready(c)
}

// Live proves that the API process can handle requests. It deliberately does
// not query dependencies, so an orchestrator will restart only a stuck process.
func (h *HealthHandler) Live(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "up"})
}

// Ready proves the API can serve traffic that requires PostgreSQL.
func (h *HealthHandler) Ready(c *fiber.Ctx) error {
	dbStatus := "connected"

	if database.DB == nil {
		dbStatus = "disconnected (DB client is nil)"
	} else {
		sqlDB, err := database.DB.DB()
		if err != nil {
			dbStatus = "disconnected (failed to get raw DB client)"
		} else {
			err = sqlDB.Ping()
			if err != nil {
				dbStatus = "disconnected (ping failed)"
			}
		}
	}

	response := fiber.Map{
		"status":   "up",
		"database": dbStatus,
		"message":  "modalin-be is healthy!",
	}

	if dbStatus != "connected" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(response)
	}

	return c.Status(fiber.StatusOK).JSON(response)
}
