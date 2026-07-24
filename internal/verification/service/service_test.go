package service

import (
	"context"
	"errors"
	"testing"

	"modalin-be/internal/model"
	"modalin-be/internal/verification/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestCalculateRiskUsesApprovedWeightsAndLevels(t *testing.T) {
	assessment := CalculateRisk(RiskInput{
		FinancialScore:    90,
		VerificationScore: 80,
		RepaymentScore:    70,
		CommunityScore:    60,
	})

	if assessment.FinalScore != 78 {
		t.Fatalf("expected weighted final score 78, got %d", assessment.FinalScore)
	}
	if assessment.RiskLevel != "medium" {
		t.Fatalf("expected medium risk, got %q", assessment.RiskLevel)
	}
}

func TestCalculateRiskTreatsMissingSignalsAsNeutral(t *testing.T) {
	assessment := CalculateRisk(RiskInput{})
	if assessment.FinalScore != 50 || assessment.RiskLevel != "high" {
		t.Fatalf("expected neutral score 50 and high risk, got %#v", assessment)
	}
}

func TestCalculateRiskIdentifiesEveryMissingComponent(t *testing.T) {
	assessment := CalculateRisk(RiskInput{FinancialScore: 80, CommunityScore: 60})
	if !assessment.DataLimited {
		t.Fatal("expected data_limited when verification and repayment are missing")
	}
	if len(assessment.MissingComponents) != 2 || assessment.MissingComponents[0] != "verification" || assessment.MissingComponents[1] != "repayment" {
		t.Fatalf("unexpected missing components: %#v", assessment.MissingComponents)
	}
}

func TestRiskFingerprintIsStableAndChangesWithSourceData(t *testing.T) {
	campaignID, businessID := uuid.New(), uuid.New()
	first := riskFingerprint(campaignID, businessID, 1, 2, 0, 2, 3, 1, nil)
	if first != riskFingerprint(campaignID, businessID, 1, 2, 0, 2, 3, 1, nil) {
		t.Fatal("fingerprint must be stable")
	}
	if first == riskFingerprint(campaignID, businessID, 2, 2, 0, 2, 3, 1, nil) {
		t.Fatal("fingerprint must change when source changes")
	}
}

type mockVerificationRepo struct {
	repository.Repository
	campaign      *model.LoanCampaign
	business      *model.Business
	request       *model.VerificationRequest
	report        *model.VerificationReport
	hasActiveRole bool
	hasFunding    bool
	isAssigned    bool
	voteUpserted  bool
	auditCreated  bool
	riskCreated   bool
	latestRisk    *model.RiskAssessment
}

func (m *mockVerificationRepo) HasActiveVerificationRequest(_ context.Context, campaignID uuid.UUID) (bool, error) {
	if m.request != nil && m.request.CampaignID != nil && *m.request.CampaignID == campaignID && (m.request.Status == "pending" || m.request.Status == "assigned" || m.request.Status == "reviewed") {
		return true, nil
	}
	return false, nil
}

