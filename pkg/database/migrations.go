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
		Version: "20260725_auth_token_version",
		Up: []string{
			"ALTER TABLE users ADD COLUMN IF NOT EXISTS token_version bigint NOT NULL DEFAULT 1",
		},
		Down: []string{
			"ALTER TABLE users DROP COLUMN IF EXISTS token_version",
		},
	},
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
	{
		Version: "20260725_audit_log_contract",
		Up: []string{
			"ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS event_id uuid",
			"ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS request_id varchar(100)",
			"UPDATE audit_logs SET event_id = gen_random_uuid() WHERE event_id IS NULL",
			"ALTER TABLE audit_logs ALTER COLUMN event_id SET NOT NULL",
			"CREATE UNIQUE INDEX IF NOT EXISTS idx_audit_logs_event_id ON audit_logs (event_id)",
			"CREATE INDEX IF NOT EXISTS idx_audit_logs_request_id ON audit_logs (request_id)",
			"CREATE UNIQUE INDEX IF NOT EXISTS idx_audit_logs_request_event ON audit_logs (request_id, action, entity_type, entity_id) WHERE request_id IS NOT NULL",
			"CREATE OR REPLACE FUNCTION modalin_prevent_audit_log_mutation() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'audit_logs is append-only'; END; $$ LANGUAGE plpgsql",
			"DROP TRIGGER IF EXISTS trg_audit_logs_append_only ON audit_logs",
			"CREATE TRIGGER trg_audit_logs_append_only BEFORE UPDATE OR DELETE ON audit_logs FOR EACH ROW EXECUTE FUNCTION modalin_prevent_audit_log_mutation()",
		},
		Down: []string{
			"DROP TRIGGER IF EXISTS trg_audit_logs_append_only ON audit_logs",
			"DROP FUNCTION IF EXISTS modalin_prevent_audit_log_mutation()",
			"DROP INDEX IF EXISTS idx_audit_logs_request_id",
			"DROP INDEX IF EXISTS idx_audit_logs_request_event",
			"DROP INDEX IF EXISTS idx_audit_logs_event_id",
			"ALTER TABLE audit_logs DROP COLUMN IF EXISTS request_id",
			"ALTER TABLE audit_logs DROP COLUMN IF EXISTS event_id",
		},
	},
	{
		Version: "20260725_dispute_contract",
		Up: []string{
			"ALTER TABLE disputes ADD COLUMN IF NOT EXISTS resolution_note text",
			"ALTER TABLE disputes ADD COLUMN IF NOT EXISTS resolved_at timestamptz",
			"CREATE UNIQUE INDEX IF NOT EXISTS idx_disputes_one_active_per_reporter_target_type ON disputes (campaign_id, reported_by, COALESCE(target_user_id, '00000000-0000-0000-0000-000000000000'::uuid), type) WHERE deleted_at IS NULL AND status IN ('open', 'under_review')",
		},
		Down: []string{
			"DROP INDEX IF EXISTS idx_disputes_one_active_per_reporter_target_type",
			"ALTER TABLE disputes DROP COLUMN IF EXISTS resolved_at",
			"ALTER TABLE disputes DROP COLUMN IF EXISTS resolution_note",
		},
	},
	{
		Version: "20260725_restructuring_contract",
		Up: []string{
			"ALTER TABLE repayment_restructuring_requests ADD COLUMN IF NOT EXISTS requested_by uuid",
			"ALTER TABLE repayment_restructuring_requests ADD COLUMN IF NOT EXISTS previous_tenor_months int",
			"ALTER TABLE repayment_restructuring_requests ADD COLUMN IF NOT EXISTS remaining_principal bigint",
			"ALTER TABLE repayment_restructuring_requests ADD COLUMN IF NOT EXISTS remaining_margin bigint",
			"CREATE UNIQUE INDEX IF NOT EXISTS idx_restructuring_one_pending_per_campaign ON repayment_restructuring_requests (campaign_id) WHERE deleted_at IS NULL AND status = 'pending'",
		},
		Down: []string{
			"DROP INDEX IF EXISTS idx_restructuring_one_pending_per_campaign",
			"ALTER TABLE repayment_restructuring_requests DROP COLUMN IF EXISTS remaining_margin",
			"ALTER TABLE repayment_restructuring_requests DROP COLUMN IF EXISTS remaining_principal",
			"ALTER TABLE repayment_restructuring_requests DROP COLUMN IF EXISTS previous_tenor_months",
			"ALTER TABLE repayment_restructuring_requests DROP COLUMN IF EXISTS requested_by",
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
