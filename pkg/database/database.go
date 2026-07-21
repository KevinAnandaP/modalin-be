package database

import (
	"fmt"
	"log"
	"time"

	"modalin-be/internal/model"
	"modalin-be/pkg/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func ConnectDB() {
	cfg := config.AppConfig

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		cfg.DBHost,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBPort,
		cfg.DBSSLMode,
		cfg.DBTimeZone,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}

	// Set database connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("Database connection successfully established!")

	// Run AutoMigrate for all 24 tables
	log.Println("Running database migrations...")
	err = db.AutoMigrate(
		&model.User{},
		&model.Role{},
		&model.UserRole{},
		&model.RoleRequest{},
		&model.BusinessCategory{},
		&model.Business{},
		&model.FinancialRecord{},
		&model.FinancialRecordProof{},
		&model.VerificationRequest{},
		&model.VerificationReport{},
		&model.CommunityVote{},
		&model.LoanCampaign{},
		&model.CampaignBudgetItem{},
		&model.CampaignMilestone{},
		&model.Funding{},
		&model.Disbursement{},
		&model.FundUsageProof{},
		&model.RevenueReport{},
		&model.RevenueReportProof{},
		&model.RepaymentSchedule{},
		&model.Repayment{},
		&model.LenderReturnDistribution{},
		&model.RiskAssessment{},
		&model.Dispute{},
		&model.AuditLog{},
		&model.StarterBusinessDetail{},
		&model.RepaymentRestructuringRequest{},
	)
	if err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}
	// Campaigns created before priority levels were introduced require borrower revision.
	if err := db.Exec(`UPDATE loan_campaigns SET status = 'needs_revision'
		WHERE status IN ('draft', 'rejected', 'admin_review')
		AND EXISTS (SELECT 1 FROM campaign_budget_items
			WHERE campaign_budget_items.campaign_id = loan_campaigns.id
			AND (priority_level IS NULL OR priority_level NOT IN ('high', 'medium', 'low')))`).Error; err != nil {
		log.Fatalf("Failed to mark legacy campaigns for revision: %v", err)
	}
	// GORM cannot express PostgreSQL partial unique indexes. This allows a user to
	// keep historical inactive businesses while enforcing exactly one active business.
	if err := db.Exec("DROP INDEX IF EXISTS idx_businesses_user_id").Error; err != nil {
		log.Fatalf("Failed to replace business user index: %v", err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_businesses_one_active_per_user ON businesses (user_id) WHERE status = 'active'").Error; err != nil {
		log.Fatalf("Failed to create active business uniqueness index: %v", err)
	}
	log.Println("Database migrations completed successfully!")

	// Run Master Data Seeder
	SeedData(db)

	DB = db
}
