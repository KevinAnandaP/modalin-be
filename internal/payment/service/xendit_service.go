package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"modalin-be/internal/model"
	"modalin-be/pkg/xendit"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type XenditPaymentService interface {
	ProcessInvoiceCallback(ctx context.Context, payload xendit.InvoiceWebhookPayload) error
	ProcessDisbursementCallback(ctx context.Context, payload xendit.DisbursementWebhookPayload) error
}

type xenditPaymentService struct {
	db *gorm.DB
}

func NewXenditPaymentService(db *gorm.DB) XenditPaymentService {
	return &xenditPaymentService{db: db}
}

func (s *xenditPaymentService) ProcessInvoiceCallback(ctx context.Context, payload xendit.InvoiceWebhookPayload) error {
	log.Printf("[Xendit Webhook] Processing invoice callback ID: %s, ExternalID: %s, Status: %s", payload.ID, payload.ExternalID, payload.Status)

	if !strings.EqualFold(payload.Status, "PAID") && !strings.EqualFold(payload.Status, "SETTLED") {
		log.Printf("[Xendit Webhook] Invoice status %s ignored", payload.Status)
		return nil
	}

	parts := strings.Split(payload.ExternalID, "-")
	if len(parts) < 2 {
		return fmt.Errorf("invalid external_id format: %s", payload.ExternalID)
	}

	prefix := parts[0]
	targetIDStr := strings.Join(parts[1:], "-")
	targetID, err := uuid.Parse(targetIDStr)
	if err != nil {
		return fmt.Errorf("failed to parse uuid from external_id %s: %w", payload.ExternalID, err)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		switch prefix {
		case "FUNDING":
			return s.handleFundingPaid(tx, targetID, payload)
		case "REPAYMENT":
			return s.handleRepaymentPaid(tx, targetID, payload)
		default:
			return fmt.Errorf("unknown external_id prefix: %s", prefix)
		}
	})
}

func (s *xenditPaymentService) handleFundingPaid(tx *gorm.DB, fundingID uuid.UUID, payload xendit.InvoiceWebhookPayload) error {
	var funding model.Funding
	if err := tx.Where("id = ?", fundingID).First(&funding).Error; err != nil {
		return fmt.Errorf("funding not found: %w", err)
	}

	if funding.Status == "paid" {
		log.Printf("[Xendit Webhook] Funding %s is already paid", fundingID)
		return nil
	}

	paidAt := time.Now()
	if payload.PaidAt != nil {
		paidAt = *payload.PaidAt
	}

	statusPaid := "PAID"
	paymentMethod := payload.PaymentChannel
	if paymentMethod == "" {
		paymentMethod = payload.PaymentMethod
	}

	funding.Status = "paid"
	funding.FundedAt = paidAt
	funding.XenditPaymentStatus = &statusPaid
	funding.XenditPaymentMethod = &paymentMethod
	funding.XenditPaidAt = &paidAt

	if err := tx.Save(&funding).Error; err != nil {
		return fmt.Errorf("failed to update funding status: %w", err)
	}

	// Update Loan Campaign FundedAmount
	var campaign model.LoanCampaign
	if err := tx.Where("id = ?", funding.CampaignID).First(&campaign).Error; err != nil {
		return fmt.Errorf("campaign not found: %w", err)
	}

	campaign.FundedAmount += funding.Amount
	if campaign.FundedAmount >= campaign.RequestedAmount && campaign.Status == "published" {
		campaign.Status = "funded"
	}

	if err := tx.Save(&campaign).Error; err != nil {
		return fmt.Errorf("failed to update campaign funded amount: %w", err)
	}

	log.Printf("[Xendit Webhook] Successfully processed funding %s for campaign %s", fundingID, campaign.ID)
	return nil
}

