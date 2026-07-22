package handler

import (
	"errors"
	"mime/multipart"
	"net/http"
	"strconv"

	"modalin-be/internal/campaign/repository"
	"modalin-be/internal/campaign/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type proofStorage interface {
	Store(*multipart.FileHeader) (string, error)
	Delete(string) error
	Read(string) ([]byte, error)
}

func (h *CampaignHandler) DownloadFundUsageProof(c *fiber.Ctx) error {
	if h.storage == nil {
		return c.Status(500).JSON(fiber.Map{"error": "proof storage is unavailable"})
	}
	campaignID, err := id(c, "id")
	if err != nil {
		return bad(c, "invalid campaign id")
	}
	proofID, err := id(c, "proofID")
	if err != nil {
		return bad(c, "invalid proof id")
	}
	roles, _ := c.Locals("roles").([]string)
	url, err := h.service.ProofFileURL(c.Context(), user(c), campaignID, proofID, roles)
	if err != nil {
		return h.result(c, nil, err, 200)
	}
	data, err := h.storage.Read(url)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "proof file not found"})
	}
	c.Set(fiber.HeaderContentType, http.DetectContentType(data))
	c.Set(fiber.HeaderContentDisposition, "attachment; filename=proof")
	return c.Send(data)
}

type CampaignHandler struct {
	service *service.CampaignService
	storage proofStorage
}

