package database

import "testing"

func TestRepaymentGuardsMigrationHasReversibleDatabaseGuards(t *testing.T) {
	if len(applicationMigrations) == 0 {
		t.Fatal("expected application migrations")
	}
	migration := applicationMigrations[0]
	if len(migration.Up) != 2 || len(migration.Down) != 2 {
		t.Fatalf("expected reversible repayment guards, got %#v", migration)
	}
	if migration.Version != "20260724_repayment_guards" {
		t.Fatalf("unexpected migration version %q", migration.Version)
	}
}