func (m *mockVerificationRepo) WithTransaction(_ context.Context, fn func(repository.Repository) error) error {
	return fn(m)
}
func (m *mockVerificationRepo) GetCampaignForUpdate(_ context.Context, id uuid.UUID) (*model.LoanCampaign, error) {
	if m.campaign != nil && m.campaign.ID == id {
		return m.campaign, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockVerificationRepo) GetCampaign(_ context.Context, id uuid.UUID) (*model.LoanCampaign, error) {
	if m.campaign != nil && m.campaign.ID == id {
		return m.campaign, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockVerificationRepo) UpdateCampaign(_ context.Context, c *model.LoanCampaign) error {
	m.campaign = c
	return nil
}
func (m *mockVerificationRepo) GetBusiness(_ context.Context, id uuid.UUID) (*model.Business, error) {
	if m.business != nil && m.business.ID == id {
		return m.business, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockVerificationRepo) UpdateBusiness(_ context.Context, b *model.Business) error {
	m.business = b
	return nil
}
func (m *mockVerificationRepo) CreateRequest(_ context.Context, req *model.VerificationRequest) error {
	m.request = req
	return nil
}
func (m *mockVerificationRepo) GetRequestForUpdate(_ context.Context, id uuid.UUID) (*model.VerificationRequest, error) {
	if m.request != nil && m.request.ID == id {
		return m.request, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockVerificationRepo) GetRequest(_ context.Context, id uuid.UUID) (*model.VerificationRequest, error) {
	if m.request != nil && m.request.ID == id {
		return m.request, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockVerificationRepo) UpdateRequest(_ context.Context, req *model.VerificationRequest) error {
	m.request = req
	return nil
}
func (m *mockVerificationRepo) HasActiveRole(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return m.hasActiveRole, nil
}
func (m *mockVerificationRepo) HasAnyActiveRole(_ context.Context, _ uuid.UUID) (bool, error) {
	return m.hasActiveRole, nil
}
func (m *mockVerificationRepo) IsAssignedVerifier(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return m.isAssigned, nil
}
func (m *mockVerificationRepo) HasFunding(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return m.hasFunding, nil
}
func (m *mockVerificationRepo) CreateReport(_ context.Context, r *model.VerificationReport) error {
	m.report = r
	return nil
}
func (m *mockVerificationRepo) HasReport(_ context.Context, _ uuid.UUID) (bool, error) {
	return m.report != nil, nil
}
func (m *mockVerificationRepo) LatestReport(_ context.Context, _ uuid.UUID) (*model.VerificationReport, error) {
	if m.report != nil {
		return m.report, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockVerificationRepo) LatestReportForCampaign(_ context.Context, _ uuid.UUID) (*model.VerificationReport, error) {
	if m.report != nil {
		return m.report, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockVerificationRepo) UpsertVote(_ context.Context, _ *model.CommunityVote) error {
	m.voteUpserted = true
	return nil
}
func (m *mockVerificationRepo) CountFinancialRecords(_ context.Context, _ uuid.UUID) (int64, error) {
	return 5, nil
}
func (m *mockVerificationRepo) RepaymentStats(_ context.Context, _ uuid.UUID) (int64, int64, int64, error) {
	return 10, 0, 10, nil
}
func (m *mockVerificationRepo) VoteStats(_ context.Context, _ uuid.UUID) (int64, int64, error) {
	return 8, 2, nil
}
func (m *mockVerificationRepo) LatestCampaignID(_ context.Context, bID uuid.UUID) (uuid.UUID, error) {
	if m.campaign != nil && m.campaign.BusinessID == bID {
		return m.campaign.ID, nil
	}
	return uuid.Nil, gorm.ErrRecordNotFound
}
func (m *mockVerificationRepo) CreateRiskAssessment(_ context.Context, a *model.RiskAssessment) error {
	m.riskCreated = true
	m.latestRisk = a
	return nil
}
func (m *mockVerificationRepo) LatestRiskAssessment(_ context.Context, _ uuid.UUID) (*model.RiskAssessment, error) {
	if m.latestRisk != nil {
		return m.latestRisk, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockVerificationRepo) CreateAuditLog(_ context.Context, _ *model.AuditLog) error {
	m.auditCreated = true
	return nil
}

func TestCreateRequestTierEligibility(t *testing.T) {
	borrowerID := uuid.New()
	business := &model.Business{ID: uuid.New(), UserID: borrowerID, CurrentBorrowingLimit: 3000000} // Tier 2 (< 5M)
	campaign := &model.LoanCampaign{ID: uuid.New(), BusinessID: business.ID, Status: "admin_review"}
	repo := &mockVerificationRepo{business: business, campaign: campaign}

	svc := New(repo)
	_, err := svc.CreateRequest(context.Background(), borrowerID, campaign.ID)
	if !errors.Is(err, ErrIneligibleCampaign) {
		t.Fatalf("expected ErrIneligibleCampaign for borrowing limit < 5M, got %v", err)
	}

	business.CurrentBorrowingLimit = 5000000 // Tier 3
	req, err := svc.CreateRequest(context.Background(), borrowerID, campaign.ID)
	if err != nil {
		t.Fatalf("expected success for Tier 3 campaign verification request, got %v", err)
	}
	if req.Status != "pending" {
		t.Fatalf("expected request status pending, got %s", req.Status)
	}
}

func TestAntiSelfAndAntiLenderVerification(t *testing.T) {
	borrowerID := uuid.New()
	lenderID := uuid.New()
	verifierID := uuid.New()
	business := &model.Business{ID: uuid.New(), UserID: borrowerID, CurrentBorrowingLimit: 15000000}
	campaign := &model.LoanCampaign{ID: uuid.New(), BusinessID: business.ID, Status: "admin_review"}
	req := &model.VerificationRequest{ID: uuid.New(), CampaignID: &campaign.ID, BusinessID: business.ID, Status: "pending"}

	// Case 1: Verifier is the borrower (Anti-Self)
	repo := &mockVerificationRepo{business: business, campaign: campaign, request: req, hasActiveRole: true}
	svc := New(repo)
	err := svc.Assign(context.Background(), uuid.New(), req.ID, borrowerID)
	if !errors.Is(err, ErrConflictOfInterest) {
		t.Fatalf("expected ErrConflictOfInterest when verifier is borrower, got %v", err)
	}

	// Case 2: Verifier is a lender who funded the campaign (Anti-Lender)
	repo.hasFunding = true
	err = svc.Assign(context.Background(), uuid.New(), req.ID, lenderID)
	if !errors.Is(err, ErrConflictOfInterest) {
		t.Fatalf("expected ErrConflictOfInterest when verifier funded campaign, got %v", err)
	}

	// Case 3: Independent verifier
	repo.hasFunding = false
	err = svc.Assign(context.Background(), uuid.New(), req.ID, verifierID)
	if err != nil {
		t.Fatalf("expected assignment success for independent verifier, got %v", err)
	}
	if req.Status != "assigned" || *req.AssignedVerifierID != verifierID {
		t.Fatalf("unexpected request state after assignment: %#v", req)
	}
}

func TestSubmitReportValidationAndRiskAssessment(t *testing.T) {
	borrowerID := uuid.New()
	verifierID := uuid.New()
	business := &model.Business{ID: uuid.New(), UserID: borrowerID, CurrentBorrowingLimit: 5000000}
	campaign := &model.LoanCampaign{ID: uuid.New(), BusinessID: business.ID, Status: "admin_review"}
	req := &model.VerificationRequest{ID: uuid.New(), CampaignID: &campaign.ID, BusinessID: business.ID, AssignedVerifierID: &verifierID, Status: "assigned"}

	repo := &mockVerificationRepo{business: business, campaign: campaign, request: req, hasActiveRole: true}
	svc := New(repo)

	// Invalid report: empty photo URL
	_, err := svc.SubmitReport(context.Background(), verifierID, req.ID, ReportInput{
		IsBusinessExists: true, IsBusinessActive: true, LocationMatch: true, PhotoURL: "", Note: "Valid", Recommendation: "approve",
	})
	if !errors.Is(err, ErrInvalidReport) {
		t.Fatalf("expected ErrInvalidReport for missing photo, got %v", err)
	}

	// Valid report submission
	report, err := svc.SubmitReport(context.Background(), verifierID, req.ID, ReportInput{
		IsBusinessExists: true, IsBusinessActive: true, LocationMatch: true, PhotoURL: "/uploads/photo.jpg", Note: "Confirmed physical store", Recommendation: "approve",
	})
	if err != nil {
		t.Fatalf("expected successful report submission, got %v", err)
	}
	if report.Recommendation != "approve" || req.Status != "reviewed" {
		t.Fatalf("unexpected report state: %#v, req status: %s", report, req.Status)
	}

	// Decide (Admin Approval)
	adminID := uuid.New()
	err = svc.Decide(context.Background(), adminID, req.ID, "approve")
	if err != nil {
		t.Fatalf("expected decide approval success, got %v", err)
	}
	if req.Status != "approved" || business.VerificationStatus != "verified" {
		t.Fatalf("unexpected decision state req: %s, business: %s", req.Status, business.VerificationStatus)
	}
	if !repo.riskCreated || repo.latestRisk == nil {
		t.Fatal("expected risk assessment to be generated on approval")
	}
}

func TestCommunityVoteRules(t *testing.T) {
	borrowerID := uuid.New()
	voterID := uuid.New()
	business := &model.Business{ID: uuid.New(), UserID: borrowerID}
	campaign := &model.LoanCampaign{ID: uuid.New(), BusinessID: business.ID}
	repo := &mockVerificationRepo{business: business, campaign: campaign, hasActiveRole: true}
	svc := New(repo)

	// Case 1: Borrower voting on own business (ErrSelfVote)
	err := svc.Vote(context.Background(), borrowerID, business.ID, "up", nil)
	if !errors.Is(err, ErrSelfVote) {
		t.Fatalf("expected ErrSelfVote when borrower votes on own business, got %v", err)
	}

	// Case 2: Assigned verifier voting on business (ErrConflictOfInterest)
	repo.isAssigned = true
	err = svc.Vote(context.Background(), voterID, business.ID, "up", nil)
	if !errors.Is(err, ErrConflictOfInterest) {
		t.Fatalf("expected ErrConflictOfInterest when assigned verifier votes, got %v", err)
	}

	// Case 3: Valid community vote
	repo.isAssigned = false
	err = svc.Vote(context.Background(), voterID, business.ID, "up", nil)
	if err != nil {
		t.Fatalf("expected successful community vote, got %v", err)
	}
	if !repo.voteUpserted {
		t.Fatal("expected community vote to be upserted")
	}
}
