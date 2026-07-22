package repository_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// This test runs only when TEST_DATABASE_DSN targets an isolated local database.
// It creates and drops a dedicated schema; application tables are never touched.
func TestPostgresConstraintsAndRollback(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" { t.Skip("TEST_DATABASE_DSN is not configured") }
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	schema := "campaign_test_" + uuid.NewString()[0:8]
	if err := db.Exec("CREATE SCHEMA " + schema).Error; err != nil { t.Fatal(err) }
	t.Cleanup(func() { _ = db.Exec("DROP SCHEMA " + schema + " CASCADE").Error })
	if err := db.Exec(fmt.Sprintf("CREATE TABLE %s.milestones (id uuid PRIMARY KEY); CREATE TABLE %s.disbursements (id uuid PRIMARY KEY, milestone_id uuid NOT NULL UNIQUE); CREATE TABLE %s.proofs (id uuid PRIMARY KEY, disbursement_id uuid NOT NULL, deleted_at timestamptz); CREATE UNIQUE INDEX active_proof_per_disbursement ON %s.proofs(disbursement_id) WHERE deleted_at IS NULL;", schema, schema, schema, schema)).Error; err != nil { t.Fatal(err) }
	milestone, first, second := uuid.New(), uuid.New(), uuid.New()
	if err := db.Exec(fmt.Sprintf("INSERT INTO %s.milestones VALUES (?)", schema), milestone).Error; err != nil { t.Fatal(err) }
	if err := db.Exec(fmt.Sprintf("INSERT INTO %s.disbursements VALUES (?, ?)", schema), first, milestone).Error; err != nil { t.Fatal(err) }
	if err := db.Exec(fmt.Sprintf("INSERT INTO %s.disbursements VALUES (?, ?)", schema), second, milestone).Error; err == nil { t.Fatal("expected duplicate milestone disbursement to fail") }
	proofA, proofB := uuid.New(), uuid.New()
	if err := db.Exec(fmt.Sprintf("INSERT INTO %s.proofs VALUES (?, ?, NULL)", schema), proofA, first).Error; err != nil { t.Fatal(err) }
	if err := db.Exec(fmt.Sprintf("INSERT INTO %s.proofs VALUES (?, ?, NULL)", schema), proofB, first).Error; err == nil { t.Fatal("expected duplicate active proof to fail") }
	if err := db.Transaction(func(tx *gorm.DB) error { if err := tx.Exec(fmt.Sprintf("INSERT INTO %s.milestones VALUES (?)", schema), uuid.New()).Error; err != nil { return err }; return context.Canceled }); err == nil { t.Fatal("expected rollback transaction error") }
	var count int64; if err := db.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s.milestones", schema)).Scan(&count).Error; err != nil { t.Fatal(err) }; if count != 1 { t.Fatalf("rollback left %d milestones", count) }
}
