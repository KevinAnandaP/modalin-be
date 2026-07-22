package repository

import (
	"context"
	"strings"

	"modalin-be/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CatalogFilter struct {
	Query, Category, RiskLevel string
	MinAmount, MaxAmount       int64
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
	ListCatalog(context.Context, CatalogFilter) ([]model.LoanCampaign, error)
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
	err := q.Order("loan_campaigns.created_at DESC").Find(&cs).Error
	return cs, err
}
