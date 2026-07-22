package service

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"modalin-be/internal/campaign/repository"
	"modalin-be/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrCampaignNotFound        = errors.New("campaign not found")
	ErrBudgetItemNotFound      = errors.New("budget item not found")
	ErrMilestoneNotFound       = errors.New("milestone not found")
	ErrAmountExceedsLimit      = errors.New("requested amount exceeds current borrowing limit")
	ErrInvalidCampaign         = errors.New("campaign title, description, positive requested amount, tenor, and benefit type are required")
	ErrInvalidBudgetItem       = errors.New("budget item name, category, positive amount, and purchase method are required")
	ErrInvalidMilestone        = errors.New("milestone title, positive amount, and positive sequence number are required")
	ErrCampaignLocked          = errors.New("campaign can only be changed while draft or rejected")
	ErrInvalidStatusTransition = errors.New("invalid campaign status transition")
	ErrIncompleteCampaignPlan  = errors.New("budget and milestone totals must each equal the requested amount")
	ErrInvalidReviewDecision   = errors.New("review decision must be publish or reject")
	ErrCampaignNeedsRevision   = errors.New("campaign needs revision: assign high, medium, or low priority to every budget item")
	ErrSelfFunding             = errors.New("campaign owner cannot fund their own campaign")
	ErrFundingUnavailable      = errors.New("campaign is not accepting funding")
	ErrInvalidFundingAmount    = errors.New("funding amount must be positive")
	ErrFundingExceedsTarget    = errors.New("funding amount exceeds campaign target")
	ErrDisbursementNotFound    = errors.New("disbursement not found")
	ErrFundUsageProofNotFound  = errors.New("fund usage proof not found")
	ErrInvalidFundUsageProof   = errors.New("file URL, supported proof type, and positive amount are required")
	ErrInvalidProofReview      = errors.New("proof review decision must be approve, reject, or revision_needed")
	ErrProofReviewUnavailable  = errors.New("fund usage proof is not pending review")
	ErrProofAccessDenied       = errors.New("not authorized to access fund usage proof")
	ErrProofAlreadySubmitted   = errors.New("fund usage proof is already awaiting review or approved")
)

type CampaignInput struct {
	Title               string   `json:"title"`
	Description         string   `json:"description"`
	RequestedAmount     int64    `json:"requested_amount"`
	LoanTenorMonths     int      `json:"loan_tenor_months"`
	BenefitType         string   `json:"benefit_type"`
	MarginPercent       *float64 `json:"margin_percent"`
	RevenueSharePercent *float64 `json:"revenue_share_percent"`
	ReturnCapPercent    *float64 `json:"return_cap_percent"`
	RiskLevel           string   `json:"risk_level"`
}
type BudgetItemInput struct {
	ItemName       string  `json:"item_name"`
	Category       string  `json:"category"`
	Amount         int64   `json:"amount"`
	PriorityLevel  string  `json:"priority_level"`
	PurchaseMethod string  `json:"purchase_method"`
	Note           *string `json:"note"`
}
type MilestoneInput struct {
	Title      string     `json:"title"`
	Amount     int64      `json:"amount"`
	SequenceNo int        `json:"sequence_no"`
	DueDate    *time.Time `json:"due_date"`
}
type FundUsageProofInput struct {
	FileURL   string  `json:"file_url"`
	ProofType string  `json:"proof_type"`
	Amount    int64   `json:"amount"`
	Note      *string `json:"note"`
}
type CampaignService struct{ repo repository.Repository }

func NewCampaignService(repo repository.Repository) *CampaignService {
	return &CampaignService{repo: repo}
}