func (s *xenditPaymentService) handleRepaymentPaid(tx *gorm.DB, repaymentID uuid.UUID, payload xendit.InvoiceWebhookPayload) error {
	var repayment model.Repayment
	if err := tx.Where("id = ?", repaymentID).First(&repayment).Error; err != nil {
		return fmt.Errorf("repayment not found: %w", err)
	}

	if repayment.Status == "verified" {
		log.Printf("[Xendit Webhook] Repayment %s is already verified", repaymentID)
		return nil
	}

	paidAt := time.Now()
	if payload.PaidAt != nil {
		paidAt = *payload.PaidAt
	}

	statusPaid := "PAID"
	paymentMethod := payload.PaymentChannel
	if paymentMethod == "" {
		paymentMethod = payload.PaymentMethod
	}

	repayment.Status = "verified"
	repayment.PaidAt = paidAt
	repayment.XenditPaymentStatus = &statusPaid
	repayment.XenditPaymentMethod = &paymentMethod
	repayment.XenditPaidAt = &paidAt

	if err := tx.Save(&repayment).Error; err != nil {
		return fmt.Errorf("failed to update repayment status: %w", err)
	}

	// Update schedule status to paid
	if err := tx.Model(&model.RepaymentSchedule{}).
		Where("id = ?", repayment.ScheduleID).
		Updates(map[string]interface{}{"status": "paid"}).Error; err != nil {
		return fmt.Errorf("failed to update repayment schedule: %w", err)
	}

	// Calculate and generate Lender Return Distributions
	var fundings []model.Funding
	if err := tx.Where("campaign_id = ? AND status = ?", repayment.CampaignID, "paid").Find(&fundings).Error; err != nil {
		return fmt.Errorf("failed to fetch campaign fundings: %w", err)
	}

	var campaign model.LoanCampaign
	if err := tx.Where("id = ?", repayment.CampaignID).First(&campaign).Error; err != nil {
		return fmt.Errorf("campaign not found: %w", err)
	}

	if campaign.FundedAmount > 0 {
		for _, f := range fundings {
			// Proportional distribution based on lender's funding share
			shareRatio := float64(f.Amount) / float64(campaign.FundedAmount)
			principalPart := int64(float64(repayment.PrincipalPaid) * shareRatio)
			benefitPart := int64(float64(repayment.BenefitPaid) * shareRatio)

			dist := model.LenderReturnDistribution{
				RepaymentID:     repayment.ID,
				FundingID:       f.ID,
				LenderUserID:    f.LenderUserID,
				PrincipalAmount: principalPart,
				BenefitAmount:   benefitPart,
				Status:          "pending",
			}
			if err := tx.Create(&dist).Error; err != nil {
				log.Printf("Warning: Failed to auto-generate return distribution for lender %s: %v", f.LenderUserID, err)
			}
		}
	}

	log.Printf("[Xendit Webhook] Successfully processed repayment %s and created distributions", repaymentID)
	return nil
}

func (s *xenditPaymentService) ProcessDisbursementCallback(ctx context.Context, payload xendit.DisbursementWebhookPayload) error {
	log.Printf("[Xendit Webhook] Processing disbursement callback ID: %s, ExternalID: %s, Status: %s", payload.ID, payload.ExternalID, payload.Status)

	parts := strings.Split(payload.ExternalID, "-")
	if len(parts) < 2 {
		return fmt.Errorf("invalid disbursement external_id format: %s", payload.ExternalID)
	}

	prefix := parts[0]
	targetIDStr := strings.Join(parts[1:], "-")
	targetID, err := uuid.Parse(targetIDStr)
	if err != nil {
		return fmt.Errorf("failed to parse uuid from external_id %s: %w", payload.ExternalID, err)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		switch prefix {
		case "DISBURSED":
			return s.handleDisbursementPayout(tx, targetID, payload)
		case "RETURN":
			return s.handleLenderReturnPayout(tx, targetID, payload)
		default:
			return fmt.Errorf("unknown disbursement external_id prefix: %s", prefix)
		}
	})
}

func (s *xenditPaymentService) handleDisbursementPayout(tx *gorm.DB, disbID uuid.UUID, payload xendit.DisbursementWebhookPayload) error {
	var disb model.Disbursement
	if err := tx.Where("id = ?", disbID).First(&disb).Error; err != nil {
		return fmt.Errorf("disbursement record not found: %w", err)
	}

	disb.XenditDisbursementStatus = &payload.Status
	disb.TransferReference = &payload.ID
	if strings.EqualFold(payload.Status, "COMPLETED") || strings.EqualFold(payload.Status, "SUCCESS") {
		disb.Status = "verified"
		now := time.Now()
		disb.ReleasedAt = &now
	} else if strings.EqualFold(payload.Status, "FAILED") {
		disb.Status = "rejected"
	}

	return tx.Save(&disb).Error
}

func (s *xenditPaymentService) handleLenderReturnPayout(tx *gorm.DB, distID uuid.UUID, payload xendit.DisbursementWebhookPayload) error {
	var dist model.LenderReturnDistribution
	if err := tx.Where("id = ?", distID).First(&dist).Error; err != nil {
		return fmt.Errorf("lender return distribution record not found: %w", err)
	}

	dist.XenditDisbursementStatus = &payload.Status
	dist.TransferReference = &payload.ID
	if strings.EqualFold(payload.Status, "COMPLETED") || strings.EqualFold(payload.Status, "SUCCESS") {
		dist.Status = "distributed"
		now := time.Now()
		dist.DistributedAt = &now
	} else if strings.EqualFold(payload.Status, "FAILED") {
		dist.Status = "failed"
	}

	return tx.Save(&dist).Error
}
