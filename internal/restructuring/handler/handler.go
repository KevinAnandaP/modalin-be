package handler

import (
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"modalin-be/internal/restructuring/service"
	"strconv"
)

type Handler struct{ s *service.Service }

func New(s *service.Service) *Handler { return &Handler{s} }
func (h *Handler) Create(c *fiber.Ctx) error {
	id, e := uuid.Parse(c.Params("id"))
	if e != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid campaign id"})
	}
	var in service.Input
	if e = c.BodyParser(&in); e != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	v, e := h.s.Create(c.Context(), user(c), id, in)
	return result(c, v, e, 201)
}
func (h *Handler) Decide(c *fiber.Ctx) error {
	id, e := uuid.Parse(c.Params("id"))
	if e != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request id"})
	}
	var x struct {
		Decision string `json:"decision"`
		Note     string `json:"note"`
	}
	if e = c.BodyParser(&x); e != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	return result(c, nil, h.s.Decide(c.Context(), user(c), id, x.Decision, x.Note), 200)
}
func (h *Handler) List(c *fiber.Ctx) error {
	id, e := uuid.Parse(c.Params("id"))
	if e != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid campaign id"})
	}
	limit, offset := page(c)
	v, t, e := h.s.List(c.Context(), user(c), &id, false, c.Query("status"), limit, offset)
	if e != nil {
		return result(c, nil, e, 500)
	}
	return c.JSON(fiber.Map{"data": v, "total": t})
}
func (h *Handler) ListAdmin(c *fiber.Ctx) error {
	limit, offset := page(c)
	v, t, e := h.s.List(c.Context(), uuid.Nil, nil, true, c.Query("status"), limit, offset)
	if e != nil {
		return result(c, nil, e, 500)
	}
	return c.JSON(fiber.Map{"data": v, "total": t})
}
func page(c *fiber.Ctx) (int, int) {
	limit := 20
	offset := 0
	if v, e := strconv.Atoi(c.Query("limit")); e == nil {
		limit = v
	}
	if v, e := strconv.Atoi(c.Query("offset")); e == nil {
		offset = v
	}
	return limit, offset
}
func user(c *fiber.Ctx) uuid.UUID { v, _ := c.Locals("user_id").(uuid.UUID); return v }
func result(c *fiber.Ctx, v any, e error, s int) error {
	if e == nil {
		return c.Status(s).JSON(fiber.Map{"data": v})
	}
	if errors.Is(e, service.ErrNotFound) {
		s = 404
	} else if errors.Is(e, service.ErrForbidden) {
		s = 403
	} else if errors.Is(e, service.ErrInvalid) {
		s = 400
	} else if errors.Is(e, service.ErrUnavailable) {
		s = 409
	} else {
		s = 500
	}
	return c.Status(s).JSON(fiber.Map{"error": e.Error()})
}
