package handler

import (
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"modalin-be/internal/dispute/service"
	"strconv"
)

type Handler struct{ s *service.Service }

func New(s *service.Service) *Handler { return &Handler{s} }
func (h *Handler) Create(c *fiber.Ctx) error {
	id, e := uuid.Parse(c.Params("id"))
	if e != nil {
		return bad(c, "invalid campaign id")
	}
	var in service.CreateInput
	if e = c.BodyParser(&in); e != nil {
		return bad(c, "invalid request body")
	}
	v, e := h.s.Create(c.Context(), user(c), id, in)
	return result(c, v, e, 201)
}
func (h *Handler) List(c *fiber.Ctx) error {
	v, t, e := h.s.List(c.Context(), user(c), false, c.Query("status"), integer(c, "limit", 20), integer(c, "offset", 0))
	if e != nil {
		return result(c, nil, e, 200)
	}
	return c.JSON(fiber.Map{"data": v, "pagination": fiber.Map{"total": t}})
}
func (h *Handler) ListAdmin(c *fiber.Ctx) error {
	v, t, e := h.s.List(c.Context(), uuid.Nil, true, c.Query("status"), integer(c, "limit", 20), integer(c, "offset", 0))
	if e != nil {
		return result(c, nil, e, 200)
	}
	return c.JSON(fiber.Map{"data": v, "pagination": fiber.Map{"total": t}})
}
func (h *Handler) Review(c *fiber.Ctx) error {
	id, e := uuid.Parse(c.Params("id"))
	if e != nil {
		return bad(c, "invalid dispute id")
	}
	return result(c, nil, h.s.Review(c.Context(), user(c), id), 200)
}
func (h *Handler) Decide(c *fiber.Ctx) error {
	id, e := uuid.Parse(c.Params("id"))
	if e != nil {
		return bad(c, "invalid dispute id")
	}
	var x struct {
		Decision string `json:"decision"`
		Note     string `json:"note"`
	}
	if e = c.BodyParser(&x); e != nil {
		return bad(c, "invalid request body")
	}
	return result(c, nil, h.s.Decide(c.Context(), user(c), id, x.Decision, x.Note), 200)
}
func user(c *fiber.Ctx) uuid.UUID { v, _ := c.Locals("user_id").(uuid.UUID); return v }
func integer(c *fiber.Ctx, k string, d int) int {
	v, e := strconv.Atoi(c.Query(k))
	if e != nil {
		return d
	}
	return v
}
func bad(c *fiber.Ctx, m string) error { return c.Status(400).JSON(fiber.Map{"error": m}) }
func result(c *fiber.Ctx, v any, e error, code int) error {
	if e == nil {
		return c.Status(code).JSON(fiber.Map{"data": v})
	}
	if errors.Is(e, service.ErrNotFound) {
		return c.Status(404).JSON(fiber.Map{"error": e.Error()})
	}
	if errors.Is(e, service.ErrForbidden) {
		return c.Status(403).JSON(fiber.Map{"error": e.Error()})
	}
	if errors.Is(e, service.ErrInvalid) {
		return c.Status(400).JSON(fiber.Map{"error": e.Error()})
	}
	if errors.Is(e, service.ErrUnavailable) {
		return c.Status(409).JSON(fiber.Map{"error": e.Error()})
	}
	return c.Status(500).JSON(fiber.Map{"error": "dispute operation failed"})
}
