package service_test

import (
	"context"
	"errors"
	"testing"

	"modalin-be/internal/business/repository"
	"modalin-be/internal/business/service"
	"modalin-be/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type fakeBusinessRepo struct {
	business       *model.Business
	starter        *model.StarterBusinessDetail
	createdRecords []model.FinancialRecord
	recordMonths   int
}

func (r *fakeBusinessRepo) CreateBusiness(_ context.Context, business *model.Business, starter *model.StarterBusinessDetail) error {
	r.business = business
	business.ID = uuid.New()
	r.starter = starter
	return nil
}

func (r *fakeBusinessRepo) GetBusinessByUserID(_ context.Context, userID uuid.UUID) (*model.Business, error) {
	if r.business != nil && r.business.UserID == userID {
		return r.business, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeBusinessRepo) GetBusinessByID(_ context.Context, id uuid.UUID) (*model.Business, error) {
	if r.business != nil && r.business.ID == id {
		return r.business, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeBusinessRepo) UpdateBusiness(_ context.Context, business *model.Business, starter *model.StarterBusinessDetail) error {
	r.business = business
	r.starter = starter
	return nil
}

func (r *fakeBusinessRepo) CreateFinancialRecord(_ context.Context, record *model.FinancialRecord, proofs []model.FinancialRecordProof) error {
	record.ID = uuid.New()
	r.createdRecords = append(r.createdRecords, *record)
	return nil
}

func (r *fakeBusinessRepo) GetFinancialRecords(_ context.Context, _ uuid.UUID, _, _ int, _ string) ([]model.FinancialRecord, error) {
	return r.createdRecords, nil
}

func (r *fakeBusinessRepo) GetFinancialRecordByID(_ context.Context, recordID uuid.UUID, _ uuid.UUID) (*model.FinancialRecord, error) {
	for i := range r.createdRecords {
		if r.createdRecords[i].ID == recordID {
			return &r.createdRecords[i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeBusinessRepo) UpdateFinancialRecord(_ context.Context, record *model.FinancialRecord) error {
	for i := range r.createdRecords {
		if r.createdRecords[i].ID == record.ID {
			r.createdRecords[i] = *record
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (r *fakeBusinessRepo) CreateFinancialRecordProof(_ context.Context, _ *model.FinancialRecordProof) error {
	return nil
}
func (r *fakeBusinessRepo) GetFinancialRecordProof(_ context.Context, _, _ uuid.UUID) (*model.FinancialRecordProof, error) {
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeBusinessRepo) GetFinancialSummary(_ context.Context, _ uuid.UUID, _, _ int) (*repository.FinancialSummary, error) {
	var income, expense int64
	for _, rec := range r.createdRecords {
		income += rec.IncomeAmount
		expense += rec.ExpenseAmount
	}
	return &repository.FinancialSummary{
		TotalIncome:  income,
		TotalExpense: expense,
		NetCashflow:  income - expense,
	}, nil
}

func (r *fakeBusinessRepo) CountDistinctFinancialRecordMonths(_ context.Context, _ uuid.UUID) (int, error) {
	return r.recordMonths, nil
}

func (r *fakeBusinessRepo) DeleteFinancialRecord(_ context.Context, recordID uuid.UUID, _ uuid.UUID) error {
	newRecords := make([]model.FinancialRecord, 0)
	for _, rec := range r.createdRecords {
		if rec.ID != recordID {
			newRecords = append(newRecords, rec)
		}
	}
	r.createdRecords = newRecords
	return nil
}

func TestCreateBusinessRejectsDuplicateForSameUser(t *testing.T) {
	userID := uuid.New()
	repo := &fakeBusinessRepo{
		business: &model.Business{ID: uuid.New(), UserID: userID, BusinessName: "Toko Merdeka"},
	}
	svc := service.NewBusinessService(repo)

	_, err := svc.CreateBusiness(context.Background(), userID, service.CreateBusinessInput{
		BusinessName:    "Toko Baru",
		CategoryID:      1,
		Description:     "Deskripsi",
		BusinessType:    "running",
		LocationAddress: "Jakarta",
	})

	if !errors.Is(err, service.ErrBusinessAlreadyExists) {
		t.Fatalf("expected ErrBusinessAlreadyExists, got %v", err)
	}
}

func TestDeactivateBusinessLetsUserCreateANewActiveBusiness(t *testing.T) {
	userID := uuid.New()
	repo := &fakeBusinessRepo{
		business: &model.Business{ID: uuid.New(), UserID: userID, BusinessName: "Toko Lama", Status: "active"},
	}
	svc := service.NewBusinessService(repo)

	if err := svc.DeactivateBusiness(context.Background(), userID); err != nil {
		t.Fatalf("deactivate business: %v", err)
	}
	if repo.business.Status != "inactive" {
		t.Fatalf("expected inactive status, got %q", repo.business.Status)
	}

	business, err := svc.CreateBusiness(context.Background(), userID, service.CreateBusinessInput{
		BusinessName:    "Toko Baru",
		CategoryID:      1,
		Description:     "Deskripsi",
		BusinessType:    "running",
		LocationAddress: "Jakarta",
	})
	if err != nil {
		t.Fatalf("create replacement active business: %v", err)
	}
	if business.Status != "active" {
		t.Fatalf("expected replacement business to be active, got %q", business.Status)
	}
}

func TestCreateStarterBusinessInitialLimitAndRequirements(t *testing.T) {
	userID := uuid.New()
	repo := &fakeBusinessRepo{}
	svc := service.NewBusinessService(repo)

	// Missing commitment
	_, err := svc.CreateBusiness(context.Background(), userID, service.CreateBusinessInput{
		BusinessName:      "Rintisan Kopi",
		CategoryID:        1,
		Description:       "Kedai kopi rintisan",
		BusinessType:      "starter",
		LocationAddress:   "Bandung",
		TargetMarket:      "Mahasiswa",
		SupplierInfo:      "Petani Kopi",
		PricingEstimation: "15k - 25k",
		CommitmentChecked: false,
	})
	if !errors.Is(err, service.ErrStarterRequirementMissing) {
		t.Fatalf("expected ErrStarterRequirementMissing, got %v", err)
	}

	// Valid Starter Application
	b, err := svc.CreateBusiness(context.Background(), userID, service.CreateBusinessInput{
		BusinessName:      "Rintisan Kopi",
		CategoryID:        1,
		Description:       "Kedai kopi rintisan",
		BusinessType:      "starter",
		LocationAddress:   "Bandung",
		TargetMarket:      "Mahasiswa",
		SupplierInfo:      "Petani Kopi",
		PricingEstimation: "15k - 25k",
		CommitmentChecked: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if b.CurrentBorrowingLimit != 300000 {
		t.Fatalf("expected initial limit 300000 for starter business, got %d", b.CurrentBorrowingLimit)
	}
}

func TestCreateRunningBusinessInitialLimit(t *testing.T) {
	userID := uuid.New()
	repo := &fakeBusinessRepo{}
	svc := service.NewBusinessService(repo)

	b, err := svc.CreateBusiness(context.Background(), userID, service.CreateBusinessInput{
		BusinessName:    "Warung Berkah",
		CategoryID:      1,
		Description:     "Warung nasi berjalan 6 bulan",
		BusinessType:    "running",
		LocationAddress: "Surabaya",
	})
	if err != nil {
		t.Fatal(err)
	}
	if b.CurrentBorrowingLimit != 500000 {
		t.Fatalf("expected initial limit 500000 for running business, got %d", b.CurrentBorrowingLimit)
	}
}

func TestCalculateBorrowingLimitTiers(t *testing.T) {
	svc := service.NewBusinessService(nil)

	// Starter tiers
	if limit := svc.CalculateBorrowingLimit("starter", 0, 0, 0); limit != 300000 {
		t.Fatalf("expected 300000, got %d", limit)
	}
	if limit := svc.CalculateBorrowingLimit("starter", 0, 2, 0); limit != 1000000 {
		t.Fatalf("expected 1000000, got %d", limit)
	}
	if limit := svc.CalculateBorrowingLimit("starter", 0, 3, 0); limit != 1500000 {
		t.Fatalf("expected 1500000, got %d", limit)
	}

	// Running tiers
	if limit := svc.CalculateBorrowingLimit("running", 1, 1, 0); limit != 1000000 {
		t.Fatalf("expected 1000000 for Tier 1, got %d", limit)
	}
	if limit := svc.CalculateBorrowingLimit("running", 3, 2, 0); limit != 3000000 {
		t.Fatalf("expected 3000000 for Tier 2, got %d", limit)
	}
	if limit := svc.CalculateBorrowingLimit("running", 5, 5, 0); limit != 5000000 {
		t.Fatalf("expected 5000000 for Tier 3, got %d", limit)
	}
	if limit := svc.CalculateBorrowingLimit("running", 10, 10, 85); limit != 15000000 {
		t.Fatalf("expected 15000000 for Tier 4, got %d", limit)
	}
}

func TestCreateFinancialRecordAndSummary(t *testing.T) {
	userID := uuid.New()
	business := &model.Business{ID: uuid.New(), UserID: userID, BusinessName: "Toko Jaya"}
	repo := &fakeBusinessRepo{business: business}
	svc := service.NewBusinessService(repo)

	// Income 500k, Expense 150k
	rec1, err := svc.CreateFinancialRecord(context.Background(), userID, service.CreateFinancialRecordInput{
		IncomeAmount:  500000,
		ExpenseAmount: 150000,
		Note:          strPtr("Penjualan harian"),
		Proofs: []service.ProofInput{
			{FileURL: "https://storage.modalin.id/proof1.jpg", ProofType: "receipt"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec1.NetAmount != 350000 {
		t.Fatalf("expected NetAmount 350000, got %d", rec1.NetAmount)
	}

	summary, err := svc.GetFinancialSummary(context.Background(), userID, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalIncome != 500000 || summary.TotalExpense != 150000 || summary.NetCashflow != 350000 {
		t.Fatalf("unexpected summary: %#v", summary)
	}
}

func TestUpdateFinancialRecordRecalculatesNetAmount(t *testing.T) {
	userID := uuid.New()
	business := &model.Business{ID: uuid.New(), UserID: userID, BusinessName: "Toko Jaya", Status: "active"}
	repo := &fakeBusinessRepo{business: business}
	svc := service.NewBusinessService(repo)

	record, err := svc.CreateFinancialRecord(context.Background(), userID, service.CreateFinancialRecordInput{IncomeAmount: 100000})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := svc.UpdateFinancialRecord(context.Background(), userID, record.ID, service.UpdateFinancialRecordInput{IncomeAmount: 225000, ExpenseAmount: 75000})
	if err != nil {
		t.Fatal(err)
	}
	if updated.NetAmount != 150000 {
		t.Fatalf("expected updated net amount 150000, got %d", updated.NetAmount)
	}
}

func strPtr(s string) *string {
	return &s
}
