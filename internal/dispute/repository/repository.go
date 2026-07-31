package repository

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"modalin-be/internal/dispute/service"
	"modalin-be/internal/model"
)

type Repo struct{ db *gorm.DB }

func New(db *gorm.DB) *Repo { return &Repo{db} }
func (r *Repo) WithTransaction(c context.Context, f func(service.Repository) error) error {
	return r.db.WithContext(c).Transaction(func(tx *gorm.DB) error { return f(&Repo{tx}) })
}
func (r *Repo) Campaign(c context.Context, id uuid.UUID) (*model.LoanCampaign, error) {
	var v model.LoanCampaign
	e := r.db.WithContext(c).Preload("Business").First(&v, "id = ?", id).Error
	return &v, e
}
func (r *Repo) HasPaidFunding(c context.Context, cid, uid uuid.UUID) (bool, error) {
	var n int64
	e := r.db.WithContext(c).Model(&model.Funding{}).Where("campaign_id=? AND lender_user_id=? AND status=?", cid, uid, "paid").Count(&n).Error
	return n > 0, e
}
func (r *Repo) Create(c context.Context, v *model.Dispute) error {
	return r.db.WithContext(c).Create(v).Error
}
func (r *Repo) GetForUpdate(c context.Context, id uuid.UUID) (*model.Dispute, error) {
	var v model.Dispute
	e := r.db.WithContext(c).Clauses(clause.Locking{Strength: "UPDATE"}).First(&v, "id=?", id).Error
	return &v, e
}
func (r *Repo) Update(c context.Context, v *model.Dispute) error {
	return r.db.WithContext(c).Save(v).Error
}
func (r *Repo) CreateAuditLog(c context.Context, v *model.AuditLog) error {
	return r.db.WithContext(c).Create(v).Error
}
func (r *Repo) List(c context.Context, uid uuid.UUID, admin bool, status string, limit, offset int) ([]model.Dispute, int64, error) {
	var rows []model.Dispute
	var total int64
	q := r.db.WithContext(c).Model(&model.Dispute{})
	if !admin {
		q = q.Joins("JOIN loan_campaigns ON loan_campaigns.id = disputes.campaign_id").Joins("JOIN businesses ON businesses.id = loan_campaigns.business_id").Joins("LEFT JOIN fundings ON fundings.campaign_id = disputes.campaign_id AND fundings.lender_user_id = ? AND fundings.status = 'paid'", uid).Where("businesses.user_id = ? OR fundings.id IS NOT NULL", uid)
	}
	if status != "" {
		q = q.Where("disputes.status=?", status)
	}
	if e := q.Count(&total).Error; e != nil {
		return nil, 0, e
	}
	e := q.Order("disputes.created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, e
}
