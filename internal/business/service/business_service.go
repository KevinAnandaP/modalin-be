package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"modalin-be/internal/business/repository"
	"modalin-be/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrBusinessAlreadyExists     = errors.New("user already owns an active business")
	ErrBusinessNotFound          = errors.New("business profile not found")
	ErrBusinessNotActive         = errors.New("business profile is not active")
	ErrInvalidBusinessType       = errors.New("invalid business type, must be 'running' or 'starter'")
	ErrStarterRequirementMissing = errors.New("starter business requires target market, supplier info, pricing estimation, readiness proof, and financial commitment")
	ErrInvalidRecordAmount       = errors.New("either income_amount or expense_amount must be greater than zero")
	ErrFinancialRecordNotFound   = errors.New("financial record not found")
)

type ProofInput struct {
	FileURL   string `json:"file_url"`
	ProofType string `json:"proof_type"` // receipt, invoice, photo, transfer, other
	Amount    *int64 `json:"amount"`
}

type CreateBusinessInput struct {
	BusinessName    string     `json:"business_name"`
	CategoryID      int        `json:"category_id"`
	Description     string     `json:"description"`
	BusinessType    string     `json:"business_type"` // running, starter
	StartedAt       *time.Time `json:"started_at"`
	LocationAddress string     `json:"location_address"`
	Latitude        *float64   `json:"latitude"`
	Longitude       *float64   `json:"longitude"`
	PhotoURL        *string    `json:"photo_url"`

	// Starter Business Details
	TargetMarket      string  `json:"target_market"`
	SupplierInfo      string  `json:"supplier_info"`
	PricingEstimation string  `json:"pricing_estimation"`
	ReadinessProofURL *string `json:"readiness_proof_url"`
	CommitmentChecked bool    `json:"commitment_checked"`
}

type UpdateBusinessInput struct {
	BusinessName    string     `json:"business_name"`
	CategoryID      int        `json:"category_id"`
	Description     string     `json:"description"`
	StartedAt       *time.Time `json:"started_at"`
	LocationAddress string     `json:"location_address"`
	Latitude        *float64   `json:"latitude"`
	Longitude       *float64   `json:"longitude"`
	PhotoURL        *string    `json:"photo_url"`

	TargetMarket      string  `json:"target_market"`
	SupplierInfo      string  `json:"supplier_info"`
	PricingEstimation string  `json:"pricing_estimation"`
	ReadinessProofURL *string `json:"readiness_proof_url"`
	CommitmentChecked bool    `json:"commitment_checked"`
}

type CreateFinancialRecordInput struct {
	RecordDate    *time.Time   `json:"record_date"`
	IncomeAmount  int64        `json:"income_amount"`
	ExpenseAmount int64        `json:"expense_amount"`
	Note          *string      `json:"note"`
	Proofs        []ProofInput `json:"proofs"`
}

type UpdateFinancialRecordInput struct {
	RecordDate    *time.Time `json:"record_date"`
	IncomeAmount  int64      `json:"income_amount"`
	ExpenseAmount int64      `json:"expense_amount"`
	Note          *string    `json:"note"`
}

type BusinessService struct {
	repo repository.Repository
}

func NewBusinessService(repo repository.Repository) *BusinessService {
	return &BusinessService{repo: repo}
}

func (s *BusinessService) CreateBusiness(ctx context.Context, userID uuid.UUID, in CreateBusinessInput) (*model.Business, error) {
	// 1. Check if user already owns a business (1 User = 1 Business restriction)
	existing, err := s.repo.GetBusinessByUserID(ctx, userID)
	if err == nil && existing != nil && existing.Status != "inactive" {
		return nil, ErrBusinessAlreadyExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	bType := strings.ToLower(strings.TrimSpace(in.BusinessType))
	if bType != "running" && bType != "starter" {
		return nil, ErrInvalidBusinessType
	}

	var starterDetail *model.StarterBusinessDetail
	var initialLimit int64

	if bType == "starter" {
		if strings.TrimSpace(in.TargetMarket) == "" ||
			strings.TrimSpace(in.SupplierInfo) == "" ||
			strings.TrimSpace(in.PricingEstimation) == "" ||
			!in.CommitmentChecked {
			return nil, ErrStarterRequirementMissing
		}
		starterDetail = &model.StarterBusinessDetail{
			TargetMarket:      strings.TrimSpace(in.TargetMarket),
			SupplierInfo:      strings.TrimSpace(in.SupplierInfo),
			PricingEstimation: strings.TrimSpace(in.PricingEstimation),
			ReadinessProofURL: in.ReadinessProofURL,
			CommitmentChecked: in.CommitmentChecked,
		}
		initialLimit = 300000 // Starter default tier Rp300.000
	} else {
		initialLimit = 500000 // Running default tier Rp500.000
	}

	business := &model.Business{
		UserID:                userID,
		BusinessName:          strings.TrimSpace(in.BusinessName),
		CategoryID:            in.CategoryID,
		Description:           strings.TrimSpace(in.Description),
		BusinessType:          bType,
		Status:                "active",
		StartedAt:             in.StartedAt,
		LocationAddress:       strings.TrimSpace(in.LocationAddress),
		Latitude:              in.Latitude,
		Longitude:             in.Longitude,
		PhotoURL:              in.PhotoURL,
		VerificationStatus:    "unverified",
		TrustScore:            0,
		CurrentBorrowingLimit: initialLimit,
	}

	if err := s.repo.CreateBusiness(ctx, business, starterDetail); err != nil {
		return nil, err
	}

	return s.repo.GetBusinessByID(ctx, business.ID)
}

func (s *BusinessService) DeactivateBusiness(ctx context.Context, userID uuid.UUID) error {
	business, err := s.repo.GetBusinessByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrBusinessNotFound
		}
		return err
	}
	if business.Status == "inactive" {
		return ErrBusinessNotActive
	}
	business.Status = "inactive"
	return s.repo.UpdateBusiness(ctx, business, nil)
}

