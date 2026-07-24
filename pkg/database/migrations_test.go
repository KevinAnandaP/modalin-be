package database

import "testing"

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
