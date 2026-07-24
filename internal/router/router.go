package router

import (
	authHandler "modalin-be/internal/auth/handler"
	authRepository "modalin-be/internal/auth/repository"
	authService "modalin-be/internal/auth/service"
	businessHandler "modalin-be/internal/business/handler"
	businessRepository "modalin-be/internal/business/repository"
	businessService "modalin-be/internal/business/service"
	businessStorage "modalin-be/internal/business/storage"
	campaignHandler "modalin-be/internal/campaign/handler"
	campaignRepository "modalin-be/internal/campaign/repository"
	campaignService "modalin-be/internal/campaign/service"
	campaignStorage "modalin-be/internal/campaign/storage"
	healthHandler "modalin-be/internal/health/handler"
	verificationHandler "modalin-be/internal/verification/handler"
	verificationRepository "modalin-be/internal/verification/repository"
	verificationService "modalin-be/internal/verification/service"
	"modalin-be/pkg/config"
	"modalin-be/pkg/database"
	"modalin-be/pkg/middleware"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
	// Root route
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Welcome to Modalin Backend API",
			"version": "1.0.0",
		})
	})

	// Initialize Handlers
	health := healthHandler.NewHealthHandler()
	auth := authHandler.NewAuthHandler(authService.NewAuthService(authRepository.NewAuthRepository(database.DB), config.AppConfig.JWTSecret))
	verificationSvc := verificationService.New(verificationRepository.New(database.DB))
	business := businessHandler.NewBusinessHandler(businessService.NewBusinessService(businessRepository.NewBusinessRepository(database.DB), verificationSvc.RefreshBusinessRisk), businessStorage.NewLocalProofStorage(config.AppConfig.UploadDir, "/uploads"))
	campaign := campaignHandler.NewCampaignHandler(campaignService.NewCampaignService(campaignRepository.NewCampaignRepository(database.DB), verificationSvc.RefreshCampaignRisk), campaignStorage.NewLocalProofStorage(config.AppConfig.UploadDir, "/uploads"))
	verification := verificationHandler.New(verificationSvc, campaignStorage.NewLocalProofStorage(config.AppConfig.UploadDir, "/uploads"))

	// Register Routes
	app.Get("/health", health.CheckHealth)

	// API group for future endpoints
	api := app.Group("/api")
	v1 := api.Group("/v1")

	// Register API v1 Routes
	v1.Get("/health", health.CheckHealth)
	v1.Get("/ping", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "pong",
		})
	})

	authRoutes := v1.Group("/auth")
	authRoutes.Post("/register", auth.Register)
	authRoutes.Post("/login", auth.Login)
	authRoutes.Post("/google", auth.GoogleAuth)
	authRoutes.Post("/google/complete", auth.CompleteGoogleAuth)
	authRoutes.Get("/me", middleware.JWTProtected(config.AppConfig.JWTSecret), auth.Me)
	authRoutes.Post("/roles", middleware.JWTProtected(config.AppConfig.JWTSecret), auth.RequestRole)

	// Admin Role Management Routes
	adminRoutes := v1.Group("/admin", middleware.JWTProtected(config.AppConfig.JWTSecret), middleware.RequireRole("admin"))
	adminRoutes.Get("/roles/requests", auth.GetRoleRequests)
	adminRoutes.Post("/roles/review", auth.ReviewRoleRequest)

	// Business Management Routes (Protected Borrower)
	businessRoutes := v1.Group("/businesses", middleware.JWTProtected(config.AppConfig.JWTSecret), middleware.RequireRole("borrower"))
	businessRoutes.Post("/", business.CreateBusiness)
	businessRoutes.Get("/me", business.GetMyBusiness)
	businessRoutes.Put("/me", business.UpdateBusiness)
	businessRoutes.Delete("/me", business.DeactivateBusiness)

	// Financial Record Routes (Protected Borrower)
	finRoutes := v1.Group("/financial-records", middleware.JWTProtected(config.AppConfig.JWTSecret), middleware.RequireRole("borrower"))
	finRoutes.Post("/", business.CreateFinancialRecord)
	finRoutes.Get("/", business.GetFinancialRecords)
	finRoutes.Get("/summary", business.GetFinancialSummary)
	finRoutes.Get("/:id", business.GetFinancialRecord)
	finRoutes.Put("/:id", business.UpdateFinancialRecord)
	finRoutes.Post("/:id/proofs", business.UploadFinancialRecordProof)
	finRoutes.Get("/proofs/:proofID/download", business.DownloadFinancialRecordProof)
	finRoutes.Delete("/:id", business.DeleteFinancialRecord)

	// Loan campaign management (borrower) and public catalogue.
	v1.Get("/campaigns", campaign.Catalog)
	lenderCampaignRoutes := v1.Group("/campaigns", middleware.JWTProtected(config.AppConfig.JWTSecret), middleware.RequireRole("lender"))
	lenderCampaignRoutes.Post("/:id/fundings", campaign.Pledge)
	lenderCampaignRoutes.Get("/fundings/me", campaign.ListLenderFundings)
	lenderRoutes := v1.Group("/lender", middleware.JWTProtected(config.AppConfig.JWTSecret), middleware.RequireRole("lender"))
	lenderRoutes.Get("/return-distributions", campaign.ListLenderReturnDistributions)
	protectedCampaignRoutes := v1.Group("/campaigns", middleware.JWTProtected(config.AppConfig.JWTSecret))
	protectedCampaignRoutes.Get("/:id/fund-usage-proofs/:proofID/download", campaign.DownloadFundUsageProof)
	protectedCampaignRoutes.Post("/:id/verification-requests", middleware.RequireRole("borrower"), middleware.SensitiveActionRateLimit(), verification.CreateRequest)
	v1.Post("/businesses/:id/community-vote", middleware.JWTProtected(config.AppConfig.JWTSecret), middleware.SensitiveActionRateLimit(), verification.Vote)
	v1.Get("/verification-requests/:id/photo/download", middleware.JWTProtected(config.AppConfig.JWTSecret), verification.DownloadPhoto)
	campaignRoutes := v1.Group("/campaigns", middleware.JWTProtected(config.AppConfig.JWTSecret), middleware.RequireRole("borrower"))
	campaignRoutes.Post("/", campaign.Create)
	campaignRoutes.Get("/me", campaign.ListMine)
	campaignRoutes.Get("/:id", campaign.GetMine)
	campaignRoutes.Put("/:id", campaign.Update)
	campaignRoutes.Delete("/:id", campaign.Delete)
	campaignRoutes.Post("/:id/submit", campaign.Submit)
	campaignRoutes.Post("/:id/budget-items", campaign.CreateBudget)
	campaignRoutes.Get("/:id/budget-items", campaign.ListBudget)
	campaignRoutes.Put("/:id/budget-items/:budgetID", campaign.UpdateBudget)
	campaignRoutes.Delete("/:id/budget-items/:budgetID", campaign.DeleteBudget)
	campaignRoutes.Get("/:id/milestones", campaign.ListMilestones)
	campaignRoutes.Post("/:id/disbursements/:disbursementID/proofs", middleware.SensitiveActionRateLimit(), campaign.UploadFundUsageProof)
	campaignRoutes.Get("/:id/disbursements", campaign.ListDisbursements)
	campaignRoutes.Post("/:id/monthly-reports", middleware.SensitiveActionRateLimit(), campaign.CreateMonthlyProgressReport)
	campaignRoutes.Get("/:id/monthly-reports", campaign.ListMonthlyProgressReports)
	campaignRoutes.Post("/:id/monthly-reports/:reportID/resubmit", middleware.SensitiveActionRateLimit(), campaign.ResubmitMonthlyProgressReport)
	campaignRoutes.Post("/:id/revenue-reports", middleware.SensitiveActionRateLimit(), campaign.CreateRevenueReport)
	campaignRoutes.Get("/:id/revenue-reports", campaign.ListRevenueReports)
	campaignRoutes.Post("/:id/revenue-reports/:reportID/resubmit", middleware.SensitiveActionRateLimit(), campaign.ResubmitRevenueReport)
	campaignRoutes.Get("/:id/repayment-schedules", campaign.ListRepaymentSchedules)
	campaignRoutes.Post("/:id/repayments", middleware.SensitiveActionRateLimit(), campaign.CreateRepayment)
	verifierRoutes := v1.Group("/verifier", middleware.JWTProtected(config.AppConfig.JWTSecret), middleware.RequireRole("verifier"))
	verifierRoutes.Post("/monthly-reports/:id/review", middleware.SensitiveActionRateLimit(), campaign.VerifyMonthlyProgressReport)
	verifierRoutes.Post("/revenue-reports/:id/review", middleware.SensitiveActionRateLimit(), campaign.VerifyRevenueReport)
	verifierRoutes.Post("/repayments/:id/review", middleware.SensitiveActionRateLimit(), campaign.VerifyRepayment)
	verifierRoutes.Get("/verification-requests", verification.ListMine)
	verifierRoutes.Post("/verification-requests/:id/report", middleware.SensitiveActionRateLimit(), verification.SubmitReport)
	verifierRoutes.Post("/verification-requests/:id/photo", middleware.SensitiveActionRateLimit(), verification.UploadPhoto)
	adminRoutes.Get("/verification-requests", verification.ListAdmin)
	adminRoutes.Post("/verification-requests/:id/assign", middleware.SensitiveActionRateLimit(), verification.Assign)
	adminRoutes.Post("/verification-requests/:id/reassign", middleware.SensitiveActionRateLimit(), verification.Reassign)
	adminRoutes.Post("/verification-requests/:id/cancel", middleware.SensitiveActionRateLimit(), verification.Cancel)
	adminRoutes.Post("/risk-assessments/backfill", middleware.SensitiveActionRateLimit(), verification.BackfillRisks)
	adminRoutes.Post("/verification-requests/:id/decision", middleware.SensitiveActionRateLimit(), verification.Decide)
	adminRoutes.Post("/campaigns/:id/review", campaign.Review)
	adminRoutes.Post("/disbursements/:id/confirm-transfer", middleware.SensitiveActionRateLimit(), campaign.ConfirmDisbursement)
	adminRoutes.Post("/fund-usage-proofs/:proofID/review", campaign.ReviewFundUsageProof)
	adminRoutes.Get("/fund-usage-proofs", campaign.ListFundUsageProofs)
	adminRoutes.Post("/revenue-reports/:id/review", middleware.SensitiveActionRateLimit(), campaign.ReviewRevenueReport)
	adminRoutes.Post("/repayments/:id/review", middleware.SensitiveActionRateLimit(), campaign.ReviewRepayment)
	adminRoutes.Post("/lender-return-distributions/:id/distribute", middleware.SensitiveActionRateLimit(), campaign.MarkLenderReturnDistributed)
}
