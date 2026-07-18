package repository

import (
	"context"

	"modalin-be/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FinancialSummary struct {
	TotalIncome  int64 `json:"total_income"`
	TotalExpense int64 `json:"total_expense"`
	NetCashflow  int64 `json:"net_cashflow"`
}

type Repository interface {
	CreateBusiness(ctx context.Context, business *model.Business, starter *model.StarterBusinessDetail) error
	GetBusinessByUserID(ctx context.Context, userID uuid.UUID) (*model.Business, error)
	GetBusinessByID(ctx context.Context, businessID uuid.UUID) (*model.Business, error)
	UpdateBusiness(ctx context.Context, business *model.Business, starter *model.StarterBusinessDetail) error

	CreateFinancialRecord(ctx context.Context, record *model.FinancialRecord, proofs []model.FinancialRecordProof) error
	GetFinancialRecords(ctx context.Context, businessID uuid.UUID, month, year int, recordType string) ([]model.FinancialRecord, error)
	GetFinancialRecordByID(ctx context.Context, recordID uuid.UUID, businessID uuid.UUID) (*model.FinancialRecord, error)
	UpdateFinancialRecord(ctx context.Context, record *model.FinancialRecord) error
	CreateFinancialRecordProof(ctx context.Context, proof *model.FinancialRecordProof) error
	GetFinancialSummary(ctx context.Context, businessID uuid.UUID, month, year int) (*FinancialSummary, error)
	CountDistinctFinancialRecordMonths(ctx context.Context, businessID uuid.UUID) (int, error)
	DeleteFinancialRecord(ctx context.Context, recordID uuid.UUID, businessID uuid.UUID) error
}

type BusinessRepository struct {
	db *gorm.DB
}

func NewBusinessRepository(db *gorm.DB) *BusinessRepository {
	return &BusinessRepository{db: db}
}

func (r *BusinessRepository) CreateBusiness(ctx context.Context, business *model.Business, starter *model.StarterBusinessDetail) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(business).Error; err != nil {
			return err
		}
		if business.BusinessType == "starter" && starter != nil {
			starter.BusinessID = business.ID
			if err := tx.Create(starter).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *BusinessRepository) GetBusinessByUserID(ctx context.Context, userID uuid.UUID) (*model.Business, error) {
	var business model.Business
	err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("StarterDetails").
		Where("user_id = ? AND status = ?", userID, "active").
		First(&business).Error
	if err != nil {
		return nil, err
	}
	return &business, nil
}

func (r *BusinessRepository) GetBusinessByID(ctx context.Context, businessID uuid.UUID) (*model.Business, error) {
	var business model.Business
	err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("StarterDetails").
		First(&business, "id = ?", businessID).Error
	if err != nil {
		return nil, err
	}
	return &business, nil
}

func (r *BusinessRepository) UpdateBusiness(ctx context.Context, business *model.Business, starter *model.StarterBusinessDetail) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(business).Error; err != nil {
			return err
		}
		if business.BusinessType == "starter" && starter != nil {
			starter.BusinessID = business.ID
			var existingStarter model.StarterBusinessDetail
			err := tx.Where("business_id = ?", business.ID).First(&existingStarter).Error
			if err == gorm.ErrRecordNotFound {
				if err := tx.Create(starter).Error; err != nil {
					return err
				}
			} else if err == nil {
				starter.ID = existingStarter.ID
				if err := tx.Save(starter).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
		return nil
	})
}

func (r *BusinessRepository) CreateFinancialRecord(ctx context.Context, record *model.FinancialRecord, proofs []model.FinancialRecordProof) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(record).Error; err != nil {
			return err
		}
		for i := range proofs {
			proofs[i].FinancialRecordID = record.ID
			if err := tx.Create(&proofs[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *BusinessRepository) GetFinancialRecords(ctx context.Context, businessID uuid.UUID, month, year int, recordType string) ([]model.FinancialRecord, error) {
	var records []model.FinancialRecord
	query := r.db.WithContext(ctx).Preload("Proofs").Where("business_id = ?", businessID)

	if year > 0 {
		query = query.Where("EXTRACT(YEAR FROM record_date) = ?", year)
	}
	if month > 0 {
		query = query.Where("EXTRACT(MONTH FROM record_date) = ?", month)
	}
	if recordType == "income" {
		query = query.Where("income_amount > 0")
	} else if recordType == "expense" {
		query = query.Where("expense_amount > 0")
	}

	err := query.Order("record_date DESC").Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (r *BusinessRepository) GetFinancialRecordByID(ctx context.Context, recordID uuid.UUID, businessID uuid.UUID) (*model.FinancialRecord, error) {
	var record model.FinancialRecord
	err := r.db.WithContext(ctx).Preload("Proofs").Where("id = ? AND business_id = ?", recordID, businessID).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *BusinessRepository) UpdateFinancialRecord(ctx context.Context, record *model.FinancialRecord) error {
	return r.db.WithContext(ctx).Save(record).Error
}

func (r *BusinessRepository) CreateFinancialRecordProof(ctx context.Context, proof *model.FinancialRecordProof) error {
	return r.db.WithContext(ctx).Create(proof).Error
}

func (r *BusinessRepository) GetFinancialSummary(ctx context.Context, businessID uuid.UUID, month, year int) (*FinancialSummary, error) {
	type result struct {
		TotalIncome  int64
		TotalExpense int64
	}
	var res result
	query := r.db.WithContext(ctx).Model(&model.FinancialRecord{}).Where("business_id = ?", businessID)

	if year > 0 {
		query = query.Where("EXTRACT(YEAR FROM record_date) = ?", year)
	}
	if month > 0 {
		query = query.Where("EXTRACT(MONTH FROM record_date) = ?", month)
	}

	err := query.Select("COALESCE(SUM(income_amount), 0) as total_income, COALESCE(SUM(expense_amount), 0) as total_expense").
		Scan(&res).Error
	if err != nil {
		return nil, err
	}

	return &FinancialSummary{
		TotalIncome:  res.TotalIncome,
		TotalExpense: res.TotalExpense,
		NetCashflow:  res.TotalIncome - res.TotalExpense,
	}, nil
}

func (r *BusinessRepository) CountDistinctFinancialRecordMonths(ctx context.Context, businessID uuid.UUID) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.FinancialRecord{}).
		Where("business_id = ?", businessID).
		Select("COUNT(DISTINCT(TO_CHAR(record_date, 'YYYY-MM')))").
		Scan(&count).Error
	return int(count), err
}

func (r *BusinessRepository) DeleteFinancialRecord(ctx context.Context, recordID uuid.UUID, businessID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("financial_record_id = ?", recordID).Delete(&model.FinancialRecordProof{}).Error; err != nil {
			return err
		}
		result := tx.Where("id = ? AND business_id = ?", recordID, businessID).Delete(&model.FinancialRecord{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}
