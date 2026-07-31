package database

import (
	"fmt"
	"log"
	"strings"
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

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(databaseLogMode(cfg.AppEnv))})
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
	DB = db
}

func databaseLogMode(appEnv string) logger.LogLevel {
	if strings.EqualFold(strings.TrimSpace(appEnv), "production") {
		return logger.Warn
	}
	return logger.Info
}

func CloseDB() error {
	if DB == nil {
		return nil
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// MigrateDB performs schema changes explicitly. It must be run as a separate
// deployment step, never implicitly when an API instance starts.
func MigrateDB() error {
	if DB == nil {
		return fmt.Errorf("database connection has not been initialized")
	}
	db := DB
	log.Println("Running database migrations...")
	if err := db.AutoMigrate(
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
		&model.MonthlyProgressReport{},
		&model.MonthlyProgressReportProof{},
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
	); err != nil {
		return fmt.Errorf("auto migrate schema: %w", err)
	}
	// Campaigns created before priority levels were introduced require borrower revision.
	if err := db.Exec(`UPDATE loan_campaigns SET status = 'needs_revision'
		WHERE status IN ('draft', 'rejected', 'admin_review')
		AND EXISTS (SELECT 1 FROM campaign_budget_items
			WHERE campaign_budget_items.campaign_id = loan_campaigns.id
			AND (priority_level IS NULL OR priority_level NOT IN ('high', 'medium', 'low')))`).Error; err != nil {
		return fmt.Errorf("mark legacy campaigns for revision: %w", err)
	}
	// GORM cannot express PostgreSQL partial unique indexes. This allows a user to
	// keep historical inactive businesses while enforcing exactly one active business.
	if err := db.Exec("DROP INDEX IF EXISTS idx_businesses_user_id").Error; err != nil {
		return fmt.Errorf("replace business user index: %w", err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_businesses_one_active_per_user ON businesses (user_id) WHERE status = 'active'").Error; err != nil {
		return fmt.Errorf("create active business uniqueness index: %w", err)
	}
	if err := runApplicationMigrations(db); err != nil {
		return fmt.Errorf("run application migrations: %w", err)
	}
	log.Println("Database migrations completed successfully!")

	// Run Master Data Seeder
	SeedData(db)
	return nil
}
