package handler

import (
	"errors"
	"net/http"
	"strconv"

	"modalin-be/internal/business/service"
	"modalin-be/internal/business/storage"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type BusinessHandler struct {
	service      *service.BusinessService
	proofStorage *storage.LocalProofStorage
}

func NewBusinessHandler(service *service.BusinessService, proofStorage *storage.LocalProofStorage) *BusinessHandler {
	return &BusinessHandler{service: service, proofStorage: proofStorage}
}

func (h *BusinessHandler) GetFinancialRecord(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "invalid authorization context"})
	}
	recordID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid record id"})
	}
	record, err := h.service.GetFinancialRecord(c.Context(), userID, recordID)
	if err != nil {
		return h.financialRecordError(c, err, "failed to fetch financial record")
	}
	return c.JSON(fiber.Map{"data": record})
}

func (h *BusinessHandler) UpdateFinancialRecord(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "invalid authorization context"})
	}
	recordID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid record id"})
	}
	var req service.UpdateFinancialRecordInput
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	record, err := h.service.UpdateFinancialRecord(c.Context(), userID, recordID, req)
	if err != nil {
		return h.financialRecordError(c, err, "failed to update financial record")
	}
	return c.JSON(fiber.Map{"data": record})
}

func (h *BusinessHandler) UploadFinancialRecordProof(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "invalid authorization context"})
	}
	recordID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid record id"})
	}
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "file is required"})
	}
	if h.proofStorage == nil {
		return c.Status(500).JSON(fiber.Map{"error": "proof storage is unavailable"})
	}
	fileURL, err := h.proofStorage.Store(file)
	if err != nil {
		if errors.Is(err, storage.ErrUnsupportedProofFile) {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(500).JSON(fiber.Map{"error": "failed to store proof file"})
	}
	var amount *int64
	if rawAmount := c.FormValue("amount"); rawAmount != "" {
		parsedAmount, parseErr := strconv.ParseInt(rawAmount, 10, 64)
		if parseErr != nil || parsedAmount < 0 {
			_ = h.proofStorage.Delete(fileURL)
			return c.Status(400).JSON(fiber.Map{"error": "amount must be a non-negative integer"})
		}
		amount = &parsedAmount
	}
	proof, err := h.service.AddFinancialRecordProof(c.Context(), userID, recordID, fileURL, c.FormValue("proof_type"), amount)
	if err != nil {
		_ = h.proofStorage.Delete(fileURL)
		return h.financialRecordError(c, err, "failed to attach proof")
	}
	return c.Status(201).JSON(fiber.Map{"data": proof})
}
func (h *BusinessHandler) DownloadFinancialRecordProof(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "invalid authorization context"})
	}
	proofID, err := uuid.Parse(c.Params("proofID"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid proof id"})
	}
	url, err := h.service.FinancialProofFileURL(c.Context(), userID, proofID)
	if err != nil {
		return h.financialRecordError(c, err, "failed to fetch proof")
	}
	data, err := h.proofStorage.Read(url)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "proof file not found"})
	}
	c.Set(fiber.HeaderContentType, http.DetectContentType(data))
	c.Set(fiber.HeaderContentDisposition, "attachment; filename=financial-proof")
	return c.Send(data)
}

func (h *BusinessHandler) financialRecordError(c *fiber.Ctx, err error, fallback string) error {
	switch {
	case errors.Is(err, service.ErrBusinessNotFound), errors.Is(err, service.ErrFinancialRecordNotFound):
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, service.ErrInvalidRecordAmount):
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	default:
		return c.Status(500).JSON(fiber.Map{"error": fallback})
	}
}

func (h *BusinessHandler) CreateBusiness(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "invalid authorization context"})
	}

	var req service.CreateBusinessInput
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.BusinessName == "" || req.CategoryID <= 0 || req.Description == "" || req.BusinessType == "" || req.LocationAddress == "" {
		return c.Status(400).JSON(fiber.Map{"error": "business_name, category_id, description, business_type, and location_address are required"})
	}

	business, err := h.service.CreateBusiness(c.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrBusinessAlreadyExists):
			return c.Status(409).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, service.ErrInvalidBusinessType), errors.Is(err, service.ErrStarterRequirementMissing):
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(500).JSON(fiber.Map{"error": "failed to create business profile"})
		}
	}

	return c.Status(201).JSON(fiber.Map{"data": business})
}

func (h *BusinessHandler) GetMyBusiness(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "invalid authorization context"})
	}

	business, err := h.service.GetMyBusiness(c.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrBusinessNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch business profile"})
	}

	return c.JSON(fiber.Map{"data": business})
}

func (h *BusinessHandler) UpdateBusiness(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "invalid authorization context"})
	}

	var req service.UpdateBusinessInput
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	business, err := h.service.UpdateBusiness(c.Context(), userID, req)
	if err != nil {
		if errors.Is(err, service.ErrBusinessNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(500).JSON(fiber.Map{"error": "failed to update business profile"})
	}

	return c.JSON(fiber.Map{"data": business})
}

func (h *BusinessHandler) DeactivateBusiness(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "invalid authorization context"})
	}
	if err := h.service.DeactivateBusiness(c.Context(), userID); err != nil {
		if errors.Is(err, service.ErrBusinessNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrBusinessNotActive) {
			return c.Status(409).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(500).JSON(fiber.Map{"error": "failed to deactivate business profile"})
	}
	return c.JSON(fiber.Map{"message": "business profile deactivated successfully"})
}

func (h *BusinessHandler) CreateFinancialRecord(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "invalid authorization context"})
	}

	var req service.CreateFinancialRecordInput
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	record, err := h.service.CreateFinancialRecord(c.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrBusinessNotFound):
			return c.Status(404).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, service.ErrInvalidRecordAmount):
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(500).JSON(fiber.Map{"error": "failed to create financial record"})
		}
	}

	return c.Status(201).JSON(fiber.Map{"data": record})
}

func (h *BusinessHandler) GetFinancialRecords(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "invalid authorization context"})
	}

	month, _ := strconv.Atoi(c.Query("month", "0"))
	year, _ := strconv.Atoi(c.Query("year", "0"))
	recordType := c.Query("type", "")

	records, err := h.service.GetFinancialRecords(c.Context(), userID, month, year, recordType)
	if err != nil {
		if errors.Is(err, service.ErrBusinessNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch financial records"})
	}

	return c.JSON(fiber.Map{"data": records})
}

func (h *BusinessHandler) GetFinancialSummary(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "invalid authorization context"})
	}

	month, _ := strconv.Atoi(c.Query("month", "0"))
	year, _ := strconv.Atoi(c.Query("year", "0"))

	summary, err := h.service.GetFinancialSummary(c.Context(), userID, month, year)
	if err != nil {
		if errors.Is(err, service.ErrBusinessNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch financial summary"})
	}

	return c.JSON(fiber.Map{"data": summary})
}

func (h *BusinessHandler) DeleteFinancialRecord(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "invalid authorization context"})
	}

	recordID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid record id"})
	}

	if err := h.service.DeleteFinancialRecord(c.Context(), userID, recordID); err != nil {
		if errors.Is(err, service.ErrBusinessNotFound) || errors.Is(err, service.ErrFinancialRecordNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(500).JSON(fiber.Map{"error": "failed to delete financial record"})
	}

	return c.JSON(fiber.Map{"message": "financial record deleted successfully"})
}
