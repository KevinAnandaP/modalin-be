package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"modalin-be/internal/campaign/repository"
	"modalin-be/internal/campaign/service"
	"modalin-be/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type fakeCampaignRepo struct {
	business                     *model.Business
	campaigns                    map[uuid.UUID]*model.LoanCampaign
	budgets                      map[uuid.UUID][]model.CampaignBudgetItem
	milestones                   map[uuid.UUID][]model.CampaignMilestone
	disbursements                map[uuid.UUID][]model.Disbursement
	proofs                       map[uuid.UUID]*model.FundUsageProof
	revenueReports               map[uuid.UUID]*model.RevenueReport
	revenueProofs                map[uuid.UUID][]model.RevenueReportProof
	monthlyReports               map[uuid.UUID]*model.MonthlyProgressReport
	monthlyProofs                map[uuid.UUID][]model.MonthlyProgressReportProof
	schedules                    map[uuid.UUID][]model.RepaymentSchedule
	repayments                   map[uuid.UUID]*model.Repayment
	distributions                []model.LenderReturnDistribution
	fundings                     []model.Funding
	audits                       []model.AuditLog
	transactions                 int
	lockedReads                  int
	proofLocks                   int
	disbursementLocks            int
	hasFunding                   bool
	hasApprovedFieldVerification bool
}