func (s *BusinessService) GetMyBusiness(ctx context.Context, userID uuid.UUID) (*model.Business, error) {
	business, err := s.repo.GetBusinessByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBusinessNotFound
		}
		return nil, err
	}

	// Recalculate limit based on running months and financial record months
	monthsRunning := 0
	if business.StartedAt != nil {
		monthsRunning = calculateMonthsBetween(*business.StartedAt, time.Now())
	}
	monthsRecords, _ := s.repo.CountDistinctFinancialRecordMonths(ctx, business.ID)
	eligibleLimit := s.CalculateBorrowingLimit(business.BusinessType, monthsRunning, monthsRecords, business.TrustScore)

	if eligibleLimit > business.CurrentBorrowingLimit {
		business.CurrentBorrowingLimit = eligibleLimit
		_ = s.repo.UpdateBusiness(ctx, business, nil)
	}

	return business, nil
}

func (s *BusinessService) UpdateBusiness(ctx context.Context, userID uuid.UUID, in UpdateBusinessInput) (*model.Business, error) {
	business, err := s.repo.GetBusinessByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBusinessNotFound
		}
		return nil, err
	}

	business.BusinessName = strings.TrimSpace(in.BusinessName)
	if in.CategoryID > 0 {
		business.CategoryID = in.CategoryID
	}
	business.Description = strings.TrimSpace(in.Description)
	if in.StartedAt != nil {
		business.StartedAt = in.StartedAt
	}
	business.LocationAddress = strings.TrimSpace(in.LocationAddress)
	if in.Latitude != nil {
		business.Latitude = in.Latitude
	}
	if in.Longitude != nil {
		business.Longitude = in.Longitude
	}
	if in.PhotoURL != nil {
		business.PhotoURL = in.PhotoURL
	}

	var starterDetail *model.StarterBusinessDetail
	if business.BusinessType == "starter" {
		starterDetail = &model.StarterBusinessDetail{
			BusinessID:        business.ID,
			TargetMarket:      strings.TrimSpace(in.TargetMarket),
			SupplierInfo:      strings.TrimSpace(in.SupplierInfo),
			PricingEstimation: strings.TrimSpace(in.PricingEstimation),
			ReadinessProofURL: in.ReadinessProofURL,
			CommitmentChecked: in.CommitmentChecked,
		}
	}

	if err := s.repo.UpdateBusiness(ctx, business, starterDetail); err != nil {
		return nil, err
	}

	return s.repo.GetBusinessByID(ctx, business.ID)
}

func (s *BusinessService) CreateFinancialRecord(ctx context.Context, userID uuid.UUID, in CreateFinancialRecordInput) (*model.FinancialRecord, error) {
	business, err := s.repo.GetBusinessByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBusinessNotFound
		}
		return nil, err
	}

	if in.IncomeAmount <= 0 && in.ExpenseAmount <= 0 {
		return nil, ErrInvalidRecordAmount
	}

	recDate := time.Now().UTC()
	if in.RecordDate != nil {
		recDate = *in.RecordDate
	}

	netAmount := in.IncomeAmount - in.ExpenseAmount

	record := &model.FinancialRecord{
		BusinessID:    business.ID,
		RecordDate:    recDate,
		IncomeAmount:  in.IncomeAmount,
		ExpenseAmount: in.ExpenseAmount,
		NetAmount:     netAmount,
		Note:          in.Note,
		Status:        "submitted",
	}

	proofs := make([]model.FinancialRecordProof, 0, len(in.Proofs))
	for _, p := range in.Proofs {
		if strings.TrimSpace(p.FileURL) != "" {
			pType := strings.ToLower(strings.TrimSpace(p.ProofType))
			if pType == "" {
				pType = "other"
			}
			proofs = append(proofs, model.FinancialRecordProof{
				FileURL:   strings.TrimSpace(p.FileURL),
				ProofType: pType,
				Amount:    p.Amount,
				Status:    "pending",
			})
		}
	}

	if err := s.repo.CreateFinancialRecord(ctx, record, proofs); err != nil {
		return nil, err
	}

	return record, nil
}

