package handler

import (
	"errors"
	"mime/multipart"
	"net/http"
	"strconv"

	"modalin-be/internal/verification/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type proofStorage interface {
	StoreCategory(string, *multipart.FileHeader) (string, error)
	Read(string) ([]byte, error)
}
type Handler struct {
	service *service.Service
	storage proofStorage
}

func New(s *service.Service, stores ...proofStorage) *Handler {
	h := &Handler{service: s}
	if len(stores) > 0 {
		h.storage = stores[0]
	}
	return h
}
func (h *Handler) CreateRequest(c *fiber.Ctx) error {
	campaignID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return bad(c, "invalid campaign id")
	}
	v, err := h.service.CreateRequest(c.Context(), user(c), campaignID)
	return result(c, v, err, fiber.StatusCreated)
}
func (h *Handler) ListAdmin(c *fiber.Ctx) error {
	v, err := h.service.ListPage(c.Context(), c.Query("status"), nil, queryID(c, "campaign_id"), queryInt(c, "limit", 20), queryInt(c, "offset", 0))
	return result(c, v, err, 200)
}
func (h *Handler) ListMine(c *fiber.Ctx) error {
	u := user(c)
	v, err := h.service.ListPage(c.Context(), c.Query("status"), &u, queryID(c, "campaign_id"), queryInt(c, "limit", 20), queryInt(c, "offset", 0))
	return result(c, v, err, 200)
}
func (h *Handler) Assign(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return bad(c, "invalid verification request id")
	}
	var in struct {
		VerifierID uuid.UUID `json:"verifier_id"`
	}
	if err := c.BodyParser(&in); err != nil {
		return bad(c, "invalid request body")
	}
	return result(c, nil, h.service.Assign(c.Context(), user(c), id, in.VerifierID), 200)
}
func (h *Handler) Reassign(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return bad(c, "invalid verification request id")
	}
	var in struct {
		VerifierID uuid.UUID `json:"verifier_id"`
	}
	if err := c.BodyParser(&in); err != nil {
		return bad(c, "invalid request body")
	}
	return result(c, nil, h.service.Reassign(c.Context(), user(c), id, in.VerifierID), 200)
}
func (h *Handler) Cancel(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return bad(c, "invalid verification request id")
	}
	return result(c, nil, h.service.Cancel(c.Context(), user(c), id), 200)
}
func (h *Handler) BackfillRisks(c *fiber.Ctx) error {
	count, err := h.service.BackfillActiveRisks(c.Context())
	if err != nil {
		return result(c, nil, err, 200)
	}
	return c.JSON(fiber.Map{"data": fiber.Map{"recalculated": count}})
}
func (h *Handler) SubmitReport(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return bad(c, "invalid verification request id")
	}
	var in service.ReportInput
	if err := c.BodyParser(&in); err != nil {
		return bad(c, "invalid request body")
	}
	v, err := h.service.SubmitReport(c.Context(), user(c), id, in)
	return result(c, v, err, fiber.StatusCreated)
}
func (h *Handler) UploadPhoto(c *fiber.Ctx) error {
	if h.storage == nil {
		return c.Status(500).JSON(fiber.Map{"error": "proof storage is unavailable"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return bad(c, "invalid verification request id")
	}
	if err := h.service.CanUploadPhoto(c.Context(), user(c), id); err != nil {
		return result(c, nil, err, 200)
	}
	file, err := c.FormFile("photo")
	if err != nil {
		return bad(c, "photo file is required")
	}
	url, err := h.storage.StoreCategory("verification-reports", file)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "photo must be JPEG, PNG, or WebP up to 5 MB"})
	}
	return c.Status(201).JSON(fiber.Map{"data": fiber.Map{"photo_url": url}})
}
func (h *Handler) DownloadPhoto(c *fiber.Ctx) error {
	if h.storage == nil {
		return c.Status(500).JSON(fiber.Map{"error": "proof storage is unavailable"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return bad(c, "invalid verification request id")
	}
	roles, _ := c.Locals("roles").([]string)
	url, err := h.service.PhotoURL(c.Context(), user(c), id, roles)
	if err != nil {
		return result(c, nil, err, 200)
	}
	data, err := h.storage.Read(url)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "photo not found"})
	}
	c.Set(fiber.HeaderContentType, http.DetectContentType(data))
	c.Set(fiber.HeaderContentDisposition, "attachment; filename=verification-photo")
	return c.Send(data)
}
func (h *Handler) Decide(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return bad(c, "invalid verification request id")
	}
	var in struct {
		Decision string `json:"decision"`
	}
	if err := c.BodyParser(&in); err != nil {
		return bad(c, "invalid request body")
	}
	return result(c, nil, h.service.Decide(c.Context(), user(c), id, in.Decision), 200)
}
func (h *Handler) Vote(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return bad(c, "invalid business id")
	}
	var in struct {
		VoteType string  `json:"vote_type"`
		Reason   *string `json:"reason"`
	}
	if err := c.BodyParser(&in); err != nil {
		return bad(c, "invalid request body")
	}
	return result(c, nil, h.service.Vote(c.Context(), user(c), id, in.VoteType, in.Reason), 200)
}
func user(c *fiber.Ctx) uuid.UUID { v, _ := c.Locals("user_id").(uuid.UUID); return v }
func queryID(c *fiber.Ctx, key string) *uuid.UUID {
	raw := c.Query(key)
	if raw == "" {
		return nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil
	}
	return &id
}
func queryInt(c *fiber.Ctx, key string, fallback int) int {
	n, err := strconv.Atoi(c.Query(key))
	if err != nil {
		return fallback
	}
	return n
}
func bad(c *fiber.Ctx, message string) error { return c.Status(400).JSON(fiber.Map{"error": message}) }
func result(c *fiber.Ctx, v any, err error, status int) error {
	if err == nil {
		if v == nil {
			return c.Status(status).JSON(fiber.Map{"message": "success"})
		}
		return c.Status(status).JSON(fiber.Map{"data": v})
	}
	switch {
	case errors.Is(err, service.ErrNotFound):
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, service.ErrConflictOfInterest), errors.Is(err, service.ErrSelfVote):
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, service.ErrUnavailable):
		return c.Status(409).JSON(fiber.Map{"error": err.Error()})
	default:
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
}
