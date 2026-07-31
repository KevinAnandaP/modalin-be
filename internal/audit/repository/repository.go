package repository

import (
	"context"
	"time"

	"modalin-be/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Filter struct {
	UserID, EntityID   *uuid.UUID
	Action, EntityType string
	From, To           *time.Time
	Limit, Offset      int
}

type Repository interface {
	List(context.Context, Filter) ([]model.AuditLog, int64, error)
}

type PostgresRepository struct{ db *gorm.DB }

func New(db *gorm.DB) *PostgresRepository { return &PostgresRepository{db: db} }

func (r *PostgresRepository) List(ctx context.Context, filter Filter) ([]model.AuditLog, int64, error) {
	var rows []model.AuditLog
	var total int64
	q := r.db.WithContext(ctx).Model(&model.AuditLog{})
	if filter.UserID != nil {
		q = q.Where("user_id = ?", *filter.UserID)
	}
	if filter.EntityID != nil {
		q = q.Where("entity_id = ?", *filter.EntityID)
	}
	if filter.Action != "" {
		q = q.Where("action = ?", filter.Action)
	}
	if filter.EntityType != "" {
		q = q.Where("entity_type = ?", filter.EntityType)
	}
	if filter.From != nil {
		q = q.Where("created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		q = q.Where("created_at <= ?", *filter.To)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("created_at DESC, id DESC").Limit(filter.Limit).Offset(filter.Offset).Find(&rows).Error
	return rows, total, err
}