func (s *BusinessService) GetFinancialRecords(ctx context.Context, userID uuid.UUID, month, year int, recordType string) ([]model.FinancialRecord, error) {
	business, err := s.repo.GetBusinessByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBusinessNotFound
		}
		return nil, err
	}

	return s.repo.GetFinancialRecords(ctx, business.ID, month, year, recordType)
}

func (s *BusinessService) GetFinancialRecord(ctx context.Context, userID, recordID uuid.UUID) (*model.FinancialRecord, error) {
	business, err := s.repo.GetBusinessByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBusinessNotFound
		}
		return nil, err
	}
	record, err := s.repo.GetFinancialRecordByID(ctx, recordID, business.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFinancialRecordNotFound
	}
	return record, err
}

func (s *BusinessService) UpdateFinancialRecord(ctx context.Context, userID, recordID uuid.UUID, in UpdateFinancialRecordInput) (*model.FinancialRecord, error) {
	record, err := s.GetFinancialRecord(ctx, userID, recordID)
	if err != nil {
		return nil, err
	}
	if in.IncomeAmount <= 0 && in.ExpenseAmount <= 0 {
		return nil, ErrInvalidRecordAmount
	}
	if in.RecordDate != nil {
		record.RecordDate = *in.RecordDate
	}
	record.IncomeAmount = in.IncomeAmount
	record.ExpenseAmount = in.ExpenseAmount
	record.NetAmount = in.IncomeAmount - in.ExpenseAmount
	record.Note = in.Note
	if err := s.repo.UpdateFinancialRecord(ctx, record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *BusinessService) AddFinancialRecordProof(ctx context.Context, userID, recordID uuid.UUID, fileURL, proofType string, amount *int64) (*model.FinancialRecordProof, error) {
	if strings.TrimSpace(fileURL) == "" {
		return nil, errors.New("proof file URL is required")
	}
	if _, err := s.GetFinancialRecord(ctx, userID, recordID); err != nil {
		return nil, err
	}
	proof := &model.FinancialRecordProof{
		FinancialRecordID: recordID,
		FileURL:           strings.TrimSpace(fileURL),
		ProofType:         normalizeProofType(proofType),
		Amount:            amount,
		Status:            "pending",
	}
	if err := s.repo.CreateFinancialRecordProof(ctx, proof); err != nil {
		return nil, err
	}
	return proof, nil
}

func (s *BusinessService) GetFinancialSummary(ctx context.Context, userID uuid.UUID, month, year int) (*repository.FinancialSummary, error) {
	business, err := s.repo.GetBusinessByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBusinessNotFound
		}
		return nil, err
	}

	return s.repo.GetFinancialSummary(ctx, business.ID, month, year)
}

func (s *BusinessService) DeleteFinancialRecord(ctx context.Context, userID uuid.UUID, recordID uuid.UUID) error {
	business, err := s.repo.GetBusinessByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrBusinessNotFound
		}
		return err
	}

	err = s.repo.DeleteFinancialRecord(ctx, recordID, business.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrFinancialRecordNotFound
	}
	return err
}

func normalizeProofType(proofType string) string {
	proofType = strings.ToLower(strings.TrimSpace(proofType))
	switch proofType {
	case "receipt", "invoice", "photo", "transfer", "other":
		return proofType
	default:
		return "other"
	}
}

func (s *BusinessService) CalculateBorrowingLimit(businessType string, monthsRunning int, monthsFinancialRecords int, repaymentScore int) int64 {
	if businessType == "starter" {
		if monthsFinancialRecords >= 3 {
			return 1500000 // Rp1.500.000 max starter
		} else if monthsFinancialRecords >= 2 {
			return 1000000 // Rp1.000.000
		}
		return 300000 // Rp300.000 default starter
	}

	// Modal Usaha Berjalan Tiering:
	if monthsRunning >= 10 && monthsFinancialRecords >= 10 && repaymentScore >= 80 {
		return 15000000 // Tier 4: Rp15.000.000
	} else if monthsRunning >= 5 && monthsFinancialRecords >= 5 {
		return 5000000 // Tier 3: Rp5.000.000
	} else if monthsRunning >= 3 && monthsFinancialRecords >= 2 {
		return 3000000 // Tier 2: Rp3.000.000
	}
	return 1000000 // Tier 1: Rp500.000 - Rp1.000.000
}

func calculateMonthsBetween(start, end time.Time) int {
	months := (end.Year()-start.Year())*12 + int(end.Month()-start.Month())
	if months < 0 {
		return 0
	}
	return months
}
