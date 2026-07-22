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
	business := businessHandler.NewBusinessHandler(businessService.NewBusinessService(businessRepository.NewBusinessRepository(database.DB)), businessStorage.NewLocalProofStorage(config.AppConfig.UploadDir, "/uploads"))
	campaign := campaignHandler.NewCampaignHandler(campaignService.NewCampaignService(campaignRepository.NewCampaignRepository(database.DB)), campaignStorage.NewLocalProofStorage(config.AppConfig.UploadDir, "/uploads"))

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
	protectedCampaignRoutes := v1.Group("/campaigns", middleware.JWTProtected(config.AppConfig.JWTSecret))
	protectedCampaignRoutes.Get("/:id/fund-usage-proofs/:proofID/download", campaign.DownloadFundUsageProof)
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
	campaignRoutes.Post("/:id/disbursements/:disbursementID/proofs", campaign.UploadFundUsageProof)
	campaignRoutes.Get("/:id/disbursements", campaign.ListDisbursements)
	adminRoutes.Post("/campaigns/:id/review", campaign.Review)
	adminRoutes.Post("/fund-usage-proofs/:proofID/review", campaign.ReviewFundUsageProof)
	adminRoutes.Get("/fund-usage-proofs", campaign.ListFundUsageProofs)
}
