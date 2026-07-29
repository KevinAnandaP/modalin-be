package repository

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"modalin-be/internal/model"
)

type Repo struct{ DB *gorm.DB }

func New(db *gorm.DB) *Repo { return &Repo{DB: db} }
func (r *Repo) Tx(ctx context.Context, fn func(*Repo) error) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return fn(&Repo{DB: tx}) })
}
func (r *Repo) Campaign(ctx context.Context, id uuid.UUID) (*model.LoanCampaign, error) {
	var v model.LoanCampaign
	e := r.DB.WithContext(ctx).Preload("Business").Clauses(clause.Locking{Strength: "UPDATE"}).First(&v, "id=?", id).Error
	return &v, e
}
func (r *Repo) Schedules(ctx context.Context, id uuid.UUID) ([]model.RepaymentSchedule, error) {
	var v []model.RepaymentSchedule
	e := r.DB.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("campaign_id=?", id).Order("due_date,id").Find(&v).Error
	return v, e
}
func (r *Repo) PendingRepayment(ctx context.Context, id uuid.UUID) (bool, error) {
	var n int64
	e := r.DB.WithContext(ctx).Model(&model.Repayment{}).Where("campaign_id=? AND status IN ?", id, []string{"pending", "verifier_checked"}).Count(&n).Error
	return n > 0, e
}
func (r *Repo) ActiveDispute(ctx context.Context, id uuid.UUID) (bool, error) {
	var n int64
	e := r.DB.WithContext(ctx).Model(&model.Dispute{}).Where("campaign_id=? AND status IN ?", id, []string{"open", "under_review"}).Count(&n).Error
	return n > 0, e
}
func (r *Repo) Create(ctx context.Context, v *model.RepaymentRestructuringRequest) error {
	return r.DB.WithContext(ctx).Create(v).Error
}
func (r *Repo) Get(ctx context.Context, id uuid.UUID) (*model.RepaymentRestructuringRequest, error) {
	var v model.RepaymentRestructuringRequest
	e := r.DB.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&v, "id=?", id).Error
	return &v, e
}
func (r *Repo) SaveRequest(ctx context.Context, v *model.RepaymentRestructuringRequest) error {
	return r.DB.WithContext(ctx).Save(v).Error
}
func (r *Repo) SaveCampaign(ctx context.Context, v *model.LoanCampaign) error {
	return r.DB.WithContext(ctx).Save(v).Error
}
func (r *Repo) SaveSchedule(ctx context.Context, v *model.RepaymentSchedule) error {
	return r.DB.WithContext(ctx).Save(v).Error
}
func (r *Repo) AddSchedules(ctx context.Context, v []model.RepaymentSchedule) error {
	return r.DB.WithContext(ctx).Create(&v).Error
}
func (r *Repo) Audit(ctx context.Context, v *model.AuditLog) error {
	return r.DB.WithContext(ctx).Create(v).Error
}
func (r *Repo) List(ctx context.Context, user uuid.UUID, campaignID *uuid.UUID, admin bool, status string, limit, offset int) ([]model.RepaymentRestructuringRequest, int64, error) {
	var rows []model.RepaymentRestructuringRequest
	var total int64
	q := r.DB.WithContext(ctx).Model(&model.RepaymentRestructuringRequest{})
	if !admin {
		q = q.Where("requested_by=?", user)
	}
	if campaignID != nil {
		q = q.Where("campaign_id=?", *campaignID)
	}
	if status != "" {
		q = q.Where("status=?", status)
	}
	if e := q.Count(&total).Error; e != nil {
		return nil, 0, e
	}
	e := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, e
}
