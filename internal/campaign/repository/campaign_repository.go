package repository

import (
	"context"
	"strings"
	"time"

	"modalin-be/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CatalogFilter struct {
	Query, Category, RiskLevel                       string
	MinAmount, MaxAmount, MinRiskScore, MaxRiskScore int64
}

type Repository interface {
	WithTransaction(context.Context, func(Repository) error) error
	GetBusinessByUserID(context.Context, uuid.UUID) (*model.Business, error)
	GetBusinessByID(context.Context, uuid.UUID) (*model.Business, error)
	CreateCampaign(context.Context, *model.LoanCampaign) error
	GetCampaignByID(context.Context, uuid.UUID) (*model.LoanCampaign, error)
	GetCampaignByIDForUpdate(context.Context, uuid.UUID) (*model.LoanCampaign, error)
	GetCampaignForBusiness(context.Context, uuid.UUID, uuid.UUID) (*model.LoanCampaign, error)
	UpdateCampaign(context.Context, *model.LoanCampaign) error
	DeleteCampaign(context.Context, uuid.UUID) error
	ListCampaignsForBusiness(context.Context, uuid.UUID) ([]model.LoanCampaign, error)
	CreateBudgetItem(context.Context, *model.CampaignBudgetItem) error
	GetBudgetItem(context.Context, uuid.UUID, uuid.UUID) (*model.CampaignBudgetItem, error)
	UpdateBudgetItem(context.Context, *model.CampaignBudgetItem) error
	DeleteBudgetItem(context.Context, uuid.UUID, uuid.UUID) error
	ListBudgetItems(context.Context, uuid.UUID) ([]model.CampaignBudgetItem, error)
	CountBudgetItems(context.Context, uuid.UUID) (int, error)
	CreateMilestone(context.Context, *model.CampaignMilestone) error
	GetMilestone(context.Context, uuid.UUID, uuid.UUID) (*model.CampaignMilestone, error)
	UpdateMilestone(context.Context, *model.CampaignMilestone) error
	DeleteMilestone(context.Context, uuid.UUID, uuid.UUID) error
	ListMilestones(context.Context, uuid.UUID) ([]model.CampaignMilestone, error)
	ReplaceMilestones(context.Context, uuid.UUID, []model.CampaignMilestone) error
	CreateFunding(context.Context, *model.Funding) error
	CreateDisbursement(context.Context, *model.Disbursement) error
	GetDisbursement(context.Context, uuid.UUID) (*model.Disbursement, error)
	GetDisbursementForUpdate(context.Context, uuid.UUID) (*model.Disbursement, error)
	UpdateDisbursement(context.Context, *model.Disbursement) error
	CreateFundUsageProof(context.Context, *model.FundUsageProof) error
	GetFundUsageProof(context.Context, uuid.UUID) (*model.FundUsageProof, error)
	GetFundUsageProofForUpdate(context.Context, uuid.UUID) (*model.FundUsageProof, error)
	GetActiveFundUsageProof(context.Context, uuid.UUID) (*model.FundUsageProof, error)
	DeleteFundUsageProof(context.Context, uuid.UUID) error
	UpdateFundUsageProof(context.Context, *model.FundUsageProof) error
	CreateAuditLog(context.Context, *model.AuditLog) error
	HasPaidFunding(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	ListFundingsForLender(context.Context, uuid.UUID) ([]model.Funding, error)
	ListDisbursements(context.Context, uuid.UUID) ([]model.Disbursement, error)
	ListFundUsageProofs(context.Context, string, int, int) ([]model.FundUsageProof, error)
	CreateMonthlyProgressReport(context.Context, *model.MonthlyProgressReport) error
	CreateMonthlyProgressReportProof(context.Context, *model.MonthlyProgressReportProof) error
	ListMonthlyProgressReports(context.Context, uuid.UUID) ([]model.MonthlyProgressReport, error)
	GetMonthlyProgressReportForUpdate(context.Context, uuid.UUID) (*model.MonthlyProgressReport, error)
	UpdateMonthlyProgressReport(context.Context, *model.MonthlyProgressReport) error
	CreateRevenueReport(context.Context, *model.RevenueReport) error
	GetRevenueReport(context.Context, uuid.UUID) (*model.RevenueReport, error)
	GetRevenueReportForUpdate(context.Context, uuid.UUID) (*model.RevenueReport, error)
	ListRevenueReports(context.Context, uuid.UUID) ([]model.RevenueReport, error)
	UpdateRevenueReport(context.Context, *model.RevenueReport) error
	CreateRevenueReportProof(context.Context, *model.RevenueReportProof) error
	ListRevenueReportProofs(context.Context, uuid.UUID) ([]model.RevenueReportProof, error)
	CreateRepaymentSchedules(context.Context, []model.RepaymentSchedule) error
	ListRepaymentSchedules(context.Context, uuid.UUID) ([]model.RepaymentSchedule, error)
	GetRepaymentScheduleForUpdate(context.Context, uuid.UUID) (*model.RepaymentSchedule, error)
	UpdateRepaymentSchedule(context.Context, *model.RepaymentSchedule) error
	CreateRepayment(context.Context, *model.Repayment) error
	HasActiveRepayment(context.Context, uuid.UUID) (bool, error)
	GetRepaymentForUpdate(context.Context, uuid.UUID) (*model.Repayment, error)
	UpdateRepayment(context.Context, *model.Repayment) error
	ListPaidFundings(context.Context, uuid.UUID) ([]model.Funding, error)
	CreateLenderReturnDistributions(context.Context, []model.LenderReturnDistribution) error
	GetLenderReturnDistributionForUpdate(context.Context, uuid.UUID) (*model.LenderReturnDistribution, error)
	UpdateLenderReturnDistribution(context.Context, *model.LenderReturnDistribution) error
	ListLenderReturnDistributions(context.Context, uuid.UUID) ([]model.LenderReturnDistribution, error)
	ListCatalog(context.Context, CatalogFilter) ([]model.LoanCampaign, error)
	HasApprovedFieldVerification(context.Context, uuid.UUID) (bool, error)
	IsAssignedVerifier(context.Context, uuid.UUID, uuid.UUID) (bool, error)
}

type CampaignRepository struct{ db *gorm.DB }

func NewCampaignRepository(db *gorm.DB) *CampaignRepository { return &CampaignRepository{db: db} }
func (r *CampaignRepository) WithTransaction(ctx context.Context, fn func(Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&CampaignRepository{db: tx})
	})
}
func (r *CampaignRepository) GetBusinessByUserID(ctx context.Context, userID uuid.UUID) (*model.Business, error) {
	var b model.Business
	err := r.db.WithContext(ctx).Where("user_id = ? AND status = ?", userID, "active").First(&b).Error
	return &b, err
}
func (r *CampaignRepository) GetBusinessByID(ctx context.Context, id uuid.UUID) (*model.Business, error) {
	var b model.Business
	err := r.db.WithContext(ctx).First(&b, "id = ?", id).Error
	return &b, err
}
func (r *CampaignRepository) CreateCampaign(ctx context.Context, c *model.LoanCampaign) error {
	return r.db.WithContext(ctx).Create(c).Error
}
func (r *CampaignRepository) GetCampaignByID(ctx context.Context, id uuid.UUID) (*model.LoanCampaign, error) {
	var c model.LoanCampaign
	err := r.db.WithContext(ctx).Preload("Business").Preload("Business.Category").Preload("BudgetItems").Preload("Milestones", func(db *gorm.DB) *gorm.DB { return db.Order("sequence_no ASC") }).First(&c, "id = ?", id).Error
	return &c, err
}
func (r *CampaignRepository) GetCampaignByIDForUpdate(ctx context.Context, id uuid.UUID) (*model.LoanCampaign, error) {
	var c model.LoanCampaign
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&c, "id = ?", id).Error
	return &c, err
}
func (r *CampaignRepository) GetCampaignForBusiness(ctx context.Context, id, businessID uuid.UUID) (*model.LoanCampaign, error) {
	var c model.LoanCampaign
	err := r.db.WithContext(ctx).Preload("BudgetItems").Preload("Milestones", func(db *gorm.DB) *gorm.DB { return db.Order("sequence_no ASC") }).Where("id = ? AND business_id = ?", id, businessID).First(&c).Error
	return &c, err
}
func (r *CampaignRepository) UpdateCampaign(ctx context.Context, c *model.LoanCampaign) error {
	return r.db.WithContext(ctx).Save(c).Error
}
func (r *CampaignRepository) DeleteCampaign(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.LoanCampaign{}, "id = ?", id).Error
}
func (r *CampaignRepository) ListCampaignsForBusiness(ctx context.Context, businessID uuid.UUID) ([]model.LoanCampaign, error) {
	var cs []model.LoanCampaign
	err := r.db.WithContext(ctx).Preload("BudgetItems").Preload("Milestones", func(db *gorm.DB) *gorm.DB { return db.Order("sequence_no ASC") }).Where("business_id = ?", businessID).Order("created_at DESC").Find(&cs).Error
	return cs, err
}
func (r *CampaignRepository) CreateBudgetItem(ctx context.Context, b *model.CampaignBudgetItem) error {
	return r.db.WithContext(ctx).Create(b).Error
}
func (r *CampaignRepository) GetBudgetItem(ctx context.Context, id, campaignID uuid.UUID) (*model.CampaignBudgetItem, error) {
	var b model.CampaignBudgetItem
	err := r.db.WithContext(ctx).Where("id = ? AND campaign_id = ?", id, campaignID).First(&b).Error
	return &b, err
}
func (r *CampaignRepository) UpdateBudgetItem(ctx context.Context, b *model.CampaignBudgetItem) error {
	return r.db.WithContext(ctx).Save(b).Error
}
func (r *CampaignRepository) DeleteBudgetItem(ctx context.Context, id, campaignID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ? AND campaign_id = ?", id, campaignID).Delete(&model.CampaignBudgetItem{}).Error
}
func (r *CampaignRepository) ListBudgetItems(ctx context.Context, campaignID uuid.UUID) ([]model.CampaignBudgetItem, error) {
	var bs []model.CampaignBudgetItem
	err := r.db.WithContext(ctx).Where("campaign_id = ?", campaignID).Order("entry_order ASC").Find(&bs).Error
	return bs, err
}
func (r *CampaignRepository) CountBudgetItems(ctx context.Context, campaignID uuid.UUID) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.CampaignBudgetItem{}).Where("campaign_id = ?", campaignID).Count(&count).Error
	return int(count), err
}
func (r *CampaignRepository) CreateMilestone(ctx context.Context, m *model.CampaignMilestone) error {
	return r.db.WithContext(ctx).Create(m).Error
}
func (r *CampaignRepository) GetMilestone(ctx context.Context, id, campaignID uuid.UUID) (*model.CampaignMilestone, error) {
	var m model.CampaignMilestone
	err := r.db.WithContext(ctx).Where("id = ? AND campaign_id = ?", id, campaignID).First(&m).Error
	return &m, err
}
func (r *CampaignRepository) UpdateMilestone(ctx context.Context, m *model.CampaignMilestone) error {
	return r.db.WithContext(ctx).Save(m).Error
}
func (r *CampaignRepository) DeleteMilestone(ctx context.Context, id, campaignID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ? AND campaign_id = ?", id, campaignID).Delete(&model.CampaignMilestone{}).Error
}
func (r *CampaignRepository) ListMilestones(ctx context.Context, campaignID uuid.UUID) ([]model.CampaignMilestone, error) {
	var ms []model.CampaignMilestone
	err := r.db.WithContext(ctx).Where("campaign_id = ?", campaignID).Order("sequence_no ASC").Find(&ms).Error
	return ms, err
}
func (r *CampaignRepository) ReplaceMilestones(ctx context.Context, campaignID uuid.UUID, milestones []model.CampaignMilestone) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("campaign_id = ?", campaignID).Delete(&model.CampaignMilestone{}).Error; err != nil {
			return err
		}
		if len(milestones) == 0 {
			return nil
		}
		return tx.Create(&milestones).Error
	})
}
func (r *CampaignRepository) CreateFunding(ctx context.Context, funding *model.Funding) error {
	return r.db.WithContext(ctx).Create(funding).Error
}
func (r *CampaignRepository) CreateDisbursement(ctx context.Context, disbursement *model.Disbursement) error {
	return r.db.WithContext(ctx).Create(disbursement).Error
}
func (r *CampaignRepository) GetDisbursement(ctx context.Context, id uuid.UUID) (*model.Disbursement, error) {
	var disbursement model.Disbursement
	err := r.db.WithContext(ctx).First(&disbursement, "id = ?", id).Error
	return &disbursement, err
}
func (r *CampaignRepository) GetDisbursementForUpdate(ctx context.Context, id uuid.UUID) (*model.Disbursement, error) {
	var disbursement model.Disbursement
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&disbursement, "id = ?", id).Error
	return &disbursement, err
}
func (r *CampaignRepository) UpdateDisbursement(ctx context.Context, disbursement *model.Disbursement) error {
	return r.db.WithContext(ctx).Save(disbursement).Error
}
func (r *CampaignRepository) CreateFundUsageProof(ctx context.Context, proof *model.FundUsageProof) error {
	return r.db.WithContext(ctx).Create(proof).Error
}
func (r *CampaignRepository) GetFundUsageProof(ctx context.Context, id uuid.UUID) (*model.FundUsageProof, error) {
	var proof model.FundUsageProof
	err := r.db.WithContext(ctx).First(&proof, "id = ?", id).Error
	return &proof, err
}
func (r *CampaignRepository) GetFundUsageProofForUpdate(ctx context.Context, id uuid.UUID) (*model.FundUsageProof, error) {
	var proof model.FundUsageProof
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&proof, "id = ?", id).Error
	return &proof, err
}
func (r *CampaignRepository) GetActiveFundUsageProof(ctx context.Context, disbursementID uuid.UUID) (*model.FundUsageProof, error) {
	var proof model.FundUsageProof
	err := r.db.WithContext(ctx).Where("disbursement_id = ?", disbursementID).First(&proof).Error
	return &proof, err
}
func (r *CampaignRepository) DeleteFundUsageProof(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.FundUsageProof{}, "id = ?", id).Error
}
func (r *CampaignRepository) UpdateFundUsageProof(ctx context.Context, proof *model.FundUsageProof) error {
	return r.db.WithContext(ctx).Save(proof).Error
}
func (r *CampaignRepository) CreateAuditLog(ctx context.Context, audit *model.AuditLog) error {
	return r.db.WithContext(ctx).Create(audit).Error
}
func (r *CampaignRepository) HasPaidFunding(ctx context.Context, campaignID, userID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.Funding{}).Where("campaign_id = ? AND lender_user_id = ? AND status = ?", campaignID, userID, "paid").Count(&n).Error
	return n > 0, err
}
func (r *CampaignRepository) ListFundingsForLender(ctx context.Context, userID uuid.UUID) ([]model.Funding, error) {
	var rows []model.Funding
	err := r.db.WithContext(ctx).Where("lender_user_id = ?", userID).Order("funded_at DESC").Find(&rows).Error
	return rows, err
}
func (r *CampaignRepository) ListDisbursements(ctx context.Context, campaignID uuid.UUID) ([]model.Disbursement, error) {
	var rows []model.Disbursement
	err := r.db.WithContext(ctx).Where("campaign_id = ?", campaignID).Order("created_at ASC").Find(&rows).Error
	return rows, err
}
func (r *CampaignRepository) ListFundUsageProofs(ctx context.Context, status string, limit, offset int) ([]model.FundUsageProof, error) {
	var rows []model.FundUsageProof
	q := r.db.WithContext(ctx).Order("created_at ASC").Limit(limit).Offset(offset)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Find(&rows).Error
	return rows, err
}
func (r *CampaignRepository) CreateMonthlyProgressReport(ctx context.Context, report *model.MonthlyProgressReport) error {
	return r.db.WithContext(ctx).Create(report).Error
}
func (r *CampaignRepository) CreateMonthlyProgressReportProof(ctx context.Context, proof *model.MonthlyProgressReportProof) error {
	return r.db.WithContext(ctx).Create(proof).Error
}
func (r *CampaignRepository) ListMonthlyProgressReports(ctx context.Context, campaignID uuid.UUID) ([]model.MonthlyProgressReport, error) {
	var reports []model.MonthlyProgressReport
	err := r.db.WithContext(ctx).Preload("Proofs").Where("campaign_id = ?", campaignID).Order("period_year DESC, period_month DESC").Find(&reports).Error
	return reports, err
}
func (r *CampaignRepository) GetMonthlyProgressReportForUpdate(ctx context.Context, id uuid.UUID) (*model.MonthlyProgressReport, error) {
	var report model.MonthlyProgressReport
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&report, "id = ?", id).Error
	return &report, err
}
func (r *CampaignRepository) UpdateMonthlyProgressReport(ctx context.Context, report *model.MonthlyProgressReport) error {
	return r.db.WithContext(ctx).Save(report).Error
}
func (r *CampaignRepository) CreateRevenueReport(ctx context.Context, report *model.RevenueReport) error {
	return r.db.WithContext(ctx).Create(report).Error
}
func (r *CampaignRepository) GetRevenueReport(ctx context.Context, id uuid.UUID) (*model.RevenueReport, error) {
	var report model.RevenueReport
	err := r.db.WithContext(ctx).Preload("Proofs").First(&report, "id = ?", id).Error
	return &report, err
}
func (r *CampaignRepository) GetRevenueReportForUpdate(ctx context.Context, id uuid.UUID) (*model.RevenueReport, error) {
	var report model.RevenueReport
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&report, "id = ?", id).Error
	return &report, err
}
func (r *CampaignRepository) ListRevenueReports(ctx context.Context, campaignID uuid.UUID) ([]model.RevenueReport, error) {
	var reports []model.RevenueReport
	err := r.db.WithContext(ctx).Preload("Proofs").Where("campaign_id = ?", campaignID).Order("period_year DESC, period_month DESC").Find(&reports).Error
	return reports, err
}
func (r *CampaignRepository) UpdateRevenueReport(ctx context.Context, report *model.RevenueReport) error {
	return r.db.WithContext(ctx).Save(report).Error
}
func (r *CampaignRepository) CreateRevenueReportProof(ctx context.Context, proof *model.RevenueReportProof) error {
	return r.db.WithContext(ctx).Create(proof).Error
}
func (r *CampaignRepository) ListRevenueReportProofs(ctx context.Context, reportID uuid.UUID) ([]model.RevenueReportProof, error) {
	var proofs []model.RevenueReportProof
	err := r.db.WithContext(ctx).Where("revenue_report_id = ?", reportID).Order("created_at ASC").Find(&proofs).Error
	return proofs, err
}
func (r *CampaignRepository) CreateRepaymentSchedules(ctx context.Context, schedules []model.RepaymentSchedule) error {
	if len(schedules) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&schedules).Error
}
func (r *CampaignRepository) ListRepaymentSchedules(ctx context.Context, campaignID uuid.UUID) ([]model.RepaymentSchedule, error) {
	var schedules []model.RepaymentSchedule
	err := r.db.WithContext(ctx).Where("campaign_id = ?", campaignID).Order("due_date ASC, id ASC").Find(&schedules).Error
	return schedules, err
}
func (r *CampaignRepository) GetRepaymentScheduleForUpdate(ctx context.Context, id uuid.UUID) (*model.RepaymentSchedule, error) {
	var schedule model.RepaymentSchedule
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&schedule, "id = ?", id).Error
	return &schedule, err
}
func (r *CampaignRepository) UpdateRepaymentSchedule(ctx context.Context, schedule *model.RepaymentSchedule) error {
	return r.db.WithContext(ctx).Save(schedule).Error
}
func (r *CampaignRepository) CreateRepayment(ctx context.Context, repayment *model.Repayment) error {
	return r.db.WithContext(ctx).Create(repayment).Error
}
func (r *CampaignRepository) HasActiveRepayment(ctx context.Context, scheduleID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Repayment{}).
		Where("schedule_id = ? AND status IN ?", scheduleID, []string{"pending", "verifier_checked", "verified"}).
		Count(&count).Error
	return count > 0, err
}
func (r *CampaignRepository) GetRepaymentForUpdate(ctx context.Context, id uuid.UUID) (*model.Repayment, error) {
	var repayment model.Repayment
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&repayment, "id = ?", id).Error
	return &repayment, err
}
func (r *CampaignRepository) UpdateRepayment(ctx context.Context, repayment *model.Repayment) error {
	return r.db.WithContext(ctx).Save(repayment).Error
}
func (r *CampaignRepository) ListPaidFundings(ctx context.Context, campaignID uuid.UUID) ([]model.Funding, error) {
	var fundings []model.Funding
	err := r.db.WithContext(ctx).Where("campaign_id = ? AND status = ?", campaignID, "paid").Order("id ASC").Find(&fundings).Error
	return fundings, err
}
func (r *CampaignRepository) CreateLenderReturnDistributions(ctx context.Context, distributions []model.LenderReturnDistribution) error {
	if len(distributions) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&distributions).Error
}
func (r *CampaignRepository) GetLenderReturnDistributionForUpdate(ctx context.Context, id uuid.UUID) (*model.LenderReturnDistribution, error) {
	var distribution model.LenderReturnDistribution
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&distribution, "id = ?", id).Error
	return &distribution, err
}
func (r *CampaignRepository) UpdateLenderReturnDistribution(ctx context.Context, distribution *model.LenderReturnDistribution) error {
	return r.db.WithContext(ctx).Save(distribution).Error
}
func (r *CampaignRepository) ListLenderReturnDistributions(ctx context.Context, lenderID uuid.UUID) ([]model.LenderReturnDistribution, error) {
	var distributions []model.LenderReturnDistribution
	err := r.db.WithContext(ctx).Where("lender_user_id = ?", lenderID).Order("created_at DESC").Find(&distributions).Error
	return distributions, err
}
func (r *CampaignRepository) ListCatalog(ctx context.Context, f CatalogFilter) ([]model.LoanCampaign, error) {
	var cs []model.LoanCampaign
	q := r.db.WithContext(ctx).Preload("Business").Preload("Business.Category").
		Joins("JOIN businesses ON businesses.id = loan_campaigns.business_id").
		Where("loan_campaigns.status = ?", "published")
	if v := strings.TrimSpace(f.Query); v != "" {
		q = q.Where("(loan_campaigns.title ILIKE ? OR loan_campaigns.description ILIKE ? OR businesses.business_name ILIKE ?)", "%"+v+"%", "%"+v+"%", "%"+v+"%")
	}
	if v := strings.TrimSpace(f.Category); v != "" {
		q = q.Joins("JOIN business_categories ON business_categories.id = businesses.category_id").Where("business_categories.name ILIKE ?", "%"+v+"%")
	}
	if f.RiskLevel != "" {
		q = q.Where("loan_campaigns.risk_level = ?", f.RiskLevel)
	}
	if f.MinAmount > 0 {
		q = q.Where("requested_amount >= ?", f.MinAmount)
	}
	if f.MaxAmount > 0 {
		q = q.Where("requested_amount <= ?", f.MaxAmount)
	}
	if f.MinRiskScore > 0 || f.MaxRiskScore > 0 {
		q = q.Joins("JOIN LATERAL (SELECT final_score FROM risk_assessments WHERE campaign_id = loan_campaigns.id ORDER BY created_at DESC, id DESC LIMIT 1) latest_risk ON TRUE")
		if f.MinRiskScore > 0 {
			q = q.Where("latest_risk.final_score >= ?", f.MinRiskScore)
		}
		if f.MaxRiskScore > 0 {
			q = q.Where("latest_risk.final_score <= ?", f.MaxRiskScore)
		}
	}
	err := q.Order("loan_campaigns.created_at DESC").Find(&cs).Error
	if err != nil || len(cs) == 0 {
		return cs, err
	}
	ids := make([]uuid.UUID, len(cs))
	for i := range cs {
		ids[i] = cs[i].ID
	}
	var assessments []model.RiskAssessment
	if err := r.db.WithContext(ctx).Where("campaign_id IN ?", ids).Order("created_at DESC").Find(&assessments).Error; err != nil {
		return nil, err
	}
	latest := map[uuid.UUID]*model.RiskAssessment{}
	for i := range assessments {
		if latest[assessments[i].CampaignID] == nil {
			latest[assessments[i].CampaignID] = &assessments[i]
		}
	}
	for i := range cs {
		cs[i].LatestRiskAssessment = latest[cs[i].ID]
		if cs[i].LatestRiskAssessment != nil && cs[i].LatestRiskAssessment.MissingComponentsRaw != "" {
			cs[i].LatestRiskAssessment.MissingComponents = strings.Split(cs[i].LatestRiskAssessment.MissingComponentsRaw, ",")
		}
	}
	var summaries []struct {
		CampaignID       uuid.UUID
		Status           string
		ReportedAt       time.Time
		IsBusinessExists bool
		IsBusinessActive bool
		LocationMatch    bool
		Recommendation   string
	}
	if err := r.db.WithContext(ctx).Table("verification_requests").Select("verification_requests.campaign_id, verification_requests.status, verification_reports.created_at AS reported_at, verification_reports.is_business_exists, verification_reports.is_business_active, verification_reports.location_match, verification_reports.recommendation").Joins("JOIN verification_reports ON verification_reports.verification_request_id = verification_requests.id").Where("verification_requests.campaign_id IN ? AND verification_requests.status = ?", ids, "approved").Order("verification_reports.created_at DESC, verification_reports.id DESC").Scan(&summaries).Error; err != nil {
		return nil, err
	}
	latestSummary := map[uuid.UUID]*model.VerificationSummary{}
	for _, row := range summaries {
		if latestSummary[row.CampaignID] == nil {
			latestSummary[row.CampaignID] = &model.VerificationSummary{Status: row.Status, ReportedAt: row.ReportedAt, IsBusinessExists: row.IsBusinessExists, IsBusinessActive: row.IsBusinessActive, LocationMatch: row.LocationMatch, Recommendation: row.Recommendation}
		}
	}
	for i := range cs {
		cs[i].VerificationSummary = latestSummary[cs[i].ID]
	}
	return cs, nil
}
func (r *CampaignRepository) HasApprovedFieldVerification(ctx context.Context, campaignID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.VerificationRequest{}).Where("campaign_id = ? AND status = ?", campaignID, "approved").Count(&count).Error
	return count > 0, err
}
func (r *CampaignRepository) IsAssignedVerifier(ctx context.Context, campaignID, userID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.VerificationRequest{}).Where("campaign_id = ? AND assigned_verifier_id = ? AND status IN ?", campaignID, userID, []string{"assigned", "reviewed"}).Count(&n).Error
	return n > 0, err
}
