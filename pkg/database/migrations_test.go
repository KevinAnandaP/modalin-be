package database

import (
	"strings"
	"testing"

	"gorm.io/gorm/logger"
)

func TestApplicationMigrationsHaveReversibleGuards(t *testing.T) {
	if len(applicationMigrations) == 0 {
		t.Fatal("expected application migrations")
	}
	var migration *migration
	for i := range applicationMigrations {
		if applicationMigrations[i].Version == "20260724_repayment_guards" {
			migration = &applicationMigrations[i]
			break
		}
	}
	if migration == nil {
		t.Fatal("expected repayment guard migration")
	}
	if len(migration.Up) != 2 || len(migration.Down) != 2 {
		t.Fatalf("expected reversible repayment guards, got %#v", migration)
	}
	if migration.Version != "20260724_repayment_guards" {
		t.Fatalf("unexpected migration version %q", migration.Version)
	}
}

func TestDatabaseLogModeAvoidsQueryLoggingInProduction(t *testing.T) {
	if got := databaseLogMode("production"); got != logger.Warn {
		t.Fatalf("production log mode = %v, want Warn", got)
	}
	if got := databaseLogMode("development"); got != logger.Info {
		t.Fatalf("development log mode = %v, want Info", got)
	}
}

func TestAuditLogContractMigrationIsAppendOnlyAndIdempotent(t *testing.T) {
	var found *migration
	for i := range applicationMigrations {
		if applicationMigrations[i].Version == "20260725_audit_log_contract" {
			found = &applicationMigrations[i]
			break
		}
	}
	if found == nil {
		t.Fatal("expected audit log contract migration")
	}
	joined := strings.Join(found.Up, "\n")
	for _, fragment := range []string{"event_id", "request_id", "idx_audit_logs_request_event", "append_only"} {
		if !strings.Contains(joined, fragment) {
			t.Fatalf("migration does not contain %q: %s", fragment, joined)
		}
	}
	if len(found.Down) == 0 {
		t.Fatal("expected reversible audit migration")
	}
}

func TestDisputeContractMigrationExists(t *testing.T) {
	for i := range applicationMigrations {
		if applicationMigrations[i].Version == "20260725_dispute_contract" {
			return
		}
	}
	t.Fatal("expected dispute contract migration")
}

func TestRestructuringContractMigrationExists(t *testing.T) {
	for i := range applicationMigrations {
		if applicationMigrations[i].Version == "20260725_restructuring_contract" {
			return
		}
	}
	t.Fatal("expected restructuring contract migration")
}

func TestAuthTokenVersionMigrationIsReversible(t *testing.T) {
	var found *migration
	for i := range applicationMigrations {
		if applicationMigrations[i].Version == "20260725_auth_token_version" {
			found = &applicationMigrations[i]
			break
		}
	}
	if found == nil {
		t.Fatal("expected auth token-version migration")
	}
	if len(found.Up) != 1 || !strings.Contains(found.Up[0], "token_version") || len(found.Down) != 1 {
		t.Fatalf("expected reversible token-version migration, got %#v", found)
	}
}

func TestRiskAssessmentBackfillMigrationMarksLegacyAssessmentsDataLimited(t *testing.T) {
	var found *migration
	for i := range applicationMigrations {
		if applicationMigrations[i].Version == "20260724_risk_assessment_backfill" {
			found = &applicationMigrations[i]
			break
		}
	}
	if found == nil {
		t.Fatal("expected risk assessment backfill migration")
	}
	if len(found.Up) != 3 || len(found.Down) != 2 {
		t.Fatalf("unexpected backfill migration %#v", found)
	}
}