func NewCampaignHandler(s *service.CampaignService, stores ...proofStorage) *CampaignHandler {
	h := &CampaignHandler{service: s}
	if len(stores) > 0 {
		h.storage = stores[0]
	}
	return h
}
func (h *CampaignHandler) Create(c *fiber.Ctx) error {
	var in service.CampaignInput
	if err := c.BodyParser(&in); err != nil {
		return bad(c, "invalid request body")
	}
	v, err := h.service.CreateCampaign(c.Context(), user(c), in)
	return h.result(c, v, err, 201)
}
func (h *CampaignHandler) ListMine(c *fiber.Ctx) error {
	v, err := h.service.ListOwnCampaigns(c.Context(), user(c))
	return h.result(c, v, err, 200)
}
func (h *CampaignHandler) GetMine(c *fiber.Ctx) error {
	id, err := id(c, "id")
	if err != nil {
		return bad(c, "invalid campaign id")
	}
	v, err := h.service.GetOwnCampaign(c.Context(), user(c), id)
	return h.result(c, v, err, 200)
}
func (h *CampaignHandler) Update(c *fiber.Ctx) error {
	id, err := id(c, "id")
	if err != nil {
		return bad(c, "invalid campaign id")
	}
	var in service.CampaignInput
	if err := c.BodyParser(&in); err != nil {
		return bad(c, "invalid request body")
	}
	v, err := h.service.UpdateCampaign(c.Context(), user(c), id, in)
	return h.result(c, v, err, 200)
}
func (h *CampaignHandler) Delete(c *fiber.Ctx) error {
	cid, err := id(c, "id")
	if err != nil {
		return bad(c, "invalid campaign id")
	}
	return h.result(c, nil, h.service.DeleteCampaign(c.Context(), user(c), cid), 200)
}
func (h *CampaignHandler) Submit(c *fiber.Ctx) error {
	cid, err := id(c, "id")
	if err != nil {
		return bad(c, "invalid campaign id")
	}
	return h.result(c, nil, h.service.SubmitCampaign(c.Context(), user(c), cid), 200)
}
func (h *CampaignHandler) CreateBudget(c *fiber.Ctx) error {
	cid, err := id(c, "id")
	if err != nil {
		return bad(c, "invalid campaign id")
	}
	var in service.BudgetItemInput
	if err := c.BodyParser(&in); err != nil {
		return bad(c, "invalid request body")
	}
	v, err := h.service.CreateBudgetItem(c.Context(), user(c), cid, in)
	return h.result(c, v, err, 201)
}
func (h *CampaignHandler) ListBudget(c *fiber.Ctx) error {
	cid, err := id(c, "id")
	if err != nil {
		return bad(c, "invalid campaign id")
	}
	v, err := h.service.ListBudgetItems(c.Context(), user(c), cid)
	return h.result(c, v, err, 200)
}
func (h *CampaignHandler) UpdateBudget(c *fiber.Ctx) error {
	cid, err := id(c, "id")
	if err != nil {
		return bad(c, "invalid campaign id")
	}
	bid, err := id(c, "budgetID")
	if err != nil {
		return bad(c, "invalid budget item id")
	}
	var in service.BudgetItemInput
	if err := c.BodyParser(&in); err != nil {
		return bad(c, "invalid request body")
	}
	v, err := h.service.UpdateBudgetItem(c.Context(), user(c), cid, bid, in)
	return h.result(c, v, err, 200)
}
func (h *CampaignHandler) DeleteBudget(c *fiber.Ctx) error {
	cid, err := id(c, "id")
	if err != nil {
		return bad(c, "invalid campaign id")
	}
	bid, err := id(c, "budgetID")
	if err != nil {
		return bad(c, "invalid budget item id")
	}
	return h.result(c, nil, h.service.DeleteBudgetItem(c.Context(), user(c), cid, bid), 200)
}
func (h *CampaignHandler) CreateMilestone(c *fiber.Ctx) error {
	cid, err := id(c, "id")
	if err != nil {
		return bad(c, "invalid campaign id")
	}
	var in service.MilestoneInput
	if err := c.BodyParser(&in); err != nil {
		return bad(c, "invalid request body")
	}
	v, err := h.service.CreateMilestone(c.Context(), user(c), cid, in)
	return h.result(c, v, err, 201)
}
func (h *CampaignHandler) ListMilestones(c *fiber.Ctx) error {
	cid, err := id(c, "id")
	if err != nil {
		return bad(c, "invalid campaign id")
	}
	v, err := h.service.ListMilestones(c.Context(), user(c), cid)
	return h.result(c, v, err, 200)
}
func (h *CampaignHandler) UpdateMilestone(c *fiber.Ctx) error {
	cid, err := id(c, "id")
	if err != nil {
		return bad(c, "invalid campaign id")
	}
	mid, err := id(c, "milestoneID")
	if err != nil {
		return bad(c, "invalid milestone id")
	}
	var in service.MilestoneInput
	if err := c.BodyParser(&in); err != nil {
		return bad(c, "invalid request body")
	}
	v, err := h.service.UpdateMilestone(c.Context(), user(c), cid, mid, in)
	return h.result(c, v, err, 200)
}
func (h *CampaignHandler) DeleteMilestone(c *fiber.Ctx) error {
	cid, err := id(c, "id")
	if err != nil {
		return bad(c, "invalid campaign id")
	}
	mid, err := id(c, "milestoneID")
	if err != nil {
		return bad(c, "invalid milestone id")
	}
	return h.result(c, nil, h.service.DeleteMilestone(c.Context(), user(c), cid, mid), 200)
}
func (h *CampaignHandler) Catalog(c *fiber.Ctx) error {
	min, _ := strconv.ParseInt(c.Query("min_amount"), 10, 64)
	max, _ := strconv.ParseInt(c.Query("max_amount"), 10, 64)
	v, err := h.service.GetCatalog(c.Context(), repository.CatalogFilter{Query: c.Query("search"), Category: c.Query("category"), RiskLevel: c.Query("risk_level"), MinAmount: min, MaxAmount: max})
	return h.result(c, v, err, 200)
}
func (h *CampaignHandler) Review(c *fiber.Ctx) error {
	cid, err := id(c, "id")
	if err != nil {
		return bad(c, "invalid campaign id")
	}
	var req struct {
		Decision string `json:"decision"`
	}
	if err := c.BodyParser(&req); err != nil {
		return bad(c, "invalid request body")
	}
	return h.result(c, nil, h.service.ReviewCampaign(c.Context(), user(c), cid, req.Decision), 200)
}
func (h *CampaignHandler) Pledge(c *fiber.Ctx) error {
	campaignID, err := id(c, "id")
	if err != nil {
		return bad(c, "invalid campaign id")
	}
	var request struct {
		Amount int64 `json:"amount"`
	}
	if err := c.BodyParser(&request); err != nil {
		return bad(c, "invalid request body")
	}
	v, err := h.service.Pledge(c.Context(), user(c), campaignID, request.Amount)
	return h.result(c, v, err, fiber.StatusCreated)
}
func (h *CampaignHandler) ListLenderFundings(c *fiber.Ctx) error {
	v, err := h.service.ListLenderFundings(c.Context(), user(c))
	return h.result(c, v, err, 200)
}
func (h *CampaignHandler) ListDisbursements(c *fiber.Ctx) error {
	cid, err := id(c, "id")
	if err != nil {
		return bad(c, "invalid campaign id")
	}
	v, err := h.service.ListOwnDisbursements(c.Context(), user(c), cid)
	return h.result(c, v, err, 200)
}
func (h *CampaignHandler) ListFundUsageProofs(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	v, err := h.service.ListFundUsageProofs(c.Context(), c.Query("status"), limit, offset)
	return h.result(c, v, err, 200)
}
func (h *CampaignHandler) UploadFundUsageProof(c *fiber.Ctx) error {
	if h.storage == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "proof storage is unavailable"})
	}
	campaignID, err := id(c, "id")
	if err != nil {
		return bad(c, "invalid campaign id")
	}
	disbursementID, err := id(c, "disbursementID")
	if err != nil {
		return bad(c, "invalid disbursement id")
	}
	file, err := c.FormFile("file")
	if err != nil {
		return bad(c, "proof file is required")
	}
	amount, err := strconv.ParseInt(c.FormValue("amount"), 10, 64)
	if err != nil {
		return bad(c, "invalid proof amount")
	}
	url, err := h.storage.Store(file)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	proof, err := h.service.SubmitFundUsageProof(c.Context(), user(c), campaignID, disbursementID, service.FundUsageProofInput{FileURL: url, ProofType: c.FormValue("proof_type"), Amount: amount, Note: stringPointer(c.FormValue("note"))})
	if err != nil {
		_ = h.storage.Delete(url)
	}
	return h.result(c, proof, err, fiber.StatusCreated)
}
func (h *CampaignHandler) ReviewFundUsageProof(c *fiber.Ctx) error {
	proofID, err := id(c, "proofID")
	if err != nil {
		return bad(c, "invalid proof id")
	}
	var request struct {
		Decision string `json:"decision"`
	}
	if err := c.BodyParser(&request); err != nil {
		return bad(c, "invalid request body")
	}
	return h.result(c, nil, h.service.ReviewFundUsageProof(c.Context(), user(c), proofID, request.Decision), fiber.StatusOK)
}
func (h *CampaignHandler) result(c *fiber.Ctx, v any, err error, status int) error {
	if err == nil {
		if v == nil {
			return c.Status(status).JSON(fiber.Map{"message": "success"})
		}
		return c.Status(status).JSON(fiber.Map{"data": v})
	}
	switch {
	case errors.Is(err, service.ErrCampaignNotFound), errors.Is(err, service.ErrBudgetItemNotFound), errors.Is(err, service.ErrMilestoneNotFound), errors.Is(err, service.ErrDisbursementNotFound), errors.Is(err, service.ErrFundUsageProofNotFound):
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, service.ErrCampaignLocked), errors.Is(err, service.ErrInvalidStatusTransition), errors.Is(err, service.ErrFundingUnavailable), errors.Is(err, service.ErrProofReviewUnavailable):
		return c.Status(409).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, service.ErrProofAccessDenied):
		return c.Status(403).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, service.ErrAmountExceedsLimit), errors.Is(err, service.ErrInvalidCampaign), errors.Is(err, service.ErrInvalidBudgetItem), errors.Is(err, service.ErrInvalidMilestone), errors.Is(err, service.ErrIncompleteCampaignPlan), errors.Is(err, service.ErrCampaignNeedsRevision), errors.Is(err, service.ErrInvalidReviewDecision), errors.Is(err, service.ErrSelfFunding), errors.Is(err, service.ErrInvalidFundingAmount), errors.Is(err, service.ErrFundingExceedsTarget), errors.Is(err, service.ErrInvalidFundUsageProof), errors.Is(err, service.ErrInvalidProofReview):
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	default:
		return c.Status(500).JSON(fiber.Map{"error": "campaign operation failed"})
	}
}
func user(c *fiber.Ctx) uuid.UUID                     { v, _ := c.Locals("user_id").(uuid.UUID); return v }
func id(c *fiber.Ctx, name string) (uuid.UUID, error) { return uuid.Parse(c.Params(name)) }
func bad(c *fiber.Ctx, msg string) error              { return c.Status(400).JSON(fiber.Map{"error": msg}) }
func stringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
