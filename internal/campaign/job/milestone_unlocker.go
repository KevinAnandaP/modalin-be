package job

import (
	"context"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

const unlockInterval = time.Minute

// ProofQualifiesForUnlock makes the approval rule reusable by proof-review flows.
func ProofQualifiesForUnlock(status string, createdAt, now time.Time) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "approved":
		return true
	default:
		return false
	}
}

// StartMilestoneUnlockScheduler reconciles milestones that have an approved proof.
// It is intentionally not started by the API: approval releases the next
// milestone synchronously in the campaign service.
func StartMilestoneUnlockScheduler(db *gorm.DB) {
	go func() {
		if _, err := UnlockEligibleMilestones(context.Background(), db, time.Now().UTC()); err != nil {
			log.Printf("campaign milestone unlock job failed: %v", err)
		}
		ticker := time.NewTicker(unlockInterval)
		defer ticker.Stop()
		for now := range ticker.C {
			if _, err := UnlockEligibleMilestones(context.Background(), db, now.UTC()); err != nil {
				log.Printf("campaign milestone unlock job failed: %v", err)
			}
		}
	}()
}

func UnlockEligibleMilestones(ctx context.Context, db *gorm.DB, now time.Time) (int64, error) {
	result := db.WithContext(ctx).Exec(`
		UPDATE campaign_milestones AS next
		SET status = 'available'
		FROM campaign_milestones AS previous, loan_campaigns AS campaign
		WHERE next.campaign_id = previous.campaign_id
		  AND campaign.id = next.campaign_id
		  AND next.sequence_no = previous.sequence_no + 1
		  AND next.status = 'locked'
		  AND next.deleted_at IS NULL
		  AND previous.deleted_at IS NULL
		  AND campaign.status IN ('published', 'funded', 'active')
		  AND EXISTS (
			SELECT 1
			FROM disbursements AS disbursement
			JOIN fund_usage_proofs AS proof ON proof.disbursement_id = disbursement.id
			WHERE disbursement.milestone_id = previous.id
			  AND disbursement.deleted_at IS NULL
			  AND proof.deleted_at IS NULL
			  AND proof.status = 'approved'
		  )`)
	return result.RowsAffected, result.Error
}