func (r *fakeCampaignRepo) GetBusinessByUserID(_ context.Context, userID uuid.UUID) (*model.Business, error) {
	if r.business != nil && r.business.UserID == userID {
		return r.business, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) GetBusinessByID(_ context.Context, id uuid.UUID) (*model.Business, error) {
	if r.business != nil && r.business.ID == id {
		return r.business, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) CreateCampaign(_ context.Context, c *model.LoanCampaign) error {
	c.ID = uuid.New()
	r.campaigns[c.ID] = c
	return nil
}
func (r *fakeCampaignRepo) GetCampaignByID(_ context.Context, id uuid.UUID) (*model.LoanCampaign, error) {
	if c := r.campaigns[id]; c != nil {
		return c, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) GetCampaignByIDForUpdate(ctx context.Context, id uuid.UUID) (*model.LoanCampaign, error) {
	r.lockedReads++
	return r.GetCampaignByID(ctx, id)
}
func (r *fakeCampaignRepo) GetFundUsageProofForUpdate(ctx context.Context, id uuid.UUID) (*model.FundUsageProof, error) {
	r.proofLocks++
	return r.GetFundUsageProof(ctx, id)
}
func (r *fakeCampaignRepo) GetActiveFundUsageProof(_ context.Context, disbursementID uuid.UUID) (*model.FundUsageProof, error) {
	for _, proof := range r.proofs {
		if proof.DisbursementID == disbursementID {
			return proof, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) DeleteFundUsageProof(_ context.Context, id uuid.UUID) error {
	delete(r.proofs, id)
	return nil
}
func (r *fakeCampaignRepo) GetDisbursementForUpdate(ctx context.Context, id uuid.UUID) (*model.Disbursement, error) {
	r.disbursementLocks++
	return r.GetDisbursement(ctx, id)
}
func (r *fakeCampaignRepo) WithTransaction(_ context.Context, fn func(repository.Repository) error) error {
	r.transactions++
	return fn(r)
}
func (r *fakeCampaignRepo) HasApprovedFieldVerification(_ context.Context, _ uuid.UUID) (bool, error) {
	return r.hasApprovedFieldVerification, nil
}
func (r *fakeCampaignRepo) IsAssignedVerifier(_ context.Context, _ uuid.UUID, _ uuid.UUID) (bool, error) {
	return false, nil
}
func (r *fakeCampaignRepo) GetCampaignForBusiness(_ context.Context, id, businessID uuid.UUID) (*model.LoanCampaign, error) {
	c, err := r.GetCampaignByID(context.Background(), id)
	if err != nil || c.BusinessID != businessID {
		return nil, gorm.ErrRecordNotFound
	}
	return c, nil
}
func (r *fakeCampaignRepo) UpdateCampaign(_ context.Context, c *model.LoanCampaign) error {
	r.campaigns[c.ID] = c
	return nil
}
func (r *fakeCampaignRepo) DeleteCampaign(_ context.Context, id uuid.UUID) error {
	delete(r.campaigns, id)
	return nil
}
func (r *fakeCampaignRepo) ListCampaignsForBusiness(_ context.Context, businessID uuid.UUID) ([]model.LoanCampaign, error) {
	out := []model.LoanCampaign{}
	for _, c := range r.campaigns {
		if c.BusinessID == businessID {
			out = append(out, *c)
		}
	}
	return out, nil
}
func (r *fakeCampaignRepo) CreateBudgetItem(_ context.Context, b *model.CampaignBudgetItem) error {
	b.ID = uuid.New()
	r.budgets[b.CampaignID] = append(r.budgets[b.CampaignID], *b)
	return nil
}
func (r *fakeCampaignRepo) GetBudgetItem(_ context.Context, id, campaignID uuid.UUID) (*model.CampaignBudgetItem, error) {
	for i := range r.budgets[campaignID] {
		if r.budgets[campaignID][i].ID == id {
			return &r.budgets[campaignID][i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) UpdateBudgetItem(_ context.Context, b *model.CampaignBudgetItem) error {
	for i := range r.budgets[b.CampaignID] {
		if r.budgets[b.CampaignID][i].ID == b.ID {
			r.budgets[b.CampaignID][i] = *b
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) DeleteBudgetItem(_ context.Context, id, campaignID uuid.UUID) error {
	for i := range r.budgets[campaignID] {
		if r.budgets[campaignID][i].ID == id {
			r.budgets[campaignID] = append(r.budgets[campaignID][:i], r.budgets[campaignID][i+1:]...)
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) ListBudgetItems(_ context.Context, campaignID uuid.UUID) ([]model.CampaignBudgetItem, error) {
	return r.budgets[campaignID], nil
}
func (r *fakeCampaignRepo) CountBudgetItems(_ context.Context, campaignID uuid.UUID) (int, error) {
	return len(r.budgets[campaignID]), nil
}
func (r *fakeCampaignRepo) CreateMilestone(_ context.Context, m *model.CampaignMilestone) error {
	m.ID = uuid.New()
	r.milestones[m.CampaignID] = append(r.milestones[m.CampaignID], *m)
	return nil
}
func (r *fakeCampaignRepo) GetMilestone(_ context.Context, id, campaignID uuid.UUID) (*model.CampaignMilestone, error) {
	for i := range r.milestones[campaignID] {
		if r.milestones[campaignID][i].ID == id {
			return &r.milestones[campaignID][i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) UpdateMilestone(_ context.Context, m *model.CampaignMilestone) error {
	for i := range r.milestones[m.CampaignID] {
		if r.milestones[m.CampaignID][i].ID == m.ID {
			r.milestones[m.CampaignID][i] = *m
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) DeleteMilestone(_ context.Context, id, campaignID uuid.UUID) error {
	for i := range r.milestones[campaignID] {
		if r.milestones[campaignID][i].ID == id {
			r.milestones[campaignID] = append(r.milestones[campaignID][:i], r.milestones[campaignID][i+1:]...)
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) ListMilestones(_ context.Context, campaignID uuid.UUID) ([]model.CampaignMilestone, error) {
	return r.milestones[campaignID], nil
}
func (r *fakeCampaignRepo) ReplaceMilestones(_ context.Context, campaignID uuid.UUID, milestones []model.CampaignMilestone) error {
	for i := range milestones {
		milestones[i].ID = uuid.New()
	}
	r.milestones[campaignID] = milestones
	return nil
}
func (r *fakeCampaignRepo) CreateFunding(_ context.Context, funding *model.Funding) error {
	funding.ID = uuid.New()
	r.fundings = append(r.fundings, *funding)
	return nil
}
func (r *fakeCampaignRepo) CreateDisbursement(_ context.Context, d *model.Disbursement) error {
	d.ID = uuid.New()
	r.disbursements[d.CampaignID] = append(r.disbursements[d.CampaignID], *d)
	return nil
}
func (r *fakeCampaignRepo) GetDisbursement(_ context.Context, id uuid.UUID) (*model.Disbursement, error) {
	for campaignID := range r.disbursements {
		for i := range r.disbursements[campaignID] {
			if r.disbursements[campaignID][i].ID == id {
				return &r.disbursements[campaignID][i], nil
			}
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) UpdateDisbursement(_ context.Context, d *model.Disbursement) error {
	for campaignID := range r.disbursements {
		for i := range r.disbursements[campaignID] {
			if r.disbursements[campaignID][i].ID == d.ID {
				r.disbursements[campaignID][i] = *d
				return nil
			}
		}
	}
	return gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) CreateFundUsageProof(_ context.Context, proof *model.FundUsageProof) error {
	proof.ID = uuid.New()
	r.proofs[proof.ID] = proof
	return nil
}
func (r *fakeCampaignRepo) GetFundUsageProof(_ context.Context, id uuid.UUID) (*model.FundUsageProof, error) {
	if proof := r.proofs[id]; proof != nil {
		return proof, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) UpdateFundUsageProof(_ context.Context, proof *model.FundUsageProof) error {
	r.proofs[proof.ID] = proof
	return nil
}
func (r *fakeCampaignRepo) CreateAuditLog(_ context.Context, audit *model.AuditLog) error {
	r.audits = append(r.audits, *audit)
	return nil
}
func (r *fakeCampaignRepo) HasPaidFunding(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return r.hasFunding, nil
}
func (r *fakeCampaignRepo) ListFundingsForLender(_ context.Context, _ uuid.UUID) ([]model.Funding, error) {
	return nil, nil
}
func (r *fakeCampaignRepo) ListDisbursements(_ context.Context, campaignID uuid.UUID) ([]model.Disbursement, error) {
	return r.disbursements[campaignID], nil
}
func (r *fakeCampaignRepo) ListFundUsageProofs(_ context.Context, _ string, _, _ int) ([]model.FundUsageProof, error) {
	return nil, nil
}
func (r *fakeCampaignRepo) CreateRevenueReport(_ context.Context, report *model.RevenueReport) error {
	report.ID = uuid.New()
	r.revenueReports[report.ID] = report
	return nil
}
func (r *fakeCampaignRepo) CreateMonthlyProgressReport(_ context.Context, report *model.MonthlyProgressReport) error {
	report.ID = uuid.New()
	r.monthlyReports[report.ID] = report
	return nil
}
func (r *fakeCampaignRepo) CreateMonthlyProgressReportProof(_ context.Context, proof *model.MonthlyProgressReportProof) error {
	proof.ID = uuid.New()
	r.monthlyProofs[proof.MonthlyProgressReportID] = append(r.monthlyProofs[proof.MonthlyProgressReportID], *proof)
	return nil
}
func (r *fakeCampaignRepo) ListMonthlyProgressReports(_ context.Context, campaignID uuid.UUID) ([]model.MonthlyProgressReport, error) {
	var reports []model.MonthlyProgressReport
	for _, report := range r.monthlyReports {
		if report.CampaignID == campaignID {
			reports = append(reports, *report)
		}
	}
	return reports, nil
}
func (r *fakeCampaignRepo) GetMonthlyProgressReportForUpdate(_ context.Context, id uuid.UUID) (*model.MonthlyProgressReport, error) {
	if report := r.monthlyReports[id]; report != nil {
		return report, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) UpdateMonthlyProgressReport(_ context.Context, report *model.MonthlyProgressReport) error {
	r.monthlyReports[report.ID] = report
	return nil
}
func (r *fakeCampaignRepo) GetRevenueReport(_ context.Context, id uuid.UUID) (*model.RevenueReport, error) {
	if report := r.revenueReports[id]; report != nil {
		return report, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) GetRevenueReportForUpdate(ctx context.Context, id uuid.UUID) (*model.RevenueReport, error) {
	return r.GetRevenueReport(ctx, id)
}
func (r *fakeCampaignRepo) ListRevenueReports(_ context.Context, campaignID uuid.UUID) ([]model.RevenueReport, error) {
	var reports []model.RevenueReport
	for _, report := range r.revenueReports {
		if report.CampaignID == campaignID {
			reports = append(reports, *report)
		}
	}
	return reports, nil
}
func (r *fakeCampaignRepo) UpdateRevenueReport(_ context.Context, report *model.RevenueReport) error {
	r.revenueReports[report.ID] = report
	return nil
}
func (r *fakeCampaignRepo) CreateRevenueReportProof(_ context.Context, proof *model.RevenueReportProof) error {
	proof.ID = uuid.New()
	r.revenueProofs[proof.RevenueReportID] = append(r.revenueProofs[proof.RevenueReportID], *proof)
	return nil
}
func (r *fakeCampaignRepo) ListRevenueReportProofs(_ context.Context, reportID uuid.UUID) ([]model.RevenueReportProof, error) {
	return r.revenueProofs[reportID], nil
}
func (r *fakeCampaignRepo) CreateRepaymentSchedules(_ context.Context, schedules []model.RepaymentSchedule) error {
	if len(schedules) == 0 {
		return nil
	}
	for i := range schedules {
		schedules[i].ID = uuid.New()
	}
	r.schedules[schedules[0].CampaignID] = schedules
	return nil
}
func (r *fakeCampaignRepo) ListRepaymentSchedules(_ context.Context, campaignID uuid.UUID) ([]model.RepaymentSchedule, error) {
	return r.schedules[campaignID], nil
}
func (r *fakeCampaignRepo) GetRepaymentScheduleForUpdate(_ context.Context, id uuid.UUID) (*model.RepaymentSchedule, error) {
	for _, schedules := range r.schedules {
		for i := range schedules {
			if schedules[i].ID == id {
				return &schedules[i], nil
			}
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) UpdateRepaymentSchedule(_ context.Context, schedule *model.RepaymentSchedule) error {
	for campaignID, schedules := range r.schedules {
		for i := range schedules {
			if schedules[i].ID == schedule.ID {
				r.schedules[campaignID][i] = *schedule
				return nil
			}
		}
	}
	return gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) CreateRepayment(_ context.Context, repayment *model.Repayment) error {
	repayment.ID = uuid.New()
	r.repayments[repayment.ID] = repayment
	return nil
}
func (r *fakeCampaignRepo) HasActiveRepayment(_ context.Context, scheduleID uuid.UUID) (bool, error) {
	for _, repayment := range r.repayments {
		if repayment.ScheduleID == scheduleID && repayment.Status != "rejected" {
			return true, nil
		}
	}
	return false, nil
}
func (r *fakeCampaignRepo) GetRepaymentForUpdate(_ context.Context, id uuid.UUID) (*model.Repayment, error) {
	if repayment := r.repayments[id]; repayment != nil {
		return repayment, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) UpdateRepayment(_ context.Context, repayment *model.Repayment) error {
	r.repayments[repayment.ID] = repayment
	return nil
}
func (r *fakeCampaignRepo) ListPaidFundings(_ context.Context, campaignID uuid.UUID) ([]model.Funding, error) {
	var fundings []model.Funding
	for _, funding := range r.fundings {
		if funding.CampaignID == campaignID && funding.Status == "paid" {
			fundings = append(fundings, funding)
		}
	}
	return fundings, nil
}
func (r *fakeCampaignRepo) CreateLenderReturnDistributions(_ context.Context, distributions []model.LenderReturnDistribution) error {
	for i := range distributions {
		distributions[i].ID = uuid.New()
	}
	r.distributions = append(r.distributions, distributions...)
	return nil
}
func (r *fakeCampaignRepo) GetLenderReturnDistributionForUpdate(_ context.Context, id uuid.UUID) (*model.LenderReturnDistribution, error) {
	for i := range r.distributions {
		if r.distributions[i].ID == id {
			return &r.distributions[i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) UpdateLenderReturnDistribution(_ context.Context, distribution *model.LenderReturnDistribution) error {
	for i := range r.distributions {
		if r.distributions[i].ID == distribution.ID {
			r.distributions[i] = *distribution
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) ListLenderReturnDistributions(_ context.Context, lenderID uuid.UUID) ([]model.LenderReturnDistribution, error) {
	var rows []model.LenderReturnDistribution
	for _, row := range r.distributions {
		if row.LenderUserID == lenderID {
			rows = append(rows, row)
		}
	}
	return rows, nil
}
func (r *fakeCampaignRepo) ListCatalog(_ context.Context, _ repository.CatalogFilter) ([]model.LoanCampaign, error) {
	return nil, nil
}

func newRepo(userID uuid.UUID, limit int64) *fakeCampaignRepo {
	return &fakeCampaignRepo{business: &model.Business{ID: uuid.New(), UserID: userID, CurrentBorrowingLimit: limit, Status: "active"}, campaigns: map[uuid.UUID]*model.LoanCampaign{}, budgets: map[uuid.UUID][]model.CampaignBudgetItem{}, milestones: map[uuid.UUID][]model.CampaignMilestone{}, disbursements: map[uuid.UUID][]model.Disbursement{}, proofs: map[uuid.UUID]*model.FundUsageProof{}, revenueReports: map[uuid.UUID]*model.RevenueReport{}, revenueProofs: map[uuid.UUID][]model.RevenueReportProof{}, monthlyReports: map[uuid.UUID]*model.MonthlyProgressReport{}, monthlyProofs: map[uuid.UUID][]model.MonthlyProgressReportProof{}, schedules: map[uuid.UUID][]model.RepaymentSchedule{}, repayments: map[uuid.UUID]*model.Repayment{}}
}

func TestCampaignSubmissionRequiresLimitAndCompletePlan(t *testing.T) {
	userID := uuid.New()
	repo := newRepo(userID, 1_000_000)
	svc := service.NewCampaignService(repo)
	_, err := svc.CreateCampaign(context.Background(), userID, service.CampaignInput{Title: "Modal kios", Description: "Menambah stok", RequestedAmount: 1_500_000, LoanTenorMonths: 6, BenefitType: "principal_only"})
	if !errors.Is(err, service.ErrAmountExceedsLimit) {
		t.Fatalf("expected tier limit rejection, got %v", err)
	}
	campaign, err := svc.CreateCampaign(context.Background(), userID, service.CampaignInput{Title: "Modal kios", Description: "Menambah stok", RequestedAmount: 1_000_000, LoanTenorMonths: 6, BenefitType: "principal_only"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SubmitCampaign(context.Background(), userID, campaign.ID); !errors.Is(err, service.ErrIncompleteCampaignPlan) {
		t.Fatalf("expected incomplete plan error, got %v", err)
	}
	_, _ = svc.CreateBudgetItem(context.Background(), userID, campaign.ID, service.BudgetItemInput{ItemName: "Stok", Category: "stock", Amount: 1_000_000, PurchaseMethod: "direct_purchase", PriorityLevel: "high"})
	if err := svc.SubmitCampaign(context.Background(), userID, campaign.ID); err != nil {
		t.Fatal(err)
	}
	if campaign.Status != "admin_review" {
		t.Fatalf("expected admin_review, got %s", campaign.Status)
	}
}

func TestCampaignReviewPublishesOnlyCompleteReview(t *testing.T) {
	userID := uuid.New()
	repo := newRepo(userID, 1_000_000)
	svc := service.NewCampaignService(repo)
	campaign, _ := svc.CreateCampaign(context.Background(), userID, service.CampaignInput{Title: "Modal kios", Description: "Menambah stok", RequestedAmount: 1_000_000, LoanTenorMonths: 6, BenefitType: "principal_only"})
	campaign.Status = "admin_review"
	adminID := uuid.New()
	if err := svc.ReviewCampaign(context.Background(), adminID, campaign.ID, "publish"); !errors.Is(err, service.ErrIncompleteCampaignPlan) {
		t.Fatalf("expected complete-plan review rejection, got %v", err)
	}
	repo.budgets[campaign.ID] = []model.CampaignBudgetItem{{CampaignID: campaign.ID, Amount: 1_000_000, PriorityLevel: "high"}}
	repo.milestones[campaign.ID] = []model.CampaignMilestone{{CampaignID: campaign.ID, Amount: 1_000_000, SequenceNo: 1}}
	if err := svc.ReviewCampaign(context.Background(), adminID, campaign.ID, "publish"); err != nil {
		t.Fatal(err)
	}
	if campaign.Status != "published" || campaign.ApprovedBy == nil || *campaign.ApprovedBy != adminID {
		t.Fatalf("campaign was not published by admin: %#v", campaign)
	}
}

func TestSubmissionBuildsMilestonesFromBudgetPriorityLevel(t *testing.T) {
	userID := uuid.New()
	repo := newRepo(userID, 10_000_000)
	svc := service.NewCampaignService(repo)
	campaign, err := svc.CreateCampaign(context.Background(), userID, service.CampaignInput{Title: "Peralatan usaha", Description: "Tiga barang", RequestedAmount: 10_000_000, LoanTenorMonths: 6, BenefitType: "principal_only"})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []service.BudgetItemInput{
		{ItemName: "Mesin kopi", Category: "equipment", Amount: 7_000_000, PurchaseMethod: "direct_purchase", PriorityLevel: "high"},
		{ItemName: "Bahan baku", Category: "stock", Amount: 2_000_000, PurchaseMethod: "direct_purchase", PriorityLevel: "medium"},
		{ItemName: "Kemasan", Category: "packaging", Amount: 1_000_000, PurchaseMethod: "direct_purchase", PriorityLevel: "low"},
	} {
		if _, err := svc.CreateBudgetItem(context.Background(), userID, campaign.ID, item); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.SubmitCampaign(context.Background(), userID, campaign.ID); err != nil {
		t.Fatal(err)
	}
	milestones := repo.milestones[campaign.ID]
	if len(milestones) != 3 {
		t.Fatalf("expected one milestone per budget item, got %d", len(milestones))
	}
	if milestones[0].Title != "Mesin kopi" || milestones[0].SequenceNo != 1 || milestones[0].Status != "available" {
		t.Fatalf("highest-priority item should be the first available milestone: %#v", milestones[0])
	}
	if milestones[1].Title != "Bahan baku" || milestones[1].SequenceNo != 2 || milestones[1].Status != "locked" {
		t.Fatalf("second-priority item should remain locked: %#v", milestones[1])
	}
	if milestones[2].Title != "Kemasan" || milestones[2].SequenceNo != 3 || milestones[2].Status != "locked" {
		t.Fatalf("third-priority item should remain locked: %#v", milestones[2])
	}
}

func TestSamePriorityUsesBorrowerInputOrder(t *testing.T) {
	userID := uuid.New()
	repo := newRepo(userID, 1_000_000)
	svc := service.NewCampaignService(repo)
	campaign, err := svc.CreateCampaign(context.Background(), userID, service.CampaignInput{Title: "Peralatan", Description: "Urutan sama", RequestedAmount: 300_000, LoanTenorMonths: 3, BenefitType: "principal_only"})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []service.BudgetItemInput{
		{ItemName: "Barang pertama", Category: "equipment", Amount: 100_000, PurchaseMethod: "direct_purchase", PriorityLevel: "high"},
		{ItemName: "Barang kedua", Category: "equipment", Amount: 100_000, PurchaseMethod: "direct_purchase", PriorityLevel: "high"},
		{ItemName: "Barang ketiga", Category: "equipment", Amount: 100_000, PurchaseMethod: "direct_purchase", PriorityLevel: "low"},
	} {
		if _, err := svc.CreateBudgetItem(context.Background(), userID, campaign.ID, item); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.SubmitCampaign(context.Background(), userID, campaign.ID); err != nil {
		t.Fatal(err)
	}
	if got := repo.milestones[campaign.ID]; got[0].Title != "Barang pertama" || got[1].Title != "Barang kedua" {
		t.Fatalf("same-priority milestones must preserve input order: %#v", got)
	}
}

func TestCreateBudgetItemRejectsInvalidPriorityLevel(t *testing.T) {
	userID := uuid.New()
	repo := newRepo(userID, 1_000_000)
	svc := service.NewCampaignService(repo)
	campaign, _ := svc.CreateCampaign(context.Background(), userID, service.CampaignInput{Title: "Stok", Description: "Stok usaha", RequestedAmount: 1_000_000, LoanTenorMonths: 6, BenefitType: "principal_only"})
	_, err := svc.CreateBudgetItem(context.Background(), userID, campaign.ID, service.BudgetItemInput{ItemName: "Stok", Category: "stock", Amount: 1_000_000, PurchaseMethod: "direct_purchase", PriorityLevel: "urgent"})
	if !errors.Is(err, service.ErrInvalidBudgetItem) {
		t.Fatalf("expected invalid priority level error, got %v", err)
	}
}

func TestPledgeRejectsSelfFundingAndOverfunding(t *testing.T) {
	ownerID := uuid.New()
	repo := newRepo(ownerID, 1_000_000)
	campaign := &model.LoanCampaign{ID: uuid.New(), BusinessID: repo.business.ID, RequestedAmount: 100_000, LoanTenorMonths: 6, Status: "published"}
	repo.campaigns[campaign.ID] = campaign
	svc := service.NewCampaignService(repo)

	if _, err := svc.Pledge(context.Background(), ownerID, campaign.ID, 50_000); !errors.Is(err, service.ErrSelfFunding) {
		t.Fatalf("expected self-funding rejection, got %v", err)
	}
	if _, err := svc.Pledge(context.Background(), uuid.New(), campaign.ID, 100_001); !errors.Is(err, service.ErrFundingExceedsTarget) {
		t.Fatalf("expected overfunding rejection, got %v", err)
	}
}

func TestFullPledgeWaitsForConfirmedDisbursementBeforeCreatingSchedule(t *testing.T) {
	ownerID := uuid.New()
	repo := newRepo(ownerID, 1_000_000)
	campaign := &model.LoanCampaign{ID: uuid.New(), BusinessID: repo.business.ID, RequestedAmount: 100_000, LoanTenorMonths: 6, Status: "published"}
	repo.campaigns[campaign.ID] = campaign
	repo.milestones[campaign.ID] = []model.CampaignMilestone{{ID: uuid.New(), CampaignID: campaign.ID, Amount: 100_000, SequenceNo: 1, Status: "available"}}
	svc := service.NewCampaignService(repo)

	if _, err := svc.Pledge(context.Background(), uuid.New(), campaign.ID, 100_000); err != nil {
		t.Fatal(err)
	}
	if campaign.Status != "active" || campaign.FundedAmount != 100_000 {
		t.Fatalf("expected active fully funded campaign, got %#v", campaign)
	}
	if len(repo.disbursements[campaign.ID]) != 1 || repo.disbursements[campaign.ID][0].Status != "pending_transfer" {
		t.Fatalf("expected pending-transfer disbursement, got %#v", repo.disbursements[campaign.ID])
	}
	if schedules := repo.schedules[campaign.ID]; len(schedules) != 0 {
		t.Fatalf("schedule must not exist before transfer confirmation, got %#v", schedules)
	}
	disbursementID := repo.disbursements[campaign.ID][0].ID
	if _, err := svc.ConfirmDisbursement(context.Background(), uuid.New(), disbursementID, service.DisbursementConfirmationInput{TransferReference: "TRF-BORROWER-001", TransferProofURL: "/uploads/disbursement-proof.png"}); err != nil {
		t.Fatal(err)
	}
	if schedules := repo.schedules[campaign.ID]; len(schedules) != 6 || schedules[0].PrincipalDue != 16_667 || schedules[5].PrincipalDue != 16_666 {
		t.Fatalf("expected deterministic six-month repayment schedule, got %#v", schedules)
	}
	if repo.disbursements[campaign.ID][0].Status != "proof_required" {
		t.Fatalf("expected confirmed disbursement to require usage proof, got %#v", repo.disbursements[campaign.ID][0])
	}
	if _, err := svc.ConfirmDisbursement(context.Background(), uuid.New(), disbursementID, service.DisbursementConfirmationInput{TransferReference: "TRF-BORROWER-001", TransferProofURL: "/uploads/retried-proof.png"}); err != nil {
		t.Fatal(err)
	}
	if schedules := repo.schedules[campaign.ID]; len(schedules) != 6 {
		t.Fatalf("retry must not create a second schedule, got %#v", schedules)
	}
}

func TestRevenueVerificationSetsRevenueShareAndVerifiedRepaymentDistributesProRata(t *testing.T) {
	ownerID := uuid.New()
	repo := newRepo(ownerID, 1_000_000)
	campaign := &model.LoanCampaign{ID: uuid.New(), BusinessID: repo.business.ID, RequestedAmount: 100_000, FundedAmount: 100_000, LoanTenorMonths: 1, BenefitType: "revenue_share", RevenueSharePercent: funcFloat(10), ReturnCapPercent: funcFloat(3), Status: "active"}
	repo.campaigns[campaign.ID] = campaign
	firstLender, secondLender := uuid.New(), uuid.New()
	repo.fundings = []model.Funding{{ID: uuid.New(), CampaignID: campaign.ID, LenderUserID: firstLender, Amount: 60_000, Status: "paid"}, {ID: uuid.New(), CampaignID: campaign.ID, LenderUserID: secondLender, Amount: 40_000, Status: "paid"}}
	repo.schedules[campaign.ID] = []model.RepaymentSchedule{{ID: uuid.New(), CampaignID: campaign.ID, DueDate: time.Now().UTC().AddDate(0, 1, 0), PrincipalDue: 100_000, TotalDue: 100_000, Status: "upcoming"}}
	svc := service.NewCampaignService(repo)
	report, err := svc.CreateRevenueReport(context.Background(), ownerID, campaign.ID, service.RevenueReportInput{PeriodMonth: int(time.Now().UTC().AddDate(0, 1, 0).Month()), PeriodYear: time.Now().UTC().AddDate(0, 1, 0).Year(), GrossRevenue: 50_000, TransactionCount: 8, BusinessStatus: "running", Proofs: []service.RevenueProofInput{{FileURL: "/uploads/revenue-proof.png", ProofType: "sales_recap"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.VerifyRevenueReport(context.Background(), uuid.New(), report.ID, "approve"); err != nil {
		t.Fatal(err)
	}
	if err := svc.ReviewRevenueReport(context.Background(), uuid.New(), report.ID, "approve", 50_000); err != nil {
		t.Fatal(err)
	}
	if schedule := repo.schedules[campaign.ID][0]; schedule.RevenueShareDue == nil || *schedule.RevenueShareDue != 3_000 || schedule.TotalDue != 103_000 {
		t.Fatalf("expected revenue-share schedule update, got %#v", schedule)
	}
	repayment, err := svc.CreateRepayment(context.Background(), ownerID, campaign.ID, service.RepaymentInput{ScheduleID: repo.schedules[campaign.ID][0].ID, PaidAmount: 103_000})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.VerifyRepayment(context.Background(), uuid.New(), repayment.ID, "approve"); err != nil {
		t.Fatal(err)
	}
	if err := svc.ReviewRepayment(context.Background(), uuid.New(), repayment.ID, "approve"); err != nil {
		t.Fatal(err)
	}
	if len(repo.distributions) != 2 || repo.distributions[0].PrincipalAmount+repo.distributions[1].PrincipalAmount != 100_000 || repo.distributions[0].BenefitAmount+repo.distributions[1].BenefitAmount != 3_000 {
		t.Fatalf("distribution must exactly allocate repayment: %#v", repo.distributions)
	}
	if err := svc.MarkLenderReturnDistributed(context.Background(), uuid.New(), repo.distributions[0].ID, "TRF-001"); err != nil {
		t.Fatal(err)
	}
	if repo.distributions[0].Status != "distributed" || repo.distributions[0].TransferReference == nil {
		t.Fatalf("expected auditable completed distribution, got %#v", repo.distributions[0])
	}
}

func TestBorrowerCreatesMonthlyProgressReportWithProof(t *testing.T) {
	ownerID := uuid.New()
	repo := newRepo(ownerID, 1_000_000)
	campaign := &model.LoanCampaign{ID: uuid.New(), BusinessID: repo.business.ID, Status: "active"}
	repo.campaigns[campaign.ID] = campaign
	report, err := service.NewCampaignService(repo).CreateMonthlyProgressReport(context.Background(), ownerID, campaign.ID, service.MonthlyProgressReportInput{PeriodMonth: 7, PeriodYear: 2026, FundUsageSummary: "Pembelian bahan baku", BusinessProgress: "Penjualan meningkat", RepaymentStatus: "current", Proofs: []service.MonthlyProgressProofInput{{FileURL: "/uploads/progress-proof.png", ProofType: "receipt"}}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "submitted" || len(repo.monthlyProofs[report.ID]) != 1 {
		t.Fatalf("expected submitted report with one proof, got %#v", report)
	}
	if err := service.NewCampaignService(repo).VerifyMonthlyProgressReport(context.Background(), uuid.New(), report.ID, "approve"); err != nil {
		t.Fatal(err)
	}
	if report.Status != "verifier_reviewed" {
		t.Fatalf("expected verifier-reviewed progress report, got %#v", report)
	}
}

func TestBorrowerCanResubmitRejectedReportsWithoutCreatingAnotherPeriod(t *testing.T) {
	ownerID := uuid.New()
	repo := newRepo(ownerID, 1_000_000)
	campaign := &model.LoanCampaign{ID: uuid.New(), BusinessID: repo.business.ID, BenefitType: "revenue_share", Status: "active"}
	repo.campaigns[campaign.ID] = campaign
	svc := service.NewCampaignService(repo)

	revenue, err := svc.CreateRevenueReport(context.Background(), ownerID, campaign.ID, service.RevenueReportInput{PeriodMonth: 7, PeriodYear: 2026, GrossRevenue: 100_000, TransactionCount: 2, BusinessStatus: "running", Proofs: []service.RevenueProofInput{{FileURL: "/uploads/first.png", ProofType: "receipt"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.VerifyRevenueReport(context.Background(), uuid.New(), revenue.ID, "reject"); err != nil {
		t.Fatal(err)
	}
	updatedRevenue, err := svc.ResubmitRevenueReport(context.Background(), ownerID, campaign.ID, revenue.ID, service.RevenueReportInput{PeriodMonth: 7, PeriodYear: 2026, GrossRevenue: 125_000, TransactionCount: 3, BusinessStatus: "running", Proofs: []service.RevenueProofInput{{FileURL: "/uploads/revised.png", ProofType: "sales_recap"}}})
	if err != nil {
		t.Fatal(err)
	}
	if updatedRevenue.ID != revenue.ID || updatedRevenue.Status != "submitted" || updatedRevenue.GrossRevenue != 125_000 || len(repo.revenueReports) != 1 || len(repo.revenueProofs[revenue.ID]) != 2 {
		t.Fatalf("expected one resubmitted revenue report with audit proofs, got %#v", updatedRevenue)
	}

	progress, err := svc.CreateMonthlyProgressReport(context.Background(), ownerID, campaign.ID, service.MonthlyProgressReportInput{PeriodMonth: 7, PeriodYear: 2026, FundUsageSummary: "Bahan baku", BusinessProgress: "Berjalan", RepaymentStatus: "current", Proofs: []service.MonthlyProgressProofInput{{FileURL: "/uploads/progress-first.png", ProofType: "receipt"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.VerifyMonthlyProgressReport(context.Background(), uuid.New(), progress.ID, "reject"); err != nil {
		t.Fatal(err)
	}
	updatedProgress, err := svc.ResubmitMonthlyProgressReport(context.Background(), ownerID, campaign.ID, progress.ID, service.MonthlyProgressReportInput{PeriodMonth: 7, PeriodYear: 2026, FundUsageSummary: "Bahan baku tambahan", BusinessProgress: "Meningkat", RepaymentStatus: "current", Proofs: []service.MonthlyProgressProofInput{{FileURL: "/uploads/progress-revised.png", ProofType: "receipt"}}})
	if err != nil {
		t.Fatal(err)
	}
	if updatedProgress.ID != progress.ID || updatedProgress.Status != "submitted" || len(repo.monthlyReports) != 1 || len(repo.monthlyProofs[progress.ID]) != 2 {
		t.Fatalf("expected one resubmitted progress report with audit proofs, got %#v", updatedProgress)
	}
}

func TestCreateRepaymentRejectsSecondActivePaymentForSameSchedule(t *testing.T) {
	ownerID := uuid.New()
	repo := newRepo(ownerID, 1_000_000)
	campaign := &model.LoanCampaign{ID: uuid.New(), BusinessID: repo.business.ID, Status: "active"}
	repo.campaigns[campaign.ID] = campaign
	schedule := model.RepaymentSchedule{ID: uuid.New(), CampaignID: campaign.ID, PrincipalDue: 100_000, TotalDue: 100_000, Status: "upcoming"}
	repo.schedules[campaign.ID] = []model.RepaymentSchedule{schedule}
	svc := service.NewCampaignService(repo)

	if _, err := svc.CreateRepayment(context.Background(), ownerID, campaign.ID, service.RepaymentInput{ScheduleID: schedule.ID, PaidAmount: schedule.TotalDue}); err != nil {
		t.Fatalf("first repayment failed: %v", err)
	}
	_, err := svc.CreateRepayment(context.Background(), ownerID, campaign.ID, service.RepaymentInput{ScheduleID: schedule.ID, PaidAmount: schedule.TotalDue})
	if !errors.Is(err, service.ErrRepaymentUnavailable) {
		t.Fatalf("expected duplicate payment to be unavailable, got %v", err)
	}
}

func funcFloat(v float64) *float64 { return &v }

func TestPledgeUsesTransactionAndLocksCampaignBeforeFunding(t *testing.T) {
	ownerID := uuid.New()
	repo := newRepo(ownerID, 1_000_000)
	campaign := &model.LoanCampaign{ID: uuid.New(), BusinessID: repo.business.ID, RequestedAmount: 100_000, LoanTenorMonths: 6, Status: "published"}
	repo.campaigns[campaign.ID] = campaign
	repo.milestones[campaign.ID] = []model.CampaignMilestone{{ID: uuid.New(), CampaignID: campaign.ID, Amount: 100_000, SequenceNo: 1, Status: "available"}}
	svc := service.NewCampaignService(repo)

	if _, err := svc.Pledge(context.Background(), uuid.New(), campaign.ID, 100_000); err != nil {
		t.Fatal(err)
	}
	if repo.transactions != 1 {
		t.Fatalf("expected pledge to run in one transaction, got %d", repo.transactions)
	}
	if repo.lockedReads != 1 {
		t.Fatalf("expected pledge to lock the campaign before funding, got %d locked reads", repo.lockedReads)
	}
}

func TestApprovedProofVerifiesMilestoneAndQueuesNextDisbursementTransfer(t *testing.T) {
	ownerID := uuid.New()
	repo := newRepo(ownerID, 1_000_000)
	campaign := &model.LoanCampaign{ID: uuid.New(), BusinessID: repo.business.ID, RequestedAmount: 200_000, FundedAmount: 200_000, Status: "active"}
	repo.campaigns[campaign.ID] = campaign
	first := model.CampaignMilestone{ID: uuid.New(), CampaignID: campaign.ID, Amount: 100_000, SequenceNo: 1, Status: "disbursed"}
	second := model.CampaignMilestone{ID: uuid.New(), CampaignID: campaign.ID, Amount: 100_000, SequenceNo: 2, Status: "locked"}
	repo.milestones[campaign.ID] = []model.CampaignMilestone{first, second}
	disbursement := model.Disbursement{ID: uuid.New(), CampaignID: campaign.ID, MilestoneID: first.ID, Amount: 100_000, Status: "proof_required"}
	repo.disbursements[campaign.ID] = []model.Disbursement{disbursement}
	proof := model.FundUsageProof{ID: uuid.New(), CampaignID: campaign.ID, DisbursementID: disbursement.ID, Amount: 100_000, FileURL: "/uploads/proof.png", ProofType: "receipt", Status: "pending"}
	repo.proofs[proof.ID] = &proof
	svc := service.NewCampaignService(repo)

	if err := svc.ReviewFundUsageProof(context.Background(), uuid.New(), proof.ID, "approve"); err != nil {
		t.Fatal(err)
	}
	if repo.milestones[campaign.ID][0].Status != "verified" || repo.milestones[campaign.ID][1].Status != "available" {
		t.Fatalf("expected verified first and available second milestone, got %#v", repo.milestones[campaign.ID])
	}
	if len(repo.disbursements[campaign.ID]) != 2 || repo.disbursements[campaign.ID][1].Status != "pending_transfer" {
		t.Fatalf("expected pending-transfer second disbursement, got %#v", repo.disbursements[campaign.ID])
	}
}

func TestFundUsageProofMustMatchDisbursementAndReviewUsesTransaction(t *testing.T) {
	ownerID := uuid.New()
	repo := newRepo(ownerID, 1_000_000)
	campaign := &model.LoanCampaign{ID: uuid.New(), BusinessID: repo.business.ID, Status: "active"}
	repo.campaigns[campaign.ID] = campaign
	milestone := model.CampaignMilestone{ID: uuid.New(), CampaignID: campaign.ID, Amount: 100_000, Status: "disbursed"}
	repo.milestones[campaign.ID] = []model.CampaignMilestone{milestone}
	disbursement := model.Disbursement{ID: uuid.New(), CampaignID: campaign.ID, MilestoneID: milestone.ID, Amount: 100_000, Status: "proof_required"}
	repo.disbursements[campaign.ID] = []model.Disbursement{disbursement}
	svc := service.NewCampaignService(repo)

	if _, err := svc.SubmitFundUsageProof(context.Background(), ownerID, campaign.ID, disbursement.ID, service.FundUsageProofInput{FileURL: "/uploads/proof.png", ProofType: "receipt", Amount: 99_999}); !errors.Is(err, service.ErrInvalidFundUsageProof) {
		t.Fatalf("expected mismatched proof amount rejection, got %v", err)
	}
	proof := &model.FundUsageProof{ID: uuid.New(), CampaignID: campaign.ID, DisbursementID: disbursement.ID, Amount: 100_000, FileURL: "/uploads/proof.png", ProofType: "receipt", Status: "pending"}
	repo.proofs[proof.ID] = proof
	if err := svc.ReviewFundUsageProof(context.Background(), uuid.New(), proof.ID, "approve"); err != nil {
		t.Fatal(err)
	}
	if repo.transactions != 1 || repo.proofLocks != 1 || repo.disbursementLocks != 1 {
		t.Fatalf("review must transactionally lock proof and disbursement: tx=%d proof=%d disbursement=%d", repo.transactions, repo.proofLocks, repo.disbursementLocks)
	}
}

func TestProofFileAccessAllowsOwnerAdminAndFundingLender(t *testing.T) {
	ownerID := uuid.New()
	repo := newRepo(ownerID, 1_000_000)
	campaign := &model.LoanCampaign{ID: uuid.New(), BusinessID: repo.business.ID}
	repo.campaigns[campaign.ID] = campaign
	proof := &model.FundUsageProof{ID: uuid.New(), CampaignID: campaign.ID, FileURL: "/uploads/fund-usage-proofs/proof.png"}
	repo.proofs[proof.ID] = proof
	svc := service.NewCampaignService(repo)
	if _, err := svc.ProofFileURL(context.Background(), ownerID, campaign.ID, proof.ID, []string{"borrower"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ProofFileURL(context.Background(), uuid.New(), campaign.ID, proof.ID, []string{"lender"}); !errors.Is(err, service.ErrProofAccessDenied) {
		t.Fatalf("expected lender denial, got %v", err)
	}
	repo.hasFunding = true
	if _, err := svc.ProofFileURL(context.Background(), uuid.New(), campaign.ID, proof.ID, []string{"lender"}); err != nil {
		t.Fatal(err)
	}
}
