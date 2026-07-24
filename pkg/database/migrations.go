package database

import (
	"fmt"

	"gorm.io/gorm"
)

// migration is intentionally small: schema discovery remains handled by GORM,
// while database invariants that GORM cannot model are versioned here.
type migration struct {
	Version string
	Up      []string
	Down    []string
}

var applicationMigrations = []migration{
	{
		Version: "20260724_risk_assessment_backfill",
		Up: []string{
			"ALTER TABLE risk_assessments ADD COLUMN IF NOT EXISTS data_limited boolean NOT NULL DEFAULT false",
			"ALTER TABLE risk_assessments ADD COLUMN IF NOT EXISTS missing_components text NOT NULL DEFAULT ''",
			"UPDATE risk_assessments SET data_limited = true, missing_components = 'financial,verification,repayment,community' WHERE missing_components = ''",
		},
		Down: []string{
			"ALTER TABLE risk_assessments DROP COLUMN IF EXISTS missing_components",
			"ALTER TABLE risk_assessments DROP COLUMN IF EXISTS data_limited",
		},
	},
	{
		Version: "20260724_verification_guards",
		Up: []string{
			"CREATE UNIQUE INDEX IF NOT EXISTS idx_verification_requests_one_active_per_campaign ON verification_requests (campaign_id) WHERE deleted_at IS NULL AND status IN ('pending', 'assigned', 'reviewed')",
			"CREATE UNIQUE INDEX IF NOT EXISTS idx_community_votes_one_active_per_business_user ON community_votes (business_id, user_id) WHERE deleted_at IS NULL",
		},
		Down: []string{
			"DROP INDEX IF EXISTS idx_community_votes_one_active_per_business_user",
			"DROP INDEX IF EXISTS idx_verification_requests_one_active_per_campaign",
		},
	},
	{
		Version: "20260724_repayment_guards",
		Up: []string{
			"CREATE UNIQUE INDEX IF NOT EXISTS idx_repayments_one_active_per_schedule ON repayments (schedule_id) WHERE deleted_at IS NULL AND status IN ('pending', 'verifier_checked', 'verified')",
			"CREATE UNIQUE INDEX IF NOT EXISTS idx_lender_return_distributions_one_per_repayment_funding ON lender_return_distributions (repayment_id, funding_id)",
		},
		Down: []string{
			"DROP INDEX IF EXISTS idx_lender_return_distributions_one_per_repayment_funding",
			"DROP INDEX IF EXISTS idx_repayments_one_active_per_schedule",
		},
	},
}

func runApplicationMigrations(db *gorm.DB) error {
	if err := db.Exec("CREATE TABLE IF NOT EXISTS schema_migrations (version varchar(255) PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())").Error; err != nil {
		return err
	}
	for _, migration := range applicationMigrations {
		var count int64
		if err := db.Table("schema_migrations").Where("version = ?", migration.Version).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			for _, statement := range migration.Up {
				if err := tx.Exec(statement).Error; err != nil {
					return err
				}
			}
			return tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", migration.Version).Error
		}); err != nil {
			return fmt.Errorf("apply migration %s: %w", migration.Version, err)
		}
	}
	return nil
}
