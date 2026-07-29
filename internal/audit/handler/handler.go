package handler

import (
	"strconv"
	"time"

	"modalin-be/internal/audit/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{ service *service.Service }

func New(service *service.Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *fiber.Ctx) error {
	userID, err := optionalID(c.Query("user_id"))
	if err != nil {
		return bad(c, "invalid user_id")
	}
	entityID, err := optionalID(c.Query("entity_id"))
	if err != nil {
		return bad(c, "invalid entity_id")
	}
	from, err := optionalTime(c.Query("from"))
	if err != nil {
		return bad(c, "invalid from timestamp")
	}
	to, err := optionalTime(c.Query("to"))
	if err != nil {
		return bad(c, "invalid to timestamp")
	}
	if from != nil && to != nil && from.After(*to) {
		return bad(c, "from must not be after to")
	}
	page, err := h.service.List(c.Context(), service.Filter{UserID: userID, EntityID: entityID, Action: c.Query("action"), EntityType: c.Query("entity_type"), From: from, To: to, Limit: queryInt(c, "limit", 20), Offset: queryInt(c, "offset", 0)})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list audit logs"})
	}
	return c.JSON(fiber.Map{"data": page.Data, "pagination": fiber.Map{"limit": page.Limit, "offset": page.Offset, "total": page.Total}})
}

func optionalID(value string) (*uuid.UUID, error) {
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func optionalTime(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func queryInt(c *fiber.Ctx, key string, fallback int) int {
	value, err := strconv.Atoi(c.Query(key))
	if err != nil {
		return fallback
	}
	return value
}

func bad(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": message})
}