func (s *CampaignService) CreateCampaign(ctx context.Context, userID uuid.UUID, in CampaignInput) (*model.LoanCampaign, error) {
	b, err := s.business(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := validateCampaign(in, b.CurrentBorrowingLimit); err != nil {
		return nil, err
	}
	c := &model.LoanCampaign{BusinessID: b.ID, Title: strings.TrimSpace(in.Title), Description: strings.TrimSpace(in.Description), RequestedAmount: in.RequestedAmount, LoanTenorMonths: in.LoanTenorMonths, BenefitType: strings.TrimSpace(in.BenefitType), MarginPercent: in.MarginPercent, RevenueSharePercent: in.RevenueSharePercent, ReturnCapPercent: in.ReturnCapPercent, RiskLevel: defaultRisk(in.RiskLevel), Status: "draft"}
	return c, s.repo.CreateCampaign(ctx, c)
}
func (s *CampaignService) UpdateCampaign(ctx context.Context, userID, id uuid.UUID, in CampaignInput) (*model.LoanCampaign, error) {
	b, c, err := s.owned(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if !editable(c.Status) {
		return nil, ErrCampaignLocked
	}
	if err := validateCampaign(in, b.CurrentBorrowingLimit); err != nil {
		return nil, err
	}
	c.Title = strings.TrimSpace(in.Title)
	c.Description = strings.TrimSpace(in.Description)
	c.RequestedAmount = in.RequestedAmount
	c.LoanTenorMonths = in.LoanTenorMonths
	c.BenefitType = strings.TrimSpace(in.BenefitType)
	c.MarginPercent = in.MarginPercent
	c.RevenueSharePercent = in.RevenueSharePercent
	c.ReturnCapPercent = in.ReturnCapPercent
	c.RiskLevel = defaultRisk(in.RiskLevel)
	return c, s.repo.UpdateCampaign(ctx, c)
}
func (s *CampaignService) GetOwnCampaign(ctx context.Context, userID, id uuid.UUID) (*model.LoanCampaign, error) {
	_, c, err := s.owned(ctx, userID, id)
	return c, err
}
func (s *CampaignService) ListOwnCampaigns(ctx context.Context, userID uuid.UUID) ([]model.LoanCampaign, error) {
	b, err := s.business(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListCampaignsForBusiness(ctx, b.ID)
}
func (s *CampaignService) DeleteCampaign(ctx context.Context, userID, id uuid.UUID) error {
	_, c, err := s.owned(ctx, userID, id)
	if err != nil {
		return err
	}
	if !editable(c.Status) {
		return ErrCampaignLocked
	}
	return s.repo.DeleteCampaign(ctx, id)
}
func (s *CampaignService) SubmitCampaign(ctx context.Context, userID, id uuid.UUID) error {
	b, c, err := s.owned(ctx, userID, id)
	if err != nil {
		return err
	}
	if !editable(c.Status) {
		return ErrInvalidStatusTransition
	}
	if c.RequestedAmount > b.CurrentBorrowingLimit {
		return ErrAmountExceedsLimit
	}
	items, err := s.validateBudgetPlan(ctx, c)
	if err != nil {
		return err
	}
	if err := s.repo.ReplaceMilestones(ctx, c.ID, buildMilestones(c.ID, items)); err != nil {
		return err
	}
	c.Status = "admin_review"
	return s.repo.UpdateCampaign(ctx, c)
}
func (s *CampaignService) ReviewCampaign(ctx context.Context, adminID, id uuid.UUID, decision string) error {
	c, err := s.repo.GetCampaignByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrCampaignNotFound
	}
	if err != nil {
		return err
	}
	if c.Status != "admin_review" {
		return ErrInvalidStatusTransition
	}
	decision = strings.ToLower(strings.TrimSpace(decision))
	if decision == "publish" {
		if err := s.validatePlan(ctx, c); err != nil {
			return err
		}
		c.Status = "published"
		c.ApprovedBy = &adminID
		now := time.Now().UTC()
		c.ApprovedAt = &now
	} else if decision == "reject" {
		if err := s.repo.ReplaceMilestones(ctx, c.ID, nil); err != nil {
			return err
		}
		c.Status = "rejected"
	} else {
		return ErrInvalidReviewDecision
	}
	return s.repo.UpdateCampaign(ctx, c)
}
func (s *CampaignService) CreateBudgetItem(ctx context.Context, userID, campaignID uuid.UUID, in BudgetItemInput) (*model.CampaignBudgetItem, error) {
	_, c, err := s.owned(ctx, userID, campaignID)
	if err != nil {
		return nil, err
	}
	if !editable(c.Status) {
		return nil, ErrCampaignLocked
	}
	if err := validateBudget(in); err != nil {
		return nil, err
	}
	entryOrder, err := s.repo.CountBudgetItems(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	b := &model.CampaignBudgetItem{CampaignID: campaignID, ItemName: strings.TrimSpace(in.ItemName), Category: strings.TrimSpace(in.Category), Amount: in.Amount, PriorityLevel: normalizePriorityLevel(in.PriorityLevel), EntryOrder: entryOrder + 1, PurchaseMethod: strings.TrimSpace(in.PurchaseMethod), Note: in.Note}
	return b, s.repo.CreateBudgetItem(ctx, b)
}
func (s *CampaignService) ListBudgetItems(ctx context.Context, userID, campaignID uuid.UUID) ([]model.CampaignBudgetItem, error) {
	if _, _, err := s.owned(ctx, userID, campaignID); err != nil {
		return nil, err
	}
	return s.repo.ListBudgetItems(ctx, campaignID)
}
func (s *CampaignService) UpdateBudgetItem(ctx context.Context, userID, campaignID, id uuid.UUID, in BudgetItemInput) (*model.CampaignBudgetItem, error) {
	_, c, err := s.owned(ctx, userID, campaignID)
	if err != nil {
		return nil, err
	}
	if !editable(c.Status) {
		return nil, ErrCampaignLocked
	}
	if err := validateBudget(in); err != nil {
		return nil, err
	}
	b, err := s.repo.GetBudgetItem(ctx, id, campaignID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrBudgetItemNotFound
	}
	if err != nil {
		return nil, err
	}
	b.ItemName = strings.TrimSpace(in.ItemName)
	b.Category = strings.TrimSpace(in.Category)
	b.Amount = in.Amount
	b.PriorityLevel = normalizePriorityLevel(in.PriorityLevel)
	b.PurchaseMethod = strings.TrimSpace(in.PurchaseMethod)
	b.Note = in.Note
	return b, s.repo.UpdateBudgetItem(ctx, b)
}
func (s *CampaignService) DeleteBudgetItem(ctx context.Context, userID, campaignID, id uuid.UUID) error {
	_, c, err := s.owned(ctx, userID, campaignID)
	if err != nil {
		return err
	}
	if !editable(c.Status) {
		return ErrCampaignLocked
	}
	return s.repo.DeleteBudgetItem(ctx, id, campaignID)
}
func (s *CampaignService) CreateMilestone(ctx context.Context, userID, campaignID uuid.UUID, in MilestoneInput) (*model.CampaignMilestone, error) {
	_, c, err := s.owned(ctx, userID, campaignID)
	if err != nil {
		return nil, err
	}
	if !editable(c.Status) {
		return nil, ErrCampaignLocked
	}
	if err := validateMilestone(in); err != nil {
		return nil, err
	}
	m := &model.CampaignMilestone{CampaignID: campaignID, Title: strings.TrimSpace(in.Title), Amount: in.Amount, SequenceNo: in.SequenceNo, DueDate: in.DueDate, Status: "locked"}
	return m, s.repo.CreateMilestone(ctx, m)
}
func (s *CampaignService) ListMilestones(ctx context.Context, userID, campaignID uuid.UUID) ([]model.CampaignMilestone, error) {
	if _, _, err := s.owned(ctx, userID, campaignID); err != nil {
		return nil, err
	}
	return s.repo.ListMilestones(ctx, campaignID)
}
func (s *CampaignService) UpdateMilestone(ctx context.Context, userID, campaignID, id uuid.UUID, in MilestoneInput) (*model.CampaignMilestone, error) {
	_, c, err := s.owned(ctx, userID, campaignID)
	if err != nil {
		return nil, err
	}
	if !editable(c.Status) {
		return nil, ErrCampaignLocked
	}
	if err := validateMilestone(in); err != nil {
		return nil, err
	}
	m, err := s.repo.GetMilestone(ctx, id, campaignID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMilestoneNotFound
	}
	if err != nil {
		return nil, err
	}
	m.Title = strings.TrimSpace(in.Title)
	m.Amount = in.Amount
	m.SequenceNo = in.SequenceNo
	m.DueDate = in.DueDate
	return m, s.repo.UpdateMilestone(ctx, m)
}
func (s *CampaignService) DeleteMilestone(ctx context.Context, userID, campaignID, id uuid.UUID) error {
	_, c, err := s.owned(ctx, userID, campaignID)
	if err != nil {
		return err
	}
	if !editable(c.Status) {
		return ErrCampaignLocked
	}
	return s.repo.DeleteMilestone(ctx, id, campaignID)
}
func (s *CampaignService) GetCatalog(ctx context.Context, f repository.CatalogFilter) ([]model.LoanCampaign, error) {
	return s.repo.ListCatalog(ctx, f)
}
func (s *CampaignService) ListLenderFundings(ctx context.Context, userID uuid.UUID) ([]model.Funding, error) {
	return s.repo.ListFundingsForLender(ctx, userID)
}
func (s *CampaignService) ListOwnDisbursements(ctx context.Context, userID, campaignID uuid.UUID) ([]model.Disbursement, error) {
	if _, _, err := s.owned(ctx, userID, campaignID); err != nil {
		return nil, err
	}
	return s.repo.ListDisbursements(ctx, campaignID)
}
func (s *CampaignService) ListFundUsageProofs(ctx context.Context, status string, limit, offset int) ([]model.FundUsageProof, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListFundUsageProofs(ctx, status, limit, offset)
}
func (s *CampaignService) ProofFileURL(ctx context.Context, userID, campaignID, proofID uuid.UUID, roles []string) (string, error) {
	proof, err := s.repo.GetFundUsageProof(ctx, proofID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrFundUsageProofNotFound
	}
	if err != nil {
		return "", err
	}
	if proof.CampaignID != campaignID {
		return "", ErrFundUsageProofNotFound
	}
	for _, role := range roles {
		if role == "admin" {
			return proof.FileURL, nil
		}
	}
	if _, _, err := s.owned(ctx, userID, campaignID); err == nil {
		return proof.FileURL, nil
	}
	funded, err := s.repo.HasPaidFunding(ctx, campaignID, userID)
	if err != nil {
		return "", err
	}
	if !funded {
		return "", ErrProofAccessDenied
	}
	return proof.FileURL, nil
}
func (s *CampaignService) Pledge(ctx context.Context, lenderID, campaignID uuid.UUID, amount int64) (*model.Funding, error) {
	if amount <= 0 {
		return nil, ErrInvalidFundingAmount
	}
	var funding *model.Funding
	err := s.repo.WithTransaction(ctx, func(repo repository.Repository) error {
		c, err := repo.GetCampaignByIDForUpdate(ctx, campaignID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCampaignNotFound
		}
		if err != nil {
			return err
		}
		business, err := repo.GetBusinessByID(ctx, c.BusinessID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCampaignNotFound
		}
		if err != nil {
			return err
		}
		if business.UserID == lenderID {
			return ErrSelfFunding
		}
		if c.Status != "published" {
			return ErrFundingUnavailable
		}
		if amount > c.RequestedAmount-c.FundedAmount {
			return ErrFundingExceedsTarget
		}

		now := time.Now().UTC()
		funding = &model.Funding{CampaignID: c.ID, LenderUserID: lenderID, Amount: amount, Status: "paid", FundedAt: now}
		if err := repo.CreateFunding(ctx, funding); err != nil {
			return err
		}
		if err := audit(ctx, repo, &lenderID, "funding.created", "funding", funding.ID, nil, map[string]any{"campaign_id": c.ID, "amount": amount, "status": funding.Status}); err != nil {
			return err
		}
		c.FundedAmount += amount
		if c.FundedAmount == c.RequestedAmount {
			c.Status = "funded"
		}
		if err := repo.UpdateCampaign(ctx, c); err != nil {
			return err
		}
		if c.Status != "funded" {
			return nil
		}
		if err := s.releaseNextMilestone(ctx, repo, c); err != nil {
			return err
		}
		c.Status = "active"
		return repo.UpdateCampaign(ctx, c)
	})
	if err != nil {
		return nil, err
	}
	return funding, nil
}
func (s *CampaignService) SubmitFundUsageProof(ctx context.Context, userID, campaignID, disbursementID uuid.UUID, in FundUsageProofInput) (*model.FundUsageProof, error) {
	if !validFundUsageProof(in) {
		return nil, ErrInvalidFundUsageProof
	}
	if _, _, err := s.owned(ctx, userID, campaignID); err != nil {
		return nil, err
	}
	d, err := s.repo.GetDisbursement(ctx, disbursementID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrDisbursementNotFound
	}
	if err != nil {
		return nil, err
	}
	if d.CampaignID != campaignID || d.Status != "proof_required" {
		return nil, ErrInvalidStatusTransition
	}
	if in.Amount != d.Amount {
		return nil, ErrInvalidFundUsageProof
	}
	var proof *model.FundUsageProof
	err = s.repo.WithTransaction(ctx, func(repo repository.Repository) error {
		existing, err := repo.GetActiveFundUsageProof(ctx, d.ID)
		if err == nil {
			if existing.Status == "pending" || existing.Status == "approved" {
				return ErrProofAlreadySubmitted
			}
			if err := repo.DeleteFundUsageProof(ctx, existing.ID); err != nil {
				return err
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		proof = &model.FundUsageProof{DisbursementID: d.ID, CampaignID: campaignID, FileURL: strings.TrimSpace(in.FileURL), ProofType: strings.ToLower(strings.TrimSpace(in.ProofType)), Amount: in.Amount, Note: in.Note, Status: "pending"}
		if err := repo.CreateFundUsageProof(ctx, proof); err != nil {
			return err
		}
		return audit(ctx, repo, &userID, "fund_usage_proof.submitted", "fund_usage_proof", proof.ID, nil, map[string]any{"disbursement_id": d.ID, "amount": proof.Amount, "status": proof.Status})
	})
	if err != nil {
		return nil, err
	}
	return proof, nil
}
func (s *CampaignService) ReviewFundUsageProof(ctx context.Context, adminID, proofID uuid.UUID, decision string) error {
	decision = strings.ToLower(strings.TrimSpace(decision))
	if decision != "approve" && decision != "reject" && decision != "revision_needed" {
		return ErrInvalidProofReview
	}
	return s.repo.WithTransaction(ctx, func(repo repository.Repository) error {
		proof, err := repo.GetFundUsageProofForUpdate(ctx, proofID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrFundUsageProofNotFound
		}
		if err != nil {
			return err
		}
		if proof.Status != "pending" {
			return ErrProofReviewUnavailable
		}
		d, err := repo.GetDisbursementForUpdate(ctx, proof.DisbursementID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrDisbursementNotFound
		}
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		proof.ReviewedBy = &adminID
		proof.ReviewedAt = &now
		if decision != "approve" {
			proof.Status = decision
			if err := repo.UpdateFundUsageProof(ctx, proof); err != nil {
				return err
			}
			return audit(ctx, repo, &adminID, "fund_usage_proof.reviewed", "fund_usage_proof", proof.ID, map[string]any{"status": "pending"}, map[string]any{"status": proof.Status})
		}
		proof.Status = "approved"
		if err := repo.UpdateFundUsageProof(ctx, proof); err != nil {
			return err
		}
		if err := audit(ctx, repo, &adminID, "fund_usage_proof.reviewed", "fund_usage_proof", proof.ID, map[string]any{"status": "pending"}, map[string]any{"status": proof.Status}); err != nil {
			return err
		}
		d.Status = "verified"
		if err := repo.UpdateDisbursement(ctx, d); err != nil {
			return err
		}
		milestone, err := repo.GetMilestone(ctx, d.MilestoneID, d.CampaignID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMilestoneNotFound
		}
		if err != nil {
			return err
		}
		milestone.Status = "verified"
		if err := repo.UpdateMilestone(ctx, milestone); err != nil {
			return err
		}
		c, err := repo.GetCampaignByID(ctx, d.CampaignID)
		if err != nil {
			return err
		}
		return s.releaseNextMilestone(ctx, repo, c)
	})
}
func (s *CampaignService) releaseNextMilestone(ctx context.Context, repo repository.Repository, c *model.LoanCampaign) error {
	milestones, err := repo.ListMilestones(ctx, c.ID)
	if err != nil {
		return err
	}
	available := false
	for _, milestone := range milestones {
		if milestone.Status == "available" {
			available = true
			break
		}
	}
	if !available {
		for i := range milestones {
			if milestones[i].Status == "locked" {
				milestones[i].Status = "available"
				if err := repo.UpdateMilestone(ctx, &milestones[i]); err != nil {
					return err
				}
				break
			}
		}
	}
	// Re-read after unlocking so the repository is the source of truth.
	milestones, err = repo.ListMilestones(ctx, c.ID)
	if err != nil {
		return err
	}
	for _, milestone := range milestones {
		if milestone.Status == "available" {
			now := time.Now().UTC()
			d := &model.Disbursement{CampaignID: c.ID, MilestoneID: milestone.ID, Amount: milestone.Amount, Method: "bank_transfer", RecipientType: "borrower", Status: "proof_required", ReleasedAt: &now}
			if err := repo.CreateDisbursement(ctx, d); err != nil {
				return err
			}
			milestone.Status = "disbursed"
			return repo.UpdateMilestone(ctx, &milestone)
		}
	}
	return nil
}
func (s *CampaignService) business(ctx context.Context, userID uuid.UUID) (*model.Business, error) {
	b, err := s.repo.GetBusinessByUserID(ctx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCampaignNotFound
	}
	return b, err
}
func (s *CampaignService) owned(ctx context.Context, userID, id uuid.UUID) (*model.Business, *model.LoanCampaign, error) {
	b, err := s.business(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	c, err := s.repo.GetCampaignForBusiness(ctx, id, b.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, ErrCampaignNotFound
	}
	return b, c, err
}
func (s *CampaignService) validatePlan(ctx context.Context, c *model.LoanCampaign) error {
	bs, err := s.repo.ListBudgetItems(ctx, c.ID)
	if err != nil {
		return err
	}
	ms, err := s.repo.ListMilestones(ctx, c.ID)
	if err != nil {
		return err
	}
	var bt, mt int64
	for _, b := range bs {
		bt += b.Amount
	}
	for _, m := range ms {
		mt += m.Amount
	}
	if _, err := validateBudgetItems(c, bs); err != nil {
		return err
	}
	if len(ms) != len(bs) || len(bs) == 0 || bt != c.RequestedAmount || mt != c.RequestedAmount {
		return ErrIncompleteCampaignPlan
	}
	return nil
}
func validateCampaign(in CampaignInput, limit int64) error {
	if strings.TrimSpace(in.Title) == "" || strings.TrimSpace(in.Description) == "" || in.RequestedAmount <= 0 || in.LoanTenorMonths <= 0 || strings.TrimSpace(in.BenefitType) == "" {
		return ErrInvalidCampaign
	}
	if in.RequestedAmount > limit {
		return ErrAmountExceedsLimit
	}
	return nil
}
func validateBudget(in BudgetItemInput) error {
	if strings.TrimSpace(in.ItemName) == "" || strings.TrimSpace(in.Category) == "" || in.Amount <= 0 || !validPriorityLevel(in.PriorityLevel) || strings.TrimSpace(in.PurchaseMethod) == "" {
		return ErrInvalidBudgetItem
	}
	return nil
}
func (s *CampaignService) validateBudgetPlan(ctx context.Context, c *model.LoanCampaign) ([]model.CampaignBudgetItem, error) {
	items, err := s.repo.ListBudgetItems(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	return validateBudgetItems(c, items)
}
func validateBudgetItems(c *model.LoanCampaign, items []model.CampaignBudgetItem) ([]model.CampaignBudgetItem, error) {
	var total int64
	for _, item := range items {
		total += item.Amount
		if !validPriorityLevel(item.PriorityLevel) {
			return nil, ErrCampaignNeedsRevision
		}
	}
	if len(items) == 0 || total != c.RequestedAmount {
		return nil, ErrIncompleteCampaignPlan
	}
	return items, nil
}
func buildMilestones(campaignID uuid.UUID, items []model.CampaignBudgetItem) []model.CampaignMilestone {
	sort.SliceStable(items, func(i, j int) bool {
		if priorityRank(items[i].PriorityLevel) == priorityRank(items[j].PriorityLevel) {
			return items[i].EntryOrder < items[j].EntryOrder
		}
		return priorityRank(items[i].PriorityLevel) > priorityRank(items[j].PriorityLevel)
	})
	milestones := make([]model.CampaignMilestone, len(items))
	for i, item := range items {
		status := "locked"
		if i == 0 {
			status = "available"
		}
		budgetItemID := item.ID
		milestones[i] = model.CampaignMilestone{CampaignID: campaignID, BudgetItemID: &budgetItemID, Title: item.ItemName, Amount: item.Amount, SequenceNo: i + 1, Status: status}
	}
	return milestones
}
func normalizePriorityLevel(v string) string { return strings.ToLower(strings.TrimSpace(v)) }
func validPriorityLevel(v string) bool {
	switch normalizePriorityLevel(v) {
	case "high", "medium", "low":
		return true
	default:
		return false
	}
}
func priorityRank(v string) int {
	switch normalizePriorityLevel(v) {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}
func validateMilestone(in MilestoneInput) error {
	if strings.TrimSpace(in.Title) == "" || in.Amount <= 0 || in.SequenceNo <= 0 {
		return ErrInvalidMilestone
	}
	return nil
}
func editable(status string) bool {
	return status == "draft" || status == "rejected" || status == "needs_revision"
}
func defaultRisk(v string) string {
	if strings.TrimSpace(v) == "" {
		return "medium"
	}
	return strings.ToLower(strings.TrimSpace(v))
}
func validFundUsageProof(in FundUsageProofInput) bool {
	if strings.TrimSpace(in.FileURL) == "" || in.Amount <= 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(in.ProofType)) {
	case "invoice", "receipt", "photo", "merchant_confirmation":
		return true
	default:
		return false
	}
}
func audit(ctx context.Context, repo repository.Repository, userID *uuid.UUID, action, entityType string, entityID uuid.UUID, oldValue, newValue any) error {
	encode := func(value any) *string {
		if value == nil {
			return nil
		}
		data, err := json.Marshal(value)
		if err != nil {
			return nil
		}
		encoded := string(data)
		return &encoded
	}
	return repo.CreateAuditLog(ctx, &model.AuditLog{UserID: userID, Action: action, EntityType: entityType, EntityID: entityID, OldValue: encode(oldValue), NewValue: encode(newValue)})
}
