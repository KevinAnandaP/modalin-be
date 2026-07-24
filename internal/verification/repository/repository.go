package repository

import (
	"context"
	"strings"

	"modalin-be/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	WithTransaction(context.Context, func(Repository) error) error
	GetCampaign(context.Context, uuid.UUID) (*model.LoanCampaign, error)
	GetCampaignForUpdate(context.Context, uuid.UUID) (*model.LoanCampaign, error)
	UpdateCampaign(context.Context, *model.LoanCampaign) error
	LatestCampaignID(context.Context, uuid.UUID) (uuid.UUID, error)
	ListActiveCampaigns(context.Context) ([]model.LoanCampaign, error)
	ListActiveCampaignsForBusiness(context.Context, uuid.UUID) ([]model.LoanCampaign, error)
	GetBusiness(context.Context, uuid.UUID) (*model.Business, error)
	GetRequest(context.Context, uuid.UUID) (*model.VerificationRequest, error)
	GetRequestForUpdate(context.Context, uuid.UUID) (*model.VerificationRequest, error)
	CreateRequest(context.Context, *model.VerificationRequest) error
	HasActiveVerificationRequest(context.Context, uuid.UUID) (bool, error)
	UpdateRequest(context.Context, *model.VerificationRequest) error
	ListRequests(context.Context, string, *uuid.UUID) ([]model.VerificationRequest, error)
	ListRequestsPage(context.Context, RequestFilter) ([]model.VerificationRequest, int64, error)
	HasActiveRole(context.Context, uuid.UUID, string) (bool, error)
	HasAnyActiveRole(context.Context, uuid.UUID) (bool, error)
	IsAssignedVerifier(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	HasFunding(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	CreateReport(context.Context, *model.VerificationReport) error
	HasReport(context.Context, uuid.UUID) (bool, error)
	UpdateBusiness(context.Context, *model.Business) error
	UpsertVote(context.Context, *model.CommunityVote) error
	CountFinancialRecords(context.Context, uuid.UUID) (int64, error)
	CountFinancialMonths(context.Context, uuid.UUID) (int, error)
	RepaymentStats(context.Context, uuid.UUID) (paid, late, total int64, err error)
	VoteStats(context.Context, uuid.UUID) (up, down int64, err error)
	LatestReport(context.Context, uuid.UUID) (*model.VerificationReport, error)
	LatestReportForCampaign(context.Context, uuid.UUID) (*model.VerificationReport, error)
	CreateRiskAssessment(context.Context, *model.RiskAssessment) error
	LatestRiskAssessment(context.Context, uuid.UUID) (*model.RiskAssessment, error)
	CreateAuditLog(context.Context, *model.AuditLog) error
}

type RequestFilter struct {
	Status                 string
	VerifierID, CampaignID *uuid.UUID
	Limit, Offset          int
}

type PostgresRepository struct{ db *gorm.DB }

func New(db *gorm.DB) *PostgresRepository { return &PostgresRepository{db: db} }
func (r *PostgresRepository) WithTransaction(ctx context.Context, fn func(Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return fn(&PostgresRepository{db: tx}) })
}
func (r *PostgresRepository) GetCampaign(ctx context.Context, id uuid.UUID) (*model.LoanCampaign, error) {
	var v model.LoanCampaign
	err := r.db.WithContext(ctx).First(&v, "id = ?", id).Error
	return &v, err
}
func (r *PostgresRepository) GetCampaignForUpdate(ctx context.Context, id uuid.UUID) (*model.LoanCampaign, error) {
	var v model.LoanCampaign
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&v, "id = ?", id).Error
	return &v, err
}
func (r *PostgresRepository) UpdateCampaign(ctx context.Context, v *model.LoanCampaign) error {
	return r.db.WithContext(ctx).Save(v).Error
}
func (r *PostgresRepository) LatestCampaignID(ctx context.Context, businessID uuid.UUID) (uuid.UUID, error) {
	var campaign model.LoanCampaign
	err := r.db.WithContext(ctx).Select("id").Where("business_id = ?", businessID).Order("created_at DESC").First(&campaign).Error
	return campaign.ID, err
}
func (r *PostgresRepository) ListActiveCampaigns(ctx context.Context) ([]model.LoanCampaign, error) {
	var campaigns []model.LoanCampaign
	err := r.db.WithContext(ctx).Where("status IN ?", []string{"published", "funded", "active"}).Find(&campaigns).Error
	return campaigns, err
}
func (r *PostgresRepository) ListActiveCampaignsForBusiness(ctx context.Context, businessID uuid.UUID) ([]model.LoanCampaign, error) {
	var campaigns []model.LoanCampaign
	err := r.db.WithContext(ctx).Where("business_id = ? AND status IN ?", businessID, []string{"published", "funded", "active"}).Find(&campaigns).Error
	return campaigns, err
}
func (r *PostgresRepository) GetBusiness(ctx context.Context, id uuid.UUID) (*model.Business, error) {
	var v model.Business
	err := r.db.WithContext(ctx).First(&v, "id = ?", id).Error
	return &v, err
}
func (r *PostgresRepository) GetRequest(ctx context.Context, id uuid.UUID) (*model.VerificationRequest, error) {
	var v model.VerificationRequest
	err := r.db.WithContext(ctx).Preload("Campaign").First(&v, "id = ?", id).Error
	return &v, err
}
func (r *PostgresRepository) GetRequestForUpdate(ctx context.Context, id uuid.UUID) (*model.VerificationRequest, error) {
	var v model.VerificationRequest
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&v, "id = ?", id).Error
	return &v, err
}
func (r *PostgresRepository) CreateRequest(ctx context.Context, v *model.VerificationRequest) error {
	return r.db.WithContext(ctx).Create(v).Error
}
func (r *PostgresRepository) HasActiveVerificationRequest(ctx context.Context, campaignID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.VerificationRequest{}).Where("campaign_id = ? AND status IN ?", campaignID, []string{"pending", "assigned", "reviewed"}).Count(&n).Error
	return n > 0, err
}
func (r *PostgresRepository) UpdateRequest(ctx context.Context, v *model.VerificationRequest) error {
	return r.db.WithContext(ctx).Save(v).Error
}
func (r *PostgresRepository) ListRequests(ctx context.Context, status string, verifier *uuid.UUID) ([]model.VerificationRequest, error) {
	var v []model.VerificationRequest
	q := r.db.WithContext(ctx).Preload("Business").Preload("Campaign").Order("created_at DESC")
	if strings.TrimSpace(status) != "" {
		q = q.Where("status = ?", status)
	}
	if verifier != nil {
		q = q.Where("assigned_verifier_id = ?", *verifier)
	}
	err := q.Find(&v).Error
	return v, err
}
func (r *PostgresRepository) ListRequestsPage(ctx context.Context, f RequestFilter) ([]model.VerificationRequest, int64, error) {
	var rows []model.VerificationRequest
	var total int64
	q := r.db.WithContext(ctx).Model(&model.VerificationRequest{})
	if strings.TrimSpace(f.Status) != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.VerifierID != nil {
		q = q.Where("assigned_verifier_id = ?", *f.VerifierID)
	}
	if f.CampaignID != nil {
		q = q.Where("campaign_id = ?", *f.CampaignID)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Preload("Business").Preload("Campaign").Order("created_at DESC").Limit(f.Limit).Offset(f.Offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
func (r *PostgresRepository) HasActiveRole(ctx context.Context, userID uuid.UUID, role string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("user_roles").Joins("JOIN roles ON roles.id = user_roles.role_id").Where("user_roles.user_id = ? AND user_roles.status = ? AND roles.name = ?", userID, "approved", role).Count(&n).Error
	return n > 0, err
}
func (r *PostgresRepository) HasAnyActiveRole(ctx context.Context, userID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.UserRole{}).Where("user_id = ? AND status = ?", userID, "approved").Count(&n).Error
	return n > 0, err
}
func (r *PostgresRepository) IsAssignedVerifier(ctx context.Context, businessID, userID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.VerificationRequest{}).Where("business_id = ? AND assigned_verifier_id = ? AND status IN ?", businessID, userID, []string{"assigned", "reviewed"}).Count(&n).Error
	return n > 0, err
}
func (r *PostgresRepository) HasFunding(ctx context.Context, campaignID, userID uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.Funding{}).Where("campaign_id = ? AND lender_user_id = ?", campaignID, userID).Count(&n).Error
	return n > 0, err
}
func (r *PostgresRepository) CreateReport(ctx context.Context, v *model.VerificationReport) error {
	return r.db.WithContext(ctx).Create(v).Error
}
func (r *PostgresRepository) HasReport(ctx context.Context, id uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.VerificationReport{}).Where("verification_request_id = ?", id).Count(&n).Error
	return n > 0, err
}
func (r *PostgresRepository) UpdateBusiness(ctx context.Context, v *model.Business) error {
	return r.db.WithContext(ctx).Save(v).Error
}
func (r *PostgresRepository) UpsertVote(ctx context.Context, v *model.CommunityVote) error {
	return r.db.WithContext(ctx).Where("business_id = ? AND user_id = ?", v.BusinessID, v.UserID).Assign(map[string]any{"vote_type": v.VoteType, "reason": v.Reason}).FirstOrCreate(v).Error
}
func (r *PostgresRepository) CountFinancialRecords(ctx context.Context, id uuid.UUID) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.FinancialRecord{}).Where("business_id = ? AND status IN ?", id, []string{"submitted", "verified"}).Count(&n).Error
	return n, err
}
func (r *PostgresRepository) CountFinancialMonths(ctx context.Context, id uuid.UUID) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.FinancialRecord{}).Where("business_id = ?", id).Distinct("date_trunc('month', record_date)").Count(&count).Error
	return int(count), err
}
func (r *PostgresRepository) RepaymentStats(ctx context.Context, id uuid.UUID) (paid, late, total int64, err error) {
	q := r.db.WithContext(ctx).Model(&model.RepaymentSchedule{}).Where("campaign_id = ?", id)
	err = q.Count(&total).Error
	if err != nil {
		return
	}
	err = r.db.WithContext(ctx).Model(&model.RepaymentSchedule{}).Where("campaign_id = ? AND status = ?", id, "paid").Count(&paid).Error
	if err != nil {
		return
	}
	err = r.db.WithContext(ctx).Model(&model.RepaymentSchedule{}).Where("campaign_id = ? AND status IN ?", id, []string{"late", "restructured"}).Count(&late).Error
	return
}
func (r *PostgresRepository) VoteStats(ctx context.Context, id uuid.UUID) (up, down int64, err error) {
	err = r.db.WithContext(ctx).Model(&model.CommunityVote{}).Where("business_id = ? AND vote_type = ?", id, "up").Count(&up).Error
	if err == nil {
		err = r.db.WithContext(ctx).Model(&model.CommunityVote{}).Where("business_id = ? AND vote_type = ?", id, "down").Count(&down).Error
	}
	return
}
func (r *PostgresRepository) LatestReport(ctx context.Context, id uuid.UUID) (*model.VerificationReport, error) {
	var v model.VerificationReport
	err := r.db.WithContext(ctx).Where("verification_request_id = ?", id).Order("created_at DESC").First(&v).Error
	return &v, err
}
func (r *PostgresRepository) LatestReportForCampaign(ctx context.Context, campaignID uuid.UUID) (*model.VerificationReport, error) {
	var v model.VerificationReport
	err := r.db.WithContext(ctx).Joins("JOIN verification_requests ON verification_requests.id = verification_reports.verification_request_id").Where("verification_requests.campaign_id = ? AND verification_requests.status = ?", campaignID, "approved").Order("verification_reports.created_at DESC").First(&v).Error
	return &v, err
}
func (r *PostgresRepository) CreateRiskAssessment(ctx context.Context, v *model.RiskAssessment) error {
	return r.db.WithContext(ctx).Create(v).Error
}
func (r *PostgresRepository) LatestRiskAssessment(ctx context.Context, campaignID uuid.UUID) (*model.RiskAssessment, error) {
	var v model.RiskAssessment
	err := r.db.WithContext(ctx).Where("campaign_id = ?", campaignID).Order("created_at DESC").First(&v).Error
	return &v, err
}
func (r *PostgresRepository) CreateAuditLog(ctx context.Context, v *model.AuditLog) error {
	return r.db.WithContext(ctx).Create(v).Error
}
