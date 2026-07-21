package handler

import (
	"errors"
	"strconv"

	"modalin-be/internal/campaign/repository"
	"modalin-be/internal/campaign/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type CampaignHandler struct{ service *service.CampaignService }

func NewCampaignHandler(s *service.CampaignService) *CampaignHandler {
	return &CampaignHandler{service: s}
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
func (h *CampaignHandler) result(c *fiber.Ctx, v any, err error, status int) error {
	if err == nil {
		if v == nil {
			return c.Status(status).JSON(fiber.Map{"message": "success"})
		}
		return c.Status(status).JSON(fiber.Map{"data": v})
	}
	switch {
	case errors.Is(err, service.ErrCampaignNotFound), errors.Is(err, service.ErrBudgetItemNotFound), errors.Is(err, service.ErrMilestoneNotFound):
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, service.ErrCampaignLocked), errors.Is(err, service.ErrInvalidStatusTransition):
		return c.Status(409).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, service.ErrAmountExceedsLimit), errors.Is(err, service.ErrInvalidCampaign), errors.Is(err, service.ErrInvalidBudgetItem), errors.Is(err, service.ErrInvalidMilestone), errors.Is(err, service.ErrIncompleteCampaignPlan), errors.Is(err, service.ErrCampaignNeedsRevision), errors.Is(err, service.ErrInvalidReviewDecision):
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	default:
		return c.Status(500).JSON(fiber.Map{"error": "campaign operation failed"})
	}
}
func user(c *fiber.Ctx) uuid.UUID                     { v, _ := c.Locals("user_id").(uuid.UUID); return v }
func id(c *fiber.Ctx, name string) (uuid.UUID, error) { return uuid.Parse(c.Params(name)) }
func bad(c *fiber.Ctx, msg string) error              { return c.Status(400).JSON(fiber.Map{"error": msg}) }
